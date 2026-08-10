package mailer

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Message struct {
	To, Subject, Text string
}

type Sender interface {
	Send(context.Context, Message) error
}

type SMTPConfig struct {
	Address, Username, Password, From, FromName string
	RequireTLS                                  bool
}

type SMTP struct{ config SMTPConfig }

func NewSMTP(config SMTPConfig) (*SMTP, error) {
	if _, _, err := net.SplitHostPort(config.Address); err != nil {
		return nil, errors.New("invalid SMTP address")
	}
	if _, err := mail.ParseAddress(config.From); err != nil {
		return nil, errors.New("invalid SMTP from address")
	}
	return &SMTP{config: config}, nil
}

func (s *SMTP) Send(ctx context.Context, message Message) error {
	to, err := mail.ParseAddress(message.To)
	if err != nil || containsNewline(message.Subject) || containsNewline(s.config.FromName) {
		return errors.New("invalid email envelope")
	}
	host, _, _ := net.SplitHostPort(s.config.Address)
	connection, err := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, "tcp", s.config.Address)
	if err != nil {
		return err
	}
	defer connection.Close()
	deadline := time.Now().Add(30 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err = connection.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); supported {
		if err = client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}); err != nil {
			return err
		}
	} else if s.config.RequireTLS {
		return errors.New("SMTP server does not support required STARTTLS")
	}
	if s.config.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.config.Username, s.config.Password, host)); err != nil {
			return err
		}
	}
	from, _ := mail.ParseAddress(s.config.From)
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	buffer := bufio.NewWriter(writer)
	headers := []string{
		"From: " + (&mail.Address{Name: s.config.FromName, Address: from.Address}).String(),
		"To: " + to.String(),
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	if _, err = fmt.Fprint(buffer, strings.Join(headers, "\r\n")+"\r\n\r\n"+normalizeBody(message.Text)); err == nil {
		err = buffer.Flush()
	}
	closeErr := writer.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return client.Quit()
}

func containsNewline(value string) bool {
	return strings.ContainsAny(value, "\r\n")
}

func normalizeBody(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", "\r\n")
}

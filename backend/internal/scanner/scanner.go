package scanner

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

var ErrThreat = errors.New("malicious content detected")

type Scanner interface {
	Scan(context.Context, io.Reader) error
	Health(context.Context) error
}

type Disabled struct{}

func (Disabled) Scan(context.Context, io.Reader) error { return nil }
func (Disabled) Health(context.Context) error          { return nil }

type ClamAV struct {
	address string
	timeout time.Duration
}

func NewClamAV(address string) *ClamAV {
	return &ClamAV{address: address, timeout: 15 * time.Second}
}

func (c *ClamAV) Health(ctx context.Context) error {
	connection, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err = io.WriteString(connection, "zPING\x00"); err != nil {
		return err
	}
	response, err := readResponse(connection)
	if err != nil {
		return err
	}
	if response != "PONG" {
		return errors.New("unexpected ClamAV health response")
	}
	return nil
}

func (c *ClamAV) Scan(ctx context.Context, reader io.Reader) error {
	connection, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer connection.Close()
	if _, err = io.WriteString(connection, "zINSTREAM\x00"); err != nil {
		return err
	}
	buffer := make([]byte, 64*1024)
	for {
		count, readErr := reader.Read(buffer)
		if count > 0 {
			if err = binary.Write(connection, binary.BigEndian, uint32(count)); err != nil {
				return err
			}
			if err = writeAll(connection, buffer[:count]); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if err = binary.Write(connection, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	response, err := readResponse(connection)
	if err != nil {
		return err
	}
	if strings.HasSuffix(response, " OK") {
		return nil
	}
	if strings.HasSuffix(response, " FOUND") {
		return ErrThreat
	}
	return fmt.Errorf("ClamAV scan failed")
}

func (c *ClamAV) connect(ctx context.Context) (net.Conn, error) {
	connection, err := (&net.Dialer{Timeout: c.timeout}).DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(c.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err = connection.SetDeadline(deadline); err != nil {
		connection.Close()
		return nil, err
	}
	return connection, nil
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func readResponse(reader io.Reader) (string, error) {
	response, err := bufio.NewReader(io.LimitReader(reader, 4096)).ReadString(0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	response = strings.TrimSpace(strings.TrimSuffix(response, "\x00"))
	if response == "" {
		return "", errors.New("empty ClamAV response")
	}
	return response, nil
}

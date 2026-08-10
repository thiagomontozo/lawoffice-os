package mailer

import "testing"

func TestSMTPRejectsHeaderInjection(t *testing.T) {
	sender, err := NewSMTP(SMTPConfig{Address: "127.0.0.1:2525", From: "noreply@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if err = sender.Send(t.Context(), Message{To: "client@example.test", Subject: "Valid\r\nBcc: attacker@example.test", Text: "body"}); err == nil {
		t.Fatal("header injection should be rejected before connecting")
	}
}

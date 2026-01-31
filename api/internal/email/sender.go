package email

import (
	"context"
	"fmt"
	"log"
	"net/smtp"

	"github.com/feather/api/internal/config"
)

type Sender interface {
	Send(ctx context.Context, to, subject, textBody, htmlBody string) error
}

type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPSender(cfg config.EmailConfig) *SMTPSender {
	return &SMTPSender{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
	}
}

func (s *SMTPSender) Send(ctx context.Context, to, subject, textBody, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	// Build message
	msg := fmt.Sprintf("From: %s\r\n", s.from)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)

	if htmlBody != "" {
		msg += "MIME-Version: 1.0\r\n"
		msg += "Content-Type: multipart/alternative; boundary=\"boundary\"\r\n"
		msg += "\r\n"
		msg += "--boundary\r\n"
		msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
		msg += "\r\n"
		msg += textBody + "\r\n"
		msg += "--boundary\r\n"
		msg += "Content-Type: text/html; charset=\"utf-8\"\r\n"
		msg += "\r\n"
		msg += htmlBody + "\r\n"
		msg += "--boundary--\r\n"
	} else {
		msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
		msg += "\r\n"
		msg += textBody + "\r\n"
	}

	err := smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("[email] Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("[email] Sent email to %s: %s", to, subject)
	return nil
}

type NoOpSender struct{}

func (s *NoOpSender) Send(ctx context.Context, to, subject, textBody, htmlBody string) error {
	log.Printf("[email/noop] Would send to %s: %s", to, subject)
	return nil
}

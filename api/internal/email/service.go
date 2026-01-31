package email

import (
	"context"
	"embed"
	"html/template"
	"log"

	"github.com/feather/api/internal/config"
)

//go:embed templates/*.html templates/*.txt
var templateFS embed.FS

type Service struct {
	sender    Sender
	templates *template.Template
	publicURL string
	enabled   bool
}

func NewService(cfg config.EmailConfig, publicURL string) (*Service, error) {
	var sender Sender
	if cfg.Enabled {
		sender = NewSMTPSender(cfg)
	} else {
		sender = &NoOpSender{}
	}

	templates, err := template.ParseFS(templateFS, "templates/*.html", "templates/*.txt")
	if err != nil {
		// Templates might not exist yet, create empty template
		templates = template.New("empty")
	}

	return &Service{
		sender:    sender,
		templates: templates,
		publicURL: publicURL,
		enabled:   cfg.Enabled,
	}, nil
}

func (s *Service) IsEnabled() bool {
	return s.enabled
}

type InviteEmailData struct {
	WorkspaceName string
	InviterName   string
	InviteURL     string
}

func (s *Service) SendWorkspaceInvite(ctx context.Context, to string, data InviteEmailData) error {
	if !s.enabled {
		log.Printf("[email] Would send workspace invite to %s: %+v", to, data)
		return nil
	}

	subject := "You've been invited to join " + data.WorkspaceName
	body := "You've been invited to join " + data.WorkspaceName + " on Feather.\n\n"
	body += "Click here to accept: " + data.InviteURL + "\n"

	return s.sender.Send(ctx, to, subject, body, "")
}

type PasswordResetEmailData struct {
	ResetURL string
}

func (s *Service) SendPasswordReset(ctx context.Context, to string, data PasswordResetEmailData) error {
	if !s.enabled {
		log.Printf("[email] Would send password reset to %s: %+v", to, data)
		return nil
	}

	subject := "Reset your Feather password"
	body := "You requested to reset your password.\n\n"
	body += "Click here to reset: " + data.ResetURL + "\n\n"
	body += "If you didn't request this, you can ignore this email.\n"

	return s.sender.Send(ctx, to, subject, body, "")
}

type VerifyEmailData struct {
	VerifyURL string
}

func (s *Service) SendEmailVerification(ctx context.Context, to string, data VerifyEmailData) error {
	if !s.enabled {
		log.Printf("[email] Would send email verification to %s: %+v", to, data)
		return nil
	}

	subject := "Verify your email address"
	body := "Please verify your email address by clicking the link below:\n\n"
	body += data.VerifyURL + "\n"

	return s.sender.Send(ctx, to, subject, body, "")
}

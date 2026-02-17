package email

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"

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
		slog.Debug("would send workspace invite", "component", "email", "to", to, "workspace", data.WorkspaceName)
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
		slog.Debug("would send password reset", "component", "email", "to", to)
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
		slog.Debug("would send email verification", "component", "email", "to", to)
		return nil
	}

	subject := "Verify your email address"
	body := "Please verify your email address by clicking the link below:\n\n"
	body += data.VerifyURL + "\n"

	return s.sender.Send(ctx, to, subject, body, "")
}

// NotificationDigestItem represents a single notification in a digest
type NotificationDigestItem struct {
	ChannelName string
	SenderName  string
	Preview     string
	Type        string
}

// NotificationDigestData contains data for notification digest emails
type NotificationDigestData struct {
	WorkspaceName string
	Items         []NotificationDigestItem
	WorkspaceURL  string
}

func (s *Service) SendNotificationDigest(ctx context.Context, to string, data NotificationDigestData) error {
	if !s.enabled {
		slog.Debug("would send notification digest", "component", "email", "to", to, "count", len(data.Items), "workspace", data.WorkspaceName)
		return nil
	}

	count := len(data.Items)
	subject := ""
	if count == 1 {
		subject = "1 new notification in " + data.WorkspaceName
	} else {
		subject = fmt.Sprintf("%d new notifications in %s", count, data.WorkspaceName)
	}

	// Build plain text body
	body := "You have new notifications in " + data.WorkspaceName + ":\n\n"
	for _, item := range data.Items {
		prefix := ""
		switch item.Type {
		case "mention":
			prefix = "[Mentioned] "
		case "dm":
			prefix = "[DM] "
		case "channel":
			prefix = "[@channel] "
		case "here":
			prefix = "[@here] "
		case "everyone":
			prefix = "[@everyone] "
		}
		body += prefix + item.SenderName + " in #" + item.ChannelName + ": " + item.Preview + "\n"
	}
	body += "\nOpen Feather: " + data.WorkspaceURL + "\n"

	return s.sender.Send(ctx, to, subject, body, "")
}

// GetPublicURL returns the public URL for the service
func (s *Service) GetPublicURL() string {
	return s.publicURL
}

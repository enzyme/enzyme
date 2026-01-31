package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

func Validate(cfg *Config) error {
	var errs []error

	// Server validation
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535"))
	}
	if cfg.Server.PublicURL != "" {
		if _, err := url.Parse(cfg.Server.PublicURL); err != nil {
			errs = append(errs, fmt.Errorf("server.public_url is not a valid URL: %w", err))
		}
	}

	// Database validation
	if cfg.Database.Path == "" {
		errs = append(errs, fmt.Errorf("database.path is required"))
	}

	// Auth validation
	if cfg.Auth.SessionDuration < time.Hour {
		errs = append(errs, fmt.Errorf("auth.session_duration must be at least 1 hour"))
	}
	if cfg.Auth.BcryptCost < 10 || cfg.Auth.BcryptCost > 31 {
		errs = append(errs, fmt.Errorf("auth.bcrypt_cost must be between 10 and 31"))
	}

	// Files validation
	if cfg.Files.StoragePath == "" {
		errs = append(errs, fmt.Errorf("files.storage_path is required"))
	}
	if cfg.Files.MaxUploadSize < 1024 {
		errs = append(errs, fmt.Errorf("files.max_upload_size must be at least 1KB"))
	}

	// Email validation (only if enabled)
	if cfg.Email.Enabled {
		if cfg.Email.Host == "" {
			errs = append(errs, fmt.Errorf("email.host is required when email is enabled"))
		}
		if cfg.Email.From == "" {
			errs = append(errs, fmt.Errorf("email.from is required when email is enabled"))
		}
		if cfg.Email.Port < 1 || cfg.Email.Port > 65535 {
			errs = append(errs, fmt.Errorf("email.port must be between 1 and 65535"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

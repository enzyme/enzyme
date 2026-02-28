package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

func Validate(cfg *Config) error {
	var errs []error

	// Log validation
	switch cfg.Log.Level {
	case "debug", "info", "warn", "error":
		// valid
	default:
		errs = append(errs, fmt.Errorf("log.level must be one of: debug, info, warn, error"))
	}
	switch cfg.Log.Format {
	case "text", "json":
		// valid
	default:
		errs = append(errs, fmt.Errorf("log.format must be one of: text, json"))
	}

	// Server validation
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("server.port must be between 1 and 65535"))
	}
	if cfg.Server.PublicURL != "" {
		if _, err := url.Parse(cfg.Server.PublicURL); err != nil {
			errs = append(errs, fmt.Errorf("server.public_url is not a valid URL: %w", err))
		}
	}

	// Allowed origins validation
	for i, origin := range cfg.Server.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" {
			errs = append(errs, fmt.Errorf("server.allowed_origins[%d] %q is not a valid URL with scheme", i, origin))
		}
	}

	// TLS validation
	switch cfg.Server.TLS.Mode {
	case "", "off":
		// no additional validation needed
	case "auto":
		if cfg.Server.TLS.Auto.Domain == "" {
			errs = append(errs, fmt.Errorf("server.tls.auto.domain is required when tls mode is auto"))
		}
		if cfg.Server.TLS.Auto.CacheDir == "" {
			errs = append(errs, fmt.Errorf("server.tls.auto.cache_dir is required when tls mode is auto"))
		}
	case "manual":
		if cfg.Server.TLS.CertFile == "" {
			errs = append(errs, fmt.Errorf("server.tls.cert_file is required when tls mode is manual"))
		}
		if cfg.Server.TLS.KeyFile == "" {
			errs = append(errs, fmt.Errorf("server.tls.key_file is required when tls mode is manual"))
		}
	default:
		errs = append(errs, fmt.Errorf("server.tls.mode must be off, auto, or manual"))
	}

	// Server timeout validation
	if cfg.Server.ReadTimeout < time.Second {
		errs = append(errs, fmt.Errorf("server.read_timeout must be at least 1s"))
	}
	if cfg.Server.WriteTimeout < time.Second {
		errs = append(errs, fmt.Errorf("server.write_timeout must be at least 1s"))
	}
	if cfg.Server.IdleTimeout < time.Second {
		errs = append(errs, fmt.Errorf("server.idle_timeout must be at least 1s"))
	}

	// Database validation
	if cfg.Database.Path == "" {
		errs = append(errs, fmt.Errorf("database.path is required"))
	}
	if cfg.Database.MaxOpenConns < 1 {
		errs = append(errs, fmt.Errorf("database.max_open_conns must be at least 1"))
	}
	if cfg.Database.BusyTimeout < 0 {
		errs = append(errs, fmt.Errorf("database.busy_timeout must be at least 0"))
	}
	if cfg.Database.MmapSize < 0 {
		errs = append(errs, fmt.Errorf("database.mmap_size must be at least 0"))
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

	// Rate limit validation (only when enabled)
	if cfg.RateLimit.Enabled {
		for _, ep := range []struct {
			name string
			cfg  RateLimitEndpoint
		}{
			{"rate_limit.login", cfg.RateLimit.Login},
			{"rate_limit.register", cfg.RateLimit.Register},
			{"rate_limit.forgot_password", cfg.RateLimit.ForgotPassword},
			{"rate_limit.reset_password", cfg.RateLimit.ResetPassword},
		} {
			if ep.cfg.Limit < 1 {
				errs = append(errs, fmt.Errorf("%s.limit must be at least 1", ep.name))
			}
			if ep.cfg.Window < time.Second {
				errs = append(errs, fmt.Errorf("%s.window must be at least 1s", ep.name))
			}
		}
	}

	// SSE validation
	if cfg.SSE.HeartbeatInterval < 5*time.Second {
		errs = append(errs, fmt.Errorf("sse.heartbeat_interval must be at least 5s"))
	}
	if cfg.SSE.ClientBufferSize < 16 {
		errs = append(errs, fmt.Errorf("sse.client_buffer_size must be at least 16"))
	}

	// Telemetry validation (only when enabled)
	if cfg.Telemetry.Enabled {
		if cfg.Telemetry.Endpoint == "" {
			errs = append(errs, fmt.Errorf("telemetry.endpoint is required when telemetry is enabled"))
		}
		switch cfg.Telemetry.Protocol {
		case "grpc", "http":
			// valid
		default:
			errs = append(errs, fmt.Errorf("telemetry.protocol must be one of: grpc, http"))
		}
		if cfg.Telemetry.SampleRate < 0 || cfg.Telemetry.SampleRate > 1 {
			errs = append(errs, fmt.Errorf("telemetry.sample_rate must be between 0.0 and 1.0"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

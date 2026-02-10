package config

import "time"

type Config struct {
	Server    ServerConfig    `koanf:"server"`
	Database  DatabaseConfig  `koanf:"database"`
	Auth      AuthConfig      `koanf:"auth"`
	Files     FilesConfig     `koanf:"files"`
	Email     EmailConfig     `koanf:"email"`
	RateLimit RateLimitConfig `koanf:"rate_limit"`
	SSE       SSEConfig       `koanf:"sse"`
}

type ServerConfig struct {
	Host           string   `koanf:"host"`
	Port           int      `koanf:"port"`
	PublicURL      string   `koanf:"public_url"`
	AllowedOrigins []string `koanf:"allowed_origins"`
}

type DatabaseConfig struct {
	Path string `koanf:"path"`
}

type AuthConfig struct {
	SessionDuration time.Duration `koanf:"session_duration"`
	BcryptCost      int           `koanf:"bcrypt_cost"`
}

type FilesConfig struct {
	StoragePath   string `koanf:"storage_path"`
	MaxUploadSize int64  `koanf:"max_upload_size"`
	SigningSecret string `koanf:"signing_secret"`
}

type EmailConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	From     string `koanf:"from"`
}

type RateLimitConfig struct {
	Enabled        bool              `koanf:"enabled"`
	Login          RateLimitEndpoint `koanf:"login"`
	Register       RateLimitEndpoint `koanf:"register"`
	ForgotPassword RateLimitEndpoint `koanf:"forgot_password"`
	ResetPassword  RateLimitEndpoint `koanf:"reset_password"`
}

type RateLimitEndpoint struct {
	Limit  int           `koanf:"limit"`
	Window time.Duration `koanf:"window"`
}

type SSEConfig struct {
	EventRetention  time.Duration `koanf:"event_retention"`
	CleanupInterval time.Duration `koanf:"cleanup_interval"`
}

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			PublicURL:      "http://localhost:8080",
			AllowedOrigins: []string{"http://localhost:3000"},
		},
		Database: DatabaseConfig{
			Path: "./data/feather.db",
		},
		Auth: AuthConfig{
			SessionDuration: 720 * time.Hour, // 30 days
			BcryptCost:      12,
		},
		Files: FilesConfig{
			StoragePath:   "./data/uploads",
			MaxUploadSize: 10 * 1024 * 1024, // 10MB
		},
		Email: EmailConfig{
			Enabled: false,
			Port:    587,
		},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			Login:          RateLimitEndpoint{Limit: 10, Window: time.Minute},
			Register:       RateLimitEndpoint{Limit: 5, Window: time.Hour},
			ForgotPassword: RateLimitEndpoint{Limit: 5, Window: 15 * time.Minute},
			ResetPassword:  RateLimitEndpoint{Limit: 10, Window: 15 * time.Minute},
		},
		SSE: SSEConfig{
			EventRetention:  24 * time.Hour,
			CleanupInterval: time.Hour,
		},
	}
}

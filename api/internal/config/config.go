package config

import "time"

type Config struct {
	Log               LogConfig              `koanf:"log"`
	Server            ServerConfig           `koanf:"server"`
	Database          DatabaseConfig         `koanf:"database"`
	Auth              AuthConfig             `koanf:"auth"`
	Storage           StorageConfig          `koanf:"storage"`
	Email             EmailConfig            `koanf:"email"`
	RateLimit         RateLimitConfig        `koanf:"rate_limit"`
	SSE               SSEConfig              `koanf:"sse"`
	PushNotifications PushNotificationConfig `koanf:"push_notifications"`
	Telemetry         TelemetryConfig        `koanf:"telemetry"`
}

type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

type ServerConfig struct {
	Host           string        `koanf:"host"`
	Port           int           `koanf:"port"`
	PublicURL      string        `koanf:"public_url"`
	AllowedOrigins []string      `koanf:"allowed_origins"`
	TLS            TLSConfig     `koanf:"tls"`
	ReadTimeout    time.Duration `koanf:"read_timeout"`
	WriteTimeout   time.Duration `koanf:"write_timeout"`
	IdleTimeout    time.Duration `koanf:"idle_timeout"`
}

type TLSConfig struct {
	Mode     string        `koanf:"mode"`      // "off", "auto", "manual"
	CertFile string        `koanf:"cert_file"` // manual mode
	KeyFile  string        `koanf:"key_file"`  // manual mode
	Auto     AutoTLSConfig `koanf:"auto"`
}

type AutoTLSConfig struct {
	Domain   string `koanf:"domain"`    // required for auto mode
	Email    string `koanf:"email"`     // Let's Encrypt contact email
	CacheDir string `koanf:"cache_dir"` // cert cache dir (default: ./data/certs)
}

type DatabaseConfig struct {
	Path         string `koanf:"path"`
	MaxOpenConns int    `koanf:"max_open_conns"`
	BusyTimeout  int    `koanf:"busy_timeout"`
	CacheSize    int    `koanf:"cache_size"`
	MmapSize     int64  `koanf:"mmap_size"`
}

type AuthConfig struct {
	SessionDuration time.Duration `koanf:"session_duration"`
	BcryptCost      int           `koanf:"bcrypt_cost"`
}

type StorageConfig struct {
	Type          string      `koanf:"type"` // "off", "local", or "s3"
	MaxUploadSize int64       `koanf:"max_upload_size"`
	Local         LocalConfig `koanf:"local"`
	S3            S3Config    `koanf:"s3"`
}

type LocalConfig struct {
	Path          string `koanf:"path"`
	SigningSecret string `koanf:"signing_secret"`
}

type S3Config struct {
	Endpoint  string `koanf:"endpoint"`
	Bucket    string `koanf:"bucket"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Region    string `koanf:"region"`
	PathStyle bool   `koanf:"path_style"`
	UseSSL    bool   `koanf:"use_ssl"`
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
	Enabled             bool              `koanf:"enabled"`
	Login               RateLimitEndpoint `koanf:"login"`
	Register            RateLimitEndpoint `koanf:"register"`
	ForgotPassword      RateLimitEndpoint `koanf:"forgot_password"`
	ResetPassword       RateLimitEndpoint `koanf:"reset_password"`
	VerifyEmail         RateLimitEndpoint `koanf:"verify_email"`
	ResendVerification  RateLimitEndpoint `koanf:"resend_verification"`
	DeviceTokenRegister RateLimitEndpoint `koanf:"device_token_register"`
}

type RateLimitEndpoint struct {
	Limit  int           `koanf:"limit"`
	Window time.Duration `koanf:"window"`
}

type SSEConfig struct {
	EventRetention    time.Duration `koanf:"event_retention"`
	CleanupInterval   time.Duration `koanf:"cleanup_interval"`
	HeartbeatInterval time.Duration `koanf:"heartbeat_interval"`
	ClientBufferSize  int           `koanf:"client_buffer_size"`
}

type PushNotificationConfig struct {
	Enabled        bool   `koanf:"enabled"`
	RelayURL       string `koanf:"relay_url"`
	IncludePreview bool   `koanf:"include_preview"`
}

type TelemetryConfig struct {
	Enabled          bool              `koanf:"enabled"`
	Endpoint         string            `koanf:"endpoint"`
	Protocol         string            `koanf:"protocol"`          // "grpc" or "http"
	Insecure         bool              `koanf:"insecure"`          // use plaintext (no TLS) for OTLP export
	SampleRate       float64           `koanf:"sample_rate"`       // 0.0 to 1.0
	ServiceName      string            `koanf:"service_name"`      // default "enzyme"
	Headers          map[string]string `koanf:"headers"`           // OTLP exporter headers (e.g. auth keys)
	Traces           bool              `koanf:"traces"`            // export traces (default true)
	Metrics          bool              `koanf:"metrics"`           // export metrics (default true)
	Logs             bool              `koanf:"logs"`              // export logs (default true)
	FrontendEndpoint string            `koanf:"frontend_endpoint"` // OTLP/HTTP endpoint for browser trace proxy (auto-derived if empty)
}

func Defaults() *Config {
	return &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			PublicURL:      "http://localhost:8080",
			AllowedOrigins: []string{"http://localhost:3000"},
			TLS: TLSConfig{
				Mode: "off",
				Auto: AutoTLSConfig{
					CacheDir: "./data/certs",
				},
			},
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Database: DatabaseConfig{
			Path:         "./data/enzyme.db",
			MaxOpenConns: 2,
			BusyTimeout:  5000,
			CacheSize:    -2000,
		},
		Auth: AuthConfig{
			SessionDuration: 720 * time.Hour, // 30 days
			BcryptCost:      12,
		},
		Storage: StorageConfig{
			Type:          "local",
			MaxUploadSize: 10 * 1024 * 1024, // 10MB
			Local: LocalConfig{
				Path: "./data/uploads",
			},
			S3: S3Config{
				UseSSL: true,
			},
		},
		Email: EmailConfig{
			Enabled: false,
			Port:    587,
		},
		RateLimit: RateLimitConfig{
			Enabled:             true,
			Login:               RateLimitEndpoint{Limit: 10, Window: time.Minute},
			Register:            RateLimitEndpoint{Limit: 5, Window: time.Hour},
			ForgotPassword:      RateLimitEndpoint{Limit: 5, Window: 15 * time.Minute},
			ResetPassword:       RateLimitEndpoint{Limit: 10, Window: 15 * time.Minute},
			VerifyEmail:         RateLimitEndpoint{Limit: 10, Window: 15 * time.Minute},
			ResendVerification:  RateLimitEndpoint{Limit: 5, Window: time.Hour},
			DeviceTokenRegister: RateLimitEndpoint{Limit: 10, Window: time.Minute},
		},
		SSE: SSEConfig{
			EventRetention:    24 * time.Hour,
			CleanupInterval:   time.Hour,
			HeartbeatInterval: 30 * time.Second,
			ClientBufferSize:  256,
		},
		PushNotifications: PushNotificationConfig{
			Enabled:        false,
			RelayURL:       "https://push.enzyme.im",
			IncludePreview: true,
		},
		Telemetry: TelemetryConfig{
			Enabled:     false,
			Endpoint:    "localhost:4317",
			Protocol:    "grpc",
			Insecure:    true,
			SampleRate:  1.0,
			ServiceName: "enzyme",
			Traces:      true,
			Metrics:     true,
			Logs:        true,
		},
	}
}

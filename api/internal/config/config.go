package config

import "time"

type Config struct {
	Server   ServerConfig   `koanf:"server"`
	Database DatabaseConfig `koanf:"database"`
	Auth     AuthConfig     `koanf:"auth"`
	Files    FilesConfig    `koanf:"files"`
	Email    EmailConfig    `koanf:"email"`
}

type ServerConfig struct {
	Host      string `koanf:"host"`
	Port      int    `koanf:"port"`
	PublicURL string `koanf:"public_url"`
}

func (s ServerConfig) Addr() string {
	return s.Host + ":" + string(rune(s.Port+'0'))
}

type DatabaseConfig struct {
	Path string `koanf:"path"`
}

type AuthConfig struct {
	SessionDuration time.Duration `koanf:"session_duration"`
	SecureCookies   bool          `koanf:"secure_cookies"`
	BcryptCost      int           `koanf:"bcrypt_cost"`
}

type FilesConfig struct {
	StoragePath   string `koanf:"storage_path"`
	MaxUploadSize int64  `koanf:"max_upload_size"`
}

type EmailConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	From     string `koanf:"from"`
}

func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:      "0.0.0.0",
			Port:      8080,
			PublicURL: "http://localhost:8080",
		},
		Database: DatabaseConfig{
			Path: "./data/feather.db",
		},
		Auth: AuthConfig{
			SessionDuration: 720 * time.Hour, // 30 days
			SecureCookies:   false,
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
	}
}

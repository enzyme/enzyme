package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

func Load(configPath string, flags *pflag.FlagSet) (*Config, error) {
	k := koanf.New(".")

	// 1. Load defaults
	defaults := Defaults()
	if err := k.Load(defaultsProvider(defaults), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. Load from config file if it exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("loading config file: %w", err)
			}
		}
	} else {
		// Try default config paths
		for _, path := range []string{"config.yaml", "config.yml"} {
			if _, err := os.Stat(path); err == nil {
				if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
					return nil, fmt.Errorf("loading config file: %w", err)
				}
				break
			}
		}
	}

	// 3. Load from environment variables (FEATHER_ prefix)
	if err := k.Load(env.Provider("FEATHER_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "FEATHER_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	// 4. Load from CLI flags
	if flags != nil {
		if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
			return nil, fmt.Errorf("loading flags: %w", err)
		}
	}

	// 5. Unmarshal into struct
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "koanf",
	}); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// 6. Validate
	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

type defaultsProviderStruct struct {
	defaults *Config
}

func defaultsProvider(defaults *Config) *defaultsProviderStruct {
	return &defaultsProviderStruct{defaults: defaults}
}

func (d *defaultsProviderStruct) ReadBytes() ([]byte, error) {
	return nil, nil
}

func (d *defaultsProviderStruct) Read() (map[string]interface{}, error) {
	return map[string]interface{}{
		"server": map[string]interface{}{
			"host":       d.defaults.Server.Host,
			"port":       d.defaults.Server.Port,
			"public_url": d.defaults.Server.PublicURL,
		},
		"database": map[string]interface{}{
			"path": d.defaults.Database.Path,
		},
		"auth": map[string]interface{}{
			"session_duration": d.defaults.Auth.SessionDuration.String(),
			"bcrypt_cost":      d.defaults.Auth.BcryptCost,
		},
		"files": map[string]interface{}{
			"storage_path":    d.defaults.Files.StoragePath,
			"max_upload_size": d.defaults.Files.MaxUploadSize,
			"signing_secret":  d.defaults.Files.SigningSecret,
		},
		"email": map[string]interface{}{
			"enabled":  d.defaults.Email.Enabled,
			"host":     d.defaults.Email.Host,
			"port":     d.defaults.Email.Port,
			"username": d.defaults.Email.Username,
			"password": d.defaults.Email.Password,
			"from":     d.defaults.Email.From,
		},
		"rate_limit": map[string]interface{}{
			"enabled": d.defaults.RateLimit.Enabled,
			"login": map[string]interface{}{
				"limit":  d.defaults.RateLimit.Login.Limit,
				"window": d.defaults.RateLimit.Login.Window.String(),
			},
			"register": map[string]interface{}{
				"limit":  d.defaults.RateLimit.Register.Limit,
				"window": d.defaults.RateLimit.Register.Window.String(),
			},
			"forgot_password": map[string]interface{}{
				"limit":  d.defaults.RateLimit.ForgotPassword.Limit,
				"window": d.defaults.RateLimit.ForgotPassword.Window.String(),
			},
			"reset_password": map[string]interface{}{
				"limit":  d.defaults.RateLimit.ResetPassword.Limit,
				"window": d.defaults.RateLimit.ResetPassword.Window.String(),
			},
		},
		"sse": map[string]interface{}{
			"event_retention":  d.defaults.SSE.EventRetention.String(),
			"cleanup_interval": d.defaults.SSE.CleanupInterval.String(),
		},
	}, nil
}

func SetupFlags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("feather", pflag.ContinueOnError)
	flags.String("config", "", "Path to config file")
	flags.String("server.host", "", "Server host")
	flags.Int("server.port", 0, "Server port")
	flags.String("server.public_url", "", "Public URL")
	flags.String("database.path", "", "Database path")
	flags.Duration("auth.session_duration", 0, "Session duration")
	flags.String("files.storage_path", "", "File storage path")
	flags.Int64("files.max_upload_size", 0, "Max upload size in bytes")
	flags.Bool("email.enabled", false, "Enable email sending")
	return flags
}

func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

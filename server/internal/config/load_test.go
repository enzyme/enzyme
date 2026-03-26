package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_TLSFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
server:
  tls:
    mode: auto
    auto:
      domain: chat.example.com
      email: admin@example.com
      cache_dir: /var/lib/enzyme/certs
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.TLS.Mode != "auto" {
		t.Fatalf("expected mode 'auto', got %q", cfg.Server.TLS.Mode)
	}
	if cfg.Server.TLS.Auto.Domain != "chat.example.com" {
		t.Fatalf("expected domain 'chat.example.com', got %q", cfg.Server.TLS.Auto.Domain)
	}
	if cfg.Server.TLS.Auto.Email != "admin@example.com" {
		t.Fatalf("expected email 'admin@example.com', got %q", cfg.Server.TLS.Auto.Email)
	}
	if cfg.Server.TLS.Auto.CacheDir != "/var/lib/enzyme/certs" {
		t.Fatalf("expected cache_dir '/var/lib/enzyme/certs', got %q", cfg.Server.TLS.Auto.CacheDir)
	}
}

func TestLoad_TLSManualFromYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
server:
  tls:
    mode: manual
    cert_file: /etc/ssl/cert.pem
    key_file: /etc/ssl/key.pem
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.TLS.Mode != "manual" {
		t.Fatalf("expected mode 'manual', got %q", cfg.Server.TLS.Mode)
	}
	if cfg.Server.TLS.CertFile != "/etc/ssl/cert.pem" {
		t.Fatalf("expected cert_file '/etc/ssl/cert.pem', got %q", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "/etc/ssl/key.pem" {
		t.Fatalf("expected key_file '/etc/ssl/key.pem', got %q", cfg.Server.TLS.KeyFile)
	}
}

func TestLoad_EnvSimpleKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.yaml")

	t.Setenv("ENZYME_SERVER_PORT", "9090")

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func TestLoad_EnvUnderscoreInLeafKey(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.yaml")

	t.Setenv("ENZYME_DATABASE_MAX_OPEN_CONNS", "5")

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Database.MaxOpenConns != 5 {
		t.Fatalf("expected max_open_conns 5, got %d", cfg.Database.MaxOpenConns)
	}
}

func TestLoad_EnvDeepNestedUnderscore(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.yaml")

	t.Setenv("ENZYME_RATE_LIMIT_FORGOT_PASSWORD_LIMIT", "3")

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.RateLimit.ForgotPassword.Limit != 3 {
		t.Fatalf("expected forgot_password limit 3, got %d", cfg.RateLimit.ForgotPassword.Limit)
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yamlContent := `
database:
  max_open_conns: 10
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ENZYME_DATABASE_MAX_OPEN_CONNS", "25")

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Database.MaxOpenConns != 25 {
		t.Fatalf("expected env override max_open_conns 25, got %d", cfg.Database.MaxOpenConns)
	}
}

func TestLoad_TLSDefaultsWithoutYAML(t *testing.T) {
	// Load with no config file — should get defaults and pass validation
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.yaml")

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.TLS.Mode != "off" {
		t.Fatalf("expected default mode 'off', got %q", cfg.Server.TLS.Mode)
	}
	if cfg.Server.TLS.Auto.CacheDir != "./data/certs" {
		t.Fatalf("expected default cache_dir './data/certs', got %q", cfg.Server.TLS.Auto.CacheDir)
	}
}

func TestLoad_TLSFromFlags(t *testing.T) {
	flags := SetupFlags()
	if err := flags.Parse([]string{
		"--server.tls.mode=manual",
		"--server.tls.cert_file=/tmp/cert.pem",
		"--server.tls.key_file=/tmp/key.pem",
	}); err != nil {
		t.Fatal(err)
	}

	// Use a nonexistent config path so only defaults + flags apply
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.yaml")

	cfg, err := Load(cfgPath, flags)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.TLS.Mode != "manual" {
		t.Fatalf("expected mode 'manual', got %q", cfg.Server.TLS.Mode)
	}
	if cfg.Server.TLS.CertFile != "/tmp/cert.pem" {
		t.Fatalf("expected cert_file '/tmp/cert.pem', got %q", cfg.Server.TLS.CertFile)
	}
	if cfg.Server.TLS.KeyFile != "/tmp/key.pem" {
		t.Fatalf("expected key_file '/tmp/key.pem', got %q", cfg.Server.TLS.KeyFile)
	}
}

func TestLoad_TLSYAMLOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Only set mode and domain — cache_dir should keep its default
	yaml := `
server:
  tls:
    mode: auto
    auto:
      domain: example.com
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath, nil)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.TLS.Mode != "auto" {
		t.Fatalf("expected mode 'auto', got %q", cfg.Server.TLS.Mode)
	}
	if cfg.Server.TLS.Auto.Domain != "example.com" {
		t.Fatalf("expected domain 'example.com', got %q", cfg.Server.TLS.Auto.Domain)
	}
	// cache_dir should retain its default since YAML didn't override it
	if cfg.Server.TLS.Auto.CacheDir != "./data/certs" {
		t.Fatalf("expected default cache_dir './data/certs', got %q", cfg.Server.TLS.Auto.CacheDir)
	}
}

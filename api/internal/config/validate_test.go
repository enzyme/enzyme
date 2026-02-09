package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() *Config {
	cfg := Defaults()
	// Satisfy all existing validation rules
	cfg.Auth.BcryptCost = 12
	return cfg
}

func TestValidate_RateLimitDefaults(t *testing.T) {
	cfg := validConfig()
	if err := Validate(cfg); err != nil {
		t.Fatalf("defaults should be valid: %v", err)
	}
}

func TestValidate_RateLimitDisabled(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimit.Enabled = false
	cfg.RateLimit.Login.Limit = 0  // invalid, but should not matter when disabled
	cfg.RateLimit.Login.Window = 0 // invalid, but should not matter when disabled
	if err := Validate(cfg); err != nil {
		t.Fatalf("disabled rate limit should skip validation: %v", err)
	}
}

func TestValidate_RateLimitInvalidLimit(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimit.Login.Limit = 0

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "rate_limit.login.limit") {
		t.Fatalf("expected error about login limit, got: %v", err)
	}
}

func TestValidate_RateLimitInvalidWindow(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimit.Register.Window = 500 * time.Millisecond

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for sub-second window")
	}
	if !strings.Contains(err.Error(), "rate_limit.register.window") {
		t.Fatalf("expected error about register window, got: %v", err)
	}
}

func TestValidate_RateLimitMultipleErrors(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimit.Login.Limit = 0
	cfg.RateLimit.ForgotPassword.Window = 0

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	if !strings.Contains(msg, "rate_limit.login.limit") {
		t.Fatalf("expected login limit error, got: %v", err)
	}
	if !strings.Contains(msg, "rate_limit.forgot_password.window") {
		t.Fatalf("expected forgot_password window error, got: %v", err)
	}
}

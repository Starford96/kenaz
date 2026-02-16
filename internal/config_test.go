package internal

import (
	"strings"
	"testing"
)

func TestAuthConfig_DisabledMode(t *testing.T) {
	cfg := AuthConfig{Mode: "disabled", Token: ""}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled mode should pass: %v", err)
	}
	if cfg.AuthEnabled() {
		t.Error("disabled mode should not be enabled")
	}
}

func TestAuthConfig_EmptyModeDefaultsDisabled(t *testing.T) {
	cfg := AuthConfig{Mode: "", Token: ""}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty mode should default to disabled: %v", err)
	}
	if cfg.Mode != AuthModeDisabled {
		t.Errorf("mode = %q, want %q", cfg.Mode, AuthModeDisabled)
	}
}

func TestAuthConfig_TokenModeValid(t *testing.T) {
	cfg := AuthConfig{Mode: "token", Token: "mysecret"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("token mode with token should pass: %v", err)
	}
	if !cfg.AuthEnabled() {
		t.Error("token mode should be enabled")
	}
}

func TestAuthConfig_TokenModeEmptyToken(t *testing.T) {
	cfg := AuthConfig{Mode: "token", Token: ""}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("token mode with empty token should fail")
	}
	if !strings.Contains(err.Error(), "token is empty") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthConfig_InvalidMode(t *testing.T) {
	cfg := AuthConfig{Mode: "magic", Token: "x"}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("invalid mode should fail validation")
	}
}

func TestFullConfig_AuthValidationCalled(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Auth.Mode = "token"
	cfg.Auth.Token = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("full config validate should catch auth error")
	}
}

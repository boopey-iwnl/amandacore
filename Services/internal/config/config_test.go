package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsLocalDefaults(t *testing.T) {
	cfg := ServiceConfig{
		ServiceName:       "auth-service",
		Host:              "127.0.0.1",
		Port:              "8081",
		Environment:       "development",
		StoreBackend:      "file",
		StorePath:         t.TempDir() + "/platform-state.json",
		LocalSeedFile:     ".secrets/amandacore.dev.env",
		AdminSeedUsername: "amanda",
		AdminToolsEnabled: true,
		BuildID:           "test-build",
		WorldEndpoint:     "http://127.0.0.1:8085",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected local defaults to validate, got %v", err)
	}
}

func TestValidateRejectsUnsupportedBackend(t *testing.T) {
	cfg := validTestConfig()
	cfg.StoreBackend = "memory"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unsupported store backend") {
		t.Fatalf("expected unsupported backend error, got %v", err)
	}
}

func TestValidateRejectsInvalidWorldEndpoint(t *testing.T) {
	cfg := validTestConfig()
	cfg.WorldEndpoint = "127.0.0.1:8085"

	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "absolute http or https URL") {
		t.Fatalf("expected endpoint validation error, got %v", err)
	}
}

func TestValidateRejectsUnsafeProductionDefaults(t *testing.T) {
	cfg := validTestConfig()
	cfg.Environment = "production"
	cfg.AdminToolsEnabled = true
	cfg.AdminSeedUsername = "amanda"
	cfg.AdminSeedPassword = "password"
	cfg.LocalSeedFile = ".secrets/amandacore.dev.env"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected production config to be rejected")
	}
	text := err.Error()
	for _, expected := range []string{
		"admin tools must be disabled",
		"local dev seed file",
		"admin seed password is too weak",
		"admin seed username must not use the local default",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in validation error %q", expected, text)
		}
	}
}

func TestValidateAcceptsSQLiteWhenPathProvided(t *testing.T) {
	cfg := validTestConfig()
	cfg.StoreBackend = "sqlite"
	cfg.SQLitePath = t.TempDir() + "/amandacore.db"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected sqlite config to validate, got %v", err)
	}
}

func TestLoadDoesNotReadLocalSeedInProduction(t *testing.T) {
	seedPath := filepath.Join(t.TempDir(), "seed.env")
	if err := os.WriteFile(seedPath, []byte("AMANDACORE_ADMIN_SEED_PASSWORD=from_seed_file\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	t.Setenv("AMANDACORE_ENVIRONMENT", "production")
	t.Setenv("AMANDACORE_LOCAL_SEED_FILE", seedPath)
	t.Setenv("AMANDACORE_WORLD_ENDPOINT", "http://127.0.0.1:8085")

	cfg := Load("auth-service", "8081")
	if cfg.AdminSeedPassword == "from_seed_file" {
		t.Fatal("production load must not import local seed file values")
	}
}

func validTestConfig() ServiceConfig {
	return ServiceConfig{
		ServiceName:       "world-service",
		Host:              "127.0.0.1",
		Port:              "8085",
		Environment:       "development",
		StoreBackend:      "file",
		StorePath:         "platform-state.json",
		LocalSeedFile:     ".secrets/amandacore.dev.env",
		AdminSeedUsername: "amanda",
		BuildID:           "test-build",
		WorldEndpoint:     "http://127.0.0.1:8085",
	}
}

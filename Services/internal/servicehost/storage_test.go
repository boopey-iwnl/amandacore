package servicehost

import (
	"path/filepath"
	"strings"
	"testing"

	"amandacore/services/internal/config"
	"amandacore/services/internal/store/sqlstore"
)

func TestOpenPlatformStoreKeepsFileBackendAvailable(t *testing.T) {
	cfg := testConfig(t)
	fileStore, report, err := OpenPlatformStore(cfg)
	if err != nil {
		t.Fatalf("expected file backend to open, got %v", err)
	}
	if fileStore == nil {
		t.Fatal("expected file store")
	}
	if report.Backend != "file" || report.MigrationState == "" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestOpenPlatformStoreChecksSQLiteBeforeRuntimeRefusal(t *testing.T) {
	sqlitePath := filepath.Join(t.TempDir(), "amandacore.sqlite")
	sqliteStore, err := sqlstore.Open(sqlitePath)
	if err != nil {
		t.Fatalf("apply sqlite migrations: %v", err)
	}
	_ = sqliteStore.Close()

	cfg := testConfig(t)
	cfg.StoreBackend = "sqlite"
	cfg.SQLitePath = sqlitePath
	cfg.RequireMigrations = true
	_, report, err := OpenPlatformStore(cfg)
	if err == nil || !strings.Contains(err.Error(), "HTTP service adapters are not enabled") {
		t.Fatalf("expected explicit sqlite runtime refusal, got %v", err)
	}
	if report.MigrationState != "sqlite-migrations-current" || report.PendingCount != 0 {
		t.Fatalf("expected current sqlite migration state, got %#v", report)
	}
}

func TestOpenPlatformStoreBlocksPendingSQLiteWhenRequired(t *testing.T) {
	cfg := testConfig(t)
	cfg.StoreBackend = "sqlite"
	cfg.SQLitePath = filepath.Join(t.TempDir(), "pending.sqlite")
	cfg.RequireMigrations = true
	_, report, err := OpenPlatformStore(cfg)
	if err == nil || !strings.Contains(err.Error(), "sqlite migrations pending") {
		t.Fatalf("expected pending migration failure, got %v", err)
	}
	if report.MigrationState != "sqlite-migrations-pending" || report.PendingCount == 0 {
		t.Fatalf("expected pending sqlite migration state, got %#v", report)
	}
}

func testConfig(t *testing.T) config.ServiceConfig {
	t.Helper()
	return config.ServiceConfig{
		ServiceName:       "world-service",
		Host:              "127.0.0.1",
		Port:              "8085",
		Environment:       "local",
		StoreBackend:      "file",
		StorePath:         filepath.Join(t.TempDir(), "platform-state.json"),
		BuildID:           "test-build",
		WorldEndpoint:     "http://127.0.0.1:8085",
		AdminSeedUsername: "amanda",
	}
}

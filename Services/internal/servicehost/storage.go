package servicehost

import (
	"context"
	"fmt"

	"amandacore/services/internal/config"
	"amandacore/services/internal/store"
	"amandacore/services/internal/store/sqlstore"
)

type StorageStartupReport struct {
	Backend        string
	Environment    string
	MigrationState string
	PendingCount   int
}

func OpenPlatformStore(cfg config.ServiceConfig) (*store.FileStore, StorageStartupReport, error) {
	report := StorageStartupReport{
		Backend:     cfg.NormalizedStoreBackend(),
		Environment: cfg.NormalizedEnvironment(),
	}

	switch report.Backend {
	case "file":
		fileStore, err := store.NewFileStore(cfg.StorePath, cfg.BuildID, cfg.WorldEndpoint)
		if err != nil {
			return nil, report, err
		}
		report.MigrationState = "file-store-migrations-applied-on-open"
		return fileStore, report, nil
	case "sqlite":
		sqliteStore, err := sqlstore.OpenWithOptions(cfg.SQLitePath, sqlstore.OpenOptions{ApplyMigrations: false})
		if err != nil {
			return nil, report, fmt.Errorf("open sqlite store: %w", err)
		}
		defer sqliteStore.Close()

		migrator, err := sqliteStore.Migrator()
		if err != nil {
			return nil, report, fmt.Errorf("open sqlite migrator: %w", err)
		}
		check, err := migrator.Check(context.Background())
		if err != nil {
			return nil, report, fmt.Errorf("sqlite migration check failed: %w", err)
		}
		report.PendingCount = len(check.Pending)
		if len(check.Pending) > 0 {
			report.MigrationState = "sqlite-migrations-pending"
			if cfg.RequireMigrations {
				return nil, report, fmt.Errorf("sqlite migrations pending: %d; run dbmigrate --backend sqlite --sqlite <path>", len(check.Pending))
			}
		} else {
			report.MigrationState = "sqlite-migrations-current"
		}
		return nil, report, fmt.Errorf("sqlite runtime backend is migration-ready but HTTP service adapters are not enabled for this release candidate")
	default:
		return nil, report, fmt.Errorf("unsupported store backend %q", cfg.StoreBackend)
	}
}

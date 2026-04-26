package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
)

func TestMigratorAppliesAllMigrationsToEmptyDatabase(t *testing.T) {
	store := newTestStore(t)

	migrator, err := store.Migrator()
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	records, err := migrator.AppliedRecords(context.Background())
	if err != nil {
		t.Fatalf("failed to list applied migrations: %v", err)
	}
	migrations := migrator.Migrations()
	if len(records) != len(migrations) {
		t.Fatalf("expected %d applied migrations, got %d", len(migrations), len(records))
	}
	if records[0].ID != "000001" {
		t.Fatalf("expected first migration 000001, got %s", records[0].ID)
	}
}

func TestMigratorRerunIsNoop(t *testing.T) {
	store := newTestStore(t)

	migrator, err := store.Migrator()
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	result, err := migrator.Apply(context.Background())
	if err != nil {
		t.Fatalf("rerun failed: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("expected rerun to apply no migrations, got %d", len(result.Applied))
	}
	if len(result.AlreadyApplied) != len(migrator.Migrations()) {
		t.Fatalf("expected all migrations to be already applied, got %d", len(result.AlreadyApplied))
	}
}

func TestMigratorDetectsChecksumMismatch(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.DB().Exec(`UPDATE ac_schema_migrations SET checksum = 'changed' WHERE id = '000001'`); err != nil {
		t.Fatalf("failed to corrupt checksum: %v", err)
	}

	migrator, err := store.Migrator()
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	_, err = migrator.Apply(context.Background())
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestMigratorStatusReturnsAppliedMigrations(t *testing.T) {
	store := newTestStore(t)

	migrator, err := store.Migrator()
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	statuses, err := migrator.Status(context.Background())
	if err != nil {
		t.Fatalf("failed to get migration status: %v", err)
	}
	if len(statuses) != len(migrator.Migrations()) {
		t.Fatalf("expected status for each migration, got %d", len(statuses))
	}
	for _, status := range statuses {
		if !status.Applied || status.Record == nil {
			t.Fatalf("expected migration %s to be applied", status.Migration.ID)
		}
	}
}

func TestFailedMigrationDoesNotWriteMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "broken.sqlite"))
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	defer db.Close()
	if err := configureDB(db); err != nil {
		t.Fatalf("failed to configure sqlite db: %v", err)
	}

	migrator := &Migrator{
		db: db,
		migrations: []Migration{
			{
				ID:       "000001",
				Name:     "broken",
				Checksum: "broken-checksum",
				SQL:      "CREATE TABLE ac_schema_migrations (id TEXT PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at INTEGER NOT NULL, duration_ms INTEGER NOT NULL); CREATE TABLE broken_table (",
			},
		},
	}
	if _, err := migrator.Apply(context.Background()); err == nil {
		t.Fatal("expected broken migration to fail")
	}
	exists, err := migrator.schemaMigrationTableExists(context.Background())
	if err != nil {
		t.Fatalf("failed to check metadata table: %v", err)
	}
	if exists {
		t.Fatal("expected broken transaction to roll back metadata table")
	}
}

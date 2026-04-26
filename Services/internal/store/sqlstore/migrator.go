package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

var ErrChecksumMismatch = errors.New("sql migration checksum mismatch")

type Migration struct {
	ID       string
	Name     string
	Path     string
	Checksum string
	SQL      string
}

type MigrationRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Checksum   string `json:"checksum"`
	AppliedAt  int64  `json:"appliedAt"`
	DurationMs int64  `json:"durationMs"`
}

type MigrationStatus struct {
	Migration Migration
	Applied   bool
	Record    *MigrationRecord
}

type MigrationResult struct {
	Applied        []MigrationRecord
	AlreadyApplied []MigrationRecord
}

type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

func NewMigrator(db *sql.DB) (*Migrator, error) {
	migrations, err := loadEmbeddedMigrations()
	if err != nil {
		return nil, err
	}
	return &Migrator{db: db, migrations: migrations}, nil
}

func (m *Migrator) Migrations() []Migration {
	return append([]Migration(nil), m.migrations...)
}

func (m *Migrator) Apply(ctx context.Context) (MigrationResult, error) {
	applied, err := m.appliedRecords(ctx)
	if err != nil {
		return MigrationResult{}, err
	}

	result := MigrationResult{}
	for _, migration := range m.migrations {
		if record, ok := applied[migration.ID]; ok {
			if record.Checksum != migration.Checksum {
				return result, fmt.Errorf("%w: %s", ErrChecksumMismatch, migration.ID)
			}
			result.AlreadyApplied = append(result.AlreadyApplied, record)
			continue
		}

		record, err := m.applyOne(ctx, migration)
		if err != nil {
			return result, err
		}
		result.Applied = append(result.Applied, record)
		applied[migration.ID] = record
	}
	return result, nil
}

func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	applied, err := m.appliedRecords(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]MigrationStatus, 0, len(m.migrations))
	for _, migration := range m.migrations {
		status := MigrationStatus{Migration: migration}
		if record, ok := applied[migration.ID]; ok {
			copied := record
			status.Applied = true
			status.Record = &copied
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (m *Migrator) AppliedRecords(ctx context.Context) ([]MigrationRecord, error) {
	applied, err := m.appliedRecords(ctx)
	if err != nil {
		return nil, err
	}

	records := make([]MigrationRecord, 0, len(applied))
	for _, record := range applied {
		records = append(records, record)
	}
	sort.Slice(records, func(left int, right int) bool {
		return records[left].ID < records[right].ID
	})
	return records, nil
}

func (m *Migrator) applyOne(ctx context.Context, migration Migration) (MigrationRecord, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return MigrationRecord{}, err
	}
	defer tx.Rollback()

	started := time.Now()
	if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
		return MigrationRecord{}, fmt.Errorf("apply %s: %w", migration.ID, err)
	}

	record := MigrationRecord{
		ID:         migration.ID,
		Name:       migration.Name,
		Checksum:   migration.Checksum,
		AppliedAt:  time.Now().Unix(),
		DurationMs: time.Since(started).Milliseconds(),
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ac_schema_migrations (id, name, checksum, applied_at, duration_ms) VALUES (?, ?, ?, ?, ?)`,
		record.ID,
		record.Name,
		record.Checksum,
		record.AppliedAt,
		record.DurationMs); err != nil {
		return MigrationRecord{}, fmt.Errorf("record %s: %w", migration.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return MigrationRecord{}, err
	}
	return record, nil
}

func (m *Migrator) appliedRecords(ctx context.Context) (map[string]MigrationRecord, error) {
	exists, err := m.schemaMigrationTableExists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]MigrationRecord{}, nil
	}

	rows, err := m.db.QueryContext(ctx, `SELECT id, name, checksum, applied_at, duration_ms FROM ac_schema_migrations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := map[string]MigrationRecord{}
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Checksum, &record.AppliedAt, &record.DurationMs); err != nil {
			return nil, err
		}
		records[record.ID] = record
	}
	return records, rows.Err()
}

func (m *Migrator) schemaMigrationTableExists(ctx context.Context) (bool, error) {
	var count int
	err := m.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'ac_schema_migrations'`).Scan(&count)
	return count > 0, err
}

func loadEmbeddedMigrations() ([]Migration, error) {
	entries, err := embeddedMigrations.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	migrations := make([]Migration, 0, len(names))
	for _, name := range names {
		migrationPath := path.Join("migrations", name)
		payload, err := embeddedMigrations.ReadFile(migrationPath)
		if err != nil {
			return nil, err
		}

		id, label := migrationIDAndName(name)
		sum := sha256.Sum256(payload)
		migrations = append(migrations, Migration{
			ID:       id,
			Name:     label,
			Path:     migrationPath,
			Checksum: hex.EncodeToString(sum[:]),
			SQL:      string(payload),
		})
	}
	return migrations, nil
}

func migrationIDAndName(fileName string) (string, string) {
	base := strings.TrimSuffix(fileName, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 1 {
		return parts[0], parts[0]
	}
	return parts[0], parts[1]
}

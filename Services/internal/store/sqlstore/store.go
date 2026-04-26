package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"amandacore/services/internal/platform"
)

type Store struct {
	db            *sql.DB
	buildManifest platform.BuildManifest
	now           func() time.Time
}

type Tx struct {
	tx    *sql.Tx
	store *Store
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := configureDB(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	store := New(db)
	migrator, err := NewMigrator(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := migrator.Apply(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func New(db *sql.DB) *Store {
	return &Store{
		db:  db,
		now: time.Now,
		buildManifest: platform.BuildManifest{
			ID:                "amandacore-sqlstore-test",
			Channel:           "test",
			DisplayVersion:    "sqlstore-test",
			AllowedForLogin:   true,
			WorldEndpointHint: "http://127.0.0.1:8085",
			RequiredServices:  []string{"auth-service", "realm-service", "character-service", "world-service"},
		},
	}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) Migrator() (*Migrator, error) {
	return NewMigrator(s.db)
}

func (s *Store) WithTransaction(operation string, fn func(*Tx) error) error {
	if fn == nil {
		return fmt.Errorf("transaction function is required")
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	wrapped := &Tx{tx: tx, store: s}
	if err := fn(wrapped); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func configureDB(db *sql.DB) error {
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	return nil
}

func randomID(prefix string) string {
	return prefix + "_" + randomToken()
}

func randomToken() string {
	buffer := make([]byte, 24)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value != 0
}

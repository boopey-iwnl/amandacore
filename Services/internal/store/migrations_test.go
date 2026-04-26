package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStateMigrationsDryRunDoesNotPersistRecords(t *testing.T) {
	fileStore := &FileStore{state: state{MigrationHistory: map[string]MigrationRecord{}}}
	result, err := fileStore.applyMigrationsLocked(MigrationOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) != len(CurrentMigrationIDs()) {
		t.Fatalf("expected dry-run to validate all migrations, got %#v", result)
	}
	if len(fileStore.state.MigrationHistory) != 0 {
		t.Fatalf("dry-run persisted migration history: %#v", fileStore.state.MigrationHistory)
	}
}

func TestStateMigrationsApplyInOrder(t *testing.T) {
	fileStore := &FileStore{state: state{MigrationHistory: map[string]MigrationRecord{}}}
	result, err := fileStore.applyMigrationsLocked(MigrationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ids := CurrentMigrationIDs()
	if len(result.Applied) != len(ids) {
		t.Fatalf("expected %d applied migrations, got %#v", len(ids), result)
	}
	for index, id := range ids {
		if result.Applied[index].ID != id {
			t.Fatalf("migration order mismatch at %d: got %s want %s", index, result.Applied[index].ID, id)
		}
	}
}

func TestMigrationChecksumMismatchFails(t *testing.T) {
	ids := CurrentMigrationIDs()
	fileStore := &FileStore{state: state{MigrationHistory: map[string]MigrationRecord{
		ids[0]: {ID: ids[0], Checksum: "changed"},
	}}}
	_, err := fileStore.applyMigrationsLocked(MigrationOptions{})
	if !errors.Is(err, ErrMigrationChecksumMismatch) {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestNewFileStorePersistsMigrationHistory(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := NewFileStore(storePath, "test-build", "http://localhost:8085")
	if err != nil {
		t.Fatal(err)
	}
	history, err := fileStore.MigrationHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != len(CurrentMigrationIDs()) {
		t.Fatalf("expected migration history records, got %#v", history)
	}
}

func TestNewFileStoreWithOptionsCanOpenReadOnlyForDryRun(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	_, err := NewFileStoreWithOptions(storePath, "test-build", "http://localhost:8085", FileStoreOpenOptions{
		ApplyMigrations: false,
		SaveOnOpen:      false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storePath); err == nil {
		t.Fatalf("read-only open created store file")
	}
}

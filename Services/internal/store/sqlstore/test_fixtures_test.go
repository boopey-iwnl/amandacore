package sqlstore

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "amandacore-test.sqlite"))
	if err != nil {
		t.Fatalf("failed to open sql store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("failed to close sql store: %v", err)
		}
	})
	return store
}

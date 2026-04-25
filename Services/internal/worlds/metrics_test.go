package worlds

import (
	"path/filepath"
	"testing"
	"time"

	"amandacore/services/internal/store"
)

func TestCleanupStaleSessionsRemovesTokenAndRecordsMetric(t *testing.T) {
	fileStore, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"), "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	server := newWorldServer(fileStore)
	server.sessionsByToken["world_stale"] = &worldSessionState{
		Token:       "world_stale",
		AccountID:   "acct_stale",
		CharacterID: "char_stale",
		RealmID:     "sunset-frontier-dev",
		ZoneID:      defaultZoneID,
		Connected:   true,
		LastSeenAt:  time.Now().Add(-worldSessionStaleAfter - time.Second).Unix(),
	}
	server.sessionTokenByChar["char_stale"] = "world_stale"

	server.cleanupStaleSessionsLocked(time.Now())

	if _, exists := server.sessionsByToken["world_stale"]; exists {
		t.Fatalf("expected stale session to be removed")
	}
	if _, exists := server.sessionTokenByChar["char_stale"]; exists {
		t.Fatalf("expected character session index to be removed")
	}

	snapshot := server.metrics.snapshot(server.sessionCountsLocked())
	if snapshot["staleSessionsDropped"].(int64) != 1 {
		t.Fatalf("expected stale session metric to be 1, got %#v", snapshot["staleSessionsDropped"])
	}
}

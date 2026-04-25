package worlds

import (
	"path/filepath"
	"testing"
	"time"

	"amandacore/services/internal/store"
)

func TestDuelRequestRejectsSafeZone(t *testing.T) {
	server, _, alice, bob, _ := newGroupCreditTestServer(t)
	alice.ZoneID = defaultZoneID
	alice.X = starterSpawnX
	alice.Y = starterSpawnY
	bob.ZoneID = defaultZoneID
	bob.X = starterSpawnX + 2.0
	bob.Y = starterSpawnY

	if _, err := server.requestDuelLocked(alice, bob); err == nil {
		t.Fatalf("expected safe zone duel request to fail")
	}
}

func TestDuelLifecycleDamageIsolationAndStatsPersistence(t *testing.T) {
	server, storePath, alice, bob, cara := newGroupCreditTestServer(t)
	duel := requestAcceptAndStartTestDuel(t, server, alice, bob)

	if duel.State != duelStateActive {
		t.Fatalf("expected active duel, got %s", duel.State)
	}
	if err := server.applyDamageToPlayerLocked(alice, cara, 25.0, "test_nonparticipant"); err == nil {
		t.Fatalf("expected nonparticipant PvP damage to be rejected")
	}
	if cara.Health != cara.MaxHealth {
		t.Fatalf("expected nonparticipant health unchanged, got %.1f/%.1f", cara.Health, cara.MaxHealth)
	}

	if err := server.applyDamageToPlayerLocked(alice, bob, bob.MaxHealth*2.0, "test_duel_defeat"); err != nil {
		t.Fatalf("failed to apply duel damage: %v", err)
	}
	if server.findDuelForCharacterLocked(alice.CharacterID) != nil || server.findDuelForCharacterLocked(bob.CharacterID) != nil {
		t.Fatalf("expected duel indexes to clear after defeat")
	}
	if !bob.Alive || bob.Health != duelDefeatHealth {
		t.Fatalf("expected loser to remain alive at safe low health, got alive=%t health=%.1f", bob.Alive, bob.Health)
	}
	if alice.PvPStats.DuelsWon != 1 || alice.PvPStats.HonorPoints != 1 {
		t.Fatalf("expected winner stats to update, got %#v", alice.PvPStats)
	}
	if bob.PvPStats.DuelsLost != 1 {
		t.Fatalf("expected loser stats to update, got %#v", bob.PvPStats)
	}

	restartedStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	reloadedWinner, err := restartedStore.GetCharacterByID(alice.CharacterID)
	if err != nil {
		t.Fatalf("failed to reload winner: %v", err)
	}
	if reloadedWinner.PvPStats.DuelsWon != 1 || reloadedWinner.PvPStats.HonorPoints != 1 {
		t.Fatalf("expected persisted winner stats, got %#v", reloadedWinner.PvPStats)
	}
}

func TestDuelEndsOnDistanceTimeoutAndDisconnect(t *testing.T) {
	t.Run("distance timeout", func(t *testing.T) {
		server, _, alice, bob, _ := newGroupCreditTestServer(t)
		duel := requestAcceptAndStartTestDuel(t, server, alice, bob)

		alice.X = duel.CenterX + duel.MaxDistance + 2.0
		if err := server.advanceActiveDuelLocked(duel, nowMillis()); err != nil {
			t.Fatalf("failed first range advance: %v", err)
		}
		if server.findDuelForCharacterLocked(alice.CharacterID) == nil {
			t.Fatalf("expected duel to allow range grace")
		}
		if err := server.advanceActiveDuelLocked(duel, nowMillis()+duelOutOfBoundsMs+1); err != nil {
			t.Fatalf("failed second range advance: %v", err)
		}
		if server.findDuelForCharacterLocked(alice.CharacterID) != nil {
			t.Fatalf("expected duel to end after range grace")
		}
	})

	t.Run("disconnect", func(t *testing.T) {
		server, _, alice, bob, _ := newGroupCreditTestServer(t)
		duel := requestAcceptAndStartTestDuel(t, server, alice, bob)

		bob.Connected = false
		if err := server.advanceActiveDuelLocked(duel, nowMillis()); err != nil {
			t.Fatalf("failed disconnect advance: %v", err)
		}
		if server.findDuelForCharacterLocked(alice.CharacterID) != nil {
			t.Fatalf("expected duel to cancel on disconnect")
		}
		if alice.PvPStats.DuelsWon != 0 || bob.PvPStats.DuelsLost != 0 {
			t.Fatalf("expected disconnect cancel to avoid win/loss stats, alice=%#v bob=%#v", alice.PvPStats, bob.PvPStats)
		}
	})
}

func TestPvPStatsDefaultAndStoreUpdate(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	character := createTestCharacter(t, fileStore, "pvp_stats_user", "StatsUser")
	if character.PvPStats.CharacterID != character.ID {
		t.Fatalf("expected default stats character id, got %#v", character.PvPStats)
	}
}

func requestAcceptAndStartTestDuel(t *testing.T, server *worldServer, alice *worldSessionState, bob *worldSessionState) *duelState {
	t.Helper()

	duel, err := server.requestDuelLocked(alice, bob)
	if err != nil {
		t.Fatalf("failed to request duel: %v", err)
	}
	duel, err = server.acceptDuelLocked(bob, duel.DuelID)
	if err != nil {
		t.Fatalf("failed to accept duel: %v", err)
	}
	duel.CountdownEndsAtMs = nowMillis() - 1
	if err := server.advanceDuelsLocked(time.Now()); err != nil {
		t.Fatalf("failed to advance duel: %v", err)
	}
	return duel
}

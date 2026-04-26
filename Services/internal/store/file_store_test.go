package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"amandacore/services/internal/platform"
)

func TestAdminSeedAndLogin(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	if err := fileStore.EnsureAdminSeed("amanda", "amanda"); err != nil {
		t.Fatalf("failed to seed admin account: %v", err)
	}

	account, err := fileStore.Authenticate("amanda", "amanda")
	if err != nil {
		t.Fatalf("failed to authenticate seeded admin: %v", err)
	}

	if account.Username != "amanda" {
		t.Fatalf("unexpected username: %s", account.Username)
	}

	session, err := fileStore.CreateSession(account.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if _, err := fileStore.ValidateAccessToken(session.AccessToken); err != nil {
		t.Fatalf("failed to validate session token: %v", err)
	}

	if err := fileStore.EnsureAdminSeed("amanda", "amanda"); err != nil {
		t.Fatalf("failed to re-seed admin account: %v", err)
	}
}

func TestCharacterCreationAndJoinTicket(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	account, err := fileStore.RegisterAccount("player_one", "secret")
	if err != nil {
		t.Fatalf("failed to register account: %v", err)
	}

	session, err := fileStore.CreateSession(account.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	character, err := fileStore.CreateCharacter(
		account.ID,
		"sunset-frontier-dev",
		"Lark",
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character: %v", err)
	}

	if character.RaceID != platform.DefaultRaceID {
		t.Fatalf("expected race %s, got %s", platform.DefaultRaceID, character.RaceID)
	}
	if character.ClassID != platform.DefaultClassID {
		t.Fatalf("expected class %s, got %s", platform.DefaultClassID, character.ClassID)
	}
	if character.ArchetypeID != platform.LegacyWayfarerArchetypeID {
		t.Fatalf("expected archetype %s, got %s", platform.LegacyWayfarerArchetypeID, character.ArchetypeID)
	}
	if character.CurrencyCopper != platform.StarterCurrencyCopper {
		t.Fatalf("expected starter copper %d, got %d", platform.StarterCurrencyCopper, character.CurrencyCopper)
	}
	if len(character.LearnedAbilityIDs) != len(platform.DefaultStartingLearnedAbilityIDs()) {
		t.Fatalf("expected %d starting learned abilities, got %d", len(platform.DefaultStartingLearnedAbilityIDs()), len(character.LearnedAbilityIDs))
	}
	if character.LearnedAbilityIDs[0] != platform.AutoAttackAbilityID {
		t.Fatalf("expected auto attack as first learned ability, got %s", character.LearnedAbilityIDs[0])
	}

	ticket, err := fileStore.IssueWorldJoinTicket(account.ID, session.ID, character.ID, "sunset-frontier-dev")
	if err != nil {
		t.Fatalf("failed to issue world join ticket: %v", err)
	}

	if ticket.CharacterID != character.ID {
		t.Fatalf("join ticket character mismatch: %s != %s", ticket.CharacterID, character.ID)
	}

	if len(character.Inventory) != platform.InventorySlotCount {
		t.Fatalf("expected %d inventory slots, got %d", platform.InventorySlotCount, len(character.Inventory))
	}
	if character.Inventory[0].ItemID != "camp_ration" || character.Inventory[0].StackCount != 3 {
		t.Fatalf("expected starter ration in slot 0, got %#v", character.Inventory[0])
	}
	if character.Inventory[1].ItemID != "linen_wrap" || character.Inventory[1].StackCount != 2 {
		t.Fatalf("expected starter wrap in slot 1, got %#v", character.Inventory[1])
	}
}

func TestLegacyCharacterIdentityDefaultsPersistOnLoad(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	legacyState := map[string]any{
		"accounts": map[string]any{},
		"realms": map[string]any{
			"sunset-frontier-dev": map[string]any{
				"id":             "sunset-frontier-dev",
				"displayName":    "Sunset Frontier Dev",
				"region":         "local",
				"endpoint":       "http://127.0.0.1:8085",
				"supportedBuild": "test-build",
				"onlinePlayers":  0,
				"online":         true,
			},
		},
		"characters": map[string]any{
			"char_legacy": map[string]any{
				"id":             "char_legacy",
				"accountId":      "acct_legacy",
				"realmId":        "sunset-frontier-dev",
				"displayName":    "Legacy",
				"archetypeId":    platform.LegacyWayfarerArchetypeID,
				"level":          1,
				"experience":     0,
				"currencyCopper": 0,
				"zoneId":         "west_approach",
				"positionX":      12.0,
				"positionY":      12.0,
				"positionZ":      0.0,
				"inventory":      platform.DefaultStarterInventory(),
				"quests":         map[string]any{},
				"lastSeenAt":     1,
			},
		},
		"sessions":         map[string]any{},
		"worldJoinTickets": map[string]any{},
		"passwordReset":    map[string]any{},
		"buildManifest": map[string]any{
			"id":                "test-build",
			"channel":           "development",
			"displayVersion":    "test-build",
			"requiredServices":  []string{"auth-service"},
			"launcherNews":      "test",
			"allowedForLogin":   true,
			"worldEndpointHint": "http://127.0.0.1:8085",
		},
	}

	payload, err := json.MarshalIndent(legacyState, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal legacy state: %v", err)
	}
	if err := os.WriteFile(storePath, payload, 0o644); err != nil {
		t.Fatalf("failed to write legacy state: %v", err)
	}

	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to open store with legacy state: %v", err)
	}

	character, err := fileStore.GetCharacterByID("char_legacy")
	if err != nil {
		t.Fatalf("failed to load legacy character: %v", err)
	}
	if character.RaceID != platform.DefaultRaceID {
		t.Fatalf("expected legacy race %s, got %s", platform.DefaultRaceID, character.RaceID)
	}
	if character.ClassID != platform.DefaultClassID {
		t.Fatalf("expected legacy class %s, got %s", platform.DefaultClassID, character.ClassID)
	}
	if len(character.LearnedAbilityIDs) != len(platform.DefaultStartingLearnedAbilityIDs()) {
		t.Fatalf("expected normalized legacy abilities, got %d entries", len(character.LearnedAbilityIDs))
	}

	if _, err := fileStore.UpdateCharacterState("char_legacy", "west_approach", 14.0, 15.0, 0.0); err != nil {
		t.Fatalf("failed to persist normalized legacy character: %v", err)
	}

	rawState, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read normalized store: %v", err)
	}

	var decoded struct {
		Characters map[string]platform.Character `json:"characters"`
	}
	if err := json.Unmarshal(rawState, &decoded); err != nil {
		t.Fatalf("failed to decode normalized store: %v", err)
	}

	persisted, ok := decoded.Characters["char_legacy"]
	if !ok {
		t.Fatalf("expected normalized legacy character to persist")
	}
	if persisted.RaceID != platform.DefaultRaceID {
		t.Fatalf("expected persisted race %s, got %s", platform.DefaultRaceID, persisted.RaceID)
	}
	if persisted.ClassID != platform.DefaultClassID {
		t.Fatalf("expected persisted class %s, got %s", platform.DefaultClassID, persisted.ClassID)
	}
	if len(persisted.LearnedAbilityIDs) != len(platform.DefaultStartingLearnedAbilityIDs()) {
		t.Fatalf("expected persisted learned abilities, got %d", len(persisted.LearnedAbilityIDs))
	}
}

func TestUpdateCharacterProgressionPersistsLearnedAbilities(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	account, err := fileStore.RegisterAccount("trainer_player", "secret")
	if err != nil {
		t.Fatalf("failed to register account: %v", err)
	}

	character, err := fileStore.CreateCharacter(
		account.ID,
		"sunset-frontier-dev",
		"Trainer",
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character: %v", err)
	}

	updated, err := fileStore.UpdateCharacterProgression(
		character.ID,
		character.Experience,
		25,
		character.Inventory,
		append(character.LearnedAbilityIDs, platform.DrivingBlowAbilityID),
		character.ActionBarSlots,
		character.Quests,
	)
	if err != nil {
		t.Fatalf("failed to update character progression: %v", err)
	}

	if len(updated.LearnedAbilityIDs) != len(platform.DefaultStartingLearnedAbilityIDs())+1 {
		t.Fatalf("expected learned abilities to persist driving blow, got %#v", updated.LearnedAbilityIDs)
	}

	reopenedStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}

	reloaded, err := reopenedStore.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatalf("failed to load character after restart: %v", err)
	}

	foundDrivingBlow := false
	for _, abilityID := range reloaded.LearnedAbilityIDs {
		if abilityID == platform.DrivingBlowAbilityID {
			foundDrivingBlow = true
			break
		}
	}
	if !foundDrivingBlow {
		t.Fatalf("expected driving blow to persist after restart, got %#v", reloaded.LearnedAbilityIDs)
	}
}

func TestMultipleStoreInstancesPersistWithoutCorruption(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to create initial file store: %v", err)
	}

	if err := fileStore.EnsureAdminSeed("amanda", "amanda"); err != nil {
		t.Fatalf("failed to seed admin account: %v", err)
	}

	var waitGroup sync.WaitGroup
	errorChannel := make(chan error, 8)

	for index := 0; index < 8; index++ {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()

			workerStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
			if err != nil {
				errorChannel <- fmt.Errorf("worker %d failed to open store: %w", worker, err)
				return
			}

			if _, err := workerStore.RegisterAccount(fmt.Sprintf("player_%d", worker), "secret"); err != nil {
				errorChannel <- fmt.Errorf("worker %d failed to register account: %w", worker, err)
			}
		}(index)
	}

	waitGroup.Wait()
	close(errorChannel)

	for err := range errorChannel {
		if err != nil {
			t.Fatal(err)
		}
	}

	rawState, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("failed to read persisted store: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rawState, &decoded); err != nil {
		t.Fatalf("persisted store is not valid JSON: %v", err)
	}

	accounts, err := fileStore.ListAccounts()
	if err != nil {
		t.Fatalf("failed to list accounts after concurrent writes: %v", err)
	}

	if len(accounts) != 9 {
		t.Fatalf("expected 9 accounts after concurrent writes, got %d", len(accounts))
	}
}

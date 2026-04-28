package sqlstore

import (
	"errors"
	"testing"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func TestAccountSessionRealmCharacterAndTicketRoundTrip(t *testing.T) {
	store := newTestStore(t)

	realm, err := SeedDevRealm(store)
	if err != nil {
		t.Fatalf("failed to seed realm: %v", err)
	}
	account, err := store.RegisterAccount("sql_player", "secret")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	loadedAccount, err := store.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("failed to load account: %v", err)
	}
	if loadedAccount.Username != "sql_player" {
		t.Fatalf("expected username sql_player, got %s", loadedAccount.Username)
	}
	if _, err := store.Authenticate("sql_player", "secret"); err != nil {
		t.Fatalf("failed to authenticate sql account: %v", err)
	}

	session, err := store.CreateSession(account.ID)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if _, err := store.ValidateAccessToken(session.AccessToken); err != nil {
		t.Fatalf("failed to validate session: %v", err)
	}

	realms, err := store.ListRealms()
	if err != nil {
		t.Fatalf("failed to list realms: %v", err)
	}
	if len(realms) != 1 || realms[0].ID != realm.ID {
		t.Fatalf("expected seeded realm, got %#v", realms)
	}

	character, err := store.CreateCharacter(
		account.ID,
		realm.ID,
		"SqlRunner",
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character: %v", err)
	}
	characters, err := store.ListCharacters(account.ID, realm.ID)
	if err != nil {
		t.Fatalf("failed to list characters: %v", err)
	}
	if len(characters) != 1 || characters[0].ID != character.ID {
		t.Fatalf("expected created character, got %#v", characters)
	}
	selected, err := store.GetCharacterByName(realm.ID, "sqlrunner")
	if err != nil {
		t.Fatalf("failed to select character by name: %v", err)
	}
	if selected.ID != character.ID {
		t.Fatalf("expected selected character %s, got %s", character.ID, selected.ID)
	}

	ticket, err := store.IssueWorldJoinTicket(account.ID, session.ID, character.ID, realm.ID)
	if err != nil {
		t.Fatalf("failed to issue world join ticket: %v", err)
	}
	consumed, err := store.ConsumeWorldJoinTicket(ticket.TicketID)
	if err != nil {
		t.Fatalf("failed to consume world join ticket: %v", err)
	}
	if consumed.ConsumedAt == 0 {
		t.Fatal("expected consumed ticket timestamp")
	}
	if _, err := store.ConsumeWorldJoinTicket(ticket.TicketID); !errors.Is(err, filestore.ErrJoinTicketConsumed) {
		t.Fatalf("expected consumed ticket error, got %v", err)
	}

	worldSession, err := store.CreateWorldSession(filestore.WorldSessionRecord{
		AccountID:   account.ID,
		CharacterID: character.ID,
		RealmID:     realm.ID,
		ZoneID:      character.ZoneID,
		Connected:   true,
		PositionX:   character.PositionX,
		PositionY:   character.PositionY,
		PositionZ:   character.PositionZ,
	})
	if err != nil {
		t.Fatalf("failed to create world session: %v", err)
	}
	worldSession.ZoneID = "stonewake_vale"
	worldSession.PositionX = 12
	worldSession.PositionY = 14
	updatedWorldSession, err := store.UpdateWorldSession(worldSession)
	if err != nil {
		t.Fatalf("failed to update world session: %v", err)
	}
	if updatedWorldSession.PositionX != 12 || updatedWorldSession.PositionY != 14 {
		t.Fatalf("expected world session position round trip, got %#v", updatedWorldSession)
	}
}

func TestGameplayStateRoundTrips(t *testing.T) {
	store := newTestStore(t)
	realm, err := SeedDevRealm(store)
	if err != nil {
		t.Fatalf("failed to seed realm: %v", err)
	}
	account, err := SeedTestAccount(store, "state_player", "secret")
	if err != nil {
		t.Fatalf("failed to seed account: %v", err)
	}
	character, err := store.CreateCharacter(account.ID, realm.ID, "StateRunner", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create character: %v", err)
	}

	inventory := platform.DefaultStarterInventory()
	inventory[5] = platform.CharacterInventorySlot{SlotIndex: 5, ItemID: "test_relic", DisplayName: "Test Relic", StackCount: 2}
	if _, err := store.UpdateCharacterInventory(character.ID, inventory); err != nil {
		t.Fatalf("failed to update inventory: %v", err)
	}
	updatedInventory, err := store.GetCharacterInventory(character.ID)
	if err != nil {
		t.Fatalf("failed to load inventory: %v", err)
	}
	if updatedInventory[5].ItemID != "test_relic" || updatedInventory[5].StackCount != 2 {
		t.Fatalf("expected inventory slot round trip, got %#v", updatedInventory[5])
	}

	learned := append(character.LearnedAbilityIDs, platform.DrivingBlowAbilityID)
	if _, err := store.UpdateLearnedAbilities(character.ID, learned); err != nil {
		t.Fatalf("failed to update learned abilities: %v", err)
	}
	abilities, err := store.GetLearnedAbilities(character.ID)
	if err != nil {
		t.Fatalf("failed to load learned abilities: %v", err)
	}
	if !containsString(abilities, platform.DrivingBlowAbilityID) {
		t.Fatalf("expected learned ability round trip, got %#v", abilities)
	}

	actionBar := character.ActionBarSlots
	actionBar[7] = platform.CharacterActionBarSlot{SlotIndex: 7, AbilityID: platform.DrivingBlowAbilityID}
	if _, err := store.UpdateActionBarSlots(character.ID, actionBar); err != nil {
		t.Fatalf("failed to update action bar: %v", err)
	}
	actionSlots, err := store.GetActionBarSlots(character.ID)
	if err != nil {
		t.Fatalf("failed to load action bar: %v", err)
	}
	if actionSlots[7].AbilityID != platform.DrivingBlowAbilityID {
		t.Fatalf("expected action-bar slot round trip, got %#v", actionSlots[7])
	}

	quests := map[string]platform.CharacterQuestProgress{
		"quest_sql_foundation": {
			QuestID:      "quest_sql_foundation",
			State:        "active",
			CurrentCount: 1,
			TargetCount:  3,
			ObjectiveProgress: map[string]platform.CharacterQuestObjectiveProgress{
				"objective_sql": {NodeID: "objective_sql", Current: 1, Target: 3},
			},
		},
	}
	if _, err := store.UpdateCharacterQuestProgress(character.ID, quests); err != nil {
		t.Fatalf("failed to update quest progress: %v", err)
	}
	loadedQuests, err := store.GetCharacterQuestProgress(character.ID)
	if err != nil {
		t.Fatalf("failed to load quest progress: %v", err)
	}
	if loadedQuests["quest_sql_foundation"].ObjectiveProgress["objective_sql"].Current != 1 {
		t.Fatalf("expected quest progress round trip, got %#v", loadedQuests)
	}
}

func TestTransactionRollback(t *testing.T) {
	store := newTestStore(t)
	expectedErr := errors.New("rollback please")

	err := store.WithTransaction("test.rollback", func(tx *Tx) error {
		_, err := tx.CreateAccount(platform.Account{
			ID:           "acct_rollback",
			Username:     "rollback_user",
			PasswordHash: "test_hash",
			Roles:        []platform.Role{platform.RolePlayer},
			CreatedAt:    1,
			UpdatedAt:    1,
		})
		if err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if _, err := store.GetAccountByID("acct_rollback"); !errors.Is(err, filestore.ErrInvalidCredentials) {
		t.Fatalf("expected rolled back account to be absent, got %v", err)
	}
}

func TestDuplicateAccountAndCharacterConstraints(t *testing.T) {
	store := newTestStore(t)
	realm, err := SeedDevRealm(store)
	if err != nil {
		t.Fatalf("failed to seed realm: %v", err)
	}
	account, err := store.RegisterAccount("duplicate_user", "secret")
	if err != nil {
		t.Fatalf("failed to create account: %v", err)
	}
	if _, err := store.RegisterAccount(" DUPLICATE_USER ", "secret"); !errors.Is(err, filestore.ErrAccountExists) {
		t.Fatalf("expected duplicate account error, got %v", err)
	}

	if _, err := store.CreateCharacter(account.ID, realm.ID, "DuplicateHero", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID); err != nil {
		t.Fatalf("failed to create character: %v", err)
	}
	if _, err := store.CreateCharacter(account.ID, realm.ID, "duplicatehero", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID); !errors.Is(err, filestore.ErrCharacterNameExists) {
		t.Fatalf("expected duplicate character error, got %v", err)
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

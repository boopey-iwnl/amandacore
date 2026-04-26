package store

import (
	"errors"
	"path/filepath"
	"testing"

	"amandacore/services/internal/platform"
)

func TestCharacterTransactionCommitsAtomicUpdate(t *testing.T) {
	fileStore, character := newStoreWithCharacter(t)
	updated, err := fileStore.UpdateCharacterAtomically("test.quest_reward", character.ID, func(character *platform.Character) error {
		character.Experience += 25
		character.CurrencyCopper += 10
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Experience != 25 || updated.CurrencyCopper != platform.StarterCurrencyCopper+10 {
		t.Fatalf("unexpected committed character: %#v", updated)
	}
}

func TestCharacterTransactionRollsBackOnError(t *testing.T) {
	fileStore, character := newStoreWithCharacter(t)
	expectedCopper := character.CurrencyCopper
	errMarker := errors.New("fail reward")
	_, err := fileStore.UpdateCharacterAtomically("test.rollback", character.ID, func(character *platform.Character) error {
		character.CurrencyCopper += 999
		return errMarker
	})
	if !errors.Is(err, errMarker) {
		t.Fatalf("expected rollback marker, got %v", err)
	}
	reloaded, err := fileStore.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.CurrencyCopper != expectedCopper {
		t.Fatalf("transaction leaked currency change: got %d want %d", reloaded.CurrencyCopper, expectedCopper)
	}
}

func newStoreWithCharacter(t *testing.T) (*FileStore, platform.Character) {
	t.Helper()
	fileStore, err := NewFileStore(filepath.Join(t.TempDir(), "platform-state.json"), "test-build", "http://localhost:8085")
	if err != nil {
		t.Fatal(err)
	}
	account, err := fileStore.RegisterAccount("player", "secret")
	if err != nil {
		t.Fatal(err)
	}
	character, err := fileStore.CreateCharacter(account.ID, "sunset-frontier-dev", "TxHero", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatal(err)
	}
	return fileStore, character
}

package store

import "testing"

func TestSessionRecoveryStateRestoresZoneAndPosition(t *testing.T) {
	fileStore, character := newStoreWithCharacter(t)
	if _, err := fileStore.UpdateCharacterState(character.ID, "brindlebrook_hollow", 44, 55, 1); err != nil {
		t.Fatal(err)
	}
	recovery, err := fileStore.LoadSessionRecoveryState(character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recovery.CharacterID != character.ID || recovery.ZoneID != "brindlebrook_hollow" || recovery.X != 44 || recovery.Y != 55 || recovery.Z != 1 {
		t.Fatalf("unexpected recovery state: %#v", recovery)
	}
	if len(recovery.Inventory) == 0 || len(recovery.ActionBarSlots) == 0 || len(recovery.LearnedAbilityIDs) == 0 {
		t.Fatalf("expected recovery to include gameplay state: %#v", recovery)
	}
}

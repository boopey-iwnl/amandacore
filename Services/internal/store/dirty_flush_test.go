package store

import (
	"context"
	"testing"
)

func TestDirtyStateFlushWritesLatestCharacterState(t *testing.T) {
	fileStore, character := newStoreWithCharacter(t)
	buffer := NewDirtyStateBuffer(DirtyStateFlushPolicy{MaxPending: 8})
	buffer.MarkCharacterState(character.ID, "stonewake_vale", 11, 12, 0, "move")
	buffer.MarkCharacterState(character.ID, "brindlebrook_hollow", 21, 22, 0, "zone_transfer")

	result, err := buffer.Flush(context.Background(), fileStore)
	if err != nil {
		t.Fatal(err)
	}
	if result.Flushed != 1 || result.Pending != 0 {
		t.Fatalf("unexpected flush result: %#v", result)
	}
	reloaded, err := fileStore.GetCharacterByID(character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.ZoneID != "brindlebrook_hollow" || reloaded.PositionX != 21 || reloaded.PositionY != 22 {
		t.Fatalf("latest dirty state was not persisted: %#v", reloaded)
	}
}

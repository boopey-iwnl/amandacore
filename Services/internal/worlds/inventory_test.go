package worlds

import "testing"

func TestInventoryMoveSwapsAndPersists(t *testing.T) {
	server, _, session, _, _ := newGroupCreditTestServer(t)

	if session.Inventory[0].ItemID == "" || session.Inventory[1].ItemID == "" {
		t.Fatalf("expected default starter inventory in slots 0 and 1, got %#v", session.Inventory[:2])
	}
	firstItem := session.Inventory[0]
	secondItem := session.Inventory[1]

	if err := server.moveInventorySlotLocked(session, 0, 5); err != nil {
		t.Fatalf("move slot 0 to 5 failed: %v", err)
	}
	if session.Inventory[0].ItemID != "" || session.Inventory[5].ItemID != firstItem.ItemID {
		t.Fatalf("expected slot 0 empty and slot 5 to contain %s, got %#v %#v", firstItem.ItemID, session.Inventory[0], session.Inventory[5])
	}

	if err := server.moveInventorySlotLocked(session, 5, 1); err != nil {
		t.Fatalf("swap slot 5 with 1 failed: %v", err)
	}
	if session.Inventory[1].ItemID != firstItem.ItemID || session.Inventory[5].ItemID != secondItem.ItemID {
		t.Fatalf("expected occupied-slot swap to persist in session, got slot1=%#v slot5=%#v", session.Inventory[1], session.Inventory[5])
	}

	reloaded, err := server.store.GetCharacterByID(session.CharacterID)
	if err != nil {
		t.Fatalf("failed to reload character inventory: %v", err)
	}
	if reloaded.Inventory[1].ItemID != firstItem.ItemID || reloaded.Inventory[5].ItemID != secondItem.ItemID {
		t.Fatalf("expected occupied-slot swap to persist in store, got slot1=%#v slot5=%#v", reloaded.Inventory[1], reloaded.Inventory[5])
	}
}

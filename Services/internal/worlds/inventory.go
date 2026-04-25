package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

func (s *worldServer) moveInventorySlotLocked(session *worldSessionState, fromSlotIndex int, toSlotIndex int) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if fromSlotIndex < 0 || fromSlotIndex >= platform.InventorySlotCount ||
		toSlotIndex < 0 || toSlotIndex >= platform.InventorySlotCount {
		return fmt.Errorf("inventory slot is out of range")
	}
	if fromSlotIndex == toSlotIndex {
		return nil
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	fromSlot := inventory[fromSlotIndex]
	toSlot := inventory[toSlotIndex]
	if fromSlot.ItemID == "" || fromSlot.StackCount <= 0 {
		return fmt.Errorf("source slot is empty")
	}

	inventory[fromSlotIndex] = platform.CharacterInventorySlot{
		SlotIndex:   fromSlotIndex,
		ItemID:      toSlot.ItemID,
		DisplayName: toSlot.DisplayName,
		StackCount:  toSlot.StackCount,
	}
	inventory[toSlotIndex] = platform.CharacterInventorySlot{
		SlotIndex:   toSlotIndex,
		ItemID:      fromSlot.ItemID,
		DisplayName: fromSlot.DisplayName,
		StackCount:  fromSlot.StackCount,
	}

	character, err := s.store.UpdateCharacterInventory(session.CharacterID, inventory)
	if err != nil {
		return err
	}
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)

	observability.LogEvent("world-service", "world.inventory_slot_moved", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"fromSlotIndex":     fromSlotIndex,
		"toSlotIndex":       toSlotIndex,
		"itemId":            fromSlot.ItemID,
		"swappedItemId":     toSlot.ItemID,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

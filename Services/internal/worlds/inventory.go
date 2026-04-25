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

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterInventory(session.CharacterID, inventory)
	s.recordPersistenceDuration("character_inventory", persistStartedAt, err)
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

func (s *worldServer) persistSessionEconomyLocked(session *worldSessionState) error {
	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterEconomy(
		session.CharacterID,
		session.CurrencyCopper,
		session.Inventory,
		session.Equipment)
	s.recordPersistenceDuration("character_economy", persistStartedAt, err)
	if err != nil {
		return err
	}

	session.CurrencyCopper = character.CurrencyCopper
	session.Inventory = platform.NormalizeInventorySlots(character.Inventory)
	session.Equipment = platform.NormalizeEquipmentSlots(character.Equipment)
	s.applyDerivedStatsLocked(session)
	return nil
}

func addDefinedItemToInventory(inventory *[]platform.CharacterInventorySlot, item itemDefinition, stackCount int) error {
	if item.ItemID == "" || stackCount <= 0 {
		return nil
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	remaining := stackCount
	maxStack := item.MaxStack
	if maxStack <= 0 {
		maxStack = 1
	}
	if !item.Stackable {
		maxStack = 1
	}

	if item.Stackable {
		for index := range slots {
			if slots[index].ItemID != item.ItemID || slots[index].StackCount >= maxStack {
				continue
			}
			available := maxStack - slots[index].StackCount
			added := minInt(remaining, available)
			slots[index].StackCount += added
			remaining -= added
			if remaining <= 0 {
				*inventory = slots
				return nil
			}
		}
	}

	for index := range slots {
		if slots[index].ItemID != "" && slots[index].StackCount > 0 {
			continue
		}

		added := 1
		if item.Stackable {
			added = minInt(remaining, maxStack)
		}
		slots[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      item.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  added,
		}
		remaining -= added
		if remaining <= 0 {
			*inventory = slots
			return nil
		}
	}

	return fmt.Errorf("inventory is full")
}

func removeInventorySlotCount(
	inventory *[]platform.CharacterInventorySlot,
	slotIndex int,
	stackCount int,
) (platform.CharacterInventorySlot, int, error) {
	if slotIndex < 0 || slotIndex >= platform.InventorySlotCount {
		return platform.CharacterInventorySlot{}, 0, fmt.Errorf("inventory slot is out of range")
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	slot := slots[slotIndex]
	if slot.ItemID == "" || slot.StackCount <= 0 {
		return platform.CharacterInventorySlot{}, 0, fmt.Errorf("inventory slot is empty")
	}
	if stackCount <= 0 {
		stackCount = slot.StackCount
	}
	if stackCount > slot.StackCount {
		return platform.CharacterInventorySlot{}, 0, fmt.Errorf("not enough items in slot")
	}

	slots[slotIndex].StackCount -= stackCount
	if slots[slotIndex].StackCount <= 0 {
		slots[slotIndex] = platform.CharacterInventorySlot{SlotIndex: slotIndex}
	}
	*inventory = slots
	return slot, stackCount, nil
}

func inventoryItemCount(inventory []platform.CharacterInventorySlot, itemID string) int {
	if itemID == "" {
		return 0
	}

	total := 0
	for _, slot := range platform.NormalizeInventorySlots(inventory) {
		if slot.ItemID == itemID && slot.StackCount > 0 {
			total += slot.StackCount
		}
	}
	return total
}

func removeInventoryItemCount(inventory *[]platform.CharacterInventorySlot, itemID string, stackCount int) error {
	if itemID == "" || stackCount <= 0 {
		return nil
	}
	if inventoryItemCount(*inventory, itemID) < stackCount {
		return fmt.Errorf("not enough materials")
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	remaining := stackCount
	for index := range slots {
		if slots[index].ItemID != itemID || slots[index].StackCount <= 0 {
			continue
		}
		removed := minInt(remaining, slots[index].StackCount)
		slots[index].StackCount -= removed
		remaining -= removed
		if slots[index].StackCount <= 0 {
			slots[index] = platform.CharacterInventorySlot{SlotIndex: index}
		}
		if remaining <= 0 {
			*inventory = slots
			return nil
		}
	}

	return fmt.Errorf("not enough materials")
}

func (s *worldServer) equipInventorySlotLocked(session *worldSessionState, slotIndex int) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if slotIndex < 0 || slotIndex >= platform.InventorySlotCount {
		return fmt.Errorf("inventory slot is out of range")
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	sourceSlot := inventory[slotIndex]
	if sourceSlot.ItemID == "" || sourceSlot.StackCount <= 0 {
		return fmt.Errorf("inventory slot is empty")
	}
	if sourceSlot.StackCount != 1 {
		return fmt.Errorf("stacked items cannot be equipped")
	}

	item, found := findItemDefinition(sourceSlot.ItemID)
	if !found {
		return fmt.Errorf("item is not defined")
	}
	if item.EquipSlot == "" || (item.Type != itemTypeWeapon && item.Type != itemTypeArmor) {
		return fmt.Errorf("item cannot be equipped")
	}
	if item.RequiredClass != "" && item.RequiredClass != session.ClassID {
		return fmt.Errorf("wrong class for this item")
	}
	if session.Level < item.RequiredLevel {
		return fmt.Errorf("level is too low to equip this item")
	}

	equipment := platform.NormalizeEquipmentSlots(session.Equipment)
	equipmentSlotIndex := -1
	for index, equipmentSlot := range equipment {
		if equipmentSlot.Slot == item.EquipSlot {
			equipmentSlotIndex = index
			break
		}
	}
	if equipmentSlotIndex < 0 {
		return fmt.Errorf("equipment slot is unavailable")
	}

	previousEquipment := equipment[equipmentSlotIndex]
	equipment[equipmentSlotIndex] = platform.CharacterEquipmentSlot{
		Slot:        item.EquipSlot,
		ItemID:      item.ItemID,
		DisplayName: item.DisplayName,
	}

	if previousEquipment.ItemID == "" {
		inventory[slotIndex] = platform.CharacterInventorySlot{SlotIndex: slotIndex}
	} else {
		inventory[slotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   slotIndex,
			ItemID:      previousEquipment.ItemID,
			DisplayName: previousEquipment.DisplayName,
			StackCount:  1,
		}
	}

	session.Inventory = inventory
	session.Equipment = equipment
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.item_equipped", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"itemId":            item.ItemID,
		"equipSlot":         item.EquipSlot,
		"sourceSlotIndex":   slotIndex,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

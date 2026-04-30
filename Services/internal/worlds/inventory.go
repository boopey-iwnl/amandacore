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

func buildInventorySlotsResponse(inventory []platform.CharacterInventorySlot) []inventorySlotResponse {
	normalized := platform.NormalizeInventorySlots(inventory)
	slots := make([]inventorySlotResponse, 0, len(normalized))
	for _, slot := range normalized {
		response := inventorySlotResponse{
			SlotIndex:   slot.SlotIndex,
			ItemID:      slot.ItemID,
			DisplayName: slot.DisplayName,
			StackCount:  slot.StackCount,
		}
		if slot.ItemID != "" && slot.StackCount > 0 {
			if item, found := findItemDefinition(slot.ItemID); found {
				enrichInventorySlotResponse(&response, item)
			} else {
				response.IconKind = "item"
			}
		}
		slots = append(slots, response)
	}
	return slots
}

func buildEquipmentSlotsResponse(equipment []platform.CharacterEquipmentSlot) []equipmentSlotResponse {
	normalized := platform.NormalizeEquipmentSlots(equipment)
	slots := make([]equipmentSlotResponse, 0, len(normalized))
	for _, slot := range normalized {
		response := equipmentSlotResponse{
			Slot:        slot.Slot,
			ItemID:      slot.ItemID,
			DisplayName: slot.DisplayName,
		}
		if slot.ItemID != "" {
			if item, found := findItemDefinition(slot.ItemID); found {
				enrichEquipmentSlotResponse(&response, item)
			}
		}
		slots = append(slots, response)
	}
	return slots
}

func enrichInventorySlotResponse(response *inventorySlotResponse, item itemDefinition) {
	response.DisplayName = item.DisplayName
	response.ItemType = item.Type
	response.ItemSubtype = item.Subtype
	response.Quality = item.Quality
	response.IconKind = itemIconKind(item)
	response.Description = item.Description
	response.EquipSlot = item.EquipSlot
	response.RequiredArchetype = item.RequiredArchetype
	response.RequiredLevel = item.RequiredLevel
	response.SellPriceCopper = item.SellPriceCopper
	response.Strength = item.Strength
	response.Stamina = item.Stamina
	response.Armor = item.Armor
}

func enrichEquipmentSlotResponse(response *equipmentSlotResponse, item itemDefinition) {
	response.DisplayName = item.DisplayName
	response.ItemType = item.Type
	response.ItemSubtype = item.Subtype
	response.Quality = item.Quality
	response.IconKind = itemIconKind(item)
	response.Description = item.Description
	response.EquipSlot = item.EquipSlot
	response.RequiredArchetype = item.RequiredArchetype
	response.RequiredLevel = item.RequiredLevel
	response.SellPriceCopper = item.SellPriceCopper
	response.Strength = item.Strength
	response.Stamina = item.Stamina
	response.Armor = item.Armor
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

type inventoryChange struct {
	SlotIndex int
	ItemID    string
	Before    int
	After     int
	Delta     int
	StackFull bool
}

type inventoryMutationResult struct {
	CharacterID string
	ItemID      string
	Quantity    int
	Changes     []inventoryChange
}

type inventoryGrant struct {
	ItemID   string
	Quantity int
	Reason   string
}

type CharacterInventory struct {
	CharacterID string
	Slots       []InventoryStack
	Capacity    int
}

type InventoryStack struct {
	SlotIndex int
	ItemID    string
	Quantity  int
}

type InventoryChange = inventoryChange
type InventoryGrant = inventoryGrant
type InventoryCapacity struct {
	MaxStacks int
}
type InventoryMutationResult = inventoryMutationResult
type ItemGrantResult struct {
	ItemID   string
	Quantity int
	Accepted bool
	Reason   string
}

func addDefinedItemToInventory(inventory *[]platform.CharacterInventorySlot, item itemDefinition, stackCount int) error {
	_, err := grantDefinedItemToInventory(inventory, item, stackCount)
	return err
}

func grantDefinedItemToInventory(inventory *[]platform.CharacterInventorySlot, item itemDefinition, stackCount int) ([]inventoryChange, error) {
	if item.ItemID == "" || stackCount <= 0 {
		return nil, nil
	}

	slots := platform.NormalizeInventorySlots(*inventory)
	remaining := stackCount
	changes := []inventoryChange{}
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
			before := slots[index].StackCount
			available := maxStack - slots[index].StackCount
			added := minInt(remaining, available)
			slots[index].StackCount += added
			remaining -= added
			changes = append(changes, inventoryChange{
				SlotIndex: index,
				ItemID:    item.ItemID,
				Before:    before,
				After:     slots[index].StackCount,
				Delta:     added,
				StackFull: slots[index].StackCount >= maxStack,
			})
			if remaining <= 0 {
				*inventory = slots
				return changes, nil
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
		changes = append(changes, inventoryChange{
			SlotIndex: index,
			ItemID:    item.ItemID,
			Before:    0,
			After:     added,
			Delta:     added,
			StackFull: added >= maxStack,
		})
		if remaining <= 0 {
			*inventory = slots
			return changes, nil
		}
	}

	return changes, fmt.Errorf("inventory is full")
}

func (s *worldServer) grantItemStackToSessionLocked(session *worldSessionState, grant inventoryGrant) (inventoryMutationResult, error) {
	result := inventoryMutationResult{
		CharacterID: sessionCharacterID(session),
		ItemID:      grant.ItemID,
		Quantity:    grant.Quantity,
	}
	s.emitWorldEventLocked(eventInventoryGrantRequested, map[string]any{
		"characterId": result.CharacterID,
		"itemId":      grant.ItemID,
		"quantity":    grant.Quantity,
		"reason":      grant.Reason,
	})
	if session == nil {
		err := fmt.Errorf("SessionInvalid")
		s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
			"characterId": result.CharacterID,
			"itemId":      grant.ItemID,
			"quantity":    grant.Quantity,
			"reason":      err.Error(),
		})
		return result, err
	}
	item, found := findItemDefinition(grant.ItemID)
	if !found {
		err := fmt.Errorf("InvalidItem")
		s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
			"characterId": session.CharacterID,
			"itemId":      grant.ItemID,
			"quantity":    grant.Quantity,
			"reason":      err.Error(),
		})
		return result, err
	}
	changes, err := grantDefinedItemToInventory(&session.Inventory, item, grant.Quantity)
	if err != nil {
		reason := "InventoryFull"
		s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
			"characterId": session.CharacterID,
			"itemId":      grant.ItemID,
			"quantity":    grant.Quantity,
			"reason":      reason,
		})
		s.emitWorldEventLocked(eventInventoryFull, map[string]any{
			"characterId": session.CharacterID,
			"itemId":      grant.ItemID,
		})
		return result, fmt.Errorf("%s", reason)
	}
	result.Changes = changes
	if err := s.persistSessionProgressionLocked(session); err != nil {
		return result, err
	}
	s.emitInventoryGrantedLocked(session, grant.ItemID, grant.Quantity, grant.Reason)
	return result, nil
}

func (s *worldServer) grantInventoryItemsLocked(session *worldSessionState, grants []InventoryGrant, reason string) error {
	if session == nil {
		s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
			"reason": "SessionInvalid",
		})
		return fmt.Errorf("SessionInvalid")
	}
	if len(grants) == 0 {
		return nil
	}
	if reason == "" {
		reason = "server_grant"
	}

	nextInventory := platform.NormalizeInventorySlots(session.Inventory)
	applied := make([]InventoryGrant, 0, len(grants))
	for _, grant := range grants {
		if grant.ItemID == "" || grant.Quantity <= 0 {
			continue
		}
		if grant.Reason == "" {
			grant.Reason = reason
		}
		s.emitWorldEventLocked(eventInventoryGrantRequested, map[string]any{
			"characterId": session.CharacterID,
			"itemId":      grant.ItemID,
			"quantity":    grant.Quantity,
			"reason":      grant.Reason,
		})
		item, found := findItemDefinition(grant.ItemID)
		if !found {
			s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
				"characterId": session.CharacterID,
				"itemId":      grant.ItemID,
				"quantity":    grant.Quantity,
				"reason":      "InvalidItem",
			})
			return fmt.Errorf("InvalidItem")
		}
		if _, err := grantDefinedItemToInventory(&nextInventory, item, grant.Quantity); err != nil {
			s.emitWorldEventLocked(eventInventoryGrantRejected, map[string]any{
				"characterId": session.CharacterID,
				"itemId":      grant.ItemID,
				"quantity":    grant.Quantity,
				"reason":      "InventoryFull",
			})
			s.emitWorldEventLocked(eventInventoryFull, map[string]any{
				"characterId": session.CharacterID,
				"itemId":      grant.ItemID,
			})
			return fmt.Errorf("InventoryFull")
		}
		applied = append(applied, grant)
	}

	if len(applied) == 0 {
		return nil
	}
	session.Inventory = nextInventory
	for _, grant := range applied {
		s.emitInventoryGrantedLocked(session, grant.ItemID, grant.Quantity, grant.Reason)
		if err := s.applyQuestItemGrantedLocked(session, grant.ItemID, grant.Quantity); err != nil {
			return err
		}
	}
	if err := s.persistSessionProgressionLocked(session); err != nil {
		return err
	}
	s.emitWorldEventLocked(eventInventoryPersisted, map[string]any{
		"characterId": session.CharacterID,
		"reason":      reason,
	})
	return nil
}

func (s *worldServer) emitInventoryGrantedLocked(session *worldSessionState, itemID string, quantity int, reason string) {
	if session == nil || itemID == "" || quantity <= 0 {
		return
	}
	s.emitWorldEventLocked(eventInventoryItemGranted, map[string]any{
		"characterId": session.CharacterID,
		"itemId":      itemID,
		"quantity":    quantity,
		"reason":      reason,
	}, inventoryDelta(session.CharacterID, itemID, quantity, reason))
	s.emitWorldEventLocked(eventInventoryStackUpdated, map[string]any{
		"characterId": session.CharacterID,
		"itemId":      itemID,
		"quantity":    inventoryItemCount(session.Inventory, itemID),
	})
	s.emitWorldEventLocked(eventInventoryPersisted, map[string]any{
		"characterId": session.CharacterID,
		"itemId":      itemID,
	})
}

func inventoryDelta(characterID string, itemID string, quantity int, reason string) stateDiff {
	return stateDiff{
		Type:     diffInventoryDelta,
		EntityID: characterID,
		Fields: map[string]any{
			"itemId":   itemID,
			"quantity": quantity,
			"reason":   reason,
		},
	}
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
	if item.RequiredArchetype != "" && session.ArchetypeID != "" && item.RequiredArchetype != session.ArchetypeID {
		return fmt.Errorf("wrong archetype for this item")
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

func (s *worldServer) unequipEquipmentSlotLocked(session *worldSessionState, slotID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if slotID == "" {
		return fmt.Errorf("equipment slot is required")
	}

	equipment := platform.NormalizeEquipmentSlots(session.Equipment)
	equipmentSlotIndex := -1
	for index, equipmentSlot := range equipment {
		if equipmentSlot.Slot == slotID {
			equipmentSlotIndex = index
			break
		}
	}
	if equipmentSlotIndex < 0 {
		return fmt.Errorf("equipment slot is unavailable")
	}

	previousEquipment := equipment[equipmentSlotIndex]
	if previousEquipment.ItemID == "" {
		return fmt.Errorf("equipment slot is empty")
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	targetSlotIndex := -1
	for index, slot := range inventory {
		if slot.ItemID == "" || slot.StackCount <= 0 {
			targetSlotIndex = index
			break
		}
	}
	if targetSlotIndex < 0 {
		return fmt.Errorf("inventory is full")
	}

	inventory[targetSlotIndex] = platform.CharacterInventorySlot{
		SlotIndex:   targetSlotIndex,
		ItemID:      previousEquipment.ItemID,
		DisplayName: previousEquipment.DisplayName,
		StackCount:  1,
	}
	equipment[equipmentSlotIndex] = platform.CharacterEquipmentSlot{Slot: previousEquipment.Slot}

	session.Inventory = inventory
	session.Equipment = equipment
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.item_unequipped", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"itemId":            previousEquipment.ItemID,
		"equipSlot":         previousEquipment.Slot,
		"targetSlotIndex":   targetSlotIndex,
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

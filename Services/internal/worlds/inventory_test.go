package worlds

import (
	"strings"
	"testing"

	"amandacore/services/internal/platform"
)

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

func TestInventoryAndEquipmentResponsesIncludeItemMetadata(t *testing.T) {
	inventory := platform.DefaultStarterInventory()
	inventory[2] = platform.CharacterInventorySlot{
		SlotIndex:   2,
		ItemID:      itemPaddedYardVestID,
		DisplayName: "stale name",
		StackCount:  1,
	}

	inventorySlots := buildInventorySlotsResponse(inventory)
	vest := inventorySlots[2]
	if vest.DisplayName != "Padded Yard Vest" ||
		vest.EquipSlot != platform.EquipmentSlotChest ||
		vest.RequiredArchetype != platform.LegacyWayfarerArchetypeID ||
		vest.RequiredLevel != 1 ||
		vest.SellPriceCopper <= 0 ||
		vest.Stamina != 1 ||
		vest.Armor != 2 ||
		vest.IconKind == "" {
		t.Fatalf("expected enriched inventory item metadata, got %#v", vest)
	}

	equipment := platform.DefaultEquipmentSlots()
	for index := range equipment {
		if equipment[index].Slot == platform.EquipmentSlotChest {
			equipment[index].ItemID = itemPaddedYardVestID
			equipment[index].DisplayName = "stale name"
			break
		}
	}
	equipmentSlots := buildEquipmentSlotsResponse(equipment)
	chest := findEquipmentResponseSlot(t, equipmentSlots, platform.EquipmentSlotChest)
	if chest.DisplayName != "Padded Yard Vest" ||
		chest.EquipSlot != platform.EquipmentSlotChest ||
		chest.RequiredArchetype != platform.LegacyWayfarerArchetypeID ||
		chest.Stamina != 1 ||
		chest.Armor != 2 {
		t.Fatalf("expected enriched equipment item metadata, got %#v", chest)
	}
}

func TestEquipInventorySlotPersistsWithoutDuplicationAndRefreshesStats(t *testing.T) {
	server, _, session, _, _ := newGroupCreditTestServer(t)
	session.Inventory = platform.NormalizeInventorySlots(session.Inventory)
	session.Inventory[2] = platform.CharacterInventorySlot{
		SlotIndex:   2,
		ItemID:      itemPaddedYardVestID,
		DisplayName: "Padded Yard Vest",
		StackCount:  1,
	}
	session.Equipment = platform.DefaultEquipmentSlots()
	beforeStats := calculatePlayerStats(session.Level, session.Equipment, session.Talents)

	if err := server.equipInventorySlotLocked(session, 2); err != nil {
		t.Fatalf("equip inventory slot failed: %v", err)
	}

	if session.Inventory[2].ItemID != "" {
		t.Fatalf("expected source inventory slot to be emptied, got %#v", session.Inventory[2])
	}
	chest := findEquipmentSlot(t, session.Equipment, platform.EquipmentSlotChest)
	if chest.ItemID != itemPaddedYardVestID {
		t.Fatalf("expected vest equipped in chest slot, got %#v", chest)
	}
	if countSessionItem(session.Inventory, session.Equipment, itemPaddedYardVestID) != 1 {
		t.Fatalf("expected exactly one vest after equip, inventory=%#v equipment=%#v", session.Inventory, session.Equipment)
	}
	afterStats := calculatePlayerStats(session.Level, session.Equipment, session.Talents)
	if afterStats.Armor <= beforeStats.Armor || afterStats.Stamina <= beforeStats.Stamina {
		t.Fatalf("expected equipment stats to increase, before=%#v after=%#v", beforeStats, afterStats)
	}

	reloaded, err := server.store.GetCharacterByID(session.CharacterID)
	if err != nil {
		t.Fatalf("failed to reload character: %v", err)
	}
	reloadedChest := findEquipmentSlot(t, reloaded.Equipment, platform.EquipmentSlotChest)
	if reloadedChest.ItemID != itemPaddedYardVestID {
		t.Fatalf("expected equipped item to persist, got %#v", reloadedChest)
	}
}

func TestUnequipEquipmentSlotPersistsAndRejectsFullInventory(t *testing.T) {
	server, _, session, _, _ := newGroupCreditTestServer(t)
	session.Inventory = platform.NormalizeInventorySlots(session.Inventory)
	session.Inventory[2] = platform.CharacterInventorySlot{
		SlotIndex:   2,
		ItemID:      itemPaddedYardVestID,
		DisplayName: "Padded Yard Vest",
		StackCount:  1,
	}
	session.Equipment = platform.DefaultEquipmentSlots()
	if err := server.equipInventorySlotLocked(session, 2); err != nil {
		t.Fatalf("equip inventory slot failed: %v", err)
	}

	if err := server.unequipEquipmentSlotLocked(session, platform.EquipmentSlotChest); err != nil {
		t.Fatalf("unequip equipment slot failed: %v", err)
	}
	if findEquipmentSlot(t, session.Equipment, platform.EquipmentSlotChest).ItemID != "" {
		t.Fatalf("expected chest slot to be empty after unequip, got %#v", session.Equipment)
	}
	if countSessionItem(session.Inventory, session.Equipment, itemPaddedYardVestID) != 1 {
		t.Fatalf("expected exactly one vest after unequip, inventory=%#v equipment=%#v", session.Inventory, session.Equipment)
	}

	if err := server.equipInventorySlotLocked(session, firstInventorySlotWithItem(t, session.Inventory, itemPaddedYardVestID)); err != nil {
		t.Fatalf("re-equip inventory slot failed: %v", err)
	}
	for index := range session.Inventory {
		session.Inventory[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      itemCampRationID,
			DisplayName: "Camp Ration",
			StackCount:  1,
		}
	}

	err := server.unequipEquipmentSlotLocked(session, platform.EquipmentSlotChest)
	if err == nil || !strings.Contains(err.Error(), "inventory is full") {
		t.Fatalf("expected full inventory rejection, got %v", err)
	}
	if findEquipmentSlot(t, session.Equipment, platform.EquipmentSlotChest).ItemID != itemPaddedYardVestID {
		t.Fatalf("expected chest item to remain equipped after rejected unequip, got %#v", session.Equipment)
	}
}

func TestEquipInventorySlotRejectsIncompatibleItem(t *testing.T) {
	server, _, session, _, _ := newGroupCreditTestServer(t)
	session.Inventory = platform.NormalizeInventorySlots(session.Inventory)
	session.Inventory[0] = platform.CharacterInventorySlot{
		SlotIndex:   0,
		ItemID:      itemCampRationID,
		DisplayName: "Camp Ration",
		StackCount:  1,
	}
	session.Equipment = platform.DefaultEquipmentSlots()

	err := server.equipInventorySlotLocked(session, 0)
	if err == nil || !strings.Contains(err.Error(), "item cannot be equipped") {
		t.Fatalf("expected incompatible item rejection, got %v", err)
	}
	if countSessionItem(session.Inventory, session.Equipment, itemCampRationID) != 1 {
		t.Fatalf("expected ration to remain only in inventory, inventory=%#v equipment=%#v", session.Inventory, session.Equipment)
	}
}

func findEquipmentSlot(t *testing.T, slots []platform.CharacterEquipmentSlot, slotID string) platform.CharacterEquipmentSlot {
	t.Helper()
	for _, slot := range platform.NormalizeEquipmentSlots(slots) {
		if slot.Slot == slotID {
			return slot
		}
	}
	t.Fatalf("equipment slot %s not found in %#v", slotID, slots)
	return platform.CharacterEquipmentSlot{}
}

func findEquipmentResponseSlot(t *testing.T, slots []equipmentSlotResponse, slotID string) equipmentSlotResponse {
	t.Helper()
	for _, slot := range slots {
		if slot.Slot == slotID {
			return slot
		}
	}
	t.Fatalf("equipment response slot %s not found in %#v", slotID, slots)
	return equipmentSlotResponse{}
}

func firstInventorySlotWithItem(t *testing.T, inventory []platform.CharacterInventorySlot, itemID string) int {
	t.Helper()
	for _, slot := range platform.NormalizeInventorySlots(inventory) {
		if slot.ItemID == itemID {
			return slot.SlotIndex
		}
	}
	t.Fatalf("item %s not found in inventory %#v", itemID, inventory)
	return -1
}

func countSessionItem(
	inventory []platform.CharacterInventorySlot,
	equipment []platform.CharacterEquipmentSlot,
	itemID string,
) int {
	count := 0
	for _, slot := range platform.NormalizeInventorySlots(inventory) {
		if slot.ItemID == itemID {
			count += slot.StackCount
		}
	}
	for _, slot := range platform.NormalizeEquipmentSlots(equipment) {
		if slot.ItemID == itemID {
			count++
		}
	}
	return count
}

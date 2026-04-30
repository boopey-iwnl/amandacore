package platform

import "testing"

func TestNormalizeEquipmentSlotsPreservesLegacySlotsAndAddsPaperDollSlots(t *testing.T) {
	legacy := []CharacterEquipmentSlot{
		{Slot: EquipmentSlotMainHand, ItemID: "worn_militia_blade", DisplayName: "Worn Militia Blade"},
		{Slot: EquipmentSlotChest, ItemID: "padded_yard_vest", DisplayName: "Padded Yard Vest"},
		{Slot: EquipmentSlotHands, ItemID: "tds_sluiceguard_handwraps", DisplayName: "Sluiceguard Handwraps"},
		{Slot: EquipmentSlotLegs},
		{Slot: EquipmentSlotFeet, ItemID: "field_boots", DisplayName: "Field Boots"},
	}

	normalized := NormalizeEquipmentSlots(legacy)
	if len(normalized) != len(EquipmentSlots) {
		t.Fatalf("expected %d equipment slots, got %d", len(EquipmentSlots), len(normalized))
	}
	assertEquipmentSlot(t, normalized, EquipmentSlotMainHand, "worn_militia_blade")
	assertEquipmentSlot(t, normalized, EquipmentSlotChest, "padded_yard_vest")
	assertEquipmentSlot(t, normalized, EquipmentSlotHands, "tds_sluiceguard_handwraps")
	assertEquipmentSlot(t, normalized, EquipmentSlotFeet, "field_boots")
	assertEquipmentSlot(t, normalized, EquipmentSlotHead, "")
	assertEquipmentSlot(t, normalized, EquipmentSlotOffHand, "")
	assertEquipmentSlot(t, normalized, EquipmentSlotCloakOrBack, "")
}

func assertEquipmentSlot(t *testing.T, slots []CharacterEquipmentSlot, slotID string, expectedItemID string) {
	t.Helper()
	for _, slot := range slots {
		if slot.Slot == slotID {
			if slot.ItemID != expectedItemID {
				t.Fatalf("expected %s to contain %q, got %#v", slotID, expectedItemID, slot)
			}
			return
		}
	}
	t.Fatalf("equipment slot %s missing from %#v", slotID, slots)
}

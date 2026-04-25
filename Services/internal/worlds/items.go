package worlds

import "amandacore/services/internal/platform"

const (
	itemWornMilitiaBladeID = "worn_militia_blade"
	itemPaddedYardVestID   = "padded_yard_vest"
	itemFieldBootsID       = "field_boots"
	itemBentBuckleID       = "bent_buckle"
	itemCrackedTuskID      = "cracked_tusk"
	itemCampRationID       = "camp_ration"

	itemTypeWeapon     = "weapon"
	itemTypeArmor      = "armor"
	itemTypeConsumable = "consumable"
	itemTypeMaterial   = "material"
	itemTypeQuest      = "quest"
	itemTypeJunk       = "junk"

	itemQualityPoor   = "poor"
	itemQualityCommon = "common"
)

type itemDefinition struct {
	ItemID          string
	DisplayName     string
	Type            string
	Subtype         string
	Quality         string
	Stackable       bool
	MaxStack        int
	SellPriceCopper int
	BuyPriceCopper  int
	RequiredClass   string
	RequiredLevel   int
	EquipSlot       string
	Strength        int
	Stamina         int
	Armor           int
}

var itemDefinitions = map[string]itemDefinition{
	itemWornRivetID: {
		ItemID:          itemWornRivetID,
		DisplayName:     "Worn Rivet",
		Type:            itemTypeMaterial,
		Subtype:         "forged_part",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        20,
		SellPriceCopper: 1,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemValeIronChipID: {
		ItemID:          itemValeIronChipID,
		DisplayName:     "Vale Iron Chip",
		Type:            itemTypeMaterial,
		Subtype:         "ore",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        20,
		SellPriceCopper: 1,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemCampRationID: {
		ItemID:          itemCampRationID,
		DisplayName:     "Camp Ration",
		Type:            itemTypeConsumable,
		Subtype:         "ration",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 1,
		BuyPriceCopper:  3,
		RequiredLevel:   1,
	},
	itemLinenWrapID: {
		ItemID:          itemLinenWrapID,
		DisplayName:     "Linen Wrap",
		Type:            itemTypeConsumable,
		Subtype:         "bandage",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 1,
		BuyPriceCopper:  4,
		RequiredLevel:   1,
	},
	itemRoadRationID: {
		ItemID:          itemRoadRationID,
		DisplayName:     "Road Ration",
		Type:            itemTypeConsumable,
		Subtype:         "ration",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 2,
		BuyPriceCopper:  5,
		RequiredLevel:   1,
	},
	itemFieldDressingID: {
		ItemID:          itemFieldDressingID,
		DisplayName:     "Field Dressing",
		Type:            itemTypeConsumable,
		Subtype:         "bandage",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 2,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemLooseKitID: {
		ItemID:          itemLooseKitID,
		DisplayName:     "Loose Kit",
		Type:            itemTypeQuest,
		Subtype:         "supplies",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 0,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemOatBundleID: {
		ItemID:          itemOatBundleID,
		DisplayName:     "Oat Bundle",
		Type:            itemTypeQuest,
		Subtype:         "supplies",
		Quality:         itemQualityCommon,
		Stackable:       true,
		MaxStack:        10,
		SellPriceCopper: 0,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemMilitiaTokenID: {
		ItemID:          itemMilitiaTokenID,
		DisplayName:     "Militia Token",
		Type:            itemTypeQuest,
		Subtype:         "token",
		Quality:         itemQualityCommon,
		Stackable:       false,
		MaxStack:        1,
		SellPriceCopper: 0,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemTornClothID: {
		ItemID:          itemTornClothID,
		DisplayName:     "Torn Cloth",
		Type:            itemTypeJunk,
		Subtype:         "cloth",
		Quality:         itemQualityPoor,
		Stackable:       true,
		MaxStack:        20,
		SellPriceCopper: 1,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemBentBuckleID: {
		ItemID:          itemBentBuckleID,
		DisplayName:     "Bent Buckle",
		Type:            itemTypeJunk,
		Subtype:         "scrap",
		Quality:         itemQualityPoor,
		Stackable:       true,
		MaxStack:        20,
		SellPriceCopper: 2,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemCrackedTuskID: {
		ItemID:          itemCrackedTuskID,
		DisplayName:     "Cracked Tusk",
		Type:            itemTypeJunk,
		Subtype:         "trophy",
		Quality:         itemQualityPoor,
		Stackable:       true,
		MaxStack:        20,
		SellPriceCopper: 3,
		BuyPriceCopper:  0,
		RequiredLevel:   1,
	},
	itemWornMilitiaBladeID: {
		ItemID:          itemWornMilitiaBladeID,
		DisplayName:     "Worn Militia Blade",
		Type:            itemTypeWeapon,
		Subtype:         "sword",
		Quality:         itemQualityCommon,
		Stackable:       false,
		MaxStack:        1,
		SellPriceCopper: 8,
		BuyPriceCopper:  24,
		RequiredClass:   platform.DefaultClassID,
		RequiredLevel:   1,
		EquipSlot:       platform.EquipmentSlotMainHand,
		Strength:        1,
	},
	itemPaddedYardVestID: {
		ItemID:          itemPaddedYardVestID,
		DisplayName:     "Padded Yard Vest",
		Type:            itemTypeArmor,
		Subtype:         "vest",
		Quality:         itemQualityCommon,
		Stackable:       false,
		MaxStack:        1,
		SellPriceCopper: 6,
		BuyPriceCopper:  18,
		RequiredClass:   platform.DefaultClassID,
		RequiredLevel:   1,
		EquipSlot:       platform.EquipmentSlotChest,
		Stamina:         1,
		Armor:           2,
	},
	itemFieldBootsID: {
		ItemID:          itemFieldBootsID,
		DisplayName:     "Field Boots",
		Type:            itemTypeArmor,
		Subtype:         "boots",
		Quality:         itemQualityCommon,
		Stackable:       false,
		MaxStack:        1,
		SellPriceCopper: 4,
		BuyPriceCopper:  12,
		RequiredClass:   platform.DefaultClassID,
		RequiredLevel:   1,
		EquipSlot:       platform.EquipmentSlotFeet,
		Armor:           1,
	},
}

func findItemDefinition(itemID string) (itemDefinition, bool) {
	item, ok := itemDefinitions[itemID]
	if !ok {
		return itemDefinition{}, false
	}
	if item.MaxStack <= 0 {
		item.MaxStack = 1
	}
	if !item.Stackable {
		item.MaxStack = 1
	}
	if item.RequiredLevel <= 0 {
		item.RequiredLevel = 1
	}
	return item, true
}

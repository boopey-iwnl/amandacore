package worlds

import "amandacore/services/internal/platform"

const (
	itemWornMilitiaBladeID        = "worn_militia_blade"
	itemPaddedYardVestID          = "padded_yard_vest"
	itemFieldBootsID              = "field_boots"
	itemTDSSluiceguardHandwrapsID = "tds_sluiceguard_handwraps"
	itemBentBuckleID              = "bent_buckle"
	itemCrackedTuskID             = "cracked_tusk"
	itemCampRationID              = "camp_ration"
	itemDevGlimmerShardID         = "dev_glimmer_shard"
	itemDevStalkerFangID          = "dev_stalker_fang"
	itemDevFieldRationID          = "dev_field_ration"
	itemDevCopperTokenID          = "dev_copper_token"

	itemTypeWeapon     = "weapon"
	itemTypeArmor      = "armor"
	itemTypeConsumable = "consumable"
	itemTypeMaterial   = "material"
	itemTypeQuest      = "quest"
	itemTypeJunk       = "junk"
	itemTypeCurrency   = "currency_token"
	itemTypeEquipment  = "equipment_placeholder"

	itemKindCurrencyToken        = itemTypeCurrency
	itemKindCraftingMaterial     = itemTypeMaterial
	itemKindConsumable           = itemTypeConsumable
	itemKindQuestItem            = itemTypeQuest
	itemKindEquipmentPlaceholder = itemTypeEquipment
	itemKindMisc                 = itemTypeJunk

	itemQualityPoor            = "poor"
	itemQualityCommon          = "common"
	itemQualityUncommon        = "uncommon"
	itemQualityRare            = "rare"
	itemQualityEpicPlaceholder = "epic_placeholder"
)

type ItemID = string
type ItemKind = string
type ItemQuality = string
type ItemDefinition = itemDefinition
type ItemCatalog = map[string]itemDefinition

type itemDefinition struct {
	ItemID          string
	DisplayName     string
	Description     string
	Kind            string
	Type            string
	Subtype         string
	Quality         string
	Stackable       bool
	MaxStack        int
	Tags            []string
	SellPriceCopper int
	BuyPriceCopper  int
	RequiredClass   string
	RequiredLevel   int
	EquipSlot       string
	Strength        int
	Stamina         int
	Armor           int
}

func itemIconKind(item itemDefinition) string {
	switch item.ItemID {
	case itemWornMilitiaBladeID:
		return "ability_auto_attack"
	case itemPaddedYardVestID:
		return "item_padded_vest"
	case itemFieldBootsID:
		return "item_field_boots"
	case itemTDSSluiceguardHandwrapsID:
		return "item_handwraps"
	case itemFieldDressingID:
		return "item_field_dressing"
	case itemRoadRationID, itemCampRationID, itemDevFieldRationID:
		return "item_road_ration"
	case itemOatBundleID:
		return "item_oat_bundle"
	case itemLooseKitID:
		return "item_scroll_supplies"
	case itemValeIronChipID, itemDevGlimmerShardID:
		return "item_ore_chunk"
	case itemMilitiaTokenID, itemDevCopperTokenID:
		return "item_militia_token"
	case itemTornClothID:
		return "item_torn_cloth"
	}

	switch item.Type {
	case itemTypeWeapon:
		return "ability_auto_attack"
	case itemTypeArmor:
		return "item_padded_vest"
	case itemTypeConsumable:
		return "item_road_ration"
	case itemTypeMaterial:
		return "item_ore_chunk"
	case itemTypeQuest:
		return "item_scroll_supplies"
	case itemTypeJunk:
		return "item_torn_cloth"
	case itemTypeCurrency:
		return "currency_copper"
	case itemTypeEquipment:
		return "item_padded_vest"
	default:
		return "icon_missing"
	}
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
	itemDevGlimmerShardID: {
		ItemID:        itemDevGlimmerShardID,
		DisplayName:   "Glimmer Shard",
		Description:   "A faintly reflective shard used by AmandaCore development scenarios.",
		Kind:          itemKindCraftingMaterial,
		Type:          itemTypeMaterial,
		Subtype:       "dev_material",
		Quality:       itemQualityCommon,
		Stackable:     true,
		MaxStack:      99,
		RequiredLevel: 1,
		Tags:          []string{"dev", "progression"},
	},
	itemDevStalkerFangID: {
		ItemID:        itemDevStalkerFangID,
		DisplayName:   "Stalker Fang",
		Description:   "A proof token recovered from an Isle Stalker.",
		Kind:          itemKindQuestItem,
		Type:          itemTypeQuest,
		Subtype:       "dev_quest",
		Quality:       itemQualityCommon,
		Stackable:     true,
		MaxStack:      20,
		RequiredLevel: 1,
		Tags:          []string{"dev", "quest"},
	},
	itemDevFieldRationID: {
		ItemID:        itemDevFieldRationID,
		DisplayName:   "Field Ration",
		Description:   "A compact ration for early AmandaCore survival loops.",
		Kind:          itemKindConsumable,
		Type:          itemTypeConsumable,
		Subtype:       "dev_ration",
		Quality:       itemQualityCommon,
		Stackable:     true,
		MaxStack:      10,
		RequiredLevel: 1,
		Tags:          []string{"dev", "consumable"},
	},
	itemDevCopperTokenID: {
		ItemID:        itemDevCopperTokenID,
		DisplayName:   "Copper Token",
		Description:   "A small placeholder currency token for progression tests.",
		Kind:          itemKindCurrencyToken,
		Type:          itemTypeCurrency,
		Subtype:       "dev_currency",
		Quality:       itemQualityCommon,
		Stackable:     true,
		MaxStack:      999,
		RequiredLevel: 1,
		Tags:          []string{"dev", "currency"},
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
	itemTDSSluiceguardHandwrapsID: {
		ItemID:          itemTDSSluiceguardHandwrapsID,
		DisplayName:     "Sluiceguard Handwraps",
		Type:            itemTypeArmor,
		Subtype:         "handwraps",
		Quality:         itemQualityCommon,
		Stackable:       false,
		MaxStack:        1,
		SellPriceCopper: 14,
		BuyPriceCopper:  0,
		RequiredClass:   platform.DefaultClassID,
		RequiredLevel:   8,
		EquipSlot:       platform.EquipmentSlotHands,
		Stamina:         2,
		Armor:           3,
	},
}

func findItemDefinition(itemID string) (itemDefinition, bool) {
	item, ok := itemDefinitions[itemID]
	if !ok {
		return itemDefinition{}, false
	}
	if item.Kind == "" {
		item.Kind = item.Type
	}
	if item.Type == "" {
		item.Type = item.Kind
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

func itemCatalogIDs() []string {
	ids := make([]string, 0, len(itemDefinitions))
	for itemID := range itemDefinitions {
		ids = append(ids, itemID)
	}
	return ids
}

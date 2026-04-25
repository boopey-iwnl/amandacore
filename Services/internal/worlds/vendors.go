package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
)

const (
	vendorQuartermasterMiraID = "vendor_quartermaster_mira"
	vendorHighmereDainID      = "vendor_highmere_dain"
	vendorPinebarrowKeviID    = "vendor_pinebarrow_kevi"
)

type vendorDefinition struct {
	ID          string
	NPCID       string
	DisplayName string
	ItemIDs     []string
}

var vendorDefinitions = map[string]vendorDefinition{
	vendorQuartermasterMiraID: {
		ID:          vendorQuartermasterMiraID,
		NPCID:       npcQuartermasterMiraID,
		DisplayName: "Quartermaster Mira Vale",
		ItemIDs: []string{
			itemWornMilitiaBladeID,
			itemPaddedYardVestID,
			itemFieldBootsID,
			itemRoadRationID,
		},
	},
}

func findVendorDefinition(vendorID string) (vendorDefinition, bool) {
	vendor, ok := vendorDefinitions[vendorID]
	return vendor, ok
}

func vendorSellsItem(vendor vendorDefinition, itemID string) bool {
	for _, vendorItemID := range vendor.ItemIDs {
		if vendorItemID == itemID {
			return true
		}
	}
	return false
}

func (s *worldServer) validateVendorAccessLocked(session *worldSessionState, vendorID string) (vendorDefinition, error) {
	if session == nil {
		return vendorDefinition{}, fmt.Errorf("world session token was not found")
	}

	vendor, found := findVendorDefinition(vendorID)
	if !found {
		return vendorDefinition{}, fmt.Errorf("vendor is not available")
	}
	if session.CurrentTargetID != vendor.NPCID {
		return vendorDefinition{}, fmt.Errorf("right-click the vendor NPC first")
	}
	if !s.friendlyInRangeLocked(session, vendor.NPCID) {
		return vendorDefinition{}, fmt.Errorf("move closer to the vendor")
	}
	return vendor, nil
}

func (s *worldServer) buyVendorItemLocked(session *worldSessionState, vendorID string, itemID string, stackCount int) error {
	vendor, err := s.validateVendorAccessLocked(session, vendorID)
	if err != nil {
		return err
	}
	if stackCount <= 0 {
		stackCount = 1
	}
	if !vendorSellsItem(vendor, itemID) {
		return fmt.Errorf("vendor does not sell that item")
	}

	item, found := findItemDefinition(itemID)
	if !found {
		return fmt.Errorf("item is not defined")
	}
	if item.BuyPriceCopper <= 0 {
		return fmt.Errorf("item is not purchasable")
	}

	totalPrice := item.BuyPriceCopper * stackCount
	if session.CurrencyCopper < totalPrice {
		return fmt.Errorf("not enough copper")
	}

	nextInventory := session.Inventory
	if err := addDefinedItemToInventory(&nextInventory, item, stackCount); err != nil {
		return err
	}

	session.Inventory = nextInventory
	session.CurrencyCopper -= totalPrice
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.vendor_item_bought", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"vendorId":          vendor.ID,
		"itemId":            item.ItemID,
		"stackCount":        stackCount,
		"priceCopper":       totalPrice,
		"currencyCopper":    session.CurrencyCopper,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) sellVendorItemLocked(session *worldSessionState, vendorID string, slotIndex int, stackCount int) error {
	vendor, err := s.validateVendorAccessLocked(session, vendorID)
	if err != nil {
		return err
	}

	nextInventory := session.Inventory
	slot, removedCount, err := removeInventorySlotCount(&nextInventory, slotIndex, stackCount)
	if err != nil {
		return err
	}

	item, found := findItemDefinition(slot.ItemID)
	if !found {
		return fmt.Errorf("item is not defined")
	}
	if item.SellPriceCopper <= 0 {
		return fmt.Errorf("item cannot be sold")
	}

	totalPrice := item.SellPriceCopper * removedCount
	session.Inventory = nextInventory
	session.CurrencyCopper += totalPrice
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.vendor_item_sold", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"vendorId":          vendor.ID,
		"itemId":            item.ItemID,
		"stackCount":        removedCount,
		"priceCopper":       totalPrice,
		"currencyCopper":    session.CurrencyCopper,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) buildVendorResponse(session *worldSessionState) map[string]any {
	if session == nil || session.CurrentTargetID == "" {
		return map[string]any{}
	}

	for _, vendor := range vendorDefinitions {
		if vendor.NPCID != session.CurrentTargetID {
			continue
		}

		offers := make([]map[string]any, 0, len(vendor.ItemIDs))
		for _, itemID := range vendor.ItemIDs {
			item, found := findItemDefinition(itemID)
			if !found {
				continue
			}
			offers = append(offers, buildItemSummary(item))
		}

		return map[string]any{
			"id":          vendor.ID,
			"npcId":       vendor.NPCID,
			"displayName": vendor.DisplayName,
			"inRange":     s.friendlyInRangeLocked(session, vendor.NPCID),
			"offers":      offers,
		}
	}

	return map[string]any{}
}

func buildItemSummary(item itemDefinition) map[string]any {
	return map[string]any{
		"itemId":          item.ItemID,
		"displayName":     item.DisplayName,
		"type":            item.Type,
		"subtype":         item.Subtype,
		"quality":         item.Quality,
		"stackable":       item.Stackable,
		"maxStack":        item.MaxStack,
		"sellPriceCopper": item.SellPriceCopper,
		"buyPriceCopper":  item.BuyPriceCopper,
		"requiredClass":   item.RequiredClass,
		"requiredLevel":   item.RequiredLevel,
		"equipSlot":       item.EquipSlot,
		"strength":        item.Strength,
		"stamina":         item.Stamina,
		"armor":           item.Armor,
	}
}

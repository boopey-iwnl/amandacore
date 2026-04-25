package worlds

import (
	"testing"

	"amandacore/services/internal/platform"
)

func TestAuctionSellSlotResponsesExposeDepositAndTradeability(t *testing.T) {
	session := &worldSessionState{
		Inventory: []platform.CharacterInventorySlot{
			{SlotIndex: 0, ItemID: itemCampRationID, DisplayName: "Camp Ration", StackCount: 3},
			{SlotIndex: 1, ItemID: itemLooseKitID, DisplayName: "Loose Kit", StackCount: 1},
		},
	}

	responses := buildAuctionSellSlotResponses(session)
	if len(responses) != 2 {
		t.Fatalf("expected two sell slot responses, got %#v", responses)
	}

	tradeable := responses[0]
	if !tradeable.Tradeable || tradeable.DepositCopper != 1 || tradeable.BlockedReason != "" {
		t.Fatalf("expected camp ration to be auctionable with a 1 copper deposit, got %#v", tradeable)
	}
	if tradeable.ItemType != itemTypeConsumable || tradeable.ItemSubtype != "ration" {
		t.Fatalf("expected item type metadata for deposit preview, got %#v", tradeable)
	}

	questItem := responses[1]
	if questItem.Tradeable || questItem.BlockedReason == "" {
		t.Fatalf("expected quest item to be blocked from auction listing, got %#v", questItem)
	}
}

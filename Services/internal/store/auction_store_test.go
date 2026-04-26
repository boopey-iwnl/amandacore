package store

import (
	"path/filepath"
	"testing"
	"time"

	"amandacore/services/internal/platform"
)

func TestAuctionListingRemovesItemAndPersists(t *testing.T) {
	fileStore, seller, _ := createAuctionTestStore(t)
	now := time.Now().Unix()
	listing := auctionTestListing(seller.ID, now, now+3600)

	created, updatedSeller, err := fileStore.CreateAuctionListing(listing, 0, 2)
	if err != nil {
		t.Fatalf("failed to create auction listing: %v", err)
	}

	if created.AuctionID == "" || created.State != platform.AuctionStateActive {
		t.Fatalf("expected active persisted listing, got %#v", created)
	}
	if updatedSeller.Inventory[0].ItemID != "camp_ration" || updatedSeller.Inventory[0].StackCount != 1 {
		t.Fatalf("expected listing to remove two camp rations from seller inventory, got %#v", updatedSeller.Inventory[0])
	}
	if updatedSeller.CurrencyCopper != seller.CurrencyCopper-created.DepositCopper {
		t.Fatalf("expected deposit deduction, got %d from %d", updatedSeller.CurrencyCopper, seller.CurrencyCopper)
	}

	listings, err := fileStore.ListAuctionListings(seller.RealmID, "camp", "consumable", "buyout_asc", 20, 0)
	if err != nil {
		t.Fatalf("failed to browse auctions: %v", err)
	}
	if len(listings) != 1 || listings[0].AuctionID != created.AuctionID {
		t.Fatalf("expected listing in browse results, got %#v", listings)
	}
}

func TestAuctionBuyoutSettlesOnceAcrossRestart(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "platform-state.json")
	fileStore, seller, buyer := createAuctionTestCharacters(t, storePath)
	now := time.Now().Unix()

	created, _, err := fileStore.CreateAuctionListing(auctionTestListing(seller.ID, now, now+3600), 0, 1)
	if err != nil {
		t.Fatalf("failed to create auction listing: %v", err)
	}

	sold, updatedBuyer, updatedSeller, mail, err := fileStore.BuyoutAuction(created.AuctionID, buyer.ID, now+10)
	if err != nil {
		t.Fatalf("failed to buyout auction: %v", err)
	}
	if sold.State != platform.AuctionStateSold {
		t.Fatalf("expected sold auction, got %#v", sold)
	}
	if updatedBuyer.CurrencyCopper != buyer.CurrencyCopper-created.BuyoutCopper {
		t.Fatalf("expected buyer currency deduction, got %d", updatedBuyer.CurrencyCopper)
	}
	if inventoryItemCountForTest(updatedBuyer.Inventory, "camp_ration") != 4 {
		t.Fatalf("expected purchased item delivered once, got inventory %#v", updatedBuyer.Inventory)
	}
	expectedCut := created.BuyoutCopper * created.CutPercent / 100
	if expectedCut <= 0 {
		expectedCut = 1
	}
	expectedSellerCopper := seller.CurrencyCopper - created.DepositCopper + created.BuyoutCopper - expectedCut + created.DepositCopper
	if updatedSeller.CurrencyCopper != expectedSellerCopper {
		t.Fatalf("expected seller proceeds %d, got %d", expectedSellerCopper, updatedSeller.CurrencyCopper)
	}
	if len(mail) != 2 {
		t.Fatalf("expected buyer item mail and seller proceeds mail, got %#v", mail)
	}

	reopenedStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	if _, _, _, _, err := reopenedStore.BuyoutAuction(created.AuctionID, buyer.ID, now+20); err == nil {
		t.Fatalf("expected repeated buyout after restart to fail")
	}
	reloadedBuyer, err := reopenedStore.GetCharacterByID(buyer.ID)
	if err != nil {
		t.Fatalf("failed to reload buyer: %v", err)
	}
	if inventoryItemCountForTest(reloadedBuyer.Inventory, "camp_ration") != 4 {
		t.Fatalf("expected no duplicate delivery after restart, got inventory %#v", reloadedBuyer.Inventory)
	}
}

func TestAuctionCancelAndExpireReturnItemsOnce(t *testing.T) {
	fileStore, seller, _ := createAuctionTestStore(t)
	now := time.Now().Unix()

	cancelListing, _, err := fileStore.CreateAuctionListing(auctionTestListing(seller.ID, now, now+3600), 0, 1)
	if err != nil {
		t.Fatalf("failed to create cancel listing: %v", err)
	}
	canceled, updatedSeller, _, err := fileStore.CancelAuction(cancelListing.AuctionID, seller.ID, now+10)
	if err != nil {
		t.Fatalf("failed to cancel auction: %v", err)
	}
	if canceled.State != platform.AuctionStateCanceled {
		t.Fatalf("expected canceled auction, got %#v", canceled)
	}
	if inventoryItemCountForTest(updatedSeller.Inventory, "camp_ration") != 3 {
		t.Fatalf("expected canceled item returned once, got inventory %#v", updatedSeller.Inventory)
	}
	if _, _, _, err := fileStore.CancelAuction(cancelListing.AuctionID, seller.ID, now+20); err == nil {
		t.Fatalf("expected repeated cancellation to fail")
	}

	expireListing, _, err := fileStore.CreateAuctionListing(auctionTestListing(seller.ID, now-100, now-10), 0, 1)
	if err != nil {
		t.Fatalf("failed to create expiring listing: %v", err)
	}
	expired, characters, mail, err := fileStore.ExpireAuctions(now, 10)
	if err != nil {
		t.Fatalf("failed to expire auctions: %v", err)
	}
	if len(expired) != 1 || expired[0].AuctionID != expireListing.AuctionID {
		t.Fatalf("expected one expired listing, got %#v", expired)
	}
	if len(characters) != 1 || inventoryItemCountForTest(characters[0].Inventory, "camp_ration") != 3 {
		t.Fatalf("expected expired item returned once, got characters %#v", characters)
	}
	if len(mail) != 1 {
		t.Fatalf("expected expired item return mail, got %#v", mail)
	}
	expiredAgain, _, _, err := fileStore.ExpireAuctions(now+10, 10)
	if err != nil {
		t.Fatalf("failed repeated expiration pass: %v", err)
	}
	if len(expiredAgain) != 0 {
		t.Fatalf("expected no repeated expiration delivery, got %#v", expiredAgain)
	}
}

func createAuctionTestStore(t *testing.T) (*FileStore, platform.Character, platform.Character) {
	t.Helper()
	return createAuctionTestCharacters(t, filepath.Join(t.TempDir(), "platform-state.json"))
}

func createAuctionTestCharacters(t *testing.T, storePath string) (*FileStore, platform.Character, platform.Character) {
	t.Helper()
	fileStore, err := NewFileStore(storePath, "test-build", "http://127.0.0.1:8085")
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	sellerAccount, err := fileStore.RegisterAccount("auction_seller", "secret")
	if err != nil {
		t.Fatalf("failed to register seller: %v", err)
	}
	buyerAccount, err := fileStore.RegisterAccount("auction_buyer", "secret")
	if err != nil {
		t.Fatalf("failed to register buyer: %v", err)
	}

	seller, err := fileStore.CreateCharacter(
		sellerAccount.ID,
		"sunset-frontier-dev",
		"Seller",
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create seller character: %v", err)
	}
	buyer, err := fileStore.CreateCharacter(
		buyerAccount.ID,
		"sunset-frontier-dev",
		"Buyer",
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("failed to create buyer character: %v", err)
	}
	return fileStore, seller, buyer
}

func auctionTestListing(sellerCharacterID string, createdAt int64, expiresAt int64) platform.AuctionListing {
	return platform.AuctionListing{
		RealmID:             "sunset-frontier-dev",
		SellerCharacterID:   sellerCharacterID,
		ItemID:              "camp_ration",
		ItemDisplayName:     "Camp Ration",
		ItemQuality:         "common",
		ItemType:            "consumable",
		ItemSubtype:         "ration",
		ItemStackable:       true,
		ItemMaxStack:        10,
		StackCount:          1,
		BuyoutCopper:        20,
		DepositCopper:       1,
		CutPercent:          5,
		CreatedAt:           createdAt,
		ExpiresAt:           expiresAt,
		SourceInventorySlot: 0,
	}
}

func inventoryItemCountForTest(inventory []platform.CharacterInventorySlot, itemID string) int {
	total := 0
	for _, slot := range platform.NormalizeInventorySlots(inventory) {
		if slot.ItemID == itemID {
			total += slot.StackCount
		}
	}
	return total
}

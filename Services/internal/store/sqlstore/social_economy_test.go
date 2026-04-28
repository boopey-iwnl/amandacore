package sqlstore

import (
	"errors"
	"sync"
	"testing"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func TestSocialRepositoryRoundTripsAndIdempotency(t *testing.T) {
	store := newTestStore(t)
	characters := seedCharacters(t, store, "SocialOwner", "SocialFriend", "SocialTarget")
	owner := characters[0]
	friend := characters[1]
	target := characters[2]

	if _, err := store.AddFriend(owner.ID, friend.ID); err != nil {
		t.Fatalf("failed to add friend: %v", err)
	}
	if _, err := store.AddFriend(owner.ID, friend.ID); !errors.Is(err, filestore.ErrFriendExists) {
		t.Fatalf("expected duplicate friend rejection, got %v", err)
	}
	friends, err := store.ListFriends(owner.ID)
	if err != nil {
		t.Fatalf("failed to list friends: %v", err)
	}
	if len(friends) != 1 || friends[0].FriendCharacterID != friend.ID {
		t.Fatalf("expected friend round trip, got %#v", friends)
	}
	if err := store.RemoveFriend(owner.ID, friend.ID); err != nil {
		t.Fatalf("failed to remove friend: %v", err)
	}

	if _, err := store.AddIgnore(owner.ID, target.ID); err != nil {
		t.Fatalf("failed to add ignore: %v", err)
	}
	ignores, err := store.ListIgnores(owner.ID)
	if err != nil {
		t.Fatalf("failed to list ignores: %v", err)
	}
	if len(ignores) != 1 || ignores[0].IgnoredCharacterID != target.ID {
		t.Fatalf("expected ignore round trip, got %#v", ignores)
	}
	if err := store.RemoveIgnore(owner.ID, target.ID); err != nil {
		t.Fatalf("failed to remove ignore: %v", err)
	}

	message, err := store.AppendChatMessage(platform.ChatMessage{
		Channel:           "party",
		PartyID:           "party_test",
		SenderCharacterID: owner.ID,
		SenderDisplayName: owner.DisplayName,
		MessageText:       "Ready at the gate.",
	})
	if err != nil {
		t.Fatalf("failed to append chat message: %v", err)
	}
	messages, err := store.ListRecentChatMessages("party", "party:party_test", 10)
	if err != nil {
		t.Fatalf("failed to list chat messages: %v", err)
	}
	if len(messages) != 1 || messages[0].MessageID != message.MessageID {
		t.Fatalf("expected chat message round trip, got %#v", messages)
	}

	invite, err := store.CreatePartyInvite(filestore.PartyInvite{
		InviterCharacterID: owner.ID,
		TargetCharacterID:  friend.ID,
	})
	if err != nil {
		t.Fatalf("failed to create party invite: %v", err)
	}
	party, err := store.AcceptPartyInvite(invite.InviteID, friend.ID, filestore.MutationOptions{MutationKey: "party-accept-1"})
	if err != nil {
		t.Fatalf("failed to accept party invite: %v", err)
	}
	replayedParty, err := store.AcceptPartyInvite(invite.InviteID, friend.ID, filestore.MutationOptions{MutationKey: "party-accept-1"})
	if err != nil {
		t.Fatalf("expected party accept replay to be idempotent, got %v", err)
	}
	if len(replayedParty.MemberCharacterIDs) != 2 || party.ID != replayedParty.ID {
		t.Fatalf("expected one party membership after replay, got %#v", replayedParty)
	}

	guild, err := store.CreateGuild("M7 Testers", owner.ID)
	if err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}
	guildInvite, err := store.CreateGuildInvite(guild.ID, owner.ID, target.ID, 0)
	if err != nil {
		t.Fatalf("failed to create guild invite: %v", err)
	}
	acceptedGuild, err := store.AcceptGuildInvite(guildInvite.InviteID, target.ID, filestore.MutationOptions{MutationKey: "guild-accept-1"})
	if err != nil {
		t.Fatalf("failed to accept guild invite: %v", err)
	}
	replayedGuild, err := store.AcceptGuildInvite(guildInvite.InviteID, target.ID, filestore.MutationOptions{MutationKey: "guild-accept-1"})
	if err != nil {
		t.Fatalf("expected guild accept replay to be idempotent, got %v", err)
	}
	if len(replayedGuild.Members) != 2 || replayedGuild.ID != acceptedGuild.ID {
		t.Fatalf("expected one guild membership after replay, got %#v", replayedGuild.Members)
	}
}

func TestConcurrentPartyAndGuildAcceptsDoNotDuplicateMembership(t *testing.T) {
	store := newTestStore(t)
	characters := seedCharacters(t, store, "ConcurrentLeader", "ConcurrentPartyTarget", "ConcurrentGuildTarget")
	leader := characters[0]
	partyTarget := characters[1]
	guildTarget := characters[2]

	partyInvite, err := store.CreatePartyInvite(filestore.PartyInvite{
		InviterCharacterID: leader.ID,
		TargetCharacterID:  partyTarget.ID,
	})
	if err != nil {
		t.Fatalf("failed to create party invite: %v", err)
	}
	runConcurrent(8, func(index int) {
		_, _ = store.AcceptPartyInvite(partyInvite.InviteID, partyTarget.ID, filestore.MutationOptions{MutationKey: "party-concurrent"})
	})
	party, err := store.GetPartyForCharacter(partyTarget.ID)
	if err != nil {
		t.Fatalf("failed to load party after concurrent accept: %v", err)
	}
	if countStringInSlice(party.MemberCharacterIDs, partyTarget.ID) != 1 {
		t.Fatalf("expected party target once, got %#v", party.MemberCharacterIDs)
	}

	guild, err := store.CreateGuild("Concurrent Guild", leader.ID)
	if err != nil {
		t.Fatalf("failed to create guild: %v", err)
	}
	guildInvite, err := store.CreateGuildInvite(guild.ID, leader.ID, guildTarget.ID, 0)
	if err != nil {
		t.Fatalf("failed to create guild invite: %v", err)
	}
	runConcurrent(8, func(index int) {
		_, _ = store.AcceptGuildInvite(guildInvite.InviteID, guildTarget.ID, filestore.MutationOptions{MutationKey: "guild-concurrent"})
	})
	loadedGuild, err := store.GetGuildForCharacter(guildTarget.ID)
	if err != nil {
		t.Fatalf("failed to load guild after concurrent accept: %v", err)
	}
	if countGuildMember(loadedGuild.Members, guildTarget.ID) != 1 {
		t.Fatalf("expected guild target once, got %#v", loadedGuild.Members)
	}
}

func TestEconomyRepositoryTransactionsAndIdempotency(t *testing.T) {
	store := newTestStore(t)
	characters := seedCharacters(t, store, "EconomySeller", "EconomyBuyer", "EconomyFullInventory", "EconomyMail")
	seller := characters[0]
	buyer := characters[1]
	fullInventoryCharacter := characters[2]
	mailRecipient := characters[3]

	entry, err := store.AppendCurrencyMutation(filestore.CurrencyLedgerEntry{
		CharacterID: buyer.ID,
		DeltaCopper: 25,
		Reason:      "test_grant",
		Operation:   "test.currency",
		MutationKey: "currency-1",
	})
	if err != nil {
		t.Fatalf("failed to append currency mutation: %v", err)
	}
	replayedEntry, err := store.AppendCurrencyMutation(filestore.CurrencyLedgerEntry{
		CharacterID: buyer.ID,
		DeltaCopper: 25,
		Reason:      "test_grant",
		Operation:   "test.currency",
		MutationKey: "currency-1",
	})
	if err != nil {
		t.Fatalf("expected idempotent currency replay, got %v", err)
	}
	if replayedEntry.BalanceAfter != entry.BalanceAfter {
		t.Fatalf("expected replayed ledger balance %d, got %d", entry.BalanceAfter, replayedEntry.BalanceAfter)
	}

	buyerAfterBuy, _, err := store.BuyVendorItem(filestore.VendorPurchaseMutation{
		CharacterID:     buyer.ID,
		ItemID:          "m7_vendor_ration",
		DisplayName:     "M7 Vendor Ration",
		Quantity:        2,
		UnitPriceCopper: 10,
		MaxStack:        10,
		Stackable:       true,
		MutationKey:     "vendor-buy-1",
		SourceID:        "vendor_test",
	})
	if err != nil {
		t.Fatalf("failed vendor buy: %v", err)
	}
	replayedBuyer, _, err := store.BuyVendorItem(filestore.VendorPurchaseMutation{
		CharacterID:     buyer.ID,
		ItemID:          "m7_vendor_ration",
		DisplayName:     "M7 Vendor Ration",
		Quantity:        2,
		UnitPriceCopper: 10,
		MaxStack:        10,
		Stackable:       true,
		MutationKey:     "vendor-buy-1",
		SourceID:        "vendor_test",
	})
	if err != nil {
		t.Fatalf("expected vendor buy replay, got %v", err)
	}
	if inventoryItemCount(replayedBuyer.Inventory, "m7_vendor_ration") != inventoryItemCount(buyerAfterBuy.Inventory, "m7_vendor_ration") {
		t.Fatalf("vendor buy replay duplicated inventory: %#v", replayedBuyer.Inventory)
	}

	fullInventory := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	for index := range fullInventory {
		fullInventory[index] = platform.CharacterInventorySlot{SlotIndex: index, ItemID: "full_slot_item", DisplayName: "Full Slot Item", StackCount: 1}
	}
	if _, err := store.UpdateCharacterInventory(fullInventoryCharacter.ID, fullInventory); err != nil {
		t.Fatalf("failed to fill inventory: %v", err)
	}
	beforeRollback, err := store.GetCharacterByID(fullInventoryCharacter.ID)
	if err != nil {
		t.Fatalf("failed to load rollback character: %v", err)
	}
	if _, _, err := store.BuyVendorItem(filestore.VendorPurchaseMutation{
		CharacterID:     fullInventoryCharacter.ID,
		ItemID:          "no_room_item",
		DisplayName:     "No Room Item",
		Quantity:        1,
		UnitPriceCopper: 1,
		MaxStack:        1,
		Stackable:       false,
		MutationKey:     "vendor-buy-full",
	}); !errors.Is(err, filestore.ErrInventoryFull) {
		t.Fatalf("expected inventory-full rollback, got %v", err)
	}
	afterRollback, err := store.GetCharacterByID(fullInventoryCharacter.ID)
	if err != nil {
		t.Fatalf("failed to reload rollback character: %v", err)
	}
	if afterRollback.CurrencyCopper != beforeRollback.CurrencyCopper {
		t.Fatalf("currency changed after failed vendor buy: before %d after %d", beforeRollback.CurrencyCopper, afterRollback.CurrencyCopper)
	}

	sellerInventory := platform.NormalizeInventorySlots(seller.Inventory)
	sellerInventory[6] = platform.CharacterInventorySlot{SlotIndex: 6, ItemID: "m7_auction_relic", DisplayName: "M7 Auction Relic", StackCount: 1}
	if _, err := store.UpdateCharacterInventory(seller.ID, sellerInventory); err != nil {
		t.Fatalf("failed to seed seller inventory: %v", err)
	}
	listing, updatedSeller, err := store.CreateAuctionListing(filestore.AuctionCreateMutation{
		Listing: platform.AuctionListing{
			SellerCharacterID: seller.ID,
			ItemID:            "m7_auction_relic",
			ItemDisplayName:   "M7 Auction Relic",
			ItemMaxStack:      1,
			StackCount:        1,
			BuyoutCopper:      40,
			DepositCopper:     5,
		},
		SourceSlotIndex: 6,
		StackCount:      1,
		MutationKey:     "auction-list-1",
	})
	if err != nil {
		t.Fatalf("failed to create auction listing: %v", err)
	}
	if inventoryItemCount(updatedSeller.Inventory, "m7_auction_relic") != 0 {
		t.Fatalf("expected auction item removed from seller inventory, got %#v", updatedSeller.Inventory)
	}
	soldListing, updatedBuyer, updatedSeller, err := store.BuyoutAuction(filestore.AuctionBuyoutMutation{
		AuctionID:        listing.AuctionID,
		BuyerCharacterID: buyer.ID,
		MutationKey:      "auction-buyout-1",
	})
	if err != nil {
		t.Fatalf("failed auction buyout: %v", err)
	}
	if soldListing.State != platform.AuctionStateSold {
		t.Fatalf("expected sold listing, got %#v", soldListing)
	}
	if inventoryItemCount(updatedBuyer.Inventory, "m7_auction_relic") != 1 {
		t.Fatalf("expected buyer to receive auction item, got %#v", updatedBuyer.Inventory)
	}
	if updatedSeller.CurrencyCopper <= seller.CurrencyCopper {
		t.Fatalf("expected seller proceeds, before %d after %d", seller.CurrencyCopper, updatedSeller.CurrencyCopper)
	}
	if _, _, _, err := store.BuyoutAuction(filestore.AuctionBuyoutMutation{
		AuctionID:        listing.AuctionID,
		BuyerCharacterID: fullInventoryCharacter.ID,
		MutationKey:      "auction-buyout-2",
	}); !errors.Is(err, filestore.ErrAuctionInactive) {
		t.Fatalf("expected duplicate buyout rejection, got %v", err)
	}
	if _, _, err := store.CancelAuction(filestore.AuctionCancelMutation{
		AuctionID:         listing.AuctionID,
		SellerCharacterID: seller.ID,
		MutationKey:       "auction-cancel-sold",
	}); !errors.Is(err, filestore.ErrAuctionInactive) {
		t.Fatalf("expected cancel after buyout to fail, got %v", err)
	}

	mail, err := store.CreateMail(platform.MailEnvelope{
		SenderDisplayName:    "M7 Test",
		RecipientCharacterID: mailRecipient.ID,
		Subject:              "M7 Claim",
		Body:                 "Claim once.",
		ItemAttachments: []platform.MailItemAttachment{
			{ItemID: "m7_mail_item", DisplayName: "M7 Mail Item", StackCount: 1},
		},
		CurrencyCopper: 9,
	})
	if err != nil {
		t.Fatalf("failed to create mail: %v", err)
	}
	claimedCharacter, claimedMail, err := store.ClaimMailAttachment(filestore.MailAttachmentClaim{
		MailID:      mail.MailID,
		CharacterID: mailRecipient.ID,
		MutationKey: "mail-claim-1",
	})
	if err != nil {
		t.Fatalf("failed to claim mail attachment: %v", err)
	}
	if inventoryItemCount(claimedCharacter.Inventory, "m7_mail_item") != 1 && claimedMail.CurrencyCopper == 0 {
		t.Fatalf("expected mail item or currency claim, character=%#v mail=%#v", claimedCharacter, claimedMail)
	}
	if _, _, err := store.ClaimMailAttachment(filestore.MailAttachmentClaim{
		MailID:      mail.MailID,
		CharacterID: mailRecipient.ID,
		MutationKey: "mail-claim-1",
	}); !errors.Is(err, filestore.ErrMailAttachmentClaimed) {
		t.Fatalf("expected duplicate mail claim prevention, got %v", err)
	}
}

func TestConcurrentAuctionBuyoutAndMailClaimApplyOnce(t *testing.T) {
	store := newTestStore(t)
	characters := seedCharacters(t, store, "ConcurrentSeller", "ConcurrentBuyerA", "ConcurrentBuyerB")
	seller := characters[0]
	buyerA := characters[1]
	buyerB := characters[2]

	inventory := platform.NormalizeInventorySlots(seller.Inventory)
	inventory[4] = platform.CharacterInventorySlot{SlotIndex: 4, ItemID: "m7_contended_item", DisplayName: "M7 Contended Item", StackCount: 1}
	if _, err := store.UpdateCharacterInventory(seller.ID, inventory); err != nil {
		t.Fatalf("failed to seed seller inventory: %v", err)
	}
	listing, _, err := store.CreateAuctionListing(filestore.AuctionCreateMutation{
		Listing: platform.AuctionListing{
			SellerCharacterID: seller.ID,
			ItemID:            "m7_contended_item",
			ItemDisplayName:   "M7 Contended Item",
			ItemMaxStack:      1,
			StackCount:        1,
			BuyoutCopper:      30,
		},
		SourceSlotIndex: 4,
		StackCount:      1,
	})
	if err != nil {
		t.Fatalf("failed to create contended listing: %v", err)
	}
	runConcurrent(2, func(index int) {
		buyerID := buyerA.ID
		if index == 1 {
			buyerID = buyerB.ID
		}
		_, _, _, _ = store.BuyoutAuction(filestore.AuctionBuyoutMutation{
			AuctionID:        listing.AuctionID,
			BuyerCharacterID: buyerID,
			MutationKey:      "auction-contended",
		})
	})
	buyerALoaded, _ := store.GetCharacterByID(buyerA.ID)
	buyerBLoaded, _ := store.GetCharacterByID(buyerB.ID)
	if inventoryItemCount(buyerALoaded.Inventory, "m7_contended_item")+inventoryItemCount(buyerBLoaded.Inventory, "m7_contended_item") != 1 {
		t.Fatalf("expected contended auction item to be granted once, buyerA=%#v buyerB=%#v", buyerALoaded.Inventory, buyerBLoaded.Inventory)
	}

	mail, err := store.CreateMail(platform.MailEnvelope{
		SenderDisplayName:    "M7 Test",
		RecipientCharacterID: buyerA.ID,
		Subject:              "Concurrent Claim",
		Body:                 "Only once.",
		ItemAttachments: []platform.MailItemAttachment{
			{ItemID: "m7_claim_once", DisplayName: "M7 Claim Once", StackCount: 1},
		},
	})
	if err != nil {
		t.Fatalf("failed to create concurrent mail: %v", err)
	}
	runConcurrent(8, func(index int) {
		_, _, _ = store.ClaimMailAttachment(filestore.MailAttachmentClaim{
			MailID:      mail.MailID,
			CharacterID: buyerA.ID,
			MutationKey: "mail-concurrent",
		})
	})
	buyerAAfterMail, _ := store.GetCharacterByID(buyerA.ID)
	if inventoryItemCount(buyerAAfterMail.Inventory, "m7_claim_once") != 1 {
		t.Fatalf("expected mail item to be claimed once, got %#v", buyerAAfterMail.Inventory)
	}
}

func seedCharacters(t *testing.T, store *Store, names ...string) []platform.Character {
	t.Helper()
	realm, err := SeedDevRealm(store)
	if err != nil {
		t.Fatalf("failed to seed realm: %v", err)
	}
	characters := make([]platform.Character, 0, len(names))
	for _, name := range names {
		account, err := SeedTestAccount(store, "acct_"+name, "secret")
		if err != nil {
			t.Fatalf("failed to seed account %s: %v", name, err)
		}
		character, err := store.CreateCharacter(account.ID, realm.ID, name, platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID)
		if err != nil {
			t.Fatalf("failed to create character %s: %v", name, err)
		}
		characters = append(characters, character)
	}
	return characters
}

func runConcurrent(count int, fn func(index int)) {
	var wg sync.WaitGroup
	wg.Add(count)
	for index := 0; index < count; index++ {
		go func(index int) {
			defer wg.Done()
			fn(index)
		}(index)
	}
	wg.Wait()
}

func countStringInSlice(values []string, value string) int {
	count := 0
	for _, candidate := range values {
		if candidate == value {
			count++
		}
	}
	return count
}

func countGuildMember(members []platform.GuildMember, characterID string) int {
	count := 0
	for _, member := range members {
		if member.CharacterID == characterID {
			count++
		}
	}
	return count
}

func inventoryItemCount(inventory []platform.CharacterInventorySlot, itemID string) int {
	count := 0
	for _, slot := range inventory {
		if slot.ItemID == itemID {
			count += slot.StackCount
		}
	}
	return count
}

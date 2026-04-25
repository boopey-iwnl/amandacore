package store

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/platform"
)

func (s *FileStore) CreateAuctionListing(
	listing platform.AuctionListing,
	sourceSlotIndex int,
	stackCount int,
) (platform.AuctionListing, platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return platform.AuctionListing{}, platform.Character{}, err
	}
	defer s.unlockState()

	seller, ok := s.state.Characters[listing.SellerCharacterID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("seller character not found")
	}
	inventory := platform.NormalizeInventorySlots(seller.Inventory)
	if sourceSlotIndex < 0 || sourceSlotIndex >= len(inventory) {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("inventory slot is out of range")
	}
	sourceSlot := inventory[sourceSlotIndex]
	if sourceSlot.ItemID == "" || sourceSlot.StackCount <= 0 {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("inventory slot is empty")
	}
	if sourceSlot.ItemID != listing.ItemID {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("source slot item changed")
	}
	if stackCount <= 0 || stackCount > sourceSlot.StackCount {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("not enough items in slot")
	}
	if seller.CurrencyCopper < listing.DepositCopper {
		return platform.AuctionListing{}, platform.Character{}, fmt.Errorf("not enough copper for deposit")
	}

	now := time.Now().Unix()
	if listing.AuctionID == "" {
		listing.AuctionID = randomID("auction")
	}
	if listing.CreatedAt == 0 {
		listing.CreatedAt = now
	}
	if listing.ExpiresAt <= listing.CreatedAt {
		listing.ExpiresAt = listing.CreatedAt + int64((24 * time.Hour).Seconds())
	}
	listing.RealmID = seller.RealmID
	listing.SellerDisplayName = seller.DisplayName
	listing.StackCount = stackCount
	listing.SourceInventorySlot = sourceSlotIndex
	listing.State = platform.AuctionStateActive
	listing.Version = 1

	inventory[sourceSlotIndex].StackCount -= stackCount
	if inventory[sourceSlotIndex].StackCount <= 0 {
		inventory[sourceSlotIndex] = platform.CharacterInventorySlot{SlotIndex: sourceSlotIndex}
	}
	seller.CurrencyCopper -= listing.DepositCopper
	seller.Inventory = cloneInventorySlots(inventory)
	seller.LastSeenAt = now
	s.state.Characters[seller.ID] = platform.NormalizeCharacter(seller)
	s.state.Auctions[listing.AuctionID] = listing
	appendAuditEventLocked(s, auctionAuditEvent("auction.listed", listing, "", ""))

	if err := s.saveLocked(); err != nil {
		return platform.AuctionListing{}, platform.Character{}, err
	}
	return cloneAuctionListing(listing), normalizedCharacterCopy(s.state.Characters[seller.ID]), nil
}

func (s *FileStore) ListAuctionListings(
	realmID string,
	search string,
	itemType string,
	sortBy string,
	limit int,
	offset int,
) ([]platform.AuctionListing, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	search = strings.ToLower(strings.TrimSpace(search))
	itemType = strings.ToLower(strings.TrimSpace(itemType))
	listings := make([]platform.AuctionListing, 0)
	for _, listing := range s.state.Auctions {
		if listing.RealmID != realmID || listing.State != platform.AuctionStateActive {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(listing.ItemDisplayName), search) {
			continue
		}
		if itemType != "" && strings.ToLower(listing.ItemType) != itemType {
			continue
		}
		listings = append(listings, cloneAuctionListing(listing))
	}

	sortAuctionListings(listings, sortBy)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset >= len(listings) {
		return []platform.AuctionListing{}, nil
	}
	end := offset + limit
	if end > len(listings) {
		end = len(listings)
	}
	return listings[offset:end], nil
}

func (s *FileStore) ListAuctionsForSeller(sellerCharacterID string) ([]platform.AuctionListing, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	listings := make([]platform.AuctionListing, 0)
	for _, listing := range s.state.Auctions {
		if listing.SellerCharacterID == sellerCharacterID {
			listings = append(listings, cloneAuctionListing(listing))
		}
	}
	sortAuctionListings(listings, "created_desc")
	return listings, nil
}

func (s *FileStore) BuyoutAuction(
	auctionID string,
	buyerCharacterID string,
	now int64,
) (platform.AuctionListing, platform.Character, platform.Character, []platform.MailEnvelope, error) {
	if err := s.lockState(true); err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, err
	}
	defer s.unlockState()

	listing, ok := s.state.Auctions[auctionID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("auction not found")
	}
	if listing.State != platform.AuctionStateActive {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("auction is not active")
	}
	if listing.ExpiresAt <= now {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("auction has expired")
	}
	if listing.SellerCharacterID == buyerCharacterID {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("cannot buy your own auction")
	}

	buyer, ok := s.state.Characters[buyerCharacterID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("buyer character not found")
	}
	seller, ok := s.state.Characters[listing.SellerCharacterID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("seller character not found")
	}
	if buyer.RealmID != listing.RealmID || seller.RealmID != listing.RealmID {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("auction is not on this realm")
	}
	if buyer.CurrencyCopper < listing.BuyoutCopper {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, fmt.Errorf("not enough copper")
	}

	nextBuyerInventory, err := addAuctionItemToInventory(buyer.Inventory, listing)
	if err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, err
	}

	cutCopper := listing.BuyoutCopper * listing.CutPercent / 100
	if listing.BuyoutCopper > 0 && cutCopper <= 0 {
		cutCopper = 1
	}
	proceedsCopper := listing.BuyoutCopper - cutCopper + listing.DepositCopper
	if proceedsCopper < 0 {
		proceedsCopper = 0
	}

	buyer.CurrencyCopper -= listing.BuyoutCopper
	buyer.Inventory = nextBuyerInventory
	buyer.LastSeenAt = now
	seller.CurrencyCopper += proceedsCopper
	seller.LastSeenAt = now

	listing.State = platform.AuctionStateSold
	listing.SoldAt = now
	listing.BuyerCharacterID = buyerCharacterID
	listing.CutCopper = cutCopper
	listing.Version++

	buyerMail := buildAuctionItemMail(listing, buyer.ID, "Market purchase delivered", "The purchased item has been delivered.", now)
	sellerMail := buildAuctionCurrencyMail(listing, seller.ID, proceedsCopper, "Market sale settled", "Your market sale has been settled.", now)
	listing.ItemDeliveredMailID = buyerMail.MailID
	listing.ProceedsDeliveredMailID = sellerMail.MailID

	s.state.Characters[buyer.ID] = platform.NormalizeCharacter(buyer)
	s.state.Characters[seller.ID] = platform.NormalizeCharacter(seller)
	s.state.Auctions[auctionID] = listing
	s.state.Mail[buyerMail.MailID] = buyerMail
	s.state.Mail[sellerMail.MailID] = sellerMail
	appendAuditEventLocked(s, auctionAuditEvent("auction.purchased", listing, buyerCharacterID, ""))
	appendAuditEventLocked(s, auctionAuditEvent("auction.proceeds_delivered", listing, buyerCharacterID, ""))

	if err := s.saveLocked(); err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.Character{}, nil, err
	}
	return cloneAuctionListing(listing),
		normalizedCharacterCopy(s.state.Characters[buyer.ID]),
		normalizedCharacterCopy(s.state.Characters[seller.ID]),
		[]platform.MailEnvelope{cloneMailEnvelope(buyerMail), cloneMailEnvelope(sellerMail)},
		nil
}

func (s *FileStore) CancelAuction(
	auctionID string,
	sellerCharacterID string,
	now int64,
) (platform.AuctionListing, platform.Character, platform.MailEnvelope, error) {
	if err := s.lockState(true); err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, err
	}
	defer s.unlockState()

	listing, ok := s.state.Auctions[auctionID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, fmt.Errorf("auction not found")
	}
	if listing.State != platform.AuctionStateActive {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, fmt.Errorf("auction is not active")
	}
	if listing.SellerCharacterID != sellerCharacterID {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, fmt.Errorf("only the seller can cancel this auction")
	}
	if listing.ExpiresAt <= now {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, fmt.Errorf("auction has expired")
	}

	seller, ok := s.state.Characters[sellerCharacterID]
	if !ok {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, fmt.Errorf("seller character not found")
	}
	nextInventory, err := addAuctionItemToInventory(seller.Inventory, listing)
	if err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, err
	}

	listing.State = platform.AuctionStateCanceled
	listing.CanceledAt = now
	listing.Version++
	returnMail := buildAuctionItemMail(listing, seller.ID, "Market listing canceled", "The listed item has been returned.", now)
	listing.ReturnDeliveredMailID = returnMail.MailID
	seller.Inventory = nextInventory
	seller.LastSeenAt = now

	s.state.Characters[seller.ID] = platform.NormalizeCharacter(seller)
	s.state.Auctions[auctionID] = listing
	s.state.Mail[returnMail.MailID] = returnMail
	appendAuditEventLocked(s, auctionAuditEvent("auction.canceled", listing, "", ""))
	appendAuditEventLocked(s, auctionAuditEvent("auction.item_returned", listing, "", "canceled"))

	if err := s.saveLocked(); err != nil {
		return platform.AuctionListing{}, platform.Character{}, platform.MailEnvelope{}, err
	}
	return cloneAuctionListing(listing), normalizedCharacterCopy(s.state.Characters[seller.ID]), cloneMailEnvelope(returnMail), nil
}

func (s *FileStore) ExpireAuctions(now int64, limit int) ([]platform.AuctionListing, []platform.Character, []platform.MailEnvelope, error) {
	if err := s.lockState(true); err != nil {
		return nil, nil, nil, err
	}
	defer s.unlockState()

	if limit <= 0 {
		limit = 50
	}
	expiredListings := make([]platform.AuctionListing, 0)
	updatedCharacters := make([]platform.Character, 0)
	mail := make([]platform.MailEnvelope, 0)
	for auctionID, listing := range s.state.Auctions {
		if len(expiredListings) >= limit {
			break
		}
		if listing.State != platform.AuctionStateActive || listing.ExpiresAt > now {
			continue
		}
		seller, ok := s.state.Characters[listing.SellerCharacterID]
		if !ok {
			continue
		}
		nextInventory, err := addAuctionItemToInventory(seller.Inventory, listing)
		if err != nil {
			appendAuditEventLocked(s, auctionAuditEvent("auction.item_return_failed", listing, "", err.Error()))
			continue
		}
		listing.State = platform.AuctionStateExpired
		listing.Version++
		returnMail := buildAuctionItemMail(listing, seller.ID, "Market listing expired", "The listed item has expired and been returned.", now)
		listing.ReturnDeliveredMailID = returnMail.MailID
		seller.Inventory = nextInventory
		seller.LastSeenAt = now

		s.state.Characters[seller.ID] = platform.NormalizeCharacter(seller)
		s.state.Auctions[auctionID] = listing
		s.state.Mail[returnMail.MailID] = returnMail
		appendAuditEventLocked(s, auctionAuditEvent("auction.expired", listing, "", ""))
		appendAuditEventLocked(s, auctionAuditEvent("auction.item_returned", listing, "", "expired"))

		expiredListings = append(expiredListings, cloneAuctionListing(listing))
		updatedCharacters = append(updatedCharacters, normalizedCharacterCopy(s.state.Characters[seller.ID]))
		mail = append(mail, cloneMailEnvelope(returnMail))
	}
	if len(expiredListings) == 0 {
		return []platform.AuctionListing{}, []platform.Character{}, []platform.MailEnvelope{}, nil
	}
	if err := s.saveLocked(); err != nil {
		return nil, nil, nil, err
	}
	return expiredListings, updatedCharacters, mail, nil
}

func (s *FileStore) ListMailForCharacter(characterID string) ([]platform.MailEnvelope, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := make([]platform.MailEnvelope, 0)
	for _, mail := range s.state.Mail {
		if mail.RecipientCharacterID == characterID {
			results = append(results, cloneMailEnvelope(mail))
		}
	}
	sort.Slice(results, func(left int, right int) bool {
		return results[left].CreatedAt > results[right].CreatedAt
	})
	return results, nil
}

func (s *FileStore) AppendAuditEvent(event platform.AuditEvent) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	appendAuditEventLocked(s, event)
	return s.saveLocked()
}

func addAuctionItemToInventory(
	source []platform.CharacterInventorySlot,
	listing platform.AuctionListing,
) ([]platform.CharacterInventorySlot, error) {
	slots := platform.NormalizeInventorySlots(source)
	remaining := listing.StackCount
	maxStack := listing.ItemMaxStack
	if maxStack <= 0 || !listing.ItemStackable {
		maxStack = 1
	}

	if listing.ItemStackable {
		for index := range slots {
			if slots[index].ItemID != listing.ItemID || slots[index].StackCount >= maxStack {
				continue
			}
			available := maxStack - slots[index].StackCount
			added := minStoreInt(remaining, available)
			slots[index].StackCount += added
			remaining -= added
			if remaining <= 0 {
				return slots, nil
			}
		}
	}

	for index := range slots {
		if slots[index].ItemID != "" && slots[index].StackCount > 0 {
			continue
		}
		added := 1
		if listing.ItemStackable {
			added = minStoreInt(remaining, maxStack)
		}
		slots[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      listing.ItemID,
			DisplayName: listing.ItemDisplayName,
			StackCount:  added,
		}
		remaining -= added
		if remaining <= 0 {
			return slots, nil
		}
	}
	return nil, fmt.Errorf("inventory is full")
}

func buildAuctionItemMail(
	listing platform.AuctionListing,
	recipientCharacterID string,
	subject string,
	body string,
	now int64,
) platform.MailEnvelope {
	return platform.MailEnvelope{
		MailID:               fmt.Sprintf("mail_%s_item_%s", listing.AuctionID, recipientCharacterID),
		AuctionID:            listing.AuctionID,
		SenderDisplayName:    "Highmere Market",
		RecipientCharacterID: recipientCharacterID,
		Subject:              subject,
		Body:                 body,
		ItemAttachments: []platform.MailItemAttachment{
			{
				ItemID:      listing.ItemID,
				DisplayName: listing.ItemDisplayName,
				StackCount:  listing.StackCount,
			},
		},
		CreatedAt:   now,
		DeliveredAt: now,
	}
}

func buildAuctionCurrencyMail(
	listing platform.AuctionListing,
	recipientCharacterID string,
	currencyCopper int,
	subject string,
	body string,
	now int64,
) platform.MailEnvelope {
	return platform.MailEnvelope{
		MailID:               fmt.Sprintf("mail_%s_currency_%s", listing.AuctionID, recipientCharacterID),
		AuctionID:            listing.AuctionID,
		SenderDisplayName:    "Highmere Market",
		RecipientCharacterID: recipientCharacterID,
		Subject:              subject,
		Body:                 body,
		CurrencyCopper:       currencyCopper,
		CreatedAt:            now,
		DeliveredAt:          now,
	}
}

func appendAuditEventLocked(s *FileStore, event platform.AuditEvent) {
	if event.ID == "" {
		event.ID = randomID("audit")
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	s.state.AuditEvents[event.ID] = event
}

func auctionAuditEvent(
	action string,
	listing platform.AuctionListing,
	buyerCharacterID string,
	reason string,
) platform.AuditEvent {
	metadata := map[string]any{
		"auctionId":     listing.AuctionID,
		"itemId":        listing.ItemID,
		"stackCount":    listing.StackCount,
		"buyoutCopper":  listing.BuyoutCopper,
		"depositCopper": listing.DepositCopper,
		"cutCopper":     listing.CutCopper,
	}
	if buyerCharacterID != "" {
		metadata["buyerCharacterId"] = buyerCharacterID
	}
	return platform.AuditEvent{
		Action:            action,
		ActorCharacterID:  listing.SellerCharacterID,
		TargetCharacterID: buyerCharacterID,
		Reason:            reason,
		Metadata:          metadata,
	}
}

func cloneAuctionListing(source platform.AuctionListing) platform.AuctionListing {
	return source
}

func cloneMailEnvelope(source platform.MailEnvelope) platform.MailEnvelope {
	source.ItemAttachments = append([]platform.MailItemAttachment(nil), source.ItemAttachments...)
	return source
}

func sortAuctionListings(listings []platform.AuctionListing, sortBy string) {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "buyout_desc":
		sort.Slice(listings, func(left int, right int) bool {
			return listings[left].BuyoutCopper > listings[right].BuyoutCopper
		})
	case "created_desc":
		sort.Slice(listings, func(left int, right int) bool {
			return listings[left].CreatedAt > listings[right].CreatedAt
		})
	default:
		sort.Slice(listings, func(left int, right int) bool {
			if listings[left].BuyoutCopper == listings[right].BuyoutCopper {
				return listings[left].ExpiresAt < listings[right].ExpiresAt
			}
			return listings[left].BuyoutCopper < listings[right].BuyoutCopper
		})
	}
}

func minStoreInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

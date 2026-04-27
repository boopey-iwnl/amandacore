package sqlstore

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) GetCurrencyBalance(characterID string) (int, error) {
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		return 0, err
	}
	return character.CurrencyCopper, nil
}

func (s *Store) AppendCurrencyMutation(entry filestore.CurrencyLedgerEntry) (filestore.CurrencyLedgerEntry, error) {
	if entry.Operation == "" {
		entry.Operation = "currency.adjust"
	}
	if entry.CreatedAt == 0 {
		entry.CreatedAt = s.now().Unix()
	}
	var applied filestore.CurrencyLedgerEntry
	err := s.WithTransaction("sqlstore.currency_mutation", func(tx *Tx) error {
		if entry.MutationKey != "" {
			replayed, found, err := tx.replayedCurrencyMutation(entry.CharacterID, entry.Operation, entry.MutationKey)
			if err != nil {
				return err
			}
			if found {
				applied = replayed
				return nil
			}
		}
		character, version, err := tx.loadCharacterForMutation(entry.CharacterID)
		if err != nil {
			return err
		}
		nextBalance := character.CurrencyCopper + entry.DeltaCopper
		if nextBalance < 0 {
			return filestore.ErrInsufficientCurrency
		}
		character.CurrencyCopper = nextBalance
		character.LastSeenAt = entry.CreatedAt
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(character), version); err != nil {
			return err
		}
		entry.BalanceAfter = nextBalance
		if entry.EntryID == "" {
			entry.EntryID = randomID("ledger")
		}
		if err := tx.insertCurrencyLedgerEntry(entry); err != nil {
			return err
		}
		applied = entry
		return nil
	})
	return applied, err
}

func (s *Store) BuyVendorItem(mutation filestore.VendorPurchaseMutation) (platform.Character, filestore.CurrencyLedgerEntry, error) {
	totalCopper := mutation.Quantity * mutation.UnitPriceCopper
	if mutation.Quantity <= 0 || mutation.ItemID == "" {
		return platform.Character{}, filestore.CurrencyLedgerEntry{}, fmt.Errorf("vendor purchase item and quantity are required")
	}
	var updated platform.Character
	var ledger filestore.CurrencyLedgerEntry
	err := s.WithTransaction("sqlstore.vendor_buy", func(tx *Tx) error {
		if mutation.MutationKey != "" {
			var response vendorMutationResponse
			if err := tx.replayEconomyMutation(mutation.CharacterID, "vendor.buy", mutation.MutationKey, &response); err == nil && response.Character.ID != "" {
				updated = response.Character
				ledger = response.Ledger
				return nil
			} else if err != nil {
				return err
			}
		}
		character, version, err := tx.loadCharacterForMutation(mutation.CharacterID)
		if err != nil {
			return err
		}
		if character.CurrencyCopper < totalCopper {
			return filestore.ErrInsufficientCurrency
		}
		inventory := platform.NormalizeInventorySlots(character.Inventory)
		if err := grantItemToInventory(&inventory, filestore.InventoryItemGrant{
			ItemID:      mutation.ItemID,
			DisplayName: mutation.DisplayName,
			Quantity:    mutation.Quantity,
			MaxStack:    mutation.MaxStack,
			Stackable:   mutation.Stackable,
		}); err != nil {
			return err
		}
		now := tx.store.now().Unix()
		character.CurrencyCopper -= totalCopper
		character.Inventory = inventory
		character.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(character), version); err != nil {
			return err
		}
		ledger = filestore.CurrencyLedgerEntry{
			EntryID:          randomID("ledger"),
			CharacterID:      character.ID,
			DeltaCopper:      -totalCopper,
			BalanceAfter:     character.CurrencyCopper,
			Reason:           "vendor_purchase",
			Operation:        "vendor.buy",
			SourceKind:       "vendor",
			SourceID:         mutation.SourceID,
			ActorCharacterID: mutation.ActorCharacterID,
			MutationKey:      mutation.MutationKey,
			CreatedAt:        now,
		}
		if err := tx.insertCurrencyLedgerEntry(ledger); err != nil {
			return err
		}
		if err := tx.insertVendorTransaction(character.ID, "buy", mutation.ItemID, mutation.Quantity, mutation.UnitPriceCopper, totalCopper, mutation.SourceID, mutation.MutationKey, now); err != nil {
			return err
		}
		updated = platform.NormalizeCharacter(character)
		if mutation.MutationKey != "" {
			return tx.recordEconomyMutation(character.ID, "vendor.buy", mutation.MutationKey, vendorMutationResponse{Character: updated, Ledger: ledger})
		}
		return nil
	})
	return updated, ledger, err
}

func (s *Store) SellVendorItem(mutation filestore.VendorSaleMutation) (platform.Character, filestore.CurrencyLedgerEntry, error) {
	totalCopper := mutation.Quantity * mutation.UnitPriceCopper
	if mutation.Quantity <= 0 {
		return platform.Character{}, filestore.CurrencyLedgerEntry{}, fmt.Errorf("vendor sale quantity is required")
	}
	var updated platform.Character
	var ledger filestore.CurrencyLedgerEntry
	err := s.WithTransaction("sqlstore.vendor_sell", func(tx *Tx) error {
		if mutation.MutationKey != "" {
			var response vendorMutationResponse
			if err := tx.replayEconomyMutation(mutation.CharacterID, "vendor.sell", mutation.MutationKey, &response); err == nil && response.Character.ID != "" {
				updated = response.Character
				ledger = response.Ledger
				return nil
			} else if err != nil {
				return err
			}
		}
		character, version, err := tx.loadCharacterForMutation(mutation.CharacterID)
		if err != nil {
			return err
		}
		inventory := platform.NormalizeInventorySlots(character.Inventory)
		soldItemID, err := removeItemFromInventory(&inventory, mutation.SlotIndex, mutation.Quantity)
		if err != nil {
			return err
		}
		now := tx.store.now().Unix()
		character.CurrencyCopper += totalCopper
		character.Inventory = inventory
		character.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(character), version); err != nil {
			return err
		}
		ledger = filestore.CurrencyLedgerEntry{
			EntryID:          randomID("ledger"),
			CharacterID:      character.ID,
			DeltaCopper:      totalCopper,
			BalanceAfter:     character.CurrencyCopper,
			Reason:           "vendor_sale",
			Operation:        "vendor.sell",
			SourceKind:       "vendor",
			SourceID:         mutation.SourceID,
			ActorCharacterID: mutation.ActorCharacterID,
			MutationKey:      mutation.MutationKey,
			CreatedAt:        now,
		}
		if err := tx.insertCurrencyLedgerEntry(ledger); err != nil {
			return err
		}
		if err := tx.insertVendorTransaction(character.ID, "sell", soldItemID, mutation.Quantity, mutation.UnitPriceCopper, totalCopper, mutation.SourceID, mutation.MutationKey, now); err != nil {
			return err
		}
		updated = platform.NormalizeCharacter(character)
		if mutation.MutationKey != "" {
			return tx.recordEconomyMutation(character.ID, "vendor.sell", mutation.MutationKey, vendorMutationResponse{Character: updated, Ledger: ledger})
		}
		return nil
	})
	return updated, ledger, err
}

func (s *Store) CreateAuctionListing(mutation filestore.AuctionCreateMutation) (platform.AuctionListing, platform.Character, error) {
	listing := mutation.Listing
	var seller platform.Character
	err := s.WithTransaction("sqlstore.auction_create", func(tx *Tx) error {
		if mutation.MutationKey != "" {
			if existing, found, err := tx.findAuctionListingByMutation(listing.SellerCharacterID, mutation.MutationKey); err != nil {
				return err
			} else if found {
				listing = existing
				loadedSeller, _, err := tx.loadCharacterForMutation(existing.SellerCharacterID)
				if err != nil {
					return err
				}
				seller = loadedSeller
				return nil
			}
		}
		loadedSeller, version, err := tx.loadCharacterForMutation(listing.SellerCharacterID)
		if err != nil {
			return err
		}
		inventory := platform.NormalizeInventorySlots(loadedSeller.Inventory)
		if mutation.SourceSlotIndex < 0 || mutation.SourceSlotIndex >= len(inventory) {
			return filestore.ErrInvalidInventoryMove
		}
		sourceSlot := inventory[mutation.SourceSlotIndex]
		if sourceSlot.ItemID == "" || sourceSlot.StackCount < mutation.StackCount || mutation.StackCount <= 0 {
			return filestore.ErrInvalidInventoryMove
		}
		if listing.ItemID == "" {
			listing.ItemID = sourceSlot.ItemID
		}
		if listing.ItemDisplayName == "" {
			listing.ItemDisplayName = sourceSlot.DisplayName
		}
		if sourceSlot.ItemID != listing.ItemID {
			return filestore.ErrInvalidInventoryMove
		}
		if loadedSeller.CurrencyCopper < listing.DepositCopper {
			return filestore.ErrInsufficientCurrency
		}
		now := tx.store.now().Unix()
		if listing.AuctionID == "" {
			listing.AuctionID = randomID("auction")
		}
		if listing.CreatedAt == 0 {
			listing.CreatedAt = now
		}
		if listing.ExpiresAt <= listing.CreatedAt {
			listing.ExpiresAt = listing.CreatedAt + int64((24 * time.Hour).Seconds())
		}
		if listing.CutPercent == 0 {
			listing.CutPercent = 5
		}
		listing.RealmID = loadedSeller.RealmID
		listing.SellerDisplayName = loadedSeller.DisplayName
		listing.StackCount = mutation.StackCount
		listing.SourceInventorySlot = mutation.SourceSlotIndex
		listing.State = platform.AuctionStateActive
		listing.Version = 1

		inventory[mutation.SourceSlotIndex].StackCount -= mutation.StackCount
		if inventory[mutation.SourceSlotIndex].StackCount <= 0 {
			inventory[mutation.SourceSlotIndex] = platform.CharacterInventorySlot{SlotIndex: mutation.SourceSlotIndex}
		}
		loadedSeller.CurrencyCopper -= listing.DepositCopper
		loadedSeller.Inventory = inventory
		loadedSeller.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(loadedSeller), version); err != nil {
			return err
		}
		if listing.DepositCopper > 0 {
			if err := tx.insertCurrencyLedgerEntry(filestore.CurrencyLedgerEntry{
				EntryID:          randomID("ledger"),
				CharacterID:      loadedSeller.ID,
				DeltaCopper:      -listing.DepositCopper,
				BalanceAfter:     loadedSeller.CurrencyCopper,
				Reason:           "auction_deposit",
				Operation:        "auction.list",
				SourceKind:       "auction",
				SourceID:         listing.AuctionID,
				ActorCharacterID: loadedSeller.ID,
				MutationKey:      mutation.MutationKey,
				CreatedAt:        now,
			}); err != nil {
				return err
			}
		}
		if err := tx.insertAuctionListing(listing, mutation.MutationKey); err != nil {
			return err
		}
		if err := tx.insertAuctionTransaction(listing.AuctionID, "listed", loadedSeller.ID, "", listing.DepositCopper, listing.ItemID, listing.StackCount, mutation.MutationKey, now); err != nil {
			return err
		}
		if err := tx.insertTransferAudit("economy.auction_listed", loadedSeller.ID, loadedSeller.ID, "", "auction", listing.AuctionID, listing.ItemID, listing.StackCount, -listing.DepositCopper, mutation.MutationKey, "applied"); err != nil {
			return err
		}
		seller = platform.NormalizeCharacter(loadedSeller)
		return nil
	})
	return listing, seller, err
}

func (s *Store) ListAuctionListings(realmID string, search string, itemType string, sortBy string, limit int, offset int) ([]platform.AuctionListing, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	search = "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
	itemType = strings.ToLower(strings.TrimSpace(itemType))
	rows, err := s.db.Query(
		`SELECT `+auctionColumns()+`
		FROM ac_auction_listings
		WHERE realm_id = ? AND state = ?
			AND (? = '%%' OR LOWER(item_display_name) LIKE ?)
			AND (? = '' OR LOWER(item_type) = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`,
		realmID,
		platform.AuctionStateActive,
		search,
		search,
		itemType,
		itemType,
		limit,
		offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listings, err := scanAuctionListings(rows)
	if err != nil {
		return nil, err
	}
	sortAuctionListings(listings, sortBy)
	return listings, nil
}

func (s *Store) ListAuctionsForSeller(sellerCharacterID string) ([]platform.AuctionListing, error) {
	rows, err := s.db.Query(
		`SELECT `+auctionColumns()+` FROM ac_auction_listings WHERE seller_character_id = ? ORDER BY created_at DESC`,
		sellerCharacterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuctionListings(rows)
}

func (s *Store) BuyoutAuction(mutation filestore.AuctionBuyoutMutation) (platform.AuctionListing, platform.Character, platform.Character, error) {
	var listing platform.AuctionListing
	var buyer platform.Character
	var seller platform.Character
	err := s.WithTransaction("sqlstore.auction_buyout", func(tx *Tx) error {
		loadedListing, err := tx.getAuctionListing(mutation.AuctionID)
		if err != nil {
			return err
		}
		now := mutation.Now
		if now == 0 {
			now = tx.store.now().Unix()
		}
		if loadedListing.State != platform.AuctionStateActive || loadedListing.ExpiresAt <= now {
			return filestore.ErrAuctionInactive
		}
		if loadedListing.SellerCharacterID == mutation.BuyerCharacterID {
			return fmt.Errorf("cannot buy your own auction")
		}
		loadedBuyer, buyerVersion, err := tx.loadCharacterForMutation(mutation.BuyerCharacterID)
		if err != nil {
			return err
		}
		loadedSeller, sellerVersion, err := tx.loadCharacterForMutation(loadedListing.SellerCharacterID)
		if err != nil {
			return err
		}
		if loadedBuyer.CurrencyCopper < loadedListing.BuyoutCopper {
			return filestore.ErrInsufficientCurrency
		}
		buyerInventory := platform.NormalizeInventorySlots(loadedBuyer.Inventory)
		if err := grantItemToInventory(&buyerInventory, filestore.InventoryItemGrant{
			ItemID:      loadedListing.ItemID,
			DisplayName: loadedListing.ItemDisplayName,
			Quantity:    loadedListing.StackCount,
			MaxStack:    loadedListing.ItemMaxStack,
			Stackable:   loadedListing.ItemStackable,
		}); err != nil {
			return err
		}
		cutCopper := loadedListing.BuyoutCopper * loadedListing.CutPercent / 100
		if loadedListing.BuyoutCopper > 0 && cutCopper <= 0 {
			cutCopper = 1
		}
		proceedsCopper := loadedListing.BuyoutCopper - cutCopper + loadedListing.DepositCopper
		if proceedsCopper < 0 {
			proceedsCopper = 0
		}
		loadedBuyer.CurrencyCopper -= loadedListing.BuyoutCopper
		loadedBuyer.Inventory = buyerInventory
		loadedBuyer.LastSeenAt = now
		loadedSeller.CurrencyCopper += proceedsCopper
		loadedSeller.LastSeenAt = now
		loadedListing.State = platform.AuctionStateSold
		loadedListing.SoldAt = now
		loadedListing.BuyerCharacterID = mutation.BuyerCharacterID
		loadedListing.CutCopper = cutCopper
		loadedListing.Version++
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(loadedBuyer), buyerVersion); err != nil {
			return err
		}
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(loadedSeller), sellerVersion); err != nil {
			return err
		}
		if err := tx.updateAuctionListing(loadedListing); err != nil {
			return err
		}
		if err := tx.insertCurrencyLedgerEntry(filestore.CurrencyLedgerEntry{
			EntryID:          randomID("ledger"),
			CharacterID:      loadedBuyer.ID,
			DeltaCopper:      -loadedListing.BuyoutCopper,
			BalanceAfter:     loadedBuyer.CurrencyCopper,
			Reason:           "auction_buyout",
			Operation:        "auction.buyout",
			SourceKind:       "auction",
			SourceID:         loadedListing.AuctionID,
			ActorCharacterID: loadedBuyer.ID,
			MutationKey:      mutation.MutationKey,
			CreatedAt:        now,
		}); err != nil {
			return err
		}
		if err := tx.insertCurrencyLedgerEntry(filestore.CurrencyLedgerEntry{
			EntryID:          randomID("ledger"),
			CharacterID:      loadedSeller.ID,
			DeltaCopper:      proceedsCopper,
			BalanceAfter:     loadedSeller.CurrencyCopper,
			Reason:           "auction_proceeds",
			Operation:        "auction.proceeds",
			SourceKind:       "auction",
			SourceID:         loadedListing.AuctionID,
			ActorCharacterID: loadedBuyer.ID,
			MutationKey:      mutation.MutationKey,
			CreatedAt:        now,
		}); err != nil {
			return err
		}
		if err := tx.insertAuctionTransaction(loadedListing.AuctionID, "buyout", loadedBuyer.ID, loadedSeller.ID, loadedListing.BuyoutCopper, loadedListing.ItemID, loadedListing.StackCount, mutation.MutationKey, now); err != nil {
			return err
		}
		if err := tx.insertTransferAudit("economy.auction_bought", loadedBuyer.ID, loadedSeller.ID, loadedBuyer.ID, "auction", loadedListing.AuctionID, loadedListing.ItemID, loadedListing.StackCount, -loadedListing.BuyoutCopper, mutation.MutationKey, "applied"); err != nil {
			return err
		}
		listing = loadedListing
		buyer = platform.NormalizeCharacter(loadedBuyer)
		seller = platform.NormalizeCharacter(loadedSeller)
		return nil
	})
	return listing, buyer, seller, err
}

func (s *Store) CancelAuction(mutation filestore.AuctionCancelMutation) (platform.AuctionListing, platform.Character, error) {
	var listing platform.AuctionListing
	var seller platform.Character
	err := s.WithTransaction("sqlstore.auction_cancel", func(tx *Tx) error {
		loadedListing, err := tx.getAuctionListing(mutation.AuctionID)
		if err != nil {
			return err
		}
		now := mutation.Now
		if now == 0 {
			now = tx.store.now().Unix()
		}
		if loadedListing.State != platform.AuctionStateActive || loadedListing.ExpiresAt <= now {
			return filestore.ErrAuctionInactive
		}
		if loadedListing.SellerCharacterID != mutation.SellerCharacterID {
			return fmt.Errorf("only the seller can cancel this auction")
		}
		loadedSeller, version, err := tx.loadCharacterForMutation(mutation.SellerCharacterID)
		if err != nil {
			return err
		}
		inventory := platform.NormalizeInventorySlots(loadedSeller.Inventory)
		if err := grantItemToInventory(&inventory, filestore.InventoryItemGrant{
			ItemID:      loadedListing.ItemID,
			DisplayName: loadedListing.ItemDisplayName,
			Quantity:    loadedListing.StackCount,
			MaxStack:    loadedListing.ItemMaxStack,
			Stackable:   loadedListing.ItemStackable,
		}); err != nil {
			return err
		}
		loadedListing.State = platform.AuctionStateCanceled
		loadedListing.CanceledAt = now
		loadedListing.Version++
		loadedSeller.Inventory = inventory
		loadedSeller.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(loadedSeller), version); err != nil {
			return err
		}
		if err := tx.updateAuctionListing(loadedListing); err != nil {
			return err
		}
		if err := tx.insertAuctionTransaction(loadedListing.AuctionID, "cancel", loadedSeller.ID, "", 0, loadedListing.ItemID, loadedListing.StackCount, mutation.MutationKey, now); err != nil {
			return err
		}
		if err := tx.insertTransferAudit("economy.auction_cancelled", loadedSeller.ID, loadedSeller.ID, "", "auction", loadedListing.AuctionID, loadedListing.ItemID, loadedListing.StackCount, 0, mutation.MutationKey, "applied"); err != nil {
			return err
		}
		listing = loadedListing
		seller = platform.NormalizeCharacter(loadedSeller)
		return nil
	})
	return listing, seller, err
}

func (s *Store) CreateMail(mail platform.MailEnvelope) (platform.MailEnvelope, error) {
	if mail.MailID == "" {
		mail.MailID = randomID("mail")
	}
	if mail.CreatedAt == 0 {
		mail.CreatedAt = s.now().Unix()
	}
	if mail.DeliveredAt == 0 {
		mail.DeliveredAt = mail.CreatedAt
	}
	err := s.WithTransaction("sqlstore.mail_create", func(tx *Tx) error {
		return tx.insertMail(mail)
	})
	return mail, err
}

func (s *Store) ListMailForCharacter(characterID string) ([]platform.MailEnvelope, error) {
	rows, err := s.db.Query(
		`SELECT mail_id FROM ac_mail_messages WHERE recipient_character_id = ? AND deleted_at = 0 ORDER BY created_at DESC`,
		characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mail []platform.MailEnvelope
	for rows.Next() {
		var mailID string
		if err := rows.Scan(&mailID); err != nil {
			return nil, err
		}
		envelope, err := s.loadMail(mailID)
		if err != nil {
			return nil, err
		}
		mail = append(mail, envelope)
	}
	return mail, rows.Err()
}

func (s *Store) ClaimMailAttachment(claim filestore.MailAttachmentClaim) (platform.Character, platform.MailEnvelope, error) {
	var character platform.Character
	var envelope platform.MailEnvelope
	err := s.WithTransaction("sqlstore.mail_claim", func(tx *Tx) error {
		now := claim.Now
		if now == 0 {
			now = tx.store.now().Unix()
		}
		loadedMail, err := tx.loadMail(claim.MailID)
		if err != nil {
			return err
		}
		if loadedMail.RecipientCharacterID != claim.CharacterID {
			return filestore.ErrMailMissing
		}
		attachment, err := tx.loadClaimableMailAttachment(claim)
		if err != nil {
			return err
		}
		loadedCharacter, version, err := tx.loadCharacterForMutation(claim.CharacterID)
		if err != nil {
			return err
		}
		inventory := platform.NormalizeInventorySlots(loadedCharacter.Inventory)
		if attachment.ItemID != "" && attachment.StackCount > 0 {
			if err := grantItemToInventory(&inventory, filestore.InventoryItemGrant{
				ItemID:      attachment.ItemID,
				DisplayName: attachment.DisplayName,
				Quantity:    attachment.StackCount,
				MaxStack:    attachment.StackCount,
				Stackable:   attachment.StackCount > 1,
			}); err != nil {
				return err
			}
		}
		loadedCharacter.Inventory = inventory
		if attachment.CurrencyCopper > 0 {
			loadedCharacter.CurrencyCopper += attachment.CurrencyCopper
		}
		loadedCharacter.LastSeenAt = now
		if err := tx.saveCharacterWithVersion(platform.NormalizeCharacter(loadedCharacter), version); err != nil {
			return err
		}
		if attachment.CurrencyCopper > 0 {
			if err := tx.insertCurrencyLedgerEntry(filestore.CurrencyLedgerEntry{
				EntryID:          randomID("ledger"),
				CharacterID:      loadedCharacter.ID,
				DeltaCopper:      attachment.CurrencyCopper,
				BalanceAfter:     loadedCharacter.CurrencyCopper,
				Reason:           "mail_claim",
				Operation:        "mail.claim",
				SourceKind:       "mail",
				SourceID:         claim.MailID,
				ActorCharacterID: claim.CharacterID,
				MutationKey:      claim.MutationKey,
				CreatedAt:        now,
			}); err != nil {
				return err
			}
		}
		if _, err := tx.tx.Exec(
			`UPDATE ac_mail_attachments SET claimed_at = ?, claimed_by_character_id = ?, claim_mutation_key = ?
			WHERE attachment_id = ? AND claimed_at = 0`,
			now,
			claim.CharacterID,
			claim.MutationKey,
			attachment.AttachmentID); err != nil {
			return err
		}
		if err := tx.insertTransferAudit("economy.mail_claimed", claim.CharacterID, "", claim.CharacterID, "mail", claim.MailID, attachment.ItemID, attachment.StackCount, attachment.CurrencyCopper, claim.MutationKey, "applied"); err != nil {
			return err
		}
		character = platform.NormalizeCharacter(loadedCharacter)
		envelope, err = tx.loadMail(claim.MailID)
		return err
	})
	return character, envelope, err
}

type vendorMutationResponse struct {
	Character platform.Character
	Ledger    filestore.CurrencyLedgerEntry
}

type mailAttachmentRow struct {
	AttachmentID   string
	ItemID         string
	DisplayName    string
	StackCount     int
	CurrencyCopper int
}

func (tx *Tx) replayedCurrencyMutation(characterID string, operation string, mutationKey string) (filestore.CurrencyLedgerEntry, bool, error) {
	row := tx.tx.QueryRow(
		`SELECT entry_id, character_id, delta_copper, balance_after, reason, operation, source_kind, source_id, actor_character_id, mutation_key, created_at
		FROM ac_currency_ledger
		WHERE character_id = ? AND operation = ? AND mutation_key = ?`,
		characterID,
		operation,
		mutationKey)
	return scanCurrencyLedgerEntry(row)
}

func scanCurrencyLedgerEntry(row rowScanner) (filestore.CurrencyLedgerEntry, bool, error) {
	var entry filestore.CurrencyLedgerEntry
	if err := row.Scan(
		&entry.EntryID,
		&entry.CharacterID,
		&entry.DeltaCopper,
		&entry.BalanceAfter,
		&entry.Reason,
		&entry.Operation,
		&entry.SourceKind,
		&entry.SourceID,
		&entry.ActorCharacterID,
		&entry.MutationKey,
		&entry.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return filestore.CurrencyLedgerEntry{}, false, nil
		}
		return filestore.CurrencyLedgerEntry{}, false, err
	}
	return entry, true, nil
}

func (tx *Tx) insertCurrencyLedgerEntry(entry filestore.CurrencyLedgerEntry) error {
	if entry.EntryID == "" {
		entry.EntryID = randomID("ledger")
	}
	if entry.Operation == "" {
		entry.Operation = "currency.adjust"
	}
	_, err := tx.tx.Exec(
		`INSERT INTO ac_currency_ledger (
			entry_id, character_id, delta_copper, balance_after, reason, created_at,
			mutation_key, operation, source_kind, source_id, actor_character_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.EntryID,
		entry.CharacterID,
		entry.DeltaCopper,
		entry.BalanceAfter,
		entry.Reason,
		entry.CreatedAt,
		entry.MutationKey,
		entry.Operation,
		entry.SourceKind,
		entry.SourceID,
		entry.ActorCharacterID)
	if err != nil && isConstraintError(err) {
		return filestore.ErrDuplicateMutation
	}
	return err
}

func (tx *Tx) insertVendorTransaction(characterID string, kind string, itemID string, stackCount int, unitPrice int, totalCopper int, sourceID string, mutationKey string, now int64) error {
	_, err := tx.tx.Exec(
		`INSERT INTO ac_vendor_transactions (
			transaction_id, character_id, transaction_type, item_id, stack_count, unit_price_copper,
			total_copper, mutation_key, source_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		randomID("vendor_tx"),
		characterID,
		kind,
		itemID,
		stackCount,
		unitPrice,
		totalCopper,
		mutationKey,
		sourceID,
		now)
	if err != nil && isConstraintError(err) {
		return filestore.ErrDuplicateMutation
	}
	return err
}

func (tx *Tx) replayEconomyMutation(actorCharacterID string, operation string, mutationKey string, out any) error {
	row := tx.tx.QueryRow(
		`SELECT response_json FROM ac_economy_mutations WHERE actor_character_id = ? AND operation = ? AND mutation_key = ?`,
		actorCharacterID,
		operation,
		mutationKey)
	var responseJSON string
	if err := row.Scan(&responseJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return decodeJSON(responseJSON, out)
}

func (tx *Tx) recordEconomyMutation(actorCharacterID string, operation string, mutationKey string, response any) error {
	payload, err := encodeJSON(response)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`INSERT INTO ac_economy_mutations (mutation_id, actor_character_id, operation, mutation_key, response_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		randomID("emut"),
		actorCharacterID,
		operation,
		mutationKey,
		payload,
		tx.store.now().Unix())
	if err != nil && isConstraintError(err) {
		return filestore.ErrDuplicateMutation
	}
	return err
}

func removeItemFromInventory(inventory *[]platform.CharacterInventorySlot, slotIndex int, quantity int) (string, error) {
	slots := platform.NormalizeInventorySlots(*inventory)
	if slotIndex < 0 || slotIndex >= len(slots) {
		return "", filestore.ErrInvalidInventoryMove
	}
	slot := slots[slotIndex]
	if slot.ItemID == "" || slot.StackCount < quantity || quantity <= 0 {
		return "", filestore.ErrInvalidInventoryMove
	}
	itemID := slot.ItemID
	slots[slotIndex].StackCount -= quantity
	if slots[slotIndex].StackCount <= 0 {
		slots[slotIndex] = platform.CharacterInventorySlot{SlotIndex: slotIndex}
	}
	*inventory = slots
	return itemID, nil
}

func auctionColumns() string {
	return `auction_id, realm_id, seller_character_id, seller_display_name, buyer_character_id, item_id,
		item_display_name, item_quality, item_type, item_subtype, item_stackable, item_max_stack,
		stack_count, buyout_copper, bid_copper, current_bid_copper, current_bidder_character_id,
		deposit_copper, cut_copper, cut_percent, created_at, expires_at, sold_at, canceled_at,
		state, source_inventory_slot, version, item_delivered_mail_id, proceeds_delivered_mail_id, return_delivered_mail_id`
}

func scanAuctionListings(rows *sql.Rows) ([]platform.AuctionListing, error) {
	listings := []platform.AuctionListing{}
	for rows.Next() {
		listing, err := scanAuctionListing(rows)
		if err != nil {
			return nil, err
		}
		listings = append(listings, listing)
	}
	return listings, rows.Err()
}

func scanAuctionListing(row rowScanner) (platform.AuctionListing, error) {
	var listing platform.AuctionListing
	var stackable int
	if err := row.Scan(
		&listing.AuctionID,
		&listing.RealmID,
		&listing.SellerCharacterID,
		&listing.SellerDisplayName,
		&listing.BuyerCharacterID,
		&listing.ItemID,
		&listing.ItemDisplayName,
		&listing.ItemQuality,
		&listing.ItemType,
		&listing.ItemSubtype,
		&stackable,
		&listing.ItemMaxStack,
		&listing.StackCount,
		&listing.BuyoutCopper,
		&listing.BidCopper,
		&listing.CurrentBidCopper,
		&listing.CurrentBidderCharacterID,
		&listing.DepositCopper,
		&listing.CutCopper,
		&listing.CutPercent,
		&listing.CreatedAt,
		&listing.ExpiresAt,
		&listing.SoldAt,
		&listing.CanceledAt,
		&listing.State,
		&listing.SourceInventorySlot,
		&listing.Version,
		&listing.ItemDeliveredMailID,
		&listing.ProceedsDeliveredMailID,
		&listing.ReturnDeliveredMailID); err != nil {
		return platform.AuctionListing{}, err
	}
	listing.ItemStackable = intToBool(stackable)
	return listing, nil
}

func (tx *Tx) findAuctionListingByMutation(sellerCharacterID string, mutationKey string) (platform.AuctionListing, bool, error) {
	row := tx.tx.QueryRow(
		`SELECT `+auctionColumns()+` FROM ac_auction_listings WHERE seller_character_id = ? AND mutation_key = ? AND mutation_key <> ''`,
		sellerCharacterID,
		mutationKey)
	listing, err := scanAuctionListing(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.AuctionListing{}, false, nil
		}
		return platform.AuctionListing{}, false, err
	}
	return listing, true, nil
}

func (tx *Tx) getAuctionListing(auctionID string) (platform.AuctionListing, error) {
	row := tx.tx.QueryRow(`SELECT `+auctionColumns()+` FROM ac_auction_listings WHERE auction_id = ?`, auctionID)
	listing, err := scanAuctionListing(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.AuctionListing{}, filestore.ErrAuctionMissing
		}
		return platform.AuctionListing{}, err
	}
	return listing, nil
}

func (tx *Tx) insertAuctionListing(listing platform.AuctionListing, mutationKey string) error {
	_, err := tx.tx.Exec(
		`INSERT INTO ac_auction_listings (
			auction_id, realm_id, seller_character_id, seller_display_name, buyer_character_id, item_id,
			item_display_name, item_quality, item_type, item_subtype, item_stackable, item_max_stack,
			stack_count, buyout_copper, bid_copper, current_bid_copper, current_bidder_character_id,
			deposit_copper, cut_copper, cut_percent, created_at, expires_at, sold_at, canceled_at,
			state, source_inventory_slot, version, item_delivered_mail_id, proceeds_delivered_mail_id,
			return_delivered_mail_id, mutation_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		listing.AuctionID,
		listing.RealmID,
		listing.SellerCharacterID,
		listing.SellerDisplayName,
		listing.BuyerCharacterID,
		listing.ItemID,
		listing.ItemDisplayName,
		listing.ItemQuality,
		listing.ItemType,
		listing.ItemSubtype,
		boolToInt(listing.ItemStackable),
		listing.ItemMaxStack,
		listing.StackCount,
		listing.BuyoutCopper,
		listing.BidCopper,
		listing.CurrentBidCopper,
		listing.CurrentBidderCharacterID,
		listing.DepositCopper,
		listing.CutCopper,
		listing.CutPercent,
		listing.CreatedAt,
		listing.ExpiresAt,
		listing.SoldAt,
		listing.CanceledAt,
		listing.State,
		listing.SourceInventorySlot,
		listing.Version,
		listing.ItemDeliveredMailID,
		listing.ProceedsDeliveredMailID,
		listing.ReturnDeliveredMailID,
		mutationKey)
	return err
}

func (tx *Tx) updateAuctionListing(listing platform.AuctionListing) error {
	_, err := tx.tx.Exec(
		`UPDATE ac_auction_listings SET
			buyer_character_id = ?, cut_copper = ?, sold_at = ?, canceled_at = ?, state = ?, version = ?,
			item_delivered_mail_id = ?, proceeds_delivered_mail_id = ?, return_delivered_mail_id = ?
		WHERE auction_id = ?`,
		listing.BuyerCharacterID,
		listing.CutCopper,
		listing.SoldAt,
		listing.CanceledAt,
		listing.State,
		listing.Version,
		listing.ItemDeliveredMailID,
		listing.ProceedsDeliveredMailID,
		listing.ReturnDeliveredMailID,
		listing.AuctionID)
	return err
}

func (tx *Tx) insertAuctionTransaction(auctionID string, kind string, actor string, counterparty string, copper int, itemID string, stackCount int, mutationKey string, now int64) error {
	_, err := tx.tx.Exec(
		`INSERT INTO ac_auction_transactions (
			transaction_id, auction_id, transaction_type, actor_character_id, counterparty_character_id,
			currency_copper, item_id, stack_count, mutation_key, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		randomID("auction_tx"),
		auctionID,
		kind,
		actor,
		counterparty,
		copper,
		itemID,
		stackCount,
		mutationKey,
		now)
	if err != nil && isConstraintError(err) {
		return filestore.ErrDuplicateMutation
	}
	return err
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

func (tx *Tx) insertMail(mail platform.MailEnvelope) error {
	_, err := tx.tx.Exec(
		`INSERT INTO ac_mail_messages (
			mail_id, auction_id, sender_character_id, sender_display_name, recipient_character_id,
			subject, body, created_at, delivered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mail.MailID,
		mail.AuctionID,
		mail.SenderCharacterID,
		mail.SenderDisplayName,
		mail.RecipientCharacterID,
		mail.Subject,
		mail.Body,
		mail.CreatedAt,
		mail.DeliveredAt)
	if err != nil {
		return err
	}
	attachmentIndex := 0
	for _, item := range mail.ItemAttachments {
		if item.ItemID == "" || item.StackCount <= 0 {
			continue
		}
		if _, err := tx.tx.Exec(
			`INSERT INTO ac_mail_attachments (
				attachment_id, mail_id, attachment_index, attachment_kind, item_id, display_name, stack_count
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			randomID("mail_att"),
			mail.MailID,
			attachmentIndex,
			filestore.MailAttachmentKindItem,
			item.ItemID,
			item.DisplayName,
			item.StackCount); err != nil {
			return err
		}
		attachmentIndex++
	}
	if mail.CurrencyCopper > 0 {
		_, err = tx.tx.Exec(
			`INSERT INTO ac_mail_attachments (
				attachment_id, mail_id, attachment_index, attachment_kind, currency_copper
			) VALUES (?, ?, ?, ?, ?)`,
			randomID("mail_att"),
			mail.MailID,
			attachmentIndex,
			filestore.MailAttachmentKindCurrency,
			mail.CurrencyCopper)
	}
	return err
}

func (s *Store) loadMail(mailID string) (platform.MailEnvelope, error) {
	return (&Tx{tx: nil, store: s}).loadMailFromDB(s.db, mailID)
}

func (tx *Tx) loadMail(mailID string) (platform.MailEnvelope, error) {
	return tx.loadMailFromDB(tx.tx, mailID)
}

type queryer interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

func (tx *Tx) loadMailFromDB(db queryer, mailID string) (platform.MailEnvelope, error) {
	row := db.QueryRow(
		`SELECT mail_id, auction_id, sender_character_id, sender_display_name, recipient_character_id,
			subject, body, created_at, delivered_at
		FROM ac_mail_messages WHERE mail_id = ? AND deleted_at = 0`,
		mailID)
	var mail platform.MailEnvelope
	if err := row.Scan(
		&mail.MailID,
		&mail.AuctionID,
		&mail.SenderCharacterID,
		&mail.SenderDisplayName,
		&mail.RecipientCharacterID,
		&mail.Subject,
		&mail.Body,
		&mail.CreatedAt,
		&mail.DeliveredAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return platform.MailEnvelope{}, filestore.ErrMailMissing
		}
		return platform.MailEnvelope{}, err
	}
	rows, err := db.Query(
		`SELECT attachment_kind, item_id, display_name, stack_count, currency_copper
		FROM ac_mail_attachments
		WHERE mail_id = ? AND claimed_at = 0
		ORDER BY attachment_index`,
		mailID)
	if err != nil {
		return platform.MailEnvelope{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var item platform.MailItemAttachment
		var currency int
		if err := rows.Scan(&kind, &item.ItemID, &item.DisplayName, &item.StackCount, &currency); err != nil {
			return platform.MailEnvelope{}, err
		}
		if kind == filestore.MailAttachmentKindCurrency {
			mail.CurrencyCopper += currency
		} else if item.ItemID != "" && item.StackCount > 0 {
			mail.ItemAttachments = append(mail.ItemAttachments, item)
		}
	}
	return mail, rows.Err()
}

func (tx *Tx) loadClaimableMailAttachment(claim filestore.MailAttachmentClaim) (mailAttachmentRow, error) {
	if claim.MutationKey != "" {
		row := tx.tx.QueryRow(
			`SELECT attachment_id, item_id, display_name, stack_count, currency_copper
			FROM ac_mail_attachments
			WHERE mail_id = ? AND claim_mutation_key = ? AND claimed_by_character_id = ?`,
			claim.MailID,
			claim.MutationKey,
			claim.CharacterID)
		var attachment mailAttachmentRow
		if err := row.Scan(&attachment.AttachmentID, &attachment.ItemID, &attachment.DisplayName, &attachment.StackCount, &attachment.CurrencyCopper); err == nil {
			return mailAttachmentRow{}, filestore.ErrMailAttachmentClaimed
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return mailAttachmentRow{}, err
		}
	}
	query := `SELECT attachment_id, item_id, display_name, stack_count, currency_copper
		FROM ac_mail_attachments WHERE mail_id = ? AND claimed_at = 0`
	args := []any{claim.MailID}
	if claim.AttachmentID != "" {
		query += ` AND attachment_id = ?`
		args = append(args, claim.AttachmentID)
	}
	query += ` ORDER BY attachment_index LIMIT 1`
	row := tx.tx.QueryRow(query, args...)
	var attachment mailAttachmentRow
	if err := row.Scan(&attachment.AttachmentID, &attachment.ItemID, &attachment.DisplayName, &attachment.StackCount, &attachment.CurrencyCopper); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mailAttachmentRow{}, filestore.ErrMailAttachmentMissing
		}
		return mailAttachmentRow{}, err
	}
	return attachment, nil
}

func (tx *Tx) insertTransferAudit(operation string, actor string, source string, target string, targetKind string, targetID string, itemID string, stackCount int, currencyDelta int, mutationKey string, result string) error {
	metadataJSON, err := encodeJSON(map[string]any{
		"sourceCharacterId": source,
		"targetCharacterId": target,
		"targetKind":        targetKind,
		"targetId":          targetID,
	})
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(
		`INSERT INTO ac_transfer_audit_events (
			event_id, operation, actor_character_id, source_character_id, target_character_id, target_kind,
			target_id, item_id, stack_count, currency_delta, mutation_key, result_status, metadata_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		randomID("transfer_audit"),
		operation,
		actor,
		source,
		target,
		targetKind,
		targetID,
		itemID,
		stackCount,
		currencyDelta,
		mutationKey,
		result,
		metadataJSON,
		tx.store.now().Unix())
	return err
}

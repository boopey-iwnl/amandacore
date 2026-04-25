package worlds

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	auctionHouseServiceID     = "auction_highmere_market"
	auctionDurationSeconds    = int64(24 * 60 * 60)
	auctionDevDurationSeconds = int64(30 * 60)
	auctionCutPercent         = 5
	auctionMaxBuyoutCopper    = 1000000000
	auctionDefaultBrowseLimit = 50
	auctionInteractionError   = "right-click the market board first"
)

type auctionListRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	SlotIndex         int    `json:"slotIndex"`
	StackCount        int    `json:"stackCount"`
	BuyoutCopper      int    `json:"buyoutCopper"`
	DurationSeconds   int64  `json:"durationSeconds"`
}

type auctionIDRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	AuctionID         string `json:"auctionId"`
}

type auctionStateResponse struct {
	ServerTime     int64                    `json:"serverTime"`
	Listings       []auctionListingResponse `json:"listings"`
	MyAuctions     []auctionListingResponse `json:"myAuctions"`
	Mail           []platform.MailEnvelope  `json:"mail"`
	CurrencyCopper int                      `json:"currencyCopper"`
}

type auctionListingResponse struct {
	AuctionID           string `json:"auctionId"`
	SellerCharacterID   string `json:"sellerCharacterId"`
	SellerDisplayName   string `json:"sellerDisplayName"`
	BuyerCharacterID    string `json:"buyerCharacterId,omitempty"`
	ItemID              string `json:"itemId"`
	ItemDisplayName     string `json:"itemDisplayName"`
	StackCount          int    `json:"stackCount"`
	ItemQuality         string `json:"itemQuality"`
	ItemType            string `json:"itemType"`
	ItemSubtype         string `json:"itemSubtype"`
	BuyoutCopper        int    `json:"buyoutCopper"`
	DepositCopper       int    `json:"depositCopper"`
	CutCopper           int    `json:"cutCopper"`
	CutPercent          int    `json:"cutPercent"`
	CreatedAt           int64  `json:"createdAt"`
	ExpiresAt           int64  `json:"expiresAt"`
	SoldAt              int64  `json:"soldAt,omitempty"`
	CanceledAt          int64  `json:"canceledAt,omitempty"`
	State               string `json:"state"`
	TimeRemainingSecond int64  `json:"timeRemainingSeconds"`
	Version             int    `json:"version"`
}

func (s *worldServer) handleAuctionBrowse(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.validateAuctionAccessLocked(session); err != nil {
		s.recordAuctionFailureLocked("auction.list_failed", session, "", "", 0, 0, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_access_failed", err.Error())
		return
	}

	now := time.Now()
	s.expireAuctionsLocked(now)
	response, err := s.buildAuctionStateResponseLocked(
		session,
		r.URL.Query().Get("search"),
		r.URL.Query().Get("itemType"),
		r.URL.Query().Get("sort"),
		queryInt(r, "limit", auctionDefaultBrowseLimit),
		queryInt(r, "offset", 0),
		now.Unix())
	if err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "auction_browse_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (s *worldServer) handleAuctionMine(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("worldSessionToken")
	if token == "" {
		httpapi.Error(w, http.StatusBadRequest, "missing_token", "worldSessionToken query parameter is required.")
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[token]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.validateAuctionAccessLocked(session); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "auction_access_failed", err.Error())
		return
	}

	now := time.Now()
	s.expireAuctionsLocked(now)
	response, err := s.buildAuctionStateResponseLocked(session, "", "", "", auctionDefaultBrowseLimit, 0, now.Unix())
	if err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "auction_mine_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, response)
}

func (s *worldServer) handleAuctionList(w http.ResponseWriter, r *http.Request) {
	var request auctionListRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.validateAuctionAccessLocked(session); err != nil {
		s.recordAuctionFailureLocked("auction.list_failed", session, "", "", request.StackCount, request.BuyoutCopper, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_access_failed", err.Error())
		return
	}

	now := time.Now()
	s.expireAuctionsLocked(now)
	listing, err := s.buildAuctionListingForRequestLocked(session, request, now.Unix())
	if err != nil {
		s.recordAuctionFailureLocked("auction.list_failed", session, "", "", request.StackCount, request.BuyoutCopper, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_list_failed", err.Error())
		return
	}

	persistStartedAt := time.Now()
	created, character, err := s.store.CreateAuctionListing(listing, request.SlotIndex, request.StackCount)
	s.recordPersistenceDuration("auction_create", persistStartedAt, err)
	if err != nil {
		s.recordAuctionFailureLocked("auction.list_failed", session, "", listing.ItemID, request.StackCount, request.BuyoutCopper, listing.DepositCopper, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_list_failed", err.Error())
		return
	}
	s.applyCharacterProgressionLocked(session, &character)
	observability.LogEvent("world-service", "auction.listed", auctionEventFields(created, "", ""))
	httpapi.WriteJSON(w, http.StatusCreated, s.mustBuildAuctionStateResponseLocked(session, now.Unix()))
}

func (s *worldServer) handleAuctionBuyout(w http.ResponseWriter, r *http.Request) {
	var request auctionIDRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.validateAuctionAccessLocked(session); err != nil {
		s.recordAuctionFailureLocked("auction.purchase_failed", session, request.AuctionID, "", 0, 0, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_access_failed", err.Error())
		return
	}

	now := time.Now()
	s.expireAuctionsLocked(now)
	persistStartedAt := time.Now()
	listing, buyer, seller, _, err := s.store.BuyoutAuction(request.AuctionID, session.CharacterID, now.Unix())
	s.recordPersistenceDuration("auction_buyout", persistStartedAt, err)
	if err != nil {
		s.recordAuctionFailureLocked("auction.purchase_failed", session, request.AuctionID, "", 0, 0, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_buyout_failed", err.Error())
		return
	}
	s.applyCharacterProgressionLocked(session, &buyer)
	if sellerSession := s.findConnectedSessionByCharacterLocked(seller.ID); sellerSession != nil {
		s.applyCharacterProgressionLocked(sellerSession, &seller)
	}
	observability.LogEvent("world-service", "auction.purchased", auctionEventFields(listing, session.CharacterID, ""))
	httpapi.WriteJSON(w, http.StatusOK, s.mustBuildAuctionStateResponseLocked(session, now.Unix()))
}

func (s *worldServer) handleAuctionCancel(w http.ResponseWriter, r *http.Request) {
	var request auctionIDRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if err := s.validateAuctionAccessLocked(session); err != nil {
		s.recordAuctionFailureLocked("auction.cancel_failed", session, request.AuctionID, "", 0, 0, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_access_failed", err.Error())
		return
	}

	now := time.Now()
	s.expireAuctionsLocked(now)
	persistStartedAt := time.Now()
	listing, character, _, err := s.store.CancelAuction(request.AuctionID, session.CharacterID, now.Unix())
	s.recordPersistenceDuration("auction_cancel", persistStartedAt, err)
	if err != nil {
		s.recordAuctionFailureLocked("auction.cancel_failed", session, request.AuctionID, "", 0, 0, 0, 0, err.Error())
		httpapi.Error(w, http.StatusBadRequest, "auction_cancel_failed", err.Error())
		return
	}
	s.applyCharacterProgressionLocked(session, &character)
	observability.LogEvent("world-service", "auction.canceled", auctionEventFields(listing, "", ""))
	httpapi.WriteJSON(w, http.StatusOK, s.mustBuildAuctionStateResponseLocked(session, now.Unix()))
}

func (s *worldServer) validateAuctionAccessLocked(session *worldSessionState) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if session.CurrentTargetID == "" {
		return fmt.Errorf(auctionInteractionError)
	}
	friendly, ok := s.findFriendlyNPCDefinition(session.CurrentTargetID)
	if !ok {
		return fmt.Errorf(auctionInteractionError)
	}
	hasAuctionService := false
	for _, service := range friendly.Services {
		if service.Type == "auction" && service.ServiceID == auctionHouseServiceID {
			hasAuctionService = true
			break
		}
	}
	if !hasAuctionService {
		return fmt.Errorf(auctionInteractionError)
	}
	if !s.friendlyInRangeLocked(session, friendly.ID) {
		return fmt.Errorf("move closer to the market board")
	}
	return nil
}

func (s *worldServer) buildAuctionListingForRequestLocked(
	session *worldSessionState,
	request auctionListRequest,
	now int64,
) (platform.AuctionListing, error) {
	if request.BuyoutCopper <= 0 || request.BuyoutCopper > auctionMaxBuyoutCopper {
		return platform.AuctionListing{}, fmt.Errorf("buyout price must be greater than zero")
	}
	if request.SlotIndex < 0 || request.SlotIndex >= platform.InventorySlotCount {
		return platform.AuctionListing{}, fmt.Errorf("inventory slot is out of range")
	}
	inventory := platform.NormalizeInventorySlots(session.Inventory)
	slot := inventory[request.SlotIndex]
	if slot.ItemID == "" || slot.StackCount <= 0 {
		return platform.AuctionListing{}, fmt.Errorf("inventory slot is empty")
	}
	if request.StackCount <= 0 {
		request.StackCount = slot.StackCount
	}
	if request.StackCount > slot.StackCount {
		return platform.AuctionListing{}, fmt.Errorf("not enough items in slot")
	}
	item, found := findItemDefinition(slot.ItemID)
	if !found {
		return platform.AuctionListing{}, fmt.Errorf("item is not defined")
	}
	if !itemAuctionTradeable(item) {
		return platform.AuctionListing{}, fmt.Errorf("item is not tradeable")
	}
	if !item.Stackable && request.StackCount != 1 {
		return platform.AuctionListing{}, fmt.Errorf("only one non-stackable item can be listed")
	}

	depositCopper := auctionDepositCopper(item, request.StackCount)
	if session.CurrencyCopper < depositCopper {
		return platform.AuctionListing{}, fmt.Errorf("not enough copper for deposit")
	}
	durationSeconds := request.DurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = auctionDurationSeconds
	}
	if durationSeconds < auctionDevDurationSeconds {
		durationSeconds = auctionDevDurationSeconds
	}

	return platform.AuctionListing{
		RealmID:             session.RealmID,
		SellerCharacterID:   session.CharacterID,
		SellerDisplayName:   session.DisplayName,
		ItemID:              item.ItemID,
		ItemDisplayName:     item.DisplayName,
		ItemQuality:         item.Quality,
		ItemType:            item.Type,
		ItemSubtype:         item.Subtype,
		ItemStackable:       item.Stackable,
		ItemMaxStack:        item.MaxStack,
		StackCount:          request.StackCount,
		BuyoutCopper:        request.BuyoutCopper,
		DepositCopper:       depositCopper,
		CutPercent:          auctionCutPercent,
		CreatedAt:           now,
		ExpiresAt:           now + durationSeconds,
		State:               platform.AuctionStateActive,
		SourceInventorySlot: request.SlotIndex,
	}, nil
}

func (s *worldServer) buildAuctionStateResponseLocked(
	session *worldSessionState,
	search string,
	itemType string,
	sortBy string,
	limit int,
	offset int,
	now int64,
) (auctionStateResponse, error) {
	listings, err := s.store.ListAuctionListings(session.RealmID, search, itemType, sortBy, limit, offset)
	if err != nil {
		return auctionStateResponse{}, err
	}
	myAuctions, err := s.store.ListAuctionsForSeller(session.CharacterID)
	if err != nil {
		return auctionStateResponse{}, err
	}
	mail, err := s.store.ListMailForCharacter(session.CharacterID)
	if err != nil {
		return auctionStateResponse{}, err
	}
	return auctionStateResponse{
		ServerTime:     now,
		Listings:       buildAuctionListingResponses(listings, now),
		MyAuctions:     buildAuctionListingResponses(myAuctions, now),
		Mail:           mail,
		CurrencyCopper: session.CurrencyCopper,
	}, nil
}

func (s *worldServer) mustBuildAuctionStateResponseLocked(session *worldSessionState, now int64) auctionStateResponse {
	response, err := s.buildAuctionStateResponseLocked(session, "", "", "", auctionDefaultBrowseLimit, 0, now)
	if err != nil {
		return auctionStateResponse{ServerTime: now, CurrencyCopper: session.CurrencyCopper}
	}
	return response
}

func (s *worldServer) expireAuctionsLocked(now time.Time) {
	expired, characters, _, err := s.store.ExpireAuctions(now.Unix(), 50)
	if err != nil {
		observability.LogEvent("world-service", "auction.expire_failed", map[string]any{"reason": err.Error()})
		return
	}
	for _, character := range characters {
		if session := s.findConnectedSessionByCharacterLocked(character.ID); session != nil {
			s.applyCharacterProgressionLocked(session, &character)
		}
	}
	for _, listing := range expired {
		observability.LogEvent("world-service", "auction.expired", auctionEventFields(listing, "", ""))
	}
}

func (s *worldServer) recordAuctionFailureLocked(
	action string,
	session *worldSessionState,
	auctionID string,
	itemID string,
	stackCount int,
	buyoutCopper int,
	depositCopper int,
	cutCopper int,
	reason string,
) {
	fields := map[string]any{
		"auctionId":     auctionID,
		"itemId":        itemID,
		"stackCount":    stackCount,
		"buyoutCopper":  buyoutCopper,
		"depositCopper": depositCopper,
		"cutCopper":     cutCopper,
		"reason":        reason,
	}
	if session != nil {
		fields["sellerCharacterId"] = session.CharacterID
	}
	observability.LogEvent("world-service", action, fields)
	_ = s.store.AppendAuditEvent(platform.AuditEvent{
		Action:           action,
		ActorAccountID:   sessionAccountID(session),
		ActorCharacterID: sessionCharacterID(session),
		Reason:           reason,
		Metadata:         fields,
	})
}

func buildAuctionListingResponses(listings []platform.AuctionListing, now int64) []auctionListingResponse {
	responses := make([]auctionListingResponse, 0, len(listings))
	for _, listing := range listings {
		remaining := listing.ExpiresAt - now
		if remaining < 0 {
			remaining = 0
		}
		cutCopper := listing.CutCopper
		if cutCopper == 0 && listing.CutPercent > 0 && listing.BuyoutCopper > 0 {
			cutCopper = listing.BuyoutCopper * listing.CutPercent / 100
			if cutCopper <= 0 {
				cutCopper = 1
			}
		}
		responses = append(responses, auctionListingResponse{
			AuctionID:           listing.AuctionID,
			SellerCharacterID:   listing.SellerCharacterID,
			SellerDisplayName:   listing.SellerDisplayName,
			BuyerCharacterID:    listing.BuyerCharacterID,
			ItemID:              listing.ItemID,
			ItemDisplayName:     listing.ItemDisplayName,
			StackCount:          listing.StackCount,
			ItemQuality:         listing.ItemQuality,
			ItemType:            listing.ItemType,
			ItemSubtype:         listing.ItemSubtype,
			BuyoutCopper:        listing.BuyoutCopper,
			DepositCopper:       listing.DepositCopper,
			CutCopper:           cutCopper,
			CutPercent:          listing.CutPercent,
			CreatedAt:           listing.CreatedAt,
			ExpiresAt:           listing.ExpiresAt,
			SoldAt:              listing.SoldAt,
			CanceledAt:          listing.CanceledAt,
			State:               listing.State,
			TimeRemainingSecond: remaining,
			Version:             listing.Version,
		})
	}
	return responses
}

func itemAuctionTradeable(item itemDefinition) bool {
	if item.ItemID == "" || item.Type == itemTypeQuest {
		return false
	}
	return true
}

func auctionDepositCopper(item itemDefinition, stackCount int) int {
	deposit := item.SellPriceCopper * stackCount / 20
	if deposit <= 0 {
		return 1
	}
	return deposit
}

func auctionEventFields(listing platform.AuctionListing, buyerCharacterID string, reason string) map[string]any {
	fields := map[string]any{
		"auctionId":         listing.AuctionID,
		"sellerCharacterId": listing.SellerCharacterID,
		"buyerCharacterId":  buyerCharacterID,
		"itemId":            listing.ItemID,
		"stackCount":        listing.StackCount,
		"buyoutCopper":      listing.BuyoutCopper,
		"depositCopper":     listing.DepositCopper,
		"cutCopper":         listing.CutCopper,
	}
	if reason != "" {
		fields["reason"] = reason
	}
	return fields
}

func sessionAccountID(session *worldSessionState) string {
	if session == nil {
		return ""
	}
	return session.AccountID
}

func sessionCharacterID(session *worldSessionState) string {
	if session == nil {
		return ""
	}
	return session.CharacterID
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

package store

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/platform"
)

type AuditQuery struct {
	ActorAccountID    string
	TargetAccountID   string
	TargetCharacterID string
	Action            string
	From              int64
	To                int64
	Limit             int
}

func (s *FileStore) SearchCharacters(query string, accountID string, realmID string) ([]platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	query = normalize(query)
	results := make([]platform.Character, 0)
	for _, character := range s.state.Characters {
		if accountID != "" && character.AccountID != accountID {
			continue
		}
		if realmID != "" && character.RealmID != realmID {
			continue
		}
		if query != "" &&
			!strings.Contains(normalize(character.DisplayName), query) &&
			!strings.Contains(normalize(character.ID), query) {
			continue
		}
		results = append(results, normalizedCharacterCopy(character))
	}
	sort.Slice(results, func(left int, right int) bool {
		return strings.ToLower(results[left].DisplayName) < strings.ToLower(results[right].DisplayName)
	})
	return results, nil
}

func (s *FileStore) RecordAuditEvent(event platform.AuditEvent) (platform.AuditEvent, error) {
	if err := s.lockState(true); err != nil {
		return platform.AuditEvent{}, err
	}
	defer s.unlockState()

	if event.ID == "" {
		event.ID = randomID("audit")
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	if s.state.AuditEvents == nil {
		s.state.AuditEvents = map[string]platform.AuditEvent{}
	}
	s.state.AuditEvents[event.ID] = event
	return event, s.saveLocked()
}

func (s *FileStore) QueryAuditEvents(query AuditQuery) ([]platform.AuditEvent, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := make([]platform.AuditEvent, 0)
	for _, event := range s.state.AuditEvents {
		if query.ActorAccountID != "" && event.ActorAccountID != query.ActorAccountID {
			continue
		}
		if query.TargetAccountID != "" && event.TargetAccountID != query.TargetAccountID {
			continue
		}
		if query.TargetCharacterID != "" && event.TargetCharacterID != query.TargetCharacterID {
			continue
		}
		if query.Action != "" && event.Action != query.Action {
			continue
		}
		if query.From > 0 && event.Timestamp < query.From {
			continue
		}
		if query.To > 0 && event.Timestamp > query.To {
			continue
		}
		results = append(results, event)
	}
	sort.Slice(results, func(left int, right int) bool {
		return results[left].Timestamp > results[right].Timestamp
	})
	if query.Limit <= 0 || query.Limit > 200 {
		query.Limit = 100
	}
	if len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return results, nil
}

func (s *FileStore) CreateSupportTicket(accountID string, characterID string, category string, subject string, body string, diagnosticsID string, buildID string, clientVersion string) (platform.SupportTicket, error) {
	if err := s.lockState(true); err != nil {
		return platform.SupportTicket{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok || character.AccountID != accountID {
		return platform.SupportTicket{}, fmt.Errorf("character not found for account")
	}

	category = strings.TrimSpace(category)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if category == "" {
		category = "general"
	}
	if subject == "" {
		return platform.SupportTicket{}, fmt.Errorf("subject is required")
	}
	if body == "" {
		return platform.SupportTicket{}, fmt.Errorf("body is required")
	}

	now := time.Now().Unix()
	ticket := platform.SupportTicket{
		TicketID:              randomID("ticket"),
		CreatedByCharacterID:  characterID,
		CreatedByAccountID:    accountID,
		Category:              category,
		Subject:               subject,
		Body:                  body,
		Status:                platform.SupportTicketOpen,
		CreatedAt:             now,
		UpdatedAt:             now,
		AttachedDiagnosticsID: strings.TrimSpace(diagnosticsID),
		BuildID:               strings.TrimSpace(buildID),
		ClientVersion:         strings.TrimSpace(clientVersion),
		Notes:                 []platform.SupportTicketNote{},
	}
	if s.state.SupportTickets == nil {
		s.state.SupportTickets = map[string]platform.SupportTicket{}
	}
	s.state.SupportTickets[ticket.TicketID] = ticket
	return ticket, s.saveLocked()
}

func (s *FileStore) ListSupportTickets(status string) ([]platform.SupportTicket, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	status = strings.TrimSpace(status)
	results := make([]platform.SupportTicket, 0)
	for _, ticket := range s.state.SupportTickets {
		if status != "" && string(ticket.Status) != status {
			continue
		}
		results = append(results, ticket)
	}
	sort.Slice(results, func(left int, right int) bool {
		return results[left].UpdatedAt > results[right].UpdatedAt
	})
	return results, nil
}

func (s *FileStore) GetSupportTicket(ticketID string) (*platform.SupportTicket, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	ticket, ok := s.state.SupportTickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("support ticket not found")
	}
	copy := ticket
	copy.Notes = append([]platform.SupportTicketNote(nil), ticket.Notes...)
	return &copy, nil
}

func (s *FileStore) UpdateSupportTicket(ticketID string, adminAccountID string, status platform.SupportTicketStatus, assignedToAdminID string, resolutionNote string, noteBody string) (platform.SupportTicket, error) {
	if err := s.lockState(true); err != nil {
		return platform.SupportTicket{}, err
	}
	defer s.unlockState()

	ticket, ok := s.state.SupportTickets[ticketID]
	if !ok {
		return platform.SupportTicket{}, fmt.Errorf("support ticket not found")
	}
	if status != "" {
		if !validTicketStatus(status) {
			return platform.SupportTicket{}, fmt.Errorf("invalid ticket status")
		}
		ticket.Status = status
	}
	if strings.TrimSpace(assignedToAdminID) != "" {
		ticket.AssignedToAdminID = strings.TrimSpace(assignedToAdminID)
	}
	if strings.TrimSpace(resolutionNote) != "" {
		ticket.ResolutionNote = strings.TrimSpace(resolutionNote)
	}
	if strings.TrimSpace(noteBody) != "" {
		ticket.Notes = append(ticket.Notes, platform.SupportTicketNote{
			NoteID:          randomID("note"),
			TicketID:        ticket.TicketID,
			AuthorAccountID: adminAccountID,
			Body:            strings.TrimSpace(noteBody),
			CreatedAt:       time.Now().Unix(),
		})
	}
	ticket.UpdatedAt = time.Now().Unix()
	s.state.SupportTickets[ticket.TicketID] = ticket
	return ticket, s.saveLocked()
}

func (s *FileStore) SetAccountSuspension(accountID string, suspended bool, reason string, durationSeconds int64) (platform.Account, platform.Account, error) {
	if err := s.lockState(true); err != nil {
		return platform.Account{}, platform.Account{}, err
	}
	defer s.unlockState()

	account, ok := s.state.Accounts[accountID]
	if !ok {
		return platform.Account{}, platform.Account{}, ErrInvalidCredentials
	}
	before := account
	now := time.Now().Unix()
	if suspended {
		if durationSeconds > 0 {
			account.SuspendedUntil = now + durationSeconds
			account.Banned = false
		} else {
			account.Banned = true
			account.SuspendedUntil = 0
		}
		account.SuspensionReason = strings.TrimSpace(reason)
	} else {
		account.Banned = false
		account.SuspendedUntil = 0
		account.SuspensionReason = ""
	}
	account.UpdatedAt = now
	s.state.Accounts[accountID] = account
	return before, account, s.saveLocked()
}

func (s *FileStore) RevokeAccountSessions(accountID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	changed := false
	for sessionID, session := range s.state.Sessions {
		if session.AccountID == accountID {
			delete(s.state.Sessions, sessionID)
			changed = true
		}
	}
	for ticketID, ticket := range s.state.WorldJoinTickets {
		if ticket.AccountID == accountID {
			delete(s.state.WorldJoinTickets, ticketID)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

func (s *FileStore) RevokeCharacterJoinTickets(characterID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	changed := false
	for ticketID, ticket := range s.state.WorldJoinTickets {
		if ticket.CharacterID == characterID {
			delete(s.state.WorldJoinTickets, ticketID)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

func (s *FileStore) SetCharacterMute(characterID string, actorAccountID string, reason string, durationSeconds int64) (platform.MuteRecord, error) {
	if durationSeconds <= 0 {
		return platform.MuteRecord{}, fmt.Errorf("mute duration must be positive")
	}
	if err := s.lockState(true); err != nil {
		return platform.MuteRecord{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.MuteRecord{}, fmt.Errorf("character not found")
	}
	now := time.Now().Unix()
	record := platform.MuteRecord{
		CharacterID:      characterID,
		AccountID:        character.AccountID,
		MutedByAccountID: actorAccountID,
		Reason:           strings.TrimSpace(reason),
		CreatedAt:        now,
		ExpiresAt:        now + durationSeconds,
	}
	if s.state.Mutes == nil {
		s.state.Mutes = map[string]platform.MuteRecord{}
	}
	s.state.Mutes[characterID] = record
	return record, s.saveLocked()
}

func (s *FileStore) ClearCharacterMute(characterID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	delete(s.state.Mutes, characterID)
	return s.saveLocked()
}

func (s *FileStore) ActiveMuteForCharacter(characterID string) (*platform.MuteRecord, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	record, ok := s.state.Mutes[characterID]
	if !ok {
		return nil, nil
	}
	now := time.Now().Unix()
	if record.ExpiresAt > 0 && record.ExpiresAt <= now {
		delete(s.state.Mutes, characterID)
		_ = s.saveLocked()
		return nil, nil
	}
	copy := record
	return &copy, nil
}

func (s *FileStore) GetHousingForCharacter(characterID string) (*platform.HousingEntitlement, *platform.HousingSpace, []platform.HousingStorageSlot, []platform.DecorationPlacement, error) {
	if err := s.lockState(true); err != nil {
		return nil, nil, nil, nil, err
	}
	defer s.unlockState()

	entitlement, ok := s.state.HousingEntitlements[characterID]
	if !ok {
		return nil, nil, []platform.HousingStorageSlot{}, []platform.DecorationPlacement{}, nil
	}
	space, ok := s.state.HousingSpaces[entitlement.HousingSpaceID]
	if !ok {
		return &entitlement, nil, []platform.HousingStorageSlot{}, []platform.DecorationPlacement{}, nil
	}
	storage := cloneHousingStorageSlots(s.state.HousingStorage[space.HousingSpaceID])
	decorations := cloneDecorationPlacements(s.state.HousingDecorations[space.HousingSpaceID])
	return &entitlement, &space, storage, decorations, nil
}

func (s *FileStore) GrantCharacterItem(characterID string, itemID string, displayName string, quantity int, maxStack int, stackable bool) (platform.Character, platform.Character, error) {
	if quantity <= 0 {
		return platform.Character{}, platform.Character{}, fmt.Errorf("quantity must be positive")
	}
	if maxStack <= 0 {
		maxStack = 1
	}
	if !stackable {
		maxStack = 1
	}
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	inventory, err := addAdminItem(character.Inventory, itemID, displayName, quantity, maxStack, stackable)
	if err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	character.Inventory = inventory
	character.LastSeenAt = time.Now().Unix()
	character = platform.NormalizeCharacter(character)
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func (s *FileStore) RemoveCharacterItem(characterID string, itemID string, quantity int) (platform.Character, platform.Character, error) {
	if quantity <= 0 {
		return platform.Character{}, platform.Character{}, fmt.Errorf("quantity must be positive")
	}
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	inventory, err := removeAdminItem(character.Inventory, itemID, quantity)
	if err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	character.Inventory = inventory
	character.LastSeenAt = time.Now().Unix()
	character = platform.NormalizeCharacter(character)
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func (s *FileStore) ChangeCharacterCurrency(characterID string, deltaCopper int) (platform.Character, platform.Character, error) {
	if deltaCopper == 0 {
		return platform.Character{}, platform.Character{}, fmt.Errorf("currency delta must be non-zero")
	}
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	nextCurrency := character.CurrencyCopper + deltaCopper
	if nextCurrency < 0 {
		return platform.Character{}, platform.Character{}, fmt.Errorf("currency cannot go negative")
	}
	character.CurrencyCopper = nextCurrency
	character.LastSeenAt = time.Now().Unix()
	character = platform.NormalizeCharacter(character)
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func (s *FileStore) ResetCharacterQuest(characterID string, questID string, targetCount int) (platform.Character, platform.Character, error) {
	if strings.TrimSpace(questID) == "" {
		return platform.Character{}, platform.Character{}, fmt.Errorf("questId is required")
	}
	if targetCount <= 0 {
		targetCount = 1
	}
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	if character.Quests == nil {
		character.Quests = map[string]platform.CharacterQuestProgress{}
	}
	character.Quests[questID] = platform.CharacterQuestProgress{
		QuestID:      questID,
		State:        "not_started",
		TargetCount:  targetCount,
		CurrentCount: 0,
		UpdatedAt:    time.Now().Unix(),
	}
	character.TrackedQuestIDs = removeStringID(character.TrackedQuestIDs, questID)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func (s *FileStore) CompleteCharacterQuestObjective(characterID string, questID string, targetCount int) (platform.Character, platform.Character, error) {
	if strings.TrimSpace(questID) == "" {
		return platform.Character{}, platform.Character{}, fmt.Errorf("questId is required")
	}
	if targetCount <= 0 {
		targetCount = 1
	}
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	if character.Quests == nil {
		character.Quests = map[string]platform.CharacterQuestProgress{}
	}
	progress := character.Quests[questID]
	if progress.RewardGrantedAt != 0 {
		return platform.Character{}, platform.Character{}, fmt.Errorf("quest reward was already granted")
	}
	now := time.Now().Unix()
	progress.QuestID = questID
	progress.State = "completed"
	progress.CurrentCount = targetCount
	progress.TargetCount = targetCount
	if progress.AcceptedAt == 0 {
		progress.AcceptedAt = now
	}
	progress.CompletedAt = now
	progress.UpdatedAt = now
	character.Quests[questID] = progress
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = now
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func (s *FileStore) NormalizeCharacterForAdmin(characterID string, restoreStarterAbilities bool, rebuildActionBar bool) (platform.Character, platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return platform.Character{}, platform.Character{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.Character{}, platform.Character{}, fmt.Errorf("character not found")
	}
	before := normalizedCharacterCopy(character)
	if restoreStarterAbilities {
		character.LearnedAbilityIDs = platform.NormalizeLearnedAbilityIDs(
			append(character.LearnedAbilityIDs, platform.DefaultStartingLearnedAbilityIDs()...))
	}
	if rebuildActionBar {
		character.ActionBarSlots = platform.DefaultActionBarSlots(character.LearnedAbilityIDs)
	}
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character
	after := normalizedCharacterCopy(character)
	return before, after, s.saveLocked()
}

func validTicketStatus(status platform.SupportTicketStatus) bool {
	switch status {
	case platform.SupportTicketOpen, platform.SupportTicketInReview, platform.SupportTicketResolved, platform.SupportTicketClosed:
		return true
	default:
		return false
	}
}

func addAdminItem(source []platform.CharacterInventorySlot, itemID string, displayName string, quantity int, maxStack int, stackable bool) ([]platform.CharacterInventorySlot, error) {
	itemID = strings.TrimSpace(itemID)
	displayName = strings.TrimSpace(displayName)
	if itemID == "" {
		return nil, fmt.Errorf("itemId is required")
	}
	slots := platform.NormalizeInventorySlots(source)
	remaining := quantity
	if stackable {
		for index := range slots {
			if slots[index].ItemID != itemID || slots[index].StackCount >= maxStack {
				continue
			}
			added := minAdminInt(remaining, maxStack-slots[index].StackCount)
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
		if stackable {
			added = minAdminInt(remaining, maxStack)
		}
		slots[index] = platform.CharacterInventorySlot{
			SlotIndex:   index,
			ItemID:      itemID,
			DisplayName: displayName,
			StackCount:  added,
		}
		remaining -= added
		if remaining <= 0 {
			return slots, nil
		}
	}
	return nil, fmt.Errorf("inventory capacity is not sufficient")
}

func removeAdminItem(source []platform.CharacterInventorySlot, itemID string, quantity int) ([]platform.CharacterInventorySlot, error) {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return nil, fmt.Errorf("itemId is required")
	}
	slots := platform.NormalizeInventorySlots(source)
	total := 0
	for _, slot := range slots {
		if slot.ItemID == itemID {
			total += slot.StackCount
		}
	}
	if total < quantity {
		return nil, fmt.Errorf("not enough items")
	}
	remaining := quantity
	for index := range slots {
		if slots[index].ItemID != itemID || slots[index].StackCount <= 0 {
			continue
		}
		removed := minAdminInt(remaining, slots[index].StackCount)
		slots[index].StackCount -= removed
		remaining -= removed
		if slots[index].StackCount <= 0 {
			slots[index] = platform.CharacterInventorySlot{SlotIndex: index}
		}
		if remaining <= 0 {
			return slots, nil
		}
	}
	return nil, fmt.Errorf("not enough items")
}

func minAdminInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func removeStringID(source []string, target string) []string {
	next := make([]string, 0, len(source))
	for _, value := range source {
		if value != target {
			next = append(next, value)
		}
	}
	return next
}

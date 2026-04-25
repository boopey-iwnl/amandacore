package store

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"golang.org/x/crypto/argon2"

	"amandacore/services/internal/platform"
)

var (
	ErrAccountExists       = errors.New("account already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountBanned       = errors.New("account is banned")
	ErrSessionExpired      = errors.New("session has expired")
	ErrCharacterNameExists = errors.New("character name already exists in this realm")
	ErrJoinTicketConsumed  = errors.New("join ticket has already been consumed")
	ErrFriendExists        = errors.New("friend entry already exists")
	ErrFriendSelf          = errors.New("cannot add yourself as a friend")
	ErrFriendMissing       = errors.New("friend entry does not exist")
	ErrPartyMissing        = errors.New("party does not exist")
)

type state struct {
	Accounts         map[string]platform.Account             `json:"accounts"`
	Realms           map[string]platform.Realm               `json:"realms"`
	Characters       map[string]platform.Character           `json:"characters"`
	Sessions         map[string]platform.Session             `json:"sessions"`
	WorldJoinTickets map[string]platform.WorldJoinTicket     `json:"worldJoinTickets"`
	PasswordReset    map[string]platform.PasswordResetTicket `json:"passwordReset"`
	Friends          map[string]platform.FriendRelationship  `json:"friends"`
	Parties          map[string]platform.Party               `json:"parties"`
	BuildManifest    platform.BuildManifest                  `json:"buildManifest"`
}

type FileStore struct {
	path     string
	fileLock *flock.Flock
	mutex    sync.Mutex
	state    state
}

func NewFileStore(path string, buildID string, worldEndpoint string) (*FileStore, error) {
	buildManifest := defaultBuildManifest(buildID, worldEndpoint)
	fileStore := &FileStore{
		path:     path,
		fileLock: flock.New(path + ".lock"),
		state: state{
			Accounts:         map[string]platform.Account{},
			Realms:           map[string]platform.Realm{},
			Characters:       map[string]platform.Character{},
			Sessions:         map[string]platform.Session{},
			WorldJoinTickets: map[string]platform.WorldJoinTicket{},
			PasswordReset:    map[string]platform.PasswordResetTicket{},
			Friends:          map[string]platform.FriendRelationship{},
			Parties:          map[string]platform.Party{},
			BuildManifest:    buildManifest,
		},
	}

	if err := fileStore.lockState(true); err != nil {
		return nil, err
	}
	defer fileStore.unlockState()

	fileStore.applyRuntimeBuildManifest(buildManifest, worldEndpoint)
	return fileStore, fileStore.saveLocked()
}

func (s *FileStore) RegisterAccount(username string, password string) (platform.Account, error) {
	if err := s.lockState(true); err != nil {
		return platform.Account{}, err
	}
	defer s.unlockState()

	normalized := normalize(username)
	for _, account := range s.state.Accounts {
		if normalize(account.Username) == normalized {
			return platform.Account{}, ErrAccountExists
		}
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return platform.Account{}, err
	}

	now := time.Now().Unix()
	account := platform.Account{
		ID:           randomID("acct"),
		Username:     username,
		PasswordHash: passwordHash,
		Roles:        []platform.Role{platform.RolePlayer},
		Banned:       false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.state.Accounts[account.ID] = account
	return account, s.saveLocked()
}

func (s *FileStore) EnsureAdminSeed(username string, password string) error {
	if strings.TrimSpace(password) == "" {
		return nil
	}

	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	normalized := normalize(username)
	for id, account := range s.state.Accounts {
		if normalize(account.Username) == normalized {
			passwordHash, err := hashPassword(password)
			if err != nil {
				return err
			}

			account.PasswordHash = passwordHash
			account.Banned = false
			if !containsRole(account.Roles, platform.RoleAdministrator) {
				account.Roles = append(account.Roles, platform.RoleAdministrator)
			}
			account.UpdatedAt = time.Now().Unix()
			s.state.Accounts[id] = account
			return s.saveLocked()
		}
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	account := platform.Account{
		ID:           randomID("acct"),
		Username:     username,
		PasswordHash: passwordHash,
		Roles:        []platform.Role{platform.RolePlayer, platform.RoleAdministrator},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.state.Accounts[account.ID] = account
	return s.saveLocked()
}

func (s *FileStore) Authenticate(username string, password string) (platform.Account, error) {
	if err := s.lockState(true); err != nil {
		return platform.Account{}, err
	}
	defer s.unlockState()

	normalized := normalize(username)
	for _, account := range s.state.Accounts {
		if normalize(account.Username) != normalized {
			continue
		}

		if account.Banned {
			return platform.Account{}, ErrAccountBanned
		}

		if !verifyPassword(account.PasswordHash, password) {
			return platform.Account{}, ErrInvalidCredentials
		}

		return account, nil
	}

	return platform.Account{}, ErrInvalidCredentials
}

func (s *FileStore) CreateSession(accountID string) (platform.Session, error) {
	if err := s.lockState(true); err != nil {
		return platform.Session{}, err
	}
	defer s.unlockState()

	now := time.Now()
	session := platform.Session{
		ID:               randomID("sess"),
		AccountID:        accountID,
		AccessToken:      randomToken(),
		RefreshToken:     randomToken(),
		AccessExpiresAt:  now.Add(30 * time.Minute).Unix(),
		RefreshExpiresAt: now.Add(7 * 24 * time.Hour).Unix(),
		CreatedAt:        now.Unix(),
	}

	s.state.Sessions[session.ID] = session
	return session, s.saveLocked()
}

func (s *FileStore) RefreshSession(refreshToken string) (platform.Session, error) {
	if err := s.lockState(true); err != nil {
		return platform.Session{}, err
	}
	defer s.unlockState()

	now := time.Now().Unix()
	for id, session := range s.state.Sessions {
		if session.RefreshToken != refreshToken {
			continue
		}

		if session.RefreshExpiresAt < now {
			delete(s.state.Sessions, id)
			_ = s.saveLocked()
			return platform.Session{}, ErrSessionExpired
		}

		session.AccessToken = randomToken()
		session.RefreshToken = randomToken()
		session.AccessExpiresAt = time.Now().Add(30 * time.Minute).Unix()
		session.RefreshExpiresAt = time.Now().Add(7 * 24 * time.Hour).Unix()
		s.state.Sessions[id] = session
		return session, s.saveLocked()
	}

	return platform.Session{}, ErrInvalidCredentials
}

func (s *FileStore) ValidateAccessToken(token string) (*platform.Session, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	now := time.Now().Unix()
	for _, session := range s.state.Sessions {
		if session.AccessToken != token {
			continue
		}

		if session.AccessExpiresAt < now {
			return nil, ErrSessionExpired
		}

		copy := session
		return &copy, nil
	}

	return nil, ErrInvalidCredentials
}

func (s *FileStore) RevokeSession(token string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	for id, session := range s.state.Sessions {
		if session.AccessToken == token || session.RefreshToken == token {
			delete(s.state.Sessions, id)
			return s.saveLocked()
		}
	}

	return nil
}

func (s *FileStore) ChangePassword(accountID string, currentPassword string, newPassword string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	account, ok := s.state.Accounts[accountID]
	if !ok {
		return ErrInvalidCredentials
	}

	if !verifyPassword(account.PasswordHash, currentPassword) {
		return ErrInvalidCredentials
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	account.PasswordHash = hash
	account.UpdatedAt = time.Now().Unix()
	s.state.Accounts[accountID] = account
	return s.saveLocked()
}

func (s *FileStore) StartPasswordReset(username string) (platform.PasswordResetTicket, error) {
	if err := s.lockState(true); err != nil {
		return platform.PasswordResetTicket{}, err
	}
	defer s.unlockState()

	normalized := normalize(username)
	for _, account := range s.state.Accounts {
		if normalize(account.Username) != normalized {
			continue
		}

		ticket := platform.PasswordResetTicket{
			ID:        randomID("reset"),
			AccountID: account.ID,
			ExpiresAt: time.Now().Add(15 * time.Minute).Unix(),
		}
		s.state.PasswordReset[ticket.ID] = ticket
		return ticket, s.saveLocked()
	}

	return platform.PasswordResetTicket{}, ErrInvalidCredentials
}

func (s *FileStore) GetAccountByID(accountID string) (*platform.Account, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	account, ok := s.state.Accounts[accountID]
	if !ok {
		return nil, ErrInvalidCredentials
	}

	copy := account
	return &copy, nil
}

func (s *FileStore) ListAccounts() ([]platform.Account, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := make([]platform.Account, 0, len(s.state.Accounts))
	for _, account := range s.state.Accounts {
		results = append(results, account)
	}
	return results, nil
}

func (s *FileStore) SetAccountBanned(accountID string, banned bool) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	account, ok := s.state.Accounts[accountID]
	if !ok {
		return ErrInvalidCredentials
	}

	account.Banned = banned
	account.UpdatedAt = time.Now().Unix()
	s.state.Accounts[accountID] = account
	return s.saveLocked()
}

func (s *FileStore) SetAccountRole(accountID string, role platform.Role) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	account, ok := s.state.Accounts[accountID]
	if !ok {
		return ErrInvalidCredentials
	}

	if !containsRole(account.Roles, role) {
		account.Roles = append(account.Roles, role)
		account.UpdatedAt = time.Now().Unix()
		s.state.Accounts[accountID] = account
	}

	return s.saveLocked()
}

func (s *FileStore) ListRealms() ([]platform.Realm, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	realms := make([]platform.Realm, 0, len(s.state.Realms))
	for _, realm := range s.state.Realms {
		realms = append(realms, realm)
	}
	return realms, nil
}

func (s *FileStore) GetBuildManifest() platform.BuildManifest {
	if err := s.lockState(true); err != nil {
		return s.state.BuildManifest
	}
	defer s.unlockState()
	return s.state.BuildManifest
}

func (s *FileStore) ListCharacters(accountID string, realmID string) ([]platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := []platform.Character{}
	for _, character := range s.state.Characters {
		if character.AccountID != accountID {
			continue
		}
		if realmID != "" && character.RealmID != realmID {
			continue
		}
		results = append(results, normalizedCharacterCopy(character))
	}

	return results, nil
}

func (s *FileStore) CreateCharacter(accountID string, realmID string, displayName string, raceID string, classID string, archetypeID string) (platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return platform.Character{}, err
	}
	defer s.unlockState()

	for _, character := range s.state.Characters {
		if character.RealmID == realmID && strings.EqualFold(character.DisplayName, displayName) {
			return platform.Character{}, ErrCharacterNameExists
		}
	}

	archetypeID, raceID, classID = platform.NormalizeCharacterIdentity(archetypeID, raceID, classID)

	character := platform.Character{
		ID:              randomID("char"),
		AccountID:       accountID,
		RealmID:         realmID,
		DisplayName:     displayName,
		RaceID:          raceID,
		ClassID:         classID,
		ArchetypeID:     archetypeID,
		Level:           1,
		Experience:      0,
		CurrencyCopper:  platform.StarterCurrencyCopper,
		ZoneID:          platform.DefaultStarterZoneID,
		PositionX:       platform.DefaultStarterSpawnX,
		PositionY:       platform.DefaultStarterSpawnY,
		PositionZ:       platform.DefaultStarterSpawnZ,
		Inventory:       platform.DefaultStarterInventory(),
		Equipment:       platform.DefaultEquipmentSlots(),
		Professions:     []platform.CharacterProfessionState{},
		Talents:         map[string]int{},
		Quests:          map[string]platform.CharacterQuestProgress{},
		TrackedQuestIDs: []string{},
		LastSeenAt:      time.Now().Unix(),
	}

	character = platform.NormalizeCharacter(character)
	s.state.Characters[character.ID] = character
	copy := normalizedCharacterCopy(character)
	return copy, s.saveLocked()
}

func (s *FileStore) IssueWorldJoinTicket(accountID string, sessionID string, characterID string, realmID string) (platform.WorldJoinTicket, error) {
	if err := s.lockState(true); err != nil {
		return platform.WorldJoinTicket{}, err
	}
	defer s.unlockState()

	realm, ok := s.state.Realms[realmID]
	if !ok {
		return platform.WorldJoinTicket{}, fmt.Errorf("realm not found")
	}

	character, ok := s.state.Characters[characterID]
	if !ok || character.AccountID != accountID || character.RealmID != realmID {
		return platform.WorldJoinTicket{}, fmt.Errorf("character not available for realm")
	}

	ticket := platform.WorldJoinTicket{
		TicketID:      randomID("ticket"),
		SessionID:     sessionID,
		AccountID:     accountID,
		CharacterID:   characterID,
		RealmID:       realmID,
		WorldEndpoint: realm.Endpoint,
		ExpiresAt:     time.Now().Add(2 * time.Minute).Unix(),
	}

	s.state.WorldJoinTickets[ticket.TicketID] = ticket
	return ticket, s.saveLocked()
}

func (s *FileStore) ValidateWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	ticket, ok := s.state.WorldJoinTickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("join ticket not found")
	}

	if ticket.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf("join ticket expired")
	}
	if ticket.ConsumedAt != 0 {
		return nil, ErrJoinTicketConsumed
	}

	copy := ticket
	return &copy, nil
}

func (s *FileStore) ConsumeWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	ticket, ok := s.state.WorldJoinTickets[ticketID]
	if !ok {
		return nil, fmt.Errorf("join ticket not found")
	}

	now := time.Now().Unix()
	if ticket.ExpiresAt < now {
		return nil, fmt.Errorf("join ticket expired")
	}
	if ticket.ConsumedAt != 0 {
		return nil, ErrJoinTicketConsumed
	}

	ticket.ConsumedAt = now
	s.state.WorldJoinTickets[ticketID] = ticket
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := ticket
	return &copy, nil
}

func (s *FileStore) GetCharacterByID(characterID string) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) GetCharacterByName(realmID string, displayName string) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	normalizedName := normalize(displayName)
	for _, character := range s.state.Characters {
		if realmID != "" && character.RealmID != realmID {
			continue
		}
		if normalize(character.DisplayName) != normalizedName {
			continue
		}

		copy := normalizedCharacterCopy(character)
		return &copy, nil
	}

	return nil, fmt.Errorf("character not found")
}

func (s *FileStore) AddFriend(ownerCharacterID string, friendCharacterID string) (platform.FriendRelationship, error) {
	if err := s.lockState(true); err != nil {
		return platform.FriendRelationship{}, err
	}
	defer s.unlockState()

	owner, ok := s.state.Characters[ownerCharacterID]
	if !ok {
		return platform.FriendRelationship{}, fmt.Errorf("owner character not found")
	}
	friend, ok := s.state.Characters[friendCharacterID]
	if !ok {
		return platform.FriendRelationship{}, fmt.Errorf("friend character not found")
	}
	if owner.ID == friend.ID {
		return platform.FriendRelationship{}, ErrFriendSelf
	}
	if owner.RealmID != friend.RealmID {
		return platform.FriendRelationship{}, fmt.Errorf("friend is not on this realm")
	}

	key := friendKey(owner.ID, friend.ID)
	if _, exists := s.state.Friends[key]; exists {
		return platform.FriendRelationship{}, ErrFriendExists
	}

	relationship := platform.FriendRelationship{
		OwnerCharacterID:  owner.ID,
		FriendCharacterID: friend.ID,
		FriendDisplayName: friend.DisplayName,
		CreatedAt:         time.Now().Unix(),
	}
	s.state.Friends[key] = relationship
	return relationship, s.saveLocked()
}

func (s *FileStore) RemoveFriend(ownerCharacterID string, friendCharacterID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	key := friendKey(ownerCharacterID, friendCharacterID)
	if _, exists := s.state.Friends[key]; !exists {
		return ErrFriendMissing
	}

	delete(s.state.Friends, key)
	return s.saveLocked()
}

func (s *FileStore) ListFriends(ownerCharacterID string) ([]platform.FriendRelationship, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := make([]platform.FriendRelationship, 0)
	for _, relationship := range s.state.Friends {
		if relationship.OwnerCharacterID != ownerCharacterID {
			continue
		}
		if friend, ok := s.state.Characters[relationship.FriendCharacterID]; ok {
			relationship.FriendDisplayName = friend.DisplayName
		}
		results = append(results, relationship)
	}
	return results, nil
}

func (s *FileStore) CreateParty(leaderCharacterID string, memberCharacterIDs []string) (platform.Party, error) {
	if err := s.lockState(true); err != nil {
		return platform.Party{}, err
	}
	defer s.unlockState()

	if _, ok := s.state.Characters[leaderCharacterID]; !ok {
		return platform.Party{}, fmt.Errorf("leader character not found")
	}

	members := normalizePartyMembers(leaderCharacterID, memberCharacterIDs)
	now := time.Now().Unix()
	party := platform.Party{
		ID:                 randomID("party"),
		LeaderCharacterID:  leaderCharacterID,
		MemberCharacterIDs: members,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.state.Parties[party.ID] = party
	return party, s.saveLocked()
}

func (s *FileStore) SaveParty(party platform.Party) (platform.Party, error) {
	if err := s.lockState(true); err != nil {
		return platform.Party{}, err
	}
	defer s.unlockState()

	if party.ID == "" {
		return platform.Party{}, ErrPartyMissing
	}
	if _, exists := s.state.Parties[party.ID]; !exists {
		return platform.Party{}, ErrPartyMissing
	}
	party.MemberCharacterIDs = normalizePartyMembers(party.LeaderCharacterID, party.MemberCharacterIDs)
	if len(party.MemberCharacterIDs) == 0 {
		delete(s.state.Parties, party.ID)
		return party, s.saveLocked()
	}
	if !containsString(party.MemberCharacterIDs, party.LeaderCharacterID) {
		party.LeaderCharacterID = party.MemberCharacterIDs[0]
	}
	party.UpdatedAt = time.Now().Unix()
	s.state.Parties[party.ID] = party
	return party, s.saveLocked()
}

func (s *FileStore) DeleteParty(partyID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	if _, exists := s.state.Parties[partyID]; !exists {
		return ErrPartyMissing
	}
	delete(s.state.Parties, partyID)
	return s.saveLocked()
}

func (s *FileStore) GetPartyByID(partyID string) (*platform.Party, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	party, ok := s.state.Parties[partyID]
	if !ok {
		return nil, ErrPartyMissing
	}
	copy := party
	copy.MemberCharacterIDs = append([]string(nil), party.MemberCharacterIDs...)
	return &copy, nil
}

func (s *FileStore) GetPartyForCharacter(characterID string) (*platform.Party, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	for _, party := range s.state.Parties {
		if !containsString(party.MemberCharacterIDs, characterID) {
			continue
		}
		copy := party
		copy.MemberCharacterIDs = append([]string(nil), party.MemberCharacterIDs...)
		return &copy, nil
	}

	return nil, ErrPartyMissing
}

func (s *FileStore) UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.ZoneID = zoneID
	character.PositionX = x
	character.PositionY = y
	character.PositionZ = z
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterProgression(
	characterID string,
	experience int,
	currencyCopper int,
	inventory []platform.CharacterInventorySlot,
	learnedAbilityIDs []string,
	actionBarSlots []platform.CharacterActionBarSlot,
	quests map[string]platform.CharacterQuestProgress,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.Experience = experience
	character.CurrencyCopper = currencyCopper
	character.Inventory = cloneInventorySlots(inventory)
	character.LearnedAbilityIDs = append([]string(nil), learnedAbilityIDs...)
	character.ActionBarSlots = platform.NormalizeActionBarSlots(actionBarSlots, character.LearnedAbilityIDs)
	character.Quests = cloneQuestProgressMap(quests)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterTrackedQuests(characterID string, trackedQuestIDs []string) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.TrackedQuestIDs = cloneStringIDs(trackedQuestIDs)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterEconomy(
	characterID string,
	currencyCopper int,
	inventory []platform.CharacterInventorySlot,
	equipment []platform.CharacterEquipmentSlot,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	if currencyCopper < 0 {
		currencyCopper = 0
	}
	character.CurrencyCopper = currencyCopper
	character.Inventory = cloneInventorySlots(inventory)
	character.Equipment = cloneEquipmentSlots(equipment)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterProfessions(
	characterID string,
	currencyCopper int,
	professions []platform.CharacterProfessionState,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	if currencyCopper < 0 {
		currencyCopper = 0
	}
	character.CurrencyCopper = currencyCopper
	character.Professions = cloneProfessionStates(professions)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterTalents(
	characterID string,
	talents map[string]int,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.Talents = cloneTalentRanks(talents)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterActionBarSlots(
	characterID string,
	actionBarSlots []platform.CharacterActionBarSlot,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.ActionBarSlots = platform.NormalizeActionBarSlots(actionBarSlots, character.LearnedAbilityIDs)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) UpdateCharacterInventory(
	characterID string,
	inventory []platform.CharacterInventorySlot,
) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	character.Inventory = cloneInventorySlots(inventory)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	s.state.Characters[characterID] = character

	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, nil
}

func (s *FileStore) load() error {
	bytes, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded state
	if err := json.Unmarshal(bytes, &loaded); err != nil {
		return err
	}

	if loaded.Accounts != nil {
		s.state.Accounts = loaded.Accounts
	}
	if loaded.Realms != nil {
		s.state.Realms = loaded.Realms
	}
	if loaded.Characters != nil {
		for characterID, character := range loaded.Characters {
			character = platform.NormalizeCharacter(character)
			loaded.Characters[characterID] = character
		}
		s.state.Characters = loaded.Characters
	}
	if loaded.Sessions != nil {
		s.state.Sessions = loaded.Sessions
	}
	if loaded.WorldJoinTickets != nil {
		s.state.WorldJoinTickets = loaded.WorldJoinTickets
	}
	if loaded.PasswordReset != nil {
		s.state.PasswordReset = loaded.PasswordReset
	}
	if loaded.Friends != nil {
		s.state.Friends = loaded.Friends
	}
	if loaded.Parties != nil {
		s.state.Parties = loaded.Parties
	}
	if loaded.BuildManifest.ID != "" {
		s.state.BuildManifest = loaded.BuildManifest
	}

	return nil
}

func (s *FileStore) reloadLocked() error {
	bytes, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded state
	if err := json.Unmarshal(bytes, &loaded); err != nil {
		return err
	}

	if loaded.Accounts == nil {
		loaded.Accounts = map[string]platform.Account{}
	}
	if loaded.Realms == nil {
		loaded.Realms = map[string]platform.Realm{}
	}
	if loaded.Characters == nil {
		loaded.Characters = map[string]platform.Character{}
	} else {
		for characterID, character := range loaded.Characters {
			character = platform.NormalizeCharacter(character)
			loaded.Characters[characterID] = character
		}
	}
	if loaded.Sessions == nil {
		loaded.Sessions = map[string]platform.Session{}
	}
	if loaded.WorldJoinTickets == nil {
		loaded.WorldJoinTickets = map[string]platform.WorldJoinTicket{}
	}
	if loaded.PasswordReset == nil {
		loaded.PasswordReset = map[string]platform.PasswordResetTicket{}
	}
	if loaded.Friends == nil {
		loaded.Friends = map[string]platform.FriendRelationship{}
	}
	if loaded.Parties == nil {
		loaded.Parties = map[string]platform.Party{}
	}
	if loaded.BuildManifest.ID == "" {
		loaded.BuildManifest = s.state.BuildManifest
	}

	s.state = loaded
	return nil
}

func (s *FileStore) save() error {
	if err := s.lockState(false); err != nil {
		return err
	}
	defer s.unlockState()
	return s.saveLocked()
}

func (s *FileStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := tempFile.Write(payload); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Rename(tempPath, s.path)
}

func defaultBuildManifest(buildID string, worldEndpoint string) platform.BuildManifest {
	clientVersion := envOrDefault("AMANDACORE_CLIENT_VERSION", buildID)
	serverVersion := envOrDefault("AMANDACORE_SERVER_VERSION", buildID)
	contentVersion := envOrDefault("AMANDACORE_CONTENT_VERSION", buildID)
	protocolVersion := envOrDefault("AMANDACORE_PROTOCOL_VERSION", "local-dev-1")
	apiVersion := envOrDefault("AMANDACORE_API_VERSION", "local-api-1")

	return platform.BuildManifest{
		ID:                         buildID,
		Channel:                    envOrDefault("AMANDACORE_BUILD_CHANNEL", "development"),
		DisplayVersion:             envOrDefault("AMANDACORE_DISPLAY_VERSION", buildID),
		ClientVersion:              clientVersion,
		ServerVersion:              serverVersion,
		ContentVersion:             contentVersion,
		ProtocolVersion:            protocolVersion,
		APIVersion:                 apiVersion,
		CompatibleClientVersions:   []string{clientVersion},
		CompatibleServerVersions:   []string{serverVersion},
		CompatibleProtocolVersions: []string{protocolVersion},
		RequiredServices: []string{
			"auth-service",
			"account-service",
			"realm-service",
			"character-service",
			"world-service",
			"admin-service",
		},
		LauncherNews:      "amandacore development environment",
		AllowedForLogin:   true,
		WorldEndpointHint: worldEndpoint,
		GeneratedAtUTC:    envOrDefault("AMANDACORE_BUILD_GENERATED_AT_UTC", time.Now().UTC().Format(time.RFC3339)),
	}
}

func (s *FileStore) applyRuntimeBuildManifest(buildManifest platform.BuildManifest, worldEndpoint string) {
	s.state.BuildManifest = buildManifest

	realm, ok := s.state.Realms["sunset-frontier-dev"]
	if !ok {
		realm = platform.Realm{
			ID:          "sunset-frontier-dev",
			DisplayName: "Sunset Frontier Dev",
			Region:      "local",
		}
	}

	realm.Endpoint = worldEndpoint
	realm.SupportedBuild = buildManifest.ID
	realm.Online = true
	s.state.Realms["sunset-frontier-dev"] = realm
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil
}

func verifyPassword(encoded string, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func randomID(prefix string) string {
	return prefix + "_" + randomToken()
}

func randomToken() string {
	buffer := make([]byte, 24)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func friendKey(ownerCharacterID string, friendCharacterID string) string {
	return ownerCharacterID + ":" + friendCharacterID
}

func normalizePartyMembers(leaderCharacterID string, memberCharacterIDs []string) []string {
	seen := map[string]struct{}{}
	members := make([]string, 0, len(memberCharacterIDs)+1)

	if leaderCharacterID != "" {
		seen[leaderCharacterID] = struct{}{}
		members = append(members, leaderCharacterID)
	}

	for _, characterID := range memberCharacterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		if _, exists := seen[characterID]; exists {
			continue
		}
		seen[characterID] = struct{}{}
		members = append(members, characterID)
	}

	return members
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsRole(roles []platform.Role, target platform.Role) bool {
	for _, role := range roles {
		if role == target {
			return true
		}
	}
	return false
}

func cloneQuestProgressMap(source map[string]platform.CharacterQuestProgress) map[string]platform.CharacterQuestProgress {
	if len(source) == 0 {
		return map[string]platform.CharacterQuestProgress{}
	}

	cloned := make(map[string]platform.CharacterQuestProgress, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneInventorySlots(source []platform.CharacterInventorySlot) []platform.CharacterInventorySlot {
	return platform.NormalizeInventorySlots(source)
}

func cloneEquipmentSlots(source []platform.CharacterEquipmentSlot) []platform.CharacterEquipmentSlot {
	return platform.NormalizeEquipmentSlots(source)
}

func cloneProfessionStates(source []platform.CharacterProfessionState) []platform.CharacterProfessionState {
	return platform.NormalizeProfessionStates(source)
}

func cloneTalentRanks(source map[string]int) map[string]int {
	return platform.NormalizeTalentRanks(source)
}

func cloneActionBarSlots(source []platform.CharacterActionBarSlot, learnedAbilityIDs []string) []platform.CharacterActionBarSlot {
	return platform.NormalizeActionBarSlots(source, learnedAbilityIDs)
}

func cloneStringIDs(source []string) []string {
	return platform.NormalizeStringIDs(source)
}

func normalizedCharacterCopy(source platform.Character) platform.Character {
	normalized := platform.NormalizeCharacter(source)
	normalized.Inventory = cloneInventorySlots(normalized.Inventory)
	normalized.Equipment = cloneEquipmentSlots(normalized.Equipment)
	normalized.Professions = cloneProfessionStates(normalized.Professions)
	normalized.Talents = cloneTalentRanks(normalized.Talents)
	normalized.ActionBarSlots = cloneActionBarSlots(normalized.ActionBarSlots, normalized.LearnedAbilityIDs)
	normalized.Quests = cloneQuestProgressMap(normalized.Quests)
	normalized.TrackedQuestIDs = cloneStringIDs(normalized.TrackedQuestIDs)
	return normalized
}

func (s *FileStore) lockState(reload bool) error {
	s.mutex.Lock()

	if err := s.fileLock.Lock(); err != nil {
		s.mutex.Unlock()
		return err
	}

	if reload {
		if err := s.reloadLocked(); err != nil {
			_ = s.fileLock.Unlock()
			s.mutex.Unlock()
			return err
		}
	}

	return nil
}

func (s *FileStore) unlockState() {
	_ = s.fileLock.Unlock()
	s.mutex.Unlock()
}

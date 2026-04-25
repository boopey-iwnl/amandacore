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
	ErrGuildMissing        = errors.New("guild does not exist")
	ErrGuildNameExists     = errors.New("guild name already exists in this realm")
	ErrGuildMemberExists   = errors.New("character is already in a guild")
	ErrGuildInviteMissing  = errors.New("guild invite does not exist")
)

type state struct {
	Accounts            map[string]platform.Account               `json:"accounts"`
	Realms              map[string]platform.Realm                 `json:"realms"`
	Characters          map[string]platform.Character             `json:"characters"`
	Sessions            map[string]platform.Session               `json:"sessions"`
	WorldJoinTickets    map[string]platform.WorldJoinTicket       `json:"worldJoinTickets"`
	PasswordReset       map[string]platform.PasswordResetTicket   `json:"passwordReset"`
	Friends             map[string]platform.FriendRelationship    `json:"friends"`
	Parties             map[string]platform.Party                 `json:"parties"`
	Guilds              map[string]platform.Guild                 `json:"guilds"`
	GuildInvites        map[string]platform.GuildInvite           `json:"guildInvites"`
	Auctions            map[string]platform.AuctionListing        `json:"auctions"`
	Mail                map[string]platform.MailEnvelope          `json:"mail"`
	AuditEvents         map[string]platform.AuditEvent            `json:"auditEvents"`
	SupportTickets      map[string]platform.SupportTicket         `json:"supportTickets"`
	Mutes               map[string]platform.MuteRecord            `json:"mutes"`
	HousingEntitlements map[string]platform.HousingEntitlement    `json:"housingEntitlements"`
	HousingSpaces       map[string]platform.HousingSpace          `json:"housingSpaces"`
	HousingStorage      map[string][]platform.HousingStorageSlot  `json:"housingStorage"`
	HousingDecorations  map[string][]platform.DecorationPlacement `json:"housingDecorations"`
	AccountProgress     map[string]platform.AccountProgressState  `json:"accountProgress"`
	BuildManifest       platform.BuildManifest                    `json:"buildManifest"`
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
			Accounts:            map[string]platform.Account{},
			Realms:              map[string]platform.Realm{},
			Characters:          map[string]platform.Character{},
			Sessions:            map[string]platform.Session{},
			WorldJoinTickets:    map[string]platform.WorldJoinTicket{},
			PasswordReset:       map[string]platform.PasswordResetTicket{},
			Friends:             map[string]platform.FriendRelationship{},
			Parties:             map[string]platform.Party{},
			Guilds:              map[string]platform.Guild{},
			GuildInvites:        map[string]platform.GuildInvite{},
			Auctions:            map[string]platform.AuctionListing{},
			Mail:                map[string]platform.MailEnvelope{},
			AuditEvents:         map[string]platform.AuditEvent{},
			SupportTickets:      map[string]platform.SupportTicket{},
			Mutes:               map[string]platform.MuteRecord{},
			HousingEntitlements: map[string]platform.HousingEntitlement{},
			HousingSpaces:       map[string]platform.HousingSpace{},
			HousingStorage:      map[string][]platform.HousingStorageSlot{},
			HousingDecorations:  map[string][]platform.DecorationPlacement{},
			AccountProgress:     map[string]platform.AccountProgressState{},
			BuildManifest:       buildManifest,
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

		now := time.Now().Unix()
		if account.Banned || account.SuspendedUntil > now {
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
	if account, ok := s.state.Accounts[accountID]; ok {
		account.LastLoginAt = session.CreatedAt
		account.LastSessionID = session.ID
		account.UpdatedAt = session.CreatedAt
		s.state.Accounts[accountID] = account
	}
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

		if account, ok := s.state.Accounts[session.AccountID]; ok {
			if account.Banned || account.SuspendedUntil > now {
				return nil, ErrAccountBanned
			}
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

func (s *FileStore) GetAccountProgress(accountID string) (platform.AccountProgressState, error) {
	if err := s.lockState(true); err != nil {
		return platform.AccountProgressState{}, err
	}
	defer s.unlockState()

	if _, ok := s.state.Accounts[accountID]; !ok {
		return platform.AccountProgressState{}, ErrInvalidCredentials
	}
	progress := platform.NormalizeAccountProgress(accountID, s.state.AccountProgress[accountID])
	return cloneAccountProgress(progress), nil
}

func (s *FileStore) SaveAccountProgress(progress platform.AccountProgressState) (platform.AccountProgressState, error) {
	if err := s.lockState(true); err != nil {
		return platform.AccountProgressState{}, err
	}
	defer s.unlockState()

	if _, ok := s.state.Accounts[progress.AccountID]; !ok {
		return platform.AccountProgressState{}, ErrInvalidCredentials
	}
	progress = platform.NormalizeAccountProgress(progress.AccountID, progress)
	progress.UpdatedAt = time.Now().Unix()
	if s.state.AccountProgress == nil {
		s.state.AccountProgress = map[string]platform.AccountProgressState{}
	}
	s.state.AccountProgress[progress.AccountID] = progress
	return cloneAccountProgress(progress), s.saveLocked()
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

func (s *FileStore) GetOrCreateHousingForCharacter(characterID string, templateID string) (platform.HousingEntitlement, platform.HousingSpace, error) {
	if err := s.lockState(true); err != nil {
		return platform.HousingEntitlement{}, platform.HousingSpace{}, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return platform.HousingEntitlement{}, platform.HousingSpace{}, fmt.Errorf("character not found")
	}
	if strings.TrimSpace(templateID) == "" {
		return platform.HousingEntitlement{}, platform.HousingSpace{}, fmt.Errorf("housing template is required")
	}

	if entitlement, ok := s.state.HousingEntitlements[characterID]; ok {
		if entitlement.TemplateID == "" {
			entitlement.TemplateID = templateID
		}
		entitlement.Unlocked = true
		space, found := s.state.HousingSpaces[entitlement.HousingSpaceID]
		if !found {
			now := time.Now().Unix()
			space = platform.HousingSpace{
				HousingSpaceID:   entitlement.HousingSpaceID,
				OwnerCharacterID: character.ID,
				OwnerAccountID:   character.AccountID,
				TemplateID:       entitlement.TemplateID,
				CreatedAt:        now,
			}
			s.state.HousingSpaces[space.HousingSpaceID] = space
		}
		if space.TemplateID == "" {
			space.TemplateID = entitlement.TemplateID
		}
		if space.OwnerCharacterID == "" {
			space.OwnerCharacterID = character.ID
		}
		if space.OwnerAccountID == "" {
			space.OwnerAccountID = character.AccountID
		}
		s.state.HousingEntitlements[characterID] = entitlement
		s.state.HousingSpaces[space.HousingSpaceID] = space
		if _, ok := s.state.HousingStorage[space.HousingSpaceID]; !ok {
			s.state.HousingStorage[space.HousingSpaceID] = platform.NormalizeHousingStorageSlots(nil)
		}
		if _, ok := s.state.HousingDecorations[space.HousingSpaceID]; !ok {
			s.state.HousingDecorations[space.HousingSpaceID] = []platform.DecorationPlacement{}
		}
		return entitlement, space, s.saveLocked()
	}

	now := time.Now().Unix()
	spaceID := randomID("house")
	entitlement := platform.HousingEntitlement{
		CharacterID:    character.ID,
		HousingSpaceID: spaceID,
		TemplateID:     templateID,
		Unlocked:       true,
		CreatedAt:      now,
	}
	space := platform.HousingSpace{
		HousingSpaceID:   spaceID,
		OwnerCharacterID: character.ID,
		OwnerAccountID:   character.AccountID,
		TemplateID:       templateID,
		CreatedAt:        now,
	}
	s.state.HousingEntitlements[character.ID] = entitlement
	s.state.HousingSpaces[spaceID] = space
	s.state.HousingStorage[spaceID] = platform.NormalizeHousingStorageSlots(nil)
	s.state.HousingDecorations[spaceID] = []platform.DecorationPlacement{}
	return entitlement, space, s.saveLocked()
}

func (s *FileStore) UpdateHousingVisit(housingSpaceID string, returnZoneID string, returnX float64, returnY float64, returnZ float64) (platform.HousingSpace, error) {
	if err := s.lockState(true); err != nil {
		return platform.HousingSpace{}, err
	}
	defer s.unlockState()

	space, ok := s.state.HousingSpaces[housingSpaceID]
	if !ok {
		return platform.HousingSpace{}, fmt.Errorf("housing space not found")
	}
	if returnZoneID != "" {
		space.ReturnZoneID = returnZoneID
		space.ReturnX = returnX
		space.ReturnY = returnY
		space.ReturnZ = returnZ
	}
	space.LastVisitedAt = time.Now().Unix()
	s.state.HousingSpaces[housingSpaceID] = space
	return space, s.saveLocked()
}

func (s *FileStore) ListHousingStorage(housingSpaceID string) ([]platform.HousingStorageSlot, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	if _, ok := s.state.HousingSpaces[housingSpaceID]; !ok {
		return nil, fmt.Errorf("housing space not found")
	}
	return cloneHousingStorageSlots(s.state.HousingStorage[housingSpaceID]), nil
}

func (s *FileStore) UpdateCharacterInventoryAndHousingStorage(
	characterID string,
	housingSpaceID string,
	inventory []platform.CharacterInventorySlot,
	storage []platform.HousingStorageSlot,
) (*platform.Character, []platform.HousingStorageSlot, error) {
	if err := s.lockState(true); err != nil {
		return nil, nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, nil, fmt.Errorf("character not found")
	}
	if err := s.validateHousingOwnerLocked(characterID, housingSpaceID); err != nil {
		return nil, nil, err
	}

	character.Inventory = cloneInventorySlots(inventory)
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	normalizedStorage := cloneHousingStorageSlots(storage)

	s.state.Characters[characterID] = character
	s.state.HousingStorage[housingSpaceID] = normalizedStorage
	if err := s.saveLocked(); err != nil {
		return nil, nil, err
	}

	copy := normalizedCharacterCopy(character)
	return &copy, cloneHousingStorageSlots(normalizedStorage), nil
}

func (s *FileStore) UpdateHousingStorage(characterID string, housingSpaceID string, storage []platform.HousingStorageSlot) ([]platform.HousingStorageSlot, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	if err := s.validateHousingOwnerLocked(characterID, housingSpaceID); err != nil {
		return nil, err
	}
	normalized := cloneHousingStorageSlots(storage)
	s.state.HousingStorage[housingSpaceID] = normalized
	return cloneHousingStorageSlots(normalized), s.saveLocked()
}

func (s *FileStore) ListHousingDecorations(housingSpaceID string) ([]platform.DecorationPlacement, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	if _, ok := s.state.HousingSpaces[housingSpaceID]; !ok {
		return nil, fmt.Errorf("housing space not found")
	}
	return cloneDecorationPlacements(s.state.HousingDecorations[housingSpaceID]), nil
}

func (s *FileStore) SaveHousingDecorations(characterID string, housingSpaceID string, placements []platform.DecorationPlacement) ([]platform.DecorationPlacement, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	if err := s.validateHousingOwnerLocked(characterID, housingSpaceID); err != nil {
		return nil, err
	}
	normalized := cloneDecorationPlacements(placements)
	s.state.HousingDecorations[housingSpaceID] = normalized
	return cloneDecorationPlacements(normalized), s.saveLocked()
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

func (s *FileStore) CreateGuild(guildName string, leaderCharacterID string) (platform.Guild, error) {
	if err := s.lockState(true); err != nil {
		return platform.Guild{}, err
	}
	defer s.unlockState()

	leader, ok := s.state.Characters[leaderCharacterID]
	if !ok {
		return platform.Guild{}, fmt.Errorf("leader character not found")
	}
	if _, found := findGuildForCharacterLocked(s.state.Guilds, leaderCharacterID); found {
		return platform.Guild{}, ErrGuildMemberExists
	}

	normalizedName := normalize(guildName)
	for _, guild := range s.state.Guilds {
		if guild.RealmID == leader.RealmID && normalize(guild.GuildName) == normalizedName {
			return platform.Guild{}, ErrGuildNameExists
		}
	}

	now := time.Now().Unix()
	guild := platform.Guild{
		ID:                   randomID("guild"),
		RealmID:              leader.RealmID,
		GuildName:            strings.TrimSpace(guildName),
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedByCharacterID: leader.ID,
		LeaderCharacterID:    leader.ID,
		Ranks:                platform.DefaultGuildRanks(),
		Members: []platform.GuildMember{
			buildGuildMember(leader, platform.GuildRankLeader, now),
		},
	}
	s.state.Guilds[guild.ID] = guild
	return cloneGuild(guild), s.saveLocked()
}

func (s *FileStore) SaveGuild(guild platform.Guild) (platform.Guild, error) {
	if err := s.lockState(true); err != nil {
		return platform.Guild{}, err
	}
	defer s.unlockState()

	if guild.ID == "" {
		return platform.Guild{}, ErrGuildMissing
	}
	if _, exists := s.state.Guilds[guild.ID]; !exists {
		return platform.Guild{}, ErrGuildMissing
	}
	guild.GuildName = strings.TrimSpace(guild.GuildName)
	guild.Ranks = normalizeGuildRanks(guild.Ranks)
	guild.Members = normalizeGuildMembers(guild.Members)
	if len(guild.Members) == 0 {
		delete(s.state.Guilds, guild.ID)
		return guild, s.saveLocked()
	}
	if !guildContainsMember(guild, guild.LeaderCharacterID) {
		guild.LeaderCharacterID = guild.Members[0].CharacterID
		guild.Members[0].RankID = platform.GuildRankLeader
	}
	for index, member := range guild.Members {
		if character, ok := s.state.Characters[member.CharacterID]; ok {
			guild.Members[index].DisplayName = character.DisplayName
			guild.Members[index].RaceID = character.RaceID
			guild.Members[index].ClassID = character.ClassID
			guild.Members[index].Level = character.Level
			guild.Members[index].LastOnlineAt = character.LastSeenAt
		}
	}
	guild.UpdatedAt = time.Now().Unix()
	s.state.Guilds[guild.ID] = guild
	return cloneGuild(guild), s.saveLocked()
}

func (s *FileStore) DeleteGuild(guildID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	if _, exists := s.state.Guilds[guildID]; !exists {
		return ErrGuildMissing
	}
	delete(s.state.Guilds, guildID)
	for inviteID, invite := range s.state.GuildInvites {
		if invite.GuildID == guildID {
			delete(s.state.GuildInvites, inviteID)
		}
	}
	return s.saveLocked()
}

func (s *FileStore) GetGuildByID(guildID string) (*platform.Guild, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	guild, ok := s.state.Guilds[guildID]
	if !ok {
		return nil, ErrGuildMissing
	}
	copy := cloneGuild(guild)
	return &copy, nil
}

func (s *FileStore) GetGuildForCharacter(characterID string) (*platform.Guild, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	guild, found := findGuildForCharacterLocked(s.state.Guilds, characterID)
	if !found {
		return nil, ErrGuildMissing
	}
	copy := cloneGuild(guild)
	return &copy, nil
}

func (s *FileStore) CreateGuildInvite(guildID string, inviterCharacterID string, targetCharacterID string, expiresAt int64) (platform.GuildInvite, error) {
	if err := s.lockState(true); err != nil {
		return platform.GuildInvite{}, err
	}
	defer s.unlockState()

	guild, ok := s.state.Guilds[guildID]
	if !ok {
		return platform.GuildInvite{}, ErrGuildMissing
	}
	if _, ok := s.state.Characters[inviterCharacterID]; !ok {
		return platform.GuildInvite{}, fmt.Errorf("inviter character not found")
	}
	target, ok := s.state.Characters[targetCharacterID]
	if !ok {
		return platform.GuildInvite{}, fmt.Errorf("target character not found")
	}
	if target.RealmID != guild.RealmID {
		return platform.GuildInvite{}, fmt.Errorf("target player is not on this realm")
	}
	if _, found := findGuildForCharacterLocked(s.state.Guilds, targetCharacterID); found {
		return platform.GuildInvite{}, ErrGuildMemberExists
	}
	for _, invite := range s.state.GuildInvites {
		if invite.GuildID == guildID && invite.TargetCharacterID == targetCharacterID {
			return platform.GuildInvite{}, fmt.Errorf("an invite is already pending for that player")
		}
	}

	now := time.Now().Unix()
	invite := platform.GuildInvite{
		InviteID:           randomID("ginvite"),
		GuildID:            guild.ID,
		GuildName:          guild.GuildName,
		InviterCharacterID: inviterCharacterID,
		TargetCharacterID:  targetCharacterID,
		CreatedAt:          now,
		ExpiresAt:          expiresAt,
	}
	s.state.GuildInvites[invite.InviteID] = invite
	return invite, s.saveLocked()
}

func (s *FileStore) GetGuildInvite(inviteID string) (*platform.GuildInvite, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	invite, ok := s.state.GuildInvites[inviteID]
	if !ok {
		return nil, ErrGuildInviteMissing
	}
	copy := invite
	return &copy, nil
}

func (s *FileStore) ListGuildInvitesForCharacter(characterID string) ([]platform.GuildInvite, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	results := make([]platform.GuildInvite, 0)
	for _, invite := range s.state.GuildInvites {
		if invite.TargetCharacterID == characterID {
			results = append(results, invite)
		}
	}
	return results, nil
}

func (s *FileStore) DeleteGuildInvite(inviteID string) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	if _, exists := s.state.GuildInvites[inviteID]; !exists {
		return ErrGuildInviteMissing
	}
	delete(s.state.GuildInvites, inviteID)
	return s.saveLocked()
}

func (s *FileStore) CleanupExpiredGuildInvites(nowUnix int64) error {
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	changed := false
	for inviteID, invite := range s.state.GuildInvites {
		if invite.ExpiresAt <= nowUnix {
			delete(s.state.GuildInvites, inviteID)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
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

func (s *FileStore) UpdateCharacterTravelState(
	characterID string,
	bindPoint platform.CharacterBindPoint,
	travelState platform.CharacterTravelState,
	mountState platform.CharacterMountState,
	currencyCopper int,
	zoneID string,
	x float64,
	y float64,
	z float64,
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
	character.BindPoint = platform.NormalizeCharacterBindPoint(characterID, bindPoint)
	character.TravelState = platform.NormalizeCharacterTravelState(travelState)
	character.MountState = platform.NormalizeCharacterMountState(mountState)
	character.CurrencyCopper = currencyCopper
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

func (s *FileStore) UpdateCharacterPvPStats(characterID string, stats platform.CharacterPvPStats) (*platform.Character, error) {
	if err := s.lockState(true); err != nil {
		return nil, err
	}
	defer s.unlockState()

	character, ok := s.state.Characters[characterID]
	if !ok {
		return nil, fmt.Errorf("character not found")
	}

	stats = platform.NormalizeCharacterPvPStats(characterID, stats)
	character.PvPStats = stats
	character = platform.NormalizeCharacter(character)
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
	if loaded.Guilds != nil {
		s.state.Guilds = loaded.Guilds
	}
	if loaded.GuildInvites != nil {
		s.state.GuildInvites = loaded.GuildInvites
	}
	if loaded.Auctions != nil {
		s.state.Auctions = loaded.Auctions
	}
	if loaded.Mail != nil {
		s.state.Mail = loaded.Mail
	}
	if loaded.AuditEvents != nil {
		s.state.AuditEvents = loaded.AuditEvents
	}
	if loaded.SupportTickets != nil {
		s.state.SupportTickets = loaded.SupportTickets
	}
	if loaded.Mutes != nil {
		s.state.Mutes = loaded.Mutes
	}
	if loaded.HousingEntitlements != nil {
		s.state.HousingEntitlements = loaded.HousingEntitlements
	}
	if loaded.HousingSpaces != nil {
		s.state.HousingSpaces = loaded.HousingSpaces
	}
	if loaded.HousingStorage != nil {
		for housingSpaceID, slots := range loaded.HousingStorage {
			loaded.HousingStorage[housingSpaceID] = platform.NormalizeHousingStorageSlots(slots)
		}
		s.state.HousingStorage = loaded.HousingStorage
	}
	if loaded.HousingDecorations != nil {
		for housingSpaceID, placements := range loaded.HousingDecorations {
			loaded.HousingDecorations[housingSpaceID] = platform.NormalizeDecorationPlacements(placements)
		}
		s.state.HousingDecorations = loaded.HousingDecorations
	}
	if loaded.AccountProgress != nil {
		for accountID, progress := range loaded.AccountProgress {
			loaded.AccountProgress[accountID] = platform.NormalizeAccountProgress(accountID, progress)
		}
		s.state.AccountProgress = loaded.AccountProgress
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
	if loaded.Guilds == nil {
		loaded.Guilds = map[string]platform.Guild{}
	}
	if loaded.GuildInvites == nil {
		loaded.GuildInvites = map[string]platform.GuildInvite{}
	}
	if loaded.Auctions == nil {
		loaded.Auctions = map[string]platform.AuctionListing{}
	}
	if loaded.Mail == nil {
		loaded.Mail = map[string]platform.MailEnvelope{}
	}
	if loaded.AuditEvents == nil {
		loaded.AuditEvents = map[string]platform.AuditEvent{}
	}
	if loaded.SupportTickets == nil {
		loaded.SupportTickets = map[string]platform.SupportTicket{}
	}
	if loaded.Mutes == nil {
		loaded.Mutes = map[string]platform.MuteRecord{}
	}
	if loaded.HousingEntitlements == nil {
		loaded.HousingEntitlements = map[string]platform.HousingEntitlement{}
	}
	if loaded.HousingSpaces == nil {
		loaded.HousingSpaces = map[string]platform.HousingSpace{}
	}
	if loaded.HousingStorage == nil {
		loaded.HousingStorage = map[string][]platform.HousingStorageSlot{}
	} else {
		for housingSpaceID, slots := range loaded.HousingStorage {
			loaded.HousingStorage[housingSpaceID] = platform.NormalizeHousingStorageSlots(slots)
		}
	}
	if loaded.HousingDecorations == nil {
		loaded.HousingDecorations = map[string][]platform.DecorationPlacement{}
	} else {
		for housingSpaceID, placements := range loaded.HousingDecorations {
			loaded.HousingDecorations[housingSpaceID] = platform.NormalizeDecorationPlacements(placements)
		}
	}
	if loaded.AccountProgress == nil {
		loaded.AccountProgress = map[string]platform.AccountProgressState{}
	} else {
		for accountID, progress := range loaded.AccountProgress {
			loaded.AccountProgress[accountID] = platform.NormalizeAccountProgress(accountID, progress)
		}
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

func cloneHousingStorageSlots(source []platform.HousingStorageSlot) []platform.HousingStorageSlot {
	return platform.NormalizeHousingStorageSlots(source)
}

func cloneDecorationPlacements(source []platform.DecorationPlacement) []platform.DecorationPlacement {
	return platform.NormalizeDecorationPlacements(source)
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

func cloneAccountProgress(source platform.AccountProgressState) platform.AccountProgressState {
	source = platform.NormalizeAccountProgress(source.AccountID, source)
	progress := platform.AccountProgressState{
		AccountID:                  source.AccountID,
		AchievementProgress:        map[string]platform.AchievementProgress{},
		UnlockedTitleIDs:           cloneStringIDs(source.UnlockedTitleIDs),
		SelectedTitleByCharacterID: map[string]string{},
		CollectionUnlocks:          map[string]platform.CollectionUnlock{},
		UpdatedAt:                  source.UpdatedAt,
	}
	for achievementID, achievementProgress := range source.AchievementProgress {
		progress.AchievementProgress[achievementID] = achievementProgress
	}
	for characterID, titleID := range source.SelectedTitleByCharacterID {
		progress.SelectedTitleByCharacterID[characterID] = titleID
	}
	for unlockID, unlock := range source.CollectionUnlocks {
		progress.CollectionUnlocks[unlockID] = unlock
	}
	return progress
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
	normalized.BindPoint = platform.NormalizeCharacterBindPoint(normalized.ID, normalized.BindPoint)
	normalized.TravelState = platform.NormalizeCharacterTravelState(normalized.TravelState)
	normalized.TravelState.DiscoveredTravelPointIDs = cloneStringIDs(normalized.TravelState.DiscoveredTravelPointIDs)
	normalized.MountState = platform.NormalizeCharacterMountState(normalized.MountState)
	normalized.MountState.UnlockedMountIDs = cloneStringIDs(normalized.MountState.UnlockedMountIDs)
	return normalized
}

func (s *FileStore) validateHousingOwnerLocked(characterID string, housingSpaceID string) error {
	if characterID == "" || housingSpaceID == "" {
		return fmt.Errorf("housing ownership is required")
	}
	if _, ok := s.state.Characters[characterID]; !ok {
		return fmt.Errorf("character not found")
	}
	space, ok := s.state.HousingSpaces[housingSpaceID]
	if !ok {
		return fmt.Errorf("housing space not found")
	}
	if space.OwnerCharacterID != characterID {
		return fmt.Errorf("character does not own this housing space")
	}
	return nil
}

func buildGuildMember(character platform.Character, rankID string, joinedAt int64) platform.GuildMember {
	character = platform.NormalizeCharacter(character)
	return platform.GuildMember{
		CharacterID:  character.ID,
		DisplayName:  character.DisplayName,
		RaceID:       character.RaceID,
		ClassID:      character.ClassID,
		Level:        character.Level,
		RankID:       rankID,
		JoinedAt:     joinedAt,
		LastOnlineAt: character.LastSeenAt,
	}
}

func cloneGuild(source platform.Guild) platform.Guild {
	source.Ranks = append([]platform.GuildRank(nil), source.Ranks...)
	for index := range source.Ranks {
		source.Ranks[index].Permissions = append([]string(nil), source.Ranks[index].Permissions...)
	}
	source.Members = append([]platform.GuildMember(nil), source.Members...)
	return source
}

func normalizeGuildRanks(source []platform.GuildRank) []platform.GuildRank {
	if len(source) == 0 {
		return platform.DefaultGuildRanks()
	}

	seen := map[string]struct{}{}
	normalized := make([]platform.GuildRank, 0, len(source))
	for _, rank := range source {
		rank.RankID = strings.TrimSpace(rank.RankID)
		if rank.RankID == "" {
			continue
		}
		if _, exists := seen[rank.RankID]; exists {
			continue
		}
		seen[rank.RankID] = struct{}{}
		rank.DisplayName = strings.TrimSpace(rank.DisplayName)
		if rank.DisplayName == "" {
			rank.DisplayName = rank.RankID
		}
		rank.Permissions = cloneStringIDs(rank.Permissions)
		normalized = append(normalized, rank)
	}
	if len(normalized) == 0 {
		return platform.DefaultGuildRanks()
	}
	return normalized
}

func normalizeGuildMembers(source []platform.GuildMember) []platform.GuildMember {
	seen := map[string]struct{}{}
	normalized := make([]platform.GuildMember, 0, len(source))
	for _, member := range source {
		member.CharacterID = strings.TrimSpace(member.CharacterID)
		if member.CharacterID == "" {
			continue
		}
		if _, exists := seen[member.CharacterID]; exists {
			continue
		}
		seen[member.CharacterID] = struct{}{}
		if member.RankID == "" {
			member.RankID = platform.GuildRankRecruit
		}
		normalized = append(normalized, member)
	}
	return normalized
}

func findGuildForCharacterLocked(guilds map[string]platform.Guild, characterID string) (platform.Guild, bool) {
	for _, guild := range guilds {
		if guildContainsMember(guild, characterID) {
			return guild, true
		}
	}
	return platform.Guild{}, false
}

func guildContainsMember(guild platform.Guild, characterID string) bool {
	for _, member := range guild.Members {
		if member.CharacterID == characterID {
			return true
		}
	}
	return false
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

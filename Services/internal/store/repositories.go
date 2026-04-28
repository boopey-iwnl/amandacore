// Package store defines AmandaCore-owned persistence boundaries.
//
// The current service runtime continues to use FileStore as the supported
// local/dev adapter. Milestone 2 introduces narrower repository contracts so a
// SQL implementation can be added incrementally without forcing service
// callers through a broad storage rewrite.
package store

import "amandacore/services/internal/platform"

type AccountRepository interface {
	RegisterAccount(username string, password string) (platform.Account, error)
	Authenticate(username string, password string) (platform.Account, error)
	GetAccountByID(accountID string) (*platform.Account, error)
	ListAccounts() ([]platform.Account, error)
}

type SessionRepository interface {
	CreateSession(accountID string) (platform.Session, error)
	RefreshSession(refreshToken string) (platform.Session, error)
	ValidateAccessToken(token string) (*platform.Session, error)
	RevokeSession(token string) error
}

type RealmRepository interface {
	ListRealms() ([]platform.Realm, error)
	GetBuildManifest() platform.BuildManifest
}

type CharacterRepository interface {
	ListCharacters(accountID string, realmID string) ([]platform.Character, error)
	CreateCharacter(accountID string, realmID string, displayName string, raceID string, classID string, archetypeID string) (platform.Character, error)
	GetCharacterByID(characterID string) (*platform.Character, error)
	GetCharacterByName(realmID string, displayName string) (*platform.Character, error)
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
}

type CharacterTransactionRepository interface {
	UpdateCharacterAtomically(operation string, characterID string, mutate func(*platform.Character) error) (*platform.Character, error)
}

type ProgressionRepository interface {
	UpdateCharacterProgression(
		characterID string,
		experience int,
		currencyCopper int,
		inventory []platform.CharacterInventorySlot,
		learnedAbilityIDs []string,
		actionBarSlots []platform.CharacterActionBarSlot,
		quests map[string]platform.CharacterQuestProgress,
	) (*platform.Character, error)
	UpdateCharacterTrackedQuests(characterID string, trackedQuestIDs []string) (*platform.Character, error)
}

type InventoryRepository interface {
	GetCharacterInventory(characterID string) ([]platform.CharacterInventorySlot, error)
	UpdateCharacterInventory(characterID string, inventory []platform.CharacterInventorySlot) (*platform.Character, error)
	UpdateCharacterEconomy(characterID string, currencyCopper int, inventory []platform.CharacterInventorySlot, equipment []platform.CharacterEquipmentSlot) (*platform.Character, error)
}

type QuestRepository interface {
	GetCharacterQuestProgress(characterID string) (map[string]platform.CharacterQuestProgress, error)
	UpdateCharacterQuestProgress(characterID string, quests map[string]platform.CharacterQuestProgress) (*platform.Character, error)
	UpdateCharacterTrackedQuests(characterID string, trackedQuestIDs []string) (*platform.Character, error)
}

type AbilityRepository interface {
	GetLearnedAbilities(characterID string) ([]string, error)
	UpdateLearnedAbilities(characterID string, learnedAbilityIDs []string) (*platform.Character, error)
}

type ActionBarRepository interface {
	GetActionBarSlots(characterID string) ([]platform.CharacterActionBarSlot, error)
	UpdateActionBarSlots(characterID string, actionBarSlots []platform.CharacterActionBarSlot) (*platform.Character, error)
}

type SessionRecoveryRepository interface {
	LoadSessionRecoveryState(characterID string) (SessionRecoveryState, error)
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
}

type WorldTicketRepository interface {
	IssueWorldJoinTicket(accountID string, sessionID string, characterID string, realmID string) (platform.WorldJoinTicket, error)
	ValidateWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error)
	ConsumeWorldJoinTicket(ticketID string) (*platform.WorldJoinTicket, error)
	RevokeCharacterJoinTickets(characterID string) error
}

type WorldSessionRecord struct {
	SessionToken string
	AccountID    string
	CharacterID  string
	RealmID      string
	ZoneID       string
	Connected    bool
	PositionX    float64
	PositionY    float64
	PositionZ    float64
	CreatedAt    int64
	UpdatedAt    int64
}

type WorldSessionRepository interface {
	CreateWorldSession(session WorldSessionRecord) (WorldSessionRecord, error)
	GetWorldSession(sessionToken string) (*WorldSessionRecord, error)
	UpdateWorldSession(session WorldSessionRecord) (WorldSessionRecord, error)
}

type AuditEventRepository interface {
	RecordAuditEvent(event platform.AuditEvent) (platform.AuditEvent, error)
	QueryAuditEvents(query AuditQuery) ([]platform.AuditEvent, error)
}

type MigrationRepository interface {
	ApplyMigrations(options MigrationOptions) (MigrationResult, error)
	MigrationHistory() ([]MigrationRecord, error)
}

type UnitOfWork interface {
	WithTransaction(operation string, fn func(*FileStoreTx) error) error
}

var (
	_ AccountRepository              = (*FileStore)(nil)
	_ SessionRepository              = (*FileStore)(nil)
	_ RealmRepository                = (*FileStore)(nil)
	_ CharacterRepository            = (*FileStore)(nil)
	_ CharacterTransactionRepository = (*FileStore)(nil)
	_ ProgressionRepository          = (*FileStore)(nil)
	_ InventoryRepository            = (*FileStore)(nil)
	_ QuestRepository                = (*FileStore)(nil)
	_ AbilityRepository              = (*FileStore)(nil)
	_ ActionBarRepository            = (*FileStore)(nil)
	_ SessionRecoveryRepository      = (*FileStore)(nil)
	_ WorldTicketRepository          = (*FileStore)(nil)
	_ AuditEventRepository           = (*FileStore)(nil)
	_ MigrationRepository            = (*FileStore)(nil)
	_ UnitOfWork                     = (*FileStore)(nil)
)

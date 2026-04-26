package store

import "amandacore/services/internal/platform"

type CharacterRepository interface {
	CreateCharacter(accountID string, realmID string, displayName string, raceID string, classID string, archetypeID string) (platform.Character, error)
	GetCharacterByID(characterID string) (*platform.Character, error)
	GetCharacterByName(realmID string, displayName string) (*platform.Character, error)
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
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
	UpdateCharacterInventory(characterID string, inventory []platform.CharacterInventorySlot) (*platform.Character, error)
	UpdateCharacterEconomy(characterID string, currencyCopper int, inventory []platform.CharacterInventorySlot, equipment []platform.CharacterEquipmentSlot) (*platform.Character, error)
}

type SessionRecoveryRepository interface {
	LoadSessionRecoveryState(characterID string) (SessionRecoveryState, error)
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
}

type MigrationRepository interface {
	ApplyMigrations(options MigrationOptions) (MigrationResult, error)
	MigrationHistory() ([]MigrationRecord, error)
}

type UnitOfWork interface {
	WithTransaction(operation string, fn func(*FileStoreTx) error) error
}

var (
	_ CharacterRepository       = (*FileStore)(nil)
	_ ProgressionRepository     = (*FileStore)(nil)
	_ InventoryRepository       = (*FileStore)(nil)
	_ SessionRecoveryRepository = (*FileStore)(nil)
	_ MigrationRepository       = (*FileStore)(nil)
	_ UnitOfWork                = (*FileStore)(nil)
)

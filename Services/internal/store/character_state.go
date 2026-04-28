package store

import (
	"errors"

	"amandacore/services/internal/platform"
)

var (
	ErrCharacterStateConflict   = errors.New("character state conflict")
	ErrIdempotencyConflict      = errors.New("character state mutation idempotency conflict")
	ErrInvalidInventoryMove     = errors.New("invalid inventory move")
	ErrInventoryFull            = errors.New("inventory is full")
	ErrAbilityNotLearned        = errors.New("ability is not learned")
	ErrQuestAlreadyAccepted     = errors.New("quest is already accepted")
	ErrQuestRewardAlreadyGiven  = errors.New("quest reward is already granted")
	ErrQuestProgressUnavailable = errors.New("quest progress is unavailable")
)

type MutationOptions struct {
	MutationKey string
}

type InventoryItemGrant struct {
	ItemID      string
	DisplayName string
	Quantity    int
	MaxStack    int
	Stackable   bool
}

type CharacterPositionSnapshot struct {
	SnapshotID        string  `json:"snapshotId"`
	CharacterID       string  `json:"characterId"`
	WorldSessionToken string  `json:"worldSessionToken,omitempty"`
	ZoneID            string  `json:"zoneId"`
	X                 float64 `json:"x"`
	Y                 float64 `json:"y"`
	Z                 float64 `json:"z"`
	CapturedAt        int64   `json:"capturedAt"`
	Reason            string  `json:"reason"`
	CharacterVersion  int64   `json:"characterVersion,omitempty"`
}

type QuestRewardMutation struct {
	QuestID             string
	ExperienceDelta     int
	CurrencyCopperDelta int
	RewardItems         []InventoryItemGrant
}

type TransactionalCharacterStateRepository interface {
	GetCharacterPositionSnapshots(characterID string, limit int) ([]CharacterPositionSnapshot, error)
	MoveInventorySlot(characterID string, fromSlotIndex int, toSlotIndex int, options MutationOptions) (*platform.Character, error)
	GrantInventoryItem(characterID string, grant InventoryItemGrant, options MutationOptions) (*platform.Character, error)
	AcceptQuestProgress(characterID string, progress platform.CharacterQuestProgress, options MutationOptions) (*platform.Character, error)
	UpdateQuestProgress(characterID string, progress platform.CharacterQuestProgress, options MutationOptions) (*platform.Character, error)
	CompleteQuestWithReward(characterID string, reward QuestRewardMutation, options MutationOptions) (*platform.Character, error)
	GrantLearnedAbility(characterID string, abilityID string, options MutationOptions) (*platform.Character, error)
	AssignActionBarSlot(characterID string, slotIndex int, abilityID string, options MutationOptions) (*platform.Character, error)
	MoveActionBarSlot(characterID string, fromSlotIndex int, toSlotIndex int, options MutationOptions) (*platform.Character, error)
	ClearActionBarSlot(characterID string, slotIndex int, options MutationOptions) (*platform.Character, error)
}

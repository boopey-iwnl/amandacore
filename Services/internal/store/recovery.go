package store

import (
	"time"

	"amandacore/services/internal/observability"
)

const (
	EventPersistenceRecoveryStarted   = "persistence.recovery_started"
	EventPersistenceRecoveryCompleted = "persistence.recovery_completed"
	EventPersistenceRecoveryFailed    = "persistence.recovery_failed"
)

type SessionRecoveryState struct {
	CharacterID    string  `json:"characterId"`
	AccountID      string  `json:"accountId"`
	RealmID        string  `json:"realmId"`
	DisplayName    string  `json:"displayName"`
	ZoneID         string  `json:"zoneId"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Z              float64 `json:"z"`
	Level          int     `json:"level"`
	Experience     int     `json:"experience"`
	CurrencyCopper int     `json:"currencyCopper"`
	LastSeenAt     int64   `json:"lastSeenAt"`
}

func (s *FileStore) LoadSessionRecoveryState(characterID string) (SessionRecoveryState, error) {
	observability.LogEvent("store", EventPersistenceRecoveryStarted, map[string]any{"characterId": characterID})
	started := time.Now()
	character, err := s.GetCharacterByID(characterID)
	if err != nil {
		observability.LogEvent("store", EventPersistenceRecoveryFailed, map[string]any{
			"characterId": characterID,
			"reason":      err.Error(),
			"durationMs":  time.Since(started).Milliseconds(),
		})
		return SessionRecoveryState{}, err
	}
	recovery := SessionRecoveryState{
		CharacterID:    character.ID,
		AccountID:      character.AccountID,
		RealmID:        character.RealmID,
		DisplayName:    character.DisplayName,
		ZoneID:         character.ZoneID,
		X:              character.PositionX,
		Y:              character.PositionY,
		Z:              character.PositionZ,
		Level:          character.Level,
		Experience:     character.Experience,
		CurrencyCopper: character.CurrencyCopper,
		LastSeenAt:     character.LastSeenAt,
	}
	observability.LogEvent("store", EventPersistenceRecoveryCompleted, map[string]any{
		"characterId": characterID,
		"zoneId":      recovery.ZoneID,
		"durationMs":  time.Since(started).Milliseconds(),
	})
	return recovery, nil
}

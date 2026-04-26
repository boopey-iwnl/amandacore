package store

import (
	"fmt"
	"strings"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	EventPersistenceTransactionStarted    = "persistence.transaction_started"
	EventPersistenceTransactionCommitted  = "persistence.transaction_committed"
	EventPersistenceTransactionRolledBack = "persistence.transaction_rolled_back"
)

type FileStoreTx struct {
	state *state
}

func (s *FileStore) WithTransaction(operation string, fn func(*FileStoreTx) error) error {
	if fn == nil {
		return fmt.Errorf("transaction function is required")
	}
	operation = normalizeOperationName(operation)
	if err := s.lockState(true); err != nil {
		return err
	}
	defer s.unlockState()

	rollbackState, err := cloneStateForMigration(s.state)
	if err != nil {
		return err
	}
	observability.LogEvent("store", EventPersistenceTransactionStarted, map[string]any{"operation": operation})
	started := time.Now()
	if err := fn(&FileStoreTx{state: &s.state}); err != nil {
		s.state = rollbackState
		observability.LogEvent("store", EventPersistenceTransactionRolledBack, map[string]any{
			"operation":  operation,
			"reason":     err.Error(),
			"durationMs": time.Since(started).Milliseconds(),
		})
		return err
	}
	if err := s.saveLocked(); err != nil {
		s.state = rollbackState
		observability.LogEvent("store", EventPersistenceTransactionRolledBack, map[string]any{
			"operation":  operation,
			"reason":     err.Error(),
			"durationMs": time.Since(started).Milliseconds(),
		})
		return err
	}
	observability.LogEvent("store", EventPersistenceTransactionCommitted, map[string]any{
		"operation":  operation,
		"durationMs": time.Since(started).Milliseconds(),
	})
	return nil
}

func (s *FileStore) UpdateCharacterAtomically(operation string, characterID string, mutate func(*platform.Character) error) (*platform.Character, error) {
	var updated platform.Character
	err := s.WithTransaction(operation, func(tx *FileStoreTx) error {
		character, err := tx.GetCharacterByID(characterID)
		if err != nil {
			return err
		}
		if mutate == nil {
			return fmt.Errorf("character mutator is required")
		}
		if err := mutate(&character); err != nil {
			return err
		}
		saved, err := tx.SaveCharacter(character)
		if err != nil {
			return err
		}
		updated = saved
		return nil
	})
	if err != nil {
		return nil, err
	}
	copy := normalizedCharacterCopy(updated)
	return &copy, nil
}

func (tx *FileStoreTx) GetCharacterByID(characterID string) (platform.Character, error) {
	character, ok := tx.state.Characters[characterID]
	if !ok {
		return platform.Character{}, fmt.Errorf("character not found")
	}
	return normalizedCharacterCopy(character), nil
}

func (tx *FileStoreTx) SaveCharacter(character platform.Character) (platform.Character, error) {
	if strings.TrimSpace(character.ID) == "" {
		return platform.Character{}, fmt.Errorf("character id is required")
	}
	if _, ok := tx.state.Characters[character.ID]; !ok {
		return platform.Character{}, fmt.Errorf("character not found")
	}
	character = platform.NormalizeCharacter(character)
	character.LastSeenAt = time.Now().Unix()
	tx.state.Characters[character.ID] = character
	return normalizedCharacterCopy(character), nil
}

func (tx *FileStoreTx) SetCharacterPosition(characterID string, zoneID string, x float64, y float64, z float64) (platform.Character, error) {
	character, err := tx.GetCharacterByID(characterID)
	if err != nil {
		return platform.Character{}, err
	}
	character.ZoneID = zoneID
	character.PositionX = x
	character.PositionY = y
	character.PositionZ = z
	return tx.SaveCharacter(character)
}

func normalizeOperationName(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return "unnamed"
	}
	return operation
}

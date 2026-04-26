package worlds

import (
	"context"
	"sync"
	"time"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/simcore"
)

type CharacterStateWriter interface {
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
}

type DirtyCharacterState struct {
	CharacterID simcore.CharacterID `json:"characterId"`
	ZoneID      simcore.ZoneID      `json:"zoneId"`
	Position    simcore.Vector3     `json:"position"`
	Reason      string              `json:"reason"`
	MarkedAt    time.Time           `json:"markedAt"`
}

type PersistenceFlushRequest struct {
	Character   DirtyCharacterState `json:"character"`
	RequestedAt time.Time           `json:"requestedAt"`
}

type PersistenceFlushResult struct {
	CharacterID simcore.CharacterID `json:"characterId"`
	ZoneID      simcore.ZoneID      `json:"zoneId"`
	Position    simcore.Vector3     `json:"position"`
	StartedAt   time.Time           `json:"startedAt"`
	CompletedAt time.Time           `json:"completedAt"`
	Error       error               `json:"-"`
}

type PersistenceStats struct {
	PendingCharacters int   `json:"pendingCharacters"`
	FlushCount        int64 `json:"flushCount"`
	FlushFailures     int64 `json:"flushFailures"`
}

type PersistenceHandoff struct {
	mutex      sync.Mutex
	writer     CharacterStateWriter
	dirty      map[simcore.CharacterID]DirtyCharacterState
	flushCount int64
	failures   int64
}

func NewPersistenceHandoff(writer CharacterStateWriter) *PersistenceHandoff {
	return &PersistenceHandoff{
		writer: writer,
		dirty:  map[simcore.CharacterID]DirtyCharacterState{},
	}
}

func (h *PersistenceHandoff) MarkCharacterDirty(characterID simcore.CharacterID, zoneID simcore.ZoneID, position simcore.Vector3, reason string, now time.Time) {
	if characterID == "" {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.dirty[characterID] = DirtyCharacterState{
		CharacterID: characterID,
		ZoneID:      zoneID,
		Position:    position,
		Reason:      reason,
		MarkedAt:    now,
	}
}

func (h *PersistenceHandoff) FlushDirty(ctx context.Context) []PersistenceFlushResult {
	h.mutex.Lock()
	pending := make([]DirtyCharacterState, 0, len(h.dirty))
	for characterID, dirty := range h.dirty {
		pending = append(pending, dirty)
		delete(h.dirty, characterID)
	}
	h.mutex.Unlock()

	results := make([]PersistenceFlushResult, 0, len(pending))
	for _, dirty := range pending {
		startedAt := time.Now().UTC()
		result := PersistenceFlushResult{
			CharacterID: dirty.CharacterID,
			ZoneID:      dirty.ZoneID,
			Position:    dirty.Position,
			StartedAt:   startedAt,
		}
		if ctx.Err() != nil {
			result.Error = ctx.Err()
		} else if h.writer != nil {
			_, result.Error = h.writer.UpdateCharacterState(
				string(dirty.CharacterID),
				string(dirty.ZoneID),
				dirty.Position.X,
				dirty.Position.Y,
				dirty.Position.Z)
		}
		result.CompletedAt = time.Now().UTC()
		results = append(results, result)

		h.mutex.Lock()
		h.flushCount++
		if result.Error != nil {
			h.failures++
			h.dirty[dirty.CharacterID] = dirty
		}
		h.mutex.Unlock()
	}
	return results
}

func (h *PersistenceHandoff) Stats() PersistenceStats {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return PersistenceStats{
		PendingCharacters: len(h.dirty),
		FlushCount:        h.flushCount,
		FlushFailures:     h.failures,
	}
}

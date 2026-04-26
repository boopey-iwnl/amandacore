package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	EventPersistenceFlushStarted   = "persistence.flush_started"
	EventPersistenceFlushCompleted = "persistence.flush_completed"
	EventPersistenceFlushFailed    = "persistence.flush_failed"
)

type CharacterStateWriter interface {
	UpdateCharacterState(characterID string, zoneID string, x float64, y float64, z float64) (*platform.Character, error)
}

type DirtyStateFlushPolicy struct {
	FlushInterval time.Duration
	MaxPending    int
}

type DirtyCharacterState struct {
	CharacterID string  `json:"characterId"`
	ZoneID      string  `json:"zoneId"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
	Reason      string  `json:"reason"`
	MarkedAt    int64   `json:"markedAt"`
}

type DirtyFlushResult struct {
	Attempted int `json:"attempted"`
	Flushed   int `json:"flushed"`
	Failed    int `json:"failed"`
	Pending   int `json:"pending"`
}

type DirtyStateBuffer struct {
	mutex   sync.Mutex
	policy  DirtyStateFlushPolicy
	pending map[string]DirtyCharacterState
}

func NewDirtyStateBuffer(policy DirtyStateFlushPolicy) *DirtyStateBuffer {
	if policy.FlushInterval <= 0 {
		policy.FlushInterval = 5 * time.Second
	}
	if policy.MaxPending <= 0 {
		policy.MaxPending = 1024
	}
	return &DirtyStateBuffer{policy: policy, pending: map[string]DirtyCharacterState{}}
}

func (b *DirtyStateBuffer) MarkCharacterState(characterID string, zoneID string, x float64, y float64, z float64, reason string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.pending) >= b.policy.MaxPending {
		if _, ok := b.pending[characterID]; !ok {
			oldestID := ""
			var oldest int64
			for pendingID, state := range b.pending {
				if oldestID == "" || state.MarkedAt < oldest {
					oldestID = pendingID
					oldest = state.MarkedAt
				}
			}
			delete(b.pending, oldestID)
		}
	}
	b.pending[characterID] = DirtyCharacterState{
		CharacterID: characterID,
		ZoneID:      zoneID,
		X:           x,
		Y:           y,
		Z:           z,
		Reason:      reason,
		MarkedAt:    time.Now().UnixNano(),
	}
}

func (b *DirtyStateBuffer) PendingCount() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.pending)
}

func (b *DirtyStateBuffer) Flush(ctx context.Context, writer CharacterStateWriter) (DirtyFlushResult, error) {
	b.mutex.Lock()
	states := make([]DirtyCharacterState, 0, len(b.pending))
	for _, state := range b.pending {
		states = append(states, state)
	}
	b.mutex.Unlock()

	sort.Slice(states, func(left int, right int) bool {
		return states[left].CharacterID < states[right].CharacterID
	})

	result := DirtyFlushResult{Attempted: len(states)}
	observability.LogEvent("store", EventPersistenceFlushStarted, map[string]any{"attempted": result.Attempted})
	for _, state := range states {
		select {
		case <-ctx.Done():
			result.Pending = b.PendingCount()
			observability.LogEvent("store", EventPersistenceFlushFailed, map[string]any{"reason": ctx.Err().Error(), "flushed": result.Flushed})
			return result, ctx.Err()
		default:
		}
		if _, err := writer.UpdateCharacterState(state.CharacterID, state.ZoneID, state.X, state.Y, state.Z); err != nil {
			result.Failed++
			observability.LogEvent("store", EventPersistenceFlushFailed, map[string]any{"characterId": state.CharacterID, "reason": err.Error()})
			continue
		}
		result.Flushed++
		b.mutex.Lock()
		if current, ok := b.pending[state.CharacterID]; ok && current.MarkedAt == state.MarkedAt {
			delete(b.pending, state.CharacterID)
		}
		b.mutex.Unlock()
	}
	result.Pending = b.PendingCount()
	observability.LogEvent("store", EventPersistenceFlushCompleted, map[string]any{
		"attempted": result.Attempted,
		"flushed":   result.Flushed,
		"failed":    result.Failed,
		"pending":   result.Pending,
	})
	return result, nil
}

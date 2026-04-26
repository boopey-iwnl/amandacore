package worlds

import (
	"context"
	"errors"
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/simcore"
)

func (s *worldServer) attachRuntimeSessionLocked(session *worldSessionState, now time.Time) {
	if s.gateway == nil || s.runtime == nil || session == nil {
		return
	}
	position := simcore.Vector3{X: session.X, Y: session.Y, Z: session.Z}
	result, err := s.gateway.Attach(AttachSessionRequest{
		SessionID:             simcore.SessionID(session.Token),
		AccountID:             simcore.AccountID(session.AccountID),
		CharacterID:           simcore.CharacterID(session.CharacterID),
		RealmID:               simcore.RealmID(session.RealmID),
		ZoneID:                simcore.ZoneID(session.ZoneID),
		AuthoritativePosition: position,
		Now:                   now,
	})
	if err != nil {
		observability.LogEvent("world-service", observability.EventWorldSessionCommandRejected, map[string]any{
			"worldSessionToken": session.Token,
			"accountId":         session.AccountID,
			"characterId":       session.CharacterID,
			"reason":            err.Error(),
		})
		return
	}
	if result.Replaced != nil {
		observability.LogEvent("world-service", observability.EventWorldSessionReplaced, map[string]any{
			"oldSessionId": string(result.Replaced.SessionID),
			"newSessionId": string(result.Session.SessionID),
			"characterId":  session.CharacterID,
			"reason":       result.Replaced.LastDisconnectReason,
		})
	}
	observability.LogEvent("world-service", observability.EventWorldSessionAttached, map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"realmId":           session.RealmID,
		"zoneId":            session.ZoneID,
		"x":                 session.X,
		"y":                 session.Y,
		"z":                 session.Z,
	})
	_ = s.runtime.RegisterOrUpdateEntity(RuntimeEntity{
		ID:       simcore.EntityID(session.CharacterID),
		Kind:     "player",
		ZoneID:   simcore.ZoneID(session.ZoneID),
		Position: position,
	})
}

func (s *worldServer) submitRuntimeCommandLocked(envelope simcore.CommandEnvelope) (simcore.CommandEnvelope, error) {
	if s.runtime == nil {
		return simcore.CommandEnvelope{}, fmt.Errorf("world runtime is not available")
	}
	queued, err := s.runtime.Enqueue(envelope)
	if err != nil {
		observability.LogEvent("world-service", observability.EventWorldQueueBackpressure, map[string]any{
			"sessionId":   string(envelope.SessionID),
			"characterId": string(envelope.CharacterID),
			"reason":      err.Error(),
		})
		observability.LogEvent("world-service", observability.EventWorldCommandRejected, map[string]any{
			"sessionId":   string(envelope.SessionID),
			"characterId": string(envelope.CharacterID),
			"reason":      err.Error(),
		})
		return simcore.CommandEnvelope{}, err
	}
	observability.LogEvent("world-service", observability.EventWorldCommandEnqueued, map[string]any{
		"commandId":   string(queued.CommandID),
		"sessionId":   string(queued.SessionID),
		"characterId": string(queued.CharacterID),
		"commandKind": string(queued.Payload.CommandKind()),
		"queueDepth":  s.runtime.PendingCommandCount(),
	})
	return queued, nil
}

func (s *worldServer) runRuntimeTickLocked(now time.Time) TickResult {
	if s.runtime == nil {
		return TickResult{}
	}
	observability.LogEvent("world-service", observability.EventWorldTickStarted, map[string]any{
		"pendingCommands": s.runtime.PendingCommandCount(),
	})
	result := s.runtime.RunTick(now)
	observability.LogEvent("world-service", observability.EventWorldTickCommandBatchProcessed, map[string]any{
		"tickId":            uint64(result.Tick.ID),
		"commandsProcessed": result.CommandsProcessed,
		"commandsRejected":  result.CommandsRejected,
		"stateDiffs":        len(result.Diffs),
	})
	observability.LogEvent("world-service", observability.EventWorldTickCompleted, map[string]any{
		"tickId":               uint64(result.Tick.ID),
		"durationMs":           float64(result.Duration.Microseconds()) / 1000.0,
		"commandsProcessed":    result.CommandsProcessed,
		"commandsRejected":     result.CommandsRejected,
		"queueDepthBeforeTick": result.QueueDepthBeforeTick,
	})
	if result.Slow {
		observability.LogEvent("world-service", observability.EventWorldTickSlow, map[string]any{
			"tickId":     uint64(result.Tick.ID),
			"durationMs": float64(result.Duration.Microseconds()) / 1000.0,
		})
	}
	if len(result.Diffs) > 0 {
		observability.LogEvent("world-service", observability.EventWorldStateDiffEmitted, map[string]any{
			"tickId":     uint64(result.Tick.ID),
			"diffCount":  len(result.Diffs),
			"deltaCount": countStateDeltas(result.Diffs),
		})
	}
	for _, dirty := range result.DirtyCharacters {
		observability.LogEvent("world-service", observability.EventWorldEntityStateDirty, map[string]any{
			"characterId": string(dirty.CharacterID),
			"zoneId":      string(dirty.ZoneID),
			"reason":      dirty.Reason,
		})
	}
	for _, event := range result.Events {
		switch typed := event.(type) {
		case simcore.PlayerMovedEvent:
			observability.LogEvent("world-service", observability.EventWorldMovementAccepted, map[string]any{
				"characterId": string(typed.CharacterID),
				"zoneId":      string(typed.ZoneID),
				"x":           typed.To.X,
				"y":           typed.To.Y,
				"z":           typed.To.Z,
			})
		case simcore.PlayerCorrectedEvent:
			observability.LogEvent("world-service", observability.EventWorldMovementCorrected, map[string]any{
				"characterId": string(typed.CharacterID),
				"zoneId":      string(typed.ZoneID),
				"reason":      typed.Reason,
				"x":           typed.Corrected.X,
				"y":           typed.Corrected.Y,
				"z":           typed.Corrected.Z,
			})
		case simcore.CommandRejectedEvent:
			observability.LogEvent("world-service", observability.EventWorldMovementRejected, map[string]any{
				"characterId": string(typed.CharacterID),
				"sessionId":   string(typed.SessionID),
				"reason":      string(typed.Reason),
				"message":     typed.Message,
			})
		}
	}
	return result
}

func (s *worldServer) flushDirtyPersistenceLocked(ctx context.Context, reason string) []PersistenceFlushResult {
	if s.persistence == nil {
		return nil
	}
	statsBefore := s.persistence.Stats()
	if statsBefore.PendingCharacters > 0 {
		observability.LogEvent("world-service", observability.EventPersistenceFlushRequested, map[string]any{
			"pendingCharacters": statsBefore.PendingCharacters,
			"reason":            reason,
		})
	}
	results := s.persistence.FlushDirty(ctx)
	for _, result := range results {
		durationMs := float64(result.CompletedAt.Sub(result.StartedAt).Microseconds()) / 1000.0
		if result.Error != nil {
			observability.LogEvent("world-service", observability.EventPersistenceFlushFailed, map[string]any{
				"characterId": string(result.CharacterID),
				"zoneId":      string(result.ZoneID),
				"reason":      result.Error.Error(),
				"durationMs":  durationMs,
			})
			continue
		}
		observability.LogEvent("world-service", observability.EventPersistenceFlushCompleted, map[string]any{
			"characterId": string(result.CharacterID),
			"zoneId":      string(result.ZoneID),
			"durationMs":  durationMs,
		})
		observability.LogEvent("world-service", observability.EventPersistenceSnapshotSaved, map[string]any{
			"aggregateKind": "character",
			"aggregateId":   string(result.CharacterID),
			"zoneId":        string(result.ZoneID),
		})
		observability.LogEvent("world-service", observability.EventWorldEntityStateFlushed, map[string]any{
			"characterId": string(result.CharacterID),
			"zoneId":      string(result.ZoneID),
		})
	}
	return results
}

func (s *worldServer) processMoveIntentLocked(session *worldSessionState, deltaX float64, deltaY float64, now time.Time) (simcore.Vector3, string, error) {
	if session == nil {
		return simcore.Vector3{}, "", errors.New("world session is required")
	}
	if s.gateway == nil || s.runtime == nil {
		nextX, nextY := s.resolveMovementLocked(session, deltaX, deltaY)
		return simcore.Vector3{X: nextX, Y: nextY, Z: session.Z}, "legacy_movement", nil
	}

	command := simcore.MoveIntentCommand{
		CharacterID: simcore.CharacterID(session.CharacterID),
		Delta:       simcore.Vector3{X: deltaX, Y: deltaY, Z: 0},
	}
	queued, err := s.submitRuntimeCommandLocked(simcore.CommandEnvelope{
		SessionID:         simcore.SessionID(session.Token),
		AccountID:         simcore.AccountID(session.AccountID),
		CharacterID:       simcore.CharacterID(session.CharacterID),
		RealmID:           simcore.RealmID(session.RealmID),
		ZoneID:            simcore.ZoneID(session.ZoneID),
		ServerReceiveTime: now,
		Payload:           command,
	})
	if err != nil {
		return simcore.Vector3{X: session.X, Y: session.Y, Z: session.Z}, "queue_rejected", err
	}

	result := s.runRuntimeTickLocked(now)
	for _, rejection := range result.Rejections {
		if rejection.CommandID == queued.CommandID {
			message := rejection.Message
			if message == "" {
				message = string(rejection.Reason)
			}
			return simcore.Vector3{X: session.X, Y: session.Y, Z: session.Z}, string(rejection.Reason), errors.New(message)
		}
	}

	authoritative := simcore.Vector3{X: session.X, Y: session.Y, Z: session.Z}
	reason := "accepted"
	for _, diff := range result.Diffs {
		for _, delta := range diff.Deltas {
			switch typed := delta.(type) {
			case simcore.PositionDelta:
				if typed.EntityID == simcore.EntityID(session.CharacterID) {
					authoritative = typed.To
				}
			case simcore.CorrectionDelta:
				if typed.EntityID == simcore.EntityID(session.CharacterID) {
					authoritative = typed.AuthoritativePosition
					reason = typed.ReasonCode
				}
			}
		}
	}
	return authoritative, reason, nil
}

func countStateDeltas(diffs []simcore.StateDiff) int {
	count := 0
	for _, diff := range diffs {
		count += len(diff.Deltas)
	}
	return count
}

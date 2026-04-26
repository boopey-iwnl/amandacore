package observability

import (
	"encoding/json"
	"log"
	"time"
)

const (
	EventAccountRegistered       = "account.registered"
	EventAuthSessionIssued       = "auth.session_issued"
	EventCharacterCreated        = "character.created"
	EventCharacterSelected       = "character.selected"
	EventWorldJoinTicketIssued   = "world.join_ticket_issued"
	EventWorldJoinTicketConsumed = "world.join_ticket_consumed"
	EventWorldPlayerSpawned      = "world.player_spawned"
	EventWorldCharacterSaved     = "world.character_saved"
	EventWorldReconnected        = "world.reconnected"

	EventWorldSessionAttached           = "world.session.attached"
	EventWorldSessionDetached           = "world.session.detached"
	EventWorldSessionReplaced           = "world.session.replaced"
	EventWorldSessionReconnectRequested = "world.session.reconnect_requested"
	EventWorldSessionReconnectCompleted = "world.session.reconnect_completed"
	EventWorldSessionExpired            = "world.session.expired"
	EventWorldSessionCommandRejected    = "world.session.command_rejected"

	EventWorldCommandEnqueued   = "world.command.enqueued"
	EventWorldCommandRejected   = "world.command.rejected"
	EventWorldQueueBackpressure = "world.queue.backpressure"

	EventWorldTickStarted               = "world.tick.started"
	EventWorldTickCompleted             = "world.tick.completed"
	EventWorldTickSlow                  = "world.tick.slow"
	EventWorldTickCommandBatchProcessed = "world.tick.command_batch_processed"

	EventWorldMovementAccepted   = "world.movement.accepted"
	EventWorldMovementCorrected  = "world.movement.corrected"
	EventWorldMovementRejected   = "world.movement.rejected"
	EventWorldStateDiffEmitted   = "world.state.diff_emitted"
	EventWorldEntityStateDirty   = "world.entity.state_dirty"
	EventWorldEntityStateFlushed = "world.entity.state_flushed"

	EventPersistenceFlushRequested = "persistence.flush.requested"
	EventPersistenceFlushCompleted = "persistence.flush.completed"
	EventPersistenceFlushFailed    = "persistence.flush.failed"
	EventPersistenceSnapshotSaved  = "persistence.snapshot.saved"

	EventLoadsimStarted   = "loadsim.started"
	EventLoadsimCompleted = "loadsim.completed"
)

func StableEventNames() []string {
	return []string{
		EventAccountRegistered,
		EventAuthSessionIssued,
		EventCharacterCreated,
		EventCharacterSelected,
		EventWorldJoinTicketIssued,
		EventWorldJoinTicketConsumed,
		EventWorldPlayerSpawned,
		EventWorldCharacterSaved,
		EventWorldReconnected,
		EventWorldSessionAttached,
		EventWorldSessionDetached,
		EventWorldSessionReplaced,
		EventWorldSessionReconnectRequested,
		EventWorldSessionReconnectCompleted,
		EventWorldSessionExpired,
		EventWorldSessionCommandRejected,
		EventWorldCommandEnqueued,
		EventWorldCommandRejected,
		EventWorldQueueBackpressure,
		EventWorldTickStarted,
		EventWorldTickCompleted,
		EventWorldTickSlow,
		EventWorldTickCommandBatchProcessed,
		EventWorldMovementAccepted,
		EventWorldMovementCorrected,
		EventWorldMovementRejected,
		EventWorldStateDiffEmitted,
		EventWorldEntityStateDirty,
		EventWorldEntityStateFlushed,
		EventPersistenceFlushRequested,
		EventPersistenceFlushCompleted,
		EventPersistenceFlushFailed,
		EventPersistenceSnapshotSaved,
		EventLoadsimStarted,
		EventLoadsimCompleted,
	}
}

func LogEvent(service string, event string, fields map[string]any) {
	payload := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"service":   service,
		"event":     event,
	}

	for key, value := range fields {
		payload[key] = value
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		log.Printf("{\"timestamp\":\"%s\",\"service\":\"%s\",\"event\":\"logging_failed\",\"message\":%q}", time.Now().UTC().Format(time.RFC3339Nano), service, err.Error())
		return
	}

	log.Print(string(serialized))
}

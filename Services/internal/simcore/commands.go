package simcore

import (
	"sort"
	"time"
)

type TickID uint64
type SessionID string
type AccountID string
type CharacterID string
type RealmID string
type ZoneID string
type EntityID string
type CommandID string

type CommandKind string
type EventKind string
type RejectionReason string

const (
	CommandMoveIntent          CommandKind = "move_intent"
	CommandStopMoveIntent      CommandKind = "stop_move_intent"
	CommandFaceDirectionIntent CommandKind = "face_direction_intent"
	CommandInteractIntent      CommandKind = "interact_intent"
	CommandUseAbilityIntent    CommandKind = "use_ability_intent"
	CommandSelectTargetIntent  CommandKind = "select_target_intent"
	CommandHeartbeatIntent     CommandKind = "heartbeat_intent"
	CommandDisconnectIntent    CommandKind = "disconnect_intent"
	CommandReconnectIntent     CommandKind = "reconnect_intent"
	CommandAdminOperation      CommandKind = "admin_operation"
)

const (
	EventPlayerSpawned            EventKind = "player_spawned"
	EventPlayerMoved              EventKind = "player_moved"
	EventPlayerCorrected          EventKind = "player_corrected"
	EventPlayerDisconnected       EventKind = "player_disconnected"
	EventPlayerReconnected        EventKind = "player_reconnected"
	EventWorldJoinTicketIssued    EventKind = "world_join_ticket_issued"
	EventWorldJoinTicketConsumed  EventKind = "world_join_ticket_consumed"
	EventCombatIntentSubmitted    EventKind = "combat_intent_submitted"
	EventAbilityResolved          EventKind = "ability_resolved"
	EventNpcSpawned               EventKind = "npc_spawned"
	EventEntityDefeated           EventKind = "entity_defeated"
	EventPersistenceSnapshotSaved EventKind = "persistence_snapshot_saved"
	EventAdminOperationRequested  EventKind = "admin_operation_requested"
	EventAdminOperationApplied    EventKind = "admin_operation_applied"
	EventCommandRejected          EventKind = "command_rejected"
)

const (
	RejectionUnauthenticated RejectionReason = "unauthenticated"
	RejectionSessionInactive RejectionReason = "session_inactive"
	RejectionSessionUnbound  RejectionReason = "session_unbound"
	RejectionQueueFull       RejectionReason = "queue_full"
	RejectionInvalidPayload  RejectionReason = "invalid_payload"
	RejectionInvalidMovement RejectionReason = "invalid_movement"
)

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ClientCommand interface {
	CommandKind() CommandKind
	CommandActorID() CharacterID
}

type MoveIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	Delta       Vector3     `json:"delta"`
}

func (c MoveIntentCommand) CommandKind() CommandKind    { return CommandMoveIntent }
func (c MoveIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type StopMoveIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
}

func (c StopMoveIntentCommand) CommandKind() CommandKind    { return CommandStopMoveIntent }
func (c StopMoveIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type FaceDirectionIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	YawDegrees  float64     `json:"yawDegrees"`
}

func (c FaceDirectionIntentCommand) CommandKind() CommandKind    { return CommandFaceDirectionIntent }
func (c FaceDirectionIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type InteractIntentCommand struct {
	CharacterID   CharacterID `json:"characterId"`
	InteractionID string      `json:"interactionId"`
	TargetID      EntityID    `json:"targetId,omitempty"`
}

func (c InteractIntentCommand) CommandKind() CommandKind    { return CommandInteractIntent }
func (c InteractIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type UseAbilityIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	AbilityID   string      `json:"abilityId"`
	TargetID    EntityID    `json:"targetId,omitempty"`
}

func (c UseAbilityIntentCommand) CommandKind() CommandKind    { return CommandUseAbilityIntent }
func (c UseAbilityIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type SelectTargetIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	TargetID    EntityID    `json:"targetId"`
}

func (c SelectTargetIntentCommand) CommandKind() CommandKind    { return CommandSelectTargetIntent }
func (c SelectTargetIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type HeartbeatIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
}

func (c HeartbeatIntentCommand) CommandKind() CommandKind    { return CommandHeartbeatIntent }
func (c HeartbeatIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type DisconnectIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	Reason      string      `json:"reason,omitempty"`
}

func (c DisconnectIntentCommand) CommandKind() CommandKind    { return CommandDisconnectIntent }
func (c DisconnectIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type ReconnectIntentCommand struct {
	CharacterID CharacterID `json:"characterId"`
	Reason      string      `json:"reason,omitempty"`
}

func (c ReconnectIntentCommand) CommandKind() CommandKind    { return CommandReconnectIntent }
func (c ReconnectIntentCommand) CommandActorID() CharacterID { return c.CharacterID }

type AdminCommand struct {
	ActorAccountID AccountID   `json:"actorAccountId"`
	Action         string      `json:"action"`
	CharacterID    CharacterID `json:"characterId,omitempty"`
	Reason         string      `json:"reason,omitempty"`
}

func (c AdminCommand) CommandKind() CommandKind    { return CommandAdminOperation }
func (c AdminCommand) CommandActorID() CharacterID { return c.CharacterID }

type CommandValidation struct {
	Accepted bool            `json:"accepted"`
	Reason   RejectionReason `json:"reason,omitempty"`
	Message  string          `json:"message,omitempty"`
}

type CommandEnvelope struct {
	CommandID         CommandID         `json:"commandId"`
	SessionID         SessionID         `json:"sessionId"`
	AccountID         AccountID         `json:"accountId,omitempty"`
	CharacterID       CharacterID       `json:"characterId"`
	RealmID           RealmID           `json:"realmId,omitempty"`
	ZoneID            ZoneID            `json:"zoneId,omitempty"`
	ClientSequence    uint64            `json:"clientSequence,omitempty"`
	ServerReceiveTime time.Time         `json:"serverReceiveTime"`
	IntendedTick      TickID            `json:"intendedTick,omitempty"`
	EnqueueTick       TickID            `json:"enqueueTick,omitempty"`
	Payload           ClientCommand     `json:"-"`
	Validation        CommandValidation `json:"validation"`
}

func CompareCommandEnvelopes(left CommandEnvelope, right CommandEnvelope) int {
	if left.IntendedTick != right.IntendedTick {
		if left.IntendedTick < right.IntendedTick {
			return -1
		}
		return 1
	}
	if !left.ServerReceiveTime.Equal(right.ServerReceiveTime) {
		if left.ServerReceiveTime.Before(right.ServerReceiveTime) {
			return -1
		}
		return 1
	}
	if left.SessionID != right.SessionID {
		if left.SessionID < right.SessionID {
			return -1
		}
		return 1
	}
	if left.CharacterID != right.CharacterID {
		if left.CharacterID < right.CharacterID {
			return -1
		}
		return 1
	}
	if left.CommandID != right.CommandID {
		if left.CommandID < right.CommandID {
			return -1
		}
		return 1
	}
	return 0
}

func SortCommandEnvelopes(commands []CommandEnvelope) {
	sort.SliceStable(commands, func(left int, right int) bool {
		return CompareCommandEnvelopes(commands[left], commands[right]) < 0
	})
}

type DomainEvent interface {
	DomainEventKind() EventKind
	DomainEventCharacterID() CharacterID
}

type PlayerSpawnedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	ZoneID      ZoneID      `json:"zoneId"`
	Position    Vector3     `json:"position"`
}

func (e PlayerSpawnedEvent) DomainEventKind() EventKind          { return EventPlayerSpawned }
func (e PlayerSpawnedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type PlayerMovedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	ZoneID      ZoneID      `json:"zoneId"`
	From        Vector3     `json:"from"`
	To          Vector3     `json:"to"`
}

func (e PlayerMovedEvent) DomainEventKind() EventKind          { return EventPlayerMoved }
func (e PlayerMovedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type PlayerCorrectedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	ZoneID      ZoneID      `json:"zoneId"`
	Requested   Vector3     `json:"requested"`
	Corrected   Vector3     `json:"corrected"`
	Reason      string      `json:"reason"`
}

func (e PlayerCorrectedEvent) DomainEventKind() EventKind          { return EventPlayerCorrected }
func (e PlayerCorrectedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type PlayerDisconnectedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	ZoneID      ZoneID      `json:"zoneId,omitempty"`
	Reason      string      `json:"reason,omitempty"`
}

func (e PlayerDisconnectedEvent) DomainEventKind() EventKind          { return EventPlayerDisconnected }
func (e PlayerDisconnectedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type PlayerReconnectedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	ZoneID      ZoneID      `json:"zoneId,omitempty"`
	Position    Vector3     `json:"position"`
	Reason      string      `json:"reason,omitempty"`
}

func (e PlayerReconnectedEvent) DomainEventKind() EventKind          { return EventPlayerReconnected }
func (e PlayerReconnectedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type CombatIntentSubmittedEvent struct {
	CharacterID CharacterID `json:"characterId"`
	AbilityID   string      `json:"abilityId"`
	TargetID    EntityID    `json:"targetId,omitempty"`
}

func (e CombatIntentSubmittedEvent) DomainEventKind() EventKind { return EventCombatIntentSubmitted }
func (e CombatIntentSubmittedEvent) DomainEventCharacterID() CharacterID {
	return e.CharacterID
}

type CommandRejectedEvent struct {
	CharacterID CharacterID     `json:"characterId"`
	SessionID   SessionID       `json:"sessionId"`
	CommandID   CommandID       `json:"commandId"`
	Reason      RejectionReason `json:"reason"`
	Message     string          `json:"message,omitempty"`
}

func (e CommandRejectedEvent) DomainEventKind() EventKind          { return EventCommandRejected }
func (e CommandRejectedEvent) DomainEventCharacterID() CharacterID { return e.CharacterID }

type StateDiff struct {
	TickID TickID       `json:"tickId"`
	Deltas []StateDelta `json:"-"`
}

type StateDelta interface {
	StateDeltaKind() string
}

type EntityStateDelta struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
}

func (d EntityStateDelta) StateDeltaKind() string { return "entity_state" }

type PositionDelta struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
	From     Vector3  `json:"from"`
	To       Vector3  `json:"to"`
}

func (d PositionDelta) StateDeltaKind() string { return "position" }

type SpawnDelta struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
	Position Vector3  `json:"position"`
}

func (d SpawnDelta) StateDeltaKind() string { return "spawn" }

type DespawnDelta struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
	Reason   string   `json:"reason,omitempty"`
}

func (d DespawnDelta) StateDeltaKind() string { return "despawn" }

type CorrectionDelta struct {
	EntityID              EntityID `json:"entityId"`
	ZoneID                ZoneID   `json:"zoneId"`
	RequestedPosition     Vector3  `json:"requestedPosition"`
	AuthoritativePosition Vector3  `json:"authoritativePosition"`
	ReasonCode            string   `json:"reasonCode"`
	ServerTick            TickID   `json:"serverTick"`
}

func (d CorrectionDelta) StateDeltaKind() string { return "correction" }

type EventDelta struct {
	Event DomainEvent `json:"-"`
}

func (d EventDelta) StateDeltaKind() string { return "event" }

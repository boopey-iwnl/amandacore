package simcore

import "time"

type TickID uint64
type ZoneID string
type InstanceID string
type EntityID string
type SessionID string
type CommandID string
type EventID string

type CommandKind string
type EventKind string

const (
	CommandMoveIntent       CommandKind = "move_intent"
	CommandAbilityIntent    CommandKind = "ability_intent"
	CommandTargetSelection  CommandKind = "target_selection"
	CommandInteractIntent   CommandKind = "interact_intent"
	CommandDisconnectIntent CommandKind = "disconnect_intent"
	CommandReconnectIntent  CommandKind = "reconnect_intent"
	CommandAdminOperation   CommandKind = "admin_operation"
)

const (
	EventPlayerSpawned            EventKind = "player_spawned"
	EventPlayerMoved              EventKind = "player_moved"
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
)

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type ClientCommand interface {
	CommandKind() CommandKind
	CommandActorID() EntityID
}

type MoveIntentCommand struct {
	EntityID EntityID `json:"entityId"`
	From     Vector3  `json:"from"`
	Delta    Vector3  `json:"delta"`
	To       Vector3  `json:"to"`
}

func (c MoveIntentCommand) CommandKind() CommandKind { return CommandMoveIntent }
func (c MoveIntentCommand) CommandActorID() EntityID { return c.EntityID }

type AbilityIntentCommand struct {
	EntityID  EntityID `json:"entityId"`
	AbilityID string   `json:"abilityId"`
	TargetID  EntityID `json:"targetId,omitempty"`
}

func (c AbilityIntentCommand) CommandKind() CommandKind { return CommandAbilityIntent }
func (c AbilityIntentCommand) CommandActorID() EntityID { return c.EntityID }

type TargetSelectionCommand struct {
	EntityID EntityID `json:"entityId"`
	TargetID EntityID `json:"targetId"`
}

func (c TargetSelectionCommand) CommandKind() CommandKind { return CommandTargetSelection }
func (c TargetSelectionCommand) CommandActorID() EntityID { return c.EntityID }

type InteractIntentCommand struct {
	EntityID      EntityID `json:"entityId"`
	InteractionID string   `json:"interactionId"`
	TargetID      EntityID `json:"targetId,omitempty"`
}

func (c InteractIntentCommand) CommandKind() CommandKind { return CommandInteractIntent }
func (c InteractIntentCommand) CommandActorID() EntityID { return c.EntityID }

type DisconnectIntentCommand struct {
	EntityID EntityID `json:"entityId"`
	Reason   string   `json:"reason,omitempty"`
}

func (c DisconnectIntentCommand) CommandKind() CommandKind { return CommandDisconnectIntent }
func (c DisconnectIntentCommand) CommandActorID() EntityID { return c.EntityID }

type ReconnectIntentCommand struct {
	EntityID EntityID `json:"entityId"`
	Reason   string   `json:"reason,omitempty"`
}

func (c ReconnectIntentCommand) CommandKind() CommandKind { return CommandReconnectIntent }
func (c ReconnectIntentCommand) CommandActorID() EntityID { return c.EntityID }

type AdminCommand struct {
	ActorAccountID string   `json:"actorAccountId"`
	Action         string   `json:"action"`
	TargetEntityID EntityID `json:"targetEntityId,omitempty"`
	Reason         string   `json:"reason,omitempty"`
}

func (c AdminCommand) CommandKind() CommandKind { return CommandAdminOperation }
func (c AdminCommand) CommandActorID() EntityID { return c.TargetEntityID }

type CommandEnvelope struct {
	CommandID  CommandID     `json:"commandId"`
	Sequence   uint64        `json:"sequence"`
	ReceivedAt time.Time     `json:"receivedAt"`
	SessionID  SessionID     `json:"sessionId,omitempty"`
	ActorID    EntityID      `json:"actorId,omitempty"`
	ZoneID     ZoneID        `json:"zoneId,omitempty"`
	Command    ClientCommand `json:"-"`
}

type DomainEvent interface {
	DomainEventKind() EventKind
	DomainEventActorID() EntityID
}

type PlayerSpawnedEvent struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
	Position Vector3  `json:"position"`
}

func (e PlayerSpawnedEvent) DomainEventKind() EventKind   { return EventPlayerSpawned }
func (e PlayerSpawnedEvent) DomainEventActorID() EntityID { return e.EntityID }

type PlayerMovedEvent struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId"`
	From     Vector3  `json:"from"`
	To       Vector3  `json:"to"`
}

func (e PlayerMovedEvent) DomainEventKind() EventKind   { return EventPlayerMoved }
func (e PlayerMovedEvent) DomainEventActorID() EntityID { return e.EntityID }

type PlayerDisconnectedEvent struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

func (e PlayerDisconnectedEvent) DomainEventKind() EventKind   { return EventPlayerDisconnected }
func (e PlayerDisconnectedEvent) DomainEventActorID() EntityID { return e.EntityID }

type PlayerReconnectedEvent struct {
	EntityID EntityID `json:"entityId"`
	ZoneID   ZoneID   `json:"zoneId,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

func (e PlayerReconnectedEvent) DomainEventKind() EventKind   { return EventPlayerReconnected }
func (e PlayerReconnectedEvent) DomainEventActorID() EntityID { return e.EntityID }

type WorldJoinTicketIssuedEvent struct {
	TicketID    string   `json:"ticketId"`
	AccountID   string   `json:"accountId"`
	CharacterID EntityID `json:"characterId"`
	RealmID     string   `json:"realmId"`
}

func (e WorldJoinTicketIssuedEvent) DomainEventKind() EventKind   { return EventWorldJoinTicketIssued }
func (e WorldJoinTicketIssuedEvent) DomainEventActorID() EntityID { return e.CharacterID }

type WorldJoinTicketConsumedEvent struct {
	TicketID    string   `json:"ticketId"`
	AccountID   string   `json:"accountId"`
	CharacterID EntityID `json:"characterId"`
	RealmID     string   `json:"realmId"`
}

func (e WorldJoinTicketConsumedEvent) DomainEventKind() EventKind {
	return EventWorldJoinTicketConsumed
}
func (e WorldJoinTicketConsumedEvent) DomainEventActorID() EntityID { return e.CharacterID }

type CombatIntentSubmittedEvent struct {
	EntityID  EntityID `json:"entityId"`
	AbilityID string   `json:"abilityId"`
	TargetID  EntityID `json:"targetId,omitempty"`
}

func (e CombatIntentSubmittedEvent) DomainEventKind() EventKind   { return EventCombatIntentSubmitted }
func (e CombatIntentSubmittedEvent) DomainEventActorID() EntityID { return e.EntityID }

type AbilityResolvedEvent struct {
	EntityID  EntityID `json:"entityId"`
	AbilityID string   `json:"abilityId"`
	TargetID  EntityID `json:"targetId,omitempty"`
	Outcome   string   `json:"outcome"`
}

func (e AbilityResolvedEvent) DomainEventKind() EventKind   { return EventAbilityResolved }
func (e AbilityResolvedEvent) DomainEventActorID() EntityID { return e.EntityID }

type NpcSpawnedEvent struct {
	EntityID    EntityID `json:"entityId"`
	ArchetypeID string   `json:"archetypeId"`
	ZoneID      ZoneID   `json:"zoneId"`
	Position    Vector3  `json:"position"`
}

func (e NpcSpawnedEvent) DomainEventKind() EventKind   { return EventNpcSpawned }
func (e NpcSpawnedEvent) DomainEventActorID() EntityID { return e.EntityID }

type EntityDefeatedEvent struct {
	EntityID     EntityID `json:"entityId"`
	DefeatedByID EntityID `json:"defeatedById,omitempty"`
	ZoneID       ZoneID   `json:"zoneId,omitempty"`
}

func (e EntityDefeatedEvent) DomainEventKind() EventKind   { return EventEntityDefeated }
func (e EntityDefeatedEvent) DomainEventActorID() EntityID { return e.EntityID }

type PersistenceSnapshotSavedEvent struct {
	AggregateID   string    `json:"aggregateId"`
	AggregateKind string    `json:"aggregateKind"`
	SavedAt       time.Time `json:"savedAt"`
}

func (e PersistenceSnapshotSavedEvent) DomainEventKind() EventKind {
	return EventPersistenceSnapshotSaved
}
func (e PersistenceSnapshotSavedEvent) DomainEventActorID() EntityID { return "" }

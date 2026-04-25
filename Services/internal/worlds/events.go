package worlds

import (
	"time"

	"amandacore/services/internal/observability"
)

const (
	eventNPCSpawnPointLoaded       = "npc.spawn_point.loaded"
	eventNPCSpawned                = "npc.spawned"
	eventNPCDied                   = "npc.died"
	eventNPCRespawnScheduled       = "npc.respawn_scheduled"
	eventNPCRespawned              = "npc.respawned"
	eventNPCAggroStarted           = "npc.aggro_started"
	eventNPCAggroLost              = "npc.aggro_lost"
	eventNPCAttackResolved         = "npc.attack_resolved"
	eventNPCLeashReset             = "npc.leash_reset"
	eventNPCTargetLost             = "npc.target_lost"
	eventWorldEntitySpawned        = "world.entity.spawned"
	eventWorldEntityRemoved        = "world.entity.removed"
	eventWorldStateDiffEmitted     = "world.state.diff_emitted"
	eventCombatTargetSelected      = "combat.target.selected"
	eventCombatTargetRejected      = "combat.target.rejected"
	eventCombatIntentSubmitted     = "combat.intent_submitted"
	eventCombatAbilityRejected     = "combat.ability_rejected"
	eventCombatAbilityResolved     = "combat.ability_resolved"
	eventCombatDamageApplied       = "combat.damage_applied"
	eventCombatCooldownStarted     = "combat.cooldown_started"
	eventCombatTargetDefeated      = "combat.target_defeated"
	eventEntityHealthChanged       = "entity.health_changed"
	eventEntityDied                = "entity.died"
	eventPlayerDied                = "player.died"
	eventProgressionKillCredit     = "progression.kill_credit_awarded"
	eventProgressionKillPersisted  = "progression.kill_credit_persisted"
	eventLoadsimCombatStarted      = "loadsim.combat.started"
	eventLoadsimCombatCompleted    = "loadsim.combat.completed"
	diffEntitySpawn                = "EntitySpawnDelta"
	diffEntityHealth               = "EntityHealthDelta"
	diffEntityCombatState          = "EntityCombatStateDelta"
	diffTargetSelection            = "TargetSelectionDelta"
	diffAbilityResult              = "AbilityResultDelta"
	diffEntityDeath                = "EntityDeathDelta"
	diffProgression                = "ProgressionDelta"
	recentWorldEventRetentionLimit = 256
)

type SimulationTick int64

type DomainEvent struct {
	Sequence int64          `json:"sequence"`
	Name     string         `json:"name"`
	Tick     SimulationTick `json:"tick"`
	AtUnixMs int64          `json:"atUnixMs"`
	Payload  map[string]any `json:"payload"`
}

type StateDiff struct {
	Sequence int64          `json:"sequence"`
	Type     string         `json:"type"`
	EntityID string         `json:"entityId,omitempty"`
	Tick     SimulationTick `json:"tick"`
	AtUnixMs int64          `json:"atUnixMs"`
	Payload  map[string]any `json:"payload"`
}

type EntitySpawnDelta struct {
	EntityID     string `json:"entityId"`
	ArchetypeID  string `json:"archetypeId,omitempty"`
	SpawnPointID string `json:"spawnPointId,omitempty"`
	ZoneID       string `json:"zoneId"`
}

type EntityHealthDelta struct {
	EntityID      string  `json:"entityId"`
	CurrentHealth float64 `json:"currentHealth"`
	MaxHealth     float64 `json:"maxHealth"`
}

type EntityCombatStateDelta struct {
	EntityID              string `json:"entityId"`
	IsInCombat            bool   `json:"isInCombat"`
	CurrentTargetEntityID string `json:"currentTargetEntityId,omitempty"`
}

type TargetSelectionDelta struct {
	CharacterID string `json:"characterId"`
	TargetID    string `json:"targetId"`
}

type AbilityResultDelta struct {
	CharacterID string  `json:"characterId"`
	AbilityID   string  `json:"abilityId"`
	TargetID    string  `json:"targetId,omitempty"`
	Damage      float64 `json:"damage,omitempty"`
}

type EntityDeathDelta struct {
	EntityID string `json:"entityId"`
	KilledBy string `json:"killedBy,omitempty"`
}

type ProgressionDelta struct {
	CharacterID string `json:"characterId"`
	ArchetypeID string `json:"archetypeId"`
	Count       int    `json:"count"`
}

func (s *worldServer) emitDomainEventLocked(name string, payload map[string]any) DomainEvent {
	if payload == nil {
		payload = map[string]any{}
	}
	s.eventSequence++
	nowMs := time.Now().UnixMilli()
	event := DomainEvent{
		Sequence: s.eventSequence,
		Name:     name,
		Tick:     SimulationTick(nowMs),
		AtUnixMs: nowMs,
		Payload:  cloneAnyMap(payload),
	}
	s.domainEvents = appendCappedDomainEvent(s.domainEvents, event)
	observability.LogEvent("world-service", name, payload)
	return event
}

func (s *worldServer) emitStateDiffLocked(diffType string, entityID string, payload map[string]any) StateDiff {
	if payload == nil {
		payload = map[string]any{}
	}
	s.eventSequence++
	nowMs := time.Now().UnixMilli()
	diff := StateDiff{
		Sequence: s.eventSequence,
		Type:     diffType,
		EntityID: entityID,
		Tick:     SimulationTick(nowMs),
		AtUnixMs: nowMs,
		Payload:  cloneAnyMap(payload),
	}
	s.stateDiffs = appendCappedStateDiff(s.stateDiffs, diff)
	s.emitDomainEventLocked(eventWorldStateDiffEmitted, map[string]any{
		"diffType": diffType,
		"entityId": entityID,
		"sequence": diff.Sequence,
	})
	return diff
}

func appendCappedDomainEvent(events []DomainEvent, event DomainEvent) []DomainEvent {
	events = append(events, event)
	if len(events) > recentWorldEventRetentionLimit {
		events = events[len(events)-recentWorldEventRetentionLimit:]
	}
	return events
}

func appendCappedStateDiff(diffs []StateDiff, diff StateDiff) []StateDiff {
	diffs = append(diffs, diff)
	if len(diffs) > recentWorldEventRetentionLimit {
		diffs = diffs[len(diffs)-recentWorldEventRetentionLimit:]
	}
	return diffs
}

func cloneAnyMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func (s *worldServer) recentDomainEventsLocked() []DomainEvent {
	return append([]DomainEvent(nil), s.domainEvents...)
}

func (s *worldServer) recentStateDiffsLocked() []StateDiff {
	return append([]StateDiff(nil), s.stateDiffs...)
}

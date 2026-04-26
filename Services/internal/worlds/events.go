package worlds

import (
	"time"

	"amandacore/services/internal/observability"
)

const (
	EventNPCSpawnPointLoaded       = "npc.spawn_point.loaded"
	EventNPCSpawned                = "npc.spawned"
	EventNPCDied                   = "npc.died"
	EventNPCRespawnScheduled       = "npc.respawn_scheduled"
	EventNPCRespawned              = "npc.respawned"
	EventNPCAggroStarted           = "npc.aggro_started"
	EventNPCAggroLost              = "npc.aggro_lost"
	EventNPCAttackResolved         = "npc.attack_resolved"
	EventNPCLeashReset             = "npc.leash_reset"
	EventNPCTargetLost             = "npc.target_lost"
	EventCombatTargetSelected      = "combat.target.selected"
	EventCombatTargetRejected      = "combat.target.rejected"
	EventCombatIntentSubmitted     = "combat.intent_submitted"
	EventCombatAbilityRejected     = "combat.ability_rejected"
	EventCombatAbilityResolved     = "combat.ability_resolved"
	EventCombatDamageApplied       = "combat.damage_applied"
	EventCombatCooldownStarted     = "combat.cooldown_started"
	EventCombatTargetDefeated      = "combat.target_defeated"
	EventEntityHealthChanged       = "entity.health_changed"
	EventEntityDied                = "entity.died"
	EventPlayerDied                = "player.died"
	EventProgressionCreditAwarded  = "progression.kill_credit_awarded"
	EventProgressionCreditSaved    = "progression.kill_credit_persisted"
	EventWorldEntitySpawned        = "world.entity.spawned"
	EventWorldEntityRemoved        = "world.entity.removed"
	EventWorldStateDiffEmitted     = "world.state.diff_emitted"
	EventLoadsimQuestStarted       = "loadsim.quest.started"
	EventLoadsimQuestCompleted     = "loadsim.quest.completed"
	EventLoadsimCombatStarted      = "loadsim.combat.started"
	EventLoadsimCombatCompleted    = "loadsim.combat.completed"
	eventItemCatalogLoaded         = "item.catalog.loaded"
	eventInventoryGrantRequested   = "inventory.item_grant_requested"
	eventInventoryItemGranted      = "inventory.item_granted"
	eventInventoryGrantRejected    = "inventory.item_grant_rejected"
	eventInventoryStackUpdated     = "inventory.stack_updated"
	eventInventoryFull             = "inventory.full"
	eventInventoryPersisted        = "inventory.persisted"
	eventLootTableLoaded           = "loot.table.loaded"
	eventLootRollStarted           = "loot.roll.started"
	eventLootRollCompleted         = "loot.roll.completed"
	eventLootContainerCreated      = "loot.container_created"
	eventLootInspectRequested      = "loot.inspect_requested"
	eventLootInspectResolved       = "loot.inspect_resolved"
	eventLootClaimRequested        = "loot.claim_requested"
	eventLootClaimCompleted        = "loot.claim_completed"
	eventLootClaimRejected         = "loot.claim_rejected"
	eventLootContainerExpired      = "loot.container_expired"
	eventProgressionCreditAwarded  = EventProgressionCreditAwarded
	eventProgressionCreditRejected = "progression.kill_credit_rejected"
	eventProgressionCreditSaved    = EventProgressionCreditSaved
	eventProgressionKillCredit     = EventProgressionCreditAwarded
	eventProgressionKillPersisted  = EventProgressionCreditSaved
	eventQuestCatalogLoaded        = "quest.catalog.loaded"
	eventQuestAcceptRequested      = "quest.accept_requested"
	eventQuestAccepted             = "quest.accepted"
	eventQuestAcceptRejected       = "quest.accept_rejected"
	eventQuestProgressUpdated      = "quest.progress_updated"
	eventQuestObjectiveCompleted   = "quest.objective_completed"
	eventQuestReadyToComplete      = "quest.ready_to_complete"
	eventQuestCompleteRequested    = "quest.complete_requested"
	eventQuestCompleted            = "quest.completed"
	eventQuestCompleteRejected     = "quest.complete_rejected"
	eventQuestRewardGranted        = "quest.reward_granted"
	eventQuestPersisted            = "quest.persisted"
	eventNPCSpawnPointLoaded       = EventNPCSpawnPointLoaded
	eventNPCSpawned                = EventNPCSpawned
	eventNPCDied                   = EventNPCDied
	eventNPCRespawnScheduled       = EventNPCRespawnScheduled
	eventNPCRespawned              = EventNPCRespawned
	eventNPCAggroStarted           = EventNPCAggroStarted
	eventNPCAggroLost              = EventNPCAggroLost
	eventNPCAttackResolved         = EventNPCAttackResolved
	eventNPCLeashReset             = EventNPCLeashReset
	eventNPCTargetLost             = EventNPCTargetLost
	eventCombatTargetSelected      = EventCombatTargetSelected
	eventCombatTargetRejected      = EventCombatTargetRejected
	eventCombatIntentSubmitted     = EventCombatIntentSubmitted
	eventCombatAbilityRejected     = EventCombatAbilityRejected
	eventCombatAbilityResolved     = EventCombatAbilityResolved
	eventCombatDamageApplied       = EventCombatDamageApplied
	eventCombatCooldownStarted     = EventCombatCooldownStarted
	eventCombatTargetDefeated      = EventCombatTargetDefeated
	eventEntityHealthChanged       = EventEntityHealthChanged
	eventEntityDied                = EventEntityDied
	eventPlayerDied                = EventPlayerDied
	eventWorldEntitySpawned        = EventWorldEntitySpawned
	eventWorldEntityRemoved        = EventWorldEntityRemoved
	eventWorldStateDiffEmitted     = EventWorldStateDiffEmitted
	eventLoadsimCombatStarted      = EventLoadsimCombatStarted
	eventLoadsimCombatCompleted    = EventLoadsimCombatCompleted
)

const (
	DiffEntitySpawn             = "EntitySpawnDelta"
	DiffEntityHealth            = "EntityHealthDelta"
	DiffEntityCombatState       = "EntityCombatStateDelta"
	DiffTargetSelection         = "TargetSelectionDelta"
	DiffAbilityResult           = "AbilityResultDelta"
	DiffEntityDeath             = "EntityDeathDelta"
	DiffProgression             = "ProgressionDelta"
	diffEntitySpawn             = DiffEntitySpawn
	diffEntityHealth            = DiffEntityHealth
	diffEntityCombatState       = DiffEntityCombatState
	diffTargetSelection         = DiffTargetSelection
	diffAbilityResult           = DiffAbilityResult
	diffEntityDeath             = DiffEntityDeath
	diffProgression             = DiffProgression
	diffInventoryDelta          = "InventoryDelta"
	diffLootContainerCreated    = "LootContainerCreatedDelta"
	diffLootContainerUpdated    = "LootContainerUpdatedDelta"
	diffLootClaimResult         = "LootClaimResultDelta"
	diffQuestAccepted           = "QuestAcceptedDelta"
	diffQuestProgress           = "QuestProgressDelta"
	diffQuestObjectiveCompleted = "QuestObjectiveCompletedDelta"
	diffQuestReady              = "QuestReadyDelta"
	diffQuestCompleted          = "QuestCompletedDelta"
	diffQuestReward             = "QuestRewardDelta"

	maxRecentDomainEvents = 256
	maxRecentStateDiffs   = 256
)

type domainEvent = DomainEvent
type stateDiff = StateDiff

type DomainEvent struct {
	Sequence     int64          `json:"sequence,omitempty"`
	Name         string         `json:"name,omitempty"`
	EventName    string         `json:"eventName,omitempty"`
	Type         string         `json:"type"`
	AtUnixMs     int64          `json:"atUnixMs,omitempty"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	CharacterID  string         `json:"characterId,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	ZoneID       string         `json:"zoneId,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

type StateDiff struct {
	Sequence     int64          `json:"sequence,omitempty"`
	DiffType     string         `json:"diffType,omitempty"`
	Type         string         `json:"type"`
	AtUnixMs     int64          `json:"atUnixMs,omitempty"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	CharacterID  string         `json:"characterId,omitempty"`
	EntityID     string         `json:"entityId,omitempty"`
	ZoneID       string         `json:"zoneId,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

func newDomainEvent(eventName string, args ...any) DomainEvent {
	fields := map[string]any{}
	characterID := ""
	entityID := ""
	zoneID := ""
	if len(args) == 1 {
		if typed, ok := args[0].(map[string]any); ok {
			fields = typed
		}
	}
	if len(args) == 4 {
		characterID, _ = args[0].(string)
		entityID, _ = args[1].(string)
		zoneID, _ = args[2].(string)
		if typed, ok := args[3].(map[string]any); ok {
			fields = typed
		}
	}
	nowMs := nowMillis()
	cloned := cloneEventFields(fields)
	if characterID != "" {
		cloned["characterId"] = characterID
	}
	if entityID != "" {
		cloned["entityId"] = entityID
	}
	if zoneID != "" {
		cloned["zoneId"] = zoneID
	}
	return DomainEvent{
		Name:         eventName,
		EventName:    eventName,
		Type:         eventName,
		AtUnixMs:     nowMs,
		OccurredAtMs: nowMs,
		CharacterID:  characterID,
		EntityID:     entityID,
		ZoneID:       zoneID,
		Payload:      cloneEventFields(cloned),
		Fields:       cloned,
	}
}

func newStateDiff(diffType string, args ...any) StateDiff {
	characterID := ""
	entityID := ""
	zoneID := ""
	fields := map[string]any{}
	if len(args) == 2 {
		entityID, _ = args[0].(string)
		if typed, ok := args[1].(map[string]any); ok {
			fields = typed
		}
	}
	if len(args) == 4 {
		characterID, _ = args[0].(string)
		entityID, _ = args[1].(string)
		zoneID, _ = args[2].(string)
		if typed, ok := args[3].(map[string]any); ok {
			fields = typed
		}
	}
	nowMs := nowMillis()
	cloned := cloneEventFields(fields)
	return StateDiff{
		DiffType:     diffType,
		Type:         diffType,
		AtUnixMs:     nowMs,
		OccurredAtMs: nowMs,
		CharacterID:  characterID,
		EntityID:     entityID,
		ZoneID:       zoneID,
		Payload:      cloneEventFields(cloned),
		Fields:       cloned,
	}
}

func (s *worldServer) emitWorldEventLocked(eventName string, fields map[string]any, diffs ...StateDiff) {
	if eventName == "" {
		return
	}
	nowMs := nowMillis()
	eventFields := cloneEventFields(fields)
	s.eventSequence++
	s.domainEvents = appendCappedDomainEvent(s.domainEvents, DomainEvent{
		Sequence:     s.eventSequence,
		Name:         eventName,
		EventName:    eventName,
		Type:         eventName,
		AtUnixMs:     nowMs,
		OccurredAtMs: nowMs,
		Payload:      cloneEventFields(eventFields),
		Fields:       eventFields,
	}, maxRecentDomainEvents)
	observability.LogEvent("world-service", eventName, eventFields)

	for _, diff := range diffs {
		diffType := diff.Type
		if diffType == "" {
			diffType = diff.DiffType
		}
		if diffType == "" {
			continue
		}
		if diff.Type == "" {
			diff.Type = diffType
		}
		if diff.DiffType == "" {
			diff.DiffType = diffType
		}
		if diff.OccurredAtMs == 0 {
			diff.OccurredAtMs = nowMs
		}
		if diff.AtUnixMs == 0 {
			diff.AtUnixMs = diff.OccurredAtMs
		}
		diff.Fields = cloneEventFields(diff.Fields)
		diff.Payload = cloneEventFields(eventPayload(diff))
		s.eventSequence++
		diff.Sequence = s.eventSequence
		s.stateDiffs = appendCappedStateDiff(s.stateDiffs, diff, maxRecentStateDiffs)
		diffFields := map[string]any{
			"diffType": diffType,
			"entityId": diff.EntityID,
		}
		if diff.CharacterID != "" {
			diffFields["characterId"] = diff.CharacterID
		}
		if diff.ZoneID != "" {
			diffFields["zoneId"] = diff.ZoneID
		}
		s.emitDomainEventLocked(EventWorldStateDiffEmitted, diffFields)
	}
}

func (s *worldServer) emitDomainEventLocked(name string, payload map[string]any) DomainEvent {
	if payload == nil {
		payload = map[string]any{}
	}
	s.eventSequence++
	nowMs := nowMillis()
	fields := cloneEventFields(payload)
	event := DomainEvent{
		Sequence:     s.eventSequence,
		Name:         name,
		EventName:    name,
		Type:         name,
		AtUnixMs:     nowMs,
		OccurredAtMs: nowMs,
		Payload:      cloneEventFields(fields),
		Fields:       fields,
	}
	s.domainEvents = appendCappedDomainEvent(s.domainEvents, event, maxRecentDomainEvents)
	observability.LogEvent("world-service", name, fields)
	return event
}

func (s *worldServer) emitStateDiffLocked(diffType string, entityID string, payload map[string]any) StateDiff {
	if payload == nil {
		payload = map[string]any{}
	}
	s.eventSequence++
	nowMs := nowMillis()
	fields := cloneEventFields(payload)
	diff := StateDiff{
		Sequence:     s.eventSequence,
		DiffType:     diffType,
		Type:         diffType,
		EntityID:     entityID,
		AtUnixMs:     nowMs,
		OccurredAtMs: nowMs,
		Payload:      cloneEventFields(fields),
		Fields:       fields,
	}
	s.stateDiffs = appendCappedStateDiff(s.stateDiffs, diff, maxRecentStateDiffs)
	s.emitDomainEventLocked(eventWorldStateDiffEmitted, map[string]any{
		"diffType": diffType,
		"entityId": entityID,
		"sequence": diff.Sequence,
	})
	return diff
}

func entitySpawnDelta(mob *mobState) StateDiff {
	if mob == nil {
		return StateDiff{}
	}
	return newStateDiff(DiffEntitySpawn, mob.ID, map[string]any{
		"archetypeId":  mob.ArchetypeID,
		"displayName":  mob.DisplayName,
		"kind":         mob.Kind,
		"zoneId":       mob.ZoneID,
		"level":        mob.Level,
		"x":            mob.X,
		"y":            mob.Y,
		"z":            mob.Z,
		"health":       mob.Health,
		"maxHealth":    mob.MaxHealth,
		"alive":        mob.Alive,
		"targetable":   mob.Targetable,
		"disposition":  mob.Disposition,
		"spawnPointId": mob.SpawnPointID,
	})
}

func entityHealthDelta(entityID string, health float64, maxHealth float64, alive bool) StateDiff {
	return newStateDiff(DiffEntityHealth, entityID, map[string]any{
		"health":    health,
		"maxHealth": maxHealth,
		"alive":     alive,
	})
}

func entityCombatStateDelta(entityID string, inCombat bool, targetID string) StateDiff {
	return newStateDiff(DiffEntityCombatState, entityID, map[string]any{
		"isInCombat":            inCombat,
		"currentTargetEntityId": targetID,
	})
}

func targetSelectionDelta(characterID string, targetID string) StateDiff {
	return newStateDiff(DiffTargetSelection, characterID, map[string]any{
		"targetId": targetID,
	})
}

func abilityResultDelta(sourceID string, targetID string, abilityID string, damage float64, accepted bool) StateDiff {
	return newStateDiff(DiffAbilityResult, targetID, map[string]any{
		"sourceEntityId": sourceID,
		"targetEntityId": targetID,
		"abilityId":      abilityID,
		"damage":         damage,
		"accepted":       accepted,
	})
}

func entityDeathDelta(entityID string, killedBy string, respawnAtMs int64) StateDiff {
	return newStateDiff(DiffEntityDeath, entityID, map[string]any{
		"killedByEntityId": killedBy,
		"respawnAtMs":      respawnAtMs,
	})
}

func progressionDelta(characterID string, archetypeID string, count int) StateDiff {
	return newStateDiff(DiffProgression, characterID, map[string]any{
		"archetypeId": archetypeID,
		"killCount":   count,
	})
}

func questDelta(characterID string, questID string, diffType string, fields map[string]any) StateDiff {
	if fields == nil {
		fields = map[string]any{}
	}
	fields = cloneEventFields(fields)
	fields["questId"] = questID
	return newStateDiff(diffType, characterID, fields)
}

func cloneDomainEvents(source []DomainEvent) []DomainEvent {
	if len(source) == 0 {
		return []DomainEvent{}
	}
	cloned := make([]DomainEvent, len(source))
	for index, event := range source {
		cloned[index] = DomainEvent{
			Sequence:     event.Sequence,
			Name:         event.Name,
			EventName:    event.EventName,
			Type:         event.Type,
			AtUnixMs:     event.AtUnixMs,
			OccurredAtMs: event.OccurredAtMs,
			CharacterID:  event.CharacterID,
			EntityID:     event.EntityID,
			ZoneID:       event.ZoneID,
			Payload:      cloneEventFields(event.Payload),
			Fields:       cloneEventFields(event.Fields),
		}
	}
	return cloned
}

func cloneStateDiffs(source []StateDiff) []StateDiff {
	if len(source) == 0 {
		return []StateDiff{}
	}
	cloned := make([]StateDiff, len(source))
	for index, diff := range source {
		cloned[index] = StateDiff{
			Sequence:     diff.Sequence,
			DiffType:     diff.DiffType,
			Type:         diff.Type,
			AtUnixMs:     diff.AtUnixMs,
			OccurredAtMs: diff.OccurredAtMs,
			CharacterID:  diff.CharacterID,
			EntityID:     diff.EntityID,
			ZoneID:       diff.ZoneID,
			Payload:      cloneEventFields(diff.Payload),
			Fields:       cloneEventFields(diff.Fields),
		}
	}
	return cloned
}

func cloneEventFields(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneAnyMap(source map[string]any) map[string]any {
	return cloneEventFields(source)
}

func eventPayload(diff StateDiff) map[string]any {
	if len(diff.Payload) > 0 {
		return diff.Payload
	}
	return diff.Fields
}

func appendCappedDomainEvent(source []DomainEvent, event DomainEvent, limits ...int) []DomainEvent {
	limit := maxRecentDomainEvents
	if len(limits) > 0 {
		limit = limits[0]
	}
	source = append(source, event)
	if limit <= 0 || len(source) <= limit {
		return source
	}
	return append([]DomainEvent(nil), source[len(source)-limit:]...)
}

func appendCappedStateDiff(source []StateDiff, diff StateDiff, limits ...int) []StateDiff {
	limit := maxRecentStateDiffs
	if len(limits) > 0 {
		limit = limits[0]
	}
	source = append(source, diff)
	if limit <= 0 || len(source) <= limit {
		return source
	}
	return append([]StateDiff(nil), source[len(source)-limit:]...)
}

func (s *worldServer) recentDomainEventsLocked() []DomainEvent {
	return cloneDomainEvents(s.domainEvents)
}

func (s *worldServer) recentStateDiffsLocked() []StateDiff {
	return cloneStateDiffs(s.stateDiffs)
}

func eventTimeFromMs(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

package worlds

import (
	"time"

	"amandacore/services/internal/observability"
)

const (
	EventNPCSpawnPointLoaded           = "npc.spawn_point.loaded"
	EventNPCSpawned                    = "npc.spawned"
	EventNPCDied                       = "npc.died"
	EventNPCRespawnScheduled           = "npc.respawn_scheduled"
	EventNPCRespawned                  = "npc.respawned"
	EventNPCAggroStarted               = "npc.aggro_started"
	EventNPCAggroLost                  = "npc.aggro_lost"
	EventNPCAttackResolved             = "npc.attack_resolved"
	EventNPCLeashReset                 = "npc.leash_reset"
	EventNPCTargetLost                 = "npc.target_lost"
	EventCombatTargetSelected          = "combat.target.selected"
	EventCombatTargetRejected          = "combat.target.rejected"
	EventCombatIntentSubmitted         = "combat.intent_submitted"
	EventCombatAbilityRejected         = "combat.ability_rejected"
	EventCombatAbilityResolved         = "combat.ability_resolved"
	EventCombatDamageApplied           = "combat.damage_applied"
	EventCombatCooldownStarted         = "combat.cooldown_started"
	EventCombatTargetDefeated          = "combat.target_defeated"
	EventEntityHealthChanged           = "entity.health_changed"
	EventEntityDied                    = "entity.died"
	EventPlayerDied                    = "player.died"
	EventProgressionCreditAwarded      = "progression.kill_credit_awarded"
	EventProgressionCreditSaved        = "progression.kill_credit_persisted"
	EventWorldEntitySpawned            = "world.entity.spawned"
	EventWorldEntityRemoved            = "world.entity.removed"
	EventWorldStateDiffEmitted         = "world.state.diff_emitted"
	EventShardWorkerStateChanged       = "shard.worker.state_changed"
	EventShardCoordinatorRejected      = "shard.coordinator.rejected"
	EventZoneHandoffRequested          = "zone.handoff.requested"
	EventZoneHandoffAccepted           = "zone.handoff.accepted"
	EventZoneHandoffCompleted          = "zone.handoff.completed"
	EventZoneHandoffRejected           = "zone.handoff.rejected"
	EventZoneHandoffRetryScheduled     = "zone.handoff.retry_scheduled"
	EventZoneHandoffReconnectCorrected = "zone.handoff.reconnect_corrected"
	EventZoneQueueBackpressure         = "zone.queue.backpressure"
	EventLoadsimQuestStarted           = "loadsim.quest.started"
	EventLoadsimQuestCompleted         = "loadsim.quest.completed"
	EventLoadsimZoneHandoffStarted     = "loadsim.zone_handoff.started"
	EventLoadsimZoneHandoffCompleted   = "loadsim.zone_handoff.completed"
	eventItemCatalogLoaded             = "item.catalog.loaded"
	eventInventoryGrantRequested       = "inventory.item_grant_requested"
	eventInventoryItemGranted          = "inventory.item_granted"
	eventInventoryGrantRejected        = "inventory.item_grant_rejected"
	eventInventoryStackUpdated         = "inventory.stack_updated"
	eventInventoryFull                 = "inventory.full"
	eventInventoryPersisted            = "inventory.persisted"
	eventLootTableLoaded               = "loot.table.loaded"
	eventLootRollStarted               = "loot.roll.started"
	eventLootRollCompleted             = "loot.roll.completed"
	eventLootContainerCreated          = "loot.container_created"
	eventLootInspectRequested          = "loot.inspect_requested"
	eventLootInspectResolved           = "loot.inspect_resolved"
	eventLootClaimRequested            = "loot.claim_requested"
	eventLootClaimCompleted            = "loot.claim_completed"
	eventLootClaimRejected             = "loot.claim_rejected"
	eventLootContainerExpired          = "loot.container_expired"
	eventProgressionCreditAwarded      = EventProgressionCreditAwarded
	eventProgressionCreditRejected     = "progression.kill_credit_rejected"
	eventProgressionCreditSaved        = EventProgressionCreditSaved
	eventQuestCatalogLoaded            = "quest.catalog.loaded"
	eventQuestAcceptRequested          = "quest.accept_requested"
	eventQuestAccepted                 = "quest.accepted"
	eventQuestAcceptRejected           = "quest.accept_rejected"
	eventQuestProgressUpdated          = "quest.progress_updated"
	eventQuestObjectiveCompleted       = "quest.objective_completed"
	eventQuestReadyToComplete          = "quest.ready_to_complete"
	eventQuestCompleteRequested        = "quest.complete_requested"
	eventQuestCompleted                = "quest.completed"
	eventQuestCompleteRejected         = "quest.complete_rejected"
	eventQuestRewardGranted            = "quest.reward_granted"
	eventQuestPersisted                = "quest.persisted"
	eventNPCSpawnPointLoaded           = EventNPCSpawnPointLoaded
	eventNPCSpawned                    = EventNPCSpawned
	eventNPCDied                       = EventNPCDied
	eventNPCRespawnScheduled           = EventNPCRespawnScheduled
	eventNPCRespawned                  = EventNPCRespawned
	eventNPCAggroStarted               = EventNPCAggroStarted
	eventNPCAggroLost                  = EventNPCAggroLost
	eventNPCAttackResolved             = EventNPCAttackResolved
	eventNPCLeashReset                 = EventNPCLeashReset
	eventNPCTargetLost                 = EventNPCTargetLost
	eventCombatTargetSelected          = EventCombatTargetSelected
	eventCombatTargetRejected          = EventCombatTargetRejected
	eventCombatIntentSubmitted         = EventCombatIntentSubmitted
	eventCombatAbilityRejected         = EventCombatAbilityRejected
	eventCombatAbilityResolved         = EventCombatAbilityResolved
	eventCombatDamageApplied           = EventCombatDamageApplied
	eventCombatCooldownStarted         = EventCombatCooldownStarted
	eventCombatTargetDefeated          = EventCombatTargetDefeated
	eventEntityHealthChanged           = EventEntityHealthChanged
	eventEntityDied                    = EventEntityDied
	eventPlayerDied                    = EventPlayerDied
	eventWorldEntitySpawned            = EventWorldEntitySpawned
	eventWorldEntityRemoved            = EventWorldEntityRemoved
	eventWorldStateDiffEmitted         = EventWorldStateDiffEmitted
	eventShardWorkerStateChanged       = EventShardWorkerStateChanged
	eventShardCoordinatorRejected      = EventShardCoordinatorRejected
	eventZoneHandoffRequested          = EventZoneHandoffRequested
	eventZoneHandoffAccepted           = EventZoneHandoffAccepted
	eventZoneHandoffCompleted          = EventZoneHandoffCompleted
	eventZoneHandoffRejected           = EventZoneHandoffRejected
	eventZoneHandoffRetryScheduled     = EventZoneHandoffRetryScheduled
	eventZoneHandoffReconnectCorrected = EventZoneHandoffReconnectCorrected
	eventZoneQueueBackpressure         = EventZoneQueueBackpressure
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
	diffZoneHandoff             = "ZoneHandoffDelta"

	maxRecentDomainEvents = 256
	maxRecentStateDiffs   = 256
)

type domainEvent = DomainEvent
type stateDiff = StateDiff

type DomainEvent struct {
	Type         string         `json:"type"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	Fields       map[string]any `json:"fields,omitempty"`
}

type StateDiff struct {
	Type         string         `json:"type"`
	OccurredAtMs int64          `json:"occurredAtMs"`
	EntityID     string         `json:"entityId,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

func newDomainEvent(eventName string, fields map[string]any) DomainEvent {
	return DomainEvent{
		Type:         eventName,
		OccurredAtMs: nowMillis(),
		Fields:       cloneEventFields(fields),
	}
}

func newStateDiff(diffType string, entityID string, fields map[string]any) StateDiff {
	return StateDiff{
		Type:         diffType,
		OccurredAtMs: nowMillis(),
		EntityID:     entityID,
		Fields:       cloneEventFields(fields),
	}
}

func (s *worldServer) emitWorldEventLocked(eventName string, fields map[string]any, diffs ...StateDiff) {
	if eventName == "" {
		return
	}
	nowMs := nowMillis()
	eventFields := cloneEventFields(fields)
	s.domainEvents = appendCappedDomainEvent(s.domainEvents, DomainEvent{
		Type:         eventName,
		OccurredAtMs: nowMs,
		Fields:       eventFields,
	}, maxRecentDomainEvents)
	observability.LogEvent("world-service", eventName, eventFields)

	for _, diff := range diffs {
		if diff.Type == "" {
			continue
		}
		if diff.OccurredAtMs == 0 {
			diff.OccurredAtMs = nowMs
		}
		diff.Fields = cloneEventFields(diff.Fields)
		s.stateDiffs = appendCappedStateDiff(s.stateDiffs, diff, maxRecentStateDiffs)
		diffFields := map[string]any{
			"diffType": diff.Type,
			"entityId": diff.EntityID,
		}
		s.domainEvents = appendCappedDomainEvent(s.domainEvents, DomainEvent{
			Type:         EventWorldStateDiffEmitted,
			OccurredAtMs: nowMs,
			Fields:       diffFields,
		}, maxRecentDomainEvents)
		observability.LogEvent("world-service", EventWorldStateDiffEmitted, diffFields)
	}
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
			Type:         event.Type,
			OccurredAtMs: event.OccurredAtMs,
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
			Type:         diff.Type,
			OccurredAtMs: diff.OccurredAtMs,
			EntityID:     diff.EntityID,
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

func appendCappedDomainEvent(source []DomainEvent, event DomainEvent, limit int) []DomainEvent {
	source = append(source, event)
	if limit <= 0 || len(source) <= limit {
		return source
	}
	return append([]DomainEvent(nil), source[len(source)-limit:]...)
}

func appendCappedStateDiff(source []StateDiff, diff StateDiff, limit int) []StateDiff {
	source = append(source, diff)
	if limit <= 0 || len(source) <= limit {
		return source
	}
	return append([]StateDiff(nil), source[len(source)-limit:]...)
}

func eventTimeFromMs(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

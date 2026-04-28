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

	EventWorldTickStarted             = "world.tick.started"
	EventWorldTickCompleted           = "world.tick.completed"
	EventWorldTickSlow                = "world.tick.slow"
	EventWorldCommandEnqueued         = "world.command.enqueued"
	EventWorldCommandRejected         = "world.command.rejected"
	EventWorldLoopStarted             = "world.loop_started"
	EventWorldLoopStopped             = "world.loop_stopped"
	EventWorldCommandAccepted         = "world.command_accepted"
	EventWorldCommandApplied          = "world.command_applied"
	EventWorldSnapshotEmitted         = "world.snapshot_emitted"
	EventWorldReplayRecorded          = "world.replay_recorded"
	EventWorldReconnectRestored       = "world.reconnect_restored"
	EventWorldCommandTimeout          = "world.command_timeout"
	EventWorldZoneLoaded              = "world.zone.loaded"
	EventWorldZoneUnloaded            = "world.zone.unloaded"
	EventWorldEntitySpawned           = "world.entity.spawned"
	EventWorldEntityDespawned         = "world.entity.despawned"
	EventCombatIntentSubmitted        = "combat.intent_submitted"
	EventCombatTargetSelected         = "combat.target_selected"
	EventCombatAutoAttackStarted      = "combat.auto_attack_started"
	EventCombatAbilityResolved        = "combat.ability_resolved"
	EventCombatDamageApplied          = "combat.damage_applied"
	EventCombatEntityDefeated         = "combat.entity_defeated"
	EventThreatUpdated                = "threat.updated"
	EventQuestObjectiveProgressed     = "quest.objective_progressed"
	EventQuestCompleted               = "quest.completed"
	EventQuestRewardClaimed           = "quest.reward_claimed"
	EventLootGenerated                = "loot.generated"
	EventLootClaimed                  = "loot.claimed"
	EventLootClaimRejected            = "loot.claim_rejected"
	EventWorldGameplayCommandApplied  = "world.loop_gameplay_command_applied"
	EventWorldGameplayCommandRejected = "world.loop_gameplay_command_rejected"
	EventReplicationSnapshotEmitted   = "replication.snapshot_emitted"
	EventReplicationDeltaEmitted      = "replication.delta_emitted"
	EventReplicationCursorAccepted    = "replication.cursor_accepted"
	EventReplicationCursorStale       = "replication.cursor_stale"
	EventReplicationResyncRequired    = "replication.resync_required"
	EventReplicationClientConverged   = "replication.client_converged"
	EventReplicationClientDiverged    = "replication.client_diverged"
	EventReplicationFrameDropped      = "replication.frame_dropped"
	EventSocialFriendAdded            = "social.friend_added"
	EventSocialFriendRemoved          = "social.friend_removed"
	EventSocialPartyInviteCreated     = "social.party_invite_created"
	EventSocialPartyInviteAccepted    = "social.party_invite_accepted"
	EventSocialPartyMemberRemoved     = "social.party_member_removed"
	EventSocialGuildCreated           = "social.guild_created"
	EventSocialGuildInviteCreated     = "social.guild_invite_created"
	EventSocialGuildInviteAccepted    = "social.guild_invite_accepted"
	EventSocialChatMessageSent        = "social.chat_message_sent"
	EventEconomyCurrencyMutated       = "economy.currency_mutated"
	EventEconomyVendorPurchase        = "economy.vendor_purchase_completed"
	EventEconomyVendorSale            = "economy.vendor_sale_completed"
	EventEconomyAuctionListed         = "economy.auction_listed"
	EventEconomyAuctionBought         = "economy.auction_bought"
	EventEconomyAuctionCancelled      = "economy.auction_cancelled"
	EventEconomyMailCreated           = "economy.mail_created"
	EventEconomyMailClaimed           = "economy.mail_claimed"
	EventEconomyDuplicateRejected     = "economy.duplicate_mutation_rejected"
	EventAbilityCastStarted           = "ability.cast_started"
	EventAbilityCastCompleted         = "ability.cast_completed"
	EventAbilityCastInterrupted       = "ability.cast_interrupted"
	EventAbilityEffectResolved        = "ability.effect_resolved"
	EventAuraApplied                  = "aura.applied"
	EventAuraRefreshed                = "aura.refreshed"
	EventAuraTicked                   = "aura.ticked"
	EventAuraExpired                  = "aura.expired"
	EventCooldownStarted              = "cooldown.started"
	EventCooldownReady                = "cooldown.ready"
	EventNPCSpawned                   = "npc.spawned"
	EventAdminActionRequested         = "admin.action_requested"
	EventAdminActionApplied           = "admin.action_applied"
	EventPersistenceSnapshotSaved     = "persistence.snapshot_saved"
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
		EventWorldTickStarted,
		EventWorldTickCompleted,
		EventWorldTickSlow,
		EventWorldCommandEnqueued,
		EventWorldCommandRejected,
		EventWorldLoopStarted,
		EventWorldLoopStopped,
		EventWorldCommandAccepted,
		EventWorldCommandApplied,
		EventWorldSnapshotEmitted,
		EventWorldReplayRecorded,
		EventWorldReconnectRestored,
		EventWorldCommandTimeout,
		EventWorldZoneLoaded,
		EventWorldZoneUnloaded,
		EventWorldEntitySpawned,
		EventWorldEntityDespawned,
		EventCombatIntentSubmitted,
		EventCombatTargetSelected,
		EventCombatAutoAttackStarted,
		EventCombatAbilityResolved,
		EventCombatDamageApplied,
		EventCombatEntityDefeated,
		EventThreatUpdated,
		EventQuestObjectiveProgressed,
		EventQuestCompleted,
		EventQuestRewardClaimed,
		EventLootGenerated,
		EventLootClaimed,
		EventLootClaimRejected,
		EventWorldGameplayCommandApplied,
		EventWorldGameplayCommandRejected,
		EventReplicationSnapshotEmitted,
		EventReplicationDeltaEmitted,
		EventReplicationCursorAccepted,
		EventReplicationCursorStale,
		EventReplicationResyncRequired,
		EventReplicationClientConverged,
		EventReplicationClientDiverged,
		EventReplicationFrameDropped,
		EventSocialFriendAdded,
		EventSocialFriendRemoved,
		EventSocialPartyInviteCreated,
		EventSocialPartyInviteAccepted,
		EventSocialPartyMemberRemoved,
		EventSocialGuildCreated,
		EventSocialGuildInviteCreated,
		EventSocialGuildInviteAccepted,
		EventSocialChatMessageSent,
		EventEconomyCurrencyMutated,
		EventEconomyVendorPurchase,
		EventEconomyVendorSale,
		EventEconomyAuctionListed,
		EventEconomyAuctionBought,
		EventEconomyAuctionCancelled,
		EventEconomyMailCreated,
		EventEconomyMailClaimed,
		EventEconomyDuplicateRejected,
		EventAbilityCastStarted,
		EventAbilityCastCompleted,
		EventAbilityCastInterrupted,
		EventAbilityEffectResolved,
		EventAuraApplied,
		EventAuraRefreshed,
		EventAuraTicked,
		EventAuraExpired,
		EventCooldownStarted,
		EventCooldownReady,
		EventNPCSpawned,
		EventAdminActionRequested,
		EventAdminActionApplied,
		EventPersistenceSnapshotSaved,
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

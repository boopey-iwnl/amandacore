package worlds

import (
	"strings"
	"testing"

	"amandacore/services/internal/platform"
)

func TestDevHostileNPCSpawnsFromSpawnPoint(t *testing.T) {
	server := newWorldServer(nil)
	mob := server.mobs[mobDevIsleStalkerID]
	if mob == nil {
		t.Fatalf("expected dev hostile NPC %s to spawn", mobDevIsleStalkerID)
	}
	if mob.ArchetypeID != mobDevIsleStalkerTypeID {
		t.Fatalf("expected archetype %s, got %s", mobDevIsleStalkerTypeID, mob.ArchetypeID)
	}
	if mob.SpawnPointID != mobDevIsleStalkerSpawn {
		t.Fatalf("expected spawn point %s, got %s", mobDevIsleStalkerSpawn, mob.SpawnPointID)
	}
	if mob.DisplayName != "Isle Stalker" || mob.Level != 1 || mob.MaxHealth != 30 {
		t.Fatalf("unexpected dev NPC state: %#v", mob)
	}
	if mob.Disposition != npcDispositionHostile || !mob.Alive || !mob.Targetable {
		t.Fatalf("expected live hostile targetable NPC, got %#v", mob)
	}
	if !hasStateDiff(server, diffEntitySpawn, mobDevIsleStalkerID) {
		t.Fatalf("expected dev NPC spawn to emit %s", diffEntitySpawn)
	}
}

func TestNPCCombatTargetSelection(t *testing.T) {
	server, mob, session := newNPCCombatLoopTestServer()
	session.X = mob.X - 1
	session.Y = mob.Y

	if err := server.setTargetLocked(session, mob.ID); err != nil {
		t.Fatalf("expected valid hostile target selection: %v", err)
	}
	if session.CurrentTargetID != mob.ID {
		t.Fatalf("expected selected target %s, got %s", mob.ID, session.CurrentTargetID)
	}
	if !hasDomainEvent(server, eventCombatTargetSelected) {
		t.Fatalf("expected target selection event")
	}

	if err := server.setTargetLocked(session, "missing_target"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("expected missing target rejection, got %v", err)
	}
	if !hasDomainEvent(server, eventCombatTargetRejected) {
		t.Fatalf("expected target rejection event")
	}
}

func TestDevBasicStrikeRulesDamageCooldownDeathAndRespawn(t *testing.T) {
	server, mob, session := newNPCCombatLoopTestServer()
	session.X = mob.X - 8
	session.Y = mob.Y
	if err := server.setTargetLocked(session, mob.ID); err != nil {
		t.Fatalf("expected target selection inside target range: %v", err)
	}
	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range ability rejection, got %v", err)
	}

	session.X = mob.X - 1
	session.Y = mob.Y
	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err != nil {
		t.Fatalf("expected Basic Strike to resolve: %v", err)
	}
	if mob.Health != 20 {
		t.Fatalf("expected deterministic 10 damage, got mob health %.1f", mob.Health)
	}
	if !hasStateDiff(server, diffEntityHealth, mob.ID) {
		t.Fatalf("expected health delta")
	}

	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err == nil || !strings.Contains(err.Error(), "cooling down") {
		t.Fatalf("expected ability cooldown rejection, got %v", err)
	}
	session.AbilityCooldowns[platform.DevBasicStrikeAbilityID] = 0
	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err != nil {
		t.Fatalf("expected second Basic Strike after cooldown reset: %v", err)
	}
	session.AbilityCooldowns[platform.DevBasicStrikeAbilityID] = 0
	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err != nil {
		t.Fatalf("expected killing Basic Strike: %v", err)
	}
	if mob.Alive || mob.Targetable || mob.Health != 0 {
		t.Fatalf("expected dead untargetable NPC at zero health, got %#v", mob)
	}
	if session.KillCredits[mobDevIsleStalkerTypeID].Count != 1 {
		t.Fatalf("expected kill credit for %s, got %#v", mobDevIsleStalkerTypeID, session.KillCredits)
	}
	if !hasDomainEvent(server, eventProgressionKillCredit) || !hasStateDiff(server, diffEntityDeath, mob.ID) {
		t.Fatalf("expected kill credit event and death delta")
	}

	session.AbilityCooldowns[platform.DevBasicStrikeAbilityID] = 0
	session.CurrentTargetID = mob.ID
	if err := server.activateAbilityLocked(session, platform.DevBasicStrikeAbilityID); err == nil || !strings.Contains(err.Error(), "target is invalid") {
		t.Fatalf("expected dead target rejection, got %v", err)
	}

	server.advanceMobLocked(mob, 0.25, mob.RespawnAtMs)
	if !mob.Alive || !mob.Targetable || mob.Health != mob.MaxHealth {
		t.Fatalf("expected respawned NPC at full health, got %#v", mob)
	}
	if !hasDomainEvent(server, eventNPCRespawned) {
		t.Fatalf("expected respawn event")
	}
}

func TestHostileNPCAggroAttackAndLeash(t *testing.T) {
	server, mob, session := newNPCCombatLoopTestServer()
	session.X = mob.X + 1
	session.Y = mob.Y
	healthBefore := session.Health

	server.advanceMobLocked(mob, 0.25, mob.AttackCadenceMs)
	if mob.CurrentTargetCharacter != session.CharacterID {
		t.Fatalf("expected hostile NPC to aggro %s, got %s", session.CharacterID, mob.CurrentTargetCharacter)
	}
	if mob.AIState != mobAIStateAttacking {
		t.Fatalf("expected attacking state, got %s", mob.AIState)
	}
	if session.Health >= healthBefore {
		t.Fatalf("expected server-authoritative NPC damage, got %.1f -> %.1f", healthBefore, session.Health)
	}
	if !hasDomainEvent(server, eventNPCAggroStarted) || !hasDomainEvent(server, eventNPCAttackResolved) {
		t.Fatalf("expected aggro and attack events")
	}

	session.X = mob.SpawnX + mob.LeashRadius + 2
	session.Y = mob.SpawnY
	server.advanceMobLocked(mob, 0.25, mob.AttackCadenceMs*2)
	if mob.AIState != mobAIStateEvading {
		t.Fatalf("expected leash reset to enter evading state, got %s", mob.AIState)
	}
	if mob.Targetable {
		t.Fatalf("expected leashed NPC to be temporarily untargetable")
	}
	if !hasDomainEvent(server, eventNPCLeashReset) {
		t.Fatalf("expected leash reset event")
	}
}

func newNPCCombatLoopTestServer() (*worldServer, *mobState, *worldSessionState) {
	server := newWorldServer(nil)
	spawn := mobSpawnDefinition{
		ID:              mobDevIsleStalkerID,
		SpawnPointID:    mobDevIsleStalkerSpawn,
		ZoneID:          defaultZoneID,
		ArchetypeID:     mobDevIsleStalkerTypeID,
		MobTypeID:       mobDevIsleStalkerTypeID,
		DisplayName:     "Isle Stalker",
		Disposition:     npcDispositionHostile,
		Level:           1,
		X:               40,
		Y:               20,
		Z:               playableGroundZ,
		MaxHealth:       30,
		AggroRadius:     8,
		AttackRange:     2.5,
		AttackDamage:    3,
		AttackCadenceMs: 1500,
		MoveSpeedPerSec: 4,
		LeashRadius:     18,
		RespawnDelayMs:  10000,
	}
	mob := newMobStateFromSpawn(spawn, "")
	session := &worldSessionState{
		Token:             "world_npc_combat",
		AccountID:         "acct_npc_combat",
		CharacterID:       "char_npc_combat",
		DisplayName:       "Combat Tester",
		ClassID:           platform.DefaultClassID,
		Level:             1,
		ZoneID:            defaultZoneID,
		Connected:         true,
		Health:            100,
		MaxHealth:         100,
		Resource:          0,
		MaxResource:       100,
		Alive:             true,
		LearnedAbilityIDs: platform.DefaultStartingLearnedAbilityIDs(),
		AbilityCooldowns:  map[string]int64{},
		QuestProgress:     map[string]platform.CharacterQuestProgress{},
		KillCredits:       map[string]platform.CharacterKillCredit{},
	}
	server.mobs = map[string]*mobState{mob.ID: mob}
	server.mobOrder = []string{mob.ID}
	server.sessionsByToken = map[string]*worldSessionState{session.Token: session}
	server.sessionTokenByChar = map[string]string{session.CharacterID: session.Token}
	server.domainEvents = nil
	server.stateDiffs = nil
	server.eventSequence = 0
	return server, mob, session
}

func hasDomainEvent(server *worldServer, name string) bool {
	for _, event := range server.domainEvents {
		if event.Name == name {
			return true
		}
	}
	return false
}

func hasStateDiff(server *worldServer, diffType string, entityID string) bool {
	for _, diff := range server.stateDiffs {
		if diff.Type == diffType && diff.EntityID == entityID {
			return true
		}
	}
	return false
}

package worlds

import "testing"

func TestMobAIStateTransitionsForAggroChaseAndAttack(t *testing.T) {
	server, mob, session := newAITestServer()
	session.X = 6.0
	session.Y = 0.0

	server.advanceMobLocked(mob, 0.25, 1_000)
	if mob.AIState != mobAIStateChasing {
		t.Fatalf("expected mob to chase after aggro, got %s", mob.AIState)
	}
	if mob.CurrentTargetCharacter != session.CharacterID {
		t.Fatalf("expected mob target %s, got %s", session.CharacterID, mob.CurrentTargetCharacter)
	}
	if mob.X <= mob.SpawnX {
		t.Fatalf("expected mob to move toward target, got x %.2f", mob.X)
	}

	session.X = mob.X + 1.0
	healthBefore := session.Health
	server.advanceMobLocked(mob, 0.25, 2_000)
	if mob.AIState != mobAIStateAttacking {
		t.Fatalf("expected mob to attack in range, got %s", mob.AIState)
	}
	if session.Health >= healthBefore {
		t.Fatalf("expected mob attack to damage player, health %.1f -> %.1f", healthBefore, session.Health)
	}
}

func TestMobEvadesAndReturnsHomeAfterLeash(t *testing.T) {
	server, mob, session := newAITestServer()
	mob.X = 5.0
	mob.Health = 40.0
	session.X = 12.0
	session.Y = 0.0

	server.advanceMobLocked(mob, 0.25, 1_000)
	if mob.AIState != mobAIStateEvading {
		t.Fatalf("expected leash break to enter evading, got %s", mob.AIState)
	}
	if mob.Targetable {
		t.Fatalf("expected evading mob to become untargetable")
	}
	if mob.CurrentTargetCharacter != "" {
		t.Fatalf("expected evading mob to clear target, got %s", mob.CurrentTargetCharacter)
	}

	server.advanceMobLocked(mob, 0.25, 1_250)
	if mob.AIState != mobAIStateReturning {
		t.Fatalf("expected evading mob to start returning, got %s", mob.AIState)
	}

	for step := 0; step < 8 && mob.AIState != mobAIStateIdle; step++ {
		server.advanceMobLocked(mob, 0.50, 1_500+int64(step*500))
	}
	if mob.AIState != mobAIStateIdle {
		t.Fatalf("expected mob to finish returning home, got %s at %.2f,%.2f", mob.AIState, mob.X, mob.Y)
	}
	if !mob.Targetable {
		t.Fatalf("expected returned mob to become targetable")
	}
	if mob.Health != mob.MaxHealth {
		t.Fatalf("expected returned mob health to reset, got %.1f/%.1f", mob.Health, mob.MaxHealth)
	}
	if mob.X != mob.SpawnX || mob.Y != mob.SpawnY {
		t.Fatalf("expected returned mob to snap home, got %.2f,%.2f want %.2f,%.2f", mob.X, mob.Y, mob.SpawnX, mob.SpawnY)
	}
}

func TestMobDeathAndRespawnUseExplicitStates(t *testing.T) {
	server, mob, session := newAITestServer()
	mob.Health = 10.0
	session.CurrentTargetID = mob.ID

	if err := server.applyDamageToMobLocked(session, mob, 10.0, "test_damage"); err != nil {
		t.Fatalf("failed to damage mob: %v", err)
	}
	if mob.Alive {
		t.Fatalf("expected mob to be dead")
	}
	if mob.Targetable {
		t.Fatalf("expected dead mob to be untargetable")
	}
	if mob.AIState != mobAIStateDead {
		t.Fatalf("expected dead state, got %s", mob.AIState)
	}
	if mob.RespawnAtMs == 0 {
		t.Fatalf("expected respawn timestamp to be set")
	}
	if session.CurrentTargetID != "" {
		t.Fatalf("expected death to clear player target, got %s", session.CurrentTargetID)
	}

	server.advanceMobLocked(mob, 0.25, mob.RespawnAtMs)
	if !mob.Alive || !mob.Targetable {
		t.Fatalf("expected respawned mob to be alive and targetable")
	}
	if mob.AIState != mobAIStateIdle {
		t.Fatalf("expected respawn to finish in idle, got %s", mob.AIState)
	}
	if mob.Health != mob.MaxHealth {
		t.Fatalf("expected respawned mob health reset, got %.1f/%.1f", mob.Health, mob.MaxHealth)
	}
}

func TestClearMobAggroForCharacterUsesStateModel(t *testing.T) {
	server, mob, session := newAITestServer()
	mob.AIState = mobAIStateChasing
	mob.CurrentTargetCharacter = session.CharacterID

	server.clearMobAggroForCharacterLocked(session.CharacterID)
	if mob.CurrentTargetCharacter != "" {
		t.Fatalf("expected cleared target, got %s", mob.CurrentTargetCharacter)
	}
	if mob.AIState != mobAIStateIdle {
		t.Fatalf("expected cleared chasing mob to idle, got %s", mob.AIState)
	}

	mob.AIState = mobAIStateReturning
	mob.CurrentTargetCharacter = session.CharacterID
	server.clearMobAggroForCharacterLocked(session.CharacterID)
	if mob.AIState != mobAIStateReturning {
		t.Fatalf("expected returning mob to keep return state, got %s", mob.AIState)
	}
}

func newAITestServer() (*worldServer, *mobState, *worldSessionState) {
	server := newWorldServer(nil)
	mob := &mobState{
		ID:              "mob_ai_test_01",
		MobTypeID:       "ai_test",
		DisplayName:     "AI Test Mob",
		Kind:            hostileMobKind,
		ZoneID:          defaultZoneID,
		X:               0.0,
		Y:               0.0,
		Z:               0.0,
		SpawnX:          0.0,
		SpawnY:          0.0,
		SpawnZ:          0.0,
		Health:          100.0,
		MaxHealth:       100.0,
		AggroRadius:     20.0,
		AttackRange:     2.0,
		AttackDamage:    5.0,
		AttackCadenceMs: 1_000,
		MoveSpeedPerSec: 4.0,
		LeashRadius:     10.0,
		RespawnDelayMs:  1_000,
		Alive:           true,
		Targetable:      true,
		AIState:         mobAIStateIdle,
	}
	session := &worldSessionState{
		Token:       "world_ai_test",
		CharacterID: "char_ai_test",
		ZoneID:      defaultZoneID,
		Connected:   true,
		Alive:       true,
		Health:      100.0,
		MaxHealth:   100.0,
	}
	server.mobs = map[string]*mobState{mob.ID: mob}
	server.mobOrder = []string{mob.ID}
	server.sessionsByToken = map[string]*worldSessionState{session.Token: session}
	server.sessionTokenByChar = map[string]string{session.CharacterID: session.Token}
	return server, mob, session
}

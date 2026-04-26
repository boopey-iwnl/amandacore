package worlds

import (
	"strings"
	"testing"
	"time"
)

func TestContentAbilityAndAuraCatalogActivate(t *testing.T) {
	server := newWorldServer(nil)

	ability, found := server.findAbilityDefinitionLocked("dev_stalker_pressure")
	if !found {
		t.Fatalf("expected content ability to be registered")
	}
	if ability.TargetRule != abilityTargetRuleEnemy || len(ability.Effects) != 1 {
		t.Fatalf("unexpected content ability shape: %#v", ability)
	}
	if _, found := server.auraCatalog["dev_pressure_mark"]; !found {
		t.Fatalf("expected content aura to be registered")
	}
}

func TestAuraApplyTickAndExpire(t *testing.T) {
	server, mob, session := newNPCCombatLoopTestServer()
	session.LearnedAbilityIDs = append(session.LearnedAbilityIDs, "dev_stalker_pressure")
	session.X = mob.X - 1
	session.Y = mob.Y
	if err := server.setTargetLocked(session, mob.ID); err != nil {
		t.Fatalf("select target: %v", err)
	}

	if err := server.activateAbilityLocked(session, "dev_stalker_pressure"); err != nil {
		t.Fatalf("apply aura ability: %v", err)
	}
	aura, active := mob.ActiveAuras["dev_pressure_mark"]
	if !active {
		t.Fatalf("expected aura on mob, got %#v", mob.ActiveAuras)
	}
	if aura.StackCount != 1 || aura.NextTickAtMs == 0 || aura.ExpiresAtMs == 0 {
		t.Fatalf("unexpected aura state: %#v", aura)
	}
	if !hasDomainEvent(server, EventAuraApplied) || !hasStateDiff(server, diffAuraState, mob.ID) {
		t.Fatalf("expected aura applied event and state diff")
	}

	healthBeforeTick := mob.Health
	if err := server.advanceAurasLocked(aura.NextTickAtMs); err != nil {
		t.Fatalf("advance aura tick: %v", err)
	}
	if mob.Health >= healthBeforeTick {
		t.Fatalf("expected periodic aura damage, got %.1f -> %.1f", healthBeforeTick, mob.Health)
	}
	if !hasDomainEvent(server, EventAuraTicked) {
		t.Fatalf("expected aura tick event")
	}

	aura = mob.ActiveAuras["dev_pressure_mark"]
	if err := server.advanceAurasLocked(aura.ExpiresAtMs); err != nil {
		t.Fatalf("advance aura expiry: %v", err)
	}
	if _, active := mob.ActiveAuras["dev_pressure_mark"]; active {
		t.Fatalf("expected aura to expire, got %#v", mob.ActiveAuras)
	}
	if !hasDomainEvent(server, EventAuraExpired) {
		t.Fatalf("expected aura expired event")
	}
}

func TestCastAbilityCompletesOnWorldTick(t *testing.T) {
	server, mob, session := newNPCCombatLoopTestServer()
	abilityID := "dev_delayed_strike"
	server.abilityCatalog[abilityID] = normalizeAbilityDefinition(abilityDefinition{
		ID:             abilityID,
		DisplayName:    "Delayed Strike",
		RequiresTarget: true,
		TargetRule:     abilityTargetRuleEnemy,
		RangeMeters:    3,
		Timing:         abilityTiming{CastMs: 50},
		Damage:         5,
	})
	session.LearnedAbilityIDs = append(session.LearnedAbilityIDs, abilityID)
	session.X = mob.X - 1
	session.Y = mob.Y
	if err := server.setTargetLocked(session, mob.ID); err != nil {
		t.Fatalf("select target: %v", err)
	}

	if err := server.activateAbilityLocked(session, abilityID); err != nil {
		t.Fatalf("start cast: %v", err)
	}
	if session.CastingAbilityID != abilityID {
		t.Fatalf("expected active cast, got %q", session.CastingAbilityID)
	}
	if mob.Health != mob.MaxHealth {
		t.Fatalf("cast should not apply damage immediately")
	}
	if !hasDomainEvent(server, EventAbilityCastStarted) {
		t.Fatalf("expected cast started event")
	}

	session.CastEndsAtMs = nowMillis() - 1
	server.lastUpdatedAt = time.Now().Add(-time.Second)
	if err := server.advanceWorldLocked(time.Now()); err != nil {
		t.Fatalf("advance world: %v", err)
	}
	if session.CastingAbilityID != "" {
		t.Fatalf("expected cast to complete, got %q", session.CastingAbilityID)
	}
	if mob.Health >= mob.MaxHealth {
		t.Fatalf("expected completed cast to damage mob, got %.1f", mob.Health)
	}
	if !hasDomainEvent(server, EventAbilityCastCompleted) {
		t.Fatalf("expected cast completed event")
	}
}

func TestCooldownCategoryBlocksSiblingAbility(t *testing.T) {
	server, _, session := newNPCCombatLoopTestServer()
	firstID := "dev_shared_focus"
	secondID := "dev_shared_guard"
	for _, abilityID := range []string{firstID, secondID} {
		server.abilityCatalog[abilityID] = normalizeAbilityDefinition(abilityDefinition{
			ID:               abilityID,
			DisplayName:      abilityID,
			TargetRule:       abilityTargetRuleSelf,
			CooldownMs:       1500,
			CooldownCategory: "dev_focus",
			HealAmount:       1,
		})
		session.LearnedAbilityIDs = append(session.LearnedAbilityIDs, abilityID)
	}

	if err := server.activateAbilityLocked(session, firstID); err != nil {
		t.Fatalf("expected first ability to resolve: %v", err)
	}
	if err := server.activateAbilityLocked(session, secondID); err == nil || !strings.Contains(err.Error(), "cooling down") {
		t.Fatalf("expected shared cooldown rejection, got %v", err)
	}
	if !hasDomainEvent(server, EventCooldownStarted) {
		t.Fatalf("expected cooldown started event")
	}
}

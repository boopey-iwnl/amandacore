package worlds

import (
	"fmt"
	"sort"
	"strings"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/platform"
)

const (
	abilityTargetRuleSelf  = "self"
	abilityTargetRuleEnemy = "enemy"
	abilityTargetRuleAlly  = "ally"
	abilityTargetRuleNone  = "none"

	abilityEffectDirectDamage = "direct_damage"
	abilityEffectHeal         = "heal"
	abilityEffectApplyAura    = "apply_aura"

	auraKindBuff    = "buff"
	auraKindDebuff  = "debuff"
	auraKindPassive = "passive"

	auraStackRefresh = "refresh"
	auraStackStack   = "stack"
	auraStackIgnore  = "ignore"

	auraTickNone     = "none"
	auraTickInterval = "interval"
)

type abilityTiming struct {
	CastMs         int64
	ChannelMs      int64
	TickIntervalMs int64
}

type abilityEffectDefinition struct {
	Kind             string
	AuraID           string
	Magnitude        float64
	UseAbilityDamage bool
	UseAbilityHeal   bool
}

type auraDefinition struct {
	ID             string
	DisplayName    string
	Kind           string
	DurationMs     int64
	MaxStacks      int
	StackRule      string
	TickRule       string
	TickIntervalMs int64
	TickEffects    []abilityEffectDefinition
	Tags           []string
}

type auraInstance struct {
	AuraID         string
	DisplayName    string
	Kind           string
	SourceEntityID string
	TargetEntityID string
	TargetKind     string
	StackCount     int
	AppliedAtMs    int64
	RefreshedAtMs  int64
	ExpiresAtMs    int64
	NextTickAtMs   int64
	LastTickAtMs   int64
	DurationMs     int64
	TickIntervalMs int64
}

type auraTarget struct {
	EntityID string
	Kind     string
	Session  *worldSessionState
	Mob      *mobState
}

func defaultAbilityCatalog() map[string]abilityDefinition {
	catalog := map[string]abilityDefinition{}
	for _, ability := range warriorAbilityCatalog {
		ability = normalizeAbilityDefinition(ability)
		catalog[ability.ID] = ability
	}
	return catalog
}

func defaultAuraCatalog() map[string]auraDefinition {
	return map[string]auraDefinition{
		"dev_pressure_mark": {
			ID:             "dev_pressure_mark",
			DisplayName:    "Pressure Mark",
			Kind:           auraKindDebuff,
			DurationMs:     3000,
			MaxStacks:      1,
			StackRule:      auraStackRefresh,
			TickRule:       auraTickInterval,
			TickIntervalMs: 1000,
			TickEffects: []abilityEffectDefinition{
				{Kind: abilityEffectDirectDamage, Magnitude: 2},
			},
			Tags: []string{"dev"},
		},
	}
}

func normalizeAbilityDefinition(ability abilityDefinition) abilityDefinition {
	ability.ID = normalizeAbilityID(ability.ID)
	ability.TargetRule = normalizeAbilityTargetRule(ability.TargetRule)
	if ability.TargetRule == "" {
		switch {
		case ability.TargetDisposition == npcDispositionHostile:
			ability.TargetRule = abilityTargetRuleEnemy
		case ability.RequiresTarget:
			ability.TargetRule = abilityTargetRuleEnemy
		case ability.HealAmount > 0:
			ability.TargetRule = abilityTargetRuleSelf
		default:
			ability.TargetRule = abilityTargetRuleNone
		}
	}
	if len(ability.Effects) == 0 {
		if ability.Damage > 0 {
			ability.Effects = append(ability.Effects, abilityEffectDefinition{
				Kind:             abilityEffectDirectDamage,
				Magnitude:        ability.Damage,
				UseAbilityDamage: true,
			})
		}
		if ability.HealAmount > 0 {
			ability.Effects = append(ability.Effects, abilityEffectDefinition{
				Kind:           abilityEffectHeal,
				Magnitude:      ability.HealAmount,
				UseAbilityHeal: true,
			})
		}
	}
	return ability
}

func normalizeAbilityTargetRule(rule string) string {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case abilityTargetRuleSelf:
		return abilityTargetRuleSelf
	case abilityTargetRuleEnemy:
		return abilityTargetRuleEnemy
	case abilityTargetRuleAlly:
		return abilityTargetRuleAlly
	case abilityTargetRuleNone:
		return abilityTargetRuleNone
	default:
		return strings.TrimSpace(rule)
	}
}

func (s *worldServer) findAbilityDefinitionLocked(abilityID string) (abilityDefinition, bool) {
	normalizedAbilityID := normalizeAbilityID(abilityID)
	if s != nil && s.abilityCatalog != nil {
		if ability, found := s.abilityCatalog[normalizedAbilityID]; found {
			return normalizeAbilityDefinition(ability), true
		}
	}
	ability, found := findAbilityDefinition(normalizedAbilityID)
	if !found {
		return abilityDefinition{}, false
	}
	return normalizeAbilityDefinition(ability), true
}

func (s *worldServer) registerContentAbilityCatalogLocked(registry contentpkg.RuntimeContentRegistry) {
	if s.abilityCatalog == nil {
		s.abilityCatalog = defaultAbilityCatalog()
	}
	if s.auraCatalog == nil {
		s.auraCatalog = defaultAuraCatalog()
	}
	for _, auraID := range contentpkg.SortedKeys(registry.Auras) {
		aura := contentAuraDefinition(registry.Auras[auraID])
		s.auraCatalog[aura.ID] = aura
	}
	for _, abilityID := range contentpkg.SortedKeys(registry.Abilities) {
		ability := contentAbilityDefinition(registry.Abilities[abilityID])
		s.abilityCatalog[ability.ID] = normalizeAbilityDefinition(ability)
	}
}

func contentAbilityDefinition(ability contentpkg.AbilityDefinition) abilityDefinition {
	effects := make([]abilityEffectDefinition, 0, len(ability.Effects))
	for _, effect := range ability.Effects {
		effects = append(effects, abilityEffectDefinition{
			Kind:      strings.ToLower(strings.TrimSpace(effect.Kind)),
			AuraID:    effect.AuraID,
			Magnitude: effect.Magnitude,
		})
	}
	return abilityDefinition{
		ID:               ability.AbilityID,
		DisplayName:      ability.DisplayName,
		Description:      ability.DisplayName,
		TooltipText:      ability.DisplayName,
		TargetRule:       ability.TargetRule,
		RequiresTarget:   ability.TargetRule == abilityTargetRuleEnemy || ability.TargetRule == abilityTargetRuleAlly,
		RangeMeters:      ability.Range,
		Timing:           abilityTiming{CastMs: int64(ability.Timing.CastMS), ChannelMs: int64(ability.Timing.ChannelMS)},
		CooldownMs:       int64(ability.CooldownMS),
		CooldownCategory: ability.CooldownCategory,
		Effects:          effects,
	}
}

func contentAuraDefinition(aura contentpkg.AuraDefinition) auraDefinition {
	tickEffects := make([]abilityEffectDefinition, 0, len(aura.TickEffects))
	for _, effect := range aura.TickEffects {
		tickEffects = append(tickEffects, abilityEffectDefinition{
			Kind:      strings.ToLower(strings.TrimSpace(effect.Kind)),
			AuraID:    effect.AuraID,
			Magnitude: effect.Magnitude,
		})
	}
	return auraDefinition{
		ID:             aura.AuraID,
		DisplayName:    aura.DisplayName,
		Kind:           strings.ToLower(strings.TrimSpace(aura.Kind)),
		DurationMs:     int64(aura.DurationMS),
		MaxStacks:      aura.MaxStacks,
		StackRule:      strings.ToLower(strings.TrimSpace(aura.StackRule)),
		TickRule:       strings.ToLower(strings.TrimSpace(aura.TickRule)),
		TickIntervalMs: int64(aura.TickIntervalMS),
		TickEffects:    tickEffects,
		Tags:           append([]string(nil), aura.Tags...),
	}
}

func (s *worldServer) validateAbilityUseLocked(session *worldSessionState, targetMob *mobState, targetPlayer *worldSessionState, ability abilityDefinition) error {
	nowMs := nowMillis()
	if session.Resource < ability.ResourceCost {
		return fmt.Errorf("not enough resource")
	}
	if cooldownEndsAt := session.abilityCooldownEndsAtFor(ability); cooldownEndsAt > nowMs {
		return fmt.Errorf("ability is cooling down")
	}
	if ability.RequiresTarget || ability.TargetRule == abilityTargetRuleEnemy || ability.TargetRule == abilityTargetRuleAlly {
		if session.CurrentTargetID == "" {
			return fmt.Errorf("no target")
		}
		if targetMob == nil && targetPlayer == nil {
			return fmt.Errorf("target is invalid")
		}
		if ability.TargetDisposition == npcDispositionHostile && targetMob == nil {
			return fmt.Errorf("target is not hostile")
		}
		if targetMob != nil {
			if !targetMob.Alive || !targetMob.Targetable {
				return fmt.Errorf("target is invalid")
			}
			if ability.TargetDisposition != "" && targetMob.Disposition != ability.TargetDisposition {
				return fmt.Errorf("target disposition is invalid")
			}
			if ability.TargetRule == abilityTargetRuleAlly || ability.TargetRule == abilityTargetRuleSelf {
				return fmt.Errorf("target is invalid")
			}
			if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > ability.RangeMeters {
				return fmt.Errorf("target is out of range")
			}
			return nil
		}
		if ability.TargetDisposition == string(NpcDispositionHostile) || ability.TargetRule == abilityTargetRuleEnemy {
			if err := s.validatePvPDamageLocked(session, targetPlayer); err != nil {
				return err
			}
		}
		if distance2D(session.X, session.Y, targetPlayer.X, targetPlayer.Y) > ability.RangeMeters {
			return fmt.Errorf("target is out of range")
		}
	}
	return nil
}

func (s *worldServer) commitAbilityUseLocked(session *worldSessionState, ability abilityDefinition) {
	nowMs := nowMillis()
	session.Resource -= ability.ResourceCost
	if ability.ResourceGeneration > 0 {
		generation := ability.ResourceGeneration
		if ability.ID == platform.RallyingCallAbilityID {
			generation += float64(session.Talents[rallyRhythmTalentID] * 6)
		}
		session.Resource = minFloat(session.MaxResource, session.Resource+generation)
	}
	if ability.TriggersGCD {
		session.GlobalCooldownEnds = nowMs + playerGlobalCooldownMs
	}
	if ability.CooldownMs <= 0 {
		return
	}

	cooldownEndsAt := nowMs + ability.CooldownMs
	cooldowns := session.ensureAbilityCooldowns()
	cooldowns[ability.ID] = cooldownEndsAt
	if ability.CooldownCategory != "" {
		cooldowns[cooldownCategoryKey(ability.CooldownCategory)] = cooldownEndsAt
	}
	fields := map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"category":          ability.CooldownCategory,
		"cooldownEndsAt":    cooldownEndsAt,
	}
	s.emitWorldEventLocked(EventCombatCooldownStarted, fields)
	s.emitWorldEventLocked(EventCooldownStarted, fields, cooldownDelta(session.CharacterID, ability.ID, ability.CooldownCategory, cooldownEndsAt, "started"))
}

func cooldownCategoryKey(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return ""
	}
	return "category:" + category
}

func (session *worldSessionState) abilityCooldownEndsAtFor(ability abilityDefinition) int64 {
	if session == nil {
		return 0
	}
	endsAt := session.abilityCooldownEndsAt(ability.ID)
	if ability.CooldownCategory != "" && session.AbilityCooldowns != nil {
		endsAt = maxInt64(endsAt, session.AbilityCooldowns[cooldownCategoryKey(ability.CooldownCategory)])
	}
	return endsAt
}

func (s *worldServer) expireReadyCooldownsLocked(session *worldSessionState, nowMs int64) {
	if session == nil || len(session.AbilityCooldowns) == 0 {
		return
	}
	keys := make([]string, 0, len(session.AbilityCooldowns))
	for key := range session.AbilityCooldowns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		endsAt := session.AbilityCooldowns[key]
		if endsAt <= 0 || endsAt > nowMs {
			continue
		}
		delete(session.AbilityCooldowns, key)
		abilityID := key
		category := ""
		if strings.HasPrefix(key, "category:") {
			abilityID = ""
			category = strings.TrimPrefix(key, "category:")
		}
		s.emitWorldEventLocked(EventCooldownReady, map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         abilityID,
			"category":          category,
		}, cooldownDelta(session.CharacterID, abilityID, category, endsAt, "ready"))
	}
}

func abilityCastDurationMs(ability abilityDefinition) int64 {
	if ability.Timing.CastMs > 0 {
		return ability.Timing.CastMs
	}
	if ability.Timing.ChannelMs > 0 {
		return ability.Timing.ChannelMs
	}
	return 0
}

func (s *worldServer) startAbilityCastLocked(session *worldSessionState, ability abilityDefinition) error {
	durationMs := abilityCastDurationMs(ability)
	if durationMs <= 0 {
		return nil
	}
	s.commitAbilityUseLocked(session, ability)
	nowMs := nowMillis()
	session.CastingAbilityID = ability.ID
	session.CastingTargetID = session.CurrentTargetID
	session.CastEndsAtMs = nowMs + durationMs
	s.emitWorldEventLocked(EventAbilityCastStarted, map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"targetId":          session.CastingTargetID,
		"castEndsAt":        session.CastEndsAtMs,
		"channel":           ability.Timing.ChannelMs > 0,
	}, castStateDelta(session.CharacterID, ability.ID, session.CastingTargetID, session.CastEndsAtMs, "started"))
	return nil
}

func (s *worldServer) completeCastLocked(session *worldSessionState) error {
	if session == nil || session.CastingAbilityID == "" || session.CastEndsAtMs == 0 {
		return nil
	}
	abilityID := session.CastingAbilityID
	targetID := session.CastingTargetID
	endsAt := session.CastEndsAtMs
	session.CastingAbilityID = ""
	session.CastingTargetID = ""
	session.CastEndsAtMs = 0

	ability, found := s.findAbilityDefinitionLocked(abilityID)
	if !found {
		return fmt.Errorf("ability is not available")
	}
	targetMob := s.findMobForSessionLocked(session, targetID)
	targetPlayer := s.findPlayerTargetForSessionLocked(session, targetID)
	if ability.RequiresTarget || ability.TargetRule == abilityTargetRuleEnemy || ability.TargetRule == abilityTargetRuleAlly {
		if targetMob == nil && targetPlayer == nil {
			s.emitWorldEventLocked(EventAbilityCastInterrupted, map[string]any{
				"worldSessionToken": session.Token,
				"characterId":       session.CharacterID,
				"abilityId":         ability.ID,
				"targetId":          targetID,
				"reason":            "target_invalid",
			}, castStateDelta(session.CharacterID, ability.ID, targetID, endsAt, "interrupted"))
			return fmt.Errorf("target is invalid")
		}
	}
	s.emitWorldEventLocked(EventAbilityCastCompleted, map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"targetId":          targetID,
	}, castStateDelta(session.CharacterID, ability.ID, targetID, endsAt, "completed"))
	return s.resolveAbilityEffectsLocked(session, targetMob, targetPlayer, ability)
}

func (s *worldServer) resolveAbilityEffectsLocked(session *worldSessionState, targetMob *mobState, targetPlayer *worldSessionState, ability abilityDefinition) error {
	effects := ability.Effects
	if len(effects) == 0 {
		s.emitWorldEventLocked(EventCombatAbilityResolved, map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         ability.ID,
			"targetId":          session.CurrentTargetID,
		}, abilityResultDelta(session.CharacterID, session.CurrentTargetID, ability.ID, 0, true))
		return nil
	}

	for _, effect := range effects {
		if err := s.resolveAbilityEffectLocked(session, targetMob, targetPlayer, ability, effect); err != nil {
			return err
		}
	}
	s.emitDomainEventLocked(eventCombatAbilityResolved, map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"targetId":          session.CurrentTargetID,
	})
	return nil
}

func (s *worldServer) resolveAbilityEffectLocked(session *worldSessionState, targetMob *mobState, targetPlayer *worldSessionState, ability abilityDefinition, effect abilityEffectDefinition) error {
	targetID := session.CurrentTargetID
	switch effect.Kind {
	case abilityEffectDirectDamage:
		damage := effect.Magnitude
		if effect.UseAbilityDamage {
			damage = s.abilityDamage(session, ability)
		}
		if targetMob != nil {
			targetID = targetMob.ID
			s.emitWorldEventLocked(EventCombatAbilityResolved, map[string]any{
				"worldSessionToken": session.Token,
				"characterId":       session.CharacterID,
				"abilityId":         ability.ID,
				"targetId":          targetMob.ID,
				"damage":            damage,
			}, abilityResultDelta(session.CharacterID, targetMob.ID, ability.ID, damage, true))
			if err := s.applyDamageToMobLocked(session, targetMob, damage, ability.ID); err != nil {
				return err
			}
		} else if targetPlayer != nil {
			targetID = targetPlayer.CharacterID
			s.emitWorldEventLocked(EventCombatAbilityResolved, map[string]any{
				"worldSessionToken": session.Token,
				"characterId":       session.CharacterID,
				"abilityId":         ability.ID,
				"targetId":          targetPlayer.CharacterID,
				"damage":            damage,
			}, abilityResultDelta(session.CharacterID, targetPlayer.CharacterID, ability.ID, damage, true))
			if err := s.applyDamageToPlayerLocked(session, targetPlayer, damage, ability.ID); err != nil {
				return err
			}
		}
		s.emitWorldEventLocked(EventAbilityEffectResolved, map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         ability.ID,
			"effectKind":        effect.Kind,
			"targetId":          targetID,
			"amount":            damage,
		}, abilityResultDelta(session.CharacterID, targetID, ability.ID, damage, true))
	case abilityEffectHeal:
		amount := effect.Magnitude
		if effect.UseAbilityHeal {
			amount = ability.HealAmount
		}
		target := session
		if targetPlayer != nil && ability.TargetRule == abilityTargetRuleAlly {
			target = targetPlayer
		}
		target.Health = minFloat(target.MaxHealth, target.Health+amount)
		s.emitWorldEventLocked(EventEntityHealthChanged, map[string]any{
			"entityId":        target.CharacterID,
			"health":          target.Health,
			"maxHealth":       target.MaxHealth,
			"sourceEntityId":  session.CharacterID,
			"sourceAbilityId": ability.ID,
		}, entityHealthDelta(target.CharacterID, target.Health, target.MaxHealth, target.Alive))
		s.emitWorldEventLocked(EventAbilityEffectResolved, map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         ability.ID,
			"effectKind":        effect.Kind,
			"targetId":          target.CharacterID,
			"amount":            amount,
		}, abilityResultDelta(session.CharacterID, target.CharacterID, ability.ID, 0, true))
	case abilityEffectApplyAura:
		target := auraTarget{EntityID: session.CharacterID, Kind: "player", Session: session}
		if ability.TargetRule != abilityTargetRuleSelf {
			if targetMob != nil {
				target = auraTarget{EntityID: targetMob.ID, Kind: "npc", Mob: targetMob}
			} else if targetPlayer != nil {
				target = auraTarget{EntityID: targetPlayer.CharacterID, Kind: "player", Session: targetPlayer}
			}
		}
		if err := s.applyAuraToTargetLocked(session, target, effect); err != nil {
			return err
		}
		s.emitWorldEventLocked(EventAbilityEffectResolved, map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         ability.ID,
			"effectKind":        effect.Kind,
			"auraId":            effect.AuraID,
			"targetId":          target.EntityID,
		}, abilityResultDelta(session.CharacterID, target.EntityID, ability.ID, 0, true))
	}
	return nil
}

func (s *worldServer) applyAuraToTargetLocked(source *worldSessionState, target auraTarget, effect abilityEffectDefinition) error {
	aura, found := s.auraCatalog[effect.AuraID]
	if !found {
		return fmt.Errorf("aura is not available")
	}
	nowMs := nowMillis()
	if aura.MaxStacks <= 0 {
		aura.MaxStacks = 1
	}
	if aura.StackRule == "" {
		aura.StackRule = auraStackRefresh
	}
	active := target.activeAuras()
	if active == nil {
		active = map[string]auraInstance{}
		target.setActiveAuras(active)
	}

	instance, exists := active[aura.ID]
	action := "applied"
	eventName := EventAuraApplied
	if !exists {
		instance = auraInstance{
			AuraID:         aura.ID,
			DisplayName:    aura.DisplayName,
			Kind:           aura.Kind,
			SourceEntityID: source.CharacterID,
			TargetEntityID: target.EntityID,
			TargetKind:     target.Kind,
			StackCount:     1,
			AppliedAtMs:    nowMs,
			DurationMs:     aura.DurationMs,
			TickIntervalMs: aura.TickIntervalMs,
		}
	} else {
		action = "refreshed"
		eventName = EventAuraRefreshed
		switch aura.StackRule {
		case auraStackStack:
			instance.StackCount = contentMinInt(aura.MaxStacks, instance.StackCount+1)
		case auraStackIgnore:
			return nil
		default:
			instance.StackCount = maxInt(1, instance.StackCount)
		}
	}
	instance.RefreshedAtMs = nowMs
	if aura.DurationMs > 0 {
		instance.ExpiresAtMs = nowMs + aura.DurationMs
	}
	if aura.TickRule == auraTickInterval && aura.TickIntervalMs > 0 {
		instance.TickIntervalMs = aura.TickIntervalMs
		instance.NextTickAtMs = nowMs + aura.TickIntervalMs
	}
	active[aura.ID] = instance
	target.setActiveAuras(active)
	s.emitWorldEventLocked(eventName, map[string]any{
		"sourceEntityId": source.CharacterID,
		"targetEntityId": target.EntityID,
		"auraId":         aura.ID,
		"stackCount":     instance.StackCount,
		"expiresAtMs":    instance.ExpiresAtMs,
	}, auraStateDelta(target.EntityID, instance, action))
	return nil
}

func (target auraTarget) activeAuras() map[string]auraInstance {
	if target.Session != nil {
		return target.Session.ActiveAuras
	}
	if target.Mob != nil {
		return target.Mob.ActiveAuras
	}
	return nil
}

func (target auraTarget) setActiveAuras(auras map[string]auraInstance) {
	if target.Session != nil {
		target.Session.ActiveAuras = auras
	}
	if target.Mob != nil {
		target.Mob.ActiveAuras = auras
	}
}

func (s *worldServer) advanceAurasLocked(nowMs int64) error {
	for _, session := range s.sessionsByToken {
		s.expireReadyCooldownsLocked(session, nowMs)
		if err := s.advanceAuraTargetLocked(nowMs, auraTarget{EntityID: session.CharacterID, Kind: "player", Session: session}); err != nil {
			return err
		}
	}
	for _, mob := range s.allHostileMobsLocked() {
		if mob == nil {
			continue
		}
		if !mob.Alive {
			s.clearAurasForTargetLocked(auraTarget{EntityID: mob.ID, Kind: "npc", Mob: mob}, "entity_dead")
			continue
		}
		if err := s.advanceAuraTargetLocked(nowMs, auraTarget{EntityID: mob.ID, Kind: "npc", Mob: mob}); err != nil {
			return err
		}
	}
	return nil
}

func (s *worldServer) advanceAuraTargetLocked(nowMs int64, target auraTarget) error {
	active := target.activeAuras()
	if len(active) == 0 {
		return nil
	}
	for _, auraID := range sortedAuraIDs(active) {
		instance := active[auraID]
		if instance.ExpiresAtMs > 0 && nowMs >= instance.ExpiresAtMs {
			delete(active, auraID)
			s.emitWorldEventLocked(EventAuraExpired, map[string]any{
				"targetEntityId": target.EntityID,
				"auraId":         auraID,
				"reason":         "duration_elapsed",
			}, auraStateDelta(target.EntityID, instance, "expired"))
			continue
		}
		if instance.NextTickAtMs <= 0 || nowMs < instance.NextTickAtMs {
			continue
		}
		if err := s.resolveAuraTickLocked(target, instance); err != nil {
			return err
		}
		instance.LastTickAtMs = nowMs
		if instance.TickIntervalMs > 0 {
			instance.NextTickAtMs = nowMs + instance.TickIntervalMs
		}
		active[auraID] = instance
	}
	target.setActiveAuras(active)
	return nil
}

func (s *worldServer) resolveAuraTickLocked(target auraTarget, instance auraInstance) error {
	aura, found := s.auraCatalog[instance.AuraID]
	if !found {
		return nil
	}
	sourceSession := s.findConnectedSessionByCharacterLocked(instance.SourceEntityID)
	for _, effect := range aura.TickEffects {
		amount := effect.Magnitude * float64(maxInt(1, instance.StackCount))
		switch effect.Kind {
		case abilityEffectDirectDamage:
			if target.Mob != nil && sourceSession != nil {
				if err := s.applyDamageToMobLocked(sourceSession, target.Mob, amount, instance.AuraID); err != nil {
					return err
				}
			} else if target.Session != nil {
				target.Session.Health = maxFloat(0, target.Session.Health-amount)
				s.emitWorldEventLocked(EventEntityHealthChanged, map[string]any{
					"entityId":        target.Session.CharacterID,
					"health":          target.Session.Health,
					"maxHealth":       target.Session.MaxHealth,
					"sourceEntityId":  instance.SourceEntityID,
					"sourceAbilityId": instance.AuraID,
				}, entityHealthDelta(target.Session.CharacterID, target.Session.Health, target.Session.MaxHealth, target.Session.Alive))
			}
		case abilityEffectHeal:
			if target.Session != nil {
				target.Session.Health = minFloat(target.Session.MaxHealth, target.Session.Health+amount)
				s.emitWorldEventLocked(EventEntityHealthChanged, map[string]any{
					"entityId":        target.Session.CharacterID,
					"health":          target.Session.Health,
					"maxHealth":       target.Session.MaxHealth,
					"sourceEntityId":  instance.SourceEntityID,
					"sourceAbilityId": instance.AuraID,
				}, entityHealthDelta(target.Session.CharacterID, target.Session.Health, target.Session.MaxHealth, target.Session.Alive))
			}
		}
	}
	s.emitWorldEventLocked(EventAuraTicked, map[string]any{
		"sourceEntityId": instance.SourceEntityID,
		"targetEntityId": target.EntityID,
		"auraId":         instance.AuraID,
		"stackCount":     instance.StackCount,
	}, auraStateDelta(target.EntityID, instance, "ticked"))
	return nil
}

func (s *worldServer) clearAurasForTargetLocked(target auraTarget, reason string) {
	active := target.activeAuras()
	if len(active) == 0 {
		return
	}
	for _, auraID := range sortedAuraIDs(active) {
		instance := active[auraID]
		delete(active, auraID)
		s.emitWorldEventLocked(EventAuraExpired, map[string]any{
			"targetEntityId": target.EntityID,
			"auraId":         auraID,
			"reason":         reason,
		}, auraStateDelta(target.EntityID, instance, "expired"))
	}
	target.setActiveAuras(active)
}

func buildAuraResponses(auras map[string]auraInstance) []auraResponse {
	if len(auras) == 0 {
		return []auraResponse{}
	}
	responses := make([]auraResponse, 0, len(auras))
	for _, auraID := range sortedAuraIDs(auras) {
		aura := auras[auraID]
		responses = append(responses, auraResponse{
			AuraID:         aura.AuraID,
			DisplayName:    aura.DisplayName,
			Kind:           aura.Kind,
			SourceEntityID: aura.SourceEntityID,
			TargetEntityID: aura.TargetEntityID,
			StackCount:     aura.StackCount,
			AppliedAtMs:    aura.AppliedAtMs,
			ExpiresAtMs:    aura.ExpiresAtMs,
			NextTickAtMs:   aura.NextTickAtMs,
		})
	}
	return responses
}

func sortedAuraIDs(auras map[string]auraInstance) []string {
	ids := make([]string, 0, len(auras))
	for auraID := range auras {
		ids = append(ids, auraID)
	}
	sort.Strings(ids)
	return ids
}

func contentMinInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

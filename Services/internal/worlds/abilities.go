package worlds

import (
	"fmt"
	"strconv"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	emberBoltAbilityAlias   = "ember_bolt"
	steadyBlastAbilityAlias = "steady_blast"
)

type abilityDefinition struct {
	ID                 string
	DisplayName        string
	ClassID            string
	Description        string
	RequirementText    string
	TooltipText        string
	RequiredLevel      int
	LearnedByDefault   bool
	TrainerLearnable   bool
	TrainerCostCopper  int
	ActionBarSlot      int
	ActionBarHotkey    string
	ActionBarLabel     string
	RequiresTarget     bool
	RangeMeters        float64
	ResourceCost       float64
	ResourceGeneration float64
	CooldownMs         int64
	TriggersGCD        bool
	Damage             float64
	AttackPowerScale   float64
	HealAmount         float64
}

var warriorAbilityCatalog = []abilityDefinition{
	{
		ID:                 platform.AutoAttackAbilityID,
		DisplayName:        "Auto Attack",
		ClassID:            platform.DefaultClassID,
		Description:        "Maintain pressure with your weapon while a target stays in melee range.",
		TooltipText:        "Automatic melee swings generate Grit.",
		RequirementText:    "Known by all Warriors.",
		RequiredLevel:      1,
		LearnedByDefault:   true,
		ActionBarSlot:      0,
		ActionBarHotkey:    "F",
		ActionBarLabel:     "Atk",
		RequiresTarget:     true,
		RangeMeters:        playerAutoAttackRange,
		ResourceGeneration: playerAutoAttackGrit,
	},
	{
		ID:                 platform.SteadyStrikeAbilityID,
		DisplayName:        "Steady Strike",
		ClassID:            platform.DefaultClassID,
		Description:        "A measured weapon strike that builds Grit with reliable melee damage.",
		TooltipText:        "Strike your target and gain 10 Grit.",
		RequirementText:    "Known by default.",
		RequiredLevel:      1,
		LearnedByDefault:   true,
		ActionBarSlot:      1,
		ActionBarHotkey:    "1",
		ActionBarLabel:     "Strike",
		RequiresTarget:     true,
		RangeMeters:        playerAutoAttackRange,
		ResourceGeneration: 10.0,
		TriggersGCD:        true,
		Damage:             8.0,
		AttackPowerScale:   0.65,
	},
	{
		ID:               platform.BraceAbilityID,
		DisplayName:      "Brace",
		ClassID:          platform.DefaultClassID,
		Description:      "Set your stance and recover a small amount of health without needing a target.",
		TooltipText:      "Recover health and prepare for the next exchange.",
		RequirementText:  "Known by default.",
		RequiredLevel:    1,
		LearnedByDefault: true,
		ActionBarSlot:    2,
		ActionBarHotkey:  "2",
		ActionBarLabel:   "Brace",
		RequiresTarget:   false,
		RangeMeters:      0.0,
		CooldownMs:       8000,
		TriggersGCD:      true,
		HealAmount:       14.0,
	},
	{
		ID:                platform.DrivingBlowAbilityID,
		DisplayName:       "Driving Blow",
		ClassID:           platform.DefaultClassID,
		Description:       "A heavy follow-through strike taught by the Warrior trainer.",
		TooltipText:       "Spend 25 Grit for a heavier melee hit.",
		RequirementText:   "Requires a Warrior trainer and 10 copper.",
		RequiredLevel:     2,
		TrainerLearnable:  true,
		TrainerCostCopper: 10,
		ActionBarSlot:     3,
		ActionBarHotkey:   "3",
		ActionBarLabel:    "Drive",
		RequiresTarget:    true,
		RangeMeters:       playerAutoAttackRange,
		ResourceCost:      25.0,
		TriggersGCD:       true,
		Damage:            12.0,
		AttackPowerScale:  0.95,
	},
	{
		ID:                 platform.RallyingCallAbilityID,
		DisplayName:        "Rallying Call",
		ClassID:            platform.DefaultClassID,
		Description:        "A focused call that restores Grit and steadies the Warrior.",
		TooltipText:        "Generate 18 Grit. Does not require a target.",
		RequirementText:    "Requires level 4 and a Warrior trainer.",
		RequiredLevel:      4,
		TrainerLearnable:   true,
		TrainerCostCopper:  25,
		ActionBarSlot:      4,
		ActionBarHotkey:    "4",
		ActionBarLabel:     "Rally",
		RequiresTarget:     false,
		ResourceGeneration: 18.0,
		CooldownMs:         12000,
		TriggersGCD:        true,
	},
	{
		ID:                platform.HamperingStrikeAbilityID,
		DisplayName:       "Hampering Strike",
		ClassID:           platform.DefaultClassID,
		Description:       "A controlling strike that deals modest damage while keeping pressure on a target.",
		TooltipText:       "Spend 15 Grit for a quick controlling strike.",
		RequirementText:   "Requires level 6 and a Warrior trainer.",
		RequiredLevel:     6,
		TrainerLearnable:  true,
		TrainerCostCopper: 40,
		ActionBarSlot:     5,
		ActionBarHotkey:   "5",
		ActionBarLabel:    "Hamper",
		RequiresTarget:    true,
		RangeMeters:       playerAutoAttackRange,
		ResourceCost:      15.0,
		CooldownMs:        5000,
		TriggersGCD:       true,
		Damage:            6.0,
		AttackPowerScale:  0.5,
	},
	{
		ID:                platform.GuardedFormAbilityID,
		DisplayName:       "Guarded Form",
		ClassID:           platform.DefaultClassID,
		Description:       "A defensive form that gives the early rotation a survival button.",
		TooltipText:       "Spend 20 Grit to recover health.",
		RequirementText:   "Requires level 8 and a Warrior trainer.",
		RequiredLevel:     8,
		TrainerLearnable:  true,
		TrainerCostCopper: 65,
		ActionBarSlot:     6,
		ActionBarHotkey:   "6",
		ActionBarLabel:    "Guard",
		RequiresTarget:    false,
		ResourceCost:      20.0,
		CooldownMs:        18000,
		TriggersGCD:       true,
		HealAmount:        28.0,
	},
	{
		ID:                platform.OverhandCutAbilityID,
		DisplayName:       "Overhand Cut",
		ClassID:           platform.DefaultClassID,
		Description:       "A committed weapon attack for Warriors who have learned to manage Grit.",
		TooltipText:       "Spend 40 Grit for a strong melee attack.",
		RequirementText:   "Requires level 10 and a Warrior trainer.",
		RequiredLevel:     10,
		TrainerLearnable:  true,
		TrainerCostCopper: 90,
		ActionBarSlot:     7,
		ActionBarHotkey:   "7",
		ActionBarLabel:    "Cut",
		RequiresTarget:    true,
		RangeMeters:       playerAutoAttackRange,
		ResourceCost:      40.0,
		CooldownMs:        3000,
		TriggersGCD:       true,
		Damage:            18.0,
		AttackPowerScale:  1.25,
	},
	{
		ID:                platform.IronResolveAbilityID,
		DisplayName:       "Iron Resolve",
		ClassID:           platform.DefaultClassID,
		Description:       "A durable Warrior passive granted through advanced starter training.",
		TooltipText:       "A passive foundation for later defensive class work.",
		RequirementText:   "Requires level 12 and a Warrior trainer.",
		RequiredLevel:     12,
		TrainerLearnable:  true,
		TrainerCostCopper: 120,
		ActionBarLabel:    "Resolve",
		RequiresTarget:    false,
	},
}

func normalizeAbilityID(abilityID string) string {
	switch abilityID {
	case emberBoltAbilityAlias:
		return platform.SteadyStrikeAbilityID
	case steadyBlastAbilityAlias:
		return platform.BraceAbilityID
	default:
		return abilityID
	}
}

func abilityIconKind(ability abilityDefinition) string {
	switch ability.ID {
	case platform.AutoAttackAbilityID:
		return "weapon"
	case platform.SteadyStrikeAbilityID,
		platform.DrivingBlowAbilityID,
		platform.HamperingStrikeAbilityID,
		platform.OverhandCutAbilityID:
		return "strike"
	case platform.BraceAbilityID,
		platform.GuardedFormAbilityID,
		platform.IronResolveAbilityID:
		return "defense"
	case platform.RallyingCallAbilityID:
		return "utility"
	default:
		return "ability"
	}
}

func findAbilityDefinition(abilityID string) (abilityDefinition, bool) {
	normalizedAbilityID := normalizeAbilityID(abilityID)
	for _, ability := range warriorAbilityCatalog {
		if ability.ID == normalizedAbilityID {
			return ability, true
		}
	}

	return abilityDefinition{}, false
}

func knownAbilitySet(learnedAbilityIDs []string) map[string]struct{} {
	normalizedAbilityIDs := platform.NormalizeLearnedAbilityIDs(learnedAbilityIDs)
	known := make(map[string]struct{}, len(normalizedAbilityIDs))
	for _, abilityID := range normalizedAbilityIDs {
		known[normalizeAbilityID(abilityID)] = struct{}{}
	}
	return known
}

func actionBarHotkey(slotIndex int) string {
	switch slotIndex {
	case 0:
		return "F"
	case 1, 2, 3, 4, 5, 6, 7, 8, 9:
		return strconv.Itoa(slotIndex)
	case 10:
		return "0"
	case 11:
		return "-"
	default:
		return ""
	}
}

func (s *worldServer) buildSpellbookResponse(session *worldSessionState) []map[string]any {
	knownAbilities := knownAbilitySet(session.LearnedAbilityIDs)
	spellbook := make([]map[string]any, 0, len(warriorAbilityCatalog))
	for _, ability := range warriorAbilityCatalog {
		_, learned := knownAbilities[ability.ID]
		spellbook = append(spellbook, map[string]any{
			"id":                 ability.ID,
			"displayName":        ability.DisplayName,
			"classId":            ability.ClassID,
			"description":        ability.Description,
			"tooltipText":        abilityTooltip(ability),
			"requiredLevel":      ability.RequiredLevel,
			"learned":            learned,
			"requirementText":    ability.RequirementText,
			"iconKind":           abilityIconKind(ability),
			"resourceName":       "Grit",
			"resourceCost":       ability.ResourceCost,
			"resourceGeneration": ability.ResourceGeneration,
			"cooldownMs":         ability.CooldownMs,
			"triggersGCD":        ability.TriggersGCD,
			"rangeMeters":        ability.RangeMeters,
			"requiresTarget":     ability.RequiresTarget,
		})
	}

	return spellbook
}

func (s *worldServer) buildActionBarResponse(session *worldSessionState) []map[string]any {
	slots := make([]map[string]any, 48)
	for slotIndex := range slots {
		slots[slotIndex] = map[string]any{
			"slotIndex":           slotIndex,
			"hotkey":              "",
			"abilityId":           "",
			"displayName":         "",
			"buttonLabel":         "",
			"requiresTarget":      false,
			"learned":             false,
			"resourceName":        "Grit",
			"resourceCost":        0.0,
			"resourceGeneration":  0.0,
			"cooldownMs":          int64(0),
			"cooldownEndsAt":      int64(0),
			"cooldownRemainingMs": int64(0),
			"triggersGCD":         false,
			"rangeMeters":         0.0,
			"tooltipText":         "",
		}
	}

	knownAbilities := knownAbilitySet(session.LearnedAbilityIDs)
	actionBarSlots := platform.NormalizeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs)
	for _, actionBarSlot := range actionBarSlots {
		if actionBarSlot.SlotIndex < 0 || actionBarSlot.SlotIndex >= len(slots) || actionBarSlot.AbilityID == "" {
			continue
		}

		ability, found := findAbilityDefinition(actionBarSlot.AbilityID)
		if !found {
			continue
		}
		if _, learned := knownAbilities[ability.ID]; !learned {
			continue
		}
		hotkey := actionBarHotkey(actionBarSlot.SlotIndex)

		cooldownEndsAt := session.abilityCooldownEndsAt(ability.ID)
		remainingMs := maxInt64(0, cooldownEndsAt-nowMillis())
		slots[actionBarSlot.SlotIndex] = map[string]any{
			"slotIndex":           actionBarSlot.SlotIndex,
			"hotkey":              hotkey,
			"abilityId":           ability.ID,
			"displayName":         ability.DisplayName,
			"buttonLabel":         ability.ActionBarLabel,
			"iconKind":            abilityIconKind(ability),
			"requiresTarget":      ability.RequiresTarget,
			"learned":             true,
			"resourceName":        "Grit",
			"resourceCost":        ability.ResourceCost,
			"resourceGeneration":  ability.ResourceGeneration,
			"cooldownMs":          ability.CooldownMs,
			"cooldownEndsAt":      cooldownEndsAt,
			"cooldownRemainingMs": remainingMs,
			"triggersGCD":         ability.TriggersGCD,
			"rangeMeters":         ability.RangeMeters,
			"tooltipText":         abilityTooltip(ability),
		}
	}

	return slots
}

func (s *worldServer) assignActionBarSlotLocked(session *worldSessionState, slotIndex int, abilityID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if slotIndex < 0 || slotIndex >= platform.ActionBarSlotCount {
		return fmt.Errorf("action bar slot is out of range")
	}

	ability, found := findAbilityDefinition(abilityID)
	if !found {
		return fmt.Errorf("ability is not available")
	}
	if !s.sessionKnowsAbilityLocked(session, ability.ID) {
		return fmt.Errorf("ability is not learned")
	}

	actionBarSlots := platform.NormalizeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs)
	actionBarSlots[slotIndex] = platform.CharacterActionBarSlot{
		SlotIndex: slotIndex,
		AbilityID: ability.ID,
	}

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
	s.recordPersistenceDuration("character_action_bar", persistStartedAt, err)
	if err != nil {
		return err
	}
	session.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, session.LearnedAbilityIDs)

	observability.LogEvent("world-service", "world.action_bar_slot_assigned", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"slotIndex":         slotIndex,
		"abilityId":         ability.ID,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) clearActionBarSlotLocked(session *worldSessionState, slotIndex int) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if slotIndex < 0 || slotIndex >= platform.ActionBarSlotCount {
		return fmt.Errorf("action bar slot is out of range")
	}

	actionBarSlots := platform.NormalizeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs)
	actionBarSlots[slotIndex] = platform.CharacterActionBarSlot{SlotIndex: slotIndex}

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
	s.recordPersistenceDuration("character_action_bar", persistStartedAt, err)
	if err != nil {
		return err
	}
	session.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, session.LearnedAbilityIDs)

	observability.LogEvent("world-service", "world.action_bar_slot_cleared", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"slotIndex":         slotIndex,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) moveActionBarSlotLocked(session *worldSessionState, fromSlotIndex int, toSlotIndex int) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if fromSlotIndex < 0 || fromSlotIndex >= platform.ActionBarSlotCount ||
		toSlotIndex < 0 || toSlotIndex >= platform.ActionBarSlotCount {
		return fmt.Errorf("action bar slot is out of range")
	}
	if fromSlotIndex == toSlotIndex {
		return nil
	}

	actionBarSlots := platform.NormalizeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs)
	fromSlot := actionBarSlots[fromSlotIndex]
	toSlot := actionBarSlots[toSlotIndex]
	if fromSlot.AbilityID == "" {
		return fmt.Errorf("source slot is empty")
	}

	actionBarSlots[fromSlotIndex] = platform.CharacterActionBarSlot{
		SlotIndex: fromSlotIndex,
		AbilityID: toSlot.AbilityID,
	}
	actionBarSlots[toSlotIndex] = platform.CharacterActionBarSlot{
		SlotIndex: toSlotIndex,
		AbilityID: fromSlot.AbilityID,
	}

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
	s.recordPersistenceDuration("character_action_bar", persistStartedAt, err)
	if err != nil {
		return err
	}
	session.ActionBarSlots = platform.NormalizeActionBarSlots(character.ActionBarSlots, session.LearnedAbilityIDs)

	observability.LogEvent("world-service", "world.action_bar_slot_moved", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"fromSlotIndex":     fromSlotIndex,
		"toSlotIndex":       toSlotIndex,
		"abilityId":         fromSlot.AbilityID,
		"swappedAbilityId":  toSlot.AbilityID,
		"updatedAt":         time.Now().Unix(),
	})
	return nil
}

func (s *worldServer) sessionKnowsAbilityLocked(session *worldSessionState, abilityID string) bool {
	if session == nil {
		return false
	}

	_, known := knownAbilitySet(session.LearnedAbilityIDs)[normalizeAbilityID(abilityID)]
	return known
}

func (s *worldServer) applyAbilityEffectLocked(session *worldSessionState, targetMob *mobState, targetPlayer *worldSessionState, ability abilityDefinition) error {
	nowMs := nowMillis()
	if session.Resource < ability.ResourceCost {
		return fmt.Errorf("not enough resource")
	}
	if cooldownEndsAt := session.abilityCooldownEndsAt(ability.ID); cooldownEndsAt > nowMs {
		return fmt.Errorf("ability is cooling down")
	}

	if ability.RequiresTarget {
		if session.CurrentTargetID == "" {
			return fmt.Errorf("no target")
		}
		if targetMob == nil && targetPlayer == nil {
			return fmt.Errorf("target is invalid")
		}
		if targetMob != nil {
			if !targetMob.Alive || !targetMob.Targetable {
				return fmt.Errorf("target is invalid")
			}
			if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > ability.RangeMeters {
				return fmt.Errorf("target is out of range")
			}
		} else {
			if err := s.validatePvPDamageLocked(session, targetPlayer); err != nil {
				return err
			}
			if distance2D(session.X, session.Y, targetPlayer.X, targetPlayer.Y) > ability.RangeMeters {
				return fmt.Errorf("target is out of range")
			}
		}
	}

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
	if ability.CooldownMs > 0 {
		session.ensureAbilityCooldowns()[ability.ID] = nowMs + ability.CooldownMs
	}

	observability.LogEvent("world-service", "world.ability_requested", map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"targetId":          session.CurrentTargetID,
	})

	if ability.Damage > 0 && targetMob != nil {
		damage := s.abilityDamage(session, ability)
		if err := s.applyDamageToMobLocked(session, targetMob, damage, ability.ID); err != nil {
			return err
		}
	}
	if ability.Damage > 0 && targetPlayer != nil {
		damage := s.abilityDamage(session, ability)
		if err := s.applyDamageToPlayerLocked(session, targetPlayer, damage, ability.ID); err != nil {
			return err
		}
	}
	if ability.HealAmount > 0 {
		session.Health = minFloat(session.MaxHealth, session.Health+ability.HealAmount)
		observability.LogEvent("world-service", "world.ability_self_applied", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"abilityId":         ability.ID,
			"health":            session.Health,
		})
	}

	return nil
}

func abilityTooltip(ability abilityDefinition) string {
	if ability.TooltipText != "" {
		return ability.TooltipText
	}
	return ability.Description
}

func (session *worldSessionState) ensureAbilityCooldowns() map[string]int64 {
	if session.AbilityCooldowns == nil {
		session.AbilityCooldowns = map[string]int64{}
	}
	return session.AbilityCooldowns
}

func (session *worldSessionState) abilityCooldownEndsAt(abilityID string) int64 {
	if session == nil || session.AbilityCooldowns == nil {
		return 0
	}
	return session.AbilityCooldowns[normalizeAbilityID(abilityID)]
}

func (s *worldServer) abilityDamage(session *worldSessionState, ability abilityDefinition) float64 {
	stats := calculatePlayerStats(session.Level, session.Equipment, session.Talents)
	damage := ability.Damage + (stats.AttackPower * ability.AttackPowerScale)
	if ability.ID == platform.SteadyStrikeAbilityID {
		damage += float64(session.Talents[hardLessonsTalentID] * 2)
	}
	return maxFloat(1.0, damage)
}

func maxInt64(left int64, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

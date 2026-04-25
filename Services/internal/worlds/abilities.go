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
	ID                string
	DisplayName       string
	Description       string
	RequirementText   string
	RequiredLevel     int
	LearnedByDefault  bool
	TrainerLearnable  bool
	TrainerCostCopper int
	ActionBarSlot     int
	ActionBarHotkey   string
	ActionBarLabel    string
	RequiresTarget    bool
	RangeMeters       float64
	ResourceCost      float64
	Damage            float64
	HealAmount        float64
}

var warriorAbilityCatalog = []abilityDefinition{
	{
		ID:               platform.AutoAttackAbilityID,
		DisplayName:      "Auto Attack",
		Description:      "Maintain pressure with your weapon while a target stays in melee range.",
		RequirementText:  "Known by all Warriors.",
		RequiredLevel:    1,
		LearnedByDefault: true,
		ActionBarSlot:    0,
		ActionBarHotkey:  "F",
		ActionBarLabel:   "Atk",
		RequiresTarget:   true,
		RangeMeters:      playerAutoAttackRange,
	},
	{
		ID:               platform.SteadyStrikeAbilityID,
		DisplayName:      "Steady Strike",
		Description:      "A measured weapon strike that spends Grit for reliable melee damage.",
		RequirementText:  "Known by default.",
		RequiredLevel:    1,
		LearnedByDefault: true,
		ActionBarSlot:    1,
		ActionBarHotkey:  "1",
		ActionBarLabel:   "Strike",
		RequiresTarget:   true,
		RangeMeters:      playerAutoAttackRange,
		ResourceCost:     15.0,
		Damage:           18.0,
	},
	{
		ID:               platform.BraceAbilityID,
		DisplayName:      "Brace",
		Description:      "Set your stance and recover a small amount of health without needing a target.",
		RequirementText:  "Known by default.",
		RequiredLevel:    1,
		LearnedByDefault: true,
		ActionBarSlot:    2,
		ActionBarHotkey:  "2",
		ActionBarLabel:   "Brace",
		RequiresTarget:   false,
		RangeMeters:      0.0,
		ResourceCost:     10.0,
		HealAmount:       14.0,
	},
	{
		ID:                platform.DrivingBlowAbilityID,
		DisplayName:       "Driving Blow",
		Description:       "A harder follow-through strike taught by the Warrior trainer.",
		RequirementText:   "Requires a Warrior trainer and 10 copper.",
		RequiredLevel:     1,
		TrainerLearnable:  true,
		TrainerCostCopper: 10,
		ActionBarSlot:     3,
		ActionBarHotkey:   "3",
		ActionBarLabel:    "Drive",
		RequiresTarget:    true,
		RangeMeters:       playerAutoAttackRange,
		ResourceCost:      25.0,
		Damage:            28.0,
	},
	{
		ID:                platform.WarCryAbilityID,
		DisplayName:       "War Cry",
		Description:       "A rallying shout that will later become available through Warrior training.",
		RequirementText:   "Requires level 5 and a Warrior trainer.",
		RequiredLevel:     5,
		TrainerLearnable:  true,
		TrainerCostCopper: 25,
		ActionBarSlot:     4,
		ActionBarHotkey:   "4",
		ActionBarLabel:    "Cry",
		RequiresTarget:    false,
	},
	{
		ID:                platform.HamperingStrikeAbilityID,
		DisplayName:       "Hampering Strike",
		Description:       "A controlling strike previewed for the next band of Warrior progression.",
		RequirementText:   "Requires level 6 and a Warrior trainer.",
		RequiredLevel:     6,
		TrainerLearnable:  true,
		TrainerCostCopper: 40,
		RequiresTarget:    true,
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
			"id":              ability.ID,
			"displayName":     ability.DisplayName,
			"description":     ability.Description,
			"requiredLevel":   ability.RequiredLevel,
			"learned":         learned,
			"requirementText": ability.RequirementText,
		})
	}

	return spellbook
}

func (s *worldServer) buildActionBarResponse(session *worldSessionState) []map[string]any {
	slots := make([]map[string]any, 48)
	for slotIndex := range slots {
		slots[slotIndex] = map[string]any{
			"slotIndex":      slotIndex,
			"hotkey":         "",
			"abilityId":      "",
			"displayName":    "",
			"buttonLabel":    "",
			"requiresTarget": false,
			"learned":        false,
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

		slots[actionBarSlot.SlotIndex] = map[string]any{
			"slotIndex":      actionBarSlot.SlotIndex,
			"hotkey":         hotkey,
			"abilityId":      ability.ID,
			"displayName":    ability.DisplayName,
			"buttonLabel":    ability.ActionBarLabel,
			"requiresTarget": ability.RequiresTarget,
			"learned":        true,
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

	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
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

	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
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

	character, err := s.store.UpdateCharacterActionBarSlots(session.CharacterID, actionBarSlots)
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

func (s *worldServer) applyAbilityEffectLocked(session *worldSessionState, targetMob *mobState, ability abilityDefinition) error {
	if session.Resource < ability.ResourceCost {
		return fmt.Errorf("not enough resource")
	}

	if ability.RequiresTarget {
		if session.CurrentTargetID == "" {
			return fmt.Errorf("no target")
		}
		if targetMob == nil || !targetMob.Alive || !targetMob.Targetable {
			return fmt.Errorf("target is invalid")
		}
		if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > ability.RangeMeters {
			return fmt.Errorf("target is out of range")
		}
	}

	session.Resource -= ability.ResourceCost
	session.GlobalCooldownEnds = nowMillis() + playerGlobalCooldownMs

	observability.LogEvent("world-service", "world.ability_requested", map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"abilityId":         ability.ID,
		"targetId":          session.CurrentTargetID,
	})

	if ability.Damage > 0 && targetMob != nil {
		if err := s.applyDamageToMobLocked(session, targetMob, ability.Damage, ability.ID); err != nil {
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

package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const warriorTrainerID = "trainer_armsmaster_corin_vale"

type trainerDefinition struct {
	ID          string
	DisplayName string
	ClassID     string
	X           float64
	Y           float64
	Z           float64
	Radius      float64
	AbilityIDs  []string
}

var warriorTrainer = trainerDefinition{
	ID:          warriorTrainerID,
	DisplayName: "Armsmaster Corin Vale",
	ClassID:     platform.DefaultClassID,
	X:           12.0,
	Y:           8.0,
	Z:           0.0,
	Radius:      starterInteractRadius,
	AbilityIDs: []string{
		platform.DrivingBlowAbilityID,
		platform.RallyingCallAbilityID,
		platform.HamperingStrikeAbilityID,
		platform.GuardedFormAbilityID,
		platform.OverhandCutAbilityID,
		platform.IronResolveAbilityID,
	},
}

func findTrainerDefinition(trainerID string) (trainerDefinition, bool) {
	if trainerID == warriorTrainer.ID {
		return warriorTrainer, true
	}

	return trainerDefinition{}, false
}

func (s *worldServer) trainerInRangeLocked(session *worldSessionState, trainer trainerDefinition) bool {
	if session == nil {
		return false
	}

	trainer = s.resolveTrainerLocationLocked(trainer)
	return distance2D(session.X, session.Y, trainer.X, trainer.Y) <= trainer.Radius
}

func (s *worldServer) resolveTrainerLocationLocked(trainer trainerDefinition) trainerDefinition {
	if npc, ok := s.friendlyNPCs[trainer.ID]; ok {
		trainer.X = npc.X
		trainer.Y = npc.Y
		trainer.Z = npc.Z
		if npc.Radius > 0 {
			trainer.Radius = npc.Radius
		}
	}
	return trainer
}

func (s *worldServer) buildTrainerResponse(session *worldSessionState) map[string]any {
	if session == nil {
		return map[string]any{}
	}

	trainer := s.resolveTrainerLocationLocked(warriorTrainer)
	inRange := s.trainerInRangeLocked(session, trainer)
	offers := make([]map[string]any, 0, len(trainer.AbilityIDs))
	for _, abilityID := range trainer.AbilityIDs {
		ability, found := findAbilityDefinition(abilityID)
		if !found || !ability.TrainerLearnable {
			continue
		}

		learned := s.sessionKnowsAbilityLocked(session, ability.ID)
		requirementText := ability.RequirementText
		canLearn := false
		if learned {
			requirementText = "Already learned."
		} else if session.ClassID != trainer.ClassID {
			requirementText = "Requires Warrior class."
		} else if session.Level < ability.RequiredLevel {
			requirementText = fmt.Sprintf("Requires level %d.", ability.RequiredLevel)
		} else if session.CurrencyCopper < ability.TrainerCostCopper {
			requirementText = fmt.Sprintf("Requires %d copper.", ability.TrainerCostCopper)
		} else if session.CurrentTargetID != trainer.ID {
			requirementText = "Right-click the Warrior trainer NPC to train."
		} else if !inRange {
			requirementText = "Move closer to the Warrior trainer."
		} else {
			canLearn = true
			requirementText = "Ready to learn."
		}

		offers = append(offers, map[string]any{
			"abilityId":          ability.ID,
			"displayName":        ability.DisplayName,
			"description":        ability.Description,
			"tooltipText":        abilityTooltip(ability),
			"requiredLevel":      ability.RequiredLevel,
			"costCopper":         ability.TrainerCostCopper,
			"learned":            learned,
			"canLearn":           canLearn,
			"requirementText":    requirementText,
			"actionBarSlot":      ability.ActionBarSlot,
			"actionBarHotkey":    ability.ActionBarHotkey,
			"actionBarLabel":     ability.ActionBarLabel,
			"requiresTarget":     ability.RequiresTarget,
			"resourceName":       "Grit",
			"resourceCost":       ability.ResourceCost,
			"resourceGeneration": ability.ResourceGeneration,
			"cooldownMs":         ability.CooldownMs,
			"rangeMeters":        ability.RangeMeters,
		})
	}

	return map[string]any{
		"id":              trainer.ID,
		"displayName":     trainer.DisplayName,
		"classId":         trainer.ClassID,
		"inRange":         inRange,
		"interactionHint": "Right-click the Warrior trainer NPC to train.",
		"offers":          offers,
	}
}

func (s *worldServer) learnTrainerAbilityLocked(session *worldSessionState, trainerID string, abilityID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}

	trainer, found := findTrainerDefinition(trainerID)
	if !found {
		return fmt.Errorf("trainer is not available")
	}
	trainer = s.resolveTrainerLocationLocked(trainer)
	if session.ClassID != trainer.ClassID {
		return fmt.Errorf("wrong class for this trainer")
	}
	if session.CurrentTargetID != trainer.ID {
		return fmt.Errorf("right-click the Warrior trainer NPC before training")
	}
	if !s.trainerInRangeLocked(session, trainer) {
		return fmt.Errorf("move closer to the Warrior trainer")
	}

	ability, found := findAbilityDefinition(abilityID)
	if !found || !ability.TrainerLearnable {
		return fmt.Errorf("ability is not trainable")
	}

	offered := false
	for _, offeredAbilityID := range trainer.AbilityIDs {
		if normalizeAbilityID(offeredAbilityID) == normalizeAbilityID(ability.ID) {
			offered = true
			break
		}
	}
	if !offered {
		return fmt.Errorf("trainer does not offer that ability")
	}

	if session.Level < ability.RequiredLevel {
		return fmt.Errorf("level is too low to learn that ability")
	}
	if s.sessionKnowsAbilityLocked(session, ability.ID) {
		return fmt.Errorf("ability is already learned")
	}
	if session.CurrencyCopper < ability.TrainerCostCopper {
		return fmt.Errorf("not enough copper")
	}

	existingActionBarSlots := platform.NormalizeActionBarSlots(session.ActionBarSlots, session.LearnedAbilityIDs)
	session.CurrencyCopper -= ability.TrainerCostCopper
	session.LearnedAbilityIDs = platform.NormalizeLearnedAbilityIDs(append(session.LearnedAbilityIDs, ability.ID))
	session.ActionBarSlots = platform.NormalizeActionBarSlots(existingActionBarSlots, session.LearnedAbilityIDs)

	if err := s.persistSessionProgressionLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.trainer_ability_learned", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"trainerId":         trainer.ID,
		"abilityId":         ability.ID,
		"costCopper":        ability.TrainerCostCopper,
		"currencyCopper":    session.CurrencyCopper,
		"level":             session.Level,
		"learnedAt":         time.Now().Unix(),
	})

	return nil
}

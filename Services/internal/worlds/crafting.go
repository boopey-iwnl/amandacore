package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

func (s *worldServer) craftRecipeLocked(session *worldSessionState, recipeID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}

	recipe, found := findRecipeDefinition(recipeID)
	if !found {
		return fmt.Errorf("recipe is not available")
	}
	if !recipe.Implemented {
		return fmt.Errorf("recipe is not available in this milestone")
	}

	professionState, found := sessionProfessionState(session, recipe.ProfessionID)
	if !found {
		return fmt.Errorf("required profession is not learned")
	}
	if !professionKnowsRecipe(professionState, recipe.ID) {
		return fmt.Errorf("recipe is not learned")
	}
	if professionState.SkillValue < recipe.RequiredSkill {
		return fmt.Errorf("profession skill is too low")
	}

	outputItem, found := findItemDefinition(recipe.OutputItemID)
	if !found {
		return fmt.Errorf("crafted item is not defined")
	}

	inventory := platform.NormalizeInventorySlots(session.Inventory)
	for _, material := range recipe.RequiredMaterials {
		if inventoryItemCount(inventory, material.ItemID) < material.Quantity {
			return fmt.Errorf("not enough %s", material.DisplayName)
		}
	}

	for _, material := range recipe.RequiredMaterials {
		if err := removeInventoryItemCount(&inventory, material.ItemID, material.Quantity); err != nil {
			return err
		}
	}
	if err := addDefinedItemToInventory(&inventory, outputItem, recipe.OutputQuantity); err != nil {
		return err
	}

	session.Inventory = inventory
	session.Professions = advanceCraftingSkill(session.Professions, recipe.ProfessionID)
	if err := s.persistSessionEconomyLocked(session); err != nil {
		return err
	}
	if err := s.persistSessionProfessionsLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.recipe_crafted", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"recipeId":          recipe.ID,
		"professionId":      recipe.ProfessionID,
		"outputItemId":      recipe.OutputItemID,
		"outputQuantity":    recipe.OutputQuantity,
		"craftedAt":         time.Now().Unix(),
	})
	return nil
}

func sessionProfessionState(
	session *worldSessionState,
	professionID string,
) (platform.CharacterProfessionState, bool) {
	if session == nil || professionID == "" {
		return platform.CharacterProfessionState{}, false
	}
	for _, profession := range platform.NormalizeProfessionStates(session.Professions) {
		if profession.ProfessionID == professionID {
			return profession, true
		}
	}
	return platform.CharacterProfessionState{}, false
}

func professionKnowsRecipe(profession platform.CharacterProfessionState, recipeID string) bool {
	for _, knownRecipeID := range platform.NormalizeKnownRecipeIDs(profession.KnownRecipeIDs) {
		if knownRecipeID == recipeID {
			return true
		}
	}
	return false
}

func advanceCraftingSkill(
	professions []platform.CharacterProfessionState,
	professionID string,
) []platform.CharacterProfessionState {
	normalized := platform.NormalizeProfessionStates(professions)
	for index := range normalized {
		if normalized[index].ProfessionID != professionID {
			continue
		}
		profession, found := findProfessionDefinition(professionID)
		if found && normalized[index].SkillValue < profession.MaxStarterSkill {
			normalized[index].SkillValue++
			normalized[index].UpdatedAt = time.Now().Unix()
		}
		return normalized
	}
	return normalized
}

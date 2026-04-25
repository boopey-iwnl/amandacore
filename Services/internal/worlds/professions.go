package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	professionCategoryCrafting  = "crafting"
	professionCategoryGathering = "gathering"
	professionCategorySecondary = "secondary"
	professionRankNovice        = "novice"

	professionTrainerServiceType = "profession_trainer"
	professionTrainerTallaID     = "profession_trainer_talla_grayspark"
	npcProfessionTrainerTallaID  = "npc_talla_grayspark"
)

type professionDefinition struct {
	ID                string
	DisplayName       string
	Category          string
	MaxStarterSkill   int
	LearnCostCopper   int
	DefaultRecipeIDs  []string
	Implemented       bool
	UnavailableReason string
}

type recipeMaterialDefinition struct {
	ItemID      string
	DisplayName string
	Quantity    int
}

type recipeDefinition struct {
	ID                string
	ProfessionID      string
	DisplayName       string
	RequiredSkill     int
	RequiredMaterials []recipeMaterialDefinition
	OutputItemID      string
	OutputDisplayName string
	OutputQuantity    int
	LearnedByDefault  bool
	TrainerTaught     bool
	CraftTimeMs       int
	Category          string
	Implemented       bool
}

type professionTrainerDefinition struct {
	ID            string
	NPCID         string
	DisplayName   string
	ProfessionIDs []string
}

var professionCatalogOrder = []string{
	"field_alchemy",
	platform.ProfessionForgecraftID,
	"runebinding",
	"tinkerworks",
	"sigilcraft",
	"gemcutting",
	"hidecraft",
	"weavecraft",
	platform.ProfessionOrekeepingID,
	"wildharvest",
	"hide_gathering",
	"hearth_cooking",
	platform.ProfessionFieldAidID,
	"line_fishing",
}

var professionDefinitions = map[string]professionDefinition{
	"field_alchemy": {
		ID:                "field_alchemy",
		DisplayName:       "Field Alchemy",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Field Alchemy is defined for a later profession slice.",
	},
	platform.ProfessionForgecraftID: {
		ID:               platform.ProfessionForgecraftID,
		DisplayName:      "Forgecraft",
		Category:         professionCategoryCrafting,
		MaxStarterSkill:  25,
		DefaultRecipeIDs: []string{"recipe_shape_worn_rivets"},
		Implemented:      true,
	},
	"runebinding": {
		ID:                "runebinding",
		DisplayName:       "Runebinding",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Runebinding is defined for a later profession slice.",
	},
	"tinkerworks": {
		ID:                "tinkerworks",
		DisplayName:       "Tinkerworks",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Tinkerworks is defined for a later profession slice.",
	},
	"sigilcraft": {
		ID:                "sigilcraft",
		DisplayName:       "Sigilcraft",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Sigilcraft is defined for a later profession slice.",
	},
	"gemcutting": {
		ID:                "gemcutting",
		DisplayName:       "Gemcutting",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Gemcutting is defined for a later profession slice.",
	},
	"hidecraft": {
		ID:                "hidecraft",
		DisplayName:       "Hidecraft",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Hidecraft is defined for a later profession slice.",
	},
	"weavecraft": {
		ID:                "weavecraft",
		DisplayName:       "Weavecraft",
		Category:          professionCategoryCrafting,
		MaxStarterSkill:   25,
		UnavailableReason: "Weavecraft is defined for a later profession slice.",
	},
	platform.ProfessionOrekeepingID: {
		ID:              platform.ProfessionOrekeepingID,
		DisplayName:     "Orekeeping",
		Category:        professionCategoryGathering,
		MaxStarterSkill: 25,
		Implemented:     true,
	},
	"wildharvest": {
		ID:                "wildharvest",
		DisplayName:       "Wildharvest",
		Category:          professionCategoryGathering,
		MaxStarterSkill:   25,
		UnavailableReason: "Wildharvest is defined for a later profession slice.",
	},
	"hide_gathering": {
		ID:                "hide_gathering",
		DisplayName:       "Hide Gathering",
		Category:          professionCategoryGathering,
		MaxStarterSkill:   25,
		UnavailableReason: "Hide Gathering is defined for a later profession slice.",
	},
	"hearth_cooking": {
		ID:                "hearth_cooking",
		DisplayName:       "Hearth Cooking",
		Category:          professionCategorySecondary,
		MaxStarterSkill:   25,
		UnavailableReason: "Hearth Cooking is defined for a later profession slice.",
	},
	platform.ProfessionFieldAidID: {
		ID:               platform.ProfessionFieldAidID,
		DisplayName:      "Field Aid",
		Category:         professionCategorySecondary,
		MaxStarterSkill:  25,
		DefaultRecipeIDs: []string{"recipe_training_bandage"},
		Implemented:      true,
	},
	"line_fishing": {
		ID:                "line_fishing",
		DisplayName:       "Line Fishing",
		Category:          professionCategorySecondary,
		MaxStarterSkill:   25,
		UnavailableReason: "Line Fishing is defined for a later profession slice.",
	},
}

var recipeDefinitions = map[string]recipeDefinition{
	"recipe_shape_worn_rivets": {
		ID:            "recipe_shape_worn_rivets",
		ProfessionID:  platform.ProfessionForgecraftID,
		DisplayName:   "Shape Worn Rivets",
		RequiredSkill: 1,
		RequiredMaterials: []recipeMaterialDefinition{
			{ItemID: "vale_iron_chip", DisplayName: "Vale Iron Chip", Quantity: 2},
		},
		OutputItemID:      "worn_rivet",
		OutputDisplayName: "Worn Rivet",
		OutputQuantity:    2,
		LearnedByDefault:  true,
		Category:          "material",
	},
	"recipe_training_bandage": {
		ID:            "recipe_training_bandage",
		ProfessionID:  platform.ProfessionFieldAidID,
		DisplayName:   "Training Bandage",
		RequiredSkill: 1,
		RequiredMaterials: []recipeMaterialDefinition{
			{ItemID: "rough_binding_cloth", DisplayName: "Rough Binding Cloth", Quantity: 2},
		},
		OutputItemID:      "training_bandage",
		OutputDisplayName: "Training Bandage",
		OutputQuantity:    1,
		LearnedByDefault:  true,
		Category:          "consumable",
	},
}

var professionTrainerDefinitions = map[string]professionTrainerDefinition{
	professionTrainerTallaID: {
		ID:          professionTrainerTallaID,
		NPCID:       npcProfessionTrainerTallaID,
		DisplayName: "Talla Grayspark",
		ProfessionIDs: []string{
			platform.ProfessionOrekeepingID,
			platform.ProfessionForgecraftID,
			platform.ProfessionFieldAidID,
			"field_alchemy",
			"runebinding",
			"tinkerworks",
			"sigilcraft",
			"gemcutting",
			"hidecraft",
			"weavecraft",
			"wildharvest",
			"hide_gathering",
			"hearth_cooking",
			"line_fishing",
		},
	},
}

func findProfessionDefinition(professionID string) (professionDefinition, bool) {
	profession, ok := professionDefinitions[professionID]
	return profession, ok
}

func findRecipeDefinition(recipeID string) (recipeDefinition, bool) {
	recipe, ok := recipeDefinitions[recipeID]
	return recipe, ok
}

func findProfessionTrainerDefinition(trainerID string) (professionTrainerDefinition, bool) {
	trainer, ok := professionTrainerDefinitions[trainerID]
	return trainer, ok
}

func professionTrainerOffers(trainer professionTrainerDefinition, professionID string) bool {
	for _, offeredProfessionID := range trainer.ProfessionIDs {
		if offeredProfessionID == professionID {
			return true
		}
	}
	return false
}

func isPrimaryProfession(profession professionDefinition) bool {
	return profession.Category == professionCategoryCrafting || profession.Category == professionCategoryGathering
}

func primaryProfessionCount(professions []platform.CharacterProfessionState) int {
	count := 0
	for _, professionState := range platform.NormalizeProfessionStates(professions) {
		profession, found := findProfessionDefinition(professionState.ProfessionID)
		if found && isPrimaryProfession(profession) {
			count++
		}
	}
	return count
}

func (s *worldServer) sessionKnowsProfessionLocked(session *worldSessionState, professionID string) bool {
	if session == nil {
		return false
	}
	for _, profession := range platform.NormalizeProfessionStates(session.Professions) {
		if profession.ProfessionID == professionID {
			return true
		}
	}
	return false
}

func (s *worldServer) learnProfessionLocked(session *worldSessionState, trainerID string, professionID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}

	trainer, found := findProfessionTrainerDefinition(trainerID)
	if !found {
		return fmt.Errorf("profession trainer is not available")
	}
	if session.CurrentTargetID != trainer.NPCID {
		return fmt.Errorf("right-click the profession trainer NPC before training")
	}
	if !s.friendlyInRangeLocked(session, trainer.NPCID) {
		return fmt.Errorf("move closer to the profession trainer")
	}
	if !professionTrainerOffers(trainer, professionID) {
		return fmt.Errorf("trainer does not offer that profession")
	}

	profession, found := findProfessionDefinition(professionID)
	if !found {
		return fmt.Errorf("profession is not available")
	}
	if s.sessionKnowsProfessionLocked(session, profession.ID) {
		return fmt.Errorf("profession is already learned")
	}
	if isPrimaryProfession(profession) && primaryProfessionCount(session.Professions) >= platform.PrimaryProfessionLimit {
		return fmt.Errorf("primary profession limit reached")
	}
	if !profession.Implemented {
		return fmt.Errorf("profession is not available in this milestone")
	}
	if session.CurrencyCopper < profession.LearnCostCopper {
		return fmt.Errorf("not enough copper")
	}

	now := time.Now().Unix()
	session.CurrencyCopper -= profession.LearnCostCopper
	session.Professions = append(platform.NormalizeProfessionStates(session.Professions), platform.CharacterProfessionState{
		ProfessionID:   profession.ID,
		SkillValue:     1,
		RankID:         professionRankNovice,
		KnownRecipeIDs: platform.NormalizeKnownRecipeIDs(profession.DefaultRecipeIDs),
		LearnedAt:      now,
		UpdatedAt:      now,
	})

	if err := s.persistSessionProfessionsLocked(session); err != nil {
		return err
	}

	observability.LogEvent("world-service", "world.profession_learned", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"trainerId":         trainer.ID,
		"professionId":      profession.ID,
		"costCopper":        profession.LearnCostCopper,
		"currencyCopper":    session.CurrencyCopper,
		"learnedAt":         now,
	})
	return nil
}

func (s *worldServer) persistSessionProfessionsLocked(session *worldSessionState) error {
	character, err := s.store.UpdateCharacterProfessions(
		session.CharacterID,
		session.CurrencyCopper,
		session.Professions)
	if err != nil {
		return err
	}

	session.CurrencyCopper = character.CurrencyCopper
	session.Professions = platform.NormalizeProfessionStates(character.Professions)
	return nil
}

func (s *worldServer) buildProfessionsResponse(session *worldSessionState) map[string]any {
	if session == nil {
		return map[string]any{}
	}

	learned := make([]map[string]any, 0, len(session.Professions))
	for _, professionState := range platform.NormalizeProfessionStates(session.Professions) {
		profession, found := findProfessionDefinition(professionState.ProfessionID)
		if !found {
			continue
		}
		learned = append(learned, buildProfessionStateSummary(profession, professionState))
	}

	catalog := make([]map[string]any, 0, len(professionCatalogOrder))
	for _, professionID := range professionCatalogOrder {
		profession, found := findProfessionDefinition(professionID)
		if !found {
			continue
		}
		catalog = append(catalog, buildProfessionCatalogSummary(profession))
	}

	return map[string]any{
		"primaryLimit": platform.PrimaryProfessionLimit,
		"learned":      learned,
		"catalog":      catalog,
	}
}

func (s *worldServer) buildProfessionTrainerResponse(session *worldSessionState) map[string]any {
	if session == nil || session.CurrentTargetID == "" {
		return map[string]any{}
	}

	for _, trainer := range professionTrainerDefinitions {
		if trainer.NPCID != session.CurrentTargetID {
			continue
		}

		inRange := s.friendlyInRangeLocked(session, trainer.NPCID)
		offers := make([]map[string]any, 0, len(trainer.ProfessionIDs))
		for _, professionID := range trainer.ProfessionIDs {
			profession, found := findProfessionDefinition(professionID)
			if !found {
				continue
			}
			offers = append(offers, s.buildProfessionTrainerOffer(session, profession, inRange))
		}

		return map[string]any{
			"id":              trainer.ID,
			"npcId":           trainer.NPCID,
			"displayName":     trainer.DisplayName,
			"inRange":         inRange,
			"interactionHint": "Right-click the profession trainer NPC to learn starter professions.",
			"offers":          offers,
		}
	}

	return map[string]any{}
}

func (s *worldServer) buildProfessionTrainerOffer(
	session *worldSessionState,
	profession professionDefinition,
	inRange bool,
) map[string]any {
	learned := s.sessionKnowsProfessionLocked(session, profession.ID)
	canLearn := false
	requirementText := "Ready to learn."
	switch {
	case learned:
		requirementText = "Already learned."
	case isPrimaryProfession(profession) && primaryProfessionCount(session.Professions) >= platform.PrimaryProfessionLimit:
		requirementText = "Primary profession limit reached."
	case !profession.Implemented:
		requirementText = profession.UnavailableReason
	case session.CurrencyCopper < profession.LearnCostCopper:
		requirementText = fmt.Sprintf("Requires %d copper.", profession.LearnCostCopper)
	case !inRange:
		requirementText = "Move closer to the profession trainer."
	default:
		canLearn = true
	}

	return map[string]any{
		"professionId":      profession.ID,
		"displayName":       profession.DisplayName,
		"category":          profession.Category,
		"maxStarterSkill":   profession.MaxStarterSkill,
		"costCopper":        profession.LearnCostCopper,
		"learned":           learned,
		"implemented":       profession.Implemented,
		"canLearn":          canLearn,
		"requirementText":   requirementText,
		"unavailableReason": profession.UnavailableReason,
	}
}

func buildProfessionCatalogSummary(profession professionDefinition) map[string]any {
	return map[string]any{
		"professionId":      profession.ID,
		"displayName":       profession.DisplayName,
		"category":          profession.Category,
		"maxStarterSkill":   profession.MaxStarterSkill,
		"implemented":       profession.Implemented,
		"unavailableReason": profession.UnavailableReason,
	}
}

func buildProfessionStateSummary(
	profession professionDefinition,
	professionState platform.CharacterProfessionState,
) map[string]any {
	knownRecipeIDs := platform.NormalizeKnownRecipeIDs(professionState.KnownRecipeIDs)
	knownRecipes := make([]map[string]any, 0, len(knownRecipeIDs))
	for _, recipeID := range knownRecipeIDs {
		recipe, found := findRecipeDefinition(recipeID)
		if !found {
			continue
		}
		knownRecipes = append(knownRecipes, buildRecipeSummary(recipe))
	}

	return map[string]any{
		"professionId":    profession.ID,
		"displayName":     profession.DisplayName,
		"category":        profession.Category,
		"maxStarterSkill": profession.MaxStarterSkill,
		"skillValue":      professionState.SkillValue,
		"rankId":          professionState.RankID,
		"knownRecipeIds":  knownRecipeIDs,
		"knownRecipes":    knownRecipes,
		"learnedAt":       professionState.LearnedAt,
		"updatedAt":       professionState.UpdatedAt,
	}
}

func buildRecipeSummary(recipe recipeDefinition) map[string]any {
	materials := make([]map[string]any, 0, len(recipe.RequiredMaterials))
	for _, material := range recipe.RequiredMaterials {
		materials = append(materials, map[string]any{
			"itemId":      material.ItemID,
			"displayName": material.DisplayName,
			"quantity":    material.Quantity,
		})
	}

	return map[string]any{
		"recipeId":          recipe.ID,
		"professionId":      recipe.ProfessionID,
		"displayName":       recipe.DisplayName,
		"requiredSkill":     recipe.RequiredSkill,
		"requiredMaterials": materials,
		"outputItemId":      recipe.OutputItemID,
		"outputDisplayName": recipe.OutputDisplayName,
		"outputQuantity":    recipe.OutputQuantity,
		"learnedByDefault":  recipe.LearnedByDefault,
		"trainerTaught":     recipe.TrainerTaught,
		"craftTimeMs":       recipe.CraftTimeMs,
		"category":          recipe.Category,
		"implemented":       recipe.Implemented,
	}
}

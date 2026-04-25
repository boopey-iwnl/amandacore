package e2e

import (
	"net/http"
	"testing"

	"amandacore/services/internal/platform"
)

const (
	forgecraftRivetRecipeID = "recipe_shape_worn_rivets"
	wornRivetID             = "worn_rivet"
)

func TestForgecraftCraftingSlice(t *testing.T) {
	t.Run("Forgecraft crafts Worn Rivets from gathered ore", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fixture.learnForgecraft(t)
		fixture.moveToOreNode(t)
		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)

		state := fixture.craftRecipe(t, forgecraftRivetRecipeID, http.StatusOK)
		assertInventoryItemCount(t, state, valeIronChipID, 0)
		assertInventoryItemCount(t, state, wornRivetID, 2)
		assertProfessionSkill(t, state, platform.ProfessionForgecraftID, 2)
	})

	t.Run("crafting without required profession rejects", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fixture.moveToOreNode(t)
		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)

		fixture.craftRecipe(t, forgecraftRivetRecipeID, http.StatusBadRequest)
		state := fixture.getWorldState(t)
		assertInventoryItemCount(t, state, wornRivetID, 0)
	})

	t.Run("unknown recipe rejects", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnForgecraft(t)

		fixture.craftRecipe(t, "recipe_not_real", http.StatusBadRequest)
	})

	t.Run("insufficient materials reject cleanly", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnForgecraft(t)

		fixture.craftRecipe(t, forgecraftRivetRecipeID, http.StatusBadRequest)
		state := fixture.getWorldState(t)
		assertInventoryItemCount(t, state, wornRivetID, 0)
	})

	t.Run("inventory full rejects without consuming materials", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnForgecraft(t)
		fillInventoryForCraftingFullTest(t, fixture)
		fixture.reconnect(t)

		fixture.craftRecipe(t, forgecraftRivetRecipeID, http.StatusBadRequest)
		state := fixture.getWorldState(t)
		assertInventoryItemCount(t, state, valeIronChipID, 3)
		assertInventoryItemCount(t, state, wornRivetID, 0)
	})

	t.Run("crafted materials persist across reconnect and restart", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fixture.learnForgecraft(t)
		fixture.moveToOreNode(t)
		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)
		fixture.craftRecipe(t, forgecraftRivetRecipeID, http.StatusOK)

		state := fixture.reconnect(t)
		assertInventoryItemCount(t, state, wornRivetID, 2)
		assertProfessionSkill(t, state, platform.ProfessionForgecraftID, 2)

		restarted := restartProfessionFixture(t, fixture)
		state = restarted.getWorldState(t)
		assertInventoryItemCount(t, state, wornRivetID, 2)
		assertProfessionSkill(t, state, platform.ProfessionForgecraftID, 2)
	})
}

func (f *combatFixture) learnForgecraft(t *testing.T) {
	t.Helper()

	f.moveToProfessionTrainer(t)
	f.learnProfession(t, platform.ProfessionForgecraftID, http.StatusOK)
}

func (f *combatFixture) craftRecipe(t *testing.T, recipeID string, expectedStatus int) map[string]any {
	t.Helper()

	var state map[string]any
	target := any(&state)
	if expectedStatus >= http.StatusBadRequest {
		target = nil
	}
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/craft", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"recipeId":          recipeID,
	}, expectedStatus, target)
	return state
}

func assertProfessionSkill(t *testing.T, state map[string]any, professionID string, expectedSkill int) {
	t.Helper()

	profession := findLearnedProfession(t, state, professionID)
	if int(profession["skillValue"].(float64)) != expectedSkill {
		t.Fatalf("expected profession %s skill %d, got %#v", professionID, expectedSkill, profession)
	}
}

func fillInventoryForCraftingFullTest(t *testing.T, fixture *combatFixture) {
	t.Helper()

	fullInventory := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	fullInventory[0] = platform.CharacterInventorySlot{
		SlotIndex:   0,
		ItemID:      valeIronChipID,
		DisplayName: "Vale Iron Chip",
		StackCount:  3,
	}
	for slotIndex := 1; slotIndex < len(fullInventory); slotIndex++ {
		fullInventory[slotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   slotIndex,
			ItemID:      wornMilitiaBladeID,
			DisplayName: "Worn Militia Blade",
			StackCount:  1,
		}
	}
	if _, err := fixture.fileStore.UpdateCharacterInventory(fixture.characterID, fullInventory); err != nil {
		t.Fatalf("failed to fill inventory for crafting rejection test: %v", err)
	}
}

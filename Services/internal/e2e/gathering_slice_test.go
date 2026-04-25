package e2e

import (
	"net/http"
	"testing"
	"time"

	"amandacore/services/internal/platform"
)

const (
	stonewakeOreNodeID = "node_vale_iron_01"
	valeIronChipID     = "vale_iron_chip"
	wornMilitiaBladeID = "worn_militia_blade"
)

func TestOrekeepingGatheringSlice(t *testing.T) {
	t.Run("gathering nodes are visible", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)

		node := findGatheringNode(t, state, stonewakeOreNodeID)
		if node["displayName"].(string) != "Exposed Vale Iron" {
			t.Fatalf("expected clean-room ore node display name, got %#v", node)
		}
		if node["professionId"].(string) != platform.ProfessionOrekeepingID {
			t.Fatalf("expected Orekeeping node requirement, got %#v", node)
		}
		if node["targetable"].(bool) != true {
			t.Fatalf("expected ready gathering node to be targetable, got %#v", node)
		}
	})

	t.Run("gathering without Orekeeping rejects", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.moveToOreNode(t)

		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusBadRequest)
		state := fixture.getWorldState(t)
		assertInventoryItemCount(t, state, valeIronChipID, 0)
	})

	t.Run("Orekeeping gathering grants material and enforces respawn", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fixture.moveToOreNode(t)

		state := fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)
		assertInventoryItemCount(t, state, valeIronChipID, 2)

		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusBadRequest)
		time.Sleep(650 * time.Millisecond)

		state = fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)
		assertInventoryItemCount(t, state, valeIronChipID, 4)
	})

	t.Run("inventory full rejects without depleting the node", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fillInventoryWithBlades(t, fixture)
		fixture.reconnect(t)
		fixture.moveToOreNode(t)

		fixture.gatherNode(t, stonewakeOreNodeID, http.StatusBadRequest)

		state := fixture.getWorldState(t)
		assertInventoryItemCount(t, state, valeIronChipID, 0)
		freeInventorySlot(t, fixture, 0)
		fixture.reconnect(t)
		fixture.moveToOreNode(t)
		state = fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)
		assertInventoryItemCount(t, state, valeIronChipID, 2)
	})

	t.Run("gathered materials persist across reconnect and restart", func(t *testing.T) {
		fixture := newCombatFixture(t)
		fixture.learnOrekeeping(t)
		fixture.moveToOreNode(t)

		state := fixture.gatherNode(t, stonewakeOreNodeID, http.StatusOK)
		assertInventoryItemCount(t, state, valeIronChipID, 2)

		state = fixture.reconnect(t)
		assertInventoryItemCount(t, state, valeIronChipID, 2)

		restarted := restartProfessionFixture(t, fixture)
		state = restarted.getWorldState(t)
		assertInventoryItemCount(t, state, valeIronChipID, 2)
	})
}

func (f *combatFixture) learnOrekeeping(t *testing.T) {
	t.Helper()

	f.moveToProfessionTrainer(t)
	f.learnProfession(t, platform.ProfessionOrekeepingID, http.StatusOK)
}

func (f *combatFixture) moveToOreNode(t *testing.T) map[string]any {
	t.Helper()

	state := f.getWorldState(t)
	node := findGatheringNode(t, state, stonewakeOreNodeID)
	return f.moveToPosition(t, node["x"].(float64)-1.0, node["y"].(float64)-1.0)
}

func (f *combatFixture) gatherNode(t *testing.T, nodeID string, expectedStatus int) map[string]any {
	t.Helper()

	var state map[string]any
	target := any(&state)
	if expectedStatus >= http.StatusBadRequest {
		target = nil
	}
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/gather", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
		"nodeId":            nodeID,
	}, expectedStatus, target)
	return state
}

func (f *combatFixture) reconnect(t *testing.T) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": f.worldSessionToken,
	}, http.StatusOK, &state)
	return state
}

func findGatheringNode(t *testing.T, state map[string]any, nodeID string) map[string]any {
	t.Helper()

	entities, ok := state["entities"].([]any)
	if !ok {
		t.Fatalf("state response missing entities: %#v", state["entities"])
	}
	for _, entityValue := range entities {
		entity, ok := entityValue.(map[string]any)
		if !ok {
			continue
		}
		if entity["id"] == nodeID {
			if entity["kind"] != "gathering_node" {
				t.Fatalf("expected gathering_node kind, got %#v", entity)
			}
			return entity
		}
	}
	t.Fatalf("expected gathering node %s in state, got %#v", nodeID, entities)
	return nil
}

func fillInventoryWithBlades(t *testing.T, fixture *combatFixture) {
	t.Helper()

	fullInventory := make([]platform.CharacterInventorySlot, platform.InventorySlotCount)
	for slotIndex := range fullInventory {
		fullInventory[slotIndex] = platform.CharacterInventorySlot{
			SlotIndex:   slotIndex,
			ItemID:      wornMilitiaBladeID,
			DisplayName: "Worn Militia Blade",
			StackCount:  1,
		}
	}
	if _, err := fixture.fileStore.UpdateCharacterInventory(fixture.characterID, fullInventory); err != nil {
		t.Fatalf("failed to fill inventory for gathering rejection test: %v", err)
	}
}

func freeInventorySlot(t *testing.T, fixture *combatFixture, slotIndex int) {
	t.Helper()

	character, err := fixture.fileStore.GetCharacterByID(fixture.characterID)
	if err != nil {
		t.Fatalf("failed to read character inventory: %v", err)
	}
	inventory := platform.NormalizeInventorySlots(character.Inventory)
	if slotIndex < 0 || slotIndex >= len(inventory) {
		t.Fatalf("inventory slot %d is out of range", slotIndex)
	}
	inventory[slotIndex] = platform.CharacterInventorySlot{SlotIndex: slotIndex}
	if _, err := fixture.fileStore.UpdateCharacterInventory(fixture.characterID, inventory); err != nil {
		t.Fatalf("failed to free inventory slot for gathering retry: %v", err)
	}
}

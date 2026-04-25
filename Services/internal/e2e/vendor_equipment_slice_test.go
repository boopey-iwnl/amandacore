package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

const (
	vendorQuartermasterMiraID = "vendor_quartermaster_mira"
	npcQuartermasterMiraID    = "npc_quartermaster_mira_vale"
	itemWornMilitiaBladeID    = "worn_militia_blade"
	itemFieldBootsID          = "field_boots"
	itemRoadRationID          = "road_ration"
)

func TestVendorEquipmentBackendSlice(t *testing.T) {
	t.Run("Quartermaster Mira exposes a vendor service", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := fixture.getWorldState(t)
		mira := findVisibleEntityByID(t, state, npcQuartermasterMiraID)
		assertEntityService(t, mira, "vendor", vendorQuartermasterMiraID)
	})

	t.Run("buy grants an item and sell updates inventory and currency", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := targetQuartermasterMira(t, fixture)
		assertVendorPayload(t, state)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemRoadRationID,
			"stackCount":        2,
		}, http.StatusOK, &state)
		assertCurrencyCopper(t, state, 115)
		roadRationSlot := assertInventoryItemCount(t, state, itemRoadRationID, 2)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/sell", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"slotIndex":         roadRationSlot,
			"stackCount":        1,
		}, http.StatusOK, &state)
		assertCurrencyCopper(t, state, 117)
		assertInventoryItemCount(t, state, itemRoadRationID, 1)
	})

	t.Run("buy rejects insufficient funds", func(t *testing.T) {
		fixture := newTrainerFixture(t, trainerFixtureOptions{startingCopper: 5})
		state := targetQuartermasterMira(t, fixture)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemFieldBootsID,
			"stackCount":        1,
		}, http.StatusBadRequest, nil)

		state = fixture.getWorldState(t)
		assertCurrencyCopper(t, state, 5)
		assertInventoryItemAbsent(t, state, itemFieldBootsID)
	})

	t.Run("equip accepts valid Warrior gear and rejects invalid items", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := targetQuartermasterMira(t, fixture)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemRoadRationID,
			"stackCount":        1,
		}, http.StatusOK, &state)
		roadRationSlot := assertInventoryItemCount(t, state, itemRoadRationID, 1)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/inventory/equip", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"slotIndex":         roadRationSlot,
		}, http.StatusBadRequest, nil)
		state = fixture.getWorldState(t)
		assertEquipmentSlot(t, state, platform.EquipmentSlotMainHand, "")

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemWornMilitiaBladeID,
			"stackCount":        1,
		}, http.StatusOK, &state)
		bladeSlot := assertInventoryItemCount(t, state, itemWornMilitiaBladeID, 1)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/inventory/equip", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"slotIndex":         bladeSlot,
		}, http.StatusOK, &state)
		assertEquipmentSlot(t, state, platform.EquipmentSlotMainHand, itemWornMilitiaBladeID)
		assertInventoryItemAbsent(t, state, itemWornMilitiaBladeID)
	})

	t.Run("equipment inventory and currency persist after reconnect and restart", func(t *testing.T) {
		fixture := newCombatFixture(t)
		state := targetQuartermasterMira(t, fixture)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemRoadRationID,
			"stackCount":        2,
		}, http.StatusOK, &state)
		roadRationSlot := assertInventoryItemCount(t, state, itemRoadRationID, 2)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/sell", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"slotIndex":         roadRationSlot,
			"stackCount":        1,
		}, http.StatusOK, &state)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/vendor/buy", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"vendorId":          vendorQuartermasterMiraID,
			"itemId":            itemWornMilitiaBladeID,
			"stackCount":        1,
		}, http.StatusOK, &state)
		bladeSlot := assertInventoryItemCount(t, state, itemWornMilitiaBladeID, 1)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/inventory/equip", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
			"slotIndex":         bladeSlot,
		}, http.StatusOK, &state)
		assertVendorEquipmentPersistedState(t, state)

		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/disconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, nil)
		postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/reconnect", nil, map[string]any{
			"worldSessionToken": fixture.worldSessionToken,
		}, http.StatusOK, &state)
		assertVendorEquipmentPersistedState(t, state)

		restarted := restartVendorEquipmentFixture(t, fixture)
		state = restarted.getWorldState(t)
		assertVendorEquipmentPersistedState(t, state)
	})
}

func restartVendorEquipmentFixture(t *testing.T, original *combatFixture) *combatFixture {
	t.Helper()

	original.server.Close()

	restartedStore, err := store.NewFileStore(original.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to reopen store after restart: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, restartedStore)
	realms.RegisterRoutes(mux, restartedStore)
	worlds.RegisterRoutes(mux, restartedStore)

	serverAfterRestart := httptest.NewServer(mux)
	t.Cleanup(serverAfterRestart.Close)

	var loginResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/auth/login", nil, map[string]string{
		"username": original.username,
		"password": original.password,
	}, http.StatusOK, &loginResponse)
	accessToken := loginResponse["accessToken"].(string)

	var ticketResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/join-ticket", bearer(accessToken), map[string]string{
		"realmId":     original.realmID,
		"characterId": original.characterID,
	}, http.StatusCreated, &ticketResponse)

	var connectResponse map[string]any
	postJSON(t, serverAfterRestart.Client(), serverAfterRestart.URL+"/v1/world/connect", nil, map[string]string{
		"ticketId": ticketResponse["ticketId"].(string),
	}, http.StatusCreated, &connectResponse)

	return &combatFixture{
		server:            serverAfterRestart,
		storePath:         original.storePath,
		username:          original.username,
		password:          original.password,
		realmID:           original.realmID,
		characterID:       original.characterID,
		worldSessionToken: connectResponse["worldSessionToken"].(string),
	}
}

func targetQuartermasterMira(t *testing.T, fixture *combatFixture) map[string]any {
	t.Helper()

	state := fixture.getWorldState(t)
	mira := findVisibleEntityByID(t, state, npcQuartermasterMiraID)
	state = fixture.moveToPosition(t, mira["x"].(float64)-1.0, mira["y"].(float64)-1.0)
	return fixture.targetFriendlyByID(t, npcQuartermasterMiraID)
}

func assertVendorPayload(t *testing.T, state map[string]any) {
	t.Helper()

	vendor, ok := state["vendor"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing vendor payload: %#v", state["vendor"])
	}
	if vendor["id"].(string) != vendorQuartermasterMiraID {
		t.Fatalf("expected Quartermaster Mira vendor payload, got %#v", vendor)
	}
	if vendor["inRange"] != true {
		t.Fatalf("expected vendor to be in range, got %#v", vendor)
	}
	offers, ok := vendor["offers"].([]any)
	if !ok || len(offers) < 3 {
		t.Fatalf("expected vendor offers, got %#v", vendor["offers"])
	}
}

func findVisibleEntityByID(t *testing.T, state map[string]any, entityID string) map[string]any {
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
		if entity["id"] == entityID {
			return entity
		}
	}
	t.Fatalf("expected entity %s in world state, got %#v", entityID, entities)
	return nil
}

func assertVendorEquipmentPersistedState(t *testing.T, state map[string]any) {
	t.Helper()

	assertCurrencyCopper(t, state, 93)
	assertInventoryItemCount(t, state, itemRoadRationID, 1)
	assertInventoryItemAbsent(t, state, itemWornMilitiaBladeID)
	assertEquipmentSlot(t, state, platform.EquipmentSlotMainHand, itemWornMilitiaBladeID)
}

func assertCurrencyCopper(t *testing.T, state map[string]any, expected int) {
	t.Helper()

	if int(state["currencyCopper"].(float64)) != expected {
		t.Fatalf("expected %d copper, got %v", expected, state["currencyCopper"])
	}
}

func assertInventoryItemCount(t *testing.T, state map[string]any, itemID string, expectedCount int) int {
	t.Helper()

	inventory := state["inventory"].(map[string]any)
	slots := inventory["slots"].([]any)
	totalCount := 0
	firstSlot := -1
	for index, slotValue := range slots {
		slot := slotValue.(map[string]any)
		if slot["itemId"].(string) != itemID {
			continue
		}
		if firstSlot < 0 {
			firstSlot = index
		}
		totalCount += int(slot["stackCount"].(float64))
	}

	if totalCount != expectedCount {
		t.Fatalf("expected inventory to contain %s x%d, got x%d in %#v", itemID, expectedCount, totalCount, slots)
	}
	return firstSlot
}

func assertInventoryItemAbsent(t *testing.T, state map[string]any, itemID string) {
	t.Helper()

	assertInventoryItemCount(t, state, itemID, 0)
}

func assertEquipmentSlot(t *testing.T, state map[string]any, equipmentSlot string, expectedItemID string) {
	t.Helper()

	equipment, ok := state["equipment"].(map[string]any)
	if !ok {
		t.Fatalf("state response missing equipment payload: %#v", state["equipment"])
	}
	slots, ok := equipment["slots"].([]any)
	if !ok {
		t.Fatalf("equipment payload missing slots: %#v", equipment["slots"])
	}
	for _, slotValue := range slots {
		slot := slotValue.(map[string]any)
		if slot["slot"].(string) != equipmentSlot {
			continue
		}
		if slot["itemId"].(string) != expectedItemID {
			t.Fatalf("expected equipment slot %s to hold %s, got %#v", equipmentSlot, expectedItemID, slot)
		}
		return
	}
	t.Fatalf("expected equipment slot %s in %#v", equipmentSlot, slots)
}

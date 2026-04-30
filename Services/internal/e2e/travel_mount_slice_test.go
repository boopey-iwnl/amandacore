package e2e

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"amandacore/services/internal/store"
)

const (
	travelMountStonewakeZoneID = "stonewake_vale"
	travelMountSecondZoneID    = "brindlebrook_roadlands"

	travelMountCommanderID = "npc_commander_elian_rook"
	travelMountCaptainID   = "npc_bb_captain_mara_voss"

	travelMountHearthwatchBindID = "bind_hearthwatch_yard"
	travelMountHighmereBindID    = "bind_highmere_crossing"

	travelMountHearthwatchPointID = "travel_hearthwatch_yard"
	travelMountHighmerePointID    = "travel_highmere_crossing"

	travelMountRoadstepperID = "mount_stonewake_roadstepper"
)

type travelMountFixture struct {
	server    *httptest.Server
	fileStore *store.FileStore
	storePath string
	realmID   string
	player    socialPlayer
}

func TestTravelMountSliceBindRecallTravelMountAndPersistence(t *testing.T) {
	fixture := newTravelMountFixture(t, "travel_mount_user", "Traveler")

	initialTravel := fixture.travelState(t)
	assertBindLocation(t, initialTravel, travelMountHearthwatchBindID)
	assertTravelPointDiscovered(t, initialTravel, travelMountHearthwatchPointID, true)
	assertTravelPointDiscovered(t, initialTravel, travelMountHighmerePointID, false)

	fixture.target(t, travelMountCommanderID)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/travel/route", nil, map[string]any{
		"worldSessionToken":  fixture.player.worldSessionToken,
		"sourcePointId":      travelMountHearthwatchPointID,
		"destinationPointId": travelMountHighmerePointID,
	}, http.StatusBadRequest, nil)

	fixture.placePlayer(t, travelMountSecondZoneID, 150, 160)
	fixture.target(t, travelMountCaptainID)

	var state map[string]any
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/bind/set", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"bindLocationId":    travelMountHighmereBindID,
	}, http.StatusOK, &state)
	assertBindLocation(t, state["travel"].(map[string]any), travelMountHighmereBindID)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/travel/discover", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"travelPointId":     travelMountHighmerePointID,
	}, http.StatusOK, &state)
	assertTravelPointDiscovered(t, state["travel"].(map[string]any), travelMountHighmerePointID, true)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/move", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"deltaX":            8,
		"deltaY":            5,
	}, http.StatusOK, &state)
	assertWorldPosition(t, state, travelMountSecondZoneID, 158, 165)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/recall/use", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusOK, &state)
	assertWorldPosition(t, state, travelMountSecondZoneID, 150, 160)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/recall/use", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusConflict, nil)

	fixture.target(t, travelMountCaptainID)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/travel/route", nil, map[string]any{
		"worldSessionToken":  fixture.player.worldSessionToken,
		"sourcePointId":      travelMountHighmerePointID,
		"destinationPointId": travelMountHearthwatchPointID,
	}, http.StatusOK, &state)
	assertWorldPosition(t, state, travelMountStonewakeZoneID, 232, 130)
	if copper := int(state["currencyCopper"].(float64)); copper != 105 {
		t.Fatalf("expected 105 copper after route cost, got %d", copper)
	}

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/unlock", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"mountId":           travelMountRoadstepperID,
	}, http.StatusOK, &state)
	assertMountState(t, state["mounts"].(map[string]any), travelMountRoadstepperID, true, true, false, 1.0)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/select", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"mountId":           travelMountRoadstepperID,
	}, http.StatusOK, &state)
	assertMountState(t, state["mounts"].(map[string]any), travelMountRoadstepperID, true, true, false, 1.0)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/summon", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusOK, &state)
	assertMountState(t, state["mounts"].(map[string]any), travelMountRoadstepperID, true, true, true, 1.5)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/dismiss", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusOK, &state)
	assertMountState(t, state["mounts"].(map[string]any), travelMountRoadstepperID, true, true, false, 1.0)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/summon", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusOK, &state)
	assertMountState(t, state["mounts"].(map[string]any), travelMountRoadstepperID, true, true, true, 1.5)

	restartedStore, err := store.NewFileStore(fixture.storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to restart store: %v", err)
	}
	restartedServer := newSocialServer(t, restartedStore)
	defer restartedServer.Close()

	restartedPlayer := connectExistingSocialPlayer(t, restartedServer, fixture.realmID, fixture.player)
	restartedState := getWorldState(t, restartedServer, restartedPlayer.worldSessionToken)
	restartedTravel := restartedState["travel"].(map[string]any)
	assertBindLocation(t, restartedTravel, travelMountHighmereBindID)
	assertTravelPointDiscovered(t, restartedTravel, travelMountHearthwatchPointID, true)
	assertTravelPointDiscovered(t, restartedTravel, travelMountHighmerePointID, true)
	assertMountState(t, restartedState["mounts"].(map[string]any), travelMountRoadstepperID, true, true, false, 1.0)
}

func TestTravelMountSliceCombatRestrictions(t *testing.T) {
	fixture := newTravelMountFixture(t, "travel_mount_combat_user", "TravelCombat")

	var state map[string]any
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/unlock", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"mountId":           travelMountRoadstepperID,
	}, http.StatusOK, &state)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/select", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
		"mountId":           travelMountRoadstepperID,
	}, http.StatusOK, &state)

	fixture.startCombat(t)

	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/recall/use", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusBadRequest, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/travel/route", nil, map[string]any{
		"worldSessionToken":  fixture.player.worldSessionToken,
		"sourcePointId":      travelMountHearthwatchPointID,
		"destinationPointId": travelMountHighmerePointID,
	}, http.StatusBadRequest, nil)
	postJSON(t, fixture.server.Client(), fixture.server.URL+"/v1/world/mount/summon", nil, map[string]any{
		"worldSessionToken": fixture.player.worldSessionToken,
	}, http.StatusBadRequest, nil)
}

func newTravelMountFixture(t *testing.T, username string, displayName string) *travelMountFixture {
	t.Helper()

	storePath := filepath.Join(t.TempDir(), "travel-mount-state.json")
	fileStore, err := store.NewFileStore(storePath, "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	server := newSocialServer(t, fileStore)
	t.Cleanup(server.Close)

	var realmResponse map[string][]map[string]any
	getJSON(t, server.Client(), server.URL+"/v1/realms", nil, http.StatusOK, &realmResponse)
	realmID := realmResponse["realms"][0]["id"].(string)

	player := createAndConnectSocialPlayer(t, server, realmID, username, "travel_pass", displayName)
	fixture := &travelMountFixture{
		server:    server,
		fileStore: fileStore,
		storePath: storePath,
		realmID:   realmID,
		player:    player,
	}
	fixture.boostPlayer(t, 4900, 125)
	return fixture
}

func (f *travelMountFixture) boostPlayer(t *testing.T, experience int, currencyCopper int) {
	t.Helper()

	character, err := f.fileStore.GetCharacterByID(f.player.characterID)
	if err != nil {
		t.Fatalf("failed to load travel character: %v", err)
	}
	if _, err := f.fileStore.UpdateCharacterProgression(
		f.player.characterID,
		experience,
		currencyCopper,
		character.Inventory,
		character.LearnedAbilityIDs,
		character.ActionBarSlots,
		character.Quests,
	); err != nil {
		t.Fatalf("failed to boost travel character: %v", err)
	}
	f.reconnect(t)
}

func (f *travelMountFixture) placePlayer(t *testing.T, zoneID string, x float64, y float64) {
	t.Helper()

	if _, err := f.fileStore.UpdateCharacterState(f.player.characterID, zoneID, x, y, 0); err != nil {
		t.Fatalf("failed to place travel character: %v", err)
	}
	f.reconnect(t)
}

func (f *travelMountFixture) reconnect(t *testing.T) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/reconnect", nil, map[string]any{
		"worldSessionToken": f.player.worldSessionToken,
	}, http.StatusOK, &state)
	return state
}

func (f *travelMountFixture) target(t *testing.T, targetID string) map[string]any {
	t.Helper()

	var state map[string]any
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": f.player.worldSessionToken,
		"targetId":          targetID,
	}, http.StatusOK, &state)
	return state
}

func (f *travelMountFixture) travelState(t *testing.T) map[string]any {
	t.Helper()

	var state map[string]any
	getJSON(t, f.server.Client(), f.server.URL+"/v1/world/travel/state?worldSessionToken="+f.player.worldSessionToken, nil, http.StatusOK, &state)
	return state
}

func (f *travelMountFixture) startCombat(t *testing.T) {
	t.Helper()

	state := getWorldState(t, f.server, f.player.worldSessionToken)
	mob := findHostileMobs(t, state)[0]
	position := state["position"].(map[string]any)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/move", nil, map[string]any{
		"worldSessionToken": f.player.worldSessionToken,
		"deltaX":            mob["x"].(float64) - 2 - position["x"].(float64),
		"deltaY":            mob["y"].(float64) - position["y"].(float64),
	}, http.StatusOK, nil)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/target", nil, map[string]any{
		"worldSessionToken": f.player.worldSessionToken,
		"targetId":          mob["id"].(string),
	}, http.StatusOK, nil)
	postJSON(t, f.server.Client(), f.server.URL+"/v1/world/attack/auto", nil, map[string]any{
		"worldSessionToken": f.player.worldSessionToken,
		"enabled":           true,
	}, http.StatusOK, nil)
}

func assertBindLocation(t *testing.T, travel map[string]any, expectedBindLocationID string) {
	t.Helper()

	bindPoint := travel["bindPoint"].(map[string]any)
	if bindPoint["bindLocationId"].(string) != expectedBindLocationID {
		t.Fatalf("expected bind location %s, got %#v", expectedBindLocationID, bindPoint)
	}
}

func assertTravelPointDiscovered(t *testing.T, travel map[string]any, travelPointID string, expected bool) {
	t.Helper()

	points := travel["travelPoints"].([]any)
	for _, pointValue := range points {
		point := pointValue.(map[string]any)
		if point["travelPointId"].(string) == travelPointID {
			if point["discovered"].(bool) != expected {
				t.Fatalf("expected travel point %s discovered=%v, got %#v", travelPointID, expected, point)
			}
			return
		}
	}
	t.Fatalf("expected travel point %s in %#v", travelPointID, travel["travelPoints"])
}

func assertMountState(t *testing.T, mounts map[string]any, mountID string, unlocked bool, selected bool, currentlyMounted bool, speedModifier float64) {
	t.Helper()

	if mounts["selectedMountId"].(string) != mountID && selected {
		t.Fatalf("expected selected mount %s, got %#v", mountID, mounts)
	}
	if mounts["currentlyMounted"].(bool) != currentlyMounted {
		t.Fatalf("expected currentlyMounted=%v, got %#v", currentlyMounted, mounts)
	}
	if mounts["currentSpeedModifier"].(float64) != speedModifier {
		t.Fatalf("expected speed modifier %.1f, got %#v", speedModifier, mounts)
	}
	for _, mountValue := range mounts["mounts"].([]any) {
		mount := mountValue.(map[string]any)
		if mount["mountId"].(string) != mountID {
			continue
		}
		if mount["unlocked"].(bool) != unlocked || mount["selected"].(bool) != selected || mount["currentlyMounted"].(bool) != currentlyMounted {
			t.Fatalf("unexpected mount row for %s: %#v", mountID, mount)
		}
		return
	}
	t.Fatalf("expected mount %s in %#v", mountID, mounts["mounts"])
}

func assertWorldPosition(t *testing.T, state map[string]any, zoneID string, x float64, y float64) {
	t.Helper()

	if state["zoneId"].(string) != zoneID {
		t.Fatalf("expected zone %s, got %#v", zoneID, state["zoneId"])
	}
	position := state["position"].(map[string]any)
	if position["x"].(float64) != x || position["y"].(float64) != y {
		t.Fatalf("expected position %.1f %.1f, got %#v", x, y, position)
	}
}

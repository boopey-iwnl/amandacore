package worlds

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
	worldloop "amandacore/services/internal/worlds/loop"
)

func TestStonewakeHTTPHotPathUsesAuthoritativeLoop(t *testing.T) {
	server, ticket := newStonewakeLoopTestServer(t)
	defer stopStonewakeLoop(t, server)

	connect := postWorldHandler(t, server.handleConnect, map[string]any{"ticketId": ticket.TicketID}, http.StatusCreated)
	token := connect["worldSessionToken"].(string)

	postWorldHandler(t, server.handleMove, map[string]any{
		"worldSessionToken": token,
		"deltaX":            3,
		"deltaY":            2,
	}, http.StatusOK)
	state := getWorldHandler(t, server.handleState, "/v1/world/state?worldSessionToken="+token, http.StatusOK)
	position := state["position"].(map[string]any)
	if position["x"].(float64) != 13 || position["y"].(float64) != 12 {
		t.Fatalf("expected loop-backed position 13,12 got %#v", position)
	}

	postWorldHandler(t, server.handleDisconnect, map[string]any{"worldSessionToken": token}, http.StatusOK)
	reconnected := postWorldHandler(t, server.handleReconnect, map[string]any{"worldSessionToken": token}, http.StatusOK)
	if reconnected["worldSessionToken"].(string) != token {
		t.Fatalf("expected reconnect to preserve token %s", token)
	}

	snapshot, err := server.stonewakeLoop.Snapshot(context.Background(), token, reconnected["characterId"].(string))
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("expected one Stonewake player in snapshot, got %d", len(snapshot.Players))
	}
	if snapshot.Players[0].Position.X != 13 || snapshot.Players[0].Position.Y != 12 {
		t.Fatalf("expected authoritative snapshot position 13,12 got %#v", snapshot.Players[0].Position)
	}

	metrics := server.stonewakeLoop.Metrics()
	if metrics.CommandsApplied < 5 {
		t.Fatalf("expected loop-applied connect/move/state/disconnect/reconnect commands, got %#v", metrics)
	}
	if metrics.ReplayRecords < 5 {
		t.Fatalf("expected replay records for hot path commands, got %#v", metrics)
	}
}

func TestStonewakeLoopRejectsInvalidSessionHTTPCommand(t *testing.T) {
	server, _ := newStonewakeLoopTestServer(t)
	defer stopStonewakeLoop(t, server)

	postWorldHandler(t, server.handleMove, map[string]any{
		"worldSessionToken": "world_missing",
		"deltaX":            1,
		"deltaY":            0,
	}, http.StatusNotFound)

	metrics := server.stonewakeLoop.Metrics()
	if metrics.CommandsRejected == 0 {
		t.Fatalf("expected rejected command metric, got %#v", metrics)
	}
}

func TestStonewakeGameplayCommandsUseAuthoritativeLoop(t *testing.T) {
	server, ticket := newStonewakeLoopTestServer(t)
	defer stopStonewakeLoop(t, server)

	connect := postWorldHandler(t, server.handleConnect, map[string]any{"ticketId": ticket.TicketID}, http.StatusCreated)
	token := connect["worldSessionToken"].(string)

	server.mutex.Lock()
	session := server.sessionsByToken[token]
	server.lootContainers = map[string]*lootContainerState{
		"loot_loop_001": {
			LootContainerID:   "loot_loop_001",
			SourceEntityID:    "mob_loop_rat",
			SourceArchetypeID: "loop_rat",
			LootTableID:       "loop_rat_loot",
			ZoneID:            session.ZoneID,
			X:                 session.X,
			Y:                 session.Y,
			Z:                 session.Z,
			OwnerCharacterID:  session.CharacterID,
			Items: []lootContainerItemState{{
				ItemID:      itemDevFieldRationID,
				DisplayName: "Field Ration",
				Quantity:    1,
			}},
			CreatedAtMs: nowMillis(),
			ExpiresAtMs: nowMillis() + defaultLootExpiry.Milliseconds(),
		},
	}
	server.lootContainerOrder = []string{"loot_loop_001"}
	server.mutex.Unlock()

	postWorldHandler(t, server.handleLootInspect, map[string]any{
		"worldSessionToken": token,
		"lootContainerId":   "loot_loop_001",
	}, http.StatusOK)
	state := postWorldHandler(t, server.handleLootClaim, map[string]any{
		"worldSessionToken": token,
		"lootContainerId":   "loot_loop_001",
	}, http.StatusOK)

	inventory := state["inventory"].(map[string]any)
	if !inventoryContainsItem(inventory, itemDevFieldRationID) {
		t.Fatalf("expected claimed loot in inventory, got %#v", inventory)
	}

	records := server.stonewakeLoop.ReplayLog()
	if !replayContainsKind(records, worldloop.CommandOpenLoot) {
		t.Fatalf("expected OpenLoot replay record, got %#v", records)
	}
	if !replayContainsKind(records, worldloop.CommandClaimLootItem) {
		t.Fatalf("expected ClaimLootItem replay record, got %#v", records)
	}
	metrics := server.stonewakeLoop.Metrics()
	if metrics.GameplayCommandsApplied < 2 {
		t.Fatalf("expected gameplay command metrics for loot, got %#v", metrics)
	}
}

func TestStonewakeStateResponseIncludesReplicationMetadata(t *testing.T) {
	server, ticket := newStonewakeLoopTestServer(t)
	defer stopStonewakeLoop(t, server)

	connect := postWorldHandler(t, server.handleConnect, map[string]any{"ticketId": ticket.TicketID}, http.StatusCreated)
	token := connect["worldSessionToken"].(string)
	connectCursor, ok := connect["cursor"].(string)
	if !ok || connectCursor == "" {
		t.Fatalf("expected connect cursor in response, got %#v", connect)
	}
	if connect["fullSnapshot"] != true {
		t.Fatalf("expected connect full snapshot metadata, got %#v", connect["replication"])
	}

	postWorldHandler(t, server.handleMove, map[string]any{
		"worldSessionToken": token,
		"deltaX":            2,
		"deltaY":            1,
	}, http.StatusOK)

	state := getWorldHandler(t, server.handleState, "/v1/world/state?worldSessionToken="+token+"&since="+connectCursor, http.StatusOK)
	if state["resyncRequired"] == true {
		t.Fatalf("expected retained delta, got resync metadata %#v", state["replication"])
	}
	if state["fullSnapshot"] == true {
		t.Fatalf("expected delta metadata for retained cursor, got %#v", state["replication"])
	}
	replicationBody := state["replication"].(map[string]any)
	if replicationBody["kind"] != "delta" {
		t.Fatalf("expected delta kind, got %#v", replicationBody)
	}
	if !changedResponseContains(state["changed"].([]any), "player", state["characterId"].(string), "position") {
		t.Fatalf("expected player position changed field, got %#v", state["changed"])
	}

	latestCursor := state["cursor"].(string)
	noop := getWorldHandler(t, server.handleState, "/v1/world/state?worldSessionToken="+token+"&since="+latestCursor, http.StatusOK)
	if noop["fullSnapshot"] == true || noop["resyncRequired"] == true {
		t.Fatalf("expected no-op delta metadata, got %#v", noop["replication"])
	}
	if noop["deltaVersion"] != state["deltaVersion"] {
		t.Fatalf("expected no-op poll to preserve version, got %v then %v", state["deltaVersion"], noop["deltaVersion"])
	}
}

func TestStonewakeStateRejectsInvalidReplicationCursor(t *testing.T) {
	server, ticket := newStonewakeLoopTestServer(t)
	defer stopStonewakeLoop(t, server)

	connect := postWorldHandler(t, server.handleConnect, map[string]any{"ticketId": ticket.TicketID}, http.StatusCreated)
	token := connect["worldSessionToken"].(string)

	getWorldHandler(t, server.handleState, "/v1/world/state?worldSessionToken="+token+"&since=not-a-cursor", http.StatusBadRequest)
}

func newStonewakeLoopTestServer(t *testing.T) (*worldServer, platform.WorldJoinTicket) {
	t.Helper()

	fileStore, err := store.NewFileStore(filepath.Join(t.TempDir(), "platform-state.json"), "test-build", "http://world.local")
	if err != nil {
		t.Fatalf("store create failed: %v", err)
	}
	account, err := fileStore.RegisterAccount("loop_user", "loop_pass")
	if err != nil {
		t.Fatalf("account create failed: %v", err)
	}
	session, err := fileStore.CreateSession(account.ID)
	if err != nil {
		t.Fatalf("session create failed: %v", err)
	}
	character, err := fileStore.CreateCharacter(account.ID, "sunset-frontier-dev", "LoopRunner", platform.DefaultRaceID, platform.DefaultClassID, platform.LegacyWayfarerArchetypeID)
	if err != nil {
		t.Fatalf("character create failed: %v", err)
	}
	ticket, err := fileStore.IssueWorldJoinTicket(account.ID, session.ID, character.ID, "sunset-frontier-dev")
	if err != nil {
		t.Fatalf("ticket create failed: %v", err)
	}
	return newWorldServer(fileStore), ticket
}

func replayContainsKind(records []worldloop.ReplayRecord, kind worldloop.CommandKind) bool {
	for _, record := range records {
		if record.CommandKind == kind {
			return true
		}
	}
	return false
}

func inventoryContainsItem(inventory map[string]any, itemID string) bool {
	slots, ok := inventory["slots"].([]any)
	if !ok {
		return false
	}
	for _, slotValue := range slots {
		slot, ok := slotValue.(map[string]any)
		if !ok {
			continue
		}
		if slot["itemId"] == itemID {
			return true
		}
	}
	return false
}

func changedResponseContains(changes []any, domain string, entityID string, field string) bool {
	for _, value := range changes {
		change, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if change["domain"] != domain || change["entityId"] != entityID {
			continue
		}
		fields, ok := change["fields"].([]any)
		if !ok {
			continue
		}
		for _, candidate := range fields {
			if candidate == field {
				return true
			}
		}
	}
	return false
}

func postWorldHandler(t *testing.T, handler http.HandlerFunc, payload map[string]any, expectedStatus int) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/world-test", bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	if recorder.Code != expectedStatus {
		t.Fatalf("expected status %d got %d body %s", expectedStatus, recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() == 0 {
		return nil
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response decode failed: %v", err)
	}
	return response
}

func getWorldHandler(t *testing.T, handler http.HandlerFunc, target string, expectedStatus int) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	if recorder.Code != expectedStatus {
		t.Fatalf("expected status %d got %d body %s", expectedStatus, recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("response decode failed: %v", err)
	}
	return response
}

func stopStonewakeLoop(t *testing.T, server *worldServer) {
	t.Helper()
	if server == nil || server.stonewakeLoop == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.stonewakeLoop.Stop(ctx); err != nil {
		t.Fatalf("stop loop failed: %v", err)
	}
}

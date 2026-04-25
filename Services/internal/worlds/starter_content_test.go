package worlds

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStonewakeStarterContentLoads(t *testing.T) {
	server := newWorldServer(nil)

	if len(server.questOrder) != 15 {
		t.Fatalf("expected 15 Stonewake starter quests, got %d", len(server.questOrder))
	}
	if len(server.friendlyNPCOrder) != 8 {
		t.Fatalf("expected 8 Stonewake friendly NPC/object entities, got %d", len(server.friendlyNPCOrder))
	}
	if len(server.mobOrder) != 37 {
		t.Fatalf("expected 37 Stonewake mob spawns, got %d", len(server.mobOrder))
	}

	requiredQuests := []string{
		"sv_first_muster",
		"sv_yard_drills",
		"sv_wall_rats",
		"sv_scattered_kit",
		"sv_stronger_lesson",
		"sv_light_the_lantern",
		"sv_bram_kettles_stand",
		"sv_westward_orders",
	}
	for _, questID := range requiredQuests {
		if _, ok := server.quests[questID]; !ok {
			t.Fatalf("expected quest %s to be loaded", questID)
		}
	}

	trainer := server.friendlyNPCs[warriorTrainerID]
	if trainer.ID != warriorTrainerID || trainer.Kind != trainerNPCKind {
		t.Fatalf("expected warrior trainer NPC, got %#v", trainer)
	}
	if len(trainer.Services) != 2 {
		t.Fatalf("expected warrior trainer to expose trainer and quest services, got %#v", trainer.Services)
	}

	finalQuest := server.quests["sv_westward_orders"]
	if finalQuest.TurnInNPCID != npcQuartermasterLyraID {
		t.Fatalf("expected final handoff to Quartermaster Lyra, got %s", finalQuest.TurnInNPCID)
	}
	if finalQuest.RewardXP != 100 || finalQuest.RewardCopper != 60 {
		t.Fatalf("unexpected final quest rewards: %#v", finalQuest)
	}
}

func TestBootstrapMapsStonewakeAsSunsetFrontierCell(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, nil)

	request := httptest.NewRequest(http.MethodGet, "/v1/world/bootstrap", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected bootstrap status 200, got %d", response.Code)
	}

	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode bootstrap response: %v", err)
	}
	if payload["zoneId"] != "sunset_frontier" {
		t.Fatalf("expected broad zone sunset_frontier, got %#v", payload["zoneId"])
	}
	if payload["cellId"] != "stonewake_vale" {
		t.Fatalf("expected Stonewake playable cell, got %#v", payload["cellId"])
	}
}

package worlds

import (
	"math"
	"testing"
)

func TestStonewakeStarterContentLoads(t *testing.T) {
	server := newWorldServer(nil)

	stonewakeQuestCount := 0
	for _, questID := range server.questOrder {
		if server.quests[questID].ZoneID == defaultZoneID {
			stonewakeQuestCount++
		}
	}
	if stonewakeQuestCount != 15 {
		t.Fatalf("expected 15 Stonewake starter quests, got %d", stonewakeQuestCount)
	}

	stonewakeFriendlyCount := 0
	for _, npcID := range server.friendlyNPCOrder {
		npc := server.friendlyNPCs[npcID]
		if npc.ZoneID == defaultZoneID && npc.Kind != professionTrainerNPCKind {
			stonewakeFriendlyCount++
		}
	}
	if stonewakeFriendlyCount != 9 {
		t.Fatalf("expected 9 Stonewake friendly NPC/object entities, got %d", stonewakeFriendlyCount)
	}

	stonewakeMobCount := 0
	for _, mobID := range server.mobOrder {
		if server.mobs[mobID].ZoneID == defaultZoneID {
			stonewakeMobCount++
		}
	}
	if stonewakeMobCount != 38 {
		t.Fatalf("expected 38 Stonewake mob spawns, got %d", stonewakeMobCount)
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

func TestStonewakeLayoutIsReadableFromSpawn(t *testing.T) {
	server := newWorldServer(nil)

	for _, npcID := range []string{npcCommanderElianRookID, warriorTrainerID, npcQuartermasterMiraID, npcHealerSellaID} {
		npc := server.friendlyNPCs[npcID]
		if distance := math.Hypot(npc.X-starterSpawnX, npc.Y-starterSpawnY); distance > 44.0 {
			t.Fatalf("expected hub NPC %s near spawn, distance %.1f", npcID, distance)
		}
	}

	for _, npcID := range []string{npcScoutRowanID, npcRoadwardenIlyaID, objWatchLanternID, npcQuartermasterLyraID} {
		npc := server.friendlyNPCs[npcID]
		if distance := math.Hypot(npc.X-starterSpawnX, npc.Y-starterSpawnY); distance < 80.0 {
			t.Fatalf("expected progression NPC %s outside spawn view, distance %.1f", npcID, distance)
		}
	}

	for _, mob := range server.mobs {
		if distance := math.Hypot(mob.X-starterSpawnX, mob.Y-starterSpawnY); distance < 28.0 {
			t.Fatalf("expected no hostile/training mobs inside the starter hub, got %s at %.1fm", mob.ID, distance)
		}
	}

	assertMobAreaCenter(t, server, mobTrainingDummyTypeID, 58.0, 32.0, 10.0)
	assertMobAreaCenter(t, server, mobDitchRatTypeID, 119.0, 62.0, 24.0)
	assertMobAreaCenter(t, server, mobFieldBoarTypeID, 163.0, 86.0, 32.0)
	assertMobAreaCenter(t, server, mobRidgeCrowTypeID, 240.0, 127.0, 34.0)
	assertMobAreaCenter(t, server, mobAshbandScoutTypeID, 313.0, 170.0, 34.0)
	assertMobAreaCenter(t, server, mobAshbandPoacherTypeID, 375.0, 210.0, 42.0)

	bram := server.mobs["mob_bram_kettle_01"]
	if distance := math.Hypot(bram.X-420.0, bram.Y-224.0); distance > 1.0 {
		t.Fatalf("expected Bram Kettle in isolated wagon stand, got %.1f, %.1f", bram.X, bram.Y)
	}
}

func TestStonewakeSpawnsAreGrounded(t *testing.T) {
	server := newWorldServer(nil)

	for _, npc := range server.friendlyNPCs {
		if npc.ZoneID == defaultZoneID && npc.Z < playableGroundZ {
			t.Fatalf("expected friendly NPC %s to be grounded at %.2f or above, got %.2f", npc.ID, playableGroundZ, npc.Z)
		}
	}
	for _, mob := range server.mobs {
		if mob.ZoneID == defaultZoneID && (mob.Z < playableGroundZ || mob.SpawnZ < playableGroundZ) {
			t.Fatalf("expected mob %s to be grounded at %.2f or above, got z %.2f spawnZ %.2f", mob.ID, playableGroundZ, mob.Z, mob.SpawnZ)
		}
	}
	for _, node := range server.gatheringNodes {
		if node.Definition.ZoneID == defaultZoneID && node.Definition.Z < playableGroundZ {
			t.Fatalf("expected gathering node %s to be grounded at %.2f or above, got %.2f", node.Definition.ID, playableGroundZ, node.Definition.Z)
		}
	}
}

func assertMobAreaCenter(t *testing.T, server *worldServer, mobTypeID string, centerX float64, centerY float64, maxRadius float64) {
	t.Helper()

	count := 0
	for _, mob := range server.mobs {
		if mob.MobTypeID != mobTypeID {
			continue
		}
		count++
		if distance := math.Hypot(mob.X-centerX, mob.Y-centerY); distance > maxRadius {
			t.Fatalf("expected %s near %.1f,%.1f within %.1f, got %s at %.1f,%.1f distance %.1f", mobTypeID, centerX, centerY, maxRadius, mob.ID, mob.X, mob.Y, distance)
		}
	}
	if count == 0 {
		t.Fatalf("expected at least one %s spawn", mobTypeID)
	}
}

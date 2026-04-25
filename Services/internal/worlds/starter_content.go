package worlds

import "amandacore/services/internal/platform"

const (
	npcCommanderElianRookID = "npc_commander_elian_rook"
	npcQuartermasterMiraID  = "npc_quartermaster_mira_vale"
	npcHealerSellaID        = "npc_healer_sella_wren"
	npcScoutRowanID         = "npc_scout_rowan_bell"
	npcRoadwardenIlyaID     = "npc_roadwarden_ilya_brant"
	npcQuartermasterLyraID  = "npc_quartermaster_lyra"
	objWatchLanternID       = "obj_watch_lantern"
	mobTrainingDummyTypeID  = "training_dummy"
	mobDitchRatTypeID       = "ditch_rat"
	mobFieldBoarTypeID      = "field_boar"
	mobRidgeCrowTypeID      = "ridge_crow"
	mobAshbandScoutTypeID   = "ashband_scout"
	mobAshbandPoacherTypeID = "ashband_poacher"
	mobBramKettleTypeID     = "bram_kettle"
	itemLooseKitID          = "loose_kit"
	itemFieldDressingID     = "field_dressing"
	itemRoadRationID        = "road_ration"
	itemOatBundleID         = "oat_bundle"
	itemTornClothID         = "torn_cloth"
	itemLinenWrapID         = "linen_wrap"
	itemMilitiaTokenID      = "militia_token"
)

var stonewakeFriendlyNPCs = []friendlyNPCDefinition{
	{
		ID:          npcCommanderElianRookID,
		DisplayName: "Commander Elian Rook",
		Kind:        questGiverNPCKind,
		X:           13.0,
		Y:           10.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: starterQuestID, Label: "Hearthwatch Orders"},
		},
	},
	{
		ID:          warriorTrainerID,
		DisplayName: "Armsmaster Corin Vale",
		Kind:        trainerNPCKind,
		X:           34.0,
		Y:           18.0,
		Z:           0.0,
		AIState:     "trainer",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "trainer", ServiceID: warriorTrainerID, Label: "Warrior Training"},
			{Type: "quest", ServiceID: "sv_yard_drills", Label: "Yard Drills"},
		},
	},
	{
		ID:          npcQuartermasterMiraID,
		DisplayName: "Quartermaster Mira Vale",
		Kind:        questGiverNPCKind,
		X:           7.0,
		Y:           24.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_scattered_kit", Label: "Supplies"},
		},
	},
	{
		ID:          npcHealerSellaID,
		DisplayName: "Sella Wren",
		Kind:        questGiverNPCKind,
		X:           21.0,
		Y:           25.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_aid_for_the_hurt", Label: "Field Aid"},
		},
	},
	{
		ID:          npcScoutRowanID,
		DisplayName: "Scout Rowan Bell",
		Kind:        questGiverNPCKind,
		X:           134.0,
		Y:           64.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_oatfield_tusks", Label: "Field Reports"},
		},
	},
	{
		ID:          npcRoadwardenIlyaID,
		DisplayName: "Roadwarden Ilya Brant",
		Kind:        questGiverNPCKind,
		X:           246.0,
		Y:           126.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_smoke_at_old_mill", Label: "Road Watch"},
		},
	},
	{
		ID:          objWatchLanternID,
		DisplayName: "Watch Lantern",
		Kind:        questGiverNPCKind,
		X:           322.0,
		Y:           174.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      4.0,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_light_the_lantern", Label: "Signal Lantern"},
		},
	},
	{
		ID:          npcQuartermasterLyraID,
		DisplayName: "Quartermaster Lyra",
		Kind:        questGiverNPCKind,
		X:           438.0,
		Y:           246.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "sv_westward_orders", Label: "Westward Orders"},
		},
	},
}

var stonewakeQuestDefinitions = []questDefinition{
	{ID: "sv_first_muster", Title: "First Muster", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Armsmaster Corin Vale at the training ring east of Hearthwatch Yard.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: warriorTrainerID, TargetEntityID: warriorTrainerID, TargetCount: 1, RewardXP: 35, RewardCopper: 5, LevelBand: "1"},
	{ID: "sv_yard_drills", Title: "Yard Drills", ObjectiveType: objectiveKill, ObjectiveText: "Break 3 Yard Dummies in the training ring.", GiverNPCID: warriorTrainerID, TurnInNPCID: warriorTrainerID, TargetMobType: mobTrainingDummyTypeID, TargetCount: 3, RewardXP: 45, RewardCopper: 5, PrerequisiteIDs: []string{"sv_first_muster"}, LevelBand: "1"},
	{ID: "sv_wall_rats", Title: "Rats at the Wall", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Ditch Rats at the Ditch Wall east of Hearthwatch Yard.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobDitchRatTypeID, TargetCount: 6, RewardXP: 55, RewardCopper: 10, PrerequisiteIDs: []string{"sv_yard_drills"}, LevelBand: "1"},
	{ID: "sv_scattered_kit", Title: "Scattered Kit", ObjectiveType: objectiveCollect, ObjectiveText: "Recover 4 Loose Kit bundles from Ditch Rats.", GiverNPCID: npcQuartermasterMiraID, TurnInNPCID: npcQuartermasterMiraID, TargetMobType: mobDitchRatTypeID, TargetItemID: itemLooseKitID, TargetItemName: "Loose Kit", TargetCount: 4, RewardXP: 60, RewardCopper: 10, RewardItems: []itemRewardDefinition{{ItemID: itemFieldDressingID, DisplayName: "Field Dressing", StackCount: 1}}, PrerequisiteIDs: []string{"sv_wall_rats"}, LevelBand: "1-2"},
	{ID: "sv_aid_for_the_hurt", Title: "Aid for the Hurt", ObjectiveType: objectiveTalk, ObjectiveText: "Bring Mira's field dressing to Sella Wren.", GiverNPCID: npcHealerSellaID, TurnInNPCID: npcHealerSellaID, TargetEntityID: npcHealerSellaID, TargetCount: 1, RewardXP: 45, RewardCopper: 5, PrerequisiteIDs: []string{"sv_scattered_kit"}, LevelBand: "2"},
	{ID: "sv_oatfield_tusks", Title: "Oatfield Tusks", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 7 Oatfield Boars beyond Scout Rowan Bell's field post.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcScoutRowanID, TargetMobType: mobFieldBoarTypeID, TargetCount: 7, RewardXP: 75, RewardCopper: 15, RewardItems: []itemRewardDefinition{{ItemID: itemRoadRationID, DisplayName: "Road Ration", StackCount: 1}}, PrerequisiteIDs: []string{"sv_aid_for_the_hurt"}, LevelBand: "2"},
	{ID: "sv_bundles_in_furrows", Title: "Bundles in the Furrows", ObjectiveType: objectiveCollect, ObjectiveText: "Gather 5 Oat Bundles from the spaced furrows and boars.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcQuartermasterMiraID, TargetMobType: mobFieldBoarTypeID, TargetItemID: itemOatBundleID, TargetItemName: "Oat Bundle", TargetCount: 5, RewardXP: 75, RewardCopper: 15, PrerequisiteIDs: []string{"sv_oatfield_tusks"}, LevelBand: "2-3"},
	{ID: "sv_stronger_lesson", Title: "A Stronger Lesson", ObjectiveType: objectiveTrainer, ObjectiveText: "Learn Driving Blow from Armsmaster Corin Vale.", GiverNPCID: warriorTrainerID, TurnInNPCID: warriorTrainerID, TargetEntityID: platform.DrivingBlowAbilityID, TargetCount: 1, RewardXP: 60, RewardCopper: 10, PrerequisiteIDs: []string{"sv_bundles_in_furrows"}, LevelBand: "3"},
	{ID: "sv_roadside_marks", Title: "Roadside Marks", ObjectiveType: objectiveExplore, ObjectiveText: "Follow the ridge track marker beyond the oatfield and report to Roadwarden Ilya Brant.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcRoadwardenIlyaID, TargetEntityID: npcRoadwardenIlyaID, TargetCount: 1, RewardXP: 70, RewardCopper: 20, PrerequisiteIDs: []string{"sv_stronger_lesson"}, LevelBand: "3", MarkerX: 232.0, MarkerY: 118.0},
	{ID: "sv_crows_and_cloth", Title: "Crows and Cloth", ObjectiveType: objectiveCollect, ObjectiveText: "Collect 6 Torn Cloth scraps from Ridge Crows.", GiverNPCID: npcHealerSellaID, TurnInNPCID: npcHealerSellaID, TargetMobType: mobRidgeCrowTypeID, TargetItemID: itemTornClothID, TargetItemName: "Torn Cloth", TargetCount: 6, RewardXP: 85, RewardCopper: 20, RewardItems: []itemRewardDefinition{{ItemID: itemLinenWrapID, DisplayName: "Linen Wrap", StackCount: 1}}, PrerequisiteIDs: []string{"sv_roadside_marks"}, LevelBand: "3-4"},
	{ID: "sv_smoke_at_old_mill", Title: "Smoke at the Old Mill", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Ashband Scouts around the old mill.", GiverNPCID: npcRoadwardenIlyaID, TurnInNPCID: npcRoadwardenIlyaID, TargetMobType: mobAshbandScoutTypeID, TargetCount: 6, RewardXP: 95, RewardCopper: 25, PrerequisiteIDs: []string{"sv_crows_and_cloth"}, LevelBand: "4"},
	{ID: "sv_light_the_lantern", Title: "Light the Lantern", ObjectiveType: objectiveUse, ObjectiveText: "Use the Watch Lantern near the old mill road.", GiverNPCID: npcRoadwardenIlyaID, TurnInNPCID: npcRoadwardenIlyaID, TargetEntityID: objWatchLanternID, TargetCount: 1, RewardXP: 90, RewardCopper: 20, PrerequisiteIDs: []string{"sv_smoke_at_old_mill"}, LevelBand: "4"},
	{ID: "sv_break_poacher_line", Title: "Break the Poacher Line", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 8 Ashband Poachers on the road beyond the old mill.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobAshbandPoacherTypeID, TargetCount: 8, RewardXP: 110, RewardCopper: 35, PrerequisiteIDs: []string{"sv_light_the_lantern"}, LevelBand: "5"},
	{ID: "sv_bram_kettles_stand", Title: "Bram Kettle's Stand", ObjectiveType: objectiveKill, ObjectiveText: "Defeat Bram Kettle at the isolated broken wagon stand before the westward gate.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobBramKettleTypeID, TargetCount: 1, RewardXP: 125, RewardCopper: 50, RewardItems: []itemRewardDefinition{{ItemID: itemMilitiaTokenID, DisplayName: "Militia Token", StackCount: 1}}, PrerequisiteIDs: []string{"sv_break_poacher_line"}, LevelBand: "5"},
	{ID: "sv_westward_orders", Title: "Westward Orders", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Quartermaster Lyra at the westward gate beyond Bram's wagon stand.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcQuartermasterLyraID, TargetEntityID: npcQuartermasterLyraID, TargetCount: 1, RewardXP: 100, RewardCopper: 60, RewardItems: []itemRewardDefinition{{ItemID: itemRoadRationID, DisplayName: "Road Ration", StackCount: 2}}, PrerequisiteIDs: []string{"sv_bram_kettles_stand"}, LevelBand: "5-6"},
}

var stonewakeMobSpawns = []mobSpawnDefinition{
	{ID: "mob_training_dummy_01", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 30.0, Y: 16.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_02", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 34.0, Y: 18.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_03", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 31.5, Y: 22.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_ditch_rat_01", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 49.0, Y: 26.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_02", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 55.0, Y: 29.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_03", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 61.0, Y: 26.5, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_04", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 64.0, Y: 34.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_05", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 50.0, Y: 36.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_06", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 57.0, Y: 38.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_field_boar_01", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 82.0, Y: 43.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_02", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 90.0, Y: 47.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_03", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 99.0, Y: 43.5, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_04", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 84.0, Y: 55.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_05", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 97.0, Y: 56.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_06", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 78.0, Y: 50.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_07", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 92.0, Y: 63.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_ridge_crow_01", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 112.0, Y: 58.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_02", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 120.0, Y: 62.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_03", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 130.0, Y: 59.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_04", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 113.0, Y: 72.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_05", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 130.0, Y: 73.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_06", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 122.0, Y: 80.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ashband_scout_01", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 146.0, Y: 82.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_02", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 154.0, Y: 84.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_03", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 164.0, Y: 82.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_04", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 149.0, Y: 96.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_05", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 166.0, Y: 96.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_06", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 157.0, Y: 104.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_poacher_01", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 178.0, Y: 100.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_02", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 188.0, Y: 105.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_03", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 200.0, Y: 110.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_04", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 181.0, Y: 118.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_05", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 198.0, Y: 123.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_06", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 210.0, Y: 124.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_07", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 190.0, Y: 132.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_08", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 214.0, Y: 112.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_bram_kettle_01", MobTypeID: mobBramKettleTypeID, DisplayName: "Bram Kettle", Level: 5, X: 222.0, Y: 118.0, MaxHealth: 180, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 30000},
}

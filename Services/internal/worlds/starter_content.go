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
	itemValeIronChipID      = "vale_iron_chip"
	itemWornRivetID         = "worn_rivet"
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
			{Type: "vendor", ServiceID: vendorQuartermasterMiraID, Label: "Starter Supplies"},
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
		ID:          npcProfessionTrainerTallaID,
		DisplayName: "Talla Grayspark",
		Kind:        professionTrainerNPCKind,
		X:           58.0,
		Y:           24.0,
		Z:           0.0,
		AIState:     "profession_trainer",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: professionTrainerServiceType, ServiceID: professionTrainerTallaID, Label: "Starter Professions"},
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

var stonewakeZoneMap = zoneMapDefinition{
	ZoneID:      defaultZoneID,
	DisplayName: "Stonewake Vale",
	MinX:        0,
	MinY:        0,
	MaxX:        460,
	MaxY:        270,
	Roads: []mapRoadDefinition{
		{
			ID:          "stonewake_main_road",
			DisplayName: "Westward Road",
			Points: []mapPointDefinition{
				{X: 10, Y: 12},
				{X: 42, Y: 22},
				{X: 98, Y: 45},
				{X: 150, Y: 76},
				{X: 232, Y: 118},
				{X: 314, Y: 174},
				{X: 420, Y: 224},
				{X: 438, Y: 246},
			},
		},
	},
	Landmarks: []mapLandmarkDefinition{
		{ID: "hearthwatch_yard", DisplayName: "Hearthwatch Yard", Kind: "hub", X: 13, Y: 10},
		{ID: "training_ring", DisplayName: "Training Ring", Kind: "training", X: 42, Y: 22},
		{ID: "ditch_wall", DisplayName: "Ditch Wall", Kind: "objective", X: 98, Y: 45},
		{ID: "oatfield_post", DisplayName: "Oatfield Post", Kind: "objective", X: 165, Y: 88},
		{ID: "ridge_track", DisplayName: "Ridge Track", Kind: "route", X: 232, Y: 118},
		{ID: "old_mill_road", DisplayName: "Old Mill Road", Kind: "objective", X: 314, Y: 174},
		{ID: "broken_wagon_stand", DisplayName: "Broken Wagon Stand", Kind: "objective", X: 420, Y: 224},
		{ID: "westward_gate", DisplayName: "Westward Gate", Kind: "handoff", X: 438, Y: 246},
	},
}

var stonewakeNavigationAreas = []navigationAreaDefinition{
	{ID: "hearthwatch_yard", DisplayName: "Hearthwatch Yard", Kind: "hub", CenterX: 13, CenterY: 10, Radius: 22, RouteHintText: "Return to the Hearthwatch Yard command post.", QuestIDs: []string{"sv_first_muster", "sv_wall_rats", "sv_break_poacher_line", "sv_bram_kettles_stand"}, TargetEntityID: npcCommanderElianRookID},
	{ID: "training_ring", DisplayName: "Training Ring", Kind: "objective", CenterX: 42, CenterY: 22, Radius: 18, RouteHintText: "Follow the packed yard path east to the training ring.", QuestIDs: []string{"sv_first_muster", "sv_yard_drills", "sv_stronger_lesson"}, TargetMobType: mobTrainingDummyTypeID, TargetEntityID: warriorTrainerID},
	{ID: "ditch_wall", DisplayName: "Ditch Wall", Kind: "objective", CenterX: 98, CenterY: 45, Radius: 28, RouteHintText: "Take the westward road out of Hearthwatch Yard until the ditch opens beside the wall.", QuestIDs: []string{"sv_wall_rats", "sv_scattered_kit"}, TargetMobType: mobDitchRatTypeID},
	{ID: "field_aid_post", DisplayName: "Field Aid Post", Kind: "service", CenterX: 21, CenterY: 25, Radius: 10, RouteHintText: "Return to Sella Wren near the yard's north path.", QuestIDs: []string{"sv_aid_for_the_hurt", "sv_crows_and_cloth"}, TargetEntityID: npcHealerSellaID},
	{ID: "oatfield_post", DisplayName: "Oatfield Post", Kind: "objective", CenterX: 165, CenterY: 88, Radius: 42, RouteHintText: "Stay on the main road beyond the ditch wall until the oatfield opens around Scout Rowan's post.", QuestIDs: []string{"sv_oatfield_tusks", "sv_bundles_in_furrows"}, TargetMobType: mobFieldBoarTypeID, TargetEntityID: npcScoutRowanID},
	{ID: "ridge_track", DisplayName: "Ridge Track", Kind: "route", CenterX: 232, CenterY: 118, Radius: 26, RouteHintText: "Follow the ridge track marker past the oatfield toward Roadwarden Ilya Brant.", QuestIDs: []string{"sv_roadside_marks"}, TargetEntityID: npcRoadwardenIlyaID},
	{ID: "ridge_crow_roosts", DisplayName: "Ridge Crow Roosts", Kind: "objective", CenterX: 241, CenterY: 128, Radius: 34, RouteHintText: "Search both sides of the ridge track for crow roosts.", QuestIDs: []string{"sv_crows_and_cloth"}, TargetMobType: mobRidgeCrowTypeID},
	{ID: "old_mill_road", DisplayName: "Old Mill Road", Kind: "objective", CenterX: 314, CenterY: 174, Radius: 36, RouteHintText: "Continue west to the old mill road and watch for smoke near the lantern.", QuestIDs: []string{"sv_smoke_at_old_mill", "sv_light_the_lantern"}, TargetMobType: mobAshbandScoutTypeID, TargetEntityID: objWatchLanternID},
	{ID: "poacher_line", DisplayName: "Poacher Line", Kind: "objective", CenterX: 378, CenterY: 213, Radius: 44, RouteHintText: "Push along the road beyond the old mill toward the poacher line.", QuestIDs: []string{"sv_break_poacher_line"}, TargetMobType: mobAshbandPoacherTypeID},
	{ID: "broken_wagon_stand", DisplayName: "Broken Wagon Stand", Kind: "objective", CenterX: 420, CenterY: 224, Radius: 22, RouteHintText: "Follow the road to the isolated broken wagon before the westward gate.", QuestIDs: []string{"sv_bram_kettles_stand"}, TargetMobType: mobBramKettleTypeID},
	{ID: "westward_gate", DisplayName: "Westward Gate", Kind: "handoff", CenterX: 438, CenterY: 246, Radius: 24, RouteHintText: "Continue to the westward gate and report to Quartermaster Lyra.", QuestIDs: []string{"sv_westward_orders"}, TargetEntityID: npcQuartermasterLyraID},
}

var stonewakeGatheringNodeDefinitions = []gatheringNodeDefinition{
	{
		ID:                   "node_vale_iron_01",
		NodeTypeID:           "stonewake_surface_ore",
		DisplayName:          "Exposed Vale Iron",
		X:                    26.0,
		Y:                    29.0,
		Z:                    0.0,
		Radius:               4.0,
		RequiredProfessionID: platform.ProfessionOrekeepingID,
		RequiredSkill:        1,
		Loot: []gatheringLootDefinition{
			{ItemID: itemValeIronChipID, MinCount: 2, MaxCount: 2, Guaranteed: true},
		},
		RespawnDelayMs:   500,
		InteractionLabel: "Gather ore chips",
	},
	{
		ID:                   "node_vale_iron_02",
		NodeTypeID:           "stonewake_surface_ore",
		DisplayName:          "Ditchwall Iron Flecks",
		X:                    91.0,
		Y:                    42.0,
		Z:                    0.0,
		Radius:               4.0,
		RequiredProfessionID: platform.ProfessionOrekeepingID,
		RequiredSkill:        1,
		Loot: []gatheringLootDefinition{
			{ItemID: itemValeIronChipID, MinCount: 1, MaxCount: 2, Guaranteed: true},
		},
		RespawnDelayMs:   500,
		InteractionLabel: "Gather ore chips",
	},
	{
		ID:                   "node_vale_iron_03",
		NodeTypeID:           "stonewake_surface_ore",
		DisplayName:          "Ridge Iron Nubs",
		X:                    232.0,
		Y:                    124.0,
		Z:                    0.0,
		Radius:               4.0,
		RequiredProfessionID: platform.ProfessionOrekeepingID,
		RequiredSkill:        1,
		Loot: []gatheringLootDefinition{
			{ItemID: itemValeIronChipID, MinCount: 2, MaxCount: 3, Guaranteed: true},
		},
		RespawnDelayMs:   500,
		InteractionLabel: "Gather ore chips",
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
	{ID: "mob_training_dummy_01", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 40.0, Y: 18.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_02", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 45.0, Y: 22.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_03", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 39.0, Y: 27.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_ditch_rat_01", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 84.0, Y: 36.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_02", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 94.0, Y: 39.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_03", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 104.0, Y: 35.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_04", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 108.0, Y: 49.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_05", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 86.0, Y: 54.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_06", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 98.0, Y: 57.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_field_boar_01", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 152.0, Y: 73.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_02", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 165.0, Y: 78.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_03", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 181.0, Y: 71.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_04", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 154.0, Y: 93.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_05", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 177.0, Y: 96.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_06", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 143.0, Y: 84.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_07", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 169.0, Y: 110.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_ridge_crow_01", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 226.0, Y: 113.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_02", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 239.0, Y: 119.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_03", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 255.0, Y: 112.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_04", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 226.0, Y: 136.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_05", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 254.0, Y: 136.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_06", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 241.0, Y: 148.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ashband_scout_01", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 296.0, Y: 158.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_02", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 310.0, Y: 151.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_03", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 326.0, Y: 159.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_04", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 300.0, Y: 178.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_05", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 329.0, Y: 180.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_06", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 314.0, Y: 194.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_poacher_01", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 350.0, Y: 188.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_02", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 366.0, Y: 195.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_03", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 383.0, Y: 202.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_04", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 352.0, Y: 214.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_05", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 377.0, Y: 219.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_06", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 398.0, Y: 221.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_07", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 363.0, Y: 236.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_08", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 407.0, Y: 205.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_bram_kettle_01", MobTypeID: mobBramKettleTypeID, DisplayName: "Bram Kettle", Level: 5, X: 420.0, Y: 224.0, MaxHealth: 180, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 30000},
}

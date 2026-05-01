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
	mobDevIsleStalkerTypeID = "dev_isle_stalker"
	mobDevIsleStalkerID     = "npc_dev_isle_stalker_01"
	mobDevIsleStalkerSpawn  = "spawn_dev_isle_stalker_01"
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
		X:           232.0,
		Y:           130.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: starterQuestID, Label: "Hearthwatch Orders"},
			{Type: bindMasterServiceType, ServiceID: bindHearthwatchYardID, Label: "Set Return Signal"},
			{Type: routeMasterServiceType, ServiceID: travelHearthwatchYardID, Label: "Hearthwatch Routes"},
		},
	},
	{
		ID:          warriorTrainerID,
		DisplayName: "Armsmaster Corin Vale",
		Kind:        trainerNPCKind,
		X:           258.0,
		Y:           151.0,
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
		X:           224.0,
		Y:           121.0,
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
		X:           242.0,
		Y:           122.0,
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
		X:           252.0,
		Y:           135.0,
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
		X:           190.0,
		Y:           82.0,
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
		X:           307.0,
		Y:           89.0,
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
		X:           374.0,
		Y:           224.0,
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
		X:           367.0,
		Y:           85.0,
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
			ID:          "stonewake_main_road_loop",
			DisplayName: "Main Road Loop",
			Points: []mapPointDefinition{
				{X: 232, Y: 130},
				{X: 197, Y: 74},
				{X: 260, Y: 70},
				{X: 313, Y: 81},
				{X: 375, Y: 77},
				{X: 358, Y: 39},
				{X: 313, Y: 81},
				{X: 344, Y: 118},
				{X: 361, Y: 157},
				{X: 380, Y: 231},
				{X: 330, Y: 197},
				{X: 292, Y: 160},
				{X: 232, Y: 130},
			},
		},
		{
			ID:          "stonewake_valefurrow_hearthwatch_road",
			DisplayName: "ValeFurrow Farm Road",
			Points: []mapPointDefinition{
				{X: 197, Y: 74},
				{X: 170, Y: 92},
				{X: 148, Y: 132},
				{X: 232, Y: 130},
			},
		},
	},
	Landmarks: []mapLandmarkDefinition{
		{ID: "hearthwatch_yard", DisplayName: "Hearthwatch Yard", Kind: "hub", X: 232, Y: 130},
		{ID: "valefurrow_farms", DisplayName: "ValeFurrow Farms", Kind: "objective", X: 197, Y: 74},
		{ID: "brookside_crossing", DisplayName: "Brookside Crossing", Kind: "route", X: 313, Y: 81},
		{ID: "stonehewn_quarry", DisplayName: "Stonehewn Quarry", Kind: "objective", X: 361, Y: 157},
		{ID: "tiderown_ruins", DisplayName: "Tiderown Ruins", Kind: "objective", X: 358, Y: 39},
		{ID: "lightkeepers_point", DisplayName: "Lightkeeper's Point", Kind: "objective", X: 380, Y: 231},
		{ID: "whispering_cave", DisplayName: "Whispering Cave", Kind: "objective", X: 375, Y: 77},
		{ID: "main_road_loop", DisplayName: "Main Road Loop", Kind: "route", X: 300, Y: 118},
	},
}

var stonewakeNavigationAreas = []navigationAreaDefinition{
	{ID: "hearthwatch_yard", DisplayName: "Hearthwatch Yard", Kind: "hub", CenterX: 232, CenterY: 130, Radius: 24, RouteHintText: "Return to the Hearthwatch Yard command post at the center of Stonewake Vale.", QuestIDs: []string{"sv_first_muster", "sv_wall_rats", "sv_break_poacher_line", "sv_bram_kettles_stand"}, TargetEntityID: npcCommanderElianRookID},
	{ID: "training_ring", DisplayName: "Training Ring", Kind: "objective", CenterX: 268, CenterY: 145, Radius: 18, RouteHintText: "Follow the yard path northeast to the training ring.", QuestIDs: []string{"sv_first_muster", "sv_yard_drills", "sv_stronger_lesson"}, TargetMobType: mobTrainingDummyTypeID, TargetEntityID: warriorTrainerID},
	{ID: "stonehewn_quarry", DisplayName: "Stonehewn Quarry", Kind: "objective", CenterX: 361, CenterY: 157, Radius: 34, RouteHintText: "Take the main loop toward Stonehewn Quarry and clear the work sheds.", QuestIDs: []string{"sv_wall_rats", "sv_scattered_kit"}, TargetMobType: mobDitchRatTypeID},
	{ID: "field_aid_post", DisplayName: "Hearthwatch Field Aid", Kind: "service", CenterX: 242, CenterY: 122, Radius: 10, RouteHintText: "Return to Sella Wren near the Hearthwatch yard's south path.", QuestIDs: []string{"sv_aid_for_the_hurt", "sv_crows_and_cloth"}, TargetEntityID: npcHealerSellaID},
	{ID: "valefurrow_farms", DisplayName: "ValeFurrow Farms", Kind: "objective", CenterX: 197, CenterY: 74, Radius: 38, RouteHintText: "Follow the southern farm road from Hearthwatch Yard to ValeFurrow Farms.", QuestIDs: []string{"sv_oatfield_tusks", "sv_bundles_in_furrows"}, TargetMobType: mobFieldBoarTypeID, TargetEntityID: npcScoutRowanID},
	{ID: "brookside_crossing", DisplayName: "Brookside Crossing", Kind: "route", CenterX: 313, CenterY: 81, Radius: 26, RouteHintText: "Follow the main road loop through Brookside Crossing toward Roadwarden Ilya Brant.", QuestIDs: []string{"sv_roadside_marks"}, TargetEntityID: npcRoadwardenIlyaID},
	{ID: "brookside_roosts", DisplayName: "Brookside Roosts", Kind: "objective", CenterX: 318, CenterY: 96, Radius: 34, RouteHintText: "Search the trees north of Brookside Crossing for crow roosts.", QuestIDs: []string{"sv_crows_and_cloth"}, TargetMobType: mobRidgeCrowTypeID},
	{ID: "lightkeepers_point", DisplayName: "Lightkeeper's Point", Kind: "objective", CenterX: 380, CenterY: 231, Radius: 36, RouteHintText: "Follow the northern road to Lightkeeper's Point and the watch lantern.", QuestIDs: []string{"sv_smoke_at_old_mill", "sv_light_the_lantern"}, TargetMobType: mobAshbandScoutTypeID, TargetEntityID: objWatchLanternID},
	{ID: "whispering_cave", DisplayName: "Whispering Cave", Kind: "objective", CenterX: 375, CenterY: 77, Radius: 38, RouteHintText: "Push along the eastern road toward Whispering Cave.", QuestIDs: []string{"sv_break_poacher_line"}, TargetMobType: mobAshbandPoacherTypeID},
	{ID: "tiderown_ruins", DisplayName: "Tiderown Ruins", Kind: "objective", CenterX: 358, CenterY: 39, Radius: 24, RouteHintText: "Follow the southern spur to Tiderown Ruins.", QuestIDs: []string{"sv_bram_kettles_stand"}, TargetMobType: mobBramKettleTypeID},
	{ID: "whispering_cave_road", DisplayName: "Whispering Cave Road", Kind: "handoff", CenterX: 367, CenterY: 85, Radius: 24, RouteHintText: "Report to Quartermaster Lyra near Whispering Cave.", QuestIDs: []string{"sv_westward_orders"}, TargetEntityID: npcQuartermasterLyraID},
}

var stonewakeGatheringNodeDefinitions = []gatheringNodeDefinition{
	{
		ID:                   "node_vale_iron_01",
		NodeTypeID:           "stonewake_surface_ore",
		DisplayName:          "Exposed Vale Iron",
		X:                    350.0,
		Y:                    150.0,
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
		X:                    365.0,
		Y:                    165.0,
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
		X:                    318.0,
		Y:                    96.0,
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
	{ID: "sv_first_muster", Title: "First Muster", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Armsmaster Corin Vale at the training ring northeast of Hearthwatch Yard.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: warriorTrainerID, TargetEntityID: warriorTrainerID, TargetCount: 1, RewardXP: 35, RewardCopper: 5, LevelBand: "1"},
	{ID: "sv_yard_drills", Title: "Yard Drills", ObjectiveType: objectiveKill, ObjectiveText: "Break 3 Yard Dummies in the training ring.", GiverNPCID: warriorTrainerID, TurnInNPCID: warriorTrainerID, TargetMobType: mobTrainingDummyTypeID, TargetCount: 3, RewardXP: 45, RewardCopper: 5, PrerequisiteIDs: []string{"sv_first_muster"}, LevelBand: "1"},
	{ID: "sv_wall_rats", Title: "Rats at Stonehewn", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Ditch Rats around the Stonehewn Quarry work sheds.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobDitchRatTypeID, TargetCount: 6, RewardXP: 55, RewardCopper: 10, PrerequisiteIDs: []string{"sv_yard_drills"}, LevelBand: "1"},
	{ID: "sv_scattered_kit", Title: "Scattered Kit", ObjectiveType: objectiveCollect, ObjectiveText: "Recover 4 Loose Kit bundles from quarry Ditch Rats.", GiverNPCID: npcQuartermasterMiraID, TurnInNPCID: npcQuartermasterMiraID, TargetMobType: mobDitchRatTypeID, TargetItemID: itemLooseKitID, TargetItemName: "Loose Kit", TargetCount: 4, RewardXP: 60, RewardCopper: 10, RewardItems: []itemRewardDefinition{{ItemID: itemFieldDressingID, DisplayName: "Field Dressing", StackCount: 1}}, PrerequisiteIDs: []string{"sv_wall_rats"}, LevelBand: "1-2"},
	{ID: "sv_aid_for_the_hurt", Title: "Aid for the Hurt", ObjectiveType: objectiveTalk, ObjectiveText: "Bring Mira's field dressing to Sella Wren.", GiverNPCID: npcHealerSellaID, TurnInNPCID: npcHealerSellaID, TargetEntityID: npcHealerSellaID, TargetCount: 1, RewardXP: 45, RewardCopper: 5, PrerequisiteIDs: []string{"sv_scattered_kit"}, LevelBand: "2"},
	{ID: "sv_oatfield_tusks", Title: "Oatfield Tusks", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 7 Oatfield Boars beyond Scout Rowan Bell's field post.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcScoutRowanID, TargetMobType: mobFieldBoarTypeID, TargetCount: 7, RewardXP: 75, RewardCopper: 15, RewardItems: []itemRewardDefinition{{ItemID: itemRoadRationID, DisplayName: "Road Ration", StackCount: 1}}, PrerequisiteIDs: []string{"sv_aid_for_the_hurt"}, LevelBand: "2"},
	{ID: "sv_bundles_in_furrows", Title: "Bundles in the Furrows", ObjectiveType: objectiveCollect, ObjectiveText: "Gather 5 Oat Bundles from the spaced furrows and boars.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcQuartermasterMiraID, TargetMobType: mobFieldBoarTypeID, TargetItemID: itemOatBundleID, TargetItemName: "Oat Bundle", TargetCount: 5, RewardXP: 75, RewardCopper: 15, PrerequisiteIDs: []string{"sv_oatfield_tusks"}, LevelBand: "2-3"},
	{ID: "sv_stronger_lesson", Title: "A Stronger Lesson", ObjectiveType: objectiveTrainer, ObjectiveText: "Learn Driving Blow from Armsmaster Corin Vale.", GiverNPCID: warriorTrainerID, TurnInNPCID: warriorTrainerID, TargetEntityID: platform.DrivingBlowAbilityID, TargetCount: 1, RewardXP: 60, RewardCopper: 10, PrerequisiteIDs: []string{"sv_bundles_in_furrows"}, LevelBand: "3"},
	{ID: "sv_roadside_marks", Title: "Roadside Marks", ObjectiveType: objectiveExplore, ObjectiveText: "Follow the main loop through Brookside Crossing and report to Roadwarden Ilya Brant.", GiverNPCID: npcScoutRowanID, TurnInNPCID: npcRoadwardenIlyaID, TargetEntityID: npcRoadwardenIlyaID, TargetCount: 1, RewardXP: 70, RewardCopper: 20, PrerequisiteIDs: []string{"sv_stronger_lesson"}, LevelBand: "3", MarkerX: 307.0, MarkerY: 89.0},
	{ID: "sv_crows_and_cloth", Title: "Crows and Cloth", ObjectiveType: objectiveCollect, ObjectiveText: "Collect 6 Torn Cloth scraps from Ridge Crows.", GiverNPCID: npcHealerSellaID, TurnInNPCID: npcHealerSellaID, TargetMobType: mobRidgeCrowTypeID, TargetItemID: itemTornClothID, TargetItemName: "Torn Cloth", TargetCount: 6, RewardXP: 85, RewardCopper: 20, RewardItems: []itemRewardDefinition{{ItemID: itemLinenWrapID, DisplayName: "Linen Wrap", StackCount: 1}}, PrerequisiteIDs: []string{"sv_roadside_marks"}, LevelBand: "3-4"},
	{ID: "sv_smoke_at_old_mill", Title: "Smoke at Lightkeeper's Point", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Ashband Scouts near Lightkeeper's Point.", GiverNPCID: npcRoadwardenIlyaID, TurnInNPCID: npcRoadwardenIlyaID, TargetMobType: mobAshbandScoutTypeID, TargetCount: 6, RewardXP: 95, RewardCopper: 25, PrerequisiteIDs: []string{"sv_crows_and_cloth"}, LevelBand: "4"},
	{ID: "sv_light_the_lantern", Title: "Light the Lantern", ObjectiveType: objectiveUse, ObjectiveText: "Use the Watch Lantern at Lightkeeper's Point.", GiverNPCID: npcRoadwardenIlyaID, TurnInNPCID: npcRoadwardenIlyaID, TargetEntityID: objWatchLanternID, TargetCount: 1, RewardXP: 90, RewardCopper: 20, PrerequisiteIDs: []string{"sv_smoke_at_old_mill"}, LevelBand: "4"},
	{ID: "sv_break_poacher_line", Title: "Break the Poacher Line", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 8 Ashband Poachers along the road to Whispering Cave.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobAshbandPoacherTypeID, TargetCount: 8, RewardXP: 110, RewardCopper: 35, PrerequisiteIDs: []string{"sv_light_the_lantern"}, LevelBand: "5"},
	{ID: "sv_bram_kettles_stand", Title: "Bram Kettle's Stand", ObjectiveType: objectiveKill, ObjectiveText: "Defeat Bram Kettle at Tiderown Ruins.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcCommanderElianRookID, TargetMobType: mobBramKettleTypeID, TargetCount: 1, RewardXP: 125, RewardCopper: 50, RewardItems: []itemRewardDefinition{{ItemID: itemMilitiaTokenID, DisplayName: "Militia Token", StackCount: 1}}, PrerequisiteIDs: []string{"sv_break_poacher_line"}, LevelBand: "5"},
	{ID: "sv_westward_orders", Title: "Westward Orders", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Quartermaster Lyra near Whispering Cave.", GiverNPCID: npcCommanderElianRookID, TurnInNPCID: npcQuartermasterLyraID, TargetEntityID: npcQuartermasterLyraID, TargetCount: 1, RewardXP: 100, RewardCopper: 60, RewardItems: []itemRewardDefinition{{ItemID: itemRoadRationID, DisplayName: "Road Ration", StackCount: 2}}, PrerequisiteIDs: []string{"sv_bram_kettles_stand"}, LevelBand: "5-6"},
}

var stonewakeMobSpawns = []mobSpawnDefinition{
	{ID: "mob_training_dummy_01", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 264.0, Y: 141.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_02", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 272.0, Y: 146.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_training_dummy_03", MobTypeID: mobTrainingDummyTypeID, DisplayName: "Yard Dummy", Level: 1, X: 266.0, Y: 152.0, MaxHealth: 30, AttackRange: 2.5, LeashRadius: 10, RespawnDelayMs: 3000},
	{ID: "mob_ditch_rat_01", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 348.0, Y: 148.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_02", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 355.0, Y: 153.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_03", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 365.0, Y: 151.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_04", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 372.0, Y: 161.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_05", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 353.0, Y: 167.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_ditch_rat_06", MobTypeID: mobDitchRatTypeID, DisplayName: "Ditch Rat", Level: 1, X: 366.0, Y: 170.0, MaxHealth: 35, AggroRadius: 1.5, AttackRange: 2.5, AttackDamage: 2, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.5, LeashRadius: 10, RespawnDelayMs: 8000},
	{ID: "mob_field_boar_01", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 182.0, Y: 62.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_02", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 196.0, Y: 66.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_03", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 211.0, Y: 62.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_04", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 185.0, Y: 83.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_05", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 207.0, Y: 87.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_06", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 170.0, Y: 74.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_field_boar_07", MobTypeID: mobFieldBoarTypeID, DisplayName: "Oatfield Boar", Level: 2, X: 218.0, Y: 96.0, MaxHealth: 70, AggroRadius: 3, AttackRange: 2.7, AttackDamage: 5, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 16, RespawnDelayMs: 10000},
	{ID: "mob_ridge_crow_01", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 306.0, Y: 84.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_02", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 318.0, Y: 91.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_03", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 333.0, Y: 86.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_04", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 307.0, Y: 108.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_05", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 332.0, Y: 111.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ridge_crow_06", MobTypeID: mobRidgeCrowTypeID, DisplayName: "Ridge Crow", Level: 3, X: 320.0, Y: 120.0, MaxHealth: 60, AggroRadius: 4, AttackRange: 2.7, AttackDamage: 6, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.5, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_ashband_scout_01", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 365.0, Y: 218.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_02", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 379.0, Y: 212.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_03", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 392.0, Y: 220.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_04", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 366.0, Y: 238.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_05", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 391.0, Y: 242.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_scout_06", MobTypeID: mobAshbandScoutTypeID, DisplayName: "Ashband Scout", Level: 4, X: 380.0, Y: 252.0, MaxHealth: 95, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 8, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_ashband_poacher_01", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 352.0, Y: 64.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_02", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 366.0, Y: 71.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_03", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 383.0, Y: 76.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_04", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 356.0, Y: 88.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_05", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 377.0, Y: 94.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_06", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 398.0, Y: 91.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_07", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 363.0, Y: 104.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_ashband_poacher_08", MobTypeID: mobAshbandPoacherTypeID, DisplayName: "Ashband Poacher", Level: 5, X: 405.0, Y: 76.0, MaxHealth: 115, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 22, RespawnDelayMs: 16000},
	{ID: "mob_bram_kettle_01", MobTypeID: mobBramKettleTypeID, DisplayName: "Bram Kettle", Level: 5, X: 358.0, Y: 39.0, MaxHealth: 180, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 30000},
	{ID: mobDevIsleStalkerID, SpawnPointID: mobDevIsleStalkerSpawn, ArchetypeID: mobDevIsleStalkerTypeID, MobTypeID: mobDevIsleStalkerTypeID, DisplayName: "Isle Stalker", Disposition: npcDispositionHostile, Level: 1, X: 300.0, Y: 118.0, MaxHealth: 30, AggroRadius: 8.0, AttackRange: 2.5, AttackDamage: 3, AttackCadenceMs: 1500, MoveSpeedPerSec: 4.0, LeashRadius: 18.0, RespawnDelayMs: 10000},
}

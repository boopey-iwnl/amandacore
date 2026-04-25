package worlds

import "amandacore/services/internal/platform"

const (
	npcBBCaptainMaraID       = "npc_bb_captain_mara_voss"
	npcBBRoadscribePellenID  = "npc_bb_roadscribe_pellen"
	npcBBQuartermasterDainID = "npc_bb_quartermaster_dain_orro"
	trainerBBJorynHaleID     = "trainer_bb_armsmaster_joryn_hale"
	npcBBHealerLinnetID      = "npc_bb_healer_linnet_cale"
	npcBBFarmerEssaID        = "npc_bb_farmer_essa_bryn"
	npcBBRiverkeeperSolaID   = "npc_bb_riverkeeper_sola_venn"
	objBBFordStakeBundleID   = "obj_bb_ford_stake_bundle"
	npcBBWardenTalikID       = "npc_bb_warden_talik_roe"
	npcBBScoutNessaID        = "npc_bb_scout_nessa_quill"
	npcBBTraderKeviID        = "npc_bb_trader_kevi_sorn"
	npcBBStonesetterBarnID   = "npc_bb_stonesetter_barn_ald"
	objBBSignalBrazierID     = "obj_bb_signal_brazier"
	npcBBWayfinderHelkaID    = "npc_bb_wayfinder_helka_roan"

	mobBBVergeProwlerTypeID       = "bb_verge_prowler"
	mobBBBrindlebackBoarTypeID    = "bb_brindleback_boar"
	mobBBRiverjawSnapperTypeID    = "bb_riverjaw_snapper"
	mobBBGlenfenStalkerTypeID     = "bb_glenfen_stalker"
	mobBBQuarryGrubberTypeID      = "bb_quarry_grubber"
	mobBBRavelmarkCutpurseTypeID  = "bb_ravelmark_cutpurse"
	mobBBRavelmarkRoadbladeTypeID = "bb_ravelmark_roadblade"
	mobBBRavelmarkSignalmanTypeID = "bb_ravelmark_signalman"
	mobBBKorrinMadbrookTypeID     = "bb_korrin_madbrook"
	mobBBRennaVaskTypeID          = "bb_renna_vask"

	itemBBRoadSlatID       = "bb_road_slat"
	itemBBFarmhandGlovesID = "bb_farmhand_gloves"
	itemBBFeedSackID       = "bb_feed_sack"
	itemBBRiverReedID      = "bb_river_reed"
	itemBBFordSatchelID    = "bb_ford_satchel"
	itemBBGoodStoneID      = "bb_good_stone"
	itemBBRedcordTagID     = "bb_redcord_tag"
	itemBBLedgerID         = "bb_ledger_in_coals"
	itemBBWatchpostVestID  = "bb_watchpost_vest"
	itemBBRoadguardBladeID = "bb_roadguard_blade"
)

var zoneDefinitions = []zoneDefinition{
	{
		ID:          defaultZoneID,
		DisplayName: "Stonewake Vale",
		LevelBand:   "1-6",
		Bounds:      zoneBoundsDefinition{MinX: 0, MinY: 0, MaxX: starterZoneMaxX, MaxY: starterZoneMaxY},
		Roads: []zoneRoadDefinition{
			{
				ID:          "stonewake_west_road",
				DisplayName: "Westward Road",
				Points: []zonePointDefinition{
					{ID: "stonewake_hearthwatch", DisplayName: "Hearthwatch Yard", Type: "hub", X: 13, Y: 10},
					{ID: "stonewake_oatfield", DisplayName: "Oatfield Post", Type: "road", X: 134, Y: 64},
					{ID: "stonewake_old_mill", DisplayName: "Old Mill Road", Type: "landmark", X: 322, Y: 174},
					{ID: "stonewake_west_gate", DisplayName: "Westward Gate", Type: "transition", X: 438, Y: 246},
				},
			},
		},
		Landmarks: []zonePointDefinition{
			{ID: "stonewake_hearthwatch_yard", DisplayName: "Hearthwatch Yard", Type: "hub", X: 13, Y: 10},
			{ID: "stonewake_training_ring", DisplayName: "Training Ring", Type: "service", X: 34, Y: 18},
			{ID: "stonewake_old_mill", DisplayName: "Old Mill", Type: "objective", X: 322, Y: 174},
		},
		Transitions: []zonePointDefinition{
			{ID: "to_brindlebrook", DisplayName: "Road to Brindlebrook", Type: "zone_exit", X: 470, Y: 260},
		},
	},
	{
		ID:          secondZoneID,
		DisplayName: "Brindlebrook Roadlands",
		LevelBand:   "5-12",
		Bounds:      zoneBoundsDefinition{MinX: 0, MinY: 0, MaxX: secondZoneMaxX, MaxY: secondZoneMaxY},
		Roads: []zoneRoadDefinition{
			{
				ID:          "bb_main_road",
				DisplayName: "Highmere Road",
				Points: []zonePointDefinition{
					{ID: "bb_vale_gate", DisplayName: "Vale Road Gate", Type: "transition", X: secondZoneEntryX, Y: secondZoneEntryY},
					{ID: "bb_highmere", DisplayName: "Highmere Crossing", Type: "hub", X: 150, Y: 160},
					{ID: "bb_ford", DisplayName: "Brindlebrook Ford", Type: "landmark", X: 310, Y: 160},
					{ID: "bb_pinebarrow", DisplayName: "Pinebarrow Watch", Type: "hub", X: 485, Y: 275},
					{ID: "bb_northspur", DisplayName: "Northspur Checkpoint", Type: "future_handoff", X: 675, Y: 352},
				},
			},
			{
				ID:          "bb_farm_spur",
				DisplayName: "Kettlehook Farm Road",
				Points: []zonePointDefinition{
					{ID: "bb_highmere_farm_start", DisplayName: "Highmere Crossing", Type: "hub", X: 150, Y: 160},
					{ID: "bb_kettlehook", DisplayName: "Kettlehook Farms", Type: "farm", X: 230, Y: 72},
				},
			},
			{
				ID:          "bb_quarry_spur",
				DisplayName: "Tallow Quarry Track",
				Points: []zonePointDefinition{
					{ID: "bb_pinebarrow_quarry_start", DisplayName: "Pinebarrow Watch", Type: "hub", X: 485, Y: 275},
					{ID: "bb_quarry", DisplayName: "Old Tallow Quarry", Type: "profession", X: 590, Y: 330},
					{ID: "bb_redcord", DisplayName: "Redcord Camp", Type: "hostile_camp", X: 585, Y: 190},
				},
			},
		},
		Landmarks: []zonePointDefinition{
			{ID: "bb_vale_road_gate", DisplayName: "Vale Road Gate", Type: "transition", X: secondZoneEntryX, Y: secondZoneEntryY},
			{ID: "bb_highmere_crossing", DisplayName: "Highmere Crossing", Type: "main_hub", X: 150, Y: 160},
			{ID: "bb_kettlehook_farms", DisplayName: "Kettlehook Farms", Type: "farm", X: 230, Y: 72},
			{ID: "bb_brindlebrook_ford", DisplayName: "Brindlebrook Ford", Type: "river", X: 310, Y: 160},
			{ID: "bb_glenfen_edge", DisplayName: "Glenfen Edge", Type: "wilds", X: 300, Y: 318},
			{ID: "bb_pinebarrow_watch", DisplayName: "Pinebarrow Watch", Type: "secondary_hub", X: 485, Y: 275},
			{ID: "bb_old_tallow_quarry", DisplayName: "Old Tallow Quarry", Type: "quarry", X: 590, Y: 330},
			{ID: "bb_redcord_camp", DisplayName: "Redcord Camp", Type: "hostile_camp", X: 585, Y: 190},
			{ID: "bb_cinder_signal", DisplayName: "Cinder Signal Tower", Type: "tower", X: 640, Y: 110},
			{ID: "bb_northspur_checkpoint", DisplayName: "Northspur Checkpoint", Type: "future_handoff", X: 675, Y: 352},
		},
		Transitions: []zonePointDefinition{
			{ID: "from_stonewake", DisplayName: "Stonewake Road", Type: "zone_entry", X: secondZoneEntryX, Y: secondZoneEntryY},
			{ID: "to_future_northspur", DisplayName: "Road Beyond Northspur", Type: "future_zone", X: 700, Y: 360},
		},
	},
}

var brindlebrookFriendlyNPCs = []friendlyNPCDefinition{
	{
		ID:          npcBBCaptainMaraID,
		ZoneID:      secondZoneID,
		DisplayName: "Captain Mara Voss",
		Kind:        questGiverNPCKind,
		X:           150.0,
		Y:           160.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_road_to_highmere", Label: "Roadlands Orders"},
			{Type: "quest", ServiceID: "bb_clear_south_verge", Label: "Road Patrol"},
			{Type: "quest", ServiceID: "bb_northspur_orders", Label: "Northspur Orders"},
		},
	},
	{
		ID:          npcBBRoadscribePellenID,
		ZoneID:      secondZoneID,
		DisplayName: "Roadscribe Pellen",
		Kind:        questGiverNPCKind,
		X:           164.0,
		Y:           166.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_road_marks_missing", Label: "Road Markers"},
		},
	},
	{
		ID:          npcBBQuartermasterDainID,
		ZoneID:      secondZoneID,
		DisplayName: "Dain Orro",
		Kind:        questGiverNPCKind,
		X:           142.0,
		Y:           148.0,
		Z:           0.0,
		AIState:     "vendor",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "vendor", ServiceID: vendorHighmereDainID, Label: "Roadland Supplies"},
		},
	},
	{
		ID:          trainerBBJorynHaleID,
		ZoneID:      secondZoneID,
		DisplayName: "Armsmaster Joryn Hale",
		Kind:        trainerNPCKind,
		X:           170.0,
		Y:           148.0,
		Z:           0.0,
		AIState:     "trainer",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "trainer", ServiceID: trainerBBJorynHaleID, Label: "Warrior Training"},
			{Type: "quest", ServiceID: "bb_crossing_rollcall", Label: "Crossing Roll Call"},
		},
	},
	{
		ID:          npcBBHealerLinnetID,
		ZoneID:      secondZoneID,
		DisplayName: "Linnet Cale",
		Kind:        questGiverNPCKind,
		X:           160.0,
		Y:           178.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_reedbed_remedy", Label: "Field Remedies"},
		},
	},
	{
		ID:          npcBBFarmerEssaID,
		ZoneID:      secondZoneID,
		DisplayName: "Essa Bryn",
		Kind:        questGiverNPCKind,
		X:           230.0,
		Y:           72.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_farmstead_rounds", Label: "Farmstead Rounds"},
			{Type: "quest", ServiceID: "bb_fence_line", Label: "Fence Line"},
		},
	},
	{
		ID:          npcBBRiverkeeperSolaID,
		ZoneID:      secondZoneID,
		DisplayName: "Sola Venn",
		Kind:        questGiverNPCKind,
		X:           310.0,
		Y:           160.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_mud_at_the_ford", Label: "Ford Watch"},
			{Type: "quest", ServiceID: "bb_set_ford_stakes", Label: "Ford Stakes"},
			{Type: "quest", ServiceID: "bb_teeth_in_shallows", Label: "Shallows"},
		},
	},
	{
		ID:          objBBFordStakeBundleID,
		ZoneID:      secondZoneID,
		DisplayName: "Ford Stake Bundle",
		Kind:        worldObjectNPCKind,
		X:           332.0,
		Y:           168.0,
		Z:           0.0,
		AIState:     "quest_object",
		Radius:      4.0,
	},
	{
		ID:          npcBBWardenTalikID,
		ZoneID:      secondZoneID,
		DisplayName: "Warden Talik Roe",
		Kind:        questGiverNPCKind,
		X:           485.0,
		Y:           275.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_watch_needs_eyes", Label: "Watch Orders"},
			{Type: "quest", ServiceID: "bb_redcord_tags", Label: "Redcord Tags"},
			{Type: "quest", ServiceID: "bb_renna_vask", Label: "Roadblock"},
		},
	},
	{
		ID:          npcBBScoutNessaID,
		ZoneID:      secondZoneID,
		DisplayName: "Nessa Quill",
		Kind:        questGiverNPCKind,
		X:           500.0,
		Y:           286.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_break_the_camp", Label: "Camp Line"},
			{Type: "quest", ServiceID: "bb_light_signal_brazier", Label: "Signal Tower"},
			{Type: "quest", ServiceID: "bb_hold_the_signal", Label: "Hold Signal"},
		},
	},
	{
		ID:          npcBBTraderKeviID,
		ZoneID:      secondZoneID,
		DisplayName: "Kevi Sorn",
		Kind:        questGiverNPCKind,
		X:           474.0,
		Y:           262.0,
		Z:           0.0,
		AIState:     "vendor",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "vendor", ServiceID: vendorPinebarrowKeviID, Label: "Watch Supplies"},
		},
	},
	{
		ID:          npcBBStonesetterBarnID,
		ZoneID:      secondZoneID,
		DisplayName: "Barn Ald",
		Kind:        professionTrainerNPCKind,
		X:           550.0,
		Y:           324.0,
		Z:           0.0,
		AIState:     "profession_support",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_quarry_noise", Label: "Quarry Noise"},
			{Type: "quest", ServiceID: "bb_good_stone_optional", Label: "Good Stone"},
		},
	},
	{
		ID:          objBBSignalBrazierID,
		ZoneID:      secondZoneID,
		DisplayName: "Signal Brazier",
		Kind:        worldObjectNPCKind,
		X:           640.0,
		Y:           110.0,
		Z:           0.0,
		AIState:     "quest_object",
		Radius:      4.0,
	},
	{
		ID:          npcBBWayfinderHelkaID,
		ZoneID:      secondZoneID,
		DisplayName: "Helka Roan",
		Kind:        questGiverNPCKind,
		X:           675.0,
		Y:           352.0,
		Z:           0.0,
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services: []npcService{
			{Type: "quest", ServiceID: "bb_northspur_orders", Label: "Future Road"},
		},
	},
}

var brindlebrookQuestDefinitions = []questDefinition{
	{ID: "bb_road_to_highmere", ZoneID: secondZoneID, Title: "Road to Highmere", ObjectiveType: objectiveExplore, ObjectiveText: "Follow the main road from the Vale Road Gate to Highmere Crossing.", GiverNPCID: npcBBCaptainMaraID, TurnInNPCID: npcBBCaptainMaraID, TargetCount: 1, RewardXP: 180, RewardCopper: 35, LevelBand: "5-6", MarkerX: 150.0, MarkerY: 160.0},
	{ID: "bb_crossing_rollcall", ZoneID: secondZoneID, Title: "Roll Call at the Crossing", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Armsmaster Joryn Hale at Highmere Crossing.", GiverNPCID: npcBBCaptainMaraID, TurnInNPCID: trainerBBJorynHaleID, TargetEntityID: trainerBBJorynHaleID, TargetCount: 1, RewardXP: 140, RewardCopper: 25, PrerequisiteIDs: []string{"bb_road_to_highmere"}, LevelBand: "6"},
	{ID: "bb_clear_south_verge", ZoneID: secondZoneID, Title: "Teeth Along the Verge", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Verge Prowlers along the road west of the Vale Road Gate.", GiverNPCID: npcBBCaptainMaraID, TurnInNPCID: npcBBCaptainMaraID, TargetMobType: mobBBVergeProwlerTypeID, TargetCount: 6, RewardXP: 220, RewardCopper: 45, PrerequisiteIDs: []string{"bb_crossing_rollcall"}, LevelBand: "6"},
	{ID: "bb_road_marks_missing", ZoneID: secondZoneID, Title: "Missing Road Marks", ObjectiveType: objectiveCollect, ObjectiveText: "Recover 5 painted road slats from Verge Prowlers and scattered verge caches.", GiverNPCID: npcBBRoadscribePellenID, TurnInNPCID: npcBBRoadscribePellenID, TargetMobType: mobBBVergeProwlerTypeID, TargetItemID: itemBBRoadSlatID, TargetItemName: "Road Slat", TargetCount: 5, RewardXP: 220, RewardCopper: 45, PrerequisiteIDs: []string{"bb_clear_south_verge"}, LevelBand: "6"},
	{ID: "bb_farmstead_rounds", ZoneID: secondZoneID, Title: "A Farmer's Warning", ObjectiveType: objectiveTalk, ObjectiveText: "Carry Highmere's warning to Essa Bryn at Kettlehook Farms.", GiverNPCID: npcBBCaptainMaraID, TurnInNPCID: npcBBFarmerEssaID, TargetEntityID: npcBBFarmerEssaID, TargetCount: 1, RewardXP: 160, RewardCopper: 30, PrerequisiteIDs: []string{"bb_road_marks_missing"}, LevelBand: "6"},
	{ID: "bb_fence_line", ZoneID: secondZoneID, Title: "Hold the Fence Line", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 7 Brindleback Boars near Kettlehook Farms.", GiverNPCID: npcBBFarmerEssaID, TurnInNPCID: npcBBFarmerEssaID, TargetMobType: mobBBBrindlebackBoarTypeID, TargetCount: 7, RewardXP: 250, RewardCopper: 55, RewardItems: []itemRewardDefinition{{ItemID: itemBBFarmhandGlovesID, DisplayName: "Farmhand Gloves", StackCount: 1}}, PrerequisiteIDs: []string{"bb_farmstead_rounds"}, LevelBand: "6-7"},
	{ID: "bb_feed_sacks", ZoneID: secondZoneID, Title: "Sacks in the Furrows", ObjectiveType: objectiveCollect, ObjectiveText: "Recover 6 feed sacks from the damaged farm rows.", GiverNPCID: npcBBFarmerEssaID, TurnInNPCID: npcBBQuartermasterDainID, TargetMobType: mobBBBrindlebackBoarTypeID, TargetItemID: itemBBFeedSackID, TargetItemName: "Feed Sack", TargetCount: 6, RewardXP: 240, RewardCopper: 50, PrerequisiteIDs: []string{"bb_fence_line"}, LevelBand: "7"},
	{ID: "bb_reedbed_remedy", ZoneID: secondZoneID, Title: "Reedbed Remedy", ObjectiveType: objectiveCollect, ObjectiveText: "Gather 5 river reeds from the ford's reedbeds and riverjaw nests.", GiverNPCID: npcBBHealerLinnetID, TurnInNPCID: npcBBHealerLinnetID, TargetMobType: mobBBRiverjawSnapperTypeID, TargetItemID: itemBBRiverReedID, TargetItemName: "River Reed", TargetCount: 5, RewardXP: 250, RewardCopper: 50, RewardItems: []itemRewardDefinition{{ItemID: itemLinenWrapID, DisplayName: "Linen Wrap", StackCount: 2}}, PrerequisiteIDs: []string{"bb_feed_sacks"}, LevelBand: "7"},
	{ID: "bb_mud_at_the_ford", ZoneID: secondZoneID, Title: "Mud at the Ford", ObjectiveType: objectiveExplore, ObjectiveText: "Inspect the Brindlebrook Ford crossing.", GiverNPCID: npcBBRiverkeeperSolaID, TurnInNPCID: npcBBRiverkeeperSolaID, TargetCount: 1, RewardXP: 230, RewardCopper: 45, PrerequisiteIDs: []string{"bb_reedbed_remedy"}, LevelBand: "7", MarkerX: 310.0, MarkerY: 160.0},
	{ID: "bb_set_ford_stakes", ZoneID: secondZoneID, Title: "Set the Ford Stakes", ObjectiveType: objectiveUse, ObjectiveText: "Use the Ford Stake Bundle to mark the safer crossing path.", GiverNPCID: npcBBRiverkeeperSolaID, TurnInNPCID: npcBBRiverkeeperSolaID, TargetEntityID: objBBFordStakeBundleID, TargetCount: 1, RewardXP: 280, RewardCopper: 60, PrerequisiteIDs: []string{"bb_mud_at_the_ford"}, LevelBand: "7-8"},
	{ID: "bb_teeth_in_shallows", ZoneID: secondZoneID, Title: "Teeth in the Shallows", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 6 Riverjaw Snappers near Brindlebrook Ford.", GiverNPCID: npcBBRiverkeeperSolaID, TurnInNPCID: npcBBRiverkeeperSolaID, TargetMobType: mobBBRiverjawSnapperTypeID, TargetCount: 6, RewardXP: 290, RewardCopper: 65, PrerequisiteIDs: []string{"bb_set_ford_stakes"}, LevelBand: "8"},
	{ID: "bb_lost_satchel", ZoneID: secondZoneID, Title: "The Lost Mill Satchel", ObjectiveType: objectiveCollect, ObjectiveText: "Recover the lost ford satchel from Riverjaw Snappers, then bring it to Warden Talik Roe.", GiverNPCID: npcBBRiverkeeperSolaID, TurnInNPCID: npcBBWardenTalikID, TargetMobType: mobBBRiverjawSnapperTypeID, TargetItemID: itemBBFordSatchelID, TargetItemName: "Ford Satchel", TargetCount: 1, RewardXP: 260, RewardCopper: 60, PrerequisiteIDs: []string{"bb_teeth_in_shallows"}, LevelBand: "8"},
	{ID: "bb_watch_needs_eyes", ZoneID: secondZoneID, Title: "The Watch Needs Eyes", ObjectiveType: objectiveExplore, ObjectiveText: "Scout the Pinebarrow overlook above the north road.", GiverNPCID: npcBBWardenTalikID, TurnInNPCID: npcBBWardenTalikID, TargetCount: 1, RewardXP: 330, RewardCopper: 70, PrerequisiteIDs: []string{"bb_lost_satchel"}, LevelBand: "8-9", MarkerX: 512.0, MarkerY: 300.0},
	{ID: "bb_quarry_noise", ZoneID: secondZoneID, Title: "Noise from Tallow Quarry", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 7 Quarry Grubbers in Old Tallow Quarry.", GiverNPCID: npcBBStonesetterBarnID, TurnInNPCID: npcBBStonesetterBarnID, TargetMobType: mobBBQuarryGrubberTypeID, TargetCount: 7, RewardXP: 360, RewardCopper: 80, PrerequisiteIDs: []string{"bb_watch_needs_eyes"}, LevelBand: "9"},
	{ID: "bb_good_stone_optional", ZoneID: secondZoneID, Title: "Good Stone for Bad Roads", ObjectiveType: objectiveCollect, ObjectiveText: "Collect 4 good stones from Quarry Grubbers and exposed quarry seams.", GiverNPCID: npcBBStonesetterBarnID, TurnInNPCID: npcBBStonesetterBarnID, TargetMobType: mobBBQuarryGrubberTypeID, TargetItemID: itemBBGoodStoneID, TargetItemName: "Good Stone", TargetCount: 4, RewardXP: 250, RewardCopper: 50, PrerequisiteIDs: []string{"bb_quarry_noise"}, LevelBand: "9"},
	{ID: "bb_redcord_tags", ZoneID: secondZoneID, Title: "Redcord Tags", ObjectiveType: objectiveCollect, ObjectiveText: "Recover 6 Redcord tags from Ravelmark raiders.", GiverNPCID: npcBBWardenTalikID, TurnInNPCID: npcBBWardenTalikID, TargetMobType: mobBBRavelmarkCutpurseTypeID, TargetItemID: itemBBRedcordTagID, TargetItemName: "Redcord Tag", TargetCount: 6, RewardXP: 380, RewardCopper: 85, PrerequisiteIDs: []string{"bb_quarry_noise"}, LevelBand: "9-10"},
	{ID: "bb_break_the_camp", ZoneID: secondZoneID, Title: "Break the Camp Line", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 8 Ravelmark raiders around Redcord Camp.", GiverNPCID: npcBBScoutNessaID, TurnInNPCID: npcBBScoutNessaID, TargetMobType: mobBBRavelmarkRoadbladeTypeID, TargetCount: 8, RewardXP: 410, RewardCopper: 90, PrerequisiteIDs: []string{"bb_redcord_tags"}, LevelBand: "10"},
	{ID: "bb_light_signal_brazier", ZoneID: secondZoneID, Title: "Light the Cinder Signal", ObjectiveType: objectiveUse, ObjectiveText: "Use the Signal Brazier at the Cinder Signal Tower.", GiverNPCID: npcBBScoutNessaID, TurnInNPCID: npcBBScoutNessaID, TargetEntityID: objBBSignalBrazierID, TargetCount: 1, RewardXP: 340, RewardCopper: 75, PrerequisiteIDs: []string{"bb_break_the_camp"}, LevelBand: "10"},
	{ID: "bb_hold_the_signal", ZoneID: secondZoneID, Title: "Hold the Signal", ObjectiveType: objectiveKill, ObjectiveText: "Defeat 3 Ravelmark Signalmen near the lit tower.", GiverNPCID: npcBBScoutNessaID, TurnInNPCID: npcBBWardenTalikID, TargetMobType: mobBBRavelmarkSignalmanTypeID, TargetCount: 3, RewardXP: 450, RewardCopper: 100, PrerequisiteIDs: []string{"bb_light_signal_brazier"}, LevelBand: "10-11"},
	{ID: "bb_renna_vask", ZoneID: secondZoneID, Title: "Renna Vask's Roadblock", ObjectiveType: objectiveKill, ObjectiveText: "Defeat Renna Vask at Redcord Camp.", GiverNPCID: npcBBWardenTalikID, TurnInNPCID: npcBBWardenTalikID, TargetMobType: mobBBRennaVaskTypeID, TargetCount: 1, RewardXP: 550, RewardCopper: 120, RewardItems: []itemRewardDefinition{{ItemID: itemBBWatchpostVestID, DisplayName: "Watchpost Vest", StackCount: 1}}, PrerequisiteIDs: []string{"bb_hold_the_signal"}, LevelBand: "11"},
	{ID: "bb_ledger_in_coals", ZoneID: secondZoneID, Title: "Ledger in the Coals", ObjectiveType: objectiveCollect, ObjectiveText: "Recover the burned ledger from Renna's camp followers.", GiverNPCID: npcBBWardenTalikID, TurnInNPCID: npcBBCaptainMaraID, TargetMobType: mobBBRavelmarkRoadbladeTypeID, TargetItemID: itemBBLedgerID, TargetItemName: "Ledger in the Coals", TargetCount: 1, RewardXP: 360, RewardCopper: 90, PrerequisiteIDs: []string{"bb_renna_vask"}, LevelBand: "11"},
	{ID: "bb_northspur_orders", ZoneID: secondZoneID, Title: "Orders for Northspur", ObjectiveType: objectiveTalk, ObjectiveText: "Report to Helka Roan at Northspur Checkpoint.", GiverNPCID: npcBBCaptainMaraID, TurnInNPCID: npcBBWayfinderHelkaID, TargetEntityID: npcBBWayfinderHelkaID, TargetCount: 1, RewardXP: 500, RewardCopper: 125, RewardItems: []itemRewardDefinition{{ItemID: itemBBRoadguardBladeID, DisplayName: "Roadguard Blade", StackCount: 1}}, PrerequisiteIDs: []string{"bb_ledger_in_coals"}, LevelBand: "11-12"},
}

var brindlebrookMobSpawns = []mobSpawnDefinition{
	{ID: "mob_bb_verge_prowler_01", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 5, X: 70.0, Y: 126.0, MaxHealth: 115, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 9, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_bb_verge_prowler_02", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 5, X: 90.0, Y: 133.0, MaxHealth: 115, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 9, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_bb_verge_prowler_03", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 6, X: 112.0, Y: 122.0, MaxHealth: 125, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_verge_prowler_04", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 6, X: 80.0, Y: 186.0, MaxHealth: 125, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_verge_prowler_05", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 6, X: 105.0, Y: 197.0, MaxHealth: 125, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_verge_prowler_06", ZoneID: secondZoneID, MobTypeID: mobBBVergeProwlerTypeID, DisplayName: "Verge Prowler", Level: 6, X: 128.0, Y: 184.0, MaxHealth: 125, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_01", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 6, X: 214.0, Y: 42.0, MaxHealth: 125, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_02", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 6, X: 238.0, Y: 48.0, MaxHealth: 125, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 10, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 18, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_03", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 7, X: 262.0, Y: 72.0, MaxHealth: 135, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_04", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 7, X: 214.0, Y: 98.0, MaxHealth: 135, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_05", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 7, X: 250.0, Y: 106.0, MaxHealth: 135, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_boar_06", ZoneID: secondZoneID, MobTypeID: mobBBBrindlebackBoarTypeID, DisplayName: "Brindleback Boar", Level: 7, X: 284.0, Y: 92.0, MaxHealth: 135, AggroRadius: 4.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 20, RespawnDelayMs: 12000},
	{ID: "mob_bb_snapper_01", ZoneID: secondZoneID, MobTypeID: mobBBRiverjawSnapperTypeID, DisplayName: "Riverjaw Snapper", Level: 7, X: 294.0, Y: 138.0, MaxHealth: 135, AggroRadius: 3.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 18, RespawnDelayMs: 13000},
	{ID: "mob_bb_snapper_02", ZoneID: secondZoneID, MobTypeID: mobBBRiverjawSnapperTypeID, DisplayName: "Riverjaw Snapper", Level: 7, X: 322.0, Y: 136.0, MaxHealth: 135, AggroRadius: 3.5, AttackRange: 2.75, AttackDamage: 11, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 18, RespawnDelayMs: 13000},
	{ID: "mob_bb_snapper_03", ZoneID: secondZoneID, MobTypeID: mobBBRiverjawSnapperTypeID, DisplayName: "Riverjaw Snapper", Level: 8, X: 346.0, Y: 164.0, MaxHealth: 150, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 20, RespawnDelayMs: 13000},
	{ID: "mob_bb_snapper_04", ZoneID: secondZoneID, MobTypeID: mobBBRiverjawSnapperTypeID, DisplayName: "Riverjaw Snapper", Level: 8, X: 310.0, Y: 190.0, MaxHealth: 150, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 20, RespawnDelayMs: 13000},
	{ID: "mob_bb_snapper_05", ZoneID: secondZoneID, MobTypeID: mobBBRiverjawSnapperTypeID, DisplayName: "Riverjaw Snapper", Level: 8, X: 340.0, Y: 202.0, MaxHealth: 150, AggroRadius: 4, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 20, RespawnDelayMs: 13000},
	{ID: "mob_bb_glenfen_01", ZoneID: secondZoneID, MobTypeID: mobBBGlenfenStalkerTypeID, DisplayName: "Glenfen Stalker", Level: 8, X: 284.0, Y: 300.0, MaxHealth: 150, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_bb_glenfen_02", ZoneID: secondZoneID, MobTypeID: mobBBGlenfenStalkerTypeID, DisplayName: "Glenfen Stalker", Level: 8, X: 318.0, Y: 326.0, MaxHealth: 150, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_bb_glenfen_03", ZoneID: secondZoneID, MobTypeID: mobBBGlenfenStalkerTypeID, DisplayName: "Glenfen Stalker", Level: 8, X: 354.0, Y: 304.0, MaxHealth: 150, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 12, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.4, LeashRadius: 20, RespawnDelayMs: 14000},
	{ID: "mob_bb_quarry_01", ZoneID: secondZoneID, MobTypeID: mobBBQuarryGrubberTypeID, DisplayName: "Quarry Grubber", Level: 9, X: 560.0, Y: 318.0, MaxHealth: 165, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_quarry_02", ZoneID: secondZoneID, MobTypeID: mobBBQuarryGrubberTypeID, DisplayName: "Quarry Grubber", Level: 9, X: 592.0, Y: 336.0, MaxHealth: 165, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_quarry_03", ZoneID: secondZoneID, MobTypeID: mobBBQuarryGrubberTypeID, DisplayName: "Quarry Grubber", Level: 9, X: 624.0, Y: 318.0, MaxHealth: 165, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_quarry_04", ZoneID: secondZoneID, MobTypeID: mobBBQuarryGrubberTypeID, DisplayName: "Quarry Grubber", Level: 9, X: 604.0, Y: 368.0, MaxHealth: 165, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 13, AttackCadenceMs: 2000, MoveSpeedPerSec: 3.8, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_cutpurse_01", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkCutpurseTypeID, DisplayName: "Ravelmark Cutpurse", Level: 9, X: 548.0, Y: 176.0, MaxHealth: 170, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.1, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_cutpurse_02", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkCutpurseTypeID, DisplayName: "Ravelmark Cutpurse", Level: 9, X: 584.0, Y: 172.0, MaxHealth: 170, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.1, LeashRadius: 22, RespawnDelayMs: 15000},
	{ID: "mob_bb_cutpurse_03", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkCutpurseTypeID, DisplayName: "Ravelmark Cutpurse", Level: 10, X: 618.0, Y: 188.0, MaxHealth: 180, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.1, LeashRadius: 24, RespawnDelayMs: 15000},
	{ID: "mob_bb_roadblade_01", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkRoadbladeTypeID, DisplayName: "Ravelmark Roadblade", Level: 10, X: 556.0, Y: 210.0, MaxHealth: 190, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 16000},
	{ID: "mob_bb_roadblade_02", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkRoadbladeTypeID, DisplayName: "Ravelmark Roadblade", Level: 10, X: 594.0, Y: 222.0, MaxHealth: 190, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 16000},
	{ID: "mob_bb_roadblade_03", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkRoadbladeTypeID, DisplayName: "Ravelmark Roadblade", Level: 10, X: 632.0, Y: 212.0, MaxHealth: 190, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 15, AttackCadenceMs: 2000, MoveSpeedPerSec: 4.2, LeashRadius: 24, RespawnDelayMs: 16000},
	{ID: "mob_bb_signalman_01", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkSignalmanTypeID, DisplayName: "Ravelmark Signalman", Level: 10, X: 622.0, Y: 104.0, MaxHealth: 180, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 22, RespawnDelayMs: 18000},
	{ID: "mob_bb_signalman_02", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkSignalmanTypeID, DisplayName: "Ravelmark Signalman", Level: 10, X: 648.0, Y: 128.0, MaxHealth: 180, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 22, RespawnDelayMs: 18000},
	{ID: "mob_bb_signalman_03", ZoneID: secondZoneID, MobTypeID: mobBBRavelmarkSignalmanTypeID, DisplayName: "Ravelmark Signalman", Level: 10, X: 674.0, Y: 104.0, MaxHealth: 180, AggroRadius: 5.5, AttackRange: 2.75, AttackDamage: 14, AttackCadenceMs: 2000, MoveSpeedPerSec: 4, LeashRadius: 22, RespawnDelayMs: 18000},
	{ID: "mob_bb_korrin_madbrook_01", ZoneID: secondZoneID, MobTypeID: mobBBKorrinMadbrookTypeID, DisplayName: "Korrin Madbrook", Level: 8, X: 366.0, Y: 184.0, MaxHealth: 245, AggroRadius: 5, AttackRange: 2.75, AttackDamage: 16, AttackCadenceMs: 1900, MoveSpeedPerSec: 4, LeashRadius: 24, RespawnDelayMs: 30000},
	{ID: "mob_bb_renna_vask_01", ZoneID: secondZoneID, MobTypeID: mobBBRennaVaskTypeID, DisplayName: "Renna Vask", Level: 11, X: 650.0, Y: 230.0, MaxHealth: 310, AggroRadius: 6, AttackRange: 2.75, AttackDamage: 18, AttackCadenceMs: 1900, MoveSpeedPerSec: 4.2, LeashRadius: 28, RespawnDelayMs: 45000},
}

var _ = platform.DefaultClassID

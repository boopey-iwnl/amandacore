package worlds

const (
	devFirstHuntQuestID       = "dev_first_hunt"
	devIsleStalkerArchetypeID = "dev_isle_stalker"
	devIsleStalkerSpawnID     = "spawn_dev_isle_stalker_01"
	devIsleStalkerEntityID    = "npc_dev_isle_stalker_01"
	devFirstHuntKillNodeID    = "node_kill_stalker"
	devFirstHuntCollectNodeID = "node_collect_fang"
)

var devProgressionQuestDefinitions = []questDefinition{
	{
		ID:                devFirstHuntQuestID,
		Title:             "First Hunt",
		Summary:           "Help secure the nearby path by defeating an Isle Stalker and recovering a fang.",
		ObjectiveType:     objectiveCollect,
		ObjectiveText:     "Defeat an Isle Stalker, then recover a Stalker Fang from its reward container.",
		TargetMobType:     devIsleStalkerArchetypeID,
		TargetItemID:      itemDevStalkerFangID,
		TargetItemName:    "Stalker Fang",
		TargetCount:       1,
		RewardItems:       []itemRewardDefinition{{ItemID: itemDevCopperTokenID, DisplayName: "Copper Token", StackCount: 5}, {ItemID: itemDevGlimmerShardID, DisplayName: "Glimmer Shard", StackCount: 1}},
		AllowDirectAccept: true,
		Tags:              []string{"dev", "progression", "clean-room"},
		ObjectiveGraph: questObjectiveGraph{
			Nodes: []questObjectiveNode{
				{
					NodeID:             devFirstHuntKillNodeID,
					Kind:               objectiveKindKillNPC,
					TargetNpcArchetype: devIsleStalkerArchetypeID,
					TargetCount:        1,
				},
				{
					NodeID:       devFirstHuntCollectNodeID,
					Kind:         objectiveKindCollectItem,
					TargetItemID: itemDevStalkerFangID,
					TargetCount:  1,
					DependsOn:    []string{devFirstHuntKillNodeID},
					Terminal:     true,
				},
			},
		},
	},
}

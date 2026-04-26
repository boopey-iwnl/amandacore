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

var devProgressionMobSpawns = []mobSpawnDefinition{
	{
		ID:              devIsleStalkerEntityID,
		SpawnPointID:    devIsleStalkerSpawnID,
		ZoneID:          defaultZoneID,
		MobTypeID:       devIsleStalkerArchetypeID,
		ArchetypeID:     devIsleStalkerArchetypeID,
		DisplayName:     "Isle Stalker",
		Level:           1,
		Disposition:     string(NpcDispositionHostile),
		LootTableID:     devIsleStalkerLootTableID,
		X:               42.0,
		Y:               18.0,
		Z:               0.0,
		MaxHealth:       30,
		AggroRadius:     8.0,
		AttackRange:     2.5,
		AttackDamage:    3,
		AttackCadenceMs: 1500,
		MoveSpeedPerSec: 3.2,
		LeashRadius:     18.0,
		RespawnDelayMs:  10_000,
	},
}

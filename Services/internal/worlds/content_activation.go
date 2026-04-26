package worlds

import (
	"fmt"
	"strings"
	"time"

	contentpkg "amandacore/services/internal/content"
	"amandacore/services/internal/observability"
)

type ContentZoneRuntime struct {
	ZoneID                  string
	DisplayName             string
	RuntimeConfig           contentpkg.ZoneRuntimeConfig
	EntryPointCount         int
	SpawnGroupCount         int
	QuestProviderCount      int
	SpawnedNPCCount         int
	RegisteredQuestProvider int
	ActivatedAt             time.Time
}

type ContentActivationResult struct {
	PackageID                string
	PackageVersion           string
	ZonesActivated           int
	CatalogsLoaded           map[string]int
	NPCsSpawned              int
	QuestProvidersRegistered int
	QuestsRegistered         int
	Errors                   []contentpkg.ContentValidationError
}

func (s *worldServer) loadConfiguredContentPackageLocked(contentPackagePath string) {
	result := contentpkg.NewContentPackageLoader().Load(contentPackagePath)
	if !result.Validation.Valid() || result.Validated == nil {
		s.contentActivation.Errors = append([]contentpkg.ContentValidationError(nil), result.Validation.Errors...)
		observability.LogEvent("world-service", contentpkg.EventPackageActivationFailed, map[string]any{
			"errorCount": len(result.Validation.Errors),
		})
		return
	}

	s.activateValidatedContentPackageLocked(*result.Validated)
}

func (s *worldServer) activateValidatedContentPackageLocked(pkg contentpkg.ValidatedContentPackage) ContentActivationResult {
	registry := pkg.Registry
	s.contentRegistry = &registry
	s.registerContentAbilityCatalogLocked(registry)
	result := ContentActivationResult{
		PackageID:      registry.PackageID,
		PackageVersion: registry.Version,
		CatalogsLoaded: map[string]int{
			"zones":       len(registry.Zones),
			"npcs":        len(registry.NPCs),
			"items":       len(registry.Items),
			"loot_tables": len(registry.LootTables),
			"quests":      len(registry.Quests),
			"abilities":   len(registry.Abilities),
			"auras":       len(registry.Auras),
		},
	}

	for _, itemID := range contentpkg.SortedKeys(registry.Items) {
		item := registry.Items[itemID]
		itemDefinitions[item.ItemID] = itemDefinition{
			ItemID:        item.ItemID,
			DisplayName:   item.DisplayName,
			Type:          contentItemKind(item.Kind),
			Subtype:       "content_package",
			Quality:       contentItemQuality(item.Quality),
			Stackable:     item.MaxStack > 1,
			MaxStack:      item.MaxStack,
			RequiredLevel: 1,
		}
	}
	for _, lootTableID := range contentpkg.SortedKeys(registry.LootTables) {
		lootTable := registry.LootTables[lootTableID]
		worldTable := LootTableDefinition{
			LootTableID: lootTable.LootTableID,
			Entries:     make([]LootEntry, 0, len(lootTable.Entries)),
		}
		for _, entry := range lootTable.Entries {
			worldTable.Entries = append(worldTable.Entries, LootEntry{
				ItemID:            entry.ItemID,
				MinQuantity:       entry.MinQuantity,
				MaxQuantity:       entry.MaxQuantity,
				DropChancePercent: entry.DropChancePercent,
				IsGuaranteed:      entry.Guaranteed,
				Tags:              append([]string(nil), entry.Tags...),
			})
		}
		devLootTables[worldTable.LootTableID] = worldTable
	}

	for _, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		zone := registry.Zones[zoneID]
		s.zones[zone.ZoneID] = zoneDefinition{
			ID:          zone.ZoneID,
			DisplayName: zone.DisplayName,
			LevelBand:   "content",
			Bounds: zoneBoundsDefinition{
				MinX: zone.Bounds.MinX,
				MinY: zone.Bounds.MinY,
				MaxX: zone.Bounds.MaxX,
				MaxY: zone.Bounds.MaxY,
			},
			Landmarks: contentZoneLandmarks(zone),
		}
		zoneRuntime := &ContentZoneRuntime{
			ZoneID:             zone.ZoneID,
			DisplayName:        zone.DisplayName,
			RuntimeConfig:      zone.Runtime,
			EntryPointCount:    len(zone.EntryPoints),
			SpawnGroupCount:    len(zone.SpawnGroups),
			QuestProviderCount: len(zone.QuestProviders),
			ActivatedAt:        time.Now().UTC(),
		}
		s.zoneRuntimes[zone.ZoneID] = zoneRuntime
		result.ZonesActivated++
		observability.LogEvent("world-service", contentpkg.EventWorldZoneRuntimeCreated, map[string]any{
			"packageId": registry.PackageID,
			"zoneId":    zone.ZoneID,
			"tickMs":    zone.Runtime.TickMS,
		})

		for _, provider := range zone.QuestProviders {
			s.registerContentQuestProviderLocked(zone.ZoneID, provider)
			result.QuestProvidersRegistered++
			zoneRuntime.RegisteredQuestProvider++
		}
		for _, spawnGroup := range zone.SpawnGroups {
			spawned := s.registerContentSpawnGroupLocked(zone.ZoneID, spawnGroup, registry)
			result.NPCsSpawned += spawned
			zoneRuntime.SpawnedNPCCount += spawned
		}
	}

	for _, questID := range contentpkg.SortedKeys(registry.Quests) {
		quest := registry.Quests[questID]
		worldQuest := s.contentQuestDefinition(registry, quest)
		if worldQuest.ID == "" {
			continue
		}
		if _, exists := s.quests[worldQuest.ID]; !exists {
			s.quests[worldQuest.ID] = worldQuest
			s.questOrder = appendUniqueString(s.questOrder, worldQuest.ID)
			result.QuestsRegistered++
		}
	}

	s.contentActivation = result
	observability.LogEvent("world-service", contentpkg.EventPackageActivated, map[string]any{
		"packageId":                result.PackageID,
		"zonesActivated":           result.ZonesActivated,
		"npcsSpawned":              result.NPCsSpawned,
		"questProvidersRegistered": result.QuestProvidersRegistered,
		"questsRegistered":         result.QuestsRegistered,
	})
	return result
}

func (s *worldServer) registerContentQuestProviderLocked(zoneID string, provider contentpkg.QuestProviderDefinition) {
	services := make([]npcService, 0, len(provider.OfferedQuestIDs))
	for _, questID := range provider.OfferedQuestIDs {
		services = append(services, npcService{
			Type:      "quest",
			ServiceID: questID,
			Label:     "Quest",
		})
	}
	npc := friendlyNPCDefinition{
		ID:          provider.ProviderID,
		ZoneID:      zoneID,
		DisplayName: provider.DisplayName,
		Kind:        questGiverNPCKind,
		X:           provider.Position.X,
		Y:           provider.Position.Y,
		Z:           clampSpawnGroundZ(provider.Position.Z),
		AIState:     "quest_giver",
		Radius:      starterInteractRadius,
		Services:    services,
	}
	s.friendlyNPCs[npc.ID] = npc
	s.friendlyNPCOrder = appendUniqueString(s.friendlyNPCOrder, npc.ID)
	observability.LogEvent("world-service", contentpkg.EventQuestProviderRegistered, map[string]any{
		"zoneId":          zoneID,
		"providerId":      provider.ProviderID,
		"offeredQuestIds": provider.OfferedQuestIDs,
	})
}

func (s *worldServer) registerContentSpawnGroupLocked(zoneID string, group contentpkg.SpawnGroupDefinition, registry contentpkg.RuntimeContentRegistry) int {
	archetype, found := registry.NPCs[group.NPCArchetypeID]
	if !found {
		return 0
	}
	spawned := 0
	for _, spawn := range group.SpawnPoints {
		spawnID := contentSpawnEntityID(spawn.SpawnPointID)
		s.contentMobSpawns = append(s.contentMobSpawns, mobSpawnDefinition{
			ID:              spawnID,
			SpawnPointID:    spawn.SpawnPointID,
			ArchetypeID:     archetype.ArchetypeID,
			ZoneID:          zoneID,
			MobTypeID:       archetype.ArchetypeID,
			DisplayName:     archetype.DisplayName,
			Level:           archetype.Level,
			X:               spawn.Position.X,
			Y:               spawn.Position.Y,
			Z:               spawn.Position.Z,
			MaxHealth:       archetype.MaxHealth,
			AggroRadius:     archetype.AggroRange,
			AttackRange:     archetype.AttackRange,
			AttackDamage:    archetype.BaseDamage,
			AttackCadenceMs: int64(archetype.AttackIntervalMS),
			MoveSpeedPerSec: 3.2,
			LeashRadius:     archetype.LeashRange,
			RespawnDelayMs:  int64(group.RespawnSeconds) * int64(time.Second/time.Millisecond),
			Disposition:     archetype.Disposition,
			LootTableID:     group.LootTableID,
		})
		spawned++
		observability.LogEvent("world-service", contentpkg.EventReferenceResolved, map[string]any{
			"sourceKind": "spawn_group",
			"sourceId":   group.SpawnGroupID,
			"targetKind": "npc_archetype",
			"targetId":   archetype.ArchetypeID,
		})
	}
	return spawned
}

func (s *worldServer) contentQuestDefinition(registry contentpkg.RuntimeContentRegistry, quest contentpkg.QuestDefinition) questDefinition {
	providerID := providerForQuest(registry, quest.QuestID)
	worldQuest := questDefinition{
		ID:              quest.QuestID,
		ZoneID:          zoneForQuest(registry, quest.QuestID),
		Title:           quest.DisplayName,
		ObjectiveText:   quest.Summary,
		GiverNPCID:      providerID,
		TurnInNPCID:     providerID,
		TargetCount:     1,
		RewardItems:     contentQuestRewards(quest, registry),
		LevelBand:       fmt.Sprintf("%d", contentMaxInt(quest.RequiredLevel, 1)),
		PrerequisiteIDs: append([]string(nil), quest.PrerequisiteQuestIDs...),
	}
	for _, node := range quest.ObjectiveGraph.Nodes {
		switch node.Kind {
		case "kill_npc":
			worldQuest.ObjectiveType = objectiveKill
			worldQuest.TargetMobType = node.TargetID
			worldQuest.TargetCount = contentMaxInt(node.RequiredCount, 1)
			return worldQuest
		case "collect_item":
			if worldQuest.ObjectiveType == "" {
				worldQuest.ObjectiveType = objectiveCollect
				worldQuest.TargetItemID = node.TargetID
				worldQuest.TargetCount = contentMaxInt(node.RequiredCount, 1)
			}
		case "talk_provider":
			if worldQuest.ObjectiveType == "" {
				worldQuest.ObjectiveType = objectiveTalk
				worldQuest.TargetEntityID = node.TargetID
				worldQuest.TargetCount = contentMaxInt(node.RequiredCount, 1)
			}
		}
	}
	if worldQuest.ObjectiveType == "" {
		worldQuest.ObjectiveType = objectiveTalk
		worldQuest.TargetEntityID = providerID
	}
	return worldQuest
}

func contentQuestRewards(quest contentpkg.QuestDefinition, registry contentpkg.RuntimeContentRegistry) []itemRewardDefinition {
	rewards := make([]itemRewardDefinition, 0, len(quest.Rewards))
	for _, reward := range quest.Rewards {
		item := registry.Items[reward.ItemID]
		rewards = append(rewards, itemRewardDefinition{
			ItemID:      reward.ItemID,
			DisplayName: item.DisplayName,
			StackCount:  reward.Quantity,
		})
	}
	return rewards
}

func contentZoneLandmarks(zone contentpkg.ZoneDefinition) []zonePointDefinition {
	points := make([]zonePointDefinition, 0, len(zone.EntryPoints)+len(zone.QuestProviders))
	for _, entry := range zone.EntryPoints {
		points = append(points, zonePointDefinition{
			ID:          entry.EntryID,
			DisplayName: entry.EntryID,
			Type:        "entry",
			X:           entry.Position.X,
			Y:           entry.Position.Y,
		})
	}
	for _, provider := range zone.QuestProviders {
		points = append(points, zonePointDefinition{
			ID:          provider.ProviderID,
			DisplayName: provider.DisplayName,
			Type:        "quest_provider",
			X:           provider.Position.X,
			Y:           provider.Position.Y,
		})
	}
	return points
}

func providerForQuest(registry contentpkg.RuntimeContentRegistry, questID string) string {
	for _, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		zone := registry.Zones[zoneID]
		for _, provider := range zone.QuestProviders {
			for _, offeredQuestID := range provider.OfferedQuestIDs {
				if offeredQuestID == questID {
					return provider.ProviderID
				}
			}
		}
	}
	return ""
}

func zoneForQuest(registry contentpkg.RuntimeContentRegistry, questID string) string {
	for _, zoneID := range contentpkg.SortedKeys(registry.Zones) {
		zone := registry.Zones[zoneID]
		for _, provider := range zone.QuestProviders {
			for _, offeredQuestID := range provider.OfferedQuestIDs {
				if offeredQuestID == questID {
					return zone.ZoneID
				}
			}
		}
	}
	return ""
}

func contentSpawnEntityID(spawnPointID string) string {
	trimmed := strings.TrimSpace(spawnPointID)
	if strings.HasPrefix(trimmed, "spawn_") {
		return "npc_content_" + strings.TrimPrefix(trimmed, "spawn_")
	}
	return "npc_content_" + trimmed
}

func contentItemKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case itemTypeWeapon, itemTypeArmor, itemTypeConsumable, itemTypeMaterial, itemTypeQuest, itemTypeJunk:
		return strings.ToLower(strings.TrimSpace(kind))
	case "currency":
		return itemTypeMaterial
	case "equipment":
		return itemTypeArmor
	default:
		return itemTypeMaterial
	}
}

func contentItemQuality(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case itemQualityPoor, itemQualityCommon:
		return strings.ToLower(strings.TrimSpace(quality))
	default:
		return itemQualityCommon
	}
}

func appendUniqueString(values []string, next string) []string {
	for _, existing := range values {
		if existing == next {
			return values
		}
	}
	return append(values, next)
}

func contentMaxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

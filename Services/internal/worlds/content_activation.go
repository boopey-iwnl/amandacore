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
	MapID                   string
	MapBounds               contentpkg.ZoneBounds
	AdjacentZoneIDs         []string
	TransitionHints         []ZoneTransitionHint
	StreamingCells          []ZoneStreamingCell
	EntryPointCount         int
	SpawnGroupCount         int
	QuestProviderCount      int
	TransitionCount         int
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
	TransitionsLoaded        int
	MapExportsLoaded         int
	StreamingCellsLoaded     int
	HandoffGatesRegistered   int
	QuestsRegistered         int
	Errors                   []contentpkg.ContentValidationError
}

type ZoneTransitionHint struct {
	TransitionID       string
	DisplayName        string
	TargetZoneID       string
	DestinationEntryID string
	StreamingCellID    string
	Hint               string
	X                  float64
	Y                  float64
	Z                  float64
	Radius             float64
}

type ZoneStreamingCell struct {
	CellID      string
	DisplayName string
	Bounds      contentpkg.ZoneBounds
	Priority    int
	Tags        []string
}

func (s *worldServer) loadConfiguredContentPackageLocked(contentPackagePath string) {
	if strings.TrimSpace(contentPackagePath) == noContentPackagePath {
		return
	}
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
			"map_exports": len(registry.MapExports),
			"npcs":        len(registry.NPCs),
			"items":       len(registry.Items),
			"loot_tables": len(registry.LootTables),
			"quests":      len(registry.Quests),
			"abilities":   len(registry.Abilities),
			"auras":       len(registry.Auras),
			"handoff":     len(registry.HandoffGates),
		},
	}

	for _, itemID := range contentpkg.SortedKeys(registry.Items) {
		item := registry.Items[itemID]
		if _, exists := itemDefinitions[item.ItemID]; exists {
			continue
		}
		itemDefinitions[item.ItemID] = itemDefinition{
			ItemID:        item.ItemID,
			DisplayName:   item.DisplayName,
			Description:   item.Description,
			Kind:          contentItemKind(item.Kind),
			Type:          contentItemKind(item.Kind),
			Subtype:       "content_package",
			Quality:       contentItemQuality(item.Quality),
			Stackable:     item.MaxStack > 1,
			MaxStack:      item.MaxStack,
			RequiredLevel: 1,
			Tags:          append([]string(nil), item.Tags...),
		}
	}
	for _, lootTableID := range contentpkg.SortedKeys(registry.LootTables) {
		lootTable := registry.LootTables[lootTableID]
		if _, exists := devLootTables[lootTable.LootTableID]; exists {
			continue
		}
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
		mapExport, hasMapExport := registry.MapExportByZone[zone.ZoneID]
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
			Landmarks:   contentZoneLandmarks(zone),
			Transitions: contentZoneTransitions(zone),
		}
		zoneRuntime := &ContentZoneRuntime{
			ZoneID:             zone.ZoneID,
			DisplayName:        zone.DisplayName,
			RuntimeConfig:      zone.Runtime,
			EntryPointCount:    len(zone.EntryPoints),
			SpawnGroupCount:    len(zone.SpawnGroups),
			QuestProviderCount: len(zone.QuestProviders),
			TransitionCount:    len(zone.Transitions),
			ActivatedAt:        time.Now().UTC(),
		}
		if hasMapExport {
			zoneRuntime.MapID = mapExport.MapID
			zoneRuntime.MapBounds = mapExport.Bounds
			zoneRuntime.AdjacentZoneIDs = contentMapAdjacentZoneIDs(mapExport)
			zoneRuntime.TransitionHints = contentMapTransitionHints(mapExport)
			zoneRuntime.StreamingCells = contentMapStreamingCells(mapExport)
			result.MapExportsLoaded++
			result.StreamingCellsLoaded += len(mapExport.StreamingCells)
		}
		s.zoneRuntimes[zone.ZoneID] = zoneRuntime
		result.ZonesActivated++
		result.TransitionsLoaded += len(zone.Transitions)
		observability.LogEvent("world-service", contentpkg.EventWorldZoneRuntimeCreated, map[string]any{
			"packageId":          registry.PackageID,
			"zoneId":             zone.ZoneID,
			"tickMs":             zone.Runtime.TickMS,
			"transitionCount":    len(zone.Transitions),
			"mapId":              zoneRuntime.MapID,
			"streamingCellCount": len(zoneRuntime.StreamingCells),
			"adjacentZoneCount":  len(zoneRuntime.AdjacentZoneIDs),
		})

		for _, provider := range zone.QuestProviders {
			s.registerContentQuestProviderLocked(zone.ZoneID, provider)
			result.QuestProvidersRegistered++
			zoneRuntime.RegisteredQuestProvider++
		}
		for _, gate := range zone.HandoffGates {
			if s.registerContentHandoffGateLocked(gate, registry) {
				result.HandoffGatesRegistered++
			}
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
		"transitionsLoaded":        result.TransitionsLoaded,
		"mapExportsLoaded":         result.MapExportsLoaded,
		"streamingCellsLoaded":     result.StreamingCellsLoaded,
		"handoffGatesRegistered":   result.HandoffGatesRegistered,
		"questsRegistered":         result.QuestsRegistered,
	})
	observability.LogEvent("world-service", contentpkg.EventZonesRegistered, map[string]any{
		"packageId": result.PackageID,
		"count":     result.ZonesActivated,
	})
	observability.LogEvent("world-service", contentpkg.EventHandoffGatesRegistered, map[string]any{
		"packageId": result.PackageID,
		"count":     result.HandoffGatesRegistered,
	})
	return result
}

func (s *worldServer) registerContentHandoffGateLocked(gate contentpkg.HandoffGateDefinition, registry contentpkg.RuntimeContentRegistry) bool {
	arrival, found := findContentSpawnPoint(registry, gate.ArrivalSpawnPointID)
	if !found {
		return false
	}
	if s.handoffGates == nil {
		s.handoffGates = defaultZoneHandoffGateDefinitions()
	}
	worldGate := ZoneHandoffGateDefinition{
		TransitionID:       gate.GateID,
		FromZoneID:         gate.SourceZoneID,
		ToZoneID:           gate.DestinationZoneID,
		GateX:              gate.Trigger.Center.X,
		GateY:              gate.Trigger.Center.Y,
		Radius:             gate.Trigger.Radius,
		ArrivalX:           arrival.Position.X,
		ArrivalY:           arrival.Position.Y,
		ArrivalZ:           clampSpawnGroundZ(arrival.Position.Z),
		Enabled:            gate.Enabled,
		RetryableWhenFails: gate.RetryableWhenUnavailable,
	}
	s.handoffGates[worldGate.TransitionID] = worldGate
	observability.LogEvent("world-service", contentpkg.EventHandoffGateRegistered, map[string]any{
		"packageId":         registry.PackageID,
		"gateId":            gate.GateID,
		"sourceZoneId":      gate.SourceZoneID,
		"destinationZoneId": gate.DestinationZoneID,
		"arrivalSpawnId":    gate.ArrivalSpawnPointID,
	})
	return true
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

func findContentSpawnPoint(registry contentpkg.RuntimeContentRegistry, spawnPointID string) (contentpkg.ZoneSpawnPointDefinition, bool) {
	if spawn, found := registry.SpawnPoints[spawnPointID]; found {
		return spawn, true
	}
	for _, zone := range registry.Zones {
		for _, spawn := range zone.SpawnPoints {
			if spawn.SpawnPointID == spawnPointID {
				return spawn, true
			}
		}
		for _, group := range zone.SpawnGroups {
			for _, spawn := range group.SpawnPoints {
				if spawn.SpawnPointID == spawnPointID {
					return contentpkg.ZoneSpawnPointDefinition{
						SpawnPointID: spawn.SpawnPointID,
						Purpose:      "npc_spawn",
						Position:     spawn.Position,
						FacingYaw:    spawn.FacingYaw,
					}, true
				}
			}
		}
	}
	return contentpkg.ZoneSpawnPointDefinition{}, false
}

func (s *worldServer) contentQuestDefinition(registry contentpkg.RuntimeContentRegistry, quest contentpkg.QuestDefinition) questDefinition {
	providerID := providerForQuest(registry, quest.QuestID)
	worldQuest := questDefinition{
		ID:              quest.QuestID,
		ZoneID:          zoneForQuest(registry, quest.QuestID),
		Title:           quest.DisplayName,
		Summary:         quest.Summary,
		ObjectiveText:   quest.Summary,
		GiverNPCID:      providerID,
		TurnInNPCID:     providerID,
		TargetCount:     1,
		RewardItems:     contentQuestRewards(quest, registry),
		LevelBand:       fmt.Sprintf("%d", contentMaxInt(quest.RequiredLevel, 1)),
		PrerequisiteIDs: append([]string(nil), quest.PrerequisiteQuestIDs...),
		Tags:            append([]string(nil), quest.Tags...),
		ObjectiveGraph:  contentObjectiveGraph(quest.ObjectiveGraph),
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

func contentObjectiveGraph(graph contentpkg.QuestObjectiveGraph) questObjectiveGraph {
	nodes := make([]questObjectiveNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		worldNode := questObjectiveNode{
			NodeID:      node.NodeID,
			TargetCount: contentMaxInt(node.RequiredCount, 1),
			DependsOn:   append([]string(nil), node.DependsOn...),
		}
		switch node.Kind {
		case "kill_npc":
			worldNode.Kind = objectiveKindKillNPC
			worldNode.TargetNpcArchetype = node.TargetID
		case "collect_item":
			worldNode.Kind = objectiveKindCollectItem
			worldNode.TargetItemID = node.TargetID
		case "talk_provider":
			worldNode.Kind = objectiveKindInteractWithEntity
			worldNode.TargetEntityID = node.TargetID
		default:
			worldNode.Kind = objectiveKindInteractWithEntity
			worldNode.TargetEntityID = node.TargetID
		}
		nodes = append(nodes, worldNode)
	}
	if len(nodes) > 0 {
		dependents := map[string]bool{}
		for _, node := range nodes {
			for _, dependency := range node.DependsOn {
				dependents[dependency] = true
			}
		}
		for index := range nodes {
			if !dependents[nodes[index].NodeID] {
				nodes[index].Terminal = true
			}
		}
	}
	return questObjectiveGraph{Nodes: nodes}
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

func contentZoneTransitions(zone contentpkg.ZoneDefinition) []zonePointDefinition {
	points := make([]zonePointDefinition, 0, len(zone.Transitions))
	for _, transition := range zone.Transitions {
		points = append(points, zonePointDefinition{
			ID:          transition.TransitionID,
			DisplayName: transition.DisplayName,
			Type:        "zone_transition",
			X:           transition.Position.X,
			Y:           transition.Position.Y,
		})
	}
	return points
}

func contentMapAdjacentZoneIDs(export contentpkg.MapExportDefinition) []string {
	zoneIDs := make([]string, 0, len(export.AdjacentZones))
	seen := map[string]struct{}{}
	for _, adjacent := range export.AdjacentZones {
		if adjacent.ZoneID == "" {
			continue
		}
		if _, exists := seen[adjacent.ZoneID]; exists {
			continue
		}
		seen[adjacent.ZoneID] = struct{}{}
		zoneIDs = append(zoneIDs, adjacent.ZoneID)
	}
	return zoneIDs
}

func contentMapTransitionHints(export contentpkg.MapExportDefinition) []ZoneTransitionHint {
	hints := make([]ZoneTransitionHint, 0, len(export.TransitionPoints))
	for _, transition := range export.TransitionPoints {
		hints = append(hints, ZoneTransitionHint{
			TransitionID:       transition.TransitionID,
			DisplayName:        transition.DisplayName,
			TargetZoneID:       transition.TargetZoneID,
			DestinationEntryID: transition.DestinationEntryID,
			StreamingCellID:    transition.StreamingCellID,
			Hint:               transition.Hint,
			X:                  transition.Position.X,
			Y:                  transition.Position.Y,
			Z:                  transition.Position.Z,
			Radius:             transition.Radius,
		})
	}
	return hints
}

func contentMapStreamingCells(export contentpkg.MapExportDefinition) []ZoneStreamingCell {
	cells := make([]ZoneStreamingCell, 0, len(export.StreamingCells))
	for _, cell := range export.StreamingCells {
		cells = append(cells, ZoneStreamingCell{
			CellID:      cell.CellID,
			DisplayName: cell.DisplayName,
			Bounds:      cell.Bounds,
			Priority:    cell.Priority,
			Tags:        append([]string(nil), cell.Tags...),
		})
	}
	return cells
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
	case itemTypeCurrency, "currency", "currencytoken":
		return itemTypeCurrency
	case itemTypeEquipment, "equipment", "equipmentplaceholder":
		return itemTypeEquipment
	case "craftingmaterial":
		return itemTypeMaterial
	case "questitem":
		return itemTypeQuest
	case "misc":
		return itemTypeJunk
	default:
		return itemTypeMaterial
	}
}

func contentItemQuality(quality string) string {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case itemQualityPoor, itemQualityCommon, itemQualityUncommon, itemQualityRare:
		return strings.ToLower(strings.TrimSpace(quality))
	case itemQualityEpicPlaceholder, "epicplaceholder":
		return itemQualityEpicPlaceholder
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

package worlds

import "amandacore/services/internal/platform"

var tallowdeepZoneMap = zoneMapDefinition{
	ZoneID:      dungeonZoneID,
	DisplayName: "Tallowdeep Sluice",
	MinX:        0,
	MinY:        0,
	MaxX:        180,
	MaxY:        60,
	Roads: []mapRoadDefinition{
		{
			ID:          "tallowdeep_main_run",
			DisplayName: "Sluice Run",
			Points: []mapPointDefinition{
				{X: 12, Y: 12},
				{X: 40, Y: 15},
				{X: 82, Y: 20},
				{X: 116, Y: 33},
				{X: 148, Y: 34},
				{X: 166, Y: 34},
			},
		},
	},
	Landmarks: []mapLandmarkDefinition{
		{ID: "tallowdeep_entry", DisplayName: "Entry Channel", Kind: "dungeon_entry", X: 12, Y: 12},
		{ID: "tallowdeep_pressure_walk", DisplayName: "Pressure Walk", Kind: "route", X: 116, Y: 33},
		{ID: "tallowdeep_exit", DisplayName: "Exit Winch", Kind: "dungeon_exit", X: 166, Y: 34},
	},
}

var tallowdeepNavigationAreas = []navigationAreaDefinition{
	{ID: "tallowdeep_entry_channel", DisplayName: "Entry Channel", Kind: "dungeon_route", CenterX: 40, CenterY: 15, Radius: 18, RouteHintText: "Advance along the sluice run."},
	{ID: "tallowdeep_pressure_walk", DisplayName: "Pressure Walk", Kind: "dungeon_route", CenterX: 116, CenterY: 33, Radius: 18, RouteHintText: "Follow the pressure walk toward the warden."},
}

func (s *worldServer) buildZoneMapResponse(session *worldSessionState) map[string]any {
	zoneID := ""
	if session != nil {
		zoneID = session.ZoneID
	}
	if zoneID == dungeonZoneID {
		return buildZoneMapPayload(tallowdeepZoneMap)
	}
	if zoneID == "" || zoneID == defaultZoneID {
		return buildZoneMapPayload(stonewakeZoneMap)
	}

	if zone, ok := s.zones[zoneID]; ok {
		roads := make([]map[string]any, 0, len(zone.Roads))
		for _, road := range zone.Roads {
			roads = append(roads, map[string]any{
				"id":          road.ID,
				"displayName": road.DisplayName,
				"points":      road.Points,
			})
		}
		landmarks := make([]map[string]any, 0, len(zone.Landmarks)+len(zone.Transitions))
		for _, landmark := range append(zone.Landmarks, zone.Transitions...) {
			landmarks = append(landmarks, map[string]any{
				"id":          landmark.ID,
				"displayName": landmark.DisplayName,
				"kind":        landmark.Type,
				"x":           landmark.X,
				"y":           landmark.Y,
			})
		}
		return map[string]any{
			"zoneId":      zone.ID,
			"displayName": zone.DisplayName,
			"bounds": map[string]float64{
				"minX": zone.Bounds.MinX,
				"minY": zone.Bounds.MinY,
				"maxX": zone.Bounds.MaxX,
				"maxY": zone.Bounds.MaxY,
			},
			"roads":     roads,
			"landmarks": landmarks,
		}
	}

	return map[string]any{
		"zoneId":      defaultZoneID,
		"displayName": "Stonewake Vale",
		"bounds": map[string]float64{
			"minX": stonewakeZoneMap.MinX,
			"minY": stonewakeZoneMap.MinY,
			"maxX": stonewakeZoneMap.MaxX,
			"maxY": stonewakeZoneMap.MaxY,
		},
		"roads":     []map[string]any{},
		"landmarks": []map[string]any{},
	}
}

func buildZoneMapPayload(zone zoneMapDefinition) map[string]any {
	roads := make([]map[string]any, 0, len(zone.Roads))
	for _, road := range zone.Roads {
		roads = append(roads, map[string]any{
			"id":          road.ID,
			"displayName": road.DisplayName,
			"points":      road.Points,
		})
	}

	landmarks := make([]map[string]any, 0, len(zone.Landmarks))
	for _, landmark := range zone.Landmarks {
		landmarks = append(landmarks, map[string]any{
			"id":          landmark.ID,
			"displayName": landmark.DisplayName,
			"kind":        landmark.Kind,
			"x":           landmark.X,
			"y":           landmark.Y,
		})
	}

	return map[string]any{
		"zoneId":      zone.ZoneID,
		"displayName": zone.DisplayName,
		"bounds": map[string]float64{
			"minX": zone.MinX,
			"minY": zone.MinY,
			"maxX": zone.MaxX,
			"maxY": zone.MaxY,
		},
		"roads":     roads,
		"landmarks": landmarks,
	}
}

func (s *worldServer) buildNavigationAreasResponse(session *worldSessionState) []map[string]any {
	source := stonewakeNavigationAreas
	if session != nil && session.ZoneID == dungeonZoneID {
		source = tallowdeepNavigationAreas
	}
	areas := make([]map[string]any, 0, len(source))
	for _, area := range source {
		areas = append(areas, map[string]any{
			"areaId":         area.ID,
			"displayName":    area.DisplayName,
			"kind":           area.Kind,
			"centerX":        area.CenterX,
			"centerY":        area.CenterY,
			"radius":         area.Radius,
			"routeHintText":  area.RouteHintText,
			"questIds":       platform.NormalizeStringIDs(area.QuestIDs),
			"targetMobType":  area.TargetMobType,
			"targetEntityId": area.TargetEntityID,
		})
	}
	return areas
}

func (s *worldServer) buildMapMarkersResponse(session *worldSessionState) []map[string]any {
	if session != nil && session.ZoneID == dungeonZoneID {
		return []map[string]any{
			{
				"id":          "dungeon_exit_" + npcTallowdeepExitID,
				"displayName": "Exit Winch",
				"kind":        "dungeon_exit",
				"entityId":    npcTallowdeepExitID,
				"x":           166.0,
				"y":           34.0,
			},
			{
				"id":            "dungeon_boss_tallowdeep",
				"displayName":   "Sluice Warden Platform",
				"kind":          "tracked_objective",
				"questId":       dungeonQuestTallowdeepID,
				"areaId":        "tds_boss_platform",
				"x":             148.0,
				"y":             34.0,
				"radius":        18.0,
				"routeHintText": "Push through the lower sluice and defeat the warden.",
			},
		}
	}
	markers := make([]map[string]any, 0, len(s.friendlyNPCOrder)+len(stonewakeNavigationAreas))

	for _, npcID := range s.friendlyNPCOrder {
		npc := s.friendlyNPCs[npcID]
		markerKind, questID := s.markerKindForNPCLocked(session, npc)
		markers = append(markers, map[string]any{
			"id":          "npc_" + npc.ID,
			"displayName": npc.DisplayName,
			"kind":        markerKind,
			"questId":     questID,
			"entityId":    npc.ID,
			"x":           npc.X,
			"y":           npc.Y,
		})
	}

	for _, questID := range session.TrackedQuestIDs {
		quest, found := s.quests[questID]
		if !found {
			continue
		}
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State != questStateActive {
			continue
		}
		area, found := s.findNavigationAreaForQuest(quest)
		if found {
			markers = append(markers, map[string]any{
				"id":            "objective_" + quest.ID,
				"displayName":   area.DisplayName,
				"kind":          "tracked_objective",
				"questId":       quest.ID,
				"areaId":        area.ID,
				"x":             area.CenterX,
				"y":             area.CenterY,
				"radius":        area.Radius,
				"routeHintText": area.RouteHintText,
			})
			continue
		}
		if quest.MarkerX != 0 || quest.MarkerY != 0 {
			markers = append(markers, map[string]any{
				"id":            "objective_" + quest.ID,
				"displayName":   quest.Title,
				"kind":          "tracked_objective",
				"questId":       quest.ID,
				"areaId":        quest.ID + "_marker",
				"x":             quest.MarkerX,
				"y":             quest.MarkerY,
				"radius":        starterInteractRadius,
				"routeHintText": "Follow the road marker toward the objective.",
			})
		}
	}

	return markers
}

func (s *worldServer) markerKindForNPCLocked(session *worldSessionState, npc friendlyNPCDefinition) (string, string) {
	isTrainer := false
	isVendor := false
	for _, service := range npc.Services {
		if service.Type == "trainer" {
			isTrainer = true
		}
		if service.Type == "vendor" {
			isVendor = true
		}
	}

	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateCompleted && quest.TurnInNPCID == npc.ID {
			return "quest_turn_in", quest.ID
		}
	}

	for _, questID := range s.questOrder {
		quest := s.quests[questID]
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateActive &&
			(quest.TargetEntityID == npc.ID || quest.TurnInNPCID == npc.ID) &&
			(quest.ObjectiveType == objectiveTalk || quest.ObjectiveType == objectiveTrainer || quest.ObjectiveType == objectiveUse || quest.ObjectiveType == objectiveExplore) {
			return "quest_objective", quest.ID
		}
	}

	for _, service := range npc.Services {
		if service.Type != "quest" {
			continue
		}
		quest, found := s.quests[service.ServiceID]
		if !found {
			continue
		}
		progress := s.normalizeQuestProgress(quest, session.QuestProgress[quest.ID])
		if progress.State == questStateNotStarted && s.prerequisitesMetLocked(session, quest) {
			return "quest_available", quest.ID
		}
	}

	if isTrainer {
		return "trainer", ""
	}
	if isVendor {
		return "vendor", ""
	}
	return "service", ""
}

func buildZoneMapPayload(zoneMap zoneMapDefinition) map[string]any {
	roads := make([]map[string]any, 0, len(zoneMap.Roads))
	for _, road := range zoneMap.Roads {
		roads = append(roads, map[string]any{
			"id":          road.ID,
			"displayName": road.DisplayName,
			"points":      road.Points,
		})
	}
	landmarks := make([]map[string]any, 0, len(zoneMap.Landmarks))
	for _, landmark := range zoneMap.Landmarks {
		landmarks = append(landmarks, map[string]any{
			"id":          landmark.ID,
			"displayName": landmark.DisplayName,
			"kind":        landmark.Kind,
			"x":           landmark.X,
			"y":           landmark.Y,
		})
	}
	return map[string]any{
		"zoneId":      zoneMap.ZoneID,
		"displayName": zoneMap.DisplayName,
		"bounds": map[string]float64{
			"minX": zoneMap.MinX,
			"minY": zoneMap.MinY,
			"maxX": zoneMap.MaxX,
			"maxY": zoneMap.MaxY,
		},
		"roads":     roads,
		"landmarks": landmarks,
	}
}

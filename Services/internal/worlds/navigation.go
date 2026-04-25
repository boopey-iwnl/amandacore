package worlds

import "amandacore/services/internal/platform"

func (s *worldServer) buildZoneMapResponse() map[string]any {
	roads := make([]map[string]any, 0, len(stonewakeZoneMap.Roads))
	for _, road := range stonewakeZoneMap.Roads {
		roads = append(roads, map[string]any{
			"id":          road.ID,
			"displayName": road.DisplayName,
			"points":      road.Points,
		})
	}

	landmarks := make([]map[string]any, 0, len(stonewakeZoneMap.Landmarks))
	for _, landmark := range stonewakeZoneMap.Landmarks {
		landmarks = append(landmarks, map[string]any{
			"id":          landmark.ID,
			"displayName": landmark.DisplayName,
			"kind":        landmark.Kind,
			"x":           landmark.X,
			"y":           landmark.Y,
		})
	}

	return map[string]any{
		"zoneId":      stonewakeZoneMap.ZoneID,
		"displayName": stonewakeZoneMap.DisplayName,
		"bounds": map[string]float64{
			"minX": stonewakeZoneMap.MinX,
			"minY": stonewakeZoneMap.MinY,
			"maxX": stonewakeZoneMap.MaxX,
			"maxY": stonewakeZoneMap.MaxY,
		},
		"roads":     roads,
		"landmarks": landmarks,
	}
}

func (s *worldServer) buildNavigationAreasResponse() []map[string]any {
	areas := make([]map[string]any, 0, len(stonewakeNavigationAreas))
	for _, area := range stonewakeNavigationAreas {
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
		if !found {
			continue
		}
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

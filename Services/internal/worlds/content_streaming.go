package worlds

func (s *worldServer) buildStreamingHintsResponse(session *worldSessionState) map[string]any {
	zoneID := ""
	if session != nil {
		zoneID = session.ZoneID
	}
	runtime := s.zoneRuntimes[zoneID]
	if runtime == nil || runtime.MapID == "" {
		return map[string]any{
			"enabled": false,
			"zoneId":  zoneID,
		}
	}

	transitionHints := make([]map[string]any, 0, len(runtime.TransitionHints))
	for _, hint := range runtime.TransitionHints {
		transitionHints = append(transitionHints, map[string]any{
			"transitionId":       hint.TransitionID,
			"displayName":        hint.DisplayName,
			"targetZoneId":       hint.TargetZoneID,
			"destinationEntryId": hint.DestinationEntryID,
			"streamingCellId":    hint.StreamingCellID,
			"hint":               hint.Hint,
			"position": map[string]float64{
				"x": hint.X,
				"y": hint.Y,
				"z": hint.Z,
			},
			"radius": hint.Radius,
		})
	}

	streamingCells := make([]map[string]any, 0, len(runtime.StreamingCells))
	for _, cell := range runtime.StreamingCells {
		streamingCells = append(streamingCells, map[string]any{
			"cellId":      cell.CellID,
			"displayName": cell.DisplayName,
			"priority":    cell.Priority,
			"tags":        append([]string(nil), cell.Tags...),
			"bounds": map[string]float64{
				"minX": cell.Bounds.MinX,
				"minY": cell.Bounds.MinY,
				"minZ": cell.Bounds.MinZ,
				"maxX": cell.Bounds.MaxX,
				"maxY": cell.Bounds.MaxY,
				"maxZ": cell.Bounds.MaxZ,
			},
		})
	}

	return map[string]any{
		"enabled":         true,
		"zoneId":          zoneID,
		"mapId":           runtime.MapID,
		"adjacentZoneIds": append([]string(nil), runtime.AdjacentZoneIDs...),
		"bounds": map[string]float64{
			"minX": runtime.MapBounds.MinX,
			"minY": runtime.MapBounds.MinY,
			"minZ": runtime.MapBounds.MinZ,
			"maxX": runtime.MapBounds.MaxX,
			"maxY": runtime.MapBounds.MaxY,
			"maxZ": runtime.MapBounds.MaxZ,
		},
		"transitionHints": transitionHints,
		"streamingCells":  streamingCells,
	}
}

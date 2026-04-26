package worlds

import (
	"fmt"
	"math"

	contentpkg "amandacore/services/internal/content"
)

type contentZoneTransitionResult struct {
	TransitionID       string
	FromZoneID         string
	ToZoneID           string
	DestinationEntryID string
	Completed          bool
}

func (s *worldServer) applyContentZoneTransitionsLocked(session *worldSessionState) (contentZoneTransitionResult, error) {
	if session == nil || s.contentRegistry == nil {
		return contentZoneTransitionResult{}, nil
	}
	zone, found := s.contentRegistry.Zones[session.ZoneID]
	if !found {
		return contentZoneTransitionResult{}, nil
	}
	for _, transition := range zone.Transitions {
		if distance2D(session.X, session.Y, transition.Position.X, transition.Position.Y) > transition.Radius {
			continue
		}
		return s.completeContentZoneTransitionLocked(session, transition)
	}
	return contentZoneTransitionResult{}, nil
}

func (s *worldServer) completeContentZoneTransitionLocked(session *worldSessionState, transition contentpkg.ZoneTransitionDefinition) (contentZoneTransitionResult, error) {
	result := contentZoneTransitionResult{
		TransitionID:       transition.TransitionID,
		FromZoneID:         session.ZoneID,
		ToZoneID:           transition.TargetZoneID,
		DestinationEntryID: transition.DestinationEntryID,
	}
	s.emitWorldEventLocked(contentpkg.EventWorldZoneTransitionStarted, map[string]any{
		"characterId":        session.CharacterID,
		"transitionId":       transition.TransitionID,
		"fromZoneId":         session.ZoneID,
		"targetZoneId":       transition.TargetZoneID,
		"destinationEntryId": transition.DestinationEntryID,
	})

	destination, found := s.contentEntryPointLocked(transition.TargetZoneID, transition.DestinationEntryID)
	if !found {
		s.emitWorldEventLocked(contentpkg.EventWorldZoneTransitionRejected, map[string]any{
			"characterId":        session.CharacterID,
			"transitionId":       transition.TransitionID,
			"fromZoneId":         session.ZoneID,
			"targetZoneId":       transition.TargetZoneID,
			"destinationEntryId": transition.DestinationEntryID,
			"reason":             "destination_entry_missing",
		})
		return result, fmt.Errorf("destination entry %s in zone %s is not available", transition.DestinationEntryID, transition.TargetZoneID)
	}

	session.ZoneID = transition.TargetZoneID
	session.X = destination.Position.X
	session.Y = destination.Position.Y
	session.Z = clampSpawnGroundZ(destination.Position.Z)
	session.CurrentTargetID = ""
	session.AutoAttackActive = false
	result.Completed = true
	s.emitWorldEventLocked(contentpkg.EventWorldZoneTransitionDone, map[string]any{
		"characterId":        session.CharacterID,
		"transitionId":       transition.TransitionID,
		"fromZoneId":         result.FromZoneID,
		"targetZoneId":       result.ToZoneID,
		"destinationEntryId": result.DestinationEntryID,
		"x":                  session.X,
		"y":                  session.Y,
		"z":                  session.Z,
	})
	return result, nil
}

func (s *worldServer) contentEntryPointLocked(zoneID string, entryID string) (contentpkg.ZoneEntryPoint, bool) {
	if s == nil || s.contentRegistry == nil {
		return contentpkg.ZoneEntryPoint{}, false
	}
	zone, found := s.contentRegistry.Zones[zoneID]
	if !found {
		return contentpkg.ZoneEntryPoint{}, false
	}
	for _, entry := range zone.EntryPoints {
		if entry.EntryID == entryID {
			return entry, true
		}
	}
	return contentpkg.ZoneEntryPoint{}, false
}

func contentTransitionDistance(session *worldSessionState, transition contentpkg.ZoneTransitionDefinition) float64 {
	if session == nil {
		return math.Inf(1)
	}
	return distance2D(session.X, session.Y, transition.Position.X, transition.Position.Y)
}

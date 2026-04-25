package worlds

import (
	"fmt"

	"amandacore/services/internal/observability"
)

func (s *worldServer) setTargetLocked(session *worldSessionState, targetID string) error {
	if targetID == "" {
		s.clearTargetLocked(session, "manual")
		return nil
	}

	targetMob := s.findMobForSessionLocked(session, targetID)
	if targetMob == nil {
		if friendly, ok := s.findFriendlyNPCDefinition(targetID); ok && friendly.ZoneID == session.ZoneID {
			if distance2D(session.X, session.Y, friendly.X, friendly.Y) > playerTargetRange {
				observability.LogEvent("world-service", "world.target_rejected", map[string]any{
					"worldSessionToken": session.Token,
					"characterId":       session.CharacterID,
					"targetId":          targetID,
					"reason":            "friendly_out_of_range",
				})
				return fmt.Errorf("target is out of range")
			}

			session.CurrentTargetID = targetID
			observability.LogEvent("world-service", "world.friendly_target_validated", map[string]any{
				"worldSessionToken": session.Token,
				"accountId":         session.AccountID,
				"characterId":       session.CharacterID,
				"targetId":          targetID,
				"kind":              friendly.Kind,
			})
			return nil
		}

		observability.LogEvent("world-service", "world.target_rejected", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"targetId":          targetID,
			"reason":            "target_missing",
		})
		return fmt.Errorf("target is not available")
	}

	if !targetMob.Alive || !targetMob.Targetable || targetMob.AIState == mobAIStateRespawning {
		observability.LogEvent("world-service", "world.target_rejected", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"targetId":          targetID,
			"reason":            "target_invalid",
		})
		return fmt.Errorf("target is not targetable")
	}

	if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > playerTargetRange {
		observability.LogEvent("world-service", "world.target_rejected", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"targetId":          targetID,
			"reason":            "out_of_range",
		})
		return fmt.Errorf("target is out of range")
	}

	session.CurrentTargetID = targetID
	observability.LogEvent("world-service", "world.target_validated", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"targetId":          targetID,
	})
	return nil
}

func (s *worldServer) findFriendlyNPCDefinition(targetID string) (friendlyNPCDefinition, bool) {
	friendly, ok := s.friendlyNPCs[targetID]
	return friendly, ok
}

func (s *worldServer) friendlyInRangeLocked(session *worldSessionState, targetID string) bool {
	if session == nil || targetID == "" {
		return false
	}
	friendly, ok := s.findFriendlyNPCDefinition(targetID)
	if !ok {
		return false
	}
	if friendly.ZoneID != "" && friendly.ZoneID != session.ZoneID {
		return false
	}
	radius := friendly.Radius
	if radius <= 0 {
		radius = starterInteractRadius
	}
	return distance2D(session.X, session.Y, friendly.X, friendly.Y) <= radius
}

func (s *worldServer) clearMobTargetFromAllSessionsLocked(targetID string, reason string) {
	for _, session := range s.sessionsByToken {
		if session.CurrentTargetID == targetID {
			s.clearTargetLocked(session, reason)
		}
	}
}

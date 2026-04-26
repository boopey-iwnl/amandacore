package worlds

import (
	"time"

	"amandacore/services/internal/platform"
)

func (s *worldServer) awardKillCreditLocked(session *worldSessionState, mob *mobState, reason string) error {
	if session == nil || mob == nil {
		return nil
	}
	archetypeID := mob.ArchetypeID
	if archetypeID == "" {
		archetypeID = mob.MobTypeID
	}
	if archetypeID == "" {
		return nil
	}

	credits := platform.NormalizeCharacterKillCredits(session.KillCredits)
	credit := credits[archetypeID]
	credit.ArchetypeID = archetypeID
	credit.Count++
	credit.Reason = reason
	credit.UpdatedAt = time.Now().Unix()
	credits[archetypeID] = credit
	session.KillCredits = credits

	payload := map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"entityId":          mob.ID,
		"archetypeId":       archetypeID,
		"count":             credit.Count,
		"reason":            reason,
	}
	s.emitDomainEventLocked(eventProgressionKillCredit, payload)
	s.emitStateDiffLocked(diffProgression, session.CharacterID, payload)

	if s.store == nil {
		return nil
	}

	persistStartedAt := time.Now()
	character, err := s.store.UpdateCharacterKillCredits(session.CharacterID, session.KillCredits)
	s.recordPersistenceDuration("character_kill_credit", persistStartedAt, err)
	if err != nil {
		return err
	}
	session.KillCredits = platform.NormalizeCharacterKillCredits(character.KillCredits)
	s.emitDomainEventLocked(eventProgressionKillPersisted, map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"archetypeId":       archetypeID,
		"count":             session.KillCredits[archetypeID].Count,
	})
	return nil
}

func buildKillCreditResponse(source map[string]platform.CharacterKillCredit) []map[string]any {
	credits := platform.NormalizeCharacterKillCredits(source)
	response := make([]map[string]any, 0, len(credits))
	for archetypeID, credit := range credits {
		response = append(response, map[string]any{
			"archetypeId": archetypeID,
			"count":       credit.Count,
			"reason":      credit.Reason,
			"updatedAt":   credit.UpdatedAt,
		})
	}
	return response
}

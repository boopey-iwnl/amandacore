package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	talentUnlockLevel    = 6
	balancedGripTalentID = "balanced_grip"
	hardLessonsTalentID  = "hard_lessons"
	stoutFrameTalentID   = "stout_frame"
	rallyRhythmTalentID  = "rally_rhythm"
)

type talentDefinition struct {
	ID          string
	DisplayName string
	Category    string
	Description string
	MaxRank     int
	MinLevel    int
	Passive     bool
}

var warriorTalentCatalog = []talentDefinition{
	{
		ID:          balancedGripTalentID,
		DisplayName: "Balanced Grip",
		Category:    "Armscraft",
		Description: "Your practiced grip adds 1 Strength per rank.",
		MaxRank:     2,
		MinLevel:    talentUnlockLevel,
		Passive:     true,
	},
	{
		ID:          hardLessonsTalentID,
		DisplayName: "Hard Lessons",
		Category:    "Armscraft",
		Description: "Steady Strike deals slightly more damage per rank.",
		MaxRank:     2,
		MinLevel:    talentUnlockLevel,
		Passive:     true,
	},
	{
		ID:          stoutFrameTalentID,
		DisplayName: "Stout Frame",
		Category:    "Endurance",
		Description: "Your conditioning adds 1 Stamina per rank.",
		MaxRank:     2,
		MinLevel:    talentUnlockLevel,
		Passive:     true,
	},
	{
		ID:          rallyRhythmTalentID,
		DisplayName: "Rally Rhythm",
		Category:    "Command",
		Description: "Rallying Call restores a small amount of Grit per rank.",
		MaxRank:     1,
		MinLevel:    talentUnlockLevel,
		Passive:     true,
	},
}

func talentPointsGranted(level int) int {
	if level < talentUnlockLevel {
		return 0
	}
	return 1 + ((level - talentUnlockLevel) / 2)
}

func talentPointsSpent(talents map[string]int) int {
	total := 0
	for _, rank := range platform.NormalizeTalentRanks(talents) {
		total += rank
	}
	return total
}

func findTalentDefinition(talentID string) (talentDefinition, bool) {
	for _, talent := range warriorTalentCatalog {
		if talent.ID == talentID {
			return talent, true
		}
	}
	return talentDefinition{}, false
}

func (s *worldServer) buildTalentsResponse(session *worldSessionState) map[string]any {
	talents := platform.NormalizeTalentRanks(session.Talents)
	pointsGranted := talentPointsGranted(session.Level)
	pointsSpent := talentPointsSpent(talents)
	entries := make([]map[string]any, 0, len(warriorTalentCatalog))
	for _, talent := range warriorTalentCatalog {
		rank := talents[talent.ID]
		canSelect := session.ClassID == platform.DefaultClassID &&
			session.Level >= talent.MinLevel &&
			pointsSpent < pointsGranted &&
			rank < talent.MaxRank
		requirementText := "Ready to train."
		if session.Level < talent.MinLevel {
			requirementText = fmt.Sprintf("Unlocks at level %d.", talent.MinLevel)
		} else if rank >= talent.MaxRank {
			requirementText = "Maximum rank reached."
		} else if pointsSpent >= pointsGranted {
			requirementText = "No talent points available."
		}

		entries = append(entries, map[string]any{
			"id":              talent.ID,
			"displayName":     talent.DisplayName,
			"category":        talent.Category,
			"description":     talent.Description,
			"rank":            rank,
			"maxRank":         talent.MaxRank,
			"minLevel":        talent.MinLevel,
			"passive":         talent.Passive,
			"canSelect":       canSelect,
			"requirementText": requirementText,
		})
	}

	return map[string]any{
		"unlocked":        session.Level >= talentUnlockLevel,
		"unlockLevel":     talentUnlockLevel,
		"pointsGranted":   pointsGranted,
		"pointsSpent":     pointsSpent,
		"pointsAvailable": maxInt(0, pointsGranted-pointsSpent),
		"categories":      []string{"Armscraft", "Endurance", "Command"},
		"talents":         entries,
	}
}

func (s *worldServer) selectTalentLocked(session *worldSessionState, talentID string) error {
	if session == nil {
		return fmt.Errorf("world session token was not found")
	}
	if session.ClassID != platform.DefaultClassID {
		return fmt.Errorf("disciplines are only available to Warriors")
	}

	talent, found := findTalentDefinition(talentID)
	if !found {
		return fmt.Errorf("talent is not available")
	}
	if session.Level < talent.MinLevel {
		return fmt.Errorf("level is too low for that talent")
	}

	talents := platform.NormalizeTalentRanks(session.Talents)
	if talents[talent.ID] >= talent.MaxRank {
		return fmt.Errorf("talent is already at maximum rank")
	}
	if talentPointsSpent(talents) >= talentPointsGranted(session.Level) {
		return fmt.Errorf("no talent points available")
	}

	talents[talent.ID]++
	character, err := s.store.UpdateCharacterTalents(session.CharacterID, talents)
	if err != nil {
		return err
	}
	session.Talents = platform.NormalizeTalentRanks(character.Talents)
	s.applyDerivedStatsLocked(session)

	observability.LogEvent("world-service", "world.talent_selected", map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"talentId":          talent.ID,
		"rank":              session.Talents[talent.ID],
		"selectedAt":        time.Now().Unix(),
	})
	return nil
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

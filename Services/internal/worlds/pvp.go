package worlds

import (
	"fmt"
	"os"
	"strings"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
)

const (
	duelStatePending   = "pending"
	duelStateCountdown = "countdown"
	duelStateActive    = "active"
	duelStateCompleted = "completed"
	duelStateCanceled  = "canceled"
	duelStateExpired   = "expired"

	duelReasonDeclined       = "declined"
	duelReasonCanceled       = "canceled"
	duelReasonDefeat         = "defeat"
	duelReasonDisconnect     = "disconnect"
	duelReasonExpired        = "expired"
	duelReasonOutOfBounds    = "out_of_bounds"
	duelReasonSurrender      = "surrender"
	duelReasonInvalidContext = "invalid_context"

	duelRequestExpireMs = int64(30_000)
	duelCountdownMs     = int64(3_000)
	duelActiveTimeoutMs = int64(300_000)
	duelOutOfBoundsMs   = int64(5_000)
	duelMaxDistance     = 42.0
	duelDefeatHealth    = 1.0
	pvpDamageScale      = 0.75
)

type safeZoneFlags struct {
	NoDuel          bool
	NoHostileAction bool
	Sanctuary       bool
}

type safeZoneDefinition struct {
	AreaID      string
	ZoneID      string
	DisplayName string
	CenterX     float64
	CenterY     float64
	Radius      float64
	Flags       safeZoneFlags
}

type duelState struct {
	DuelID                string
	ChallengerCharacterID string
	TargetCharacterID     string
	State                 string
	CreatedAtMs           int64
	AcceptedAtMs          int64
	StartedAtMs           int64
	EndedAtMs             int64
	WinnerCharacterID     string
	ReasonEnded           string
	CenterX               float64
	CenterY               float64
	MaxDistance           float64
	ZoneID                string
	ExpiresAtMs           int64
	CountdownEndsAtMs     int64
	TimeoutAtMs           int64
	OutOfBoundsSinceMs    int64
}

type duelResultState struct {
	DuelID              string `json:"duelId"`
	Result              string `json:"result"`
	Reason              string `json:"reason"`
	OpponentCharacterID string `json:"opponentCharacterId"`
	OpponentDisplayName string `json:"opponentDisplayName"`
	WinnerCharacterID   string `json:"winnerCharacterId,omitempty"`
	EndedAt             int64  `json:"endedAt"`
}

var pvpSafeZoneDefinitions = []safeZoneDefinition{
	{
		AreaID:      "stonewake_hearthwatch_yard",
		ZoneID:      defaultZoneID,
		DisplayName: "Hearthwatch Yard",
		CenterX:     13.0,
		CenterY:     10.0,
		Radius:      28.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
	{
		AreaID:      "stonewake_training_services",
		ZoneID:      defaultZoneID,
		DisplayName: "Training Ring Services",
		CenterX:     40.0,
		CenterY:     22.0,
		Radius:      18.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
	{
		AreaID:      "stonewake_field_aid_post",
		ZoneID:      defaultZoneID,
		DisplayName: "Field Aid Post",
		CenterX:     21.0,
		CenterY:     25.0,
		Radius:      12.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
	{
		AreaID:      "stonewake_westward_gate",
		ZoneID:      defaultZoneID,
		DisplayName: "Westward Gate",
		CenterX:     438.0,
		CenterY:     246.0,
		Radius:      24.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
	{
		AreaID:      "brindlebrook_highmere_crossing",
		ZoneID:      secondZoneID,
		DisplayName: "Highmere Crossing",
		CenterX:     150.0,
		CenterY:     160.0,
		Radius:      36.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
	{
		AreaID:      "brindlebrook_northspur_checkpoint",
		ZoneID:      secondZoneID,
		DisplayName: "Northspur Checkpoint",
		CenterX:     620.0,
		CenterY:     305.0,
		Radius:      30.0,
		Flags:       safeZoneFlags{NoDuel: true, NoHostileAction: true, Sanctuary: true},
	},
}

func pvpDuelsEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("AMANDACORE_PVP_DUELS_ENABLED")))
	return value != "0" && value != "false" && value != "disabled" && value != "off"
}

func (s *worldServer) requestDuelLocked(challenger *worldSessionState, target *worldSessionState) (*duelState, error) {
	if !pvpDuelsEnabled() {
		return nil, fmt.Errorf("duels are disabled")
	}
	if challenger == nil || target == nil {
		return nil, fmt.Errorf("target unavailable")
	}
	if challenger.CharacterID == target.CharacterID {
		return nil, fmt.Errorf("cannot duel yourself")
	}
	if !challenger.Connected || !target.Connected {
		return nil, fmt.Errorf("target unavailable")
	}
	if !challenger.Alive || !target.Alive {
		return nil, fmt.Errorf("dead players cannot duel")
	}
	if challenger.ZoneID != target.ZoneID || challenger.InstanceID != target.InstanceID {
		return nil, fmt.Errorf("target unavailable")
	}
	if distance2D(challenger.X, challenger.Y, target.X, target.Y) > playerTargetRange {
		return nil, fmt.Errorf("target is out of range")
	}
	if s.findDuelForCharacterLocked(challenger.CharacterID) != nil {
		return nil, fmt.Errorf("challenger is already dueling")
	}
	if s.findDuelForCharacterLocked(target.CharacterID) != nil {
		return nil, fmt.Errorf("target already dueling")
	}
	if s.sessionInNoDuelAreaLocked(challenger) || s.sessionInNoDuelAreaLocked(target) {
		return nil, fmt.Errorf("cannot duel here")
	}

	nowMs := nowMillis()
	s.duelCounter++
	duel := &duelState{
		DuelID:                fmt.Sprintf("duel_%06d", s.duelCounter),
		ChallengerCharacterID: challenger.CharacterID,
		TargetCharacterID:     target.CharacterID,
		State:                 duelStatePending,
		CreatedAtMs:           nowMs,
		CenterX:               (challenger.X + target.X) / 2.0,
		CenterY:               (challenger.Y + target.Y) / 2.0,
		MaxDistance:           duelMaxDistance,
		ZoneID:                challenger.ZoneID,
		ExpiresAtMs:           nowMs + duelRequestExpireMs,
	}
	s.duels[duel.DuelID] = duel
	s.duelByCharacter[challenger.CharacterID] = duel.DuelID
	s.duelByCharacter[target.CharacterID] = duel.DuelID

	s.sendSystemMessageLocked(fmt.Sprintf("%s has challenged %s to a duel.", challenger.DisplayName, target.DisplayName), recipientSet(challenger.CharacterID, target.CharacterID))
	observability.LogEvent("world-service", "world.duel_requested", map[string]any{
		"duelId":       duel.DuelID,
		"challengerId": challenger.CharacterID,
		"targetId":     target.CharacterID,
		"zoneId":       duel.ZoneID,
		"x":            duel.CenterX,
		"y":            duel.CenterY,
	})
	return duel, nil
}

func (s *worldServer) acceptDuelLocked(session *worldSessionState, duelID string) (*duelState, error) {
	duel := s.findActionableDuelLocked(session, duelID)
	if duel == nil || duel.TargetCharacterID != session.CharacterID || duel.State != duelStatePending {
		return nil, fmt.Errorf("duel request is not available")
	}

	challenger := s.findConnectedSessionByCharacterLocked(duel.ChallengerCharacterID)
	target := s.findConnectedSessionByCharacterLocked(duel.TargetCharacterID)
	if err := s.validateDuelStartLocked(challenger, target); err != nil {
		_ = s.endDuelLocked(duel, duelReasonInvalidContext, "")
		return nil, err
	}

	nowMs := nowMillis()
	duel.State = duelStateCountdown
	duel.AcceptedAtMs = nowMs
	duel.CountdownEndsAtMs = nowMs + duelCountdownMs
	duel.CenterX = (challenger.X + target.X) / 2.0
	duel.CenterY = (challenger.Y + target.Y) / 2.0
	duel.ZoneID = challenger.ZoneID
	s.sendSystemMessageLocked("Duel accepted. Prepare yourself.", recipientSet(duel.ChallengerCharacterID, duel.TargetCharacterID))
	observability.LogEvent("world-service", "world.duel_accepted", map[string]any{
		"duelId":       duel.DuelID,
		"challengerId": duel.ChallengerCharacterID,
		"targetId":     duel.TargetCharacterID,
		"startsAt":     duel.CountdownEndsAtMs,
	})
	return duel, nil
}

func (s *worldServer) declineDuelLocked(session *worldSessionState, duelID string) error {
	duel := s.findActionableDuelLocked(session, duelID)
	if duel == nil || duel.TargetCharacterID != session.CharacterID || duel.State != duelStatePending {
		return fmt.Errorf("duel request is not available")
	}
	return s.endDuelLocked(duel, duelReasonDeclined, "")
}

func (s *worldServer) cancelDuelLocked(session *worldSessionState, duelID string) error {
	duel := s.findActionableDuelLocked(session, duelID)
	if duel == nil {
		return fmt.Errorf("duel is not available")
	}
	if duel.State == duelStateActive {
		return s.surrenderDuelLocked(session, duel.DuelID)
	}
	if duel.ChallengerCharacterID != session.CharacterID && duel.TargetCharacterID != session.CharacterID {
		return fmt.Errorf("duel is not available")
	}
	return s.endDuelLocked(duel, duelReasonCanceled, "")
}

func (s *worldServer) surrenderDuelLocked(session *worldSessionState, duelID string) error {
	duel := s.findActionableDuelLocked(session, duelID)
	if duel == nil {
		return fmt.Errorf("duel is not available")
	}
	if duel.ChallengerCharacterID != session.CharacterID && duel.TargetCharacterID != session.CharacterID {
		return fmt.Errorf("duel is not available")
	}
	if duel.State != duelStateActive {
		return s.endDuelLocked(duel, duelReasonCanceled, "")
	}
	winnerID := duel.ChallengerCharacterID
	if session.CharacterID == duel.ChallengerCharacterID {
		winnerID = duel.TargetCharacterID
	}
	return s.endDuelLocked(duel, duelReasonSurrender, winnerID)
}

func (s *worldServer) validateDuelStartLocked(challenger *worldSessionState, target *worldSessionState) error {
	if challenger == nil || target == nil || !challenger.Connected || !target.Connected {
		return fmt.Errorf("target unavailable")
	}
	if !challenger.Alive || !target.Alive {
		return fmt.Errorf("dead players cannot duel")
	}
	if challenger.ZoneID != target.ZoneID || challenger.InstanceID != target.InstanceID {
		return fmt.Errorf("target unavailable")
	}
	if s.sessionInNoDuelAreaLocked(challenger) || s.sessionInNoDuelAreaLocked(target) {
		return fmt.Errorf("cannot duel here")
	}
	return nil
}

func (s *worldServer) advanceDuelsLocked(now time.Time) error {
	nowMs := now.UnixMilli()
	for _, duel := range s.duels {
		switch duel.State {
		case duelStatePending:
			if nowMs >= duel.ExpiresAtMs {
				if err := s.endDuelLocked(duel, duelReasonExpired, ""); err != nil {
					return err
				}
			}
		case duelStateCountdown:
			challenger := s.findConnectedSessionByCharacterLocked(duel.ChallengerCharacterID)
			target := s.findConnectedSessionByCharacterLocked(duel.TargetCharacterID)
			if err := s.validateDuelStartLocked(challenger, target); err != nil {
				if endErr := s.endDuelLocked(duel, duelReasonInvalidContext, ""); endErr != nil {
					return endErr
				}
				continue
			}
			if nowMs >= duel.CountdownEndsAtMs {
				duel.State = duelStateActive
				duel.StartedAtMs = nowMs
				duel.TimeoutAtMs = nowMs + duelActiveTimeoutMs
				s.sendSystemMessageLocked("Duel started.", recipientSet(duel.ChallengerCharacterID, duel.TargetCharacterID))
				observability.LogEvent("world-service", "world.duel_started", map[string]any{
					"duelId":       duel.DuelID,
					"challengerId": duel.ChallengerCharacterID,
					"targetId":     duel.TargetCharacterID,
				})
			}
		case duelStateActive:
			if err := s.advanceActiveDuelLocked(duel, nowMs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *worldServer) advanceActiveDuelLocked(duel *duelState, nowMs int64) error {
	challenger := s.findConnectedSessionByCharacterLocked(duel.ChallengerCharacterID)
	target := s.findConnectedSessionByCharacterLocked(duel.TargetCharacterID)
	if challenger == nil || target == nil {
		return s.endDuelLocked(duel, duelReasonDisconnect, "")
	}
	if !challenger.Alive || !target.Alive || challenger.ZoneID != target.ZoneID || challenger.InstanceID != target.InstanceID {
		return s.endDuelLocked(duel, duelReasonInvalidContext, "")
	}
	if s.sessionInNoHostileActionAreaLocked(challenger) || s.sessionInNoHostileActionAreaLocked(target) {
		return s.endDuelLocked(duel, duelReasonInvalidContext, "")
	}
	if duel.TimeoutAtMs > 0 && nowMs >= duel.TimeoutAtMs {
		return s.endDuelLocked(duel, duelReasonExpired, "")
	}

	challengerOut := distance2D(challenger.X, challenger.Y, duel.CenterX, duel.CenterY) > duel.MaxDistance
	targetOut := distance2D(target.X, target.Y, duel.CenterX, duel.CenterY) > duel.MaxDistance
	if challengerOut || targetOut {
		if duel.OutOfBoundsSinceMs == 0 {
			duel.OutOfBoundsSinceMs = nowMs
			s.sendSystemMessageLocked("Duel boundary warning: return to the duel area.", recipientSet(duel.ChallengerCharacterID, duel.TargetCharacterID))
			return nil
		}
		if nowMs-duel.OutOfBoundsSinceMs >= duelOutOfBoundsMs {
			return s.endDuelLocked(duel, duelReasonOutOfBounds, "")
		}
		return nil
	}
	duel.OutOfBoundsSinceMs = 0
	return nil
}

func (s *worldServer) endDuelLocked(duel *duelState, reason string, winnerCharacterID string) error {
	if duel == nil {
		return nil
	}
	if isTerminalDuelState(duel.State) {
		return nil
	}

	nowMs := nowMillis()
	duel.EndedAtMs = nowMs
	duel.ReasonEnded = reason
	duel.WinnerCharacterID = winnerCharacterID
	if reason == duelReasonExpired {
		duel.State = duelStateExpired
	} else if winnerCharacterID != "" {
		duel.State = duelStateCompleted
	} else {
		duel.State = duelStateCanceled
	}

	challenger := s.findSessionByCharacterLocked(duel.ChallengerCharacterID)
	target := s.findSessionByCharacterLocked(duel.TargetCharacterID)
	if challenger != nil {
		s.resetSessionCombatStateLocked(challenger, "duel_ended")
	}
	if target != nil {
		s.resetSessionCombatStateLocked(target, "duel_ended")
	}
	if winnerCharacterID != "" {
		loser := s.duelOpponentID(duel, winnerCharacterID)
		if loserSession := s.findSessionByCharacterLocked(loser); loserSession != nil && loserSession.Health < duelDefeatHealth {
			loserSession.Health = duelDefeatHealth
			loserSession.Alive = true
		}
		if err := s.recordDuelOutcomeLocked(winnerCharacterID, loser); err != nil {
			return err
		}
	}

	s.setDuelResultLocked(duel)
	delete(s.duelByCharacter, duel.ChallengerCharacterID)
	delete(s.duelByCharacter, duel.TargetCharacterID)
	delete(s.duels, duel.DuelID)

	message := "Duel ended."
	switch reason {
	case duelReasonDeclined:
		message = "Duel declined."
	case duelReasonCanceled:
		message = "Duel canceled."
	case duelReasonDisconnect:
		message = "Duel canceled because a participant disconnected."
	case duelReasonExpired:
		message = "Duel expired."
	case duelReasonOutOfBounds:
		message = "Duel canceled because an opponent moved too far away."
	case duelReasonSurrender:
		message = "Duel ended by surrender."
	case duelReasonDefeat:
		message = "Duel ended."
	}
	s.sendSystemMessageLocked(message, recipientSet(duel.ChallengerCharacterID, duel.TargetCharacterID))
	observability.LogEvent("world-service", "world.duel_ended", map[string]any{
		"duelId":   duel.DuelID,
		"state":    duel.State,
		"reason":   reason,
		"winnerId": winnerCharacterID,
	})
	return nil
}

func (s *worldServer) recordDuelOutcomeLocked(winnerCharacterID string, loserCharacterID string) error {
	if winnerCharacterID == "" || loserCharacterID == "" {
		return nil
	}

	now := time.Now().Unix()
	winnerStats := s.pvpStatsForCharacterLocked(winnerCharacterID)
	winnerStats.DuelsWon++
	winnerStats.HonorPoints++
	winnerStats.LastDuelWonAt = now
	winnerStats.UpdatedAt = now
	if err := s.savePvPStatsLocked(winnerCharacterID, winnerStats); err != nil {
		return err
	}

	loserStats := s.pvpStatsForCharacterLocked(loserCharacterID)
	loserStats.DuelsLost++
	loserStats.UpdatedAt = now
	return s.savePvPStatsLocked(loserCharacterID, loserStats)
}

func (s *worldServer) pvpStatsForCharacterLocked(characterID string) platform.CharacterPvPStats {
	if session := s.findSessionByCharacterLocked(characterID); session != nil {
		return platform.NormalizeCharacterPvPStats(characterID, session.PvPStats)
	}
	if s.store != nil {
		if character, err := s.store.GetCharacterByID(characterID); err == nil {
			return platform.NormalizeCharacterPvPStats(characterID, character.PvPStats)
		}
	}
	return platform.NormalizeCharacterPvPStats(characterID, platform.CharacterPvPStats{})
}

func (s *worldServer) savePvPStatsLocked(characterID string, stats platform.CharacterPvPStats) error {
	stats = platform.NormalizeCharacterPvPStats(characterID, stats)
	if s.store != nil {
		persistStartedAt := time.Now()
		character, err := s.store.UpdateCharacterPvPStats(characterID, stats)
		s.recordPersistenceDuration("character_pvp_stats", persistStartedAt, err)
		if err != nil {
			return err
		}
		stats = platform.NormalizeCharacterPvPStats(characterID, character.PvPStats)
	}
	if session := s.findSessionByCharacterLocked(characterID); session != nil {
		session.PvPStats = stats
	}
	return nil
}

func (s *worldServer) applyDamageToPlayerLocked(attacker *worldSessionState, target *worldSessionState, amount float64, source string) error {
	if err := s.validatePvPDamageLocked(attacker, target); err != nil {
		return err
	}
	if amount <= 0 {
		return nil
	}

	damage := maxFloat(1.0, amount*pvpDamageScale)
	target.Health -= damage
	if target.Health <= duelDefeatHealth {
		target.Health = duelDefeatHealth
		target.Alive = true
		observability.LogEvent("world-service", "world.pvp_damage_applied", map[string]any{
			"sourceCharacterId": attacker.CharacterID,
			"targetCharacterId": target.CharacterID,
			"source":            source,
			"amount":            damage,
			"remainingHealth":   target.Health,
			"defeat":            true,
		})
		duel := s.duelForPairLocked(attacker.CharacterID, target.CharacterID)
		return s.endDuelLocked(duel, duelReasonDefeat, attacker.CharacterID)
	}

	observability.LogEvent("world-service", "world.pvp_damage_applied", map[string]any{
		"sourceCharacterId": attacker.CharacterID,
		"targetCharacterId": target.CharacterID,
		"source":            source,
		"amount":            damage,
		"remainingHealth":   target.Health,
		"defeat":            false,
	})
	return nil
}

func (s *worldServer) validatePvPDamageLocked(attacker *worldSessionState, target *worldSessionState) error {
	if attacker == nil || target == nil || attacker.CharacterID == target.CharacterID {
		return fmt.Errorf("target is invalid")
	}
	if !attacker.Connected || !target.Connected || !attacker.Alive || !target.Alive {
		return fmt.Errorf("target is invalid")
	}
	if attacker.ZoneID != target.ZoneID || attacker.InstanceID != target.InstanceID {
		return fmt.Errorf("target unavailable")
	}
	duel := s.duelForPairLocked(attacker.CharacterID, target.CharacterID)
	if duel == nil || duel.State != duelStateActive {
		return fmt.Errorf("target is not a valid PvP opponent")
	}
	if s.sessionInNoHostileActionAreaLocked(attacker) || s.sessionInNoHostileActionAreaLocked(target) {
		return fmt.Errorf("cannot attack here")
	}
	return nil
}

func (s *worldServer) findPlayerTargetForSessionLocked(session *worldSessionState, targetID string) *worldSessionState {
	if session == nil || targetID == "" || targetID == session.CharacterID {
		return nil
	}
	target := s.findConnectedSessionByCharacterLocked(targetID)
	if target == nil || target.ZoneID != session.ZoneID || target.InstanceID != session.InstanceID {
		return nil
	}
	return target
}

func (s *worldServer) resolveDuelRequestTargetLocked(session *worldSessionState, request duelRequest) (*worldSessionState, error) {
	if request.TargetCharacterID != "" {
		target := s.findConnectedSessionByCharacterLocked(request.TargetCharacterID)
		if target == nil {
			return nil, fmt.Errorf("target unavailable")
		}
		return target, nil
	}
	if strings.TrimSpace(request.TargetName) != "" {
		if s.store == nil {
			return nil, fmt.Errorf("target unavailable")
		}
		character, err := s.store.GetCharacterByName(session.RealmID, request.TargetName)
		if err != nil {
			return nil, fmt.Errorf("target unavailable")
		}
		target := s.findConnectedSessionByCharacterLocked(character.ID)
		if target == nil {
			return nil, fmt.Errorf("target unavailable")
		}
		return target, nil
	}
	if session.CurrentTargetID != "" {
		target := s.findConnectedSessionByCharacterLocked(session.CurrentTargetID)
		if target != nil {
			return target, nil
		}
	}
	return nil, fmt.Errorf("target unavailable")
}

func (s *worldServer) findDuelForCharacterLocked(characterID string) *duelState {
	duelID := s.duelByCharacter[characterID]
	if duelID == "" {
		return nil
	}
	duel := s.duels[duelID]
	if duel == nil || isTerminalDuelState(duel.State) {
		delete(s.duelByCharacter, characterID)
		return nil
	}
	return duel
}

func (s *worldServer) findActionableDuelLocked(session *worldSessionState, duelID string) *duelState {
	if session == nil {
		return nil
	}
	if duelID != "" {
		duel := s.duels[duelID]
		if duel == nil || isTerminalDuelState(duel.State) {
			return nil
		}
		if duel.ChallengerCharacterID != session.CharacterID && duel.TargetCharacterID != session.CharacterID {
			return nil
		}
		return duel
	}
	return s.findDuelForCharacterLocked(session.CharacterID)
}

func (s *worldServer) duelForPairLocked(firstCharacterID string, secondCharacterID string) *duelState {
	first := s.findDuelForCharacterLocked(firstCharacterID)
	second := s.findDuelForCharacterLocked(secondCharacterID)
	if first == nil || second == nil || first.DuelID != second.DuelID {
		return nil
	}
	return first
}

func (s *worldServer) cancelDuelForCharacterLocked(characterID string, reason string) {
	duel := s.findDuelForCharacterLocked(characterID)
	if duel == nil {
		return
	}
	if err := s.endDuelLocked(duel, reason, ""); err != nil {
		observability.LogEvent("world-service", "world.duel_cancel_failed", map[string]any{
			"duelId":      duel.DuelID,
			"characterId": characterID,
			"reason":      reason,
			"error":       err.Error(),
		})
	}
}

func (s *worldServer) findSessionByCharacterLocked(characterID string) *worldSessionState {
	if characterID == "" {
		return nil
	}
	if token := s.sessionTokenByChar[characterID]; token != "" {
		if session := s.sessionsByToken[token]; session != nil {
			return session
		}
	}
	for _, session := range s.sessionsByToken {
		if session.CharacterID == characterID {
			return session
		}
	}
	return nil
}

func (s *worldServer) duelOpponentID(duel *duelState, characterID string) string {
	if duel == nil {
		return ""
	}
	if duel.ChallengerCharacterID == characterID {
		return duel.TargetCharacterID
	}
	if duel.TargetCharacterID == characterID {
		return duel.ChallengerCharacterID
	}
	return ""
}

func (s *worldServer) setDuelResultLocked(duel *duelState) {
	if duel == nil {
		return
	}
	for _, characterID := range []string{duel.ChallengerCharacterID, duel.TargetCharacterID} {
		session := s.findSessionByCharacterLocked(characterID)
		if session == nil {
			continue
		}
		opponentID := s.duelOpponentID(duel, characterID)
		opponentName := ""
		if opponent := s.findSessionByCharacterLocked(opponentID); opponent != nil {
			opponentName = opponent.DisplayName
		}
		result := "canceled"
		if duel.WinnerCharacterID != "" {
			if duel.WinnerCharacterID == characterID {
				result = "won"
			} else {
				result = "lost"
			}
		} else if duel.State == duelStateExpired {
			result = "expired"
		}
		session.LastDuelResult = &duelResultState{
			DuelID:              duel.DuelID,
			Result:              result,
			Reason:              duel.ReasonEnded,
			OpponentCharacterID: opponentID,
			OpponentDisplayName: opponentName,
			WinnerCharacterID:   duel.WinnerCharacterID,
			EndedAt:             duel.EndedAtMs,
		}
	}
}

func isTerminalDuelState(state string) bool {
	return state == duelStateCompleted || state == duelStateCanceled || state == duelStateExpired
}

func (s *worldServer) sessionInNoDuelAreaLocked(session *worldSessionState) bool {
	flags := s.safeZoneFlagsForSessionLocked(session)
	return flags.NoDuel
}

func (s *worldServer) sessionInNoHostileActionAreaLocked(session *worldSessionState) bool {
	flags := s.safeZoneFlagsForSessionLocked(session)
	return flags.NoHostileAction
}

func (s *worldServer) safeZoneFlagsForSessionLocked(session *worldSessionState) safeZoneFlags {
	flags := safeZoneFlags{}
	for _, area := range s.safeZonesForSessionLocked(session) {
		flags.NoDuel = flags.NoDuel || area.Flags.NoDuel
		flags.NoHostileAction = flags.NoHostileAction || area.Flags.NoHostileAction
		flags.Sanctuary = flags.Sanctuary || area.Flags.Sanctuary
	}
	return flags
}

func (s *worldServer) safeZonesForSessionLocked(session *worldSessionState) []safeZoneDefinition {
	if session == nil {
		return nil
	}
	areas := make([]safeZoneDefinition, 0)
	for _, area := range pvpSafeZoneDefinitions {
		if area.ZoneID != session.ZoneID {
			continue
		}
		if area.Radius <= 0 {
			continue
		}
		if distance2D(session.X, session.Y, area.CenterX, area.CenterY) <= area.Radius {
			areas = append(areas, area)
		}
	}
	return areas
}

func (s *worldServer) buildPvPResponseLocked(session *worldSessionState) map[string]any {
	response := map[string]any{
		"duelsEnabled": pvpDuelsEnabled(),
		"stats":        buildPvPStatsResponse(platform.NormalizeCharacterPvPStats(session.CharacterID, session.PvPStats)),
		"safeZone":     s.buildSafeZoneResponseLocked(session),
	}
	if session.LastDuelResult != nil {
		response["lastResult"] = session.LastDuelResult
	}
	if duel := s.findDuelForCharacterLocked(session.CharacterID); duel != nil {
		opponentID := s.duelOpponentID(duel, session.CharacterID)
		opponentName := ""
		if opponent := s.findSessionByCharacterLocked(opponentID); opponent != nil {
			opponentName = opponent.DisplayName
		}
		response["duelId"] = duel.DuelID
		response["duelState"] = duel.State
		response["opponentCharacterId"] = opponentID
		response["opponentDisplayName"] = opponentName
		response["incomingDuel"] = duel.State == duelStatePending && duel.TargetCharacterID == session.CharacterID
		response["outgoingDuel"] = duel.State == duelStatePending && duel.ChallengerCharacterID == session.CharacterID
		response["countdownEndsAt"] = duel.CountdownEndsAtMs
		response["startedAt"] = duel.StartedAtMs
		response["timeoutAt"] = duel.TimeoutAtMs
		response["boundary"] = map[string]any{
			"centerX":     duel.CenterX,
			"centerY":     duel.CenterY,
			"maxDistance": duel.MaxDistance,
		}
	}
	return response
}

func buildPvPStatsResponse(stats platform.CharacterPvPStats) map[string]any {
	return map[string]any{
		"characterId":   stats.CharacterID,
		"duelsWon":      stats.DuelsWon,
		"duelsLost":     stats.DuelsLost,
		"honorPoints":   stats.HonorPoints,
		"lastDuelWonAt": stats.LastDuelWonAt,
		"updatedAt":     stats.UpdatedAt,
	}
}

func (s *worldServer) buildSafeZoneResponseLocked(session *worldSessionState) map[string]any {
	flags := s.safeZoneFlagsForSessionLocked(session)
	areas := s.safeZonesForSessionLocked(session)
	areaResponses := make([]map[string]any, 0, len(areas))
	for _, area := range areas {
		areaResponses = append(areaResponses, map[string]any{
			"areaId":      area.AreaID,
			"zoneId":      area.ZoneID,
			"displayName": area.DisplayName,
			"centerX":     area.CenterX,
			"centerY":     area.CenterY,
			"radius":      area.Radius,
			"flags": map[string]bool{
				"noDuel":          area.Flags.NoDuel,
				"noHostileAction": area.Flags.NoHostileAction,
				"sanctuary":       area.Flags.Sanctuary,
			},
		})
	}
	return map[string]any{
		"noDuel":          flags.NoDuel,
		"noHostileAction": flags.NoHostileAction,
		"sanctuary":       flags.Sanctuary,
		"areas":           areaResponses,
	}
}

func (s *worldServer) pvpStateForVisiblePlayerLocked(viewer *worldSessionState, candidate *worldSessionState) (string, bool) {
	if viewer == nil || candidate == nil {
		return "", false
	}
	duel := s.duelForPairLocked(viewer.CharacterID, candidate.CharacterID)
	if duel == nil {
		if s.findDuelForCharacterLocked(candidate.CharacterID) != nil {
			return "dueling", false
		}
		return "", false
	}
	switch duel.State {
	case duelStatePending:
		return "duel_pending", true
	case duelStateCountdown:
		return "duel_countdown", true
	case duelStateActive:
		return "duel_opponent", true
	default:
		return "", false
	}
}

package worlds

import (
	"fmt"
	"time"

	"amandacore/services/internal/observability"
)

func (s *worldServer) stopAutoAttackLocked(session *worldSessionState, reason string) bool {
	if !session.AutoAttackActive {
		return false
	}

	session.AutoAttackActive = false
	observability.LogEvent("world-service", "world.auto_attack_stopped", map[string]any{
		"worldSessionToken": session.Token,
		"characterId":       session.CharacterID,
		"reason":            reason,
	})
	return true
}

func (s *worldServer) cancelCastLocked(session *worldSessionState) bool {
	if session.CastingAbilityID == "" && session.CastingTargetID == "" && session.CastEndsAtMs == 0 {
		return false
	}

	session.CastingAbilityID = ""
	session.CastingTargetID = ""
	session.CastEndsAtMs = 0
	return true
}

func (s *worldServer) clearTargetLocked(session *worldSessionState, reason string) {
	s.stopAutoAttackLocked(session, reason)
	s.cancelCastLocked(session)

	if session.CurrentTargetID == "" {
		return
	}

	clearedTargetID := session.CurrentTargetID
	session.CurrentTargetID = ""
	observability.LogEvent("world-service", "world.target_cleared", map[string]any{
		"worldSessionToken": session.Token,
		"accountId":         session.AccountID,
		"characterId":       session.CharacterID,
		"targetId":          clearedTargetID,
		"reason":            reason,
	})
}

func (s *worldServer) resetSessionCombatStateLocked(session *worldSessionState, reason string) {
	s.clearTargetLocked(session, reason)
	s.stopAutoAttackLocked(session, reason)
	s.cancelCastLocked(session)
}

func (s *worldServer) reviveSessionLocked(session *worldSessionState) {
	session.Alive = true
	session.Health = session.MaxHealth
	session.Resource = session.MaxResource
}

func (s *worldServer) findMobByIDLocked(mobID string) *mobState {
	if mobID == "" || len(s.mobs) == 0 {
		return nil
	}

	return s.mobs[mobID]
}

func (s *worldServer) hostileMobsLocked() []*mobState {
	if len(s.mobOrder) == 0 {
		return nil
	}

	mobs := make([]*mobState, 0, len(s.mobOrder))
	for _, mobID := range s.mobOrder {
		if mob := s.mobs[mobID]; mob != nil {
			mobs = append(mobs, mob)
		}
	}

	return mobs
}

func (s *worldServer) clearMobAggroForCharacterLocked(characterID string) {
	for _, mob := range s.hostileMobsLocked() {
		if mob.CurrentTargetCharacter != characterID {
			continue
		}

		mob.CurrentTargetCharacter = ""
		if mob.Alive && mob.AIState != "leash_return" {
			mob.AIState = "idle"
		}
	}
}

func (s *worldServer) setAutoAttackLocked(session *worldSessionState, enabled bool) error {
	if !session.Alive {
		return fmt.Errorf("player is dead")
	}

	if enabled {
		if session.CurrentTargetID == "" {
			return fmt.Errorf("no target")
		}

		targetMob := s.findMobByIDLocked(session.CurrentTargetID)
		if targetMob == nil || !targetMob.Alive || !targetMob.Targetable {
			return fmt.Errorf("target is invalid")
		}
		if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > playerAutoAttackRange {
			return fmt.Errorf("target is out of range")
		}

		session.AutoAttackActive = true
		session.LastAutoAttackAtMs = nowMillis() - playerAutoAttackCadenceMs
		observability.LogEvent("world-service", "world.auto_attack_started", map[string]any{
			"worldSessionToken": session.Token,
			"characterId":       session.CharacterID,
			"targetId":          session.CurrentTargetID,
		})
		return nil
	}

	s.stopAutoAttackLocked(session, "manual")
	return nil
}

func (s *worldServer) activateAbilityLocked(session *worldSessionState, abilityID string) error {
	if !session.Alive {
		return fmt.Errorf("player is dead")
	}
	if session.CastingAbilityID != "" && session.CastEndsAtMs > nowMillis() {
		return fmt.Errorf("already casting")
	}
	if session.GlobalCooldownEnds > nowMillis() {
		return fmt.Errorf("on cooldown")
	}
	if !s.sessionKnowsAbilityLocked(session, abilityID) {
		return fmt.Errorf("ability is not learned")
	}

	ability, found := findAbilityDefinition(abilityID)
	if !found {
		return fmt.Errorf("ability is not available")
	}

	targetMob := s.findMobByIDLocked(session.CurrentTargetID)
	return s.applyAbilityEffectLocked(session, targetMob, ability)
}

func (s *worldServer) advanceWorldLocked(now time.Time) error {
	s.ensureMobsLocked()
	if s.lastUpdatedAt.IsZero() {
		s.lastUpdatedAt = now
		return nil
	}

	deltaSeconds := clampSeconds(now.Sub(s.lastUpdatedAt).Seconds())
	s.lastUpdatedAt = now
	if deltaSeconds <= 0 {
		return nil
	}

	nowMs := now.UnixMilli()
	for _, session := range s.sessionsByToken {
		if session.Resource < session.MaxResource {
			session.Resource = minFloat(session.MaxResource, session.Resource+(playerResourceRegenPerSec*deltaSeconds))
		}

		if session.CastingAbilityID != "" && session.CastEndsAtMs > 0 && nowMs >= session.CastEndsAtMs {
			s.cancelCastLocked(session)
		}

		if !session.AutoAttackActive || !session.Alive {
			continue
		}

		targetMob := s.findMobByIDLocked(session.CurrentTargetID)
		if targetMob == nil || session.CurrentTargetID == "" {
			s.stopAutoAttackLocked(session, "target_invalid")
			continue
		}
		if !targetMob.Alive {
			s.clearTargetLocked(session, "target_dead")
			continue
		}
		if !targetMob.Targetable {
			s.clearTargetLocked(session, "target_invalid")
			continue
		}
		if distance2D(session.X, session.Y, targetMob.X, targetMob.Y) > playerAutoAttackRange {
			s.stopAutoAttackLocked(session, "out_of_range")
			continue
		}
		if nowMs-session.LastAutoAttackAtMs >= playerAutoAttackCadenceMs {
			session.LastAutoAttackAtMs = nowMs
			if err := s.applyDamageToMobLocked(session, targetMob, playerAutoAttackDamage, "auto_attack"); err != nil {
				return err
			}
		}
	}

	for _, mob := range s.hostileMobsLocked() {
		s.advanceMobLocked(mob, deltaSeconds, nowMs)
	}
	return nil
}

func (s *worldServer) advanceMobLocked(mob *mobState, deltaSeconds float64, nowMs int64) {
	if mob == nil {
		return
	}

	if !mob.Alive {
		if mob.RespawnAtMs != 0 && nowMs >= mob.RespawnAtMs {
			mob.Alive = true
			mob.Targetable = true
			mob.AIState = "idle"
			mob.Health = mob.MaxHealth
			mob.X = mob.SpawnX
			mob.Y = mob.SpawnY
			mob.Z = mob.SpawnZ
			mob.CurrentTargetCharacter = ""
			mob.LastAttackAtMs = 0
			mob.RespawnAtMs = 0
			observability.LogEvent("world-service", "world.mob_respawned", map[string]any{
				"mobId":  mob.ID,
				"zoneId": mob.ZoneID,
				"x":      mob.X,
				"y":      mob.Y,
				"z":      mob.Z,
			})
		}
		return
	}

	targetSession := s.resolveMobTargetLocked(mob)
	if targetSession == nil {
		if mob.AIState == "leash_return" {
			s.moveMobTowardsLocked(mob, mob.SpawnX, mob.SpawnY, deltaSeconds)
			if distance2D(mob.X, mob.Y, mob.SpawnX, mob.SpawnY) <= 0.25 {
				mob.X = mob.SpawnX
				mob.Y = mob.SpawnY
				mob.Health = mob.MaxHealth
				mob.Targetable = true
				mob.AIState = "idle"
			}
		} else {
			mob.AIState = "idle"
		}
		return
	}

	if distance2D(targetSession.X, targetSession.Y, mob.SpawnX, mob.SpawnY) > mob.LeashRadius {
		s.enterLeashReturnLocked(mob)
		return
	}

	distanceToTarget := distance2D(mob.X, mob.Y, targetSession.X, targetSession.Y)
	if distanceToTarget > mob.AttackRange {
		mob.AIState = "chase"
		s.moveMobTowardsLocked(mob, targetSession.X, targetSession.Y, deltaSeconds)
		return
	}

	mob.AIState = "aggro"
	if nowMs-mob.LastAttackAtMs < mob.AttackCadenceMs {
		return
	}
	mob.LastAttackAtMs = nowMs

	if !targetSession.Alive {
		return
	}

	targetSession.Health = maxFloat(0.0, targetSession.Health-mob.AttackDamage)
	observability.LogEvent("world-service", "world.damage_applied", map[string]any{
		"source":            mob.ID,
		"targetCharacterId": targetSession.CharacterID,
		"amount":            mob.AttackDamage,
		"remainingHealth":   targetSession.Health,
	})

	if targetSession.Health <= 0.0 {
		targetSession.Alive = false
		s.resetSessionCombatStateLocked(targetSession, "player_dead")
		mob.CurrentTargetCharacter = ""
		mob.AIState = "idle"
	}
}

func (s *worldServer) resolveMobTargetLocked(mob *mobState) *worldSessionState {
	if mob == nil || !mob.Alive || !mob.Targetable {
		return nil
	}

	if mob.CurrentTargetCharacter != "" {
		if current := s.findConnectedSessionByCharacterLocked(mob.CurrentTargetCharacter); current != nil && current.Alive {
			return current
		}
		mob.CurrentTargetCharacter = ""
	}

	var closest *worldSessionState
	closestDistance := mob.AggroRadius
	for _, session := range s.sessionsByToken {
		if !session.Connected || !session.Alive || session.ZoneID != mob.ZoneID {
			continue
		}
		distance := distance2D(session.X, session.Y, mob.X, mob.Y)
		if distance > closestDistance {
			continue
		}
		closest = session
		closestDistance = distance
	}

	if closest != nil {
		mob.CurrentTargetCharacter = closest.CharacterID
		if mob.AIState == "idle" {
			observability.LogEvent("world-service", "world.mob_aggroed", map[string]any{
				"mobId":       mob.ID,
				"characterId": closest.CharacterID,
				"distance":    closestDistance,
			})
		}
	}

	return closest
}

func (s *worldServer) moveMobTowardsLocked(mob *mobState, targetX float64, targetY float64, deltaSeconds float64) {
	if mob == nil {
		return
	}

	deltaX := targetX - mob.X
	deltaY := targetY - mob.Y
	distance := distance2D(mob.X, mob.Y, targetX, targetY)
	if distance <= 0.001 {
		return
	}

	step := minFloat(distance, mob.MoveSpeedPerSec*deltaSeconds)
	mob.X += (deltaX / distance) * step
	mob.Y += (deltaY / distance) * step
}

func (s *worldServer) applyDamageToMobLocked(session *worldSessionState, mob *mobState, amount float64, source string) error {
	if mob == nil || !mob.Alive || !mob.Targetable {
		return nil
	}

	mob.CurrentTargetCharacter = session.CharacterID
	if mob.AIState == "idle" {
		observability.LogEvent("world-service", "world.mob_aggroed", map[string]any{
			"mobId":       mob.ID,
			"characterId": session.CharacterID,
			"reason":      source,
		})
	}

	mob.Health = maxFloat(0.0, mob.Health-amount)
	observability.LogEvent("world-service", "world.damage_applied", map[string]any{
		"sourceCharacterId": session.CharacterID,
		"targetId":          mob.ID,
		"source":            source,
		"amount":            amount,
		"remainingHealth":   mob.Health,
	})

	if mob.Health > 0.0 {
		return nil
	}

	mob.Alive = false
	mob.Targetable = false
	mob.AIState = "dead"
	mob.RespawnAtMs = nowMillis() + mob.RespawnDelayMs
	mob.CurrentTargetCharacter = ""
	s.clearMobTargetFromAllSessionsLocked(mob.ID, "target_dead")
	if err := s.applyQuestKillCreditLocked(session, mob.MobTypeID); err != nil {
		return err
	}
	observability.LogEvent("world-service", "world.mob_died", map[string]any{
		"mobId":     mob.ID,
		"killedBy":  session.CharacterID,
		"respawnAt": mob.RespawnAtMs,
	})
	return nil
}

func (s *worldServer) enterLeashReturnLocked(mob *mobState) {
	if mob == nil {
		return
	}

	mob.AIState = "leash_return"
	mob.Targetable = false
	mob.CurrentTargetCharacter = ""
	s.clearMobTargetFromAllSessionsLocked(mob.ID, "leash_reset")
}

func minFloat(left float64, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

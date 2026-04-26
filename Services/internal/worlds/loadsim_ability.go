package worlds

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"amandacore/services/internal/observability"
	"amandacore/services/internal/platform"
	"amandacore/services/internal/store"
)

const loadsimCastAbilityID = "dev_loadsim_delayed_strike"

type AbilityAuraLoadsimOptions struct {
	Clients  int
	Duration time.Duration
	CmdRate  int
}

type AbilityAuraLoadsimReport struct {
	SimulatedClients        int
	AbilityCommandsSent     int
	AbilityCommandsAccepted int
	AbilityCommandsRejected int
	EffectEvents            int
	AurasApplied            int
	AuraTicks               int
	AuraExpired             int
	CastsStarted            int
	CastsCompleted          int
	CooldownsStarted        int
	AverageTickDuration     time.Duration
	MaxTickDuration         time.Duration
	MaxQueueDepth           int
	Errors                  []string
}

func RunAbilityAuraLoadsim(opts AbilityAuraLoadsimOptions) (AbilityAuraLoadsimReport, error) {
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.CmdRate <= 0 {
		opts.CmdRate = 2
	}
	report := AbilityAuraLoadsimReport{SimulatedClients: opts.Clients}
	observability.LogEvent("loadsim", EventLoadsimAbilityStarted, map[string]any{
		"scenario": "ability-aura-basic",
		"clients":  opts.Clients,
		"duration": opts.Duration.String(),
		"cmdRate":  opts.CmdRate,
	})

	tempDir, err := os.MkdirTemp("", "amandacore-ability-loadsim-*")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(tempDir)

	fileStore, err := store.NewFileStore(filepath.Join(tempDir, "platform-state.json"), "loadsim-ability-aura-basic", "http://localhost:8085")
	if err != nil {
		return report, err
	}
	server := newWorldServer(fileStore)
	server.abilityCatalog[loadsimCastAbilityID] = normalizeAbilityDefinition(abilityDefinition{
		ID:             loadsimCastAbilityID,
		DisplayName:    "Delayed Strike",
		RequiresTarget: true,
		TargetRule:     abilityTargetRuleEnemy,
		RangeMeters:    3,
		Timing:         abilityTiming{CastMs: 25},
		Damage:         1,
	})

	tickDurations := []time.Duration{}
	recordTick := func(startedAt time.Time) {
		elapsed := time.Since(startedAt)
		tickDurations = append(tickDurations, elapsed)
		if elapsed > report.MaxTickDuration {
			report.MaxTickDuration = elapsed
		}
	}

	for clientIndex := 0; clientIndex < opts.Clients; clientIndex++ {
		startedAt := time.Now()
		accepted, rejected, err := runAbilityAuraClient(server, fileStore, clientIndex)
		report.AbilityCommandsSent += accepted + rejected
		report.AbilityCommandsAccepted += accepted
		report.AbilityCommandsRejected += rejected
		if err != nil {
			report.Errors = append(report.Errors, err.Error())
		}
		recordTick(startedAt)
	}

	var total time.Duration
	for _, tick := range tickDurations {
		total += tick
	}
	if len(tickDurations) > 0 {
		report.AverageTickDuration = total / time.Duration(len(tickDurations))
	}
	report.MaxQueueDepth = 0
	report.EffectEvents = server.countEvents(EventAbilityEffectResolved)
	report.AurasApplied = server.countEvents(EventAuraApplied)
	report.AuraTicks = server.countEvents(EventAuraTicked)
	report.AuraExpired = server.countEvents(EventAuraExpired)
	report.CastsStarted = server.countEvents(EventAbilityCastStarted)
	report.CastsCompleted = server.countEvents(EventAbilityCastCompleted)
	report.CooldownsStarted = server.countEvents(EventCooldownStarted)

	observability.LogEvent("loadsim", EventLoadsimAbilityCompleted, map[string]any{
		"scenario":                "ability-aura-basic",
		"clients":                 opts.Clients,
		"abilityCommandsAccepted": report.AbilityCommandsAccepted,
		"abilityCommandsRejected": report.AbilityCommandsRejected,
		"aurasApplied":            report.AurasApplied,
		"auraTicks":               report.AuraTicks,
		"errors":                  len(report.Errors),
	})
	return report, nil
}

func runAbilityAuraClient(server *worldServer, fileStore *store.FileStore, clientIndex int) (int, int, error) {
	account, err := fileStore.RegisterAccount(fmt.Sprintf("ability_loadsim_%03d", clientIndex+1), "loadsim-secret")
	if err != nil {
		return 0, 1, fmt.Errorf("client %d account failed: %w", clientIndex+1, err)
	}
	character, err := fileStore.CreateCharacter(
		account.ID,
		"sunset-frontier-dev",
		fmt.Sprintf("AbilitySim%03d", clientIndex+1),
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		return 0, 1, fmt.Errorf("client %d character failed: %w", clientIndex+1, err)
	}

	server.mutex.Lock()
	defer server.mutex.Unlock()

	session := &worldSessionState{
		Token:            fmt.Sprintf("loadsim_ability_%03d", clientIndex+1),
		AccountID:        account.ID,
		CharacterID:      character.ID,
		DisplayName:      character.DisplayName,
		ClassID:          character.ClassID,
		RealmID:          character.RealmID,
		ZoneID:           defaultZoneID,
		Connected:        true,
		LastSeenAt:       time.Now().Unix(),
		Health:           playerMaxHealth,
		MaxHealth:        playerMaxHealth,
		Resource:         0,
		MaxResource:      playerMaxResource,
		Alive:            true,
		AbilityCooldowns: map[string]int64{},
		QuestProgress:    map[string]platform.CharacterQuestProgress{},
		KillCredits:      map[string]platform.CharacterKillCredit{},
	}
	server.applyCharacterProgressionLocked(session, &character)
	session.LearnedAbilityIDs = append(platform.DefaultStartingLearnedAbilityIDs(), "dev_stalker_pressure", loadsimCastAbilityID)
	server.sessionsByToken[session.Token] = session
	server.sessionTokenByChar[session.CharacterID] = session.Token

	mob := server.mobs[devIsleStalkerEntityID]
	if mob == nil {
		return 0, 1, fmt.Errorf("client %d dev stalker missing", clientIndex+1)
	}
	resetMobForAbilityLoadsim(mob)
	session.ZoneID = mob.ZoneID
	session.X = mob.X - 1
	session.Y = mob.Y
	session.Z = mob.Z
	session.CurrentTargetID = mob.ID

	accepted := 0
	rejected := 0
	if err := server.activateAbilityLocked(session, "dev_stalker_pressure"); err != nil {
		rejected++
		return accepted, rejected, fmt.Errorf("client %d aura ability failed: %w", clientIndex+1, err)
	}
	accepted++
	aura := mob.ActiveAuras["dev_pressure_mark"]
	if aura.NextTickAtMs > 0 {
		if err := server.advanceAurasLocked(aura.NextTickAtMs); err != nil {
			return accepted, rejected, fmt.Errorf("client %d aura tick failed: %w", clientIndex+1, err)
		}
	}
	if refreshed := mob.ActiveAuras["dev_pressure_mark"]; refreshed.ExpiresAtMs > 0 {
		if err := server.advanceAurasLocked(refreshed.ExpiresAtMs); err != nil {
			return accepted, rejected, fmt.Errorf("client %d aura expiry failed: %w", clientIndex+1, err)
		}
	}

	session.AbilityCooldowns = map[string]int64{}
	if err := server.activateAbilityLocked(session, loadsimCastAbilityID); err != nil {
		rejected++
		return accepted, rejected, fmt.Errorf("client %d cast ability failed: %w", clientIndex+1, err)
	}
	accepted++
	session.CastEndsAtMs = nowMillis() - 1
	server.lastUpdatedAt = time.Now().Add(-time.Second)
	if err := server.advanceWorldLocked(time.Now()); err != nil {
		return accepted, rejected, fmt.Errorf("client %d cast completion failed: %w", clientIndex+1, err)
	}
	return accepted, rejected, nil
}

func resetMobForAbilityLoadsim(mob *mobState) {
	mob.Alive = true
	mob.Targetable = true
	mob.Health = mob.MaxHealth
	mob.CurrentTargetCharacter = ""
	mob.LastDamagedByCharacter = ""
	mob.LastDamagedByEntityID = ""
	mob.ActiveAuras = map[string]auraInstance{}
	mob.RespawnAtMs = 0
	mob.DeathTick = 0
	mob.RespawnTick = 0
	mob.AIState = mobAIStateIdle
}

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

type QuestBasicLoadsimOptions struct {
	Clients  int
	Duration time.Duration
	CmdRate  int
}

type QuestBasicLoadsimReport struct {
	SimulatedClients      int
	QuestsAccepted        int
	NPCKills              int
	KillCreditsAwarded    int
	LootContainersCreated int
	LootClaimsAttempted   int
	LootClaimsCompleted   int
	InventoryGrants       int
	ObjectiveUpdates      int
	QuestsReady           int
	QuestsCompleted       int
	RewardsGranted        int
	RejectedCommands      int
	AverageTickDuration   time.Duration
	MaxTickDuration       time.Duration
	MaxQueueDepth         int
	Errors                []string
}

func RunQuestBasicLoadsim(opts QuestBasicLoadsimOptions) (QuestBasicLoadsimReport, error) {
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.CmdRate <= 0 {
		opts.CmdRate = 2
	}
	report := QuestBasicLoadsimReport{SimulatedClients: opts.Clients}
	observability.LogEvent("loadsim", EventLoadsimQuestStarted, map[string]any{
		"scenario": "quest-basic",
		"clients":  opts.Clients,
		"duration": opts.Duration.String(),
		"cmdRate":  opts.CmdRate,
	})

	tempDir, err := os.MkdirTemp("", "amandacore-quest-loadsim-*")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(tempDir)

	fileStore, err := store.NewFileStore(filepath.Join(tempDir, "platform-state.json"), "loadsim-quest-basic", "http://localhost:8085")
	if err != nil {
		return report, err
	}
	server := newWorldServer(fileStore)
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
		if err := runQuestBasicClient(server, fileStore, clientIndex); err != nil {
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
	report.QuestsAccepted = server.countEvents(eventQuestAccepted)
	report.NPCKills = server.countEvents(eventNPCDied)
	report.KillCreditsAwarded = server.countEvents(eventProgressionCreditAwarded)
	report.LootContainersCreated = server.countEvents(eventLootContainerCreated)
	report.LootClaimsAttempted = server.countEvents(eventLootClaimRequested)
	report.LootClaimsCompleted = server.countEvents(eventLootClaimCompleted)
	report.InventoryGrants = server.countEvents(eventInventoryItemGranted)
	report.ObjectiveUpdates = server.countEvents(eventQuestProgressUpdated)
	report.QuestsReady = server.countEvents(eventQuestReadyToComplete)
	report.QuestsCompleted = server.countEvents(eventQuestCompleted)
	report.RewardsGranted = server.countEvents(eventQuestRewardGranted)
	report.RejectedCommands = server.countEvents(eventQuestAcceptRejected) +
		server.countEvents(eventQuestCompleteRejected) +
		server.countEvents(eventLootClaimRejected) +
		server.countEvents(eventInventoryGrantRejected)

	observability.LogEvent("loadsim", EventLoadsimQuestCompleted, map[string]any{
		"scenario":        "quest-basic",
		"clients":         opts.Clients,
		"questsCompleted": report.QuestsCompleted,
		"errors":          len(report.Errors),
	})
	return report, nil
}

func runQuestBasicClient(server *worldServer, fileStore *store.FileStore, clientIndex int) error {
	account, err := fileStore.RegisterAccount(fmt.Sprintf("quest_loadsim_%03d", clientIndex+1), "loadsim-secret")
	if err != nil {
		return fmt.Errorf("client %d account failed: %w", clientIndex+1, err)
	}
	character, err := fileStore.CreateCharacter(
		account.ID,
		"sunset-frontier-dev",
		fmt.Sprintf("QuestSim%03d", clientIndex+1),
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		return fmt.Errorf("client %d character failed: %w", clientIndex+1, err)
	}

	server.mutex.Lock()
	defer server.mutex.Unlock()

	session := &worldSessionState{
		Token:       fmt.Sprintf("loadsim_quest_%03d", clientIndex+1),
		AccountID:   account.ID,
		CharacterID: character.ID,
		DisplayName: character.DisplayName,
		ClassID:     character.ClassID,
		RealmID:     character.RealmID,
		ZoneID:      defaultZoneID,
		X:           starterSpawnX,
		Y:           starterSpawnY,
		Z:           starterSpawnZ,
		Connected:   true,
		LastSeenAt:  time.Now().Unix(),
		Health:      playerMaxHealth,
		MaxHealth:   playerMaxHealth,
		Resource:    0,
		MaxResource: playerMaxResource,
		Alive:       true,
	}
	server.applyCharacterProgressionLocked(session, &character)
	server.sessionsByToken[session.Token] = session
	server.sessionTokenByChar[session.CharacterID] = session.Token

	if err := server.acceptQuestLocked(session, devFirstHuntQuestID); err != nil {
		return fmt.Errorf("client %d accept quest failed: %w", clientIndex+1, err)
	}
	mob := server.mobs[devIsleStalkerEntityID]
	if mob == nil {
		return fmt.Errorf("client %d dev stalker missing", clientIndex+1)
	}
	session.ZoneID = mob.ZoneID
	session.X = mob.X
	session.Y = mob.Y
	session.Z = mob.Z
	session.CurrentTargetID = mob.ID

	ability := abilityDefinition{
		ID:                platform.DevBasicStrikeAbilityID,
		DisplayName:       "Basic Strike",
		ClassID:           platform.DefaultClassID,
		RequiresTarget:    true,
		RangeMeters:       3.0,
		TargetDisposition: string(NpcDispositionHostile),
		Damage:            10.0,
	}
	for mob.Alive {
		session.GlobalCooldownEnds = 0
		session.AbilityCooldowns = map[string]int64{}
		if err := server.applyAbilityEffectLocked(session, mob, nil, ability); err != nil {
			return fmt.Errorf("client %d combat failed: %w", clientIndex+1, err)
		}
	}

	var containerID string
	for _, id := range server.lootContainerOrder {
		container := server.lootContainers[id]
		if container != nil && container.OwnerCharacterID == session.CharacterID && container.SourceEntityID == mob.ID {
			containerID = id
			break
		}
	}
	if containerID == "" {
		return fmt.Errorf("client %d loot container missing", clientIndex+1)
	}
	if _, err := server.inspectLootLocked(session, containerID); err != nil {
		return fmt.Errorf("client %d inspect loot failed: %w", clientIndex+1, err)
	}
	if err := server.claimLootLocked(session, containerID); err != nil {
		return fmt.Errorf("client %d claim loot failed: %w", clientIndex+1, err)
	}
	if err := server.completeQuestLocked(session, devFirstHuntQuestID); err != nil {
		return fmt.Errorf("client %d complete quest failed: %w", clientIndex+1, err)
	}
	return nil
}

func (s *worldServer) countEvents(eventName string) int {
	count := 0
	for _, event := range s.domainEvents {
		if event.Type == eventName {
			count++
		}
	}
	return count
}

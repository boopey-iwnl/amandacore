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

type ZoneHandoffLoadsimOptions struct {
	Clients         int
	Duration        time.Duration
	CmdRate         int
	TransitionLoops int
	Shards          int
	QueueCapacity   int
}

type ZoneHandoffLoadsimReport struct {
	SimulatedClients    int
	TransitionLoops     int
	ShardCount          int
	QueueCapacity       int
	HandoffsRequested   int
	HandoffsAccepted    int
	HandoffsCompleted   int
	HandoffsRejected    int
	HandoffsRetried     int
	ExpectedRejections  int
	JournalEntries      int
	ZonePopulation      map[string]int
	ShardPopulation     map[string]int
	ShardAssignments    map[string]string
	MaxQueueDepth       int
	QueueBackpressure   int
	AverageTickDuration time.Duration
	MaxTickDuration     time.Duration
	Errors              []string
}

func RunZoneHandoffLoadsim(opts ZoneHandoffLoadsimOptions) (ZoneHandoffLoadsimReport, error) {
	if opts.Clients <= 0 {
		opts.Clients = 1
	}
	if opts.CmdRate <= 0 {
		opts.CmdRate = 2
	}
	if opts.TransitionLoops <= 0 {
		opts.TransitionLoops = 2
	}
	if opts.Shards <= 0 {
		opts.Shards = defaultZoneShardCount
	}
	if opts.QueueCapacity <= 0 {
		opts.QueueCapacity = defaultZoneCommandQueueCapacity
	}
	report := ZoneHandoffLoadsimReport{
		SimulatedClients: opts.Clients,
		TransitionLoops:  opts.TransitionLoops,
		ShardCount:       opts.Shards,
		QueueCapacity:    opts.QueueCapacity,
	}
	observability.LogEvent("loadsim", EventLoadsimZoneHandoffStarted, map[string]any{
		"scenario":        "zone-handoff-basic",
		"clients":         opts.Clients,
		"duration":        opts.Duration.String(),
		"cmdRate":         opts.CmdRate,
		"transitionLoops": opts.TransitionLoops,
		"shards":          opts.Shards,
		"queueCapacity":   opts.QueueCapacity,
	})

	tempDir, err := os.MkdirTemp("", "amandacore-zone-handoff-loadsim-*")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(tempDir)

	fileStore, err := store.NewFileStore(filepath.Join(tempDir, "platform-state.json"), "loadsim-zone-handoff-basic", "http://localhost:8085")
	if err != nil {
		return report, err
	}
	server := newWorldServer(fileStore)
	coordinator, err := NewShardCoordinator(server.zones, ShardCoordinatorOptions{ShardCount: opts.Shards, QueueCapacity: opts.QueueCapacity})
	if err != nil {
		return report, err
	}
	server.shardCoordinator = coordinator

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
		if err := runZoneHandoffLoadsimClient(server, fileStore, clientIndex, opts.TransitionLoops, &report); err != nil {
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
	report.ShardAssignments = server.shardCoordinator.ShardAssignmentSummary()
	report.ZonePopulation = server.shardCoordinator.ZonePopulation()
	report.ShardPopulation = server.shardCoordinator.ShardPopulation()
	report.MaxQueueDepth = server.shardCoordinator.MaxQueueDepth()
	for _, entry := range server.shardCoordinator.Journal() {
		report.JournalEntries++
		switch entry.Status {
		case ZoneHandoffRequested:
			report.HandoffsRequested++
		case ZoneHandoffAccepted:
			report.HandoffsAccepted++
		case ZoneHandoffCompleted:
			report.HandoffsCompleted++
		case ZoneHandoffRejected:
			report.HandoffsRejected++
			if entry.Reason == ZoneHandoffRejectQueueFull {
				report.QueueBackpressure++
			}
		case ZoneHandoffRetried:
			report.HandoffsRetried++
		}
	}

	observability.LogEvent("loadsim", EventLoadsimZoneHandoffCompleted, map[string]any{
		"scenario":          "zone-handoff-basic",
		"clients":           opts.Clients,
		"handoffsCompleted": report.HandoffsCompleted,
		"handoffsRejected":  report.HandoffsRejected,
		"retries":           report.HandoffsRetried,
		"errors":            len(report.Errors),
	})
	return report, nil
}

func runZoneHandoffLoadsimClient(server *worldServer, fileStore *store.FileStore, clientIndex int, transitionLoops int, report *ZoneHandoffLoadsimReport) error {
	account, err := fileStore.RegisterAccount(fmt.Sprintf("handoff_loadsim_%03d", clientIndex+1), "loadsim-secret")
	if err != nil {
		return fmt.Errorf("client %d account failed: %w", clientIndex+1, err)
	}
	character, err := fileStore.CreateCharacter(
		account.ID,
		"sunset-frontier-dev",
		fmt.Sprintf("HandoffSim%03d", clientIndex+1),
		platform.DefaultRaceID,
		platform.DefaultClassID,
		platform.LegacyWayfarerArchetypeID)
	if err != nil {
		return fmt.Errorf("client %d character failed: %w", clientIndex+1, err)
	}

	server.mutex.Lock()
	defer server.mutex.Unlock()

	session := &worldSessionState{
		Token:             fmt.Sprintf("loadsim_handoff_%03d", clientIndex+1),
		AccountID:         account.ID,
		CharacterID:       character.ID,
		DisplayName:       character.DisplayName,
		ClassID:           character.ClassID,
		Level:             6,
		RealmID:           character.RealmID,
		ZoneID:            defaultZoneID,
		X:                 470,
		Y:                 260,
		Z:                 playableGroundZ,
		Connected:         true,
		LastSeenAt:        time.Now().Unix(),
		Health:            playerMaxHealth,
		MaxHealth:         playerMaxHealth,
		Resource:          0,
		MaxResource:       playerMaxResource,
		Alive:             true,
		Inventory:         platform.NormalizeInventorySlots(nil),
		LearnedAbilityIDs: platform.DefaultStartingLearnedAbilityIDs(),
		ActionBarSlots:    platform.DefaultActionBarSlots(platform.DefaultStartingLearnedAbilityIDs()),
	}
	server.applyCharacterProgressionLocked(session, &character)
	server.sessionsByToken[session.Token] = session
	server.sessionTokenByChar[session.CharacterID] = session.Token
	server.syncSessionZoneOwnershipLocked(session)

	for step := 0; step < transitionLoops; step++ {
		transitionID := "to_brindlebrook"
		if session.ZoneID == secondZoneID {
			transitionID = "from_stonewake"
		}
		if clientIndex == 0 && step == 0 {
			gate := zoneHandoffGateDefinitions[transitionID]
			destination, err := server.shardCoordinator.ResolveZone(gate.ToZoneID)
			if err != nil {
				return err
			}
			if err := server.setShardWorkerStateLocked(destination.ShardID, ShardWorkerUnavailable, "loadsim_expected_retry"); err != nil {
				return err
			}
			if _, err := server.requestZoneHandoffLocked(session, transitionID); err == nil {
				return fmt.Errorf("client %d expected unavailable shard rejection", clientIndex+1)
			}
			report.ExpectedRejections++
			if err := server.setShardWorkerStateLocked(destination.ShardID, ShardWorkerActive, "loadsim_retry"); err != nil {
				return err
			}
			server.shardCoordinator.appendJournal(ZoneHandoffJournalEntry{
				HandoffID:    "loadsim_retry",
				CharacterID:  session.CharacterID,
				TransitionID: transitionID,
				Status:       ZoneHandoffRetried,
				FromZoneID:   session.ZoneID,
				ToZoneID:     gate.ToZoneID,
				AtMs:         nowMillis(),
				Message:      "retry after expected unavailable destination shard",
			})
		}
		if _, err := server.requestZoneHandoffLocked(session, transitionID); err != nil {
			return fmt.Errorf("client %d handoff %d failed: %w", clientIndex+1, step+1, err)
		}
	}
	return nil
}

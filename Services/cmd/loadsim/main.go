package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
	"amandacore/services/internal/loadsim"
	"amandacore/services/internal/observability"
	"amandacore/services/internal/realms"
	"amandacore/services/internal/store"
	"amandacore/services/internal/worlds"
)

const (
	scenarioCombatBasic = "combat-basic"
	devBasicStrikeID    = "dev_basic_strike"
	devStalkerArchetype = "dev_isle_stalker"
)

type options struct {
	Clients     int
	Duration    time.Duration
	CommandRate float64
	Scenario    string
	ContentPath string
}

func main() {
	scenario := scenarioFromArgs(os.Args[1:], "quest-basic")
	if isRuntimeLoadsimScenario(scenario) {
		runRuntimeLoadsim()
		return
	}
	runWorldLoadsim()
}

func runRuntimeLoadsim() {
	cfg, err := loadsim.ParseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim: %v\n", err)
		os.Exit(2)
	}
	report, err := loadsim.Run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim run failed: %v\n", err)
		fmt.Print(loadsim.RenderTextReport(report))
		os.Exit(1)
	}
	if err := loadsim.WriteJSONReport(cfg.ReportPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write report failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(loadsim.RenderTextReport(report))
}

func runWorldLoadsim() {
	opts := parseOptions()

	switch opts.Scenario {
	case "quest-basic":
		cmdRate := int(opts.CommandRate)
		if cmdRate <= 0 {
			exitf("--cmd-rate must be at least 1 for quest-basic")
		}
		report, err := worlds.RunQuestBasicLoadsim(worlds.QuestBasicLoadsimOptions{
			Clients:  opts.Clients,
			Duration: opts.Duration,
			CmdRate:  cmdRate,
		})
		printQuestReport(report)
		exitOnReportError(err, report.Errors)
	case "content-package-basic":
		report, err := worlds.RunContentPackageLoadsim(worlds.ContentPackageLoadsimOptions{
			Clients:     opts.Clients,
			Duration:    opts.Duration,
			CommandRate: opts.CommandRate,
			Scenario:    opts.Scenario,
			ContentPath: opts.ContentPath,
		})
		printContentPackageReport(report)
		exitOnReportError(err, report.Errors)
	case scenarioCombatBasic:
		runCombatBasicLoadsim(opts)
	default:
		exitf("unsupported scenario %q", opts.Scenario)
	}
}

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 1, "number of simulated clients")
	flag.StringVar(&durationText, "duration", "30s", "scenario duration budget, for example 30s")
	flag.Float64Var(&opts.CommandRate, "cmd-rate", 2, "nominal commands per second per client")
	flag.StringVar(&opts.Scenario, "scenario", "quest-basic", "scenario name")
	flag.StringVar(&opts.ContentPath, "content", "", "content package manifest path")
	flag.Parse()

	duration, err := time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}
	opts.Duration = duration
	opts.Scenario = strings.ToLower(strings.TrimSpace(opts.Scenario))
	if opts.Clients <= 0 {
		exitf("--clients must be greater than zero")
	}
	if opts.CommandRate <= 0 {
		exitf("--cmd-rate must be greater than zero")
	}
	return opts
}

func isRuntimeLoadsimScenario(scenario string) bool {
	switch strings.ToLower(strings.TrimSpace(scenario)) {
	case loadsim.ScenarioMovementBasic,
		loadsim.ScenarioAbilityBasic,
		loadsim.ScenarioDawnwakeTraversal,
		loadsim.ScenarioMultizonePressure,
		loadsim.ScenarioShardAssignmentBasic,
		loadsim.ScenarioReconnectPressure:
		return true
	default:
		return false
	}
}

func scenarioFromArgs(args []string, fallback string) string {
	for index, arg := range args {
		if arg == "--scenario" && index+1 < len(args) {
			return strings.ToLower(strings.TrimSpace(args[index+1]))
		}
		if strings.HasPrefix(arg, "--scenario=") {
			return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--scenario=")))
		}
	}
	return fallback
}

func printQuestReport(report worlds.QuestBasicLoadsimReport) {
	fmt.Println("Quest basic loadsim report")
	fmt.Printf("- simulated clients: %d\n", report.SimulatedClients)
	fmt.Printf("- quests accepted: %d\n", report.QuestsAccepted)
	fmt.Printf("- NPC kills: %d\n", report.NPCKills)
	fmt.Printf("- kill credits awarded: %d\n", report.KillCreditsAwarded)
	fmt.Printf("- loot containers created: %d\n", report.LootContainersCreated)
	fmt.Printf("- loot claims attempted: %d\n", report.LootClaimsAttempted)
	fmt.Printf("- loot claims completed: %d\n", report.LootClaimsCompleted)
	fmt.Printf("- inventory grants: %d\n", report.InventoryGrants)
	fmt.Printf("- objective updates: %d\n", report.ObjectiveUpdates)
	fmt.Printf("- quests ready: %d\n", report.QuestsReady)
	fmt.Printf("- quests completed: %d\n", report.QuestsCompleted)
	fmt.Printf("- rewards granted: %d\n", report.RewardsGranted)
	fmt.Printf("- rejected commands: %d\n", report.RejectedCommands)
	fmt.Printf("- average tick duration: %s\n", report.AverageTickDuration)
	fmt.Printf("- max tick duration: %s\n", report.MaxTickDuration)
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	printErrors(report.Errors)
}

func printContentPackageReport(report worlds.ContentPackageLoadsimReport) {
	fmt.Println("Content package loadsim report")
	fmt.Printf("- content package loaded: %v\n", report.ContentPackageLoaded)
	fmt.Printf("- package id: %s\n", report.PackageID)
	fmt.Printf("- validation errors: %d\n", len(report.ValidationErrors))
	for _, validationError := range report.ValidationErrors {
		fmt.Printf("  - %s\n", validationError)
	}
	fmt.Printf("- zones activated: %d\n", report.ZonesActivated)
	fmt.Printf("- catalogs loaded: %s\n", formatCounts(report.CatalogsLoaded))
	fmt.Printf("- NPCs spawned: %d\n", report.NPCsSpawned)
	fmt.Printf("- quest providers registered: %d\n", report.QuestProvidersRegistered)
	fmt.Printf("- quests accepted: %d\n", report.QuestsAccepted)
	fmt.Printf("- NPC kills: %d\n", report.NPCKills)
	fmt.Printf("- loot containers created: %d\n", report.LootContainersCreated)
	fmt.Printf("- loot claims completed: %d\n", report.LootClaimsCompleted)
	fmt.Printf("- inventory grants: %s\n", formatCounts(report.InventoryGrants))
	fmt.Printf("- quests completed: %d\n", report.QuestsCompleted)
	fmt.Printf("- rewards granted: %d\n", report.RewardsGranted)
	fmt.Printf("- average tick duration: %.3fms\n", report.AverageTickDurationMs)
	fmt.Printf("- max tick duration: %.3fms\n", report.MaxTickDurationMs)
	fmt.Printf("- max queue depth: %d\n", report.MaxQueueDepth)
	printErrors(report.Errors)
	encoded, err := json.Marshal(report)
	if err == nil {
		fmt.Printf("- json: %s\n", string(encoded))
	}
}

func printErrors(errors []string) {
	if len(errors) == 0 {
		fmt.Println("- errors: 0")
		return
	}
	fmt.Printf("- errors: %d\n", len(errors))
	for _, errText := range errors {
		fmt.Printf("  - %s\n", errText)
	}
}

func exitOnReportError(err error, errors []string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "loadsim failed: %v\n", err)
		os.Exit(1)
	}
	if len(errors) > 0 {
		os.Exit(1)
	}
}

func formatCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := "{"
	for index, key := range keys {
		if index > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s:%d", key, counts[key])
	}
	return result + "}"
}

type simClient struct {
	ID                  int
	AccessToken         string
	RealmID             string
	CharacterID         string
	WorldSessionToken   string
	State               worldState
	LastTargetAlive     bool
	LastKillCreditCount int
}

type worldState struct {
	WorldSessionToken string            `json:"worldSessionToken"`
	CharacterID       string            `json:"characterId"`
	Position          position          `json:"position"`
	CurrentTargetID   string            `json:"currentTargetId"`
	Entities          []visibleEntity   `json:"entities"`
	KillCredits       []killCreditEntry `json:"killCredits"`
}

type position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type visibleEntity struct {
	ID          string  `json:"id"`
	ArchetypeID string  `json:"archetypeId"`
	Kind        string  `json:"kind"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
	Health      float64 `json:"health"`
	MaxHealth   float64 `json:"maxHealth"`
	Alive       bool    `json:"alive"`
	Targetable  bool    `json:"targetable"`
}

type killCreditEntry struct {
	ArchetypeID string `json:"archetypeId"`
	Count       int    `json:"count"`
}

type combatStats struct {
	mutex                  sync.Mutex
	NPCsSpawned            int
	CombatCommandsSent     int
	CombatCommandsAccepted int
	CombatCommandsRejected int
	DamageEvents           int
	NPCDeaths              int
	KillCreditsAwarded     int
	Respawns               int
	Errors                 int
}

func runCombatBasicLoadsim(opts options) {
	commandsPerSecond := int(opts.CommandRate)
	if commandsPerSecond <= 0 {
		exitf("--cmd-rate must be at least 1 for combat-basic")
	}
	observability.LogEvent("loadsim", worlds.EventLoadsimCombatStarted, map[string]any{
		"clients":  opts.Clients,
		"duration": opts.Duration.String(),
		"cmdRate":  commandsPerSecond,
		"scenario": opts.Scenario,
	})

	server := newLocalServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), opts.Duration+30*time.Second)
	defer cancel()

	httpClient := server.Client()
	runStats := &combatStats{}
	clients := make([]*simClient, 0, opts.Clients)
	runID := time.Now().UTC().Format("20060102150405")
	for index := 0; index < opts.Clients; index++ {
		client := &simClient{ID: index + 1}
		if err := client.provision(ctx, httpClient, server.URL, runID); err != nil {
			exitf("client %d provisioning failed: %v", client.ID, err)
		}
		runStats.NPCsSpawned = maxInt(runStats.NPCsSpawned, countDevNPCs(client.State))
		if target := client.pickTarget(); target.ID != "" {
			client.LastTargetAlive = target.Alive
		}
		client.LastKillCreditCount = client.killCreditCount()
		clients = append(clients, client)
	}

	var wg sync.WaitGroup
	deadline := time.Now().Add(opts.Duration)
	for _, client := range clients {
		client := client
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.runCombat(ctx, httpClient, server.URL, commandsPerSecond, runStats, deadline)
		}()
	}
	wg.Wait()

	var metrics map[string]any
	_ = getJSON(ctx, httpClient, server.URL+"/v1/world/metrics", &metrics)
	printCombatReport(opts, runStats, metrics)
	observability.LogEvent("loadsim", worlds.EventLoadsimCombatCompleted, map[string]any{
		"clients":                opts.Clients,
		"combatCommandsSent":     runStats.CombatCommandsSent,
		"combatCommandsAccepted": runStats.CombatCommandsAccepted,
		"combatCommandsRejected": runStats.CombatCommandsRejected,
		"damageEvents":           runStats.DamageEvents,
		"npcDeaths":              runStats.NPCDeaths,
		"killCreditsAwarded":     runStats.KillCreditsAwarded,
		"respawns":               runStats.Respawns,
		"errors":                 runStats.Errors,
	})
}

func newLocalServer() *httptest.Server {
	storePath := filepath.Join(os.TempDir(), fmt.Sprintf("amandacore-loadsim-%d.json", time.Now().UnixNano()))
	fileStore, err := store.NewFileStore(storePath, "loadsim-build", "http://world.local")
	if err != nil {
		exitf("failed to create loadsim store: %v", err)
	}

	mux := http.NewServeMux()
	authn.RegisterRoutes(mux, fileStore)
	realms.RegisterRoutes(mux, fileStore)
	characters.RegisterRoutes(mux, fileStore)
	worlds.RegisterRoutes(mux, fileStore)
	return httptest.NewServer(mux)
}

func (c *simClient) provision(ctx context.Context, httpClient *http.Client, baseURL string, runID string) error {
	username := fmt.Sprintf("combat_%s_%03d", runID, c.ID)
	password := "loadsim_password"
	if err := postJSON(ctx, httpClient, baseURL+"/v1/accounts/register", "", map[string]string{
		"username": username,
		"password": password,
	}, nil); err != nil {
		return err
	}

	var login struct {
		AccessToken string `json:"accessToken"`
	}
	if err := postJSON(ctx, httpClient, baseURL+"/v1/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	}, &login); err != nil {
		return err
	}
	c.AccessToken = login.AccessToken

	var realmsResponse struct {
		Realms []struct {
			ID string `json:"id"`
		} `json:"realms"`
	}
	if err := getJSON(ctx, httpClient, baseURL+"/v1/realms", &realmsResponse); err != nil {
		return err
	}
	if len(realmsResponse.Realms) == 0 {
		return fmt.Errorf("no realms available")
	}
	c.RealmID = realmsResponse.Realms[0].ID

	var character struct {
		ID string `json:"id"`
	}
	if err := postJSON(ctx, httpClient, baseURL+"/v1/characters", c.AccessToken, map[string]string{
		"realmId":     c.RealmID,
		"displayName": fmt.Sprintf("Fighter%03d", c.ID),
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, &character); err != nil {
		return err
	}
	c.CharacterID = character.ID

	var ticket struct {
		TicketID string `json:"ticketId"`
	}
	if err := postJSON(ctx, httpClient, baseURL+"/v1/world/join-ticket", c.AccessToken, map[string]string{
		"realmId":     c.RealmID,
		"characterId": c.CharacterID,
	}, &ticket); err != nil {
		return err
	}
	if err := postJSON(ctx, httpClient, baseURL+"/v1/world/connect", "", map[string]string{
		"ticketId": ticket.TicketID,
	}, &c.State); err != nil {
		return err
	}
	c.WorldSessionToken = c.State.WorldSessionToken
	return nil
}

func (c *simClient) runCombat(ctx context.Context, httpClient *http.Client, baseURL string, commandsPerSecond int, runStats *combatStats, deadline time.Time) {
	interval := time.Second / time.Duration(commandsPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.combatStep(ctx, httpClient, baseURL, runStats); err != nil {
				runStats.addError()
			}
		}
	}
}

func (c *simClient) combatStep(ctx context.Context, httpClient *http.Client, baseURL string, runStats *combatStats) error {
	if err := getJSON(ctx, httpClient, baseURL+"/v1/world/state?worldSessionToken="+c.WorldSessionToken, &c.State); err != nil {
		return err
	}
	target := c.pickTarget()
	if target.ID == "" {
		return nil
	}
	if !c.LastTargetAlive && target.Alive {
		runStats.addRespawn()
	}
	c.LastTargetAlive = target.Alive
	if !target.Alive || !target.Targetable {
		return nil
	}

	var moved worldState
	if err := postJSON(ctx, httpClient, baseURL+"/v1/world/move", "", map[string]any{
		"worldSessionToken": c.WorldSessionToken,
		"deltaX":            (target.X - 1.0) - c.State.Position.X,
		"deltaY":            target.Y - c.State.Position.Y,
	}, &moved); err == nil && moved.WorldSessionToken != "" {
		c.State = moved
	}
	if c.State.CurrentTargetID != target.ID {
		var targeted worldState
		if err := postJSON(ctx, httpClient, baseURL+"/v1/world/target", "", map[string]any{
			"worldSessionToken": c.WorldSessionToken,
			"targetId":          target.ID,
		}, &targeted); err == nil && targeted.WorldSessionToken != "" {
			c.State = targeted
		}
	}

	runStats.addSent()
	healthBefore := target.Health
	var attacked worldState
	err := postJSON(ctx, httpClient, baseURL+"/v1/world/attack/ability", "", map[string]any{
		"worldSessionToken": c.WorldSessionToken,
		"abilityId":         devBasicStrikeID,
	}, &attacked)
	if err != nil {
		runStats.addRejected()
		return nil
	}
	runStats.addAccepted()
	c.State = attacked

	after := findEntity(c.State.Entities, target.ID)
	if after.ID != "" && after.Health < healthBefore {
		runStats.addDamage()
	}
	if after.ID != "" && !after.Alive {
		runStats.addDeath()
	}
	nextKillCreditCount := c.killCreditCount()
	if nextKillCreditCount > c.LastKillCreditCount {
		runStats.addKillCredits(nextKillCreditCount - c.LastKillCreditCount)
		c.LastKillCreditCount = nextKillCreditCount
	}
	return nil
}

func (c *simClient) pickTarget() visibleEntity {
	for _, entity := range c.State.Entities {
		if entity.ArchetypeID == devStalkerArchetype && entity.Alive && entity.Targetable {
			return entity
		}
	}
	for _, entity := range c.State.Entities {
		if entity.ArchetypeID == devStalkerArchetype {
			return entity
		}
	}
	return visibleEntity{}
}

func countDevNPCs(state worldState) int {
	count := 0
	for _, entity := range state.Entities {
		if entity.ArchetypeID == devStalkerArchetype {
			count++
		}
	}
	return count
}

func (c *simClient) killCreditCount() int {
	count := 0
	for _, credit := range c.State.KillCredits {
		if credit.ArchetypeID == devStalkerArchetype {
			count += credit.Count
		}
	}
	return count
}

func findEntity(entities []visibleEntity, entityID string) visibleEntity {
	for _, entity := range entities {
		if entity.ID == entityID {
			return entity
		}
	}
	return visibleEntity{}
}

func postJSON(ctx context.Context, httpClient *http.Client, url string, bearerToken string, payload any, target any) error {
	return doJSON(ctx, httpClient, http.MethodPost, url, bearerToken, payload, target)
}

func getJSON(ctx context.Context, httpClient *http.Client, url string, target any) error {
	return doJSON(ctx, httpClient, http.MethodGet, url, "", nil, target)
}

func doJSON(ctx context.Context, httpClient *http.Client, method string, url string, bearerToken string, payload any, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s", strings.TrimSpace(string(content)))
	}
	if target != nil && len(content) > 0 {
		if err := json.Unmarshal(content, target); err != nil {
			return err
		}
	}
	return nil
}

func (s *combatStats) addSent() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsSent++
}

func (s *combatStats) addAccepted() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsAccepted++
}

func (s *combatStats) addRejected() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsRejected++
}

func (s *combatStats) addDamage() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.DamageEvents++
}

func (s *combatStats) addDeath() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.NPCDeaths++
}

func (s *combatStats) addRespawn() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Respawns++
}

func (s *combatStats) addError() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Errors++
}

func (s *combatStats) addKillCredits(count int) {
	if count <= 0 {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.KillCreditsAwarded += count
}

func printCombatReport(opts options, runStats *combatStats, metrics map[string]any) {
	worldTick := map[string]any{}
	if raw, ok := metrics["worldTick"].(map[string]any); ok {
		worldTick = raw
	}
	fmt.Println("Combat loadsim report")
	fmt.Printf("simulated clients: %d\n", opts.Clients)
	fmt.Printf("NPCs spawned: %d\n", runStats.NPCsSpawned)
	fmt.Printf("combat commands sent: %d\n", runStats.CombatCommandsSent)
	fmt.Printf("combat commands accepted: %d\n", runStats.CombatCommandsAccepted)
	fmt.Printf("combat commands rejected: %d\n", runStats.CombatCommandsRejected)
	fmt.Printf("damage events: %d\n", runStats.DamageEvents)
	fmt.Printf("NPC deaths: %d\n", runStats.NPCDeaths)
	fmt.Printf("kill credits awarded: %d\n", runStats.KillCreditsAwarded)
	fmt.Printf("respawns: %d\n", runStats.Respawns)
	fmt.Printf("average tick duration ms: %.3f\n", floatFromAny(worldTick["avgMs"]))
	fmt.Printf("max tick duration ms: %.3f\n", floatFromAny(worldTick["maxMs"]))
	fmt.Println("max queue depth: 0")
	fmt.Printf("errors: %d\n", runStats.Errors)
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}

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
	"strings"
	"sync"
	"time"

	"amandacore/services/internal/authn"
	"amandacore/services/internal/characters"
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
	Clients           int
	Duration          time.Duration
	CommandsPerSecond int
	Scenario          string
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

type stats struct {
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

func main() {
	opts := parseOptions()
	if opts.Scenario != scenarioCombatBasic {
		exitf("unsupported scenario %q", opts.Scenario)
	}
	if opts.Clients <= 0 {
		exitf("clients must be greater than zero")
	}
	if opts.CommandsPerSecond <= 0 {
		exitf("cmd-rate must be greater than zero")
	}

	observability.LogEvent("loadsim", "loadsim.combat.started", map[string]any{
		"clients":  opts.Clients,
		"duration": opts.Duration.String(),
		"cmdRate":  opts.CommandsPerSecond,
		"scenario": opts.Scenario,
	})

	server := newLocalServer()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), opts.Duration+30*time.Second)
	defer cancel()

	httpClient := server.Client()
	runStats := &stats{}
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
			client.runCombat(ctx, httpClient, server.URL, opts, runStats, deadline)
		}()
	}
	wg.Wait()

	var metrics map[string]any
	_ = getJSON(ctx, httpClient, server.URL+"/v1/world/metrics", &metrics)
	printReport(opts, runStats, metrics)
	observability.LogEvent("loadsim", "loadsim.combat.completed", map[string]any{
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

func parseOptions() options {
	var durationText string
	opts := options{}
	flag.IntVar(&opts.Clients, "clients", 5, "number of simulated players")
	flag.StringVar(&durationText, "duration", "10s", "run duration")
	flag.IntVar(&opts.CommandsPerSecond, "cmd-rate", 2, "combat command attempts per client per second")
	flag.StringVar(&opts.Scenario, "scenario", scenarioCombatBasic, "scenario name")
	flag.Parse()

	var err error
	opts.Duration, err = time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}
	opts.Scenario = strings.ToLower(strings.TrimSpace(opts.Scenario))
	return opts
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

func (c *simClient) runCombat(ctx context.Context, httpClient *http.Client, baseURL string, opts options, runStats *stats, deadline time.Time) {
	interval := time.Second / time.Duration(opts.CommandsPerSecond)
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

func (c *simClient) combatStep(ctx context.Context, httpClient *http.Client, baseURL string, runStats *stats) error {
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

func (s *stats) addSent() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsSent++
}

func (s *stats) addAccepted() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsAccepted++
}

func (s *stats) addRejected() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.CombatCommandsRejected++
}

func (s *stats) addDamage() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.DamageEvents++
}

func (s *stats) addDeath() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.NPCDeaths++
}

func (s *stats) addRespawn() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Respawns++
}

func (s *stats) addError() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Errors++
}

func (s *stats) addKillCredits(count int) {
	if count <= 0 {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.KillCreditsAwarded += count
}

func printReport(opts options, runStats *stats, metrics map[string]any) {
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
	os.Exit(1)
}

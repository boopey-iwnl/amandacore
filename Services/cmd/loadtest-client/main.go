package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type options struct {
	AuthEndpoint      string
	RealmEndpoint     string
	CharacterEndpoint string
	WorldEndpoint     string
	Clients           int
	Duration          time.Duration
	Scenario          string
	StepInterval      time.Duration
	OutRoot           string
}

type loadClient struct {
	ID                int
	Username          string
	Password          string
	AccessToken       string
	RealmID           string
	CharacterID       string
	WorldSessionToken string
	State             worldState
}

type runStats struct {
	mutex        sync.Mutex
	startedAt    time.Time
	finishedAt   time.Time
	counts       map[string]int64
	errors       map[string]int64
	timingsMs    map[string][]float64
	statusCounts map[string]map[int]int64
	desyncs      []string
	eventFile    *os.File
	eventEncoder *json.Encoder
}

type requestEvent struct {
	Timestamp  string  `json:"timestamp"`
	ClientID   int     `json:"clientId"`
	Operation  string  `json:"operation"`
	StatusCode int     `json:"statusCode"`
	DurationMs float64 `json:"durationMs"`
	Error      string  `json:"error,omitempty"`
}

type loginResponse struct {
	AccessToken string `json:"accessToken"`
	AccountID   string `json:"accountId"`
}

type manifestResponse struct {
	ID             string `json:"id"`
	DisplayVersion string `json:"displayVersion"`
	Channel        string `json:"channel"`
}

type realmsResponse struct {
	Realms []realmSummary `json:"realms"`
}

type realmSummary struct {
	ID string `json:"id"`
}

type characterResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type ticketResponse struct {
	TicketID string `json:"ticketId"`
}

type worldState struct {
	WorldSessionToken string          `json:"worldSessionToken"`
	CharacterID       string          `json:"characterId"`
	ZoneID            string          `json:"zoneId"`
	DisplayName       string          `json:"displayName"`
	Position          position        `json:"position"`
	Health            float64         `json:"health"`
	MaxHealth         float64         `json:"maxHealth"`
	Alive             bool            `json:"alive"`
	CurrentTargetID   string          `json:"currentTargetId"`
	AutoAttackActive  bool            `json:"autoAttackActive"`
	Entities          []visibleEntity `json:"entities"`
	Quests            []questSummary  `json:"quests"`
}

type position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type visibleEntity struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Z          float64 `json:"z"`
	Health     float64 `json:"health"`
	MaxHealth  float64 `json:"maxHealth"`
	Alive      bool    `json:"alive"`
	Targetable bool    `json:"targetable"`
}

type questSummary struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

func main() {
	opts := parseOptions()
	if opts.Clients <= 0 {
		exitf("clients must be greater than zero")
	}

	runID := time.Now().UTC().Format("20060102-150405")
	runDir := filepath.Join(opts.OutRoot, fmt.Sprintf("loadtest-%s-%s-clients%d", runID, opts.Scenario, opts.Clients))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		exitf("failed to create output directory: %v", err)
	}

	stats, err := newRunStats(filepath.Join(runDir, "events.jsonl"))
	if err != nil {
		exitf("failed to create event log: %v", err)
	}
	defer stats.close()

	ctx, cancel := context.WithTimeout(context.Background(), opts.Duration+2*time.Minute)
	defer cancel()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	manifest := fetchManifest(ctx, httpClient, opts, stats)
	clients := provisionClients(ctx, httpClient, opts, stats, runID)

	var wg sync.WaitGroup
	deadline := time.Now().Add(opts.Duration)
	for index := range clients {
		client := clients[index]
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.run(ctx, httpClient, opts, stats, deadline)
		}()
	}
	wg.Wait()
	stats.finishedAt = time.Now().UTC()

	serverMetrics := map[string]any{}
	_, _ = doJSON(ctx, httpClient, http.MethodGet, opts.WorldEndpoint+"/v1/world/metrics", "", nil, &serverMetrics, "world_metrics", -1, stats)

	if err := writeSummary(runDir, opts, manifest, stats, serverMetrics); err != nil {
		exitf("failed to write summary: %v", err)
	}

	fmt.Printf("Load test complete. Summary: %s\n", filepath.Join(runDir, "summary.md"))
}

func parseOptions() options {
	var durationText string
	var stepText string
	opts := options{}
	flag.StringVar(&opts.AuthEndpoint, "auth-endpoint", "http://localhost:8081", "auth/account service base URL")
	flag.StringVar(&opts.RealmEndpoint, "realm-endpoint", "http://localhost:8083", "realm service base URL")
	flag.StringVar(&opts.CharacterEndpoint, "character-endpoint", "http://localhost:8084", "character service base URL")
	flag.StringVar(&opts.WorldEndpoint, "world-endpoint", "http://localhost:8085", "world service base URL")
	flag.IntVar(&opts.Clients, "clients", 2, "number of simulated clients")
	flag.StringVar(&durationText, "duration", "5m", "run duration, for example 1m, 5m, or 15m")
	flag.StringVar(&opts.Scenario, "scenario", "mixed", "idle, move, combat, reconnect, or mixed")
	flag.StringVar(&stepText, "step-interval", "250ms", "per-client action interval")
	flag.StringVar(&opts.OutRoot, "out", filepath.Join("Infra", "dev", "load-tests"), "output root directory")
	flag.Parse()

	var err error
	opts.Duration, err = time.ParseDuration(durationText)
	if err != nil {
		exitf("invalid duration: %v", err)
	}
	opts.StepInterval, err = time.ParseDuration(stepText)
	if err != nil {
		exitf("invalid step interval: %v", err)
	}
	opts.Scenario = strings.ToLower(strings.TrimSpace(opts.Scenario))
	switch opts.Scenario {
	case "idle", "move", "combat", "reconnect", "mixed":
	default:
		exitf("unsupported scenario %q", opts.Scenario)
	}
	return opts
}

func newRunStats(eventPath string) (*runStats, error) {
	eventFile, err := os.Create(eventPath)
	if err != nil {
		return nil, err
	}
	return &runStats{
		startedAt:    time.Now().UTC(),
		counts:       map[string]int64{},
		errors:       map[string]int64{},
		timingsMs:    map[string][]float64{},
		statusCounts: map[string]map[int]int64{},
		eventFile:    eventFile,
		eventEncoder: json.NewEncoder(eventFile),
	}, nil
}

func (s *runStats) close() {
	if s.eventFile != nil {
		_ = s.eventFile.Close()
	}
}

func (s *runStats) record(clientID int, operation string, statusCode int, duration time.Duration, err error) {
	durationMs := float64(duration.Microseconds()) / 1000.0
	errorText := ""
	if err != nil {
		errorText = err.Error()
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.counts[operation]++
	if err != nil || statusCode >= 400 {
		s.errors[operation]++
	}
	s.timingsMs[operation] = append(s.timingsMs[operation], durationMs)
	if s.statusCounts[operation] == nil {
		s.statusCounts[operation] = map[int]int64{}
	}
	s.statusCounts[operation][statusCode]++
	_ = s.eventEncoder.Encode(requestEvent{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		ClientID:   clientID,
		Operation:  operation,
		StatusCode: statusCode,
		DurationMs: durationMs,
		Error:      errorText,
	})
}

func (s *runStats) recordDesync(clientID int, message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.desyncs = append(s.desyncs, fmt.Sprintf("client %d: %s", clientID, message))
}

func fetchManifest(ctx context.Context, httpClient *http.Client, opts options, stats *runStats) manifestResponse {
	var manifest manifestResponse
	_, _ = doJSON(ctx, httpClient, http.MethodGet, opts.RealmEndpoint+"/v1/patch/manifest", "", nil, &manifest, "manifest", -1, stats)
	return manifest
}

func provisionClients(ctx context.Context, httpClient *http.Client, opts options, stats *runStats, runID string) []*loadClient {
	clients := make([]*loadClient, 0, opts.Clients)
	for index := 0; index < opts.Clients; index++ {
		client := &loadClient{
			ID:       index + 1,
			Username: fmt.Sprintf("load_%s_%03d", runID, index+1),
			Password: "loadtest_password",
		}
		if err := client.provision(ctx, httpClient, opts, stats); err != nil {
			exitf("client %d provisioning failed: %v", client.ID, err)
		}
		clients = append(clients, client)
	}
	return clients
}

func (c *loadClient) provision(ctx context.Context, httpClient *http.Client, opts options, stats *runStats) error {
	_, err := doJSON(ctx, httpClient, http.MethodPost, opts.AuthEndpoint+"/v1/accounts/register", "", map[string]string{
		"username": c.Username,
		"password": c.Password,
	}, nil, "register", c.ID, stats)
	if err != nil {
		return err
	}

	var login loginResponse
	_, err = doJSON(ctx, httpClient, http.MethodPost, opts.AuthEndpoint+"/v1/auth/login", "", map[string]string{
		"username": c.Username,
		"password": c.Password,
	}, &login, "login", c.ID, stats)
	if err != nil {
		return err
	}
	c.AccessToken = login.AccessToken

	var realms realmsResponse
	_, err = doJSON(ctx, httpClient, http.MethodGet, opts.RealmEndpoint+"/v1/realms", "", nil, &realms, "realms", c.ID, stats)
	if err != nil {
		return err
	}
	if len(realms.Realms) == 0 {
		return fmt.Errorf("no realms returned")
	}
	c.RealmID = realms.Realms[0].ID

	var character characterResponse
	_, err = doJSON(ctx, httpClient, http.MethodPost, opts.CharacterEndpoint+"/v1/characters", c.AccessToken, map[string]string{
		"realmId":     c.RealmID,
		"displayName": fmt.Sprintf("Load%03d%s", c.ID, c.Username[len(c.Username)-4:]),
		"raceId":      "human",
		"classId":     "warrior",
		"archetypeId": "wayfarer_warden",
	}, &character, "character_create", c.ID, stats)
	if err != nil {
		return err
	}
	c.CharacterID = character.ID

	var ticket ticketResponse
	_, err = doJSON(ctx, httpClient, http.MethodPost, opts.WorldEndpoint+"/v1/world/join-ticket", c.AccessToken, map[string]string{
		"realmId":     c.RealmID,
		"characterId": c.CharacterID,
	}, &ticket, "join_ticket", c.ID, stats)
	if err != nil {
		return err
	}

	_, err = doJSON(ctx, httpClient, http.MethodPost, opts.WorldEndpoint+"/v1/world/connect", "", map[string]string{
		"ticketId": ticket.TicketID,
	}, &c.State, "connect", c.ID, stats)
	if err != nil {
		return err
	}
	c.WorldSessionToken = c.State.WorldSessionToken
	return nil
}

func (c *loadClient) run(ctx context.Context, httpClient *http.Client, opts options, stats *runStats, deadline time.Time) {
	scenario := opts.Scenario
	if scenario == "mixed" {
		switch c.ID % 4 {
		case 0:
			scenario = "reconnect"
		case 1:
			scenario = "move"
		case 2:
			scenario = "combat"
		default:
			scenario = "idle"
		}
	}

	ticker := time.NewTicker(opts.StepInterval)
	defer ticker.Stop()

	step := 0
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			step++
			c.runStep(ctx, httpClient, opts, stats, scenario, step)
		}
	}
}

func (c *loadClient) runStep(ctx context.Context, httpClient *http.Client, opts options, stats *runStats, scenario string, step int) {
	switch scenario {
	case "idle":
		if step%4 == 0 {
			_ = c.fetchState(ctx, httpClient, opts, stats)
		}
	case "move":
		c.movePattern(ctx, httpClient, opts, stats, step)
	case "combat":
		c.combatStep(ctx, httpClient, opts, stats, step)
	case "reconnect":
		if step%12 == 0 {
			c.disconnectReconnect(ctx, httpClient, opts, stats)
			return
		}
		if step%4 == 0 {
			_ = c.fetchState(ctx, httpClient, opts, stats)
		}
	}
}

func (c *loadClient) movePattern(ctx context.Context, httpClient *http.Client, opts options, stats *runStats, step int) {
	pattern := []position{
		{X: 0.5, Y: 0},
		{X: 0.5, Y: 0},
		{X: 0, Y: 0.5},
		{X: -0.5, Y: 0},
		{X: -0.5, Y: 0},
		{X: 0, Y: -0.5},
	}
	delta := pattern[(step+c.ID)%len(pattern)]
	previous := c.State.Position
	var state worldState
	err := postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/move", map[string]any{
		"worldSessionToken": c.WorldSessionToken,
		"deltaX":            delta.X,
		"deltaY":            delta.Y,
	}, &state, "move", c.ID, stats)
	if err != nil {
		return
	}

	expectedX := previous.X + delta.X
	expectedY := previous.Y + delta.Y
	drift := math.Hypot(state.Position.X-expectedX, state.Position.Y-expectedY)
	if drift > 2.0 {
		stats.recordDesync(c.ID, fmt.Sprintf("position drift %.3f after move", drift))
	}
	c.State = state
}

func (c *loadClient) combatStep(ctx context.Context, httpClient *http.Client, opts options, stats *runStats, step int) {
	if step%8 == 1 || c.State.WorldSessionToken == "" {
		_ = c.fetchState(ctx, httpClient, opts, stats)
	}

	target := c.pickTarget()
	if target.ID == "" {
		return
	}

	if c.State.CurrentTargetID != target.ID {
		var state worldState
		_ = postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/move", map[string]any{
			"worldSessionToken": c.WorldSessionToken,
			"deltaX":            (target.X - 3.0) - c.State.Position.X,
			"deltaY":            (target.Y - 2.0) - c.State.Position.Y,
		}, &state, "move", c.ID, stats)
		if state.WorldSessionToken != "" {
			c.State = state
		}
		if err := postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/target", map[string]any{
			"worldSessionToken": c.WorldSessionToken,
			"targetId":          target.ID,
		}, &state, "target", c.ID, stats); err == nil {
			c.State = state
		}
	}

	if !c.State.AutoAttackActive && c.State.CurrentTargetID != "" {
		var state worldState
		if err := postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/attack/auto", map[string]any{
			"worldSessionToken": c.WorldSessionToken,
			"enabled":           true,
		}, &state, "attack_auto", c.ID, stats); err == nil {
			c.State = state
		}
	}
}

func (c *loadClient) pickTarget() visibleEntity {
	for _, entity := range c.State.Entities {
		if entity.Kind == "hostile_mob" && entity.Alive && entity.Targetable {
			return entity
		}
	}
	return visibleEntity{}
}

func (c *loadClient) fetchState(ctx context.Context, httpClient *http.Client, opts options, stats *runStats) error {
	var state worldState
	_, err := doJSON(ctx, httpClient, http.MethodGet, opts.WorldEndpoint+"/v1/world/state?worldSessionToken="+c.WorldSessionToken, "", nil, &state, "state", c.ID, stats)
	if err != nil {
		return err
	}
	c.State = state
	return nil
}

func (c *loadClient) disconnectReconnect(ctx context.Context, httpClient *http.Client, opts options, stats *runStats) {
	_ = postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/disconnect", map[string]any{
		"worldSessionToken": c.WorldSessionToken,
	}, nil, "disconnect", c.ID, stats)

	time.Sleep(100 * time.Millisecond)

	var state worldState
	if err := postWorld(ctx, httpClient, opts.WorldEndpoint+"/v1/world/reconnect", map[string]any{
		"worldSessionToken": c.WorldSessionToken,
	}, &state, "reconnect", c.ID, stats); err == nil {
		c.State = state
	}
}

func postWorld(ctx context.Context, httpClient *http.Client, url string, payload any, target any, operation string, clientID int, stats *runStats) error {
	_, err := doJSON(ctx, httpClient, http.MethodPost, url, "", payload, target, operation, clientID, stats)
	return err
}

func doJSON(
	ctx context.Context,
	httpClient *http.Client,
	method string,
	url string,
	bearerToken string,
	payload any,
	target any,
	operation string,
	clientID int,
	stats *runStats,
) (int, error) {
	startedAt := time.Now()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			stats.record(clientID, operation, 0, time.Since(startedAt), err)
			return 0, err
		}
		body = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		stats.record(clientID, operation, 0, time.Since(startedAt), err)
		return 0, err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		stats.record(clientID, operation, 0, time.Since(startedAt), err)
		return 0, err
	}
	defer response.Body.Close()

	content, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		stats.record(clientID, operation, response.StatusCode, time.Since(startedAt), readErr)
		return response.StatusCode, readErr
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err = fmt.Errorf("%s", strings.TrimSpace(string(content)))
		stats.record(clientID, operation, response.StatusCode, time.Since(startedAt), err)
		return response.StatusCode, err
	}

	if target != nil && len(content) > 0 {
		if err = json.Unmarshal(content, target); err != nil {
			stats.record(clientID, operation, response.StatusCode, time.Since(startedAt), err)
			return response.StatusCode, err
		}
	}

	stats.record(clientID, operation, response.StatusCode, time.Since(startedAt), nil)
	return response.StatusCode, nil
}

func writeSummary(runDir string, opts options, manifest manifestResponse, stats *runStats, serverMetrics map[string]any) error {
	summary := stats.summary(opts, manifest, serverMetrics)
	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), summaryJSON, 0o644); err != nil {
		return err
	}

	markdown := renderMarkdownSummary(summary)
	return os.WriteFile(filepath.Join(runDir, "summary.md"), []byte(markdown), 0o644)
}

func (s *runStats) summary(opts options, manifest manifestResponse, serverMetrics map[string]any) map[string]any {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	finishedAt := s.finishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}

	operations := map[string]any{}
	for operation, timings := range s.timingsMs {
		operations[operation] = map[string]any{
			"count":        s.counts[operation],
			"errors":       s.errors[operation],
			"errorRate":    rate(s.counts[operation], s.errors[operation]),
			"latencyMs":    percentileSummary(timings),
			"statusCounts": statusCountsForSummary(s.statusCounts[operation]),
		}
	}

	return map[string]any{
		"startedAt":     s.startedAt.Format(time.RFC3339Nano),
		"finishedAt":    finishedAt.Format(time.RFC3339Nano),
		"durationSec":   finishedAt.Sub(s.startedAt).Seconds(),
		"clients":       opts.Clients,
		"scenario":      opts.Scenario,
		"stepInterval":  opts.StepInterval.String(),
		"buildId":       manifest.ID,
		"buildVersion":  manifest.DisplayVersion,
		"buildChannel":  manifest.Channel,
		"operations":    operations,
		"desyncCount":   len(s.desyncs),
		"desyncs":       append([]string(nil), s.desyncs...),
		"serverMetrics": serverMetrics,
	}
}

func renderMarkdownSummary(summary map[string]any) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Load Test Summary\n\n")
	fmt.Fprintf(&builder, "- Scenario: `%v`\n", summary["scenario"])
	fmt.Fprintf(&builder, "- Clients: `%v`\n", summary["clients"])
	fmt.Fprintf(&builder, "- Duration: `%.1fs`\n", summary["durationSec"])
	fmt.Fprintf(&builder, "- Build: `%v`\n", summary["buildId"])
	fmt.Fprintf(&builder, "- Desyncs: `%v`\n\n", summary["desyncCount"])

	operations, _ := summary["operations"].(map[string]any)
	names := make([]string, 0, len(operations))
	for name := range operations {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintf(&builder, "| Operation | Count | Errors | Error Rate | p50 ms | p95 ms | p99 ms | Max ms |\n")
	fmt.Fprintf(&builder, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, name := range names {
		row, _ := operations[name].(map[string]any)
		latency, _ := row["latencyMs"].(map[string]any)
		fmt.Fprintf(
			&builder,
			"| `%s` | %v | %v | %.4f | %.3f | %.3f | %.3f | %.3f |\n",
			name,
			row["count"],
			row["errors"],
			row["errorRate"],
			latency["p50"],
			latency["p95"],
			latency["p99"],
			latency["max"],
		)
	}

	return builder.String()
}

func percentileSummary(values []float64) map[string]any {
	if len(values) == 0 {
		return map[string]any{"p50": 0.0, "p95": 0.0, "p99": 0.0, "max": 0.0, "avg": 0.0}
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	total := 0.0
	for _, value := range sorted {
		total += value
	}

	return map[string]any{
		"p50": percentile(sorted, 0.50),
		"p95": percentile(sorted, 0.95),
		"p99": percentile(sorted, 0.99),
		"max": sorted[len(sorted)-1],
		"avg": total / float64(len(sorted)),
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(p*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func statusCountsForSummary(source map[int]int64) map[string]int64 {
	result := map[string]int64{}
	for status, count := range source {
		result[fmt.Sprint(status)] = count
	}
	return result
}

func rate(count int64, errors int64) float64 {
	if count == 0 {
		return 0
	}
	return float64(errors) / float64(count)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

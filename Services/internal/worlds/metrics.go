package worlds

import (
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"amandacore/services/internal/httpapi"
	"amandacore/services/internal/observability"
)

const (
	worldMetricsSnapshotInterval = 30 * time.Second
	worldSessionStaleAfter       = 5 * time.Minute
)

type worldSessionCounts struct {
	Active       int `json:"active"`
	Connected    int `json:"connected"`
	Disconnected int `json:"disconnected"`
	Mobs         int `json:"mobs"`
}

type durationMetric struct {
	Count           int64
	Errors          int64
	TotalDurationMs float64
	MaxDurationMs   float64
}

type endpointMetric struct {
	durationMetric
	StatusCounts map[int]int64
}

type worldMetrics struct {
	mutex              sync.Mutex
	startedAt          time.Time
	endpoints          map[string]*endpointMetric
	persistence        map[string]*durationMetric
	sessionEvents      map[string]int64
	tick               durationMetric
	lastSnapshotLogAt  time.Time
	staleSessionsTotal int64
}

type statusRecordingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newWorldMetrics() *worldMetrics {
	now := time.Now().UTC()
	return &worldMetrics{
		startedAt:         now,
		endpoints:         map[string]*endpointMetric{},
		persistence:       map[string]*durationMetric{},
		sessionEvents:     map[string]int64{},
		lastSnapshotLogAt: now,
	}
}

func (w *statusRecordingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusRecordingResponseWriter) Write(payload []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(payload)
}

func (s *worldServer) instrumentEndpoint(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecordingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		statusCode := recorder.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		s.metrics.recordEndpoint(name, statusCode, time.Since(startedAt))
	})
}

func (s *worldServer) instrumentEndpointFunc(name string, next http.HandlerFunc) http.HandlerFunc {
	return s.instrumentEndpoint(name, next).ServeHTTP
}

func (s *worldServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	counts := s.sessionCountsLocked()
	s.mutex.Unlock()

	snapshot := s.metrics.snapshot(counts)
	if s.stonewakeLoop != nil {
		snapshot["stonewakeLoop"] = s.stonewakeLoop.Metrics()
	}
	httpapi.WriteJSON(w, http.StatusOK, snapshot)
}

func (s *worldServer) sessionCountsLocked() worldSessionCounts {
	counts := worldSessionCounts{
		Active: len(s.sessionsByToken),
		Mobs:   len(s.mobs),
	}
	for _, session := range s.sessionsByToken {
		if session.Connected {
			counts.Connected++
		} else {
			counts.Disconnected++
		}
	}
	return counts
}

func (s *worldServer) recordPersistenceDuration(operation string, startedAt time.Time, err error) {
	s.metrics.recordPersistence(operation, time.Since(startedAt), err)
}

func (s *worldServer) maybeLogMetricsSnapshotLocked(now time.Time) {
	if !s.metrics.shouldLogSnapshot(now) {
		return
	}

	observability.LogEvent("world-service", "world.metrics_snapshot", s.metrics.snapshot(s.sessionCountsLocked()))
}

func (s *worldServer) touchSessionLocked(session *worldSessionState) {
	if session != nil {
		session.LastSeenAt = time.Now().Unix()
	}
}

func (s *worldServer) cleanupStaleSessionsLocked(now time.Time) {
	cutoff := now.Add(-worldSessionStaleAfter).Unix()
	dropped := 0

	for token, session := range s.sessionsByToken {
		if session == nil || session.LastSeenAt == 0 || session.LastSeenAt > cutoff {
			continue
		}

		s.cancelDuelForCharacterLocked(session.CharacterID, duelReasonDisconnect)
		delete(s.sessionsByToken, token)
		if s.sessionTokenByChar[session.CharacterID] == token {
			delete(s.sessionTokenByChar, session.CharacterID)
		}
		dropped++
		observability.LogEvent("world-service", "world.session_stale_dropped", map[string]any{
			"worldSessionToken": token,
			"accountId":         session.AccountID,
			"characterId":       session.CharacterID,
			"realmId":           session.RealmID,
			"zoneId":            session.ZoneID,
			"connected":         session.Connected,
			"lastSeenAt":        session.LastSeenAt,
			"staleAfterSeconds": worldSessionStaleAfter.Seconds(),
		})
	}

	s.metrics.recordStaleSessionsDropped(dropped)
}

func (m *worldMetrics) recordEndpoint(name string, statusCode int, duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metric := m.endpoints[name]
	if metric == nil {
		metric = &endpointMetric{StatusCounts: map[int]int64{}}
		m.endpoints[name] = metric
	}
	metric.record(duration, statusCode >= 400)
	metric.StatusCounts[statusCode]++
}

func (m *worldMetrics) recordTick(duration time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.tick.record(duration, false)
}

func (m *worldMetrics) recordPersistence(operation string, duration time.Duration, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metric := m.persistence[operation]
	if metric == nil {
		metric = &durationMetric{}
		m.persistence[operation] = metric
	}
	metric.record(duration, err != nil)
}

func (m *worldMetrics) recordSessionEvent(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.sessionEvents[name]++
}

func (m *worldMetrics) recordStaleSessionsDropped(count int) {
	if count <= 0 {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.staleSessionsTotal += int64(count)
	m.sessionEvents["stale_dropped"] += int64(count)
}

func (m *worldMetrics) shouldLogSnapshot(now time.Time) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if now.Sub(m.lastSnapshotLogAt) < worldMetricsSnapshotInterval {
		return false
	}
	m.lastSnapshotLogAt = now
	return true
}

func (m *worldMetrics) snapshot(counts worldSessionCounts) map[string]any {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	return map[string]any{
		"startedAt":             m.startedAt.Format(time.RFC3339Nano),
		"uptimeSeconds":         time.Since(m.startedAt).Seconds(),
		"sessions":              counts,
		"sessionEvents":         cloneCounterMap(m.sessionEvents),
		"staleSessionsDropped":  m.staleSessionsTotal,
		"endpoints":             endpointSnapshots(m.endpoints),
		"worldTick":             durationMetricSnapshot(m.tick),
		"persistence":           durationMetricSnapshots(m.persistence),
		"goroutines":            runtime.NumGoroutine(),
		"memoryAllocBytes":      memory.Alloc,
		"memorySysBytes":        memory.Sys,
		"memoryHeapAllocBytes":  memory.HeapAlloc,
		"memoryHeapObjects":     memory.HeapObjects,
		"memoryTotalAllocBytes": memory.TotalAlloc,
	}
}

func (m *durationMetric) record(duration time.Duration, failed bool) {
	durationMs := float64(duration.Microseconds()) / 1000.0
	m.Count++
	if failed {
		m.Errors++
	}
	m.TotalDurationMs += durationMs
	if durationMs > m.MaxDurationMs {
		m.MaxDurationMs = durationMs
	}
}

func endpointSnapshots(source map[string]*endpointMetric) []map[string]any {
	names := make([]string, 0, len(source))
	for name := range source {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]map[string]any, 0, len(names))
	for _, name := range names {
		metric := source[name]
		snapshot := durationMetricSnapshot(metric.durationMetric)
		snapshot["name"] = name
		snapshot["statusCounts"] = cloneStatusCounts(metric.StatusCounts)
		results = append(results, snapshot)
	}
	return results
}

func durationMetricSnapshots(source map[string]*durationMetric) []map[string]any {
	names := make([]string, 0, len(source))
	for name := range source {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]map[string]any, 0, len(names))
	for _, name := range names {
		snapshot := durationMetricSnapshot(*source[name])
		snapshot["name"] = name
		results = append(results, snapshot)
	}
	return results
}

func durationMetricSnapshot(metric durationMetric) map[string]any {
	averageMs := 0.0
	if metric.Count > 0 {
		averageMs = metric.TotalDurationMs / float64(metric.Count)
	}

	return map[string]any{
		"count":     metric.Count,
		"errors":    metric.Errors,
		"avgMs":     averageMs,
		"maxMs":     metric.MaxDurationMs,
		"totalMs":   metric.TotalDurationMs,
		"errorRate": errorRate(metric.Count, metric.Errors),
	}
}

func cloneCounterMap(source map[string]int64) map[string]int64 {
	cloned := make(map[string]int64, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneStatusCounts(source map[int]int64) map[string]int64 {
	cloned := make(map[string]int64, len(source))
	for key, value := range source {
		cloned[strconv.Itoa(key)] = value
	}
	return cloned
}

func errorRate(count int64, errors int64) float64 {
	if count == 0 {
		return 0
	}
	return float64(errors) / float64(count)
}

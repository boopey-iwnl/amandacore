package worlds

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"amandacore/services/internal/httpapi"
)

const (
	defaultZoneShardCount           = 2
	defaultZoneCommandQueueCapacity = 64
	defaultZoneHandoffRadius        = 18.0
)

type ShardID string

type ShardWorkerState string

const (
	ShardWorkerRegistered  ShardWorkerState = "registered"
	ShardWorkerActive      ShardWorkerState = "active"
	ShardWorkerDraining    ShardWorkerState = "draining"
	ShardWorkerUnavailable ShardWorkerState = "unavailable"
)

type ZoneHandoffStatus string

const (
	ZoneHandoffRequested ZoneHandoffStatus = "requested"
	ZoneHandoffAccepted  ZoneHandoffStatus = "accepted"
	ZoneHandoffCompleted ZoneHandoffStatus = "completed"
	ZoneHandoffRejected  ZoneHandoffStatus = "rejected"
	ZoneHandoffRetried   ZoneHandoffStatus = "retried"
)

type ZoneHandoffRejectionReason string

const (
	ZoneHandoffRejectSessionInvalid              ZoneHandoffRejectionReason = "SessionInvalid"
	ZoneHandoffRejectCharacterDead               ZoneHandoffRejectionReason = "CharacterDead"
	ZoneHandoffRejectInvalidState                ZoneHandoffRejectionReason = "InvalidState"
	ZoneHandoffRejectTransitionMissing           ZoneHandoffRejectionReason = "TransitionMissing"
	ZoneHandoffRejectTransitionDisabled          ZoneHandoffRejectionReason = "TransitionDisabled"
	ZoneHandoffRejectWrongSourceZone             ZoneHandoffRejectionReason = "WrongSourceZone"
	ZoneHandoffRejectOutOfRange                  ZoneHandoffRejectionReason = "OutOfRange"
	ZoneHandoffRejectDestinationZoneMissing      ZoneHandoffRejectionReason = "DestinationZoneMissing"
	ZoneHandoffRejectDestinationEntryMissing     ZoneHandoffRejectionReason = "DestinationEntryMissing"
	ZoneHandoffRejectSourceShardMissing          ZoneHandoffRejectionReason = "SourceShardMissing"
	ZoneHandoffRejectDestinationShardMissing     ZoneHandoffRejectionReason = "DestinationShardMissing"
	ZoneHandoffRejectDestinationShardUnavailable ZoneHandoffRejectionReason = "DestinationShardUnavailable"
	ZoneHandoffRejectDuplicatePendingHandoff     ZoneHandoffRejectionReason = "DuplicatePendingHandoff"
	ZoneHandoffRejectQueueFull                   ZoneHandoffRejectionReason = "QueueFull"
	ZoneHandoffRejectPersistenceFailed           ZoneHandoffRejectionReason = "PersistenceFailed"
)

type ShardAssignmentPolicy struct {
	ShardCount int `json:"shardCount"`
}

type ZoneShardAssignment struct {
	ZoneID  string  `json:"zoneId"`
	ShardID ShardID `json:"shardId"`
	Index   int     `json:"index"`
}

type ShardCoordinatorOptions struct {
	ShardCount    int `json:"shardCount"`
	QueueCapacity int `json:"queueCapacity"`
}

type ZoneCommandQueue struct {
	ZoneID   string  `json:"zoneId"`
	ShardID  ShardID `json:"shardId"`
	Capacity int     `json:"capacity"`
	Depth    int     `json:"depth"`
	MaxDepth int     `json:"maxDepth"`
}

type CharacterZoneOwnership struct {
	CharacterID string  `json:"characterId"`
	ZoneID      string  `json:"zoneId"`
	ShardID     ShardID `json:"shardId"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
	UpdatedAtMs int64   `json:"updatedAtMs"`
}

type ZoneHandoffGateDefinition struct {
	TransitionID       string  `json:"transitionId"`
	FromZoneID         string  `json:"fromZoneId"`
	ToZoneID           string  `json:"toZoneId"`
	GateX              float64 `json:"gateX"`
	GateY              float64 `json:"gateY"`
	Radius             float64 `json:"radius"`
	ArrivalX           float64 `json:"arrivalX"`
	ArrivalY           float64 `json:"arrivalY"`
	ArrivalZ           float64 `json:"arrivalZ"`
	Enabled            bool    `json:"enabled"`
	RetryableWhenFails bool    `json:"retryableWhenFails"`
}

type ZoneHandoffRequest struct {
	HandoffID    string  `json:"handoffId,omitempty"`
	CharacterID  string  `json:"characterId"`
	SessionToken string  `json:"sessionToken,omitempty"`
	TransitionID string  `json:"transitionId"`
	FromZoneID   string  `json:"fromZoneId"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Z            float64 `json:"z"`
	RequestedAt  int64   `json:"requestedAtMs"`
}

type ZoneHandoffDecision struct {
	HandoffID          string                     `json:"handoffId,omitempty"`
	CharacterID        string                     `json:"characterId"`
	TransitionID       string                     `json:"transitionId"`
	FromZoneID         string                     `json:"fromZoneId"`
	ToZoneID           string                     `json:"toZoneId,omitempty"`
	SourceShardID      ShardID                    `json:"sourceShardId,omitempty"`
	DestinationShardID ShardID                    `json:"destinationShardId,omitempty"`
	Status             ZoneHandoffStatus          `json:"status"`
	Accepted           bool                       `json:"accepted"`
	Retryable          bool                       `json:"retryable"`
	Reason             ZoneHandoffRejectionReason `json:"reason,omitempty"`
	Message            string                     `json:"message,omitempty"`
	Arrival            worldPosition              `json:"arrival,omitempty"`
}

type ZoneHandoffRecord struct {
	HandoffID          string
	CharacterID        string
	TransitionID       string
	FromZoneID         string
	ToZoneID           string
	SourceShardID      ShardID
	DestinationShardID ShardID
	Status             ZoneHandoffStatus
	Arrival            worldPosition
	RequestedAtMs      int64
	UpdatedAtMs        int64
}

type ZoneHandoffJournalEntry struct {
	Sequence           int64                      `json:"sequence"`
	HandoffID          string                     `json:"handoffId"`
	CharacterID        string                     `json:"characterId"`
	TransitionID       string                     `json:"transitionId"`
	Status             ZoneHandoffStatus          `json:"status"`
	Reason             ZoneHandoffRejectionReason `json:"reason,omitempty"`
	FromZoneID         string                     `json:"fromZoneId"`
	ToZoneID           string                     `json:"toZoneId,omitempty"`
	SourceShardID      ShardID                    `json:"sourceShardId,omitempty"`
	DestinationShardID ShardID                    `json:"destinationShardId,omitempty"`
	AtMs               int64                      `json:"atMs"`
	Message            string                     `json:"message,omitempty"`
}

type ZoneHandoffJournalWriter interface {
	AppendZoneHandoffJournalEntry(entry ZoneHandoffJournalEntry) error
}

type ShardCoordinator struct {
	assignments        map[string]ZoneShardAssignment
	workers            map[ShardID]ShardWorkerState
	queues             map[string]*ZoneCommandQueue
	ownership          map[string]CharacterZoneOwnership
	pendingByCharacter map[string]string
	handoffs           map[string]ZoneHandoffRecord
	journal            []ZoneHandoffJournalEntry
	nextHandoff        int64
	nextJournal        int64
	journalWriter      ZoneHandoffJournalWriter
}

type zoneHandoffIntentRequest struct {
	WorldSessionToken string `json:"worldSessionToken"`
	TransitionID      string `json:"transitionId"`
}

func defaultZoneHandoffGateDefinitions() map[string]ZoneHandoffGateDefinition {
	gates := map[string]ZoneHandoffGateDefinition{
		"to_brindlebrook": {
			TransitionID:       "to_brindlebrook",
			FromZoneID:         defaultZoneID,
			ToZoneID:           secondZoneID,
			GateX:              470,
			GateY:              260,
			Radius:             defaultZoneHandoffRadius,
			ArrivalX:           secondZoneEntryX,
			ArrivalY:           secondZoneEntryY,
			ArrivalZ:           playableGroundZ,
			Enabled:            true,
			RetryableWhenFails: true,
		},
		"from_stonewake": {
			TransitionID:       "from_stonewake",
			FromZoneID:         secondZoneID,
			ToZoneID:           defaultZoneID,
			GateX:              secondZoneEntryX,
			GateY:              secondZoneEntryY,
			Radius:             defaultZoneHandoffRadius,
			ArrivalX:           470,
			ArrivalY:           260,
			ArrivalZ:           playableGroundZ,
			Enabled:            true,
			RetryableWhenFails: true,
		},
		"to_future_northspur": {
			TransitionID:       "to_future_northspur",
			FromZoneID:         secondZoneID,
			ToZoneID:           "northspur_checkpoint",
			GateX:              700,
			GateY:              360,
			Radius:             defaultZoneHandoffRadius,
			ArrivalX:           12,
			ArrivalY:           12,
			ArrivalZ:           playableGroundZ,
			Enabled:            false,
			RetryableWhenFails: false,
		},
	}
	return gates
}

func BuildZoneShardAssignments(zones map[string]zoneDefinition, policy ShardAssignmentPolicy) (map[string]ZoneShardAssignment, error) {
	if policy.ShardCount <= 0 {
		policy.ShardCount = 1
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("cannot assign shards without loaded zones")
	}
	assignments := map[string]ZoneShardAssignment{}
	for index, zoneID := range sortedZoneIDs(zones) {
		shardIndex := index % policy.ShardCount
		assignments[zoneID] = ZoneShardAssignment{
			ZoneID:  zoneID,
			ShardID: ShardID(fmt.Sprintf("zone_shard_%02d", shardIndex+1)),
			Index:   index,
		}
	}
	return assignments, nil
}

func ResolveZoneShard(assignments map[string]ZoneShardAssignment, zoneID string) (ZoneShardAssignment, error) {
	trimmed := strings.TrimSpace(zoneID)
	if trimmed == "" {
		return ZoneShardAssignment{}, fmt.Errorf("zone id is required")
	}
	assignment, found := assignments[trimmed]
	if !found {
		return ZoneShardAssignment{}, fmt.Errorf("zone %q has no shard assignment", trimmed)
	}
	return assignment, nil
}

func NewShardCoordinator(zones map[string]zoneDefinition, options ShardCoordinatorOptions) (*ShardCoordinator, error) {
	if options.ShardCount <= 0 {
		options.ShardCount = defaultZoneShardCount
	}
	if options.QueueCapacity <= 0 {
		options.QueueCapacity = defaultZoneCommandQueueCapacity
	}
	assignments, err := BuildZoneShardAssignments(zones, ShardAssignmentPolicy{ShardCount: options.ShardCount})
	if err != nil {
		return nil, err
	}
	coordinator := &ShardCoordinator{
		assignments:        assignments,
		workers:            map[ShardID]ShardWorkerState{},
		queues:             map[string]*ZoneCommandQueue{},
		ownership:          map[string]CharacterZoneOwnership{},
		pendingByCharacter: map[string]string{},
		handoffs:           map[string]ZoneHandoffRecord{},
	}
	for _, assignment := range assignments {
		coordinator.workers[assignment.ShardID] = ShardWorkerActive
		coordinator.queues[assignment.ZoneID] = &ZoneCommandQueue{
			ZoneID:   assignment.ZoneID,
			ShardID:  assignment.ShardID,
			Capacity: options.QueueCapacity,
		}
	}
	return coordinator, nil
}

func (c *ShardCoordinator) SetWorkerState(shardID ShardID, state ShardWorkerState) error {
	if c == nil {
		return fmt.Errorf("shard coordinator is not available")
	}
	if state == "" {
		state = ShardWorkerActive
	}
	if _, found := c.workers[shardID]; !found {
		return fmt.Errorf("shard %q is not registered", shardID)
	}
	c.workers[shardID] = state
	return nil
}

func (c *ShardCoordinator) WorkerState(shardID ShardID) ShardWorkerState {
	if c == nil {
		return ShardWorkerUnavailable
	}
	state := c.workers[shardID]
	if state == "" {
		return ShardWorkerUnavailable
	}
	return state
}

func (c *ShardCoordinator) ResolveZone(zoneID string) (ZoneShardAssignment, error) {
	if c == nil {
		return ZoneShardAssignment{}, fmt.Errorf("shard coordinator is not available")
	}
	return ResolveZoneShard(c.assignments, zoneID)
}

func (c *ShardCoordinator) SyncCharacterOwnership(session *worldSessionState) (CharacterZoneOwnership, error) {
	if session == nil || strings.TrimSpace(session.CharacterID) == "" {
		return CharacterZoneOwnership{}, fmt.Errorf("character id is required")
	}
	assignment, err := c.ResolveZone(session.ZoneID)
	if err != nil {
		return CharacterZoneOwnership{}, err
	}
	ownership := CharacterZoneOwnership{
		CharacterID: session.CharacterID,
		ZoneID:      assignment.ZoneID,
		ShardID:     assignment.ShardID,
		X:           session.X,
		Y:           session.Y,
		Z:           session.Z,
		UpdatedAtMs: nowMillis(),
	}
	c.ownership[session.CharacterID] = ownership
	return ownership, nil
}

func (c *ShardCoordinator) CharacterOwnership(characterID string) (CharacterZoneOwnership, bool) {
	if c == nil {
		return CharacterZoneOwnership{}, false
	}
	ownership, found := c.ownership[characterID]
	return ownership, found
}

func (c *ShardCoordinator) ShardAssignmentSummary() map[string]string {
	summary := map[string]string{}
	if c == nil {
		return summary
	}
	for _, zoneID := range sortedAssignmentZoneIDs(c.assignments) {
		summary[zoneID] = string(c.assignments[zoneID].ShardID)
	}
	return summary
}

func (c *ShardCoordinator) ZonePopulation() map[string]int {
	population := map[string]int{}
	if c == nil {
		return population
	}
	for _, ownership := range c.ownership {
		population[ownership.ZoneID]++
	}
	return population
}

func (c *ShardCoordinator) ShardPopulation() map[string]int {
	population := map[string]int{}
	if c == nil {
		return population
	}
	for _, ownership := range c.ownership {
		population[string(ownership.ShardID)]++
	}
	return population
}

func (c *ShardCoordinator) QueueSnapshots() []ZoneCommandQueue {
	if c == nil {
		return []ZoneCommandQueue{}
	}
	zoneIDs := make([]string, 0, len(c.queues))
	for zoneID := range c.queues {
		zoneIDs = append(zoneIDs, zoneID)
	}
	sort.Strings(zoneIDs)
	snapshots := make([]ZoneCommandQueue, 0, len(zoneIDs))
	for _, zoneID := range zoneIDs {
		queue := c.queues[zoneID]
		if queue != nil {
			snapshots = append(snapshots, *queue)
		}
	}
	return snapshots
}

func (c *ShardCoordinator) MaxQueueDepth() int {
	maxDepth := 0
	if c == nil {
		return maxDepth
	}
	for _, queue := range c.queues {
		if queue != nil && queue.MaxDepth > maxDepth {
			maxDepth = queue.MaxDepth
		}
	}
	return maxDepth
}

func (c *ShardCoordinator) Journal() []ZoneHandoffJournalEntry {
	if c == nil || len(c.journal) == 0 {
		return []ZoneHandoffJournalEntry{}
	}
	return append([]ZoneHandoffJournalEntry(nil), c.journal...)
}

func (c *ShardCoordinator) beginHandoff(request ZoneHandoffRequest, gate ZoneHandoffGateDefinition) (ZoneHandoffRecord, ZoneHandoffDecision, error) {
	nowMs := nowMillis()
	if pendingID := c.pendingByCharacter[request.CharacterID]; pendingID != "" {
		return ZoneHandoffRecord{}, c.rejectDecision(request, gate, ZoneHandoffRejectDuplicatePendingHandoff, "character already has a pending zone handoff", true, nowMs), fmt.Errorf("%s", ZoneHandoffRejectDuplicatePendingHandoff)
	}
	source, err := c.ResolveZone(gate.FromZoneID)
	if err != nil {
		return ZoneHandoffRecord{}, c.rejectDecision(request, gate, ZoneHandoffRejectSourceShardMissing, err.Error(), true, nowMs), fmt.Errorf("%s", ZoneHandoffRejectSourceShardMissing)
	}
	destination, err := c.ResolveZone(gate.ToZoneID)
	if err != nil {
		return ZoneHandoffRecord{}, c.rejectDecision(request, gate, ZoneHandoffRejectDestinationShardMissing, err.Error(), true, nowMs), fmt.Errorf("%s", ZoneHandoffRejectDestinationShardMissing)
	}
	if c.WorkerState(destination.ShardID) != ShardWorkerActive {
		return ZoneHandoffRecord{}, c.rejectDecision(request, gate, ZoneHandoffRejectDestinationShardUnavailable, "destination shard is not active", true, nowMs), fmt.Errorf("%s", ZoneHandoffRejectDestinationShardUnavailable)
	}
	queue := c.queues[destination.ZoneID]
	if queue == nil || !queue.tryEnqueue() {
		return ZoneHandoffRecord{}, c.rejectDecision(request, gate, ZoneHandoffRejectQueueFull, "destination zone command queue is full", true, nowMs), fmt.Errorf("%s", ZoneHandoffRejectQueueFull)
	}

	c.nextHandoff++
	handoffID := fmt.Sprintf("handoff_%06d", c.nextHandoff)
	record := ZoneHandoffRecord{
		HandoffID:          handoffID,
		CharacterID:        request.CharacterID,
		TransitionID:       gate.TransitionID,
		FromZoneID:         gate.FromZoneID,
		ToZoneID:           gate.ToZoneID,
		SourceShardID:      source.ShardID,
		DestinationShardID: destination.ShardID,
		Status:             ZoneHandoffAccepted,
		Arrival:            worldPosition{ZoneID: gate.ToZoneID, X: gate.ArrivalX, Y: gate.ArrivalY, Z: gate.ArrivalZ},
		RequestedAtMs:      nowMs,
		UpdatedAtMs:        nowMs,
	}
	c.pendingByCharacter[request.CharacterID] = handoffID
	c.handoffs[handoffID] = record
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          handoffID,
		CharacterID:        request.CharacterID,
		TransitionID:       gate.TransitionID,
		Status:             ZoneHandoffRequested,
		FromZoneID:         gate.FromZoneID,
		ToZoneID:           gate.ToZoneID,
		SourceShardID:      source.ShardID,
		DestinationShardID: destination.ShardID,
		AtMs:               nowMs,
	})
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          handoffID,
		CharacterID:        request.CharacterID,
		TransitionID:       gate.TransitionID,
		Status:             ZoneHandoffAccepted,
		FromZoneID:         gate.FromZoneID,
		ToZoneID:           gate.ToZoneID,
		SourceShardID:      source.ShardID,
		DestinationShardID: destination.ShardID,
		AtMs:               nowMs,
	})
	return record, handoffDecisionFromRecord(record, ZoneHandoffAccepted, "", "", true), nil
}

func (c *ShardCoordinator) completeHandoff(record ZoneHandoffRecord, session *worldSessionState) ZoneHandoffDecision {
	nowMs := nowMillis()
	record.Status = ZoneHandoffCompleted
	record.UpdatedAtMs = nowMs
	c.handoffs[record.HandoffID] = record
	delete(c.pendingByCharacter, record.CharacterID)
	if queue := c.queues[record.ToZoneID]; queue != nil {
		queue.completeOne()
	}
	c.ownership[record.CharacterID] = CharacterZoneOwnership{
		CharacterID: record.CharacterID,
		ZoneID:      record.ToZoneID,
		ShardID:     record.DestinationShardID,
		X:           session.X,
		Y:           session.Y,
		Z:           session.Z,
		UpdatedAtMs: nowMs,
	}
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          record.HandoffID,
		CharacterID:        record.CharacterID,
		TransitionID:       record.TransitionID,
		Status:             ZoneHandoffCompleted,
		FromZoneID:         record.FromZoneID,
		ToZoneID:           record.ToZoneID,
		SourceShardID:      record.SourceShardID,
		DestinationShardID: record.DestinationShardID,
		AtMs:               nowMs,
	})
	return handoffDecisionFromRecord(record, ZoneHandoffCompleted, "", "", true)
}

func (c *ShardCoordinator) failAcceptedHandoff(record ZoneHandoffRecord, reason ZoneHandoffRejectionReason, message string) ZoneHandoffDecision {
	nowMs := nowMillis()
	record.Status = ZoneHandoffRejected
	record.UpdatedAtMs = nowMs
	c.handoffs[record.HandoffID] = record
	delete(c.pendingByCharacter, record.CharacterID)
	if queue := c.queues[record.ToZoneID]; queue != nil {
		queue.completeOne()
	}
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          record.HandoffID,
		CharacterID:        record.CharacterID,
		TransitionID:       record.TransitionID,
		Status:             ZoneHandoffRejected,
		Reason:             reason,
		FromZoneID:         record.FromZoneID,
		ToZoneID:           record.ToZoneID,
		SourceShardID:      record.SourceShardID,
		DestinationShardID: record.DestinationShardID,
		AtMs:               nowMs,
		Message:            message,
	})
	return handoffDecisionFromRecord(record, ZoneHandoffRejected, reason, message, true)
}

func (c *ShardCoordinator) rejectDecision(request ZoneHandoffRequest, gate ZoneHandoffGateDefinition, reason ZoneHandoffRejectionReason, message string, retryable bool, nowMs int64) ZoneHandoffDecision {
	c.nextHandoff++
	handoffID := fmt.Sprintf("handoff_%06d", c.nextHandoff)
	decision := ZoneHandoffDecision{
		HandoffID:    handoffID,
		CharacterID:  request.CharacterID,
		TransitionID: request.TransitionID,
		FromZoneID:   request.FromZoneID,
		ToZoneID:     gate.ToZoneID,
		Status:       ZoneHandoffRejected,
		Retryable:    retryable,
		Reason:       reason,
		Message:      message,
	}
	if source, err := c.ResolveZone(gate.FromZoneID); err == nil {
		decision.SourceShardID = source.ShardID
	}
	if destination, err := c.ResolveZone(gate.ToZoneID); err == nil {
		decision.DestinationShardID = destination.ShardID
	}
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          handoffID,
		CharacterID:        request.CharacterID,
		TransitionID:       request.TransitionID,
		Status:             ZoneHandoffRequested,
		FromZoneID:         request.FromZoneID,
		ToZoneID:           gate.ToZoneID,
		SourceShardID:      decision.SourceShardID,
		DestinationShardID: decision.DestinationShardID,
		AtMs:               nowMs,
	})
	c.appendJournal(ZoneHandoffJournalEntry{
		HandoffID:          handoffID,
		CharacterID:        request.CharacterID,
		TransitionID:       request.TransitionID,
		Status:             ZoneHandoffRejected,
		Reason:             reason,
		FromZoneID:         request.FromZoneID,
		ToZoneID:           gate.ToZoneID,
		SourceShardID:      decision.SourceShardID,
		DestinationShardID: decision.DestinationShardID,
		AtMs:               nowMs,
		Message:            message,
	})
	return decision
}

func (c *ShardCoordinator) appendJournal(entry ZoneHandoffJournalEntry) {
	c.nextJournal++
	entry.Sequence = c.nextJournal
	if entry.AtMs == 0 {
		entry.AtMs = nowMillis()
	}
	c.journal = append(c.journal, entry)
	if c.journalWriter != nil {
		_ = c.journalWriter.AppendZoneHandoffJournalEntry(entry)
	}
}

func (q *ZoneCommandQueue) tryEnqueue() bool {
	if q == nil || q.Capacity <= 0 {
		return false
	}
	if q.Depth >= q.Capacity {
		return false
	}
	q.Depth++
	if q.Depth > q.MaxDepth {
		q.MaxDepth = q.Depth
	}
	return true
}

func (q *ZoneCommandQueue) completeOne() {
	if q == nil || q.Depth <= 0 {
		return
	}
	q.Depth--
}

func (s *worldServer) ensureShardCoordinatorLocked() {
	if s.shardCoordinator != nil {
		return
	}
	coordinator, err := NewShardCoordinator(s.zones, ShardCoordinatorOptions{})
	if err != nil {
		s.emitWorldEventLocked(EventShardCoordinatorRejected, map[string]any{
			"reason": "init_failed",
			"error":  err.Error(),
		})
		return
	}
	s.shardCoordinator = coordinator
}

func (s *worldServer) syncSessionZoneOwnershipLocked(session *worldSessionState) {
	if session == nil {
		return
	}
	s.ensureShardCoordinatorLocked()
	if s.shardCoordinator == nil {
		return
	}
	if _, err := s.shardCoordinator.SyncCharacterOwnership(session); err != nil {
		s.emitWorldEventLocked(EventShardCoordinatorRejected, map[string]any{
			"characterId": session.CharacterID,
			"zoneId":      session.ZoneID,
			"reason":      "ownership_sync_failed",
			"error":       err.Error(),
		})
	}
}

func (s *worldServer) setShardWorkerStateLocked(shardID ShardID, state ShardWorkerState, reason string) error {
	s.ensureShardCoordinatorLocked()
	if s.shardCoordinator == nil {
		return fmt.Errorf("shard coordinator is not available")
	}
	if err := s.shardCoordinator.SetWorkerState(shardID, state); err != nil {
		return err
	}
	s.emitWorldEventLocked(EventShardWorkerStateChanged, map[string]any{
		"shardId": shardID,
		"state":   state,
		"reason":  reason,
	})
	return nil
}

func (s *worldServer) correctSessionFromShardOwnershipLocked(session *worldSessionState) bool {
	if session == nil || s.shardCoordinator == nil {
		return false
	}
	ownership, found := s.shardCoordinator.CharacterOwnership(session.CharacterID)
	if !found || ownership.ZoneID == "" {
		return false
	}
	if session.ZoneID == ownership.ZoneID && session.X == ownership.X && session.Y == ownership.Y && session.Z == ownership.Z {
		return false
	}
	session.ZoneID = ownership.ZoneID
	session.X = ownership.X
	session.Y = ownership.Y
	session.Z = ownership.Z
	s.emitWorldEventLocked(EventZoneHandoffReconnectCorrected, map[string]any{
		"characterId": session.CharacterID,
		"zoneId":      session.ZoneID,
		"shardId":     ownership.ShardID,
		"x":           session.X,
		"y":           session.Y,
		"z":           session.Z,
	})
	return true
}

func (s *worldServer) requestZoneHandoffLocked(session *worldSessionState, transitionID string) (ZoneHandoffDecision, error) {
	s.ensureShardCoordinatorLocked()
	request := ZoneHandoffRequest{TransitionID: strings.TrimSpace(transitionID), RequestedAt: nowMillis()}
	if session != nil {
		request.CharacterID = session.CharacterID
		request.SessionToken = session.Token
		request.FromZoneID = session.ZoneID
		request.X = session.X
		request.Y = session.Y
		request.Z = session.Z
	}
	gate := s.handoffGateLocked(request.TransitionID)
	reject := func(reason ZoneHandoffRejectionReason, message string, retryable bool) (ZoneHandoffDecision, error) {
		decision := s.shardCoordinator.rejectDecision(request, gate, reason, message, retryable, nowMillis())
		s.emitHandoffRejectedLocked(decision)
		return decision, fmt.Errorf("%s", reason)
	}

	if s.shardCoordinator == nil {
		return ZoneHandoffDecision{}, fmt.Errorf("shard coordinator is not available")
	}
	if session == nil || !session.Connected || strings.TrimSpace(session.CharacterID) == "" {
		return reject(ZoneHandoffRejectSessionInvalid, "session is not attached", false)
	}
	if !session.Alive {
		return reject(ZoneHandoffRejectCharacterDead, "dead characters cannot change zones", false)
	}
	if err := s.validateTransportAllowedLocked(session); err != nil {
		return reject(ZoneHandoffRejectInvalidState, err.Error(), true)
	}
	if request.TransitionID == "" {
		return reject(ZoneHandoffRejectTransitionMissing, "transition id is required", false)
	}
	if gate.TransitionID == "" {
		return reject(ZoneHandoffRejectTransitionMissing, "transition is not available", false)
	}
	if !gate.Enabled {
		return reject(ZoneHandoffRejectTransitionDisabled, "transition is not enabled", false)
	}
	if session.ZoneID != gate.FromZoneID {
		return reject(ZoneHandoffRejectWrongSourceZone, "character is not in the transition source zone", false)
	}
	if distance2D(session.X, session.Y, gate.GateX, gate.GateY) > gate.Radius {
		return reject(ZoneHandoffRejectOutOfRange, "move closer to the transition gate", true)
	}
	if _, found := s.zones[gate.ToZoneID]; !found {
		return reject(ZoneHandoffRejectDestinationZoneMissing, "destination zone is not available", true)
	}
	if err := s.validateDestinationPositionLocked(gate.ToZoneID, gate.ArrivalX, gate.ArrivalY); err != nil {
		return reject(ZoneHandoffRejectDestinationEntryMissing, err.Error(), true)
	}

	record, decision, err := s.shardCoordinator.beginHandoff(request, gate)
	if err != nil {
		if decision.Reason == ZoneHandoffRejectQueueFull {
			s.emitWorldEventLocked(EventZoneQueueBackpressure, map[string]any{
				"characterId":  session.CharacterID,
				"transitionId": request.TransitionID,
				"zoneId":       gate.ToZoneID,
				"reason":       decision.Reason,
			})
		}
		s.emitHandoffRejectedLocked(decision)
		return decision, err
	}
	s.emitWorldEventLocked(EventZoneHandoffRequested, handoffEventFields(decision))
	s.emitWorldEventLocked(EventZoneHandoffAccepted, handoffEventFields(decision), zoneHandoffDelta(session.CharacterID, decision))

	previous := worldPosition{ZoneID: session.ZoneID, X: session.X, Y: session.Y, Z: session.Z}
	session.CurrentlyTraveling = true
	defer func() { session.CurrentlyTraveling = false }()
	s.forceDismountLocked(session, "zone_handoff")
	s.resetSessionCombatStateLocked(session, "zone_handoff")
	session.ZoneID = record.Arrival.ZoneID
	session.X = record.Arrival.X
	session.Y = record.Arrival.Y
	session.Z = record.Arrival.Z
	session.LastSeenAt = nowMillis() / 1000
	if s.store != nil {
		if _, err := s.store.UpdateCharacterState(session.CharacterID, session.ZoneID, session.X, session.Y, session.Z); err != nil {
			session.ZoneID = previous.ZoneID
			session.X = previous.X
			session.Y = previous.Y
			session.Z = previous.Z
			decision := s.shardCoordinator.failAcceptedHandoff(record, ZoneHandoffRejectPersistenceFailed, err.Error())
			s.emitHandoffRejectedLocked(decision)
			return decision, fmt.Errorf("%s", ZoneHandoffRejectPersistenceFailed)
		}
	}

	decision = s.shardCoordinator.completeHandoff(record, session)
	s.emitWorldEventLocked(EventZoneHandoffCompleted, handoffEventFields(decision), zoneHandoffDelta(session.CharacterID, decision))
	return decision, nil
}

func (s *worldServer) emitHandoffRejectedLocked(decision ZoneHandoffDecision) {
	s.emitWorldEventLocked(EventZoneHandoffRejected, handoffEventFields(decision), zoneHandoffDelta(decision.CharacterID, decision))
	if decision.Retryable {
		s.emitWorldEventLocked(EventZoneHandoffRetryScheduled, handoffEventFields(decision))
	}
}

func (s *worldServer) handleZoneHandoff(w http.ResponseWriter, r *http.Request) {
	var request zoneHandoffIntentRequest
	if err := httpapi.DecodeJSON(r, &request); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := s.advanceWorldLocked(time.Now()); err != nil {
		httpapi.Error(w, http.StatusInternalServerError, "world_advance_failed", err.Error())
		return
	}

	session, ok := s.sessionsByToken[request.WorldSessionToken]
	if !ok {
		httpapi.Error(w, http.StatusNotFound, "world_session_missing", "World session token was not found.")
		return
	}
	if _, err := s.requestZoneHandoffLocked(session, request.TransitionID); err != nil {
		httpapi.Error(w, http.StatusBadRequest, "zone_handoff_failed", err.Error())
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, s.buildResponse(session))
}

func handoffDecisionFromRecord(record ZoneHandoffRecord, status ZoneHandoffStatus, reason ZoneHandoffRejectionReason, message string, accepted bool) ZoneHandoffDecision {
	return ZoneHandoffDecision{
		HandoffID:          record.HandoffID,
		CharacterID:        record.CharacterID,
		TransitionID:       record.TransitionID,
		FromZoneID:         record.FromZoneID,
		ToZoneID:           record.ToZoneID,
		SourceShardID:      record.SourceShardID,
		DestinationShardID: record.DestinationShardID,
		Status:             status,
		Accepted:           accepted && status != ZoneHandoffRejected,
		Retryable:          reason == ZoneHandoffRejectDestinationShardUnavailable || reason == ZoneHandoffRejectQueueFull || reason == ZoneHandoffRejectOutOfRange || reason == ZoneHandoffRejectInvalidState,
		Reason:             reason,
		Message:            message,
		Arrival:            record.Arrival,
	}
}

func handoffEventFields(decision ZoneHandoffDecision) map[string]any {
	return map[string]any{
		"handoffId":          decision.HandoffID,
		"characterId":        decision.CharacterID,
		"transitionId":       decision.TransitionID,
		"fromZoneId":         decision.FromZoneID,
		"toZoneId":           decision.ToZoneID,
		"sourceShardId":      decision.SourceShardID,
		"destinationShardId": decision.DestinationShardID,
		"status":             decision.Status,
		"accepted":           decision.Accepted,
		"retryable":          decision.Retryable,
		"reason":             decision.Reason,
		"message":            decision.Message,
	}
}

func zoneHandoffDelta(characterID string, decision ZoneHandoffDecision) StateDiff {
	return newStateDiff(diffZoneHandoff, characterID, handoffEventFields(decision))
}

func (s *worldServer) buildZoneHandoffResponseLocked(session *worldSessionState) map[string]any {
	response := map[string]any{
		"availableTransitions": []map[string]any{},
		"currentShardId":       "",
		"zoneShardAssignments": map[string]string{},
		"queues":               []ZoneCommandQueue{},
	}
	if session == nil || s.shardCoordinator == nil {
		return response
	}
	if assignment, err := s.shardCoordinator.ResolveZone(session.ZoneID); err == nil {
		response["currentShardId"] = string(assignment.ShardID)
	}
	response["zoneShardAssignments"] = s.shardCoordinator.ShardAssignmentSummary()
	response["queues"] = s.shardCoordinator.QueueSnapshots()
	transitions := []map[string]any{}
	for _, gate := range s.zoneHandoffGatesForZoneLocked(session.ZoneID) {
		transitions = append(transitions, map[string]any{
			"transitionId": gate.TransitionID,
			"fromZoneId":   gate.FromZoneID,
			"toZoneId":     gate.ToZoneID,
			"enabled":      gate.Enabled,
			"gateX":        gate.GateX,
			"gateY":        gate.GateY,
			"radius":       gate.Radius,
			"arrivalX":     gate.ArrivalX,
			"arrivalY":     gate.ArrivalY,
			"arrivalZ":     gate.ArrivalZ,
		})
	}
	response["availableTransitions"] = transitions
	if ownership, found := s.shardCoordinator.CharacterOwnership(session.CharacterID); found {
		response["ownership"] = ownership
	}
	return response
}

func (s *worldServer) handoffGateLocked(transitionID string) ZoneHandoffGateDefinition {
	if s == nil || s.handoffGates == nil {
		return ZoneHandoffGateDefinition{}
	}
	return s.handoffGates[strings.TrimSpace(transitionID)]
}

func (s *worldServer) zoneHandoffGatesForZoneLocked(zoneID string) []ZoneHandoffGateDefinition {
	gates := []ZoneHandoffGateDefinition{}
	if s == nil {
		return gates
	}
	for _, gate := range s.handoffGates {
		if gate.FromZoneID == zoneID {
			gates = append(gates, gate)
		}
	}
	sort.Slice(gates, func(left, right int) bool {
		return gates[left].TransitionID < gates[right].TransitionID
	})
	return gates
}

func sortedZoneIDs(zones map[string]zoneDefinition) []string {
	zoneIDs := make([]string, 0, len(zones))
	for zoneID := range zones {
		zoneIDs = append(zoneIDs, zoneID)
	}
	sort.Strings(zoneIDs)
	return zoneIDs
}

func sortedAssignmentZoneIDs(assignments map[string]ZoneShardAssignment) []string {
	zoneIDs := make([]string, 0, len(assignments))
	for zoneID := range assignments {
		zoneIDs = append(zoneIDs, zoneID)
	}
	sort.Strings(zoneIDs)
	return zoneIDs
}

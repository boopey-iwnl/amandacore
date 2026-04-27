// Package replication defines AmandaCore-owned snapshot, delta, cursor, and
// convergence contracts for transport-neutral world-state synchronization.
package replication

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const ProtocolVersion = "amandacore.replication.v1"

type FrameKind string

const (
	FrameSnapshot FrameKind = "snapshot"
	FrameDelta    FrameKind = "delta"
	FrameNoop     FrameKind = "noop"
)

type SnapshotReason string

const (
	SnapshotReasonBootstrap SnapshotReason = "bootstrap"
	SnapshotReasonConnect   SnapshotReason = "connect"
	SnapshotReasonReconnect SnapshotReason = "reconnect"
	SnapshotReasonResync    SnapshotReason = "resync"
	SnapshotReasonPoll      SnapshotReason = "poll"
)

type DeltaReason string

const (
	DeltaReasonCommand DeltaReason = "command"
	DeltaReasonPoll    DeltaReason = "poll"
	DeltaReasonNoop    DeltaReason = "noop"
)

var ErrInvalidCursor = errors.New("invalid replication cursor")

type Cursor struct {
	ShardID      string `json:"shardId"`
	ZoneID       string `json:"zoneId"`
	StateVersion uint64 `json:"stateVersion"`
	Sequence     uint64 `json:"sequence"`
	Tick         uint64 `json:"tick"`
}

func (c Cursor) Empty() bool {
	return c.StateVersion == 0 && c.Sequence == 0 && c.Tick == 0 && c.ShardID == "" && c.ZoneID == ""
}

func (c Cursor) Token() string {
	if c.Empty() {
		return ""
	}
	return fmt.Sprintf("%s:%s:%d:%d:%d", c.ShardID, c.ZoneID, c.StateVersion, c.Sequence, c.Tick)
}

func ParseCursor(value string) (Cursor, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Cursor{}, nil
	}
	if version, err := strconv.ParseUint(value, 10, 64); err == nil {
		return Cursor{StateVersion: version}, nil
	}

	parts := strings.Split(value, ":")
	if len(parts) != 5 {
		return Cursor{}, fmt.Errorf("%w: expected shard:zone:stateVersion:sequence:tick", ErrInvalidCursor)
	}

	stateVersion, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: stateVersion must be an unsigned integer", ErrInvalidCursor)
	}
	sequence, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: sequence must be an unsigned integer", ErrInvalidCursor)
	}
	tick, err := strconv.ParseUint(parts[4], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: tick must be an unsigned integer", ErrInvalidCursor)
	}

	return Cursor{
		ShardID:      strings.TrimSpace(parts[0]),
		ZoneID:       strings.TrimSpace(parts[1]),
		StateVersion: stateVersion,
		Sequence:     sequence,
		Tick:         tick,
	}, nil
}

type ClientAck struct {
	WorldSessionToken string `json:"worldSessionToken,omitempty"`
	Cursor            Cursor `json:"cursor"`
	ClientVersion     string `json:"clientVersion,omitempty"`
	RequestedResync   bool   `json:"requestedResync,omitempty"`
}

type EntityVersion struct {
	Domain  string `json:"domain"`
	ID      string `json:"id"`
	Version uint64 `json:"version"`
}

type ChangedFields struct {
	Domain   string   `json:"domain"`
	EntityID string   `json:"entityId"`
	Fields   []string `json:"fields"`
	Version  uint64   `json:"version"`
}

type Snapshot struct {
	ProtocolVersion string          `json:"protocolVersion"`
	SnapshotVersion uint64          `json:"snapshotVersion"`
	Cursor          Cursor          `json:"cursor"`
	Reason          SnapshotReason  `json:"reason"`
	FullSnapshot    bool            `json:"fullSnapshot"`
	EntityVersions  []EntityVersion `json:"entityVersions,omitempty"`
	State           any             `json:"state"`
	Changed         []ChangedFields `json:"changed,omitempty"`
}

type Delta struct {
	ProtocolVersion string          `json:"protocolVersion"`
	FromCursor      Cursor          `json:"fromCursor"`
	ToCursor        Cursor          `json:"toCursor"`
	DeltaVersion    uint64          `json:"deltaVersion"`
	Reason          DeltaReason     `json:"reason"`
	Changed         []ChangedFields `json:"changed"`
	State           any             `json:"state,omitempty"`
}

type Frame struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Kind            FrameKind       `json:"kind"`
	ShardID         string          `json:"shardId"`
	ZoneID          string          `json:"zoneId"`
	Cursor          Cursor          `json:"cursor"`
	SnapshotVersion uint64          `json:"snapshotVersion"`
	DeltaVersion    uint64          `json:"deltaVersion"`
	FullSnapshot    bool            `json:"fullSnapshot"`
	ResyncRequired  bool            `json:"resyncRequired"`
	Changed         []ChangedFields `json:"changed,omitempty"`
	Snapshot        *Snapshot       `json:"snapshot,omitempty"`
	Delta           *Delta          `json:"delta,omitempty"`
	Reason          string          `json:"reason,omitempty"`
}

type ConvergenceReport struct {
	Converged        bool   `json:"converged"`
	PreviousVersion  uint64 `json:"previousVersion"`
	AppliedVersion   uint64 `json:"appliedVersion"`
	ResyncRequired   bool   `json:"resyncRequired"`
	RejectedAsStale  bool   `json:"rejectedAsStale"`
	RejectedAsFuture bool   `json:"rejectedAsFuture"`
	Message          string `json:"message,omitempty"`
}

func NewSnapshotFrame(cursor Cursor, reason SnapshotReason, state any, changed []ChangedFields) Frame {
	changed = NormalizeChangedFields(changed)
	snapshot := Snapshot{
		ProtocolVersion: ProtocolVersion,
		SnapshotVersion: cursor.StateVersion,
		Cursor:          cursor,
		Reason:          reason,
		FullSnapshot:    true,
		State:           state,
		Changed:         changed,
	}
	return Frame{
		ProtocolVersion: ProtocolVersion,
		Kind:            FrameSnapshot,
		ShardID:         cursor.ShardID,
		ZoneID:          cursor.ZoneID,
		Cursor:          cursor,
		SnapshotVersion: cursor.StateVersion,
		DeltaVersion:    cursor.StateVersion,
		FullSnapshot:    true,
		Changed:         changed,
		Snapshot:        &snapshot,
		Reason:          string(reason),
	}
}

func NewDeltaFrame(from Cursor, to Cursor, reason DeltaReason, changed []ChangedFields, state any) Frame {
	changed = NormalizeChangedFields(changed)
	delta := Delta{
		ProtocolVersion: ProtocolVersion,
		FromCursor:      from,
		ToCursor:        to,
		DeltaVersion:    to.StateVersion,
		Reason:          reason,
		Changed:         changed,
		State:           state,
	}
	return Frame{
		ProtocolVersion: ProtocolVersion,
		Kind:            FrameDelta,
		ShardID:         to.ShardID,
		ZoneID:          to.ZoneID,
		Cursor:          to,
		SnapshotVersion: to.StateVersion,
		DeltaVersion:    to.StateVersion,
		FullSnapshot:    false,
		Changed:         changed,
		Delta:           &delta,
		Reason:          string(reason),
	}
}

func NewNoopFrame(cursor Cursor, reason DeltaReason) Frame {
	return Frame{
		ProtocolVersion: ProtocolVersion,
		Kind:            FrameNoop,
		ShardID:         cursor.ShardID,
		ZoneID:          cursor.ZoneID,
		Cursor:          cursor,
		SnapshotVersion: cursor.StateVersion,
		DeltaVersion:    cursor.StateVersion,
		FullSnapshot:    false,
		Reason:          string(reason),
	}
}

func NewResyncFrame(cursor Cursor, reason SnapshotReason, state any, changed []ChangedFields) Frame {
	frame := NewSnapshotFrame(cursor, reason, state, changed)
	frame.ResyncRequired = true
	return frame
}

func NormalizeChangedFields(changes []ChangedFields) []ChangedFields {
	if len(changes) == 0 {
		return nil
	}

	type key struct {
		domain string
		id     string
	}
	grouped := map[key]ChangedFields{}
	for _, change := range changes {
		if change.Domain == "" {
			continue
		}
		k := key{domain: change.Domain, id: change.EntityID}
		current := grouped[k]
		current.Domain = change.Domain
		current.EntityID = change.EntityID
		if change.Version > current.Version {
			current.Version = change.Version
		}
		fieldSet := map[string]bool{}
		for _, field := range current.Fields {
			fieldSet[field] = true
		}
		for _, field := range change.Fields {
			field = strings.TrimSpace(field)
			if field != "" {
				fieldSet[field] = true
			}
		}
		current.Fields = current.Fields[:0]
		for field := range fieldSet {
			current.Fields = append(current.Fields, field)
		}
		sort.Strings(current.Fields)
		grouped[k] = current
	}

	normalized := make([]ChangedFields, 0, len(grouped))
	for _, change := range grouped {
		normalized = append(normalized, change)
	}
	sort.Slice(normalized, func(left, right int) bool {
		if normalized[left].Domain != normalized[right].Domain {
			return normalized[left].Domain < normalized[right].Domain
		}
		return normalized[left].EntityID < normalized[right].EntityID
	})
	return normalized
}

func AcceptFrame(current Cursor, frame Frame) (Cursor, ConvergenceReport) {
	report := ConvergenceReport{
		PreviousVersion: current.StateVersion,
		AppliedVersion:  frame.Cursor.StateVersion,
		ResyncRequired:  frame.ResyncRequired,
	}
	if frame.ResyncRequired {
		report.Message = "full resync required"
		return current, report
	}
	if frame.Cursor.StateVersion < current.StateVersion {
		report.RejectedAsStale = true
		report.Message = "frame version is older than current client state"
		return current, report
	}
	if frame.Kind == FrameDelta && frame.Delta != nil && frame.Delta.FromCursor.StateVersion > current.StateVersion {
		report.RejectedAsFuture = true
		report.Message = "delta starts after current client state"
		return current, report
	}
	report.Converged = true
	return frame.Cursor, report
}

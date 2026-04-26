package sqlstore

import (
	"sort"

	"amandacore/services/internal/platform"
	filestore "amandacore/services/internal/store"
)

func (s *Store) RecordAuditEvent(event platform.AuditEvent) (platform.AuditEvent, error) {
	if event.ID == "" {
		event.ID = randomID("audit")
	}
	if event.Timestamp == 0 {
		event.Timestamp = s.now().Unix()
	}
	beforeJSON, err := encodeJSON(event.BeforeSummary)
	if err != nil {
		return platform.AuditEvent{}, err
	}
	afterJSON, err := encodeJSON(event.AfterSummary)
	if err != nil {
		return platform.AuditEvent{}, err
	}
	metadataJSON, err := encodeJSON(event.Metadata)
	if err != nil {
		return platform.AuditEvent{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO ac_audit_events (
			audit_event_id, timestamp, action, actor_account_id, actor_character_id, target_account_id,
			target_character_id, reason, before_summary_json, after_summary_json, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Timestamp,
		event.Action,
		event.ActorAccountID,
		event.ActorCharacterID,
		event.TargetAccountID,
		event.TargetCharacterID,
		event.Reason,
		beforeJSON,
		afterJSON,
		metadataJSON)
	return event, err
}

func (s *Store) QueryAuditEvents(query filestore.AuditQuery) ([]platform.AuditEvent, error) {
	rows, err := s.db.Query(
		`SELECT audit_event_id, timestamp, action, actor_account_id, actor_character_id, target_account_id,
			target_character_id, reason, before_summary_json, after_summary_json, metadata_json
		FROM ac_audit_events`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []platform.AuditEvent
	for rows.Next() {
		var event platform.AuditEvent
		var beforeJSON, afterJSON, metadataJSON string
		if err := rows.Scan(
			&event.ID,
			&event.Timestamp,
			&event.Action,
			&event.ActorAccountID,
			&event.ActorCharacterID,
			&event.TargetAccountID,
			&event.TargetCharacterID,
			&event.Reason,
			&beforeJSON,
			&afterJSON,
			&metadataJSON); err != nil {
			return nil, err
		}
		if err := decodeJSON(beforeJSON, &event.BeforeSummary); err != nil {
			return nil, err
		}
		if err := decodeJSON(afterJSON, &event.AfterSummary); err != nil {
			return nil, err
		}
		if err := decodeJSON(metadataJSON, &event.Metadata); err != nil {
			return nil, err
		}
		if query.ActorAccountID != "" && event.ActorAccountID != query.ActorAccountID {
			continue
		}
		if query.TargetAccountID != "" && event.TargetAccountID != query.TargetAccountID {
			continue
		}
		if query.TargetCharacterID != "" && event.TargetCharacterID != query.TargetCharacterID {
			continue
		}
		if query.Action != "" && event.Action != query.Action {
			continue
		}
		if query.From > 0 && event.Timestamp < query.From {
			continue
		}
		if query.To > 0 && event.Timestamp > query.To {
			continue
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(events, func(left int, right int) bool {
		return events[left].Timestamp > events[right].Timestamp
	})
	if query.Limit <= 0 || query.Limit > 200 {
		query.Limit = 100
	}
	if len(events) > query.Limit {
		events = events[:query.Limit]
	}
	return events, nil
}

CREATE TABLE IF NOT EXISTS ac_domain_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    aggregate_id TEXT NOT NULL DEFAULT '',
    aggregate_kind TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    occurred_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ac_domain_events_type_time ON ac_domain_events(event_type, occurred_at);

CREATE TABLE IF NOT EXISTS ac_audit_events (
    audit_event_id TEXT PRIMARY KEY,
    timestamp INTEGER NOT NULL,
    action TEXT NOT NULL,
    actor_account_id TEXT NOT NULL,
    actor_character_id TEXT NOT NULL DEFAULT '',
    target_account_id TEXT NOT NULL DEFAULT '',
    target_character_id TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    before_summary_json TEXT NOT NULL,
    after_summary_json TEXT NOT NULL,
    metadata_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ac_audit_events_actor ON ac_audit_events(actor_account_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_ac_audit_events_target_account ON ac_audit_events(target_account_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_ac_audit_events_target_character ON ac_audit_events(target_character_id, timestamp);

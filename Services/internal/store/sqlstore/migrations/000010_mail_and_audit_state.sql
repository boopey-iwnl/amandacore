CREATE TABLE IF NOT EXISTS ac_mail_messages (
    mail_id TEXT PRIMARY KEY,
    auction_id TEXT NOT NULL DEFAULT '',
    sender_character_id TEXT NOT NULL DEFAULT '',
    sender_display_name TEXT NOT NULL,
    recipient_character_id TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    delivered_at INTEGER NOT NULL DEFAULT 0,
    read_at INTEGER NOT NULL DEFAULT 0,
    deleted_at INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (recipient_character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_mail_messages_recipient
    ON ac_mail_messages(recipient_character_id, created_at);

CREATE TABLE IF NOT EXISTS ac_mail_attachments (
    attachment_id TEXT PRIMARY KEY,
    mail_id TEXT NOT NULL,
    attachment_index INTEGER NOT NULL,
    attachment_kind TEXT NOT NULL,
    item_id TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    stack_count INTEGER NOT NULL DEFAULT 0,
    currency_copper INTEGER NOT NULL DEFAULT 0,
    claimed_at INTEGER NOT NULL DEFAULT 0,
    claimed_by_character_id TEXT NOT NULL DEFAULT '',
    claim_mutation_key TEXT NOT NULL DEFAULT '',
    UNIQUE (mail_id, attachment_index),
    FOREIGN KEY (mail_id) REFERENCES ac_mail_messages(mail_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ac_mail_attachments_claim_key
    ON ac_mail_attachments(mail_id, claim_mutation_key)
    WHERE claim_mutation_key <> '';

CREATE TABLE IF NOT EXISTS ac_transfer_audit_events (
    event_id TEXT PRIMARY KEY,
    operation TEXT NOT NULL,
    actor_character_id TEXT NOT NULL DEFAULT '',
    source_character_id TEXT NOT NULL DEFAULT '',
    target_character_id TEXT NOT NULL DEFAULT '',
    target_kind TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    item_id TEXT NOT NULL DEFAULT '',
    stack_count INTEGER NOT NULL DEFAULT 0,
    currency_delta INTEGER NOT NULL DEFAULT 0,
    mutation_key TEXT NOT NULL DEFAULT '',
    result_status TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ac_transfer_audit_events_actor
    ON ac_transfer_audit_events(actor_character_id, created_at);

CREATE INDEX IF NOT EXISTS idx_ac_transfer_audit_events_target
    ON ac_transfer_audit_events(target_character_id, created_at);

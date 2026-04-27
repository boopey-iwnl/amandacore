ALTER TABLE ac_characters ADD COLUMN state_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE ac_character_inventory ADD COLUMN slot_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE ac_character_inventory ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;

ALTER TABLE ac_character_quests ADD COLUMN progress_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE ac_character_quests ADD COLUMN updated_at_row INTEGER NOT NULL DEFAULT 0;

ALTER TABLE ac_learned_abilities ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;

ALTER TABLE ac_action_bar_slots ADD COLUMN slot_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE ac_action_bar_slots ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;

ALTER TABLE ac_currency_ledger ADD COLUMN mutation_key TEXT NOT NULL DEFAULT '';

ALTER TABLE ac_world_sessions ADD COLUMN session_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE ac_character_position_snapshots ADD COLUMN world_session_token TEXT NOT NULL DEFAULT '';
ALTER TABLE ac_character_position_snapshots ADD COLUMN character_version INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS ac_character_state_mutations (
    mutation_id TEXT PRIMARY KEY,
    character_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    mutation_key TEXT NOT NULL,
    response_json TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (character_id, operation, mutation_key),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ac_character_state_mutations_character
    ON ac_character_state_mutations(character_id, created_at);

CREATE TABLE IF NOT EXISTS ac_character_inventory (
    character_id TEXT NOT NULL,
    slot_index INTEGER NOT NULL,
    item_id TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    stack_count INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, slot_index),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_character_quests (
    character_id TEXT NOT NULL,
    quest_id TEXT NOT NULL,
    state TEXT NOT NULL,
    current_count INTEGER NOT NULL,
    target_count INTEGER NOT NULL,
    accepted_at INTEGER NOT NULL DEFAULT 0,
    completed_at INTEGER NOT NULL DEFAULT 0,
    reward_granted_at INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL DEFAULT 0,
    objective_progress_json TEXT NOT NULL,
    PRIMARY KEY (character_id, quest_id),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_learned_abilities (
    character_id TEXT NOT NULL,
    ability_id TEXT NOT NULL,
    learned_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (character_id, ability_id),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_action_bar_slots (
    character_id TEXT NOT NULL,
    slot_index INTEGER NOT NULL,
    ability_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (character_id, slot_index),
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS ac_currency_ledger (
    entry_id TEXT PRIMARY KEY,
    character_id TEXT NOT NULL,
    delta_copper INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    reason TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (character_id) REFERENCES ac_characters(id) ON DELETE CASCADE
);

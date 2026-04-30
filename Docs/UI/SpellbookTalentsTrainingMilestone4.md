# UI Milestone 4 - Spellbook, Talents, Training, and Professions

Milestone 4 moves the ability interface toward a first-party MMORPG shell without adding addon support. The implementation keeps world and character state server-authoritative and uses additive payload fields so older clients can continue to ignore new metadata.

## Audit Summary

Current ability state is emitted from the world session payload through `spellbook`, `learnedAbilityIds`, `actionBar`, `trainer`, `talents`, `professions`, and `professionTrainer`. The O3DE NetClient parses those payloads into `WorldSessionResponse`, GameCore applies the server response, and UiClient renders the panels.

The adopted prework added ability metadata for category, active/passive type, trainability, and action-bar assignability. This milestone completed the missing client plumbing for profession learning, corrected player-facing class labels to `Warrior`, and kept internal compatibility IDs such as `wayfarer_warden` where they are still required by existing storage and content references.

## Implemented Behavior

- Spellbook tabs: All, Active, Passive, Utility, Warrior.
- Spellbook rows show learned, unlearned, active, passive, trainable, and disabled states from server metadata.
- Ability tooltips centralize name, active/passive type, category, cost/generation, cooldown, range, description, trainer requirement, and assignment guidance.
- Learned active abilities can be dragged from Spellbook to the action bar.
- Passive or unlearned abilities do not create drag payloads; server assignment also rejects passive abilities.
- Action-bar click, keybind activation, drag move, and edit-mode clear remain server-backed.
- Trainer UI calls Learn only when the server marks an offer `canLearn`.
- Talents remain server-backed through `/v1/world/talent/select`; no fake point spending is introduced.
- Professions panel is learn/view only: known professions, catalog, trainer offers, and read-only recipe summaries.

## Deferred Work

- No client-side profession crafting actions are exposed in M4.
- No new talent trees, ranks, formulas, icons, or copied external MMO data are introduced.
- No new texture imports were required. Existing procedural icons and repo-side icon identifiers remain the default path.

## Safety Notes

- No addon API, Lua loading, AddOns folder, plugin runtime, user-installed UI module, or arbitrary UI script execution is part of this milestone.
- The external local texture source remains read-only. Runtime, manifests, materials, and package scripts must reference only repo-controlled assets.

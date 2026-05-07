# Gossip And Dialogue Contract

The gossip/dialogue panel is a first-party NPC interaction panel for quest and service context.

## Data Source

- NPC identity comes from visible entity payloads and NPC service metadata.
- Quest options come from `quests`, `quest`, current target, giver NPC IDs, turn-in NPC IDs, objective state, and reward data.
- Dialogue text must come from AmandaCore repo content only. The current M5 panel uses quest objective and state text while the content-backed dialogue tree remains a later expansion.

## Quest Actions

- Available quest: show title, objective, real rewards, and an Accept button that calls `POST /v1/world/quest/accept`.
- Active quest: show current progress and allow completion only when the current quest state and target make it valid; completion calls `POST /v1/world/quest/complete`.
- Ready-to-turn-in quest: show reward preview and a Complete Quest button that calls `POST /v1/world/quest/complete`.
- Completed/reward-granted quest: show persisted completion state and do not offer another claim.

## Service Interaction

- Trainer, profession trainer, vendor, auction, dungeon entrance, and dungeon exit shortcuts keep their existing first-party flows.
- Walking away or changing targets closes the active NPC panel through existing interaction cleanup.
- Failed server requests must surface the returned error through existing client error state.

No copied external gossip text, quest text, UI scripts, addon modules, or Lua execution is allowed.

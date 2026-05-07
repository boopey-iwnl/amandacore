# Quest Log Contract

The Quest Log is a built-in AmandaCore gameplay panel. It consumes server-authoritative quest state from the world session payload and never mutates quest truth locally.

## Data Source

- Primary payload fields: `quest`, `quests`, and `trackedQuestIds`.
- Quest summary fields used by UI: `id`, `title`, `category`, `statusBucket`, `levelBand`, `objectiveType`, `objectiveText`, `objectiveGraph`, `state`, `currentCount`, `targetCount`, `giverNpcId`, `turnInNpcId`, `rewardXp`, `rewardCurrency`, `rewardItems`, `objectiveArea`, `partyShareable`, `groupRecommended`, `recommendedPlayers`, `partyNearbyCount`, `partyEligibleCount`, `partyStatusText`, and `tracked`.
- Missing optional fields must render as clear unavailable states, not fabricated data.

## Panel Rules

- The left pane lists quests by status bucket: active, ready to turn in, available, and completed.
- The right pane shows the selected quest state, objectives, objective graph progress when present, area hints, party hints, and reward preview.
- Track and untrack controls call `POST /v1/world/quest/track` only for accepted or ready-to-turn-in quests.
- Accept and complete actions are not owned by the Quest Log unless a future server contract explicitly supports safe log-side actions.
- Abandon is hidden until a server-authoritative abandon path exists.

## Compatibility

All quest fields are optional/additive for older clients. Existing clients that ignore `rewardItems`, `objectiveGraph`, or `levelBand` remain compatible.

No addon integration is allowed.

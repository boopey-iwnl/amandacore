# Runtime Contract

Milestone 1 contract-freeze inventory lives in `Docs/Contracts/Milestone1-ContractInventory.md`. The registered HTTP route manifest lives in `Docs/Contracts/http-api-v1.json` and is checked by `Services/internal/contracts` tests.

This is the current playable single-zone contract used by the launcher, the O3DE client/runtime, and the backend services. It stays intentionally limited to the single-realm local slice:

- realm: `sunset-frontier-dev`
- zone: `sunset_frontier`
- spawn cell: `west_approach`
- player archetype: `wayfarer_warden`

The account-to-world HTTP flow remains request/response only. World connect consumes a short-lived join ticket exactly once.

## Account and auth

### `POST /v1/accounts/register`

Request:

```json
{
  "username": "player_one",
  "password": "secret"
}
```

Response:

```json
{
  "accountId": "acct_...",
  "username": "player_one"
}
```

### `POST /v1/auth/login`

Request:

```json
{
  "username": "player_one",
  "password": "secret"
}
```

Response:

```json
{
  "accessToken": "token",
  "refreshToken": "token",
  "accountId": "acct_...",
  "roles": ["player"]
}
```

## Realm and character

### `GET /v1/realms`

Response:

```json
{
  "realms": [
    {
      "id": "sunset-frontier-dev",
      "displayName": "Sunset Frontier Dev",
      "region": "local",
      "endpoint": "http://127.0.0.1:8085",
      "supportedBuild": "amandacore-local-0.2.0",
      "onlinePlayers": 0,
      "online": true
    }
  ]
}
```

### `GET /v1/characters?realmId=<realm>`

Requires `Authorization: Bearer <accessToken>`.

### `POST /v1/characters`

Requires `Authorization: Bearer <accessToken>`.

Request:

```json
{
  "realmId": "sunset-frontier-dev",
  "displayName": "Runner",
  "archetypeId": "wayfarer_warden"
}
```

## World join and session

### `POST /v1/world/join-ticket`

Requires `Authorization: Bearer <accessToken>`.

Request:

```json
{
  "realmId": "sunset-frontier-dev",
  "characterId": "char_..."
}
```

Response:

```json
{
  "ticketId": "ticket_...",
  "sessionId": "sess_...",
  "accountId": "acct_...",
  "characterId": "char_...",
  "realmId": "sunset-frontier-dev",
  "worldEndpoint": "http://127.0.0.1:8085",
  "expiresAt": 1776499999,
  "consumedAt": 0
}
```

### `POST /v1/world/connect`

Consumes the join ticket. Reusing the same ticket must fail with `401 invalid_ticket`.

Request:

```json
{
  "ticketId": "ticket_..."
}
```

Response:

```json
{
  "worldSessionToken": "world_...",
  "characterId": "char_...",
  "realmId": "sunset-frontier-dev",
  "zoneId": "west_approach",
  "displayName": "Runner",
  "position": { "x": 12, "y": 12, "z": 0 },
  "entities": [
    {
      "id": "mob_ember_hound_01",
      "displayName": "Ember Hound",
      "kind": "hostile_mob",
      "x": 18,
      "y": 24,
      "z": 0
    },
    {
      "id": "mob_ember_hound_02",
      "displayName": "Ember Hound",
      "kind": "hostile_mob",
      "x": 26,
      "y": 28,
      "z": 0
    },
    {
      "id": "mob_ember_hound_03",
      "displayName": "Ember Hound",
      "kind": "hostile_mob",
      "x": 34,
      "y": 24,
      "z": 0
    }
  ]
}
```

### `POST /v1/world/move`

Request:

```json
{
  "worldSessionToken": "world_...",
  "deltaX": 1,
  "deltaY": 0
}
```

### `POST /v1/world/disconnect`

Request:

```json
{
  "worldSessionToken": "world_..."
}
```

### `POST /v1/world/reconnect`

Request:

```json
{
  "worldSessionToken": "world_..."
}
```

### `GET /v1/world/state?worldSessionToken=<token>`

Returns the same shape as `/v1/world/connect`.

## Fixed behavior for milestone `0.1`

- Character selection is bound to the authenticated account through `POST /v1/world/join-ticket`.
- The world service validates that the requested character belongs to the authenticated account and the selected realm.
- First connect spawns the character at its persisted position. A brand-new character begins at the fixed `west_approach` spawn point `(12, 12, 0)`.
- `POST /v1/world/reconnect` restores the current in-process world session after disconnect.
- Full service restart recovery uses the normal login -> join-ticket -> connect flow and restores persisted position from character storage.

## Additive Milestone `0.2` O3DE client notes

- The launcher must prefer the O3DE client executable `amandacore.GameLauncher.exe` when the O3DE build is present.
- The fallback `.NET` world client remains diagnostic only and is not the supported playable-slice path.
- The launcher continues to pass the same launch arguments:
  - `--join-ticket <ticketId>`
  - `--world-endpoint <endpoint>`
- `TestZone01` is the Milestone `0.2` client level mapped to `sunset_frontier / west_approach`.
- The O3DE client keeps the same HTTP world flow:
  - startup logs `client.world_connect_started`
  - level load completion logs `client.level_ready`
  - successful world connect logs `client.world_connected`
  - authoritative spawn logs `client.player_spawned`
- Runtime validation expects the ordering:
  - `client.world_connect_started`
  - `client.level_ready`
  - `client.world_connected`
- Player spawn must occur only after successful world connect/bootstrap validation.

## Additive Milestone `0.3` combat slice notes

- `TestZone01` now includes one hostile mob archetype:
  - `id`: `mob_ember_hound_01`
  - `displayName`: `Ember Hound`
  - `kind`: `hostile_mob`
- Combat remains fully server-authoritative through the Go world service.
- The O3DE client sends intent only:
  - `Tab` or `LMB` target request
  - `F` auto-attack toggle
  - `1` instant ability `ember_bolt`
  - `2` cast-time ability `steady_blast`
- The verification HUD is intentionally minimal and shows:
  - player health
  - player resource
  - current target name
  - current target health
  - cast progress only while the cast-time ability is active
- Combat teardown is authoritative and immediate:
  - mob death clears target and stops auto-attack
  - leash/reset clears target and stops auto-attack
  - reconnect or a fresh connect that resumes an existing dead session revives the player to full health/resource and clears combat state

### `POST /v1/world/target`

Request:

```json
{
  "worldSessionToken": "world_...",
  "targetId": "mob_ember_hound_01"
}
```

### `POST /v1/world/attack/auto`

Request:

```json
{
  "worldSessionToken": "world_...",
  "enabled": true
}
```

### `POST /v1/world/attack/ability`

Request:

```json
{
  "worldSessionToken": "world_...",
  "abilityId": "ember_bolt"
}
```

### Expanded world session/state response

`POST /v1/world/connect`, `POST /v1/world/move`, `POST /v1/world/reconnect`, and `GET /v1/world/state` now return additive combat fields:

```json
{
  "worldSessionToken": "world_...",
  "characterId": "char_...",
  "realmId": "sunset-frontier-dev",
  "zoneId": "west_approach",
  "displayName": "Runner",
  "position": { "x": 12, "y": 12, "z": 0 },
  "health": 100,
  "maxHealth": 100,
  "resource": 100,
  "maxResource": 100,
  "alive": true,
  "currentTargetId": "",
  "autoAttackActive": false,
  "globalCooldownEndsAt": 0,
  "castEndsAt": 0,
  "castingAbilityId": "",
  "entities": [
    {
      "id": "mob_ember_hound_01",
      "displayName": "Ember Hound",
      "kind": "hostile_mob",
      "x": 29,
      "y": 22,
      "z": 0,
      "health": 90,
      "maxHealth": 90,
      "alive": true,
      "targetable": true,
      "aiState": "idle"
    }
  ]
}
```

### Required structured log events for milestone `0.3`

Client:

- `client.target_selected`
- `client.target_cleared`
- `client.auto_attack_started`
- `client.auto_attack_stopped`
- `client.ability_requested`
- `client.authoritative_combat_state_applied`
- `client.mob_proxy_spawned`
- `client.mob_death_observed`
- `client.mob_respawn_observed`

Server:

- `world.target_validated`
- `world.target_rejected`
- `world.target_cleared`
- `world.auto_attack_started`
- `world.auto_attack_stopped`
- `world.ability_requested`
- `world.damage_applied`
- `world.mob_aggroed`
- `world.mob_died`
- `world.mob_respawned`

## Additive Milestone `0.4` quest and reward slice notes

- `TestZone01` now exposes one server-authoritative quest:
  - `id`: `defeat_ember_hounds_01`
  - `title`: `Contain the Ember Hounds`
  - `objectiveText`: `Defeat 3 Ember Hounds`
- Quest truth and reward issuance remain in the Go world service.
- The O3DE client requests quest accept and renders authoritative quest state only.
- The single reward for milestone `0.4` is persisted `experience`.
- Milestone `0.4` also adds a minimal persisted currency wallet stored as total copper and rendered as gold / silver / copper.
- Quest progress increments only from authoritative Ember Hound kill events in the Go world service.
- Reward is granted automatically on completion; there is no turn-in step in this milestone.
- Quest accept is valid only while the player is within the frontier command-post radius around `(12, 12)` in `west_approach`.

## Additive UI Milestone 3 character panel notes

- World session responses include `archetypeId` for in-world character identity display.
- `inventory.slots[]` and `equipment.slots[]` expose optional item metadata for built-in Character panel tooltips.
- Item metadata prefers AmandaCore terminology such as `requiredArchetype`; legacy internal class checks remain server-side compatibility details.
- Equipment mutations remain server-authoritative.
- `POST /v1/world/inventory/equip` equips a single item from an inventory slot after validating compatibility.
- `POST /v1/world/inventory/unequip` unequips an occupied equipment slot into the first empty inventory slot and rejects full bags.
- Reputation UI remains an empty/read-only shell unless real runtime faction standings are added.

### `POST /v1/world/quest/accept`

Request:

```json
{
  "worldSessionToken": "world_...",
  "questId": "defeat_ember_hounds_01"
}
```

### Expanded world session/state response

`POST /v1/world/connect`, `POST /v1/world/move`, `POST /v1/world/reconnect`, `POST /v1/world/quest/accept`, `POST /v1/world/attack/auto`, `POST /v1/world/attack/ability`, and `GET /v1/world/state` now include additive quest/reward fields:

```json
{
  "worldSessionToken": "world_...",
  "characterId": "char_...",
  "realmId": "sunset-frontier-dev",
  "zoneId": "west_approach",
  "displayName": "Runner",
  "position": { "x": 12, "y": 12, "z": 0 },
  "health": 100,
  "maxHealth": 100,
  "resource": 100,
  "maxResource": 100,
  "experience": 25,
  "currencyCopper": 125,
  "currency": {
    "gold": 0,
    "silver": 1,
    "copper": 25
  },
  "alive": true,
  "quest": {
    "id": "defeat_ember_hounds_01",
    "title": "Contain the Ember Hounds",
    "objectiveType": "kill_hostile_mob",
    "objectiveText": "Defeat 3 Ember Hounds",
    "state": "reward_granted",
    "currentCount": 3,
    "targetCount": 3,
    "rewardXp": 25,
    "rewardCurrencyCopper": 125,
    "rewardCurrency": {
      "gold": 0,
      "silver": 1,
      "copper": 25
    }
  },
  "currentTargetId": "",
  "autoAttackActive": false,
  "globalCooldownEndsAt": 0,
  "castEndsAt": 0,
  "castingAbilityId": "",
  "entities": [
    {
      "id": "mob_ember_hound_01",
      "displayName": "Ember Hound",
      "kind": "hostile_mob",
      "x": 29,
      "y": 22,
      "z": 0,
      "health": 90,
      "maxHealth": 90,
      "alive": true,
      "targetable": true,
      "aiState": "idle"
    }
  ]
}
```

### Required structured log events for milestone `0.4`

Client:

- `client.quest_state_applied`

Server:

- `world.quest_accepted`
- `world.quest_progressed`
- `world.quest_completed`
- `world.quest_reward_granted`

## Required structured log events for milestone `0.1`

- `account.registered`
- `auth.session_issued`
- `character.created`
- `character.selected`
- `world.join_ticket_issued`
- `world.join_ticket_consumed`
- `world.player_spawned`
- `world.character_saved`
- `world.reconnected`

## Additive Milestone `0.5` multi-mob encounter validation notes

- `TestZone01` now includes a fixed set of three hostile Ember Hound instances:
  - `mob_ember_hound_01`
  - `mob_ember_hound_02`
  - `mob_ember_hound_03`
- All three hostile mob instances share the same archetype/type and all count toward the existing quest `defeat_ember_hounds_01`.
- The world response continues to use the existing `entities` array and now returns all hostile instances in the same zone.
- Targeting remains server-authoritative:
  - `Tab` cycles valid hostile mob ids deterministically on the client, then submits the chosen `targetId`
  - `LMB` selects the intended hostile under the cursor and submits that `targetId`
  - the server validates that the chosen mob instance exists, is alive, is targetable, and is in range
- Combat remains server-authoritative per hostile mob instance:
  - auto-attack, cast completion, aggro, leash, death, and respawn all resolve against the selected mob id
  - death and leash only clear the affected target instance
- Reconnect continues to clear transient combat state while preserving persisted quest, XP, and currency truth.

### Expanded hostile entity response shape

Hostile entities now include an additive `mobTypeId` field:

```json
{
  "id": "mob_ember_hound_02",
  "displayName": "Ember Hound",
  "kind": "hostile_mob",
  "mobTypeId": "ember_hound",
  "x": 50,
  "y": 30,
  "z": 0,
  "health": 90,
  "maxHealth": 90,
  "alive": true,
  "targetable": true,
  "aiState": "idle"
}
```

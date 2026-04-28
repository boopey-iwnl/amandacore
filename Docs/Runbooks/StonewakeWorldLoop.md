# Stonewake World Loop Runbook

## Scope

This runbook covers the Milestone 4 single-writer Stonewake loop for the current local playable slice. It is for development and validation, not production cutover.

## How The Loop Starts

The world service creates `worldServer` through `newConfiguredWorldServer`. Server construction starts a `ShardLoop` for:

- shard ID: `stonewake_vale.primary`
- zone ID: `stonewake_vale`
- queue limit: `1024`
- command timeout: `3s`

Startup emits `world.loop_started`. Stop support exists in the loop package for tests, but the HTTP service does not expose an operator stop endpoint.

## Command Lifecycle

1. HTTP handler validates and decodes the existing request.
2. Handler submits an AmandaCore command to the Stonewake loop.
3. The loop accepts or times out the command through a bounded queue.
4. The single loop worker applies the command.
5. Existing world helpers mutate session/NPC state inside the command apply function.
6. The adapter syncs compact player/NPC state into the loop snapshot.
7. The handler returns the existing HTTP response shape.

Important events:

- `world.command_accepted`
- `world.command_applied`
- `world.command_rejected`
- `world.command_timeout`
- `world.snapshot_emitted`
- `world.replay_recorded`
- `world.reconnect_restored`
- `world.loop_stopped`

## State Ownership

Loop-owned for Stonewake hot paths:

- connect
- disconnect
- reconnect
- movement
- target selection
- auto-attack toggle
- ability activation
- quest accept/complete/track
- inventory move/equip
- action-bar assign/move/clear
- snapshot/poll

Existing direct locked helpers remain for systems outside the M4/M5 boundary, including social/economy/admin/housing/dungeon/travel/gathering/crafting/vendor flows. Loot inspect and claim are now loop-backed.

## Persistence Boundary

The loop orders mutations. Existing persistence methods still save state:

- `UpdateCharacterState`
- `UpdateCharacterInventory`
- `UpdateCharacterActionBarSlots`
- `UpdateCharacterProgression`
- `UpdateCharacterTrackedQuests`

SQL transactional repositories remain test-backed foundation work from Milestone 3 and are not the default runtime path in this milestone.

## Replay Testing

Run focused loop tests:

```powershell
Push-Location Services
go test ./internal/worlds/loop -count=1
go test ./internal/worlds -run Stonewake -count=1
Pop-Location
```

Replay is in-memory only. The loop records logical ticks and command payloads and can rebuild a compact final snapshot for deterministic tests.

## Current HTTP API Mapping

- `POST /v1/world/connect` -> `ConnectWorldSession`
- `POST /v1/world/disconnect` -> `DisconnectWorldSession`
- `POST /v1/world/reconnect` -> `ReconnectWorldSession`
- `POST /v1/world/move` -> `ApplyMovement`
- `POST /v1/world/target` -> `SelectTarget` or `ClearTarget`
- `POST /v1/world/attack/auto` -> `StartAutoAttack`
- `POST /v1/world/attack/ability` -> `UseAbility`
- `POST /v1/world/quest/accept` -> `AcceptQuest`
- `POST /v1/world/quest/complete` -> `ClaimQuestReward`
- `POST /v1/world/quest/track` -> `ProgressQuestObjective`
- `POST /v1/world/loot/inspect` -> `OpenLoot`
- `POST /v1/world/loot/claim` -> `ClaimLootItem`
- `POST /v1/world/inventory/move` and `POST /v1/world/inventory/equip` -> `MoveInventoryItem`
- `POST /v1/world/action-bar/*` -> `UpdateActionBar`
- `GET /v1/world/state` -> `RequestSnapshot`

The mapping is internal. Route names and JSON shapes stay unchanged.

## Validation

Required automated validation:

```powershell
git diff --check

Push-Location Services
go test ./... -count=1 -timeout 15m
Pop-Location

powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

If `start-local.ps1 -StartLauncher` starts successfully but no one plays manually, report startup smoke only and do not claim manual gameplay validation.

## Operational Risks

- The loop queues commands but does not yet run a continuous authoritative fixed tick independent of HTTP traffic.
- The compact loop snapshot is not yet the full replication contract.
- Some world systems still mutate through direct locked helpers.
- Long-running persistence in a command apply function can still hold up the shard queue.

## Clean-Room Note

This loop model is AmandaCore-owned. It does not use external MMO packet/opcode tables, schemas, script names, command names, class layouts, IDs, or module structures.

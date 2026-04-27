# Stonewake Combat, Quest, And Loot Runbook

## Scope

This runbook covers Milestone 5 validation for combat, threat, loot, quest progression, and reward commands routed through the Stonewake world loop.

## Command Lifecycle

1. The HTTP handler decodes the existing request.
2. The handler submits a gameplay command to the Stonewake loop.
3. The loop serializes the command with all other Stonewake mutations.
4. Existing AmandaCore gameplay helpers validate range, session, ability, quest, loot, and inventory rules.
5. The adapter syncs session and NPC state back into the compact loop snapshot.
6. The handler returns the existing response shape.

## Loop-Owned Gameplay State

The loop model now carries:

- player target, health, resource, auto-attack, quest progress, inventory slots, loot claim keys, and currency
- NPC health, target, alive/targetable state, respawn tick, and threat table
- loot containers and claim state

## HTTP Mapping

- `POST /v1/world/target` -> `SelectTarget` or `ClearTarget`
- `POST /v1/world/attack/auto` -> `StartAutoAttack` or `StopAutoAttack`
- `POST /v1/world/attack/ability` -> `UseAbility`
- `POST /v1/world/quest/accept` -> `AcceptQuest`
- `POST /v1/world/quest/complete` -> `ClaimQuestReward`
- `POST /v1/world/loot/inspect` -> `OpenLoot`
- `POST /v1/world/loot/claim` -> `ClaimLootItem`
- `GET /v1/world/state` -> `RequestSnapshot`

## Replay Testing

Run focused tests:

```powershell
Push-Location Services
go test ./internal/worlds/loop -count=1
go test ./internal/worlds -run "Stonewake|Loop" -count=1
Pop-Location
```

Replay scenarios cover:

- connect, target, ability damage, death
- quest accept, objective progress, completion, reward
- loot generation, claim, inventory update
- duplicate loot/reward retry
- concurrent reward claim ordering

## Duplication Checks

The expected behavior is:

- concurrent loot claims grant at most one item set
- concurrent quest reward claims grant at most one item/currency set
- already claimed loot returns a safe rejection or replay without additional inventory mutation
- already rewarded quests return a safe rejection without additional inventory/currency mutation

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

If startup succeeds but no one plays manually, report startup smoke only.

## Manual Gameplay Focus

Human testing should verify:

- target hostile
- start/stop auto-attack
- use ability
- hostile health and death state
- kill quest progress
- loot container appears and can be claimed once
- quest reward can be claimed once
- disconnect/reconnect preserves quest, loot, inventory, combat-relevant state

## Clean-Room Note

The combat, threat, loot, quest, replay, command, and test behavior here is AmandaCore-original. Do not introduce external MMO opcode, schema, formula, ID, script, command, or module artifacts.

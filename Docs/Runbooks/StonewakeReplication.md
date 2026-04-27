# Stonewake Replication Runbook

## Purpose

This runbook describes the Milestone 6 replication/convergence path for the current Stonewake playable slice.

## How The Loop Emits Frames

The Stonewake loop compares the authoritative snapshot before and after each accepted command. If player, NPC, loot, combat, quest, inventory, action-bar, or world identity state changes, the loop increments `stateVersion` and records a retained replication frame.

No-op polls may advance command sequence and tick but do not advance `stateVersion`.

## Cursor Usage

Current HTTP clients can poll without a cursor:

```text
GET /v1/world/state?worldSessionToken=<token>
```

Clients that want delta metadata can send the latest cursor:

```text
GET /v1/world/state?worldSessionToken=<token>&since=<cursor>
```

Cursor format:

```text
shardId:zoneId:stateVersion:sequence:tick
```

Malformed cursors return `400 invalid_cursor`.

## Response Metadata

Loop-backed world responses include additive fields:

- `snapshotVersion`
- `deltaVersion`
- `cursor`
- `fullSnapshot`
- `resyncRequired`
- `changed`
- `replication`

Existing clients may ignore these fields. O3DE stores them and drops stale non-resync frames.

## Resync Behavior

The loop retains a bounded in-memory delta history. If a client cursor is too old or newer than the server state, the server emits a resync-required full snapshot frame. The client should replace its local state from the full response payload and store the fresh cursor.

## Observability

Expected events:

- `replication.snapshot_emitted`
- `replication.delta_emitted`
- `replication.cursor_accepted`
- `replication.cursor_stale`
- `replication.resync_required`
- `replication.client_converged`
- `replication.client_diverged`
- `replication.frame_dropped`

World metrics expose loop state version and retained replication frame bounds through `/v1/world/metrics`.

## Validation

Run from the repository root:

```powershell
Push-Location Services
go test ./internal/worlds/replication -count=1
go test ./internal/worlds/loop -count=1
go test ./internal/worlds -run "Replication|Stonewake" -count=1
go test ./... -count=1 -timeout 15m
Pop-Location
```

For client-facing changes, also run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-local.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\build-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\verify-o3de-client.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\Infra\dev\start-local.ps1 -StartLauncher
```

## Manual Smoke

After startup, confirm:

- connect returns a cursor
- movement changes `snapshotVersion`
- `GET /v1/world/state&since=<cursor>` returns delta metadata
- reconnect returns a fresh full snapshot cursor
- gameplay still works through the existing UI

## Known Limitations

- HTTP polling still receives the full world-session payload.
- Delta frames are retained only in memory.
- Push replication is future work.
- Social/economy systems are not yet converted to this convergence model.

## Clean-Room Note

The replication protocol, frame names, cursor format, tests, and runbook are AmandaCore-owned and do not copy external MMO packet/update-field systems.

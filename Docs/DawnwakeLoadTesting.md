# Dawnwake Load Testing And Sharding Skeleton

## Scope

This milestone expands the Dawnwake Isles server-side skeleton with deterministic load simulation, zone population distribution, transition stress, zone command queue pressure, and a shard assignment skeleton. It is still a single-process runtime. No production cluster coordinator, terrain streamer, navmesh, O3DE runtime bridge, or distributed entity replication is implemented yet.

## Scenarios

Run scenarios from the Go module root:

```powershell
Push-Location Services
go run ./cmd/loadsim --clients 100 --duration 60s --cmd-rate 5 --scenario dawnwake-multizone-sharding --content ../Content/Packs/dawnwake_isles/package.json
Pop-Location
```

Available scenarios:

- `dawnwake-traversal-basic`: smoke test for default spawn, first gate transfer, and visibility before/after transfer.
- `dawnwake-multizone-sharding`: distributes players across zones, routes commands through the in-process shard coordinator, attempts transitions, evaluates visibility, probes reconnect restore, and reports queue pressure.
- `dawnwake-multizone-population`: distributes players across zones and evaluates same-zone visibility once per player.
- `dawnwake-transition-stress`: repeatedly transfers players through enabled transition gates and checks single-zone ownership.
- `dawnwake-command-pressure`: routes simulated commands into bounded shard queues and reports backpressure counts.
- `dawnwake-zone-isolation`: assigns zones across five in-process shard IDs and validates ownership isolation.

## Flags

- `--clients`: number of simulated players.
- `--duration`: simulated scenario budget; scenarios use this to derive command or transition iterations.
- `--cmd-rate`: simulated commands per second.
- `--scenario`: scenario name.
- `--content`: package manifest path.
- `--zone-distribution`: optional weighted zone placement list.
- `--transition-rate`: fraction of accepted commands that should attempt a transition when the movement pattern is mixed.
- `--movement-pattern`: `mixed`, `transitions`, or `local`.
- `--shards`: in-process shard count.
- `--queue-limit`: per-shard queue pressure limit used by the load simulator.
- `--spawn-zone`: optional single-zone spawn override.
- `--report-json`: optional path for a JSON report.

Example distribution:

```powershell
--zone-distribution "dawnwake_landing=40,amberglass_fields=25,mistwood_reach=15,highroad_pass=10,kingsfall_harbor=10"
```

Weights are normalized, so they do not have to total 100. Unknown zones, repeated zones, empty zones, and non-positive weights are rejected.

## Reporting

The report includes:

- content package loaded
- continent activated
- zones activated
- transition gates loaded
- players attached
- zone population
- shard assignment count
- transitions requested, completed, and rejected
- commands enqueued, dequeued, and backpressured
- commands by type
- visibility evaluations and deltas
- NPCs spawned
- average, p50, p95, p99, and max tick duration
- max queue depth
- zone queue depths
- backpressure events
- errors

Do not commit generated JSON reports.

## Queue And Backpressure Behavior

Each `ZoneRuntime` now owns a lightweight command queue. The queue is deterministic and in-memory:

- capacity `0` means unbounded
- positive capacity enables backpressure
- accepted commands increment enqueue counters
- rejected commands increment backpressure counters
- dequeue operations preserve FIFO order
- queue depth and max depth are observable

This queue is not yet a production worker system. It is the first testable boundary for future command scheduling, backpressure, and shard-local work loops.

## Shard Assignment Skeleton

`ContinentRuntime.AssignZonesToShards` binds active zones to shard IDs through a simple policy:

- one zone has exactly one owning shard
- one shard can own multiple zones
- assignment can use a max-zones-per-shard value
- transfer validation rejects destinations that are not bound to a shard
- character ownership remains zone-scoped

The default Dawnwake activation binds all active zones to a single shard. The zone isolation scenario uses one zone per shard to validate the future distributed boundary without adding networking or cross-process handoff.

## Current Non-goals

- no production distributed shard coordinator
- no multi-process handoff
- no cross-zone entity replication
- no terrain or navmesh streaming
- no O3DE client interaction
- no real player behavior model
- no combat load beyond command skeleton accounting
- no generated load reports committed to source

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

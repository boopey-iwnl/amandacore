# Multi-zone Load Testing And Sharding Skeleton

## Scope

This milestone adds AmandaCore's first deterministic multi-zone load simulation and an in-memory zone-to-shard assignment skeleton. It validates Dawnwake package topology under repeated simulated transitions without introducing a distributed runtime, networked shard coordinator, or production cross-zone handoff.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Shard Assignment Model

The first skeleton lives in `Services/internal/worlds/sharding.go`.

Concepts:

- `ShardID`: stable string identifier for an in-process simulated zone shard.
- `ZoneShardAssignment`: a zone-to-shard ownership record.
- `ShardAssignmentPolicy`: currently defines shard count only.
- `BuildZoneShardAssignments`: assigns sorted package zones to shards in deterministic round-robin order.
- `ResolveZoneShard`: rejects unknown zones before a simulated transition can complete.

This is deliberately narrow. It proves ownership routing data exists and is deterministic, but it does not move simulation to separate goroutines, processes, machines, or persistence domains yet.

## Loadsim Scenario

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario dawnwake-multizone-sharding-basic --transition-loops 3 --shards 2 --seed 42 --content ..\Content\Packs\dawnwake_isles\package.json
```

The scenario:

- loads and validates `dawnwake_isles`
- activates all package zones into the world runtime
- builds deterministic shard assignments
- starts simulated players at the continent default entry
- performs repeated transition probes through enabled transition gates
- rejects missing destination zones, missing destination entry points, disabled gates, and missing shard assignments
- reports per-zone population, per-shard population, transition counts, queue pressure, and tick timing percentiles

## Report Fields

The Dawnwake loadsim report includes:

- package and continent identifiers
- validation errors
- activated zones and catalogs
- transition gates loaded
- players attached
- transition loops and deterministic seed
- shard count and zone shard assignments
- final zone population
- final shard population
- transition counts by transition and source zone
- requested, completed, and rejected transition totals
- visibility and streaming hint counters
- spawned NPC and quest provider counts
- average, p50, p95, p99, and max tick duration
- max simulated queue depth
- rejected command count
- errors

## Current Limitations

- Shard assignments are in-memory and deterministic only.
- Simulation still runs in one process and one world server instance.
- Transition probes validate package topology and destination ownership, but do not perform production player handoff.
- Queue depth is a simulated command pressure value, not a live gateway queue.
- Tick percentiles are sampled from local world ticks during the scenario duration.

## Next Milestone

The next recommended milestone is production zone handoff and shard coordinator design:

- character zone ownership service contract
- handoff request/ack/reject flow
- durable handoff journal
- reconnect correction after handoff
- per-zone command queue abstraction
- shard worker lifecycle
- scenario tests for rejected and retried handoffs

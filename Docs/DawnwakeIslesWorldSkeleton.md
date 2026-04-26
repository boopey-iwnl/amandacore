# Dawnwake Isles World Skeleton

## Scope

Dawnwake Isles is AmandaCore's first original multi-zone continent skeleton. This milestone adds the server-side topology, zone activation, movement boundary checks, transition validation, runtime handoff, visibility deltas, and a traversal load simulator. It does not add final terrain, navmesh import, city services, art placement, production streaming, or an O3DE renderer.

Current coordinate data is scaffolding. The owner-supplied final Dawnwake Isles and Kingsfall Harbor map images were not present in the local repository during this implementation, so the package uses provisional rectangular bounds with:

- `accuracy`: `placeholder`
- `source`: `pending_map_trace`
- TODO metadata in package and continent files for replacing coordinates from the final maps before 0.5.

## Package Layout

The package root is `Content/Packs/dawnwake_isles/package.json`.

```text
Content/Packs/dawnwake_isles/
  package.json
  continent/dawnwake_isles.continent.json
  zones/dawnwake_landing.zone.json
  zones/amberglass_fields.zone.json
  zones/mistwood_reach.zone.json
  zones/highroad_pass.zone.json
  zones/kingsfall_harbor.zone.json
  npcs/dawnwake_npcs.json
  items/dawnwake_items.json
  loot/dawnwake_loot.json
  quests/dawnwake_quests.json
  abilities/dawnwake_abilities.json
  auras/dawnwake_auras.json
```

Package identity:

- `package_id`: `dawnwake_isles`
- `display_name`: `Dawnwake Isles`
- `schema_version`: `1`
- `version`: `0.1.0`

## Continent Definition

`dawnwake_isles.continent.json` declares:

- `continent_id`
- display metadata and units
- ordered zone references
- adjacency metadata
- a continent `default_entry`
- streaming defaults such as interest radius and gate hint radius
- tags and placeholder source metadata

The runtime validates that continent zone references exist, zone IDs are unique, adjacency endpoints exist, default entry points exist, and transition destination entry points are valid.

## Zone Skeletons

The initial zone IDs are stable AmandaCore-style snake_case IDs:

- `dawnwake_landing`: starter coastal zone.
- `amberglass_fields`: inland wilds zone.
- `mistwood_reach`: forest/wilds zone.
- `highroad_pass`: route and pass zone.
- `kingsfall_harbor`: capital city skeleton.

Each zone file includes:

- `zone_id`
- `display_name`
- `description`
- axis-aligned `bounds`
- `entry_points`
- `transition_gates`
- `spawn_groups`
- `quest_providers`
- runtime config
- streaming hints
- placeholder accuracy metadata

`kingsfall_harbor` is tagged as a city and capital. Its runtime hints mark high player density, lower hostile density, social/vendor/quest provider placeholders, and future city service hooks. Full city systems are intentionally not implemented yet.

## Coordinate Convention

Dawnwake Isles uses a local continent coordinate system measured in AmandaCore server units:

- `X`: east/west position, with positive X east.
- `Y`: coastal/inland position, with positive Y inland/north in the placeholder layout.
- `Z`: elevation.
- Bounds are axis-aligned boxes for this milestone.
- Entry points and spawn points must be inside their zone bounds.
- Transition gates must be inside the source zone bounds and sit near a shared border.
- Zone bounds must not overlap unless a future zone explicitly allows it.

O3DE mapping remains a separate transform layer. Future tooling should convert authored or traced O3DE/worldbuilding coordinates into these server units without making O3DE asset placement the authoritative simulation model.

## Adjacency And Transition Gates

The continent adjacency list describes broad topology. Zone transition gates provide actionable movement handoff metadata:

- `transition_id`
- `from_zone_id`
- `to_zone_id`
- `kind`
- `gate_bounds`
- arrival `entry_point_id_on_arrival`
- optional required flags, disabled state, and tags

The current transition kinds are authored as AmandaCore names such as `Road`, `MountainPass`, and `CityGate`.

Runtime movement can request a transfer when a character exits its current zone through a transition gate. Transfer validation checks:

- source zone exists
- destination zone exists
- character is owned by the source zone
- character is inside or near the gate bounds
- destination entry point exists
- transition is enabled

Successful transfer updates character ownership, places the character at the destination entry point, and emits zone exit, zone enter, routing update, transition completed, and state diff data. Rejected transfer requests emit a rejection event and leave ownership unchanged.

## Runtime Ownership

The current implementation activates validated package zones into the existing Go `worldServer`. Each loaded zone gets a `ZoneRuntime` record for package activation metadata, and the package also contributes normal world zone definitions, transition landmarks, NPC spawn definitions, quest providers, loot tables, items, quests, abilities, and auras.

The current model is intentionally conservative:

- A character still belongs to one active world zone at a time.
- Package activation validates transition metadata before the server accepts it.
- Transition gates are exposed as server-side zone transition landmarks for navigation and loadsim reporting.
- Production cross-zone handoff, sharding, and cross-zone interest replication are future milestones.
- Existing single-zone package behavior is preserved by treating a package with one zone as normal additive runtime content.

## Visibility And Streaming Hints

The interest-management skeleton is deliberately simple:

- Visibility queries scan same-zone entities by radius.
- Far same-zone entities are excluded.
- Visibility deltas report entered and exited entity IDs.
- Zone transfer clears the old visibility set and evaluates from the new zone.
- Optional adjacent-zone streaming hints are emitted when a player is near a transition gate.

This is a naive scan for now. The extension point is a future grid, quadtree, or cell streamer behind the same query/delta surface.

## Persistence And Reconnect

`CharacterZoneStore` defines the minimal reconnect state boundary:

- current zone ID
- position
- optional facing

The in-memory implementation restores a character to the saved zone when the saved zone exists and the saved position is inside bounds. Invalid saved zone or out-of-bounds position falls back to the continent default entry and emits a correction event. Durable persistence can replace the interface without changing the runtime handoff contract.

## Load Simulator

The Dawnwake traversal scenario validates package load, continent activation, default spawn, transition handoff, and visibility before and after transfer.

```powershell
Push-Location Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ../Content/Packs/dawnwake_isles/package.json
go run ./cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario dawnwake-multizone-sharding-basic --transition-loops 3 --shards 2 --seed 42 --content ../Content/Packs/dawnwake_isles/package.json
Pop-Location
```

The legacy scenario alias `dawnwake-traversal-basic` is still accepted. The `dawnwake-streaming-basic` report includes package load, continent ID, zones activated, transition gates loaded, players attached, transition counts, visibility evaluation counts, streaming hint counts, NPCs spawned, quest providers, tick duration summaries, queue depth, and errors.

The `dawnwake-multizone-sharding-basic` scenario performs repeated transition probes across enabled gates and reports deterministic zone shard assignments, final zone population, final shard population, transition counts, rejected commands, simulated queue depth, and tick timing percentiles. This is still a local loadsim and sharding skeleton, not production distributed runtime ownership.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

## Current Limitations

- Bounds and transition coordinates are placeholder scaffolding pending map tracing.
- Transition movement is currently metadata validation and loadsim probing, not production handoff.
- Visibility and streaming hints are loadsim-level counters for this milestone, not cross-zone entity replication.
- Kingsfall Harbor has city runtime hints and provider placeholders, not full city services.
- Durable zone handoff persistence is not implemented for package-authored Dawnwake traversal yet.
- No navmesh, terrain, O3DE import, phasing, city economy, or production streamer is included.

## Next Milestone

The next recommended milestone is production zone handoff and shard coordinator design:

- character zone ownership service contract
- handoff request/ack/reject flow
- durable handoff journal
- reconnect correction after handoff
- per-zone command queue abstraction
- shard worker lifecycle
- scenario tests for rejected and retried handoffs

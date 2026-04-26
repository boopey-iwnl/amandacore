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
- `schema_version`: `amandacore.content.v1`
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

`ContinentRuntime` activates validated zone definitions through `ZoneRuntimeFactory`. Each active `ZoneRuntime` owns its local entity registry and zone definition. Character ownership is tracked by `CharacterZoneState`.

The current model is single-owner:

- A character or active entity belongs to exactly one zone runtime.
- Commands route through the character's owning `ZoneRuntime`.
- Zone transfer removes the character from the source registry and inserts it into the destination registry.
- Existing single-zone package behavior is preserved by treating one activated zone as the whole runtime.

`WorldRuntime` can activate all continent definitions in a package and route commands by character ID. This keeps the future sharding boundary explicit without adding a production shard coordinator yet.

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
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ../Content/Packs/dawnwake_isles/package.json
Pop-Location
```

The report includes package load, continent activation, zones activated, transition gates loaded, players attached, transition counts, visibility evaluation counts, NPCs spawned, tick duration summaries, queue depth, and errors.

Additional Dawnwake load scenarios are documented in `Docs/DawnwakeLoadTesting.md`. They cover configurable zone population distribution, repeated transition stress, command queue pressure, queue backpressure, tick duration percentiles, and the shard assignment skeleton.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

## Current Limitations

- Bounds and transition coordinates are placeholder scaffolding pending map tracing.
- Transition movement uses axis-aligned gate bounds only.
- Visibility uses a naive same-zone scan.
- Adjacent-zone visibility is exposed as streaming hints, not cross-zone entity replication.
- Kingsfall Harbor has city runtime hints and provider placeholders, not full city services.
- Reconnect persistence is currently represented by an interface and in-memory implementation.
- Load simulations are deterministic skeletons, not representative player behavior models.
- Shard assignment is single-process and does not yet move work across machines.
- No navmesh, terrain, O3DE import, phasing, city economy, or production streamer is included.

## Next Milestone

The next recommended milestone is larger load tests and multi-zone sharding:

- multi-zone load simulation
- configurable simulated player counts
- zone population distribution
- transition stress testing
- command queue pressure tests
- tick duration percentiles
- session gateway pressure tests
- zone runtime isolation tests
- backpressure behavior
- shard assignment skeleton
- tests and reporting

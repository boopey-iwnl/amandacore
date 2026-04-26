# Content Package Loader

AmandaCore content packages are server-side JSON manifests owned by AmandaCore. They give the Go world runtime a validation boundary between authored content and active simulation state.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Layout

The compact progression package lives at:

```text
Content/Packs/dev_foundation/
  package.json
  zones/dev_isle_edge.zone.json
  npcs/dev_npcs.json
  items/dev_items.json
  loot/dev_loot.json
  quests/dev_quests.json
  abilities/dev_abilities.json
  auras/dev_auras.json
```

The first multi-zone package lives at:

```text
Content/Packs/dawnwake_isles/
  package.json
  continent/dawnwake_isles.continent.json
  zones/*.zone.json
  npcs/dawnwake_npcs.json
  items/dawnwake_items.json
  loot/dawnwake_loot.json
  quests/dawnwake_quests.json
  abilities/dawnwake_abilities.json
  auras/dawnwake_auras.json
```

The default local package is `Content/Packs/dev_foundation/package.json`. Override it with:

```powershell
$env:AMANDACORE_CONTENT_PACKAGE="Content/Packs/dev_foundation/package.json"
```

Relative paths are resolved from the current working directory first, then by walking up parent directories. This lets service tests run from `Services` while still finding repo-root content.

## Manifest

`package.json` contains:

- `package_id`
- `display_name`
- `version`
- `schema_version`
- `description`
- `authorship`
- optional `continent_files`
- `zones`
- `npc_catalogs`
- `item_catalogs`
- `loot_catalogs`
- `quest_catalogs`
- `ability_catalogs`
- `aura_catalogs`
- `tags`

`schema_version` is currently `1`. Unsupported schema versions are rejected before runtime activation.

## Continent And Zone Format

A continent file can define:

- `continent_id`, display metadata, origin, units, and tags
- ordered zone references
- adjacency edges and transition gate references
- a default entry zone and entry point
- streaming defaults such as interest radius and gate hint radius

A zone file defines:

- `zone_id`, `display_name`, `description`, optional `continent_id`
- `bounds` with `min_x`, `min_y`, `min_z`, `max_x`, `max_y`, `max_z`
- `entry_points`
- `transition_gates`
- `spawn_groups`
- `quest_providers`
- `runtime` with `tick_ms`, `max_players`, `max_entities`
- optional `streaming` hints and metadata

Validation requires valid bounds, positions inside bounds, unique zone IDs, positive runtime limits, and a tick interval from 16ms to 250ms.

## Catalogs

NPC catalogs define archetypes with health, level, disposition, combat ranges, damage, cadence, optional default abilities, and tags.

Item catalogs define item ID, display name, description, kind, quality, max stack, and tags.

Loot catalogs define loot tables with item references, quantity ranges, drop chance percentages, and guaranteed flags.

Quest catalogs define quest metadata, prerequisite quest IDs, objective graph nodes, and item rewards. Objective nodes currently support `kill_npc`, `collect_item`, and `talk_provider`.

Ability catalogs define ability ID, school, target rule, range, timing, cooldown, effects, and tags. Aura catalogs define aura ID, kind, duration, stack behavior, tick rule, modifiers, and tags. Content-loaded abilities and auras are validated and registered, but the current combat runtime still uses the existing hardcoded ability catalog.

## Validation

The loader reports all practical errors without panicking. Error codes include:

- `MissingFile`
- `MalformedJson`
- `UnsupportedSchemaVersion`
- `MissingRequiredField`
- `DuplicateID`
- `InvalidID`
- `InvalidEnum`
- `InvalidNumberRange`
- `BrokenReference`
- `PositionOutOfBounds`
- `ObjectiveGraphCycle`
- `RuntimeConfigInvalid`
- `TransitionInvalid`

Package-level validation catches duplicate IDs, missing files, malformed JSON, broken spawn/NPC/loot/item/quest/ability/aura references, invalid numeric ranges, positions outside zone bounds, invalid runtime config, quest objective dependency cycles, missing continent zone references, broken adjacency references, duplicate transition IDs, transition gates outside source bounds, missing destination zones, and missing destination entry points.

## Runtime Activation

Only validated content activates. Activation builds a `RuntimeContentRegistry`, creates `ZoneRuntime` records, converts package zones to current world zone definitions, registers quest providers as friendly NPCs, converts spawn groups to mob spawn definitions, registers content loot tables, and converts package quest objective graphs into the current server-authoritative quest graph model.

Existing hardcoded Stonewake, Brindlebrook, and dev progression flows remain as fallback. The content package is additive and does not replace current starter content. If a package item, loot table, or quest ID already exists in built-in content, runtime activation preserves the built-in definition and keeps the package definition in the `RuntimeContentRegistry` for validation/reporting. Package-spawned NPC entity IDs use a content prefix so authored spawn points cannot collide with built-in dev spawn IDs.

Structured events include:

- `content.package.load_started`
- `content.package.load_completed`
- `content.package.load_failed`
- `content.package.validation_started`
- `content.package.validation_completed`
- `content.package.validation_failed`
- `content.package.activated`
- `content.package.activation_failed`
- `content.continent.loaded`
- `content.zone.loaded`
- `content.zone_transition.loaded`
- `content.zone_adjacency.loaded`
- `content.catalog.loaded`
- `content.reference.resolved`
- `content.reference.broken`
- `content.quest_provider.registered`
- `world.zone.runtime_created`
- `loadsim.content.started`
- `loadsim.content.completed`
- `loadsim.dawnwake.started`
- `loadsim.dawnwake.completed`

## Loadsim

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario content-package-basic --content ..\Content\Packs\dev_foundation\package.json
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ..\Content\Packs\dawnwake_isles\package.json
go run ./cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario dawnwake-multizone-sharding-basic --transition-loops 3 --shards 2 --seed 42 --content ..\Content\Packs\dawnwake_isles\package.json
```

The `content-package-basic` scenario loads and validates the compact dev package, activates zones, counts spawned NPCs, accepts `dev_first_hunt`, resolves a stalker kill, claims deterministic guaranteed loot, grants quest rewards, and reports tick timing.

The `dawnwake-streaming-basic` scenario loads and validates Dawnwake continent topology, activates package zones, counts transition gates, simulates one transition probe per client from the default entry, records visibility and streaming-hint counts, and reports tick timing.

The `dawnwake-multizone-sharding-basic` scenario repeats transition probes across enabled gates, builds deterministic zone shard assignments, reports final zone and shard populations, records transition counts, and emits tick timing percentiles for local pressure testing.

## Current Limitations

- Ability and aura package entries are validated and registered, but combat still resolves the existing hardcoded Warrior ability catalog.
- Runtime quest activation supports kill, collect, and provider-interaction objective graph nodes. Broader branching, optional objectives, repeatable quests, and editor-authored quest UI are future work.
- Dawnwake transition handling is metadata and loadsim validation only. The sharding skeleton reports deterministic zone ownership assignments but does not yet run distributed shard workers or production player handoff.
- Dawnwake coordinates are placeholder server rectangles pending map tracing.
- Loadsim loot claiming is deterministic and server-side. Client UI for package-authored loot windows and quest offers is future work.
- This is not a content editor, O3DE export pipeline, terrain streamer, or compiled binary content format.

## Next Milestone

Production zone handoff and shard coordinator design should add a character zone ownership service contract, handoff request/ack/reject flow, durable handoff journal, reconnect correction after handoff, per-zone command queue abstraction, shard worker lifecycle, and scenario tests for rejected and retried handoffs.

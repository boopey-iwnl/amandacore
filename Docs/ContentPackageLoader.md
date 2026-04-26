# Content Package Loader

AmandaCore content packages are server-side JSON manifests owned by AmandaCore. They give the Go world runtime a validation boundary between authored content and active simulation state.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Layout

The first package lives at:

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
  zones/dawnwake_landing.zone.json
  zones/dawnwake_tideglass_shoal.zone.json
  zones/dawnwake_windspur_rise.zone.json
  maps/dawnwake_landing.map.json
  maps/dawnwake_tideglass_shoal.map.json
  maps/dawnwake_windspur_rise.map.json
  npcs/dawnwake_npcs.json
  items/dawnwake_items.json
  loot/dawnwake_loot.json
  quests/dawnwake_quests.json
  abilities/dawnwake_abilities.json
  auras/dawnwake_auras.json
```

The loader default used by content tests and loadsim is `Content/Packs/dev_foundation/package.json`. The HTTP world service keeps the existing hardcoded starter world unless a package is explicitly selected, preserving current local startup behavior. Select a package with:

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
- `zones`
- `map_exports`
- `npc_catalogs`
- `item_catalogs`
- `loot_catalogs`
- `quest_catalogs`
- `ability_catalogs`
- `aura_catalogs`
- `tags`

`schema_version` is currently `1`. Unsupported schema versions are rejected before runtime activation.

## Zone Format

A zone file defines:

- `zone_id`, `display_name`, `description`
- `bounds` with `min_x`, `min_y`, `min_z`, `max_x`, `max_y`, `max_z`
- `entry_points`
- `spawn_groups`
- `quest_providers`
- `transitions`
- `runtime` with `tick_ms`, `max_players`, `max_entities`

Validation requires valid bounds, positions inside bounds, unique zone IDs, positive runtime limits, and a tick interval from 16ms to 250ms.

`transitions` are the current server-side adjacency hook for streamed-world expansion. Each transition has:

- `transition_id`
- `display_name`
- `target_zone_id`
- `destination_entry_id`
- `position`
- `radius`
- `tags`

Validation rejects transition positions outside the source zone, missing target zones, and missing destination entry points.

## Map Export Format

Map export files are AmandaCore-owned placeholder metadata for server validation and future O3DE streaming hooks. They are not terrain, prefab, asset, or world-partition files.

A map export defines:

- `map_id`, `zone_id`, `display_name`
- `coordinate_space`, currently `amandacore_server` or `o3de_placeholder`
- `bounds`
- `entry_points`
- `adjacent_zones`
- `transition_points`
- `streaming_cells`
- `landmarks`
- `authoring_source`
- `tags`

Validation requires each map export to reference an existing zone, have bounds that contain the zone bounds, keep entry points, transition points, landmarks, and streaming cells inside map bounds, reference existing target zones and destination entries, and match transition metadata already declared by the zone. Adjacent zones can require reciprocal adjacency; Dawnwake uses that for its current transition loop.

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

Package-level validation catches duplicate IDs, missing files, malformed JSON, broken spawn/NPC/loot/item/quest/ability/aura/map references, invalid numeric ranges, positions outside zone or map bounds, invalid runtime config, missing reciprocal adjacency where required, transition metadata mismatches, and quest objective dependency cycles.

## Runtime Activation

Only validated content activates. Activation builds a `RuntimeContentRegistry`, creates `ZoneRuntime` records, converts package zones and transition points to current world zone definitions, attaches map export metadata to zone runtimes, registers quest providers as friendly NPCs, converts spawn groups to mob spawn definitions, registers simple quest projections, and merges package items into the current item lookup path.

For zones with map exports, `ZoneRuntime` includes:

- `MapID`
- map bounds
- adjacent zone IDs
- transition hints
- streaming cells

World responses include a `streaming` payload for the active zone. It exposes map ID, bounds, adjacent zones, transition hints, and placeholder streaming cells for future client wiring while keeping Go authoritative over traversal.

Existing hardcoded Stonewake and Brindlebrook flows remain as fallback. The content package is additive and does not replace the current starter content yet.

Structured events include:

- `content.package.load_started`
- `content.package.load_completed`
- `content.package.load_failed`
- `content.package.validation_started`
- `content.package.validation_completed`
- `content.package.validation_failed`
- `content.package.activated`
- `content.package.activation_failed`
- `content.zone.loaded`
- `content.zone.transition.loaded`
- `content.zone.transition.validation_failed`
- `content.map_export.loaded`
- `content.map_export.validation_failed`
- `content.catalog.loaded`
- `content.reference.resolved`
- `content.reference.broken`
- `content.quest_provider.registered`
- `world.zone.runtime_created`
- `world.zone.transition_started`
- `world.zone.transition_completed`
- `world.zone.transition_rejected`
- `loadsim.content.started`
- `loadsim.content.completed`
- `loadsim.dawnwake.started`
- `loadsim.dawnwake.completed`
- `loadsim.streaming.started`
- `loadsim.streaming.completed`

## Loadsim

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario content-package-basic --content ..\Content\Packs\dev_foundation\package.json
```

The `content-package-basic` scenario loads and validates the package, activates zones, counts spawned NPCs, accepts `dev_first_hunt`, resolves a stalker kill, claims deterministic guaranteed loot, grants quest rewards, and reports tick timing.

The Dawnwake traversal scenario validates the first multi-zone package and completes one loaded zone transition:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

The report includes loaded package status, validation errors, activated zones, loaded catalogs, spawned NPCs, registered quest providers, transition counts, quest acceptance, NPC kills, loot claims, inventory grants, quest completion, reward grants, tick timing, queue depth, and errors.

The Dawnwake streaming scenario validates the map exports and traverses the current full Dawnwake loop:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

The report adds map exports loaded, streaming cells loaded, streaming hints observed, and zones entered.

## Current Limitations

- Ability and aura package entries are validated and registered, but combat still resolves the existing hardcoded Warrior ability catalog.
- Runtime quest activation projects the first supported objective into the current single-objective quest shape while retaining the full objective graph in `RuntimeContentRegistry`.
- Loot claiming in loadsim is deterministic and server-side; public world HTTP loot endpoints are not introduced in this milestone.
- Dawnwake transition handling is a server-side runtime skeleton. It validates adjacency, attaches placeholder streaming hints, and moves sessions to destination entry points, but does not stream terrain, O3DE assets, or client-side world partitions yet.
- This is not a content editor, O3DE export pipeline, terrain streamer, or compiled binary content format.

## Next Milestone

The next milestone should turn the placeholder map metadata into a generated AmandaCore export path from O3DE-owned authoring data, then add client-facing transition previews and broader streamed-world handoff coverage.

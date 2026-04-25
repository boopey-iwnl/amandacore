# Content Package Loader

## Purpose

The Go content loader turns AmandaCore-authored JSON packages into a validated `RuntimeContentRegistry`. It is the server-side boundary between source content files and runtime activation.

The first multi-zone package is `Content/Packs/dawnwake_isles/package.json`.

## Manifest Shape

`ContentPackageManifest` currently supports:

- package identity: `package_id`, `display_name`, `schema_version`, `version`
- one optional continent file
- one or more zone files
- optional NPC, item, loot, quest, ability, and aura files
- tags and metadata

The supported schema version is `amandacore.content.v1`.

## Loaded Runtime Registry

`RuntimeContentRegistry` stores loaded package data by AmandaCore-native IDs:

- `Continents`
- `Zones`
- `NpcArchetypes`
- `NpcSpawns`
- `QuestProviders`
- `Items`
- `LootTables`
- `Quests`
- `Abilities`
- `Auras`

The registry can activate a validated continent into `ContinentRuntime`. The runtime creates one `ZoneRuntime` per zone definition and initializes placeholder NPC and quest provider entities.

## Validation

Package validation rejects invalid topology before runtime activation. Current checks include:

- missing continent zone references
- duplicate zone IDs
- missing default entry zone or entry point
- adjacency references to missing zones
- transition references to missing destination zones
- transition references to missing destination entry points
- transition gate bounds outside the source zone
- entry points outside zone bounds
- spawn points outside zone bounds
- duplicate transition IDs
- overlapping zone bounds when overlap is not allowed
- missing city hints for city-tagged zones

Validation errors carry stable codes such as `MissingContinentZone`, `MissingDefaultEntry`, `MissingTransitionDestination`, `MissingTransitionEntryPoint`, `TransitionGateOutOfBounds`, `ZoneBoundsOverlap`, `DuplicateTransitionID`, and `InvalidTopology`.

## Runtime Concepts

Core runtime concepts are:

- `ZoneRuntimeFactory`: creates a zone runtime from a zone definition and loaded content.
- `ZoneRuntime`: owns one zone definition and its active entity registry.
- `EntityRegistry`: stores runtime entities for naive visibility and ownership checks.
- `ContinentRuntime`: coordinates multiple zone runtimes, character ownership, transfers, visibility, and reconnect placement.
- `WorldRuntime`: activates all continents and routes commands to character-owning zones.
- `CharacterZoneStore`: persistence interface for zone ID, position, and facing.

For this milestone, activation is intentionally simple and testable. It prepares for a future shard coordinator without requiring production distributed ownership yet.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

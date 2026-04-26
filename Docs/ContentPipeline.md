# AmandaCore Content Pipeline

AmandaCore content must be authored, validated, compiled, and loaded through original AmandaCore formats. Runtime servers must consume AmandaCore content packages, not TrinityCore or AzerothCore database tables, schemas, IDs, scripts, or data.

## Current Flow

The current flow is:

1. Author JSON package files under `Content/Packs`.
2. Validate package manifests, zones, spawn groups, quest providers, and catalogs.
3. Activate validated content into Go runtime structures.
4. Exercise packages through unit tests and loadsim scenarios.

`dev_foundation` is the strict activation fixture. `dawnwake_isles` is the original multi-zone package for traversal, sharding, streaming, and future O3DE map-trace integration.

## Authoritative Data Boundary

The server owns gameplay authority. Content packages may define authored inputs such as zone bounds, NPC archetypes, spawn points, loot references, quests, and traversal gates, but they do not define packet formats or client authority.

Package data must remain AmandaCore-owned:

- original IDs
- original JSON shapes
- original NPC, item, quest, ability, aura, and loot definitions
- original zone topology
- original map-coordinate transforms

## Placeholder Versus Map-accurate Data

Dawnwake package coordinates are placeholder server rectangles until the owner-supplied Dawnwake Isles references are traced into documented normalized coordinates and world-space transforms.

Placeholder package data must be clearly marked with accuracy/source metadata so future map-driven rebuilds can replace it without guessing.

## Future Compiler Responsibilities

The compiler/tooling layer should eventually:

- validate references across all package catalogs
- compile normalized map coordinates into server world-space transforms
- emit compact runtime package artifacts
- produce package summaries for review
- generate replay fixtures for scenario tests
- reject stale or incompatible package versions before activation

Hot reload should be added only where package compatibility can be proven without corrupting live simulation state.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore and AzerothCore were used only as high-level architectural reference. Dawnwake Isles is AmandaCore-original world content. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

# Content Package Loader

AmandaCore content packages are server-side JSON manifests owned by AmandaCore. They give the Go world runtime a validation boundary between authored content and active simulation state.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore and AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

## Layout

Packages live under `Content/Packs`.

Current packages:

- `dev_foundation`: small runtime package used by content-loader tests and the `content-package-basic` loadsim.
- `dawnwake_isles`: original multi-zone Dawnwake skeleton used by traversal and multizone loadsim coverage.

The active Go runtime loader is the `Services/internal/content` package. It loads package manifests, zone files, and optional catalogs, then activates validated content into `worldServer` through `Services/internal/worlds/content_activation.go`.

## Manifest

The loader accepts an AmandaCore package manifest with:

- package identity and schema metadata
- zone file paths
- optional NPC, item, loot, quest, ability, and aura catalog paths
- tags and authoring metadata

The `dev_foundation` package is the strict loader fixture. The richer Dawnwake package carries additional continent and streaming metadata used by load simulation and future map-tracing work.

## Zone Format

Zones define:

- `zone_id`
- `display_name`
- bounds
- entry points
- spawn groups
- quest providers
- runtime hints

The loadsim reader also accepts Dawnwake traversal fields such as `entry_point_id`, `entry_point_id_on_arrival`, disabled transition gates, and placeholder map-trace metadata.

## Runtime Activation

Validated package content is additive. The server keeps built-in starter content available, then activates content-package zones, NPC spawns, quest providers, item definitions, loot tables, and quest definitions.

Content packages do not replace authoritative simulation logic. They provide AmandaCore-owned runtime inputs.

## Loadsim

From `Services`:

```powershell
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario content-package-basic --content ..\Content\Packs\dev_foundation\package.json
go run ./cmd/loadsim --clients 5 --duration 10s --cmd-rate 2 --scenario multizone-pressure --content ..\Content\Packs\dawnwake_isles\package.json --seed 42
```

## Current Limitations

- Dawnwake coordinates are placeholder server rectangles pending map tracing.
- O3DE world-space transforms are not yet generated from package coordinates.
- Hot reload is not enabled; content is loaded at runtime initialization.
- The full Dawnwake package carries richer authoring metadata than the first activation loader consumes.

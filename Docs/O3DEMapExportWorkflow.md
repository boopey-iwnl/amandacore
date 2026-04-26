# O3DE Map Export Workflow

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Purpose

`Services/cmd/content-exporter` turns AmandaCore-owned placeholder authoring metadata into server map exports. This gives the content package loader deterministic, reviewable map metadata without requiring real O3DE terrain, prefab, or asset processor output yet.

The workflow is:

```text
Content/Authoring/DawnwakeIsles/*.authoring.json
  -> Services/cmd/content-exporter
  -> Content/Packs/dawnwake_isles/maps/*.map.json
  -> ContentPackageLoader validation
  -> ZoneRuntime streaming hints
  -> world response streaming payload
  -> client transition preview state
```

## Authoring Metadata

Authoring files describe placeholder O3DE editor markers:

- map and zone IDs
- coordinate space
- source scene name
- bounds
- entry markers
- transition markers
- streaming cell markers
- landmark markers
- adjacency declarations

Marker records include `marker_id` and `entity_name` so future O3DE editor or asset processor integrations have stable names to bind to. These names are AmandaCore-authored placeholders.

## Commands

Regenerate map exports:

```powershell
cd Services
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps
```

Check committed exports:

```powershell
cd Services
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps --check
```

Validate runtime traversal:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

## Generated Output

Generated map exports are checked in because they are runtime content consumed by the package loader. The exporter is deterministic:

- sorted input files
- sorted generated collections
- stable JSON indentation
- no timestamps
- no machine-local paths
- `generated_by: amandacore-content-exporter`

## Current Limits

- The source authoring files are JSON placeholders, not O3DE asset products.
- The exporter does not inspect `.prefab`, terrain, world partition, or asset processor output.
- The client only retains and prints transition preview state; it does not prefetch cells or load assets yet.
- Server traversal remains immediate and radius-based.

## Next Step

Replace the placeholder JSON source with AmandaCore-owned O3DE editor metadata or asset processor output while preserving the same generated map export format and validation boundary.

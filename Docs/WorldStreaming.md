# World Streaming Hooks

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Purpose

The current streaming work is a server-side metadata boundary. It connects validated content packages to placeholder map exports, zone adjacency, transition hints, and streaming cell records without requiring O3DE terrain, prefab, or asset output.

The Go world runtime remains authoritative. Map exports tell the runtime what zones are adjacent and what cells a future client should prepare near a transition; they do not move authority into O3DE.

## Runtime Shape

Validated map exports activate into `ZoneRuntime`:

- `MapID`
- map bounds
- adjacent zone IDs
- transition hints
- streaming cells

The active world response includes a `streaming` payload with:

- `enabled`
- `zoneId`
- `mapId`
- `adjacentZoneIds`
- `bounds`
- `transitionHints`
- `streamingCells`

This gives the future O3DE client enough stable metadata to preview nearby transitions and prepare placeholder cells while the server continues to own movement validation and zone transfer.

## Dawnwake Coverage

`Content/Packs/dawnwake_isles` currently defines three map exports:

- `dw_map_landing`
- `dw_map_tideglass_shoal`
- `dw_map_windspur_rise`

Those exports validate the current traversal loop:

```text
dawnwake_landing -> dawnwake_tideglass_shoal -> dawnwake_windspur_rise -> dawnwake_tideglass_shoal -> dawnwake_landing
```

## Loadsim

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

The scenario validates map exports, activates zone runtimes, confirms streaming cell metadata, traverses the full current Dawnwake loop, and prints transition and streaming hint counts.

## Current Limits

- Map exports are hand-authored placeholder metadata.
- No O3DE asset product, terrain, prefab, or world-partition file is consumed.
- Transition handling is immediate and radius-based.
- Streaming cells are runtime hints, not client-loaded asset chunks.
- There is no interest-management or cross-worker shard handoff yet.

## Next Step

Generate the placeholder map export JSON from AmandaCore-owned O3DE authoring metadata, then add client-side transition previews and cell prefetch behavior against the existing server response payload.

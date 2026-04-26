# World Streaming Hooks

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Purpose

The current streaming work is a server-side metadata boundary. It connects validated content packages to generated placeholder map exports, zone adjacency, transition hints, and streaming cell records without requiring O3DE terrain, prefab, or asset output.

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

The console world client now deserializes the same payload into a lightweight streaming preview model. That model computes the current streaming cell from bounds, tracks visible cell deltas, tracks adjacent zones, and identifies the nearest transition hint from the player's authoritative server position.

## Client Streaming Prototype

The first client-side prototype lives in `Client/Game/AmandaCore.WorldClient` and deliberately has no O3DE runtime dependency yet.

Client responsibilities:

- deserialize the server-owned `streaming` payload
- build a `ClientStreamingFrame` for the active zone, map, cells, and nearest transition
- compute the current cell from server-provided cell bounds
- emit preview events through `IWorldStreamingPreviewSink`
- keep all transition and position authority on the Go world service

The sink boundary is the placeholder O3DE hook. A future O3DE implementation should bind these callbacks to scene or prefab streaming:

- `ZoneEntered`
- `CellBecameVisible`
- `CellBecameHidden`
- `CurrentCellChanged`
- `TransitionHintChanged`
- `MapBoundsChanged`

The current implementation uses `ConsoleWorldStreamingPreviewSink` so local development can see the same event flow without the O3DE SDK installed.

## Export Workflow

Dawnwake map exports are generated from AmandaCore-owned placeholder authoring files:

```text
Content/Authoring/DawnwakeIsles/
  dawnwake_landing.authoring.json
  dawnwake_tideglass_shoal.authoring.json
  dawnwake_windspur_rise.authoring.json
```

Run from the Go module root:

```powershell
cd Services
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps --check
```

The exporter writes deterministic JSON and marks outputs with `generated_by: amandacore-content-exporter`.

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

- Map exports are generated from hand-authored AmandaCore placeholder metadata.
- No O3DE asset product, terrain, prefab, or world-partition file is consumed.
- Transition handling is immediate and radius-based.
- Streaming cells are runtime hints and console preview events, not loaded O3DE asset chunks.
- There is no interest-management or cross-worker shard handoff yet.

## Next Step

Bind `IWorldStreamingPreviewSink` to an O3DE-facing adapter that can show placeholder zone volumes, cell bounds, transition affordances, and future cell prefetch behavior.

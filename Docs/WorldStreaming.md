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

The current implementation can use `ConsoleWorldStreamingPreviewSink`, `PlaceholderSceneStreamingAdapter`, or both. `PlaceholderSceneStreamingAdapter` translates the callback stream into structured scene commands that the O3DE `ZoneStreaming` Gem can consume through its debug request API without changing the world-service contract.

Select the preview sink with:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands
```

Write a deterministic JSON Lines command stream for bridge testing with:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:TEMP\amandacore\streaming.commands.jsonl"
```

Supported sink modes:

- `console`: human-readable streaming event lines
- `scene-commands`: JSON placeholder scene commands
- `both`: both console output and placeholder scene commands

## Placeholder Scene Commands

The placeholder scene adapter emits these command names:

- `CreateZoneBoundsVolume`
- `CreateStreamingCellVolume`
- `HideStreamingCellVolume`
- `HighlightCurrentCell`
- `ClearCurrentCellHighlight`
- `ShowTransitionAffordance`
- `ClearTransitionAffordance`

These commands are intentionally presentation-only. They carry AmandaCore zone IDs, map IDs, cell IDs, display names, bounds, transition positions, transition readiness, and tags from the server-owned streaming payload. They do not grant client authority over traversal, visibility, combat, or persistence.

`Gems/ZoneStreaming` now mirrors this contract as `ZoneStreaming::PlaceholderSceneCommand` and exposes `ZoneStreaming::IZoneStreamingDebugRequests::ApplyPlaceholderSceneCommand`. The Gem stores debug state and draws zone bounds, visible streaming cells, current-cell highlights, and transition affordance markers through O3DE AuxGeom. The JSONL file output is optional and deterministic; it exists so local bridge work can be tested without introducing an O3DE SDK dependency into the C# client.

Run the command translation checks with:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient.Tests/AmandaCore.WorldClient.Tests.csproj
```

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
- Streaming cells are runtime hints and debug volumes, not loaded O3DE asset chunks.
- There is no interest-management or cross-worker shard handoff yet.
- The `ZoneStreaming` Gem consumes the C++ command representation; direct JSONL parsing inside the Gem is deferred.

## Next Step

Connect the launcher/O3DE client bridge so live streaming callbacks feed `IZoneStreamingDebugRequests` in-engine, then replace placeholder authoring metadata with AmandaCore-owned O3DE editor metadata or asset processor output.

# Dawnwake Isles Zone Skeletons

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Package

`Content/Packs/dawnwake_isles/package.json` is the first AmandaCore-owned multi-zone runtime package. It contains three placeholder zones:

- `dawnwake_landing`
- `dawnwake_tideglass_shoal`
- `dawnwake_windspur_rise`

Each zone has original bounds, entry points, spawn groups, quest providers, runtime caps, transition metadata, and matching generated placeholder map export metadata. The package is intentionally small; it validates the runtime loader and traversal boundary before a terrain or asset pipeline exists.

## Map Exports

The package includes three AmandaCore-owned placeholder map export files:

- `maps/dawnwake_landing.map.json`
- `maps/dawnwake_tideglass_shoal.map.json`
- `maps/dawnwake_windspur_rise.map.json`

These files define map IDs, coordinate space, bounds, entry points, adjacent zones, transition hints, streaming cells, and landmarks. They are generated from `Content/Authoring/DawnwakeIsles/*.authoring.json` by `Services/cmd/content-exporter`. They are validation and runtime metadata only; they are not O3DE asset products, terrain data, or prefab data.

Regenerate and verify from the Go module root:

```powershell
cd Services
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps
go run ./cmd/content-exporter --input ..\Content\Authoring\DawnwakeIsles --output ..\Content\Packs\dawnwake_isles\maps --check
```

## Zone Transitions

Transitions are server-side adjacency records:

- source zone owns the transition position and radius
- target zone must exist in the same validated package
- destination entry point must exist in the target zone
- runtime movement places the session at the destination entry point

Current transition coverage:

- `dawnwake_landing.to_tideglass_shoal`
- `dawnwake_tideglass_shoal.to_landing`
- `dawnwake_tideglass_shoal.to_windspur_rise`
- `dawnwake_windspur_rise.to_tideglass_shoal`

These are future streaming hooks, not terrain streams. They prove authoritative traversal state and validation before client streaming assets are introduced.

## Runtime Activation

When the package is loaded, the world runtime:

- creates three `ZoneRuntime` records
- attaches three map exports to those zone runtimes
- registers nine placeholder streaming cells
- registers four transition points
- exposes transition hints in the world response `streaming` payload
- registers three quest providers
- spawns ten placeholder NPCs from loaded spawn groups
- adds package items to the item catalog
- projects supported content quests into the current quest runtime

Existing Stonewake and Brindlebrook hardcoded flows remain available for current tests and local play.

## Client Streaming Preview

The minimal world client now consumes Dawnwake streaming metadata through a dedicated client preview model:

- active zone and map ID
- map bounds
- adjacent zone IDs
- visible streaming cells
- current cell computed from the server-authoritative player position
- nearest transition hint and ready state

`IWorldStreamingPreviewSink` is the first O3DE-facing adapter boundary. The console client uses `ConsoleWorldStreamingPreviewSink` by default, and `PlaceholderSceneStreamingAdapter` can emit structured placeholder commands for zone bounds, streaming cells, current-cell highlighting, and transition affordances. `Gems/ZoneStreaming` now consumes the mirrored C++ command contract through `IZoneStreamingDebugRequests` and draws debug-only in-engine volumes.

Run the client with placeholder scene commands:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands
```

Write deterministic JSON Lines for bridge testing:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:TEMP\amandacore\dawnwake-streaming.commands.jsonl"
```

## Loadsim

Run from the Go module root:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-traversal-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

The scenario validates the package, activates all zones, enters `dawnwake_landing`, completes the first transition to `dawnwake_tideglass_shoal`, verifies spawned NPC content, resolves the `dw_tideglass_sparks` placeholder quest path, claims deterministic guaranteed loot, grants the placeholder reward, and prints a concise report.

Run the streaming hook scenario:

```powershell
cd Services
go run ./cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario dawnwake-streaming-basic --content ..\Content\Packs\dawnwake_isles\package.json
```

This scenario validates map exports, verifies active streaming metadata, traverses landing -> tideglass -> windspur -> tideglass -> landing, observes transition hints, and still exercises the placeholder quest/loot completion path.

## Current Limitations

- No O3DE terrain, prefab, asset, or world-partition data is loaded yet.
- The client streaming hook emits console preview events or placeholder scene commands only; the `ZoneStreaming` Gem currently renders debug AuxGeom volumes from the mirrored command contract.
- Map export files are generated from AmandaCore-owned placeholder authoring metadata, not O3DE asset products.
- Zone bounds and positions are placeholder server coordinates authored for this package only.
- Transition handling is radius-based and immediate.
- Combat and loot in the loadsim are deterministic validation summaries, not a full client session.
- Ability and aura package entries are validated and registered, but combat still uses the existing runtime ability path.

## Next Milestone

Connect the launcher/O3DE client bridge so live Dawnwake streaming callbacks feed the `ZoneStreaming` Gem in-engine, then connect the exporter to real AmandaCore O3DE editor metadata or asset processor output.

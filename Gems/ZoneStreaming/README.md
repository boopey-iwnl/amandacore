# ZoneStreaming Gem Debug Volumes

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, map formats, zone tables, spawn schemas, coordinates, or database structures were copied or adapted.

## Purpose

The `ZoneStreaming` Gem is the first O3DE-side debug binding for AmandaCore's server-owned streaming preview state. It consumes the placeholder scene command contract produced by `Client/Game/AmandaCore.WorldClient` and represents the current zone, visible streaming cells, current cell, and nearest transition as debug-only volumes.

The Gem does not own traversal, zone authority, spawn state, quest state, validation, terrain streaming, prefab streaming, or asset prefetch. Those responsibilities remain in the Go world service and content package pipeline.

## Command API

The in-engine binding is exposed through `ZoneStreaming::IZoneStreamingDebugRequests`:

```cpp
auto* debugStreaming = ZoneStreaming::IZoneStreamingDebugRequests::Get();
if (debugStreaming)
{
    debugStreaming->ApplyPlaceholderSceneCommand(command);
}
```

`ApplyPlaceholderSceneCommand` accepts `ZoneStreaming::PlaceholderSceneCommand`, a C++ mirror of the AmandaCore placeholder command contract:

- `m_command`
- `m_zoneId`
- `m_mapId`
- `m_cellId`
- `m_transitionId`
- `m_targetZoneId`
- `m_streamingCellId`
- `m_displayName`
- `m_hint`
- `m_bounds`
- `m_position`
- `m_radius`
- `m_ready`
- `m_tags`

Supported command names:

- `CreateZoneBoundsVolume`
- `CreateStreamingCellVolume`
- `HideStreamingCellVolume`
- `HighlightCurrentCell`
- `ClearCurrentCellHighlight`
- `ShowTransitionAffordance`
- `ClearTransitionAffordance`

The same interface also exposes the local live bridge:

```cpp
auto* debugStreaming = ZoneStreaming::IZoneStreamingDebugRequests::Get();
if (debugStreaming)
{
    debugStreaming->SetCommandStreamPath("C:/Temp/amandacore/streaming.commands.jsonl");
    auto status = debugStreaming->GetCommandStreamBridgeStatus();
}
```

## Debug Visuals

`ZoneStreamingSystemComponent` stores the active debug scene state and draws it every tick through O3DE AuxGeom:

- zone bounds: blue shaded AABB
- visible streaming cells: cyan shaded AABBs
- highlighted current cell: gold shaded AABB
- transition affordance: sphere marker at the server-provided transition position
- ready transition: green marker
- non-ready transition hint: amber marker

The component also exposes read-only accessors for lightweight validation:

- `GetZoneVolume`
- `GetCellVolume`
- `GetHighlightedCell`
- `GetTransitionAffordance`
- `GetCommandStreamBridgeStatus`
- `GetVisibleCellCount`

## Live Command File Bridge

The C# world client can write a deterministic JSON Lines command stream while it is connected to the live world service:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:TEMP\amandacore\streaming.commands.jsonl"
```

The file contains one serialized placeholder scene command per line and no timestamps. The writer opens the file with read/write sharing so the O3DE Gem can tail it while commands are appended.

Point the O3DE Gem at the same file with either:

```powershell
$env:AMANDACORE_STREAMING_COMMAND_FILE="$env:TEMP\amandacore\streaming.commands.jsonl"
```

or a direct in-engine call to `SetCommandStreamPath`. `ZoneStreamingSystemComponent` polls the file, parses new lines, and feeds valid commands into `ApplyPlaceholderSceneCommand`. This is a local debug bridge only; the JSONL file is not a runtime authority source.

## Verification Flow

1. Start the local AmandaCore services with Dawnwake content enabled.
2. Set `AMANDACORE_STREAMING_COMMAND_FILE` for the O3DE process or call `SetCommandStreamPath` after the Gem activates.
3. Join the world with `Client/Game/AmandaCore.WorldClient`.
4. Select `--streaming-sink scene-commands` to print placeholder commands and add `--streaming-command-file` with the same path the Gem is watching.
5. Enable the `ZoneStreaming` Gem in an O3DE client build.
6. Add or load the `ZoneStreamingSystemComponent`.
7. Verify `zone_streaming.command_stream_active` appears after commands are read.
8. Verify the scene shows zone bounds, visible cells, the current-cell highlight, and transition affordance markers.

## Current Limits

- No real terrain, prefab, entity, or asset streaming is implemented.
- The live bridge is a local JSONL tailer, not production IPC or replication.
- Debug labels are not drawn yet; the first visualization uses shapes and deterministic colors.
- The world client remains a .NET prototype and does not take an O3DE SDK dependency.
- The binding is debug-only and should not be used for authoritative movement or transition decisions.

## Next Milestone

Replace the local JSONL bridge with a launcher/O3DE runtime bridge that feeds the same command API directly, then connect the exporter to AmandaCore-owned O3DE editor metadata or asset processor output while preserving the server-side validation boundary.

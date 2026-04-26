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
- `GetVisibleCellCount`

## Deterministic Command File

The C# world client can also write a deterministic JSON Lines command stream for local bridge testing:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:TEMP\amandacore\streaming.commands.jsonl"
```

The file contains one serialized placeholder scene command per line and no timestamps. It is a bridge/test artifact, not a runtime authority source. A later launcher or O3DE client integration can translate this stream, IPC, or direct network callbacks into `ApplyPlaceholderSceneCommand` calls without changing the world-service contract.

## Verification Flow

1. Start the local AmandaCore services with Dawnwake content enabled.
2. Join the world with `Client/Game/AmandaCore.WorldClient`.
3. Select `--streaming-sink scene-commands` to print placeholder commands, and optionally add `--streaming-command-file` to persist JSONL.
4. Enable the `ZoneStreaming` Gem in an O3DE client build.
5. Add or load the `ZoneStreamingSystemComponent`.
6. Feed placeholder commands into `IZoneStreamingDebugRequests::ApplyPlaceholderSceneCommand`.
7. Verify the scene shows zone bounds, visible cells, the current-cell highlight, and transition affordance markers.

## Current Limits

- No real terrain, prefab, entity, or asset streaming is implemented.
- The Gem does not parse JSONL directly yet; it consumes the stable C++ command representation.
- Debug labels are not drawn yet; the first visualization uses shapes and deterministic colors.
- The world client remains a .NET prototype and does not take an O3DE SDK dependency.
- The binding is debug-only and should not be used for authoritative movement or transition decisions.

## Next Milestone

Connect the AmandaCore launcher/O3DE client bridge so live world-client streaming callbacks feed `IZoneStreamingDebugRequests` in-engine, then replace placeholder authoring files with AmandaCore-owned O3DE editor metadata or asset processor output while preserving the server-side validation boundary.

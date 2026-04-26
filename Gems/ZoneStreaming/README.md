# ZoneStreaming Gem Placeholder Adapter

The `ZoneStreaming` Gem is still an O3DE scaffold. The local workspace does not currently build engine-linked Gem code, so the first runnable adapter lives in `Client/Game/AmandaCore.WorldClient` as `PlaceholderSceneStreamingAdapter`.

The adapter emits presentation-only commands that this Gem should consume when the O3DE integration is ready:

- `CreateZoneBoundsVolume`
- `CreateStreamingCellVolume`
- `HideStreamingCellVolume`
- `HighlightCurrentCell`
- `ClearCurrentCellHighlight`
- `ShowTransitionAffordance`
- `ClearTransitionAffordance`

These commands are derived from AmandaCore's server-owned `streaming` payload. They should create placeholder scene volumes and transition affordances only; traversal, entity visibility, combat, persistence, and zone transfer authority remain on the Go world service.

## Future Gem Binding

The first engine-backed implementation should:

- receive the placeholder command stream from the client networking layer
- map zone bounds to debug or editor-only volumes
- map streaming cells to simple placeholder entities
- highlight the current cell without affecting server authority
- show transition affordances near server-provided transition points
- keep asset prefetch and terrain streaming deferred until real O3DE content products exist

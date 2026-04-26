# amandacore World Client

This is the minimal runnable world client for milestone `0.1`.

It is intentionally thin and not an O3DE scene yet. Its job is to prove the live account-to-world loop:

- consume a join ticket
- connect to the world service
- move around a test zone
- disconnect and reconnect
- retain persisted state

It also hosts the first Dawnwake streaming preview hook. The client reads the world service `streaming` payload, builds a `ClientStreamingFrame`, computes the current cell from server-provided bounds, and emits changes through `IWorldStreamingPreviewSink`. `ConsoleWorldStreamingPreviewSink` prints those events, and `PlaceholderSceneStreamingAdapter` emits structured placeholder scene commands for the O3DE `ZoneStreaming` Gem debug binding.

Example:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085
```

Scene-command preview:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands
```

Deterministic command-file preview:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:TEMP\amandacore\streaming.commands.jsonl"
```

The command file is JSON Lines with one placeholder scene command per line and no timestamps. It is meant for local bridge/debug validation; the O3DE Gem consumes the same command contract through `ZoneStreaming::IZoneStreamingDebugRequests`.

Adapter checks:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient.Tests/AmandaCore.WorldClient.Tests.csproj
```

# amandacore World Client

This is the minimal runnable world client for the local gameplay proof.

It is intentionally thin and not an O3DE scene yet. Its job is to prove the live account-to-world loop:

- consume a join ticket
- connect to the world service
- move around a test zone
- select a hostile NPC target
- submit the server-authoritative `dev_basic_strike` ability
- display authoritative health, cooldown, aura, death, and kill-credit updates
- disconnect and reconnect
- retain persisted state

It also hosts the first Dawnwake streaming preview hook. The client reads the world service `streaming` payload, builds a `ClientStreamingFrame`, computes the current cell from server-provided bounds, and emits changes through `IWorldStreamingPreviewSink`. `ConsoleWorldStreamingPreviewSink` prints those events, and `PlaceholderSceneStreamingAdapter` emits structured placeholder scene commands for the O3DE `ZoneStreaming` Gem debug binding.

Example:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://127.0.0.1:8085
```

Controls:

- `W/A/S/D`: move
- `T`: move near and select the nearest visible hostile target
- `F`: submit `dev_basic_strike`
- `P`: poll authoritative world state
- `R`: reconnect
- `X`: disconnect
- `Q`: quit

The client never calculates combat results. It only sends target and ability intents, then renders the server response.

Automated diagnostic run:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --auto-combat-demo
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

Live O3DE debug bridge:

```powershell
$env:AMANDACORE_STREAMING_COMMAND_FILE="$env:TEMP\amandacore\streaming.commands.jsonl"
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085 --streaming-sink scene-commands --streaming-command-file "$env:AMANDACORE_STREAMING_COMMAND_FILE"
```

Run the O3DE client with the same `AMANDACORE_STREAMING_COMMAND_FILE` value. The `ZoneStreaming` Gem tails that file and renders debug volumes from the streamed commands.

Adapter checks:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient.Tests/AmandaCore.WorldClient.Tests.csproj
```

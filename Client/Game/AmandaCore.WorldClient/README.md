# amandacore World Client

This is the minimal runnable world client for milestone `0.1`.

It is intentionally thin and not an O3DE scene yet. Its job is to prove the live account-to-world loop:

- consume a join ticket
- connect to the world service
- move around a test zone
- disconnect and reconnect
- retain persisted state

It also hosts the first Dawnwake streaming preview hook. The client reads the world service `streaming` payload, builds a `ClientStreamingFrame`, computes the current cell from server-provided bounds, and emits changes through `IWorldStreamingPreviewSink`. `ConsoleWorldStreamingPreviewSink` prints those events today; a future O3DE adapter can bind the same callbacks to placeholder scene volumes or asset prefetch behavior.

Example:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085
```

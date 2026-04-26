# amandacore World Client

This is the minimal runnable world client for milestone `0.1`.

It is intentionally thin and not an O3DE scene yet. Its job is to prove the live account-to-world loop:

- consume a join ticket
- connect to the world service
- move around a test zone
- disconnect and reconnect
- retain persisted state

Example:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://127.0.0.1:8085
```

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

Example:

```powershell
dotnet run --project Client/Game/AmandaCore.WorldClient -- --join-ticket <ticket> --world-endpoint http://localhost:8085
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

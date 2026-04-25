# amandacore Game Client

This directory is the handoff point for the O3DE client project. The runtime gameplay seams are split across Gems in `Gems/`, and shared gameplay/platform contracts live in `Shared/AmandaCoreShared`.

The launcher authenticates against the backend, fetches realms and characters, and starts the game client with a short-lived world join ticket. The in-engine login and world bootstrap flow should use the same shared message and ticket contracts as the backend services.

## Runtime paths

- Preferred Milestone 2 client: `build/o3de-windows/bin/profile/amandacore.GameLauncher.exe`
- Fallback client during stabilization: `Client/Game/AmandaCore.WorldClient/bin/Debug/net8.0/AmandaCore.WorldClient.exe`

The launcher should prefer the O3DE `GameLauncher` build when it exists and fall back to the minimal `.NET` client only when the O3DE path is unavailable.

## O3DE launch handoff

The launcher starts the O3DE client with:

```text
--join-ticket <ticketId> --world-endpoint <endpoint>
```

Milestone 2 remains movement-only. It proves:

- `TestZone01` loads
- world connect/bootstrap happens after `client.level_ready`
- the player spawns from the authoritative world session
- WASD movement and third-person camera work
- disconnect/reconnect restores persisted position

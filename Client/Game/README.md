# amandacore Game Client

This directory is the handoff point for the O3DE client project. The runtime gameplay seams are split across Gems in `Gems/`, and shared gameplay/platform contracts live in `Shared/AmandaCoreShared`.

The launcher acts as a patch/build status and bootstrapper surface. It starts the O3DE game client without player credentials or a world join ticket. The game client owns account login, remembered-session restore, realm selection, character selection/creation, join-ticket request, and world entry.

## Runtime paths

- Preferred Milestone 2 client: `build/o3de-windows/bin/profile/amandacore.GameLauncher.exe`
- Fallback client during stabilization: `Client/Game/AmandaCore.WorldClient/bin/Debug/net8.0/AmandaCore.WorldClient.exe`

The launcher should prefer the O3DE `GameLauncher` build. The fallback `.NET` client is diagnostic-only and is not used for the normal in-client login flow.

## O3DE launch handoff

The launcher starts the O3DE client with safe service endpoint arguments and existing O3DE runtime arguments, for example:

```text
--auth-endpoint <endpoint> --realm-endpoint <endpoint> --character-endpoint <endpoint> --world-service-endpoint <endpoint>
```

The legacy `--join-ticket <ticketId> --world-endpoint <endpoint>` handoff remains a development/backward-compatible direct-world path only.

The O3DE client path remains the preferred playable slice. The fallback `.NET` client is diagnostic only, and now proves both movement and the server-authoritative combat command path:

- `TestZone01` loads
- world connect/bootstrap happens after `client.level_ready`
- the player spawns from the authoritative world session
- WASD movement and third-person camera work
- disconnect/reconnect restores persisted position
- diagnostic target selection sends the world target intent
- diagnostic Basic Strike input sends the world ability intent
- target health, NPC death, cooldowns, aura state, and kill credit are rendered from authoritative responses

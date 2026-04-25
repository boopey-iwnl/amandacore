# Playable Slice Acceptance

Use this checklist to verify the single-zone `sunset_frontier / west_approach` slice end to end.

## Launcher and World Entry

- Start the local stack with `Infra/dev/start-local.ps1`.
- Open the launcher and confirm it resolves `build/o3de-windows/bin/profile/amandacore.GameLauncher.exe`.
- Register or log in, load the local realm, create/select a character, and click `Join World`.
- Confirm `TestZone01` loads and the client logs `client.world_connect_started`, `client.level_ready`, `client.world_bootstrap_applied`, `client.world_connected`, and `client.player_spawned` in order.

## Grounded Third-Person Runtime

- Verify the player spawns near `(12, 12, 0)` and remains grounded while moving.
- Verify the third-person camera is active and the local HUD identifies `Sunset Frontier - West Approach`.
- Confirm the world shows the command-post marker at spawn, the route markers leading west, and the encounter marker around the hostile pocket.

## Quest, Targeting, and Combat

- At the command post, accept the field orders for `Contain the Ember Hounds`.
- Move along the marker trail toward the boulder choke and confirm exactly three visible hostile Ember Hounds are present.
- Use `Tab` to cycle targets in deterministic order and `LMB` to select the intended hostile under the cursor.
- Use `F`, `1`, and `2` to complete the fight loop and confirm target teardown, aggro, death, and respawn all remain stable.
- Confirm the quest progresses only from authoritative hostile kills and rewards exactly once on the third kill.

## Persistence

- Disconnect and reconnect; verify the player returns with persisted position, XP, currency, and quest state while transient combat state is cleared.
- Restart the local stack, log back in, and re-enter the world.
- Verify completed quest state and reward persistence survive the restart without granting a duplicate reward.

# Playable Slice Acceptance

Use this checklist to verify the Alpha 0.1 `stonewake_vale` slice end to end.

## Local Controls, Launcher, and World Entry

- Open `C:\Users\forwo\OneDrive\Desktop\Local Playable Slice Controls.lnk`.
- Confirm the GUI paths point at `C:\Users\forwo\OneDrive\Desktop\Code Project - Alpha Integration`.
- Start the local stack from the GUI and wait until all services are healthy.
- Open the launcher and confirm it resolves `build/o3de-windows/bin/profile/amandacore.GameLauncher.exe`.
- Register or log in, load the local realm, create/select a character, and click `Join World`.
- Confirm the client logs `client.world_connect_started`, `client.level_ready`, `client.world_bootstrap_applied`, `client.world_connected`, and `client.player_spawned` in order.

## Grounded Third-Person Runtime

- Verify the player spawns in Stonewake Vale and remains grounded while moving.
- Verify the third-person camera is active and the HUD identifies the current Stonewake slice.
- Confirm the first viewport clearly separates the quest giver, trainer/service area, road/path, training ring, and first hostile pocket.
- Confirm the center of the screen remains available for gameplay.

## NPC Interaction Lifecycle

- Right-click the first quest giver and confirm the quest/gossip window opens.
- Accept a quest and confirm the accept/gossip window closes or transitions to a valid active state.
- Confirm stale repeated accept requests do not occur after the quest is active.
- Walk out of interaction range and confirm the NPC window closes.
- Switch target or right-click another NPC and confirm the old NPC context closes cleanly.
- Press `Esc` with an NPC window open and confirm it closes the interaction before opening the main menu.
- Confirm trainer and vendor interactions still open and close correctly.

## Map, Targeting, and Combat

- Open the zone map and confirm the player marker, active objective, trainer/service, landmarks, and road/path are readable.
- Confirm map/minimap labels do not overlap into illegibility.
- Move to the first hostile/objective pocket and confirm hostiles are staged in a readable group.
- Use `Tab` to cycle targets and `LMB` to select the intended hostile under the cursor.
- Use `F`, `1`, and `2` to complete the fight loop and confirm target teardown, aggro, death, and respawn remain stable.
- Confirm the quest progresses only from authoritative hostile kills and rewards exactly once on turn-in.

## UI Layout

- Confirm player/target frames are top-left.
- Confirm minimap is top-right and objectives sit below/near it without overlap.
- Confirm chat is bottom-left and usable.
- Confirm primary and secondary action bars are bottom-center with compact spacing.
- Confirm right-side action bars are compact, aligned, and not cropped.
- Confirm the micro-menu is near the action bars and exposes only functional or honestly unavailable Alpha 0.1 systems.
- Confirm spellbook, character, quest log, map, bag, and settings windows open in usable positions.

## Persistence

- Disconnect and reconnect; verify the player returns with persisted position, XP, currency, inventory, and quest state while transient combat and interaction state is cleared.
- Restart the local stack, log back in, and re-enter the world.
- Verify completed quest state and reward persistence survive the restart without granting a duplicate reward.

# In-Client Login and Character Creation Runbook

## Setup

1. Check out `codex/ui-m2-login-character-creation`.
2. Start the local stack with `Infra/dev/start-local.ps1 -StartLauncher`.
3. Use the launcher path and the in-client path in the same service session when possible.

## Launcher Compatibility Path

1. Open the launcher.
2. Log in or register with local test credentials.
3. Load realms.
4. Load characters.
5. Create a character if needed.
6. Join world.
7. Confirm the world loads, movement works, camera works, and the UI M1 HUD remains intact.

## In-Client Path

1. Start the O3DE client without `--join-ticket` or `--world-endpoint`.
2. Confirm the login screen appears.
3. Log in with a local account.
4. Select a realm.
5. Select an existing character or open Create New.
6. In creation, adjust preview options, rotate/zoom/reset/randomize the preview, and enter a valid name.
7. Create the character.
8. Confirm the new character appears in character select.
9. Enter world.
10. Confirm world load, movement, camera, action bars, inventory, and chat.

## Known Limitations

Appearance customization is preview-only. The backend does not persist appearance fields in this milestone.

Character deletion remains disabled because there is no safe backend delete contract for this flow.

## Safety Checks

Before commit and push, verify:

- no secrets, logs, diagnostics, screenshots, zips, local DBs, runtime tickets, generated packages, Cache/build output, or temp files are staged
- no runtime/package reference to the local texture source folder
- no AddOns folder, addon API, Lua addon loading, plugin runtime, user-installed UI modules, or arbitrary UI script execution
- no copied external game assets, private-server code, addon code, or protected UI trade dress

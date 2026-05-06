# Settings, Keybinds, and Accessibility Local Test

## Setup

- Confirm the milestone branch is checked out.
- Confirm the worktree is clean before startup smoke.
- Start the local stack with the launcher.
- Confirm the launcher opens, client login works if enabled, join world works, visible world loads, and no crash occurs.

## Game Menu

- Press ESC with no gameplay panel open and confirm the Game Menu opens.
- Open a gameplay panel, press ESC, and confirm only the topmost panel closes.
- Confirm Return to Game closes the Game Menu.
- Confirm Settings/Options opens Interface.
- Confirm Keybinds opens Keybinds.
- Confirm UI Layout/Edit Mode opens UI Layout.
- Confirm Logout/Character Select is disabled with clear alpha-build status.
- Confirm Exit Game is safe.
- Confirm there is no AddOns button or tab.

## Options

- Open and close Settings.
- Confirm Interface, Keybinds, UI Layout, Accessibility, Audio, Video, Gameplay, Chat, Combat, Nameplates, and Objectives / Map categories are visible.
- Confirm functional controls apply immediately or after Apply.
- Confirm unsupported controls are clearly disabled.
- Confirm Defaults restores safe baseline values.
- Confirm local settings persist after restart when practical.

## Keybinds

- Rebind a UI panel key and confirm it works.
- Rebind an action-bar key and confirm action activation still uses the server-backed action path.
- Bind a duplicate key and confirm the warning appears and the previous binding is cleared.
- Clear a binding.
- Reset UI panel keybinds.
- Reset action-bar keybinds.
- Reset all keybinds.
- Start capture and press ESC; confirm capture cancels.
- Confirm chat and text fields block keybind capture.

## UI Layout

- Unlock frames and enable edit mode.
- Move chat, objectives, action bar, pack, and navigator where available.
- Save Custom profile.
- Reset HUD layout.
- Return to Default profile.
- Confirm action-bar and inventory drag/drop still work after edit mode.

## Accessibility And QoL

- Adjust UI scale and readability scale.
- Toggle floating combat text.
- Toggle nameplates.
- Toggle reduced UI motion.
- Toggle tooltip comparison and inspect an equippable item.
- Toggle chat frame, objective tracker, navigator/minimap, and action bar visibility.

## Regression

- Confirm action bars, inventory, spellbook, character panel, quest log, map, combat HUD, social/economy panels, chat, movement, and camera still work.
- Confirm no addon runtime, Lua loader, plugin UI runtime, user-installed UI module, AddOns folder, arbitrary UI scripting, or addon settings surface exists.
- Confirm no runtime or package dependency on machine-local texture source folders exists.

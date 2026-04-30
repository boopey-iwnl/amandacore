# Default Screen Contract

## Screen Regions

- Top-left: player frame, target frame, and future focus or pet frames.
- Top-right: navigator/minimap, zone label, clock or status line, and small utility indicators.
- Right side: objective tracker below the navigator.
- Bottom-left: chat frame with system and Say shell.
- Bottom-center: main action bar cluster and supported secondary action row.
- Bottom-right/right edge: bag, utility menu, and optional right-side action bars.
- Center/modal band: character, spellbook, quest log, map, trainer, vendor/market, social, settings, and confirmation panels.

## Window Rules

- Normal gameplay mode keeps default HUD frames locked.
- Built-in edit mode unlocks supported frame anchors and draws edit outlines.
- Escape closes the topmost gameplay panel first. If no gameplay panel is open, Escape opens the settings/menu shell.
- Panels opened from utility buttons must toggle reliably and must not leak clicks into world movement or targeting.
- Multiple RPG panels may remain open when they fit the modal band, but dense combinations should close one panel at a time through Escape.
- Bags open inward from the right-side utility area.
- Tooltips should appear above the item/action being inspected and should not cover critical action-bar labels when avoidable.
- Confirmation popups must be modal when an irreversible player action is requested.

## Action-Bar Rules

- Mouse click activates a usable action in normal gameplay mode.
- Keybinds activate usable actions in normal gameplay mode.
- Dragging a learned spellbook entry onto a slot assigns it.
- Dragging an action slot onto another action slot rearranges it.
- Holding Shift or enabling built-in UI edit mode allows click-to-place, click-to-move, and right-click clear behavior.
- Dragging an action to the clear target clears that slot.
- Clearing a slot does not remove the ability from the spellbook.
- High-resolution repo-side icons should render whenever available.
- Empty, cooldown-blocked, target-blocked, and resource-blocked slots must remain visually distinct.

## Inventory Rules

- The pack opens and closes through the utility button and keybind.
- Item icons use repo-side assets or the visible missing-icon fallback.
- Dragging items between slots rearranges inventory through the existing server-authoritative move path.
- Click-to-move remains as a fallback for item rearranging.
- Item tooltips should identify the item and stack count.
- Sort/search controls must not pretend to work before they are actually implemented.

## Chat Rules

- Enter focuses chat input.
- Escape exits chat focus without sending.
- Sending a message clears focus and returns movement input to gameplay.
- The Say channel shell remains readable.
- System messages remain visible in the chat scrollback.
- Chat must not block gameplay input when it is not focused.

## No-Addon Rule

The default screen is first-party AmandaCore UI only. It must not load user UI modules, execute arbitrary user scripts, mount runtime plugins, or expose an addon API.

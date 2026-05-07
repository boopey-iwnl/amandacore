# Panel Stacking And Input Contract

This contract defines the release-candidate behavior for first-party AmandaCore UI panels, input focus, and drag/drop.

## Panel Stack

- Opening a closable gameplay panel records it as the most recent panel.
- Reopening or focusing an already open panel moves it to the top of the stack.
- Closing a panel removes it from the stack and clears pending UI drag/move state.
- If stale stack entries exist, they are discarded before ESC closes a panel.
- NPC and modal surfaces close before ordinary gameplay panels.

## ESC Order

1. Cancel keybind capture.
2. Cancel chat focus without sending.
3. Let the pre-world front-end shell handle its own back/settings/help behavior.
4. Close active NPC/modal surfaces.
5. Close the most recently opened closable gameplay panel.
6. Open the Game Menu when no higher-priority surface owns ESC.

## Input Focus

- Chat owns typing only while focused.
- Keybind capture owns keys only while capture is active.
- Text fields block first-party keybind handling while active.
- UI clicks and drag/drop must consume UI interaction and must not trigger world target, movement, camera, or action commands.
- Movement and camera must recover when chat, keybind capture, modal prompts, or UI edit mode exits.

## Drag And Drop

- Spellbook drag payloads may assign only learned, action-bar-assignable abilities.
- Action-bar payloads must contain a valid source slot and may move only to a valid destination slot.
- Inventory payloads must contain a valid source slot and may move only to a valid destination slot.
- Equipment payloads must contain a non-empty equipment slot before unequip is requested.
- Inventory-to-equipment drop must validate source item existence and slot compatibility before requesting equip.
- Pending action/inventory/equipment drag or move state must be cleared when panels close, edit mode exits, settings reset, chat starts, or the client leaves the in-world state.

## Settings And Edit Mode

- UI edit mode is first-party only and cannot load external modules.
- Frame locks disable edit mode and clear pending UI moves.
- Resetting UI settings restores defaults, clears pending interactions, and keeps movement/camera/keybind recovery safe.

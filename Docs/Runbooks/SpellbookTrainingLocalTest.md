# Spellbook and Training Local Test

## Setup

1. Check out `codex/ui-m4-spellbook-talents-training`.
2. Start the local stack with the launcher.
3. Log in, select a realm, select or create a character, and join the world.

## Spellbook

1. Open Spellbook from the Spells utility button or bound key.
2. Verify All, Active, Passive, Utility, and Warrior tabs render.
3. Hover active and passive abilities and confirm readable tooltips.
4. Drag a learned active ability to an action slot.
5. Confirm passive and unlearned entries cannot be assigned.

## Action Bar

1. Click the assigned active ability with a valid target when required.
2. Press the matching keybind and confirm the same server-backed activation path.
3. Move a slot by drag/drop.
4. Hold SHIFT or use edit mode and clear a slot with the supported gesture.
5. Confirm cooldown/resource/target-disabled overlays show only when real session state supports them.

## Trainer

1. Target and interact with the Warrior trainer.
2. Confirm learned, available, unavailable, and cost/requirement states are readable.
3. Use Learn only on an offer marked available.
4. Confirm the Spellbook updates after a successful server response.

## Talents

1. Open Talents from the utility menu.
2. Confirm available or unavailable state is clear.
3. Select only entries enabled by the server.
4. Confirm no fake point spending is shown.

## Professions

1. Open Professions from the utility menu.
2. Target and interact with a profession trainer.
3. Confirm learned professions, catalog, trainer offers, and recipe summaries are readable.
4. Learn only when the server marks an offer available.
5. Confirm recipe summaries are read-only and no crafting button appears.

## Regression Checks

- Character panel still opens.
- Inventory opens and rearranges.
- Equipment drag still works where available.
- Chat focus/send/cancel still works.
- Movement and camera still work.
- No addon tab or addon runtime is present.
- No runtime, package, material, or manifest reference points to external local texture source folders.

# Character Panel Contract

## Tabs

The Character panel is a built-in AmandaCore MMORPG panel. It contains:

- Character: paper doll, preview, identity summary, equipment slots
- Stats: live health, resource, and server-derived stat summary
- Currency: live copper wallet breakdown
- Reputation: read-only shell or real runtime faction data when available
- Details: character identifiers and clear unavailable states

There is no AddOns tab.

## Character Tab

The Character tab displays only real runtime identity fields:

- display name
- archetype
- level
- zone
- lineage/origin only when a real payload field exists

Equipment slots are deterministic AmandaCore slot IDs:

- `head`
- `shoulders`
- `chest`
- `hands`
- `waist`
- `legs`
- `feet`
- `main_hand`
- `off_hand`
- `ranged_or_focus`
- `accessory_1`
- `accessory_2`
- `trinket_1`
- `trinket_2`
- `cloak_or_back`

Empty slots must remain visible. Equipped items use server item metadata and icon fallback routing. Dragging from the pack to a compatible slot requests equip. Dragging from an equipment slot to the pack requests unequip.

## Stats Tab

The Stats tab displays real runtime values only:

- health
- resource
- strength
- stamina
- armor
- attack power
- armor reduction

Unsupported future stats are hidden or clearly unavailable. The UI does not define combat formulas.

## Currency Tab

The Currency tab displays live wallet values only:

- total formatted currency
- gold/silver/copper breakdown

No unsupported auction, mail, or alternate currency systems are shown as working features.

## Reputation Tab

Reputation remains a read-only shell unless real runtime faction-standing data exists. The empty state is:

`No faction standings available yet.`

Do not invent faction names, ranks, or progress.

## Tooltip And Comparison

Item tooltips can show:

- item name
- item type/subtype
- compatible equipment slot
- quality color
- required archetype
- required level
- sell value
- stat lines
- description

When hovering an equip-compatible inventory item and a matching equipped item exists, the tooltip shows the equipped comparison. Missing comparison data is not fabricated.

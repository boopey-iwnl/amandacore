# Equipment And Stats Contract

## Server Authority

Inventory, equipment, stats, and currency are server-authoritative. The client previews and requests changes, but the world service validates and persists every equip/unequip mutation.

## Equipment Payload

`equipment.slots[]` is additive and compatible with older slot payloads. Each row includes:

- `slot`
- `itemId`
- `displayName`
- `itemType`
- `itemSubtype`
- `quality`
- `iconKind`
- `description`
- `equipSlot`
- `requiredArchetype`
- `requiredLevel`
- `sellPriceCopper`
- `strength`
- `stamina`
- `armor`

Empty slots keep `slot` and omit item metadata.

## Inventory Payload

`inventory.slots[]` keeps the existing `slotIndex`, `itemId`, `displayName`, and `stackCount` fields and adds the same optional metadata used by equipment rows.

## Equip

`POST /v1/world/inventory/equip` accepts:

- `worldSessionToken`
- `slotIndex`

The server validates:

- source inventory slot exists and has exactly one item
- item is equip-compatible
- required archetype/class compatibility remains satisfied
- required level is satisfied
- target equipment slot exists

The server swaps any previous equipment item into the source inventory slot, persists inventory/equipment/currency together, refreshes derived stats, and returns a world session payload.

## Unequip

`POST /v1/world/inventory/unequip` accepts:

- `worldSessionToken`
- `slot`

The server validates:

- equipment slot exists
- equipment slot is occupied
- at least one inventory slot is empty

The server moves the item into the first empty inventory slot, clears the equipment slot, persists inventory/equipment/currency together, refreshes derived stats, and returns a world session payload. Full bags reject the request without duplicating or removing the item.

## Stats

The visible stats are derived from existing runtime rules:

- strength
- stamina
- armor
- attack power
- armor reduction
- max health through the existing world stat application path

M3 does not add final combat formula design. Unsupported stats are not shown as active mechanics.

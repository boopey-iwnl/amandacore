# Character Panel Milestone 3

## Current Data Audit

The in-world session payload now carries the Character panel's live data:

- character identity: `characterId`, `displayName`, `archetypeId`, `realmId`, `zoneId`, `level`
- vitals and stats: `health`, `maxHealth`, `resource`, `maxResource`, `resourceName`, and `stats`
- currency: `currencyCopper` plus `currency.gold`, `currency.silver`, and `currency.copper`
- inventory: `inventory.slotCount` and `inventory.slots[]`
- equipment: `equipment.slots[]`

Inventory and equipment item rows expose additive tooltip metadata when the server has a known item definition:

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

The backend already persisted inventory, currency, equipment, talents, and derived stats before this milestone. M3 expands the visible slot model and exposes more item metadata rather than moving authority into the UI.

## Implementation Summary

The built-in Character panel is a first-party ImGui/O3DE panel with these tabs:

- Character
- Stats
- Currency
- Reputation
- Details

The Character tab renders a paper doll around a stylized preview pane. Empty equipment slots remain visible. Equipped item slots use repo-local icon routing or the existing missing-icon/procedural fallback. Inventory items can be dragged onto compatible equipment slots, and equipped items can be dragged back to the pack to request unequip.

Equip and unequip remain server-authoritative:

- bag-to-slot equip calls `POST /v1/world/inventory/equip`
- slot-to-bag unequip calls `POST /v1/world/inventory/unequip`
- full bags reject unequip
- incompatible drops are rejected before mutation
- the server persists inventory/equipment together and returns a refreshed world session

## Unsupported Or Placeholder Areas

Reputation is intentionally a clear empty shell unless real runtime faction-standing data is added. It does not invent faction names, ranks, or progress.

Lineage/origin is shown as unavailable until the runtime exposes real character-origin data.

The preview pane is stylized UI presentation. It does not claim saved 3D appearance state beyond the current live world model path.

## Policy Notes

M3 does not add addon support, Lua loading, an AddOns folder, plugin runtime, user-installed UI modules, or arbitrary UI script execution.

The local Downloads texture source folder was treated as read-only source material. M3 uses existing repo-local icons and procedural UI styling by default. No runtime, package, manifest, material, or docs-as-config path points to Downloads.

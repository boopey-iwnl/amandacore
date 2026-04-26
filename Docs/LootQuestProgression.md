# Loot, Kill Credit, and Quest Progression

AmandaCore now has an original server-authoritative progression skeleton for the first kill -> loot -> quest completion loop. The implementation is intentionally small: it proves reusable runtime ownership, objective tracking, reward grants, and state/event surfaces without adding a full economy, inventory UI, quest editor, or external content pipeline.

## Runtime Flow

The current dev loop is:

1. A character accepts `dev_first_hunt`.
2. The character defeats `dev_isle_stalker`.
3. The world runtime records kill credit for the killing character.
4. The death flow creates a character-owned loot container from `dev_isle_stalker_loot`.
5. The character inspects and claims the loot container.
6. Claimed items are granted through the authoritative inventory mutation path.
7. The granted `dev_stalker_fang` updates the quest objective graph.
8. The quest becomes ready to complete.
9. Server-side completion grants reward items and persists progression state.

The real client does not need inventory or quest UI yet. The backend response now exposes loot containers, recent domain events, and recent state diffs so future UI can render quest offers, objective progress, loot windows, inventory deltas, quest completion, and reward grants.

## Item Catalog

Dev item definitions live in the world item catalog and use AmandaCore-owned IDs, names, and categories:

- `dev_glimmer_shard`: crafting material, common, max stack 99.
- `dev_stalker_fang`: quest item, common, max stack 20.
- `dev_field_ration`: consumable, common, max stack 10.
- `dev_copper_token`: currency token, common, max stack 999.

The catalog exposes item identity, display name, optional description, kind, quality, max stack, and tags. Existing item response paths continue to use the older item type fields for compatibility, with kind normalized from type when needed.

## Inventory Placeholder

Inventory remains a server-owned character state surface backed by `platform.CharacterInventorySlot`. The new mutation path validates item IDs against the item catalog, respects each item's max stack, uses the platform inventory capacity, rejects full inventories deterministically, emits inventory events and deltas, and persists through the existing character progression writer when a FileStore is available.

Current full-inventory behavior is all-or-nothing for multi-item grants. Loot claims first validate every grant against a candidate inventory; if any item cannot fit, the claim is rejected and the container remains unclaimed.

## Loot Tables and Containers

Loot tables are AmandaCore-native runtime definitions. `dev_isle_stalker_loot` contains only original placeholder items:

- Guaranteed `dev_stalker_fang` quantity 1.
- Optional `dev_glimmer_shard` quantity 1-2.
- Optional `dev_field_ration` quantity 1.
- Guaranteed `dev_copper_token` quantity 1-3.

Loot generation accepts an injectable roll source, so tests can use deterministic seeds. Runtime NPC death creates a loot container owned by the killing character, bound to the source entity, archetype, zone, instance, position, item list, and expiry time.

Loot interactions validate:

- live connected session
- alive character
- existing unexpired container
- unclaimed container
- owner eligibility
- matching zone and instance
- interaction range

Invalid interactions return stable rejection reasons such as `LootMissing`, `LootExpired`, `LootAlreadyClaimed`, `NotLootOwner`, `OutOfRange`, `InventoryFull`, `SessionInvalid`, `CharacterDead`, and `InvalidState`.

## Kill Credit Ledger

NPC death records kill credit with character ID, source entity ID, NPC archetype ID, zone ID, optional instance ID, tick time, and reason. The first reason is `killing_blow`. The ledger is intentionally local to the world runtime for this milestone, but quest-relevant progress is persisted through the character quest log.

Future group, party, and raid credit rules should extend the recipient selection step without changing the quest objective event contract.

## Quest Catalog and Objective Graph

The dev quest `dev_first_hunt` is a direct-accept test quest:

- Display name: First Hunt.
- Summary: Help secure the nearby path by defeating an Isle Stalker and recovering a fang.
- Objective graph:
  - `node_kill_stalker`: kill one `dev_isle_stalker`.
  - `node_collect_fang`: collect one `dev_stalker_fang`; depends on `node_kill_stalker`.
- Rewards:
  - `dev_copper_token` quantity 5.
  - `dev_glimmer_shard` quantity 1.

Objective nodes are active only when their dependencies are complete. Progress updates are event-driven through kill-credit and item-grant events. Quest completion is authoritative: the server validates graph terminal nodes before granting rewards.

The direct accept path is a dev quest source placeholder for loadsim and tests. Client-side quest provider UI is future work.

## Events and State Diffs

The runtime emits stable domain events for catalog loading, inventory grants, loot rolls and claims, kill credit, quest lifecycle, and the quest load simulator. State diffs cover entity spawn/health/combat/death plus inventory, loot container, loot claim, quest accepted, quest progress, objective completed, ready, completed, and reward deltas.

Recent `domainEvents` and `stateDiffs` are included in world responses as a temporary UI integration surface. A future client protocol can replace this with a subscribed event stream.

## Persistence

When a `FileStore` is attached, inventory state, quest progress, completed/rewarded quest state, learned abilities, action bars, experience, and copper totals persist through the existing character progression writer. Tests cover the dev quest path against a real temporary FileStore.

When the world runtime is constructed without a store, progression remains in memory. This is used by fast package tests and should not be treated as production behavior.

## Loadsim

The new loadsim scenario is:

```powershell
go run ./Services/cmd/loadsim --clients 1 --duration 30s --cmd-rate 2 --scenario quest-basic
go run ./Services/cmd/loadsim --clients 25 --duration 30s --cmd-rate 4 --scenario zone-handoff-basic --transition-loops 3 --shards 2 --queue-capacity 64
```

It creates temporary test accounts and characters, accepts `dev_first_hunt`, defeats the dev hostile NPC through the server combat path, claims loot, completes the quest, and prints counts for accepted quests, NPC kills, kill credits, loot containers, claims, inventory grants, objective updates, ready quests, completed quests, rewards, rejected commands, tick duration, queue depth, and errors.

The zone handoff scenario is documented separately in `Docs/ZoneHandoffShardCoordinator.md`.

## Current Limitations

- Inventory is still a logical slot placeholder, not a full bag/equipment/economy model.
- Loot ownership supports one owner; eligible participant lists are future work.
- Kill credit records killing-blow credit and does not yet implement group credit rules for the dev quest.
- Quest graph support is intentionally minimal: dependencies and terminal completion exist, but optional branches and repeatable quests do not.
- Loot tables and quests are in runtime dev definitions until the next content package loader milestone.
- Recent event/diff response payloads are a temporary integration surface, not a final streaming protocol.

## Next Milestone

The next recommended milestone is the zone content package loader with handoff gate integration:

- original AmandaCore content manifest format
- zone definition loading
- content-authored transition and handoff gates
- spawn point loading
- NPC archetype references
- loot table references
- quest provider references
- validation of authored content
- compiled/runtime content package boundary
- tests and loadsim coverage

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore/AzerothCore were used only as high-level architectural reference.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, loot tables, quest tables, item IDs, creature IDs, spell IDs, quest text, reward tables, or database structures were copied or adapted.

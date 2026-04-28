# World Loop Combat, Quest, And Loot

## Purpose

Milestone 5 extends the Stonewake single-writer loop from movement/session ownership into the current gameplay mutation path: target selection, combat resolution, threat updates, kill credit, quest progress, loot generation, loot claim, and quest reward claim.

The goal is not full MMO combat parity. The goal is to make AmandaCore's playable Stonewake path deterministic, replayable, and duplicate-resistant while preserving the existing HTTP/JSON contract used by the launcher, O3DE client, and fallback client.

## Current Combat Mutation Paths

Before this milestone, world HTTP handlers submitted command closures to the Stonewake loop, but most combat state still changed in legacy helpers under `worldServer.mutex`.

The active paths are:

- `/v1/world/target` validates and updates `CurrentTargetID`.
- `/v1/world/attack/auto` toggles `AutoAttackActive`.
- `/v1/world/attack/ability` validates learned abilities, cooldowns, target state, and applies effects.
- `/v1/world/state` advances the world, including aura ticks, auto attacks, mob AI, damage, death, and respawn.
- `applyDamageToMobLocked` resolves mob health, death, kill credit, quest progress, and loot container creation.

Milestone 5 keeps these proven domain helpers but formalizes the command names, replay records, threat state, and loot/reward routing around the loop authority boundary.

## Current Quest Progression Paths

Quest accept, complete, tracking, kill-credit, and reward helpers persist through the current file-backed runtime store by default. SQL transactional character-state APIs remain available for later cutover but are not the default runtime backend.

Kill credit is currently generated after authoritative mob death resolution. Quest counters are updated only when the killed mob matches an active quest objective and party-credit rules allow it.

## Current Loot And Reward Paths

Mob death can create an owner-bound loot container. Loot inspect and claim now enter the Stonewake loop before invoking existing loot validation and inventory grant helpers.

Quest rewards continue to use the existing progression persistence helper, but the HTTP completion path is now represented as a `ClaimQuestReward` loop command. Reward retries are rejected or replayed safely by the current quest state checks, and inventory grants happen before persisted reward state is marked complete.

## Current Threat And Targeting Behavior

Stonewake mobs now maintain a simple AmandaCore threat table keyed by character ID. Damage adds threat. The highest threat target is selected deterministically with character ID as the tie-breaker. Threat clears on death, leash reset, respawn, or player removal from aggro.

This is intentionally simple. It does not add pathfinding, roles, taunts, faction tables, or complex AI.

## World-Loop Ownership Target

Loop command coverage now includes:

- combat commands: `SelectTarget`, `ClearTarget`, `StartAutoAttack`, `StopAutoAttack`, `UseAbility`, `CancelCast`, `ApplyDamage`, `ApplyHeal`, `ResolveDeath`, `RespawnNpc`, `ScheduleRespawn`, `RequestCombatSnapshot`
- threat commands: `AddThreat`, `DecayThreat`, `ResetThreat`, `SelectNpcTarget`, `ClearThreatOnDeath`, `ClearThreatOnLeash`
- quest commands: `AcceptQuest`, `AbandonQuest`, `ProgressQuestObjective`, `CompleteQuest`, `ClaimQuestReward`
- loot and reward commands: `GenerateLoot`, `OpenLoot`, `ClaimLootItem`, `ClaimCurrencyReward`, `CloseLoot`, `ApplyQuestReward`, `ApplyKillLoot`, `ApplyCurrencyDelta`, `ApplyItemGrant`

The HTTP adapter still returns the existing response shape. Internally, gameplay operations now have explicit loop command names and replay payloads instead of being anonymous locked mutations.

## Transaction And Persistence Boundaries

The runtime path remains file-store compatible:

- movement and disconnect persist position through the current store
- inventory and quest progression persist through existing aggregate helpers
- loot claim grants inventory before marking a loot container claimed
- quest reward grants inventory/currency before persisting reward completion
- duplicate reward and loot attempts cannot grant a second reward because current quest and loot state reject already-claimed/already-rewarded state

SQL transactional repositories from Milestone 3 remain the later cutover target. This milestone does not flip the default runtime store.

## Idempotency Strategy

At the loop package level, reward and loot commands accept mutation keys and replay the successful result when the same key is submitted again. At the current HTTP service level, duplicate loot/quest reward attempts are prevented by authoritative in-memory and persisted state:

- claimed loot containers reject a second claim
- quest rewards with `RewardGrantedAt` reject a second reward
- inventory grant failure prevents the claimed/rewarded marker from being set

Milestone 6/7 can extend this into client-provided idempotency keys for public endpoints.

## Determinism And Replay Model

The loop records gameplay commands in accepted order with logical ticks. Replay support now handles combat damage, threat, quest progress, quest rewards, loot generation, loot claims, inventory grants, and currency deltas. Tests replay the same command stream from an empty initial snapshot and assert the same final gameplay state.

## Client Compatibility Model

No route names or JSON request shapes changed. Existing polling clients still consume the full world response from `buildResponse`. The compact loop snapshot now carries additional combat, threat, loot, quest, inventory, and currency state for replay and future replication work.

## Known Limitations

- Combat ticks still advance when HTTP commands or polls enter the world service.
- The file store remains the runtime persistence path.
- Loot is single-owner/personal for this milestone.
- Party loot, auctions, guilds, mail, broad social/economy transactionality, and full push replication remain later work.
- The loop's compact snapshot is not yet the Milestone 6 client-state convergence contract.

## Non-Goals

- no binary opcode transport
- no push replication
- no full multi-zone continent loop
- no full SQL runtime cutover
- no file-store removal
- no auctions, guilds, mail, or broad economy expansion
- no external MMO combat formulas, schema layouts, packet layouts, command names, IDs, scripts, or module structures

## Clean-Room Notes

The command names, threat rules, loot/reward semantics, replay behavior, tests, and documentation are AmandaCore-original. Public MMO-server architecture informed only the broad goals of single-writer gameplay authority, deterministic replay, and duplicate-resistant reward handling.

## Risks For Milestone 6

- Polling compatibility still returns a richer service response than the compact loop snapshot.
- Future replication must define exact snapshot/delta contracts for combat, loot, quest, inventory, and action-bar state.
- Persistence failures inside loop command closures can still block the shard queue.
- Client convergence tests need to compare repeated polling/reconnect snapshots against the authoritative loop mirror.

## Milestone 8 Content Boundary Note

The Milestone 8 content compiler adds validated quest, NPC, loot, ability, vendor, trainer, dialogue, and hook catalogs to the existing package loader. Combat, loot, and quest loop code should prefer `RuntimeContentRegistry` catalog interfaces for new package-backed behavior, but this document still describes the current Stonewake runtime path. Full package-backed gameplay cutover is intentionally separate from the compiler foundation.

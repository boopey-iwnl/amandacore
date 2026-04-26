# AmandaCore NPC Combat Loop

## Scope

This milestone adds the first server-authoritative hostile NPC loop:

Login/Register -> Select/Create Character -> Join World -> Spawn Player -> Spawn Hostile NPC -> Move Near NPC -> Select NPC -> Use Basic Strike -> Server Applies Damage -> NPC Dies -> Kill Credit Is Awarded -> NPC Respawns.

The real client can continue to use the existing world state response. The server now also emits protocol-agnostic domain events and state diffs for combat state changes.

## NPC Archetype Model

The dev hostile NPC is original AmandaCore placeholder content:

- ArchetypeID: `dev_isle_stalker`
- DisplayName: `Isle Stalker`
- Level: `1`
- MaxHealth: `30`
- Disposition: `Hostile`
- AttackRange: `2.5`
- AggroRange: `8.0`
- LeashRange: `18.0`
- BaseDamage: `3`
- AttackInterval: `1500ms`
- RespawnDelay: `10s`

Runtime NPC state tracks entity ID, archetype ID, spawn point ID, position, health, disposition, target, last damaging entity, death tick, and respawn tick.

## Spawn Lifecycle

World startup loads authored spawn definitions into the world server's entity registry maps. Each spawned NPC emits:

- `npc.spawn_point.loaded`
- `npc.spawned`
- `world.entity.spawned`
- `EntitySpawnDelta`

On death, the NPC becomes dead and untargetable, emits death/removal deltas, and schedules respawn. When the respawn tick is reached, the NPC returns to its spawn point at full health and emits a fresh spawn delta.

## Target Selection

Players select targets through the existing world target command. Hostile NPC targets must exist in the same world context, be alive, be targetable, and be inside target-selection range.

Successful hostile target selection emits `combat.target.selected` and `TargetSelectionDelta`. Missing, dead, untargetable, or out-of-range targets emit `combat.target.rejected`.

## Basic Strike

`dev_basic_strike` is the milestone ability:

- DisplayName: `Basic Strike`
- Range: `3.0`
- Cooldown: `1500ms`
- Damage: `10`
- RequiresTarget: `true`
- TargetDisposition: `Hostile`

Damage is fixed and deterministic. The client never supplies damage. The server validates player state, target state, range, and cooldown before applying the result.

## Hostile NPC Behavior

Hostile NPC AI is deliberately minimal:

- Detect connected, alive players within aggro range.
- Enter combat with the closest valid target.
- Move toward targets outside attack range.
- Attack on the configured interval when in range.
- Leash if the target moves beyond leash range from the spawn point.
- Become temporarily untargetable while returning home, then reset to full health at spawn.

Movement is a deterministic step toward the target each tick using the NPC movement speed. Pathfinding and avoidance are future work.

## Health, Death, Respawn

Damage clamps at zero. Dead NPCs are untargetable and cannot be damaged by ability commands. Player death is minimal: the player is marked dead, combat state is reset, and a player death event/delta is emitted. Full player respawn flow remains future work.

NPC death emits:

- `combat.target_defeated`
- `entity.died`
- `npc.died`
- `world.entity.removed`
- `EntityDeathDelta`
- `npc.respawn_scheduled`

## Kill Credit Placeholder

Killing an NPC records a per-character kill credit keyed by NPC archetype and persists it to the local file store when available. This is not a quest system replacement; it is the integration point future quest objective tracking should consume.

Events:

- `progression.kill_credit_awarded`
- `progression.kill_credit_persisted`
- `ProgressionDelta`

## Loadsim

Run the local combat simulator from the Services module:

```powershell
Push-Location Services
go run ./cmd/loadsim --clients 5 --duration 10s --cmd-rate 2 --scenario combat-basic
Pop-Location
```

The report includes simulated clients, NPC spawns, command counts, accepted/rejected combat commands, damage events, NPC deaths, kill credits, respawns, tick duration, queue depth placeholder, and errors.

## Limitations And Next Steps

- Only one dev hostile archetype is added.
- Basic Strike is fixed damage with no crit, dodge, miss, mitigation, cast time, or effect graph.
- NPC movement is direct step movement, not navmesh pathing.
- Player death has no full respawn flow.
- Loot tables are intentionally not implemented.
- Quest objective integration consumes the new kill-credit boundary later.

Next recommended milestone: ability/effect/aura skeleton.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data. TrinityCore/AzerothCore were used only as high-level architectural reference. No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, assets, formulas, or stat tables were copied or adapted.

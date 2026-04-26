# Server Interaction Pipeline

## Current Pipeline

The Dawnwake skeleton keeps client-facing protocol concerns separate from server authority. Runtime tests and loadsim drive canonical server operations directly:

```text
client/session intent -> world command -> owning zone runtime -> validation -> state mutation -> domain event/state diff -> visibility evaluation
```

For zone transfer, the sequence is:

```text
movement delta -> boundary check -> transition request -> topology validation -> source zone exit -> destination zone enter -> route update -> visibility delta
```

## Multi-zone Routing

`WorldRuntime` and `ContinentRuntime` track the character-to-zone ownership index. Commands are routed to the current owner. A character is active in one zone runtime at a time.

Future protocol adapters and session gateway work should submit canonical commands into this same routing layer. The adapter should not own topology or transition decisions.

## Visibility Output

Visibility is emitted as internal state diff data for future networking and O3DE streaming:

- same-zone entities inside radius enter visibility
- same-zone entities outside radius exit visibility
- zone transfers reset the previous visibility set
- nearby transition gates emit adjacent-zone streaming hints

The implementation uses a naive scan for the milestone. A spatial partition can replace it behind the same visibility query and delta contract.

## Canonical Flow

The current server interaction path is:

1. The client sends an intent to an HTTP world endpoint.
2. The world server validates session attachment from the world session token.
3. The world advances authoritative simulation time.
4. The command handler validates player state, target state, range, cooldown, and world context.
5. The world mutates authoritative state under the world mutex.
6. The server emits domain events and state diffs.
7. The response returns the latest authoritative player state, visible entities, combat fields, kill credits, state diffs, and recent domain events.

## Combat Commands

Target selection uses the existing target endpoint and now emits `combat.target.selected` or `combat.target.rejected`.

Ability use goes through the existing ability endpoint. `dev_basic_strike` remains the original fixed-damage hostile-NPC strike, but resolution now flows through the AmandaCore effect resolver.

The resolver keeps the command protocol semantic and server-authoritative:

```text
ability request -> ability definition -> target validation -> cooldown/category gate -> cast/channel gate -> effect resolver -> aura lifecycle -> state diffs
```

Current supported effect primitives:

- direct damage
- healing
- aura application
- periodic aura ticks
- cast/channel timing placeholders
- per-ability cooldowns
- shared cooldown categories

The first authored aura test path is `dev_stalker_pressure -> dev_pressure_mark`, defined in the AmandaCore dev content package. The server applies the aura, ticks deterministic damage, expires the aura, and exposes aura state in the world response.

## State Diffs

The world response includes protocol-neutral deltas for future clients:

- `EntitySpawnDelta`
- `EntityHealthDelta`
- `EntityCombatStateDelta`
- `TargetSelectionDelta`
- `AbilityResultDelta`
- `AuraStateDelta`
- `CooldownDelta`
- `CastStateDelta`
- `EntityDeathDelta`
- `ProgressionDelta`

These names are AmandaCore state contracts, not wire packet names. Transport adapters should map them into whatever client protocol is current without changing simulation authority.

## Diagnostic Client Consumption

`Client/Game/AmandaCore.WorldClient` now exercises this contract from the client side. It sends only:

- movement deltas
- target-selection intent
- ability-use intent
- state poll requests

The client then displays:

- player and target health
- target aura state
- action bar cooldowns
- cast state
- recent combat domain events
- recent state diffs
- kill credits

This is intentionally a thin diagnostic UI. It proves the command/state/event shape the O3DE HUD can later consume without moving combat authority into the client.

## O3DE HUD Consumption

The O3DE `NetClient` now parses the protocol-neutral combat response fields from the world service: auras, kill credits, domain events, state diffs, target state, cooldowns, cast state, action bar slots, and visible entity health.

`UiClient` consumes those fields as presentation data only. The HUD can display target selection, server-resolved ability outcomes, cooldown overlays, active auras, death state, kill credit, and recent authoritative event/diff summaries, but gameplay decisions stay in the world service.

The client-side observability contract includes:

- `client.combat_hud_ready`
- `client.target_frame_updated`
- `client.action_bar_cooldown_rendered`
- `client.kill_credit_displayed`
- `client.combat_hud_state_applied`

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

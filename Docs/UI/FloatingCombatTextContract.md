# Floating Combat Text Contract

Floating combat feedback is a first-party HUD feedback layer sourced only from authoritative combat events.

## Data Source

- Combat pulses are derived from `domainEvents` and `stateDiffs`.
- Eligible event types are the existing combat, NPC, ability, aura, cooldown, entity health/death, target selection, ability result, aura, cooldown, cast, and progression delta families.

## Behavior

- Feedback is screen-space, non-interactive, and rate-limited.
- Pulses fade automatically and do not block movement, camera, targeting, chat, panels, or action bars.
- Text uses AmandaCore event summaries or event type names from the current payload.
- The combat feed remains the durable log; pulses are transient attention cues.

## Unsupported States

- World-space floating damage numbers are future work.
- No copied external combat text, fonts, animation timing, phrasing, or addon behavior is used.
- The client does not infer damage, healing, miss, dodge, resist, or threat events when the server has not reported them.

No addon integration is allowed.

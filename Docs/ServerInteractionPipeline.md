# Server Interaction Pipeline

## Current Pipeline

The Dawnwake skeleton keeps client-facing protocol concerns separate from server authority. Runtime tests and loadsim drive canonical server operations directly:

```text
client/session intent -> world command -> owning zone runtime -> validation -> state mutation -> domain event/state diff -> visibility evaluation
```

For future production zone transfer, the intended sequence is:

```text
movement delta -> boundary check -> transition request -> topology validation -> source zone exit -> destination zone enter -> route update -> visibility delta
```

## Multi-zone Routing

The current Dawnwake milestone validates package-authored transition topology and exposes transition landmarks through world content activation. Production character handoff between package-authored zones is still future work. A character should remain active in one authoritative zone owner at a time when that handoff layer is implemented.

Future protocol adapters and session gateway work should submit canonical commands into this same routing layer. The adapter should not own topology or transition decisions.

## Visibility Output

Visibility is emitted as internal state diff data for future networking and O3DE streaming:

- same-zone entities inside radius enter visibility
- same-zone entities outside radius exit visibility
- zone transfers reset the previous visibility set
- nearby transition gates emit adjacent-zone streaming hints

The implementation uses a naive scan for the milestone. A spatial partition can replace it behind the same visibility query and delta contract.

## Clean-room reference boundary

This implementation uses original AmandaCore code and data.

TrinityCore and AzerothCore were used only as high-level architectural reference.

Dawnwake Isles is AmandaCore-original world content.

No source code, SQL, packet layouts, opcodes, command names, schemas, content IDs, scripts, scripting APIs, assets, formulas, map formats, area tables, zone tables, spawn schemas, coordinates, quest tables, item tables, creature tables, spell tables, aura tables, reward schemas, or database structures were copied or adapted.

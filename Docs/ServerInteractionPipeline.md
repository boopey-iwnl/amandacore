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

## Queue And Backpressure Skeleton

Each active `ZoneRuntime` can now own an in-memory command queue. The queue is FIFO, can be bounded by capacity, and reports enqueue, dequeue, backpressure, current depth, and max depth counters. Capacity `0` is unbounded; positive capacity rejects new commands once the zone queue is full.

This is a scheduling and observability boundary, not a production worker pool. Future work can replace the queue internals with a shard-local work loop while keeping command routing and backpressure reporting stable.

## Shard Assignment Skeleton

`ContinentRuntime.AssignZonesToShards` binds active zones to shard IDs. The current policy supports multiple zones per shard or one zone per shard. Transfers validate that both source and destination zones are bound to shard owners, while character ownership remains zone-scoped.

The initial implementation is single-process. It prepares the code for a future distributed shard coordinator without introducing cluster dependencies into the current milestone.

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

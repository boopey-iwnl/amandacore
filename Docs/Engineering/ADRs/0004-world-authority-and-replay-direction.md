# 0004 - World Authority and Replay Direction

Status: Accepted for Milestone 1

Date: 2026-04-26

## Context

Current world behavior is functional through HTTP handlers, in-process state, command queues, and tests. Later milestones need stronger authority guarantees before expanding combat, loot, quests, replication, and persistence load.

## Decision

AmandaCore's target world model is single-writer authority per active zone or instance. Client-facing transports submit intent, adapters translate intent into AmandaCore commands, and the owning world execution context validates and mutates state. Deterministic replay is a design requirement: command streams, input state, tick timing rules, and content package versions must be sufficient to reproduce final authoritative state for test scenarios.

Milestone 1 records this direction only. It must not implement the full Stonewake shard loop or rewrite gameplay handlers.

## Consequences

- Future gameplay mutations should converge on command-queue execution rather than ad hoc shared mutation.
- Replay metadata and content package versions become part of contract thinking.
- Clients remain presentation and intent producers.
- Milestone 1 validation can check contract documentation while later milestones add deterministic replay tests.

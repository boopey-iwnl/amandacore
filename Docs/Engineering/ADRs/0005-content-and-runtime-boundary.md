# 0005 - Content and Runtime Boundary

Status: Accepted for Milestone 1

Date: 2026-04-26

## Context

AmandaCore has authored content packages, JSON schemas, map exports, runtime activation, and placeholder O3DE streaming hooks. Future milestones need scalable content authoring and script boundaries without embedding every quest, NPC, item, spell, or world rule directly in service code.

## Decision

AmandaCore content must remain data-driven, validated before runtime, and separated from authoritative simulation code. Runtime code may load AmandaCore-original package manifests, validate them, activate supported definitions, and expose protocol-neutral state. Content packages must use AmandaCore-owned schemas, identifiers, names, maps, assets, quest text, event hooks, and validation reports.

Future script or plugin work must cross a narrow event-hook boundary. Scripts or declarative rules may request AmandaCore-defined effects, but authoritative state mutation remains owned by the runtime.

## Consequences

- Invalid content should fail validation before activation.
- New content should not require broad service rewrites once the compiler boundary matures.
- External schemas, content IDs, packet layouts, script names, and data layouts remain forbidden.
- O3DE presentation and streaming hooks consume runtime output; they do not own authoritative gameplay state.

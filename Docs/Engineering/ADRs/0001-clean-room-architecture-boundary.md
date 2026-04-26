# 0001 - Clean-room Architecture Boundary

Status: Accepted for Milestone 1

Date: 2026-04-26

## Context

AmandaCore is an original MMO foundation. External MMO server projects may be useful as high-level architectural references, but the repository must not preserve external implementation artifacts, database layouts, protocols, content identifiers, command vocabularies, comments, or module structures.

## Decision

AmandaCore architecture work must be re-specified in AmandaCore terms before implementation. Designs may describe general responsibilities such as identity control planes, realm directories, single-use join tickets, server-authoritative simulation, relational persistence, deterministic replay, content validation, RBAC, and observability. Implementations must use AmandaCore-owned names, schemas, commands, routes, DTOs, event names, tests, and content IDs.

Any work influenced by external projects must pass through a neutral design note before code is written. The design note must describe behavior and failure modes without copying source, SQL, packet layouts, opcodes, script structures, content data, command names, or comments.

## Consequences

- Contract and architecture docs are source-of-truth for future implementation.
- Similar broad MMO concepts are allowed only when expressed as AmandaCore-owned contracts.
- Review must reject accidental compatibility artifacts, copied layouts, copied names, and provenance ambiguity.
- Clean-room boundaries apply equally to code, tests, generated manifests, content, operations docs, and release material.

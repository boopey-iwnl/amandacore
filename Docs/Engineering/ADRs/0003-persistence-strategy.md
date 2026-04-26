# 0003 - Persistence Strategy

Status: Accepted for Milestone 1

Date: 2026-04-26

## Context

AmandaCore currently uses a shared file-backed store for local/dev/staging service flows. The codebase already has repository interfaces, transactions, migration records, and recovery-oriented types, but later milestones will need relational persistence without breaking launcher/login/realm/character/world flows.

## Decision

Milestone 1 freezes the persistence direction without replacing storage. Services should continue to run against the current file-backed adapter while documentation and tests define the future boundary:

- repository interfaces own domain access;
- migrations are ordered, immutable, checksumable, and idempotent;
- transactional mutations protect character, inventory, quest, action-bar, and economy state;
- hot world loops should not depend on raw storage implementation details;
- relational schemas must be AmandaCore-original and derived from AmandaCore aggregates.

## Consequences

- No production database cutover happens in Milestone 1.
- File-backed storage remains a dev/legacy adapter until a later approved milestone replaces it.
- New persistence work must attach to repository and migration boundaries instead of direct JSON mutation.
- Future relational schema review must check for clean-room originality and explicit migration discipline.

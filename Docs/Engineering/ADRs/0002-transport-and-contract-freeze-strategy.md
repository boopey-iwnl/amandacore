# 0002 - Transport and Contract Freeze Strategy

Status: Accepted for Milestone 1

Date: 2026-04-26

## Context

Alpha 0.15 uses HTTP/JSON between the C# launcher, diagnostic world client, O3DE-facing client code, and Go services. Future milestones need room for push replication, binary gateways, or other transports, but Milestone 1 must not rewrite runtime transport behavior.

## Decision

The current HTTP/JSON surface is frozen as the Milestone 1 compatibility contract. The freeze records registered routes, launcher-used endpoints, world session bootstrap payloads, canonical command/event names, and known drift points. Future transport adapters must translate into AmandaCore canonical commands and must not move gameplay authority into clients or presentation layers.

Contract validation starts with route-manifest parity. Every registered service route must be represented in `Docs/Contracts/http-api-v1.json`, and the manifest must not list routes that are absent from the codebase.

## Consequences

- Milestone 1 can add documentation and tests around existing routes without changing handler behavior.
- Later transport work can add adapters beside the current HTTP polling path.
- Route additions, removals, or renames become explicit contract changes that fail validation until documented.
- DTO shape validation can be layered after route parity without destabilizing Alpha 0.15 behavior.

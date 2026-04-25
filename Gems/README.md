# Gem Manifests

These Gem manifests mark the intended subsystem split for future O3DE integration. They are intentionally lightweight in this scaffold because the workspace does not currently include an engine checkout or generated module boilerplate.

When you wire this into O3DE, each Gem should grow into a normal layout with:

- `Code/Include/<GemName>/...`
- `Code/Source/...`
- module and system component registration
- asset builders or editor components where needed

The shared gameplay rules should remain in `Shared/AmandaCoreShared` and be linked into both client and server-facing Gems instead of being reimplemented per Gem.

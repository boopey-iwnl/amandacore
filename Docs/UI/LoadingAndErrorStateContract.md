# Loading and Error State Contract

Loading, connecting, and error states must be readable, actionable, and safe for players.

## Covered States

- startup and boot
- login and authentication
- realm list loading
- character list and create flow
- world join
- visible world loading
- disconnected or reconnecting state
- expired or invalid session
- unavailable services
- invalid, duplicate, or rate-limited action

## Rules

- UI text must not expose secrets, tokens, stack traces, raw tickets, or credentials.
- Technical details may remain in logs, but visible UI should use player-facing language.
- Errors should clear on successful retry or successful state transition.
- Retry/back controls are shown only where current client interfaces safely support them.
- Serious validation failures must remain visible in UI or logs; do not hide them silently.

## M9 Implementation

M9 maps common raw transport, session, service, invalid-action, duplicate-action, and rate-limit text into friendlier client-facing wording. It does not change backend error contracts or HTTP response shapes.

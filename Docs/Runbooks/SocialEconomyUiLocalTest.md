# Social Economy UI Local Test

## Setup

1. Check out `codex/ui-m7-social-economy-shells`.
2. Confirm the worktree is clean before human testing.
3. Start the local stack with `Infra/dev/start-local.ps1 -StartLauncher`.
4. Use the launcher patcher/play UI to enter the O3DE client.

## Smoke Checklist

- launcher opens patcher/play UI
- client login works
- realm select works
- character select works
- join world works
- visible world loads
- no crash
- Enter focuses chat
- Escape exits chat focus
- sending chat returns movement input to gameplay
- chat channel filters are readable
- Say and System messages are readable
- Party and Guild chat controls are disabled unless membership exists
- Social button opens the Social panel
- Friends tab is readable
- Blocked tab is clearly disabled
- Party tab is readable or clearly empty
- Guild tab is readable or clearly empty
- Trade tab is clearly disabled
- vendor panel opens only when targeting/interacting with a vendor NPC
- vendor buy/sell works if a reachable vendor and currency/items are available
- vendor unavailable/insufficient-currency states are clear
- auction browse/list/buy/cancel works if a reachable auction NPC is available
- Mail tab is read-only or clearly empty
- inventory opens and rearranges
- action bars work
- spellbook works
- character panel works
- quest log and map work
- combat HUD works
- movement and camera work
- no AddOns tab or addon runtime is present
- no runtime reference to `Downloads/textures` is present

## Expected Limitations

- Ignore/block lists are disabled shells.
- Mail claim/send is disabled.
- Player trade is disabled.
- Vendor buy/sell depends on targeting a real vendor NPC and receiving a populated `worldSession.vendor` payload.
- Slow-network duplicate-click in-flight affordances are minimal; server state remains authoritative.

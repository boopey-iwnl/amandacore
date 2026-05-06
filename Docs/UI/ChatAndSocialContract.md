# Chat And Social UI Contract

## Chat

The chat frame is a first-party AmandaCore panel owned by `UiClient`. Enter focuses chat input. Escape exits chat input before it closes gameplay panels. Sending a message clears focus and returns discrete input to gameplay.

Supported display filters:

- All
- System
- Say
- Party when party membership is present
- Guild when guild membership is present
- Whisper

Supported send channels:

- Say
- Party when `SocialStateResponse` reports party membership
- Guild when `SocialStateResponse` reports guild membership
- Whisper with a target name

Party and Guild controls remain visible but disabled with an unavailable state when the runtime payload does not support sending on that channel. System is display-only.

The client must not display passwords, tokens, session IDs, or launch tickets in chat. Slash commands continue to route through GameCore social calls and server-side validation.

## Social Panel

The Social panel opens from the first-party micro menu. It has these tabs:

- Friends
- Blocked
- Party
- Guild
- Trade

Friends add/remove uses existing live friend routes. Invite buttons use existing party invite routes and are shown only for online friends reported by the server.

The Blocked tab is disabled because ignore/block runtime HTTP and client wiring are not exposed. It must not pretend to add or remove blocked characters.

The Party tab renders live party membership and incoming invite state when present. Invite, accept, decline, leave, and disband remain server-owned mutations.

The Guild tab renders live guild roster, rank, permission, invite, and message-of-the-day state when present. Controls are enabled only when the current permission payload supports them.

The Trade tab is a disabled placeholder until a server-authoritative player-trade contract exists.

## Input Rules

Typing in chat and social text fields must not trigger movement, camera, or action-bar keybinds. Panel clicks are consumed by ImGui and must not target or move in the world. Escape closes the topmost applicable UI surface before gameplay input resumes.

## No Addons

No AddOns tab, addon loader, Lua integration, plugin runtime, user UI module loading, or arbitrary script execution is part of this contract.

# Party And Guild UI Contract

## Party

Party UI reads from `SocialStateResponse.party` and `SocialStateResponse.partyInvites`.

Live actions:

- invite by name or known character ID
- accept invite
- decline invite
- leave party
- disband party when supported by the current server route

Party membership is never client-authoritative. The client asks GameCore to call the world service and applies only the returned social payload.

The compact party HUD frames render only when the social payload reports an active party. The Social panel Party tab remains available even when no party exists and shows a clear empty state.

## Guild

Guild UI reads from `SocialStateResponse.guild` and `SocialStateResponse.guildInvites`.

Live actions:

- create guild
- invite when `invite_member` permission is present
- accept invite
- decline invite
- leave guild
- set message of the day when `edit_motd` permission is present
- promote/demote/remove when the corresponding server permission is present
- disband when `disband_guild` permission is present

The client renders rank and permission names exactly as the AmandaCore backend provides them. It does not add copied external guild rank names or UI text.

## Disabled Scope

Guild bank, guild permissions editing, guild calendar, guild achievements, and guild economy features are not included. They require separate server-authoritative contracts before UI controls can be enabled.

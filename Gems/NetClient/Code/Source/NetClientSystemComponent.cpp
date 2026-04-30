#include <NetClient/NetClientSystemComponent.h>

#include <AzCore/Debug/Trace.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <NetClient/WorldHttpClient.h>

namespace NetClient
{
    bool LoginRequest(
        const AZStd::string& authEndpoint,
        const AZStd::string& username,
        const AZStd::string& password,
        AuthSessionResponse& outResponse,
        AZStd::string& outError);

    bool ListRealmsRequest(
        const AZStd::string& realmEndpoint,
        AZStd::vector<RealmDescriptor>& outRealms,
        AZStd::string& outError);

    bool ListCharactersRequest(
        const AZStd::string& characterEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        AZStd::vector<CharacterSummary>& outCharacters,
        AZStd::string& outError);

    bool CreateCharacterRequest(
        const AZStd::string& characterEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        const AZStd::string& displayName,
        const AZStd::string& archetypeId,
        CharacterSummary& outCharacter,
        AZStd::string& outError);

    bool CreateJoinTicketRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        const AZStd::string& characterId,
        WorldJoinTicketResponse& outTicket,
        AZStd::string& outError);

    bool ConnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& ticketId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool BootstrapRequest(
        const AZStd::string& worldEndpoint,
        WorldBootstrapResponse& outResponse,
        AZStd::string& outError);

    bool MoveRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        double deltaX,
        double deltaY,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool DisconnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        AZStd::string& outError);

    bool StateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& cursor,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool SocialStateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& afterMessageId,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool SendChatRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& channel,
        const AZStd::string& targetName,
        const AZStd::string& messageText,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool FriendRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& name,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool InvitePartyRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        const AZStd::string& targetCharacterId,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool PartyInviteActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool PartyActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool GuildCreateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& guildName,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool GuildNameActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool GuildInviteActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool GuildMOTDRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& messageOfTheDay,
        SocialStateResponse& outResponse,
        AZStd::string& outError);

    bool SetTargetRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool AcceptQuestRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool EnterDungeonRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& dungeonId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool ExitDungeonRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool TrackQuestRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        bool tracked,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool SetAutoAttackRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        bool enabled,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool ActivateAbilityRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool RequestDuelRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetCharacterId,
        const AZStd::string& targetName,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool DuelActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool LearnTrainerAbilityRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& trainerId,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool SelectTalentRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& talentId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool AssignActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool MoveActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool ClearActionBarSlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool MoveInventorySlotRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    bool AuctionStateRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& search,
        const AZStd::string& itemType,
        const AZStd::string& sort,
        AuctionStateResponse& outResponse,
        AZStd::string& outError);

    bool ListAuctionItemRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        int stackCount,
        int buyoutCopper,
        AZ::s64 durationSeconds,
        AuctionStateResponse& outResponse,
        AZStd::string& outError);

    bool AuctionIdActionRequest(
        const AZStd::string& worldEndpoint,
        const wchar_t* path,
        const AZStd::string& worldSessionToken,
        const AZStd::string& auctionId,
        AuctionStateResponse& outResponse,
        AZStd::string& outError);

    bool ReconnectRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError);

    void NetClientSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<NetClientSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void NetClientSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("NetClientService"));
    }

    void NetClientSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("NetClientService"));
    }

    void NetClientSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void NetClientSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void NetClientSystemComponent::Activate()
    {
        IWorldHttpClient::Register(this);
        AZ_Printf("amandacore", "NetClient ready.");
    }

    void NetClientSystemComponent::Deactivate()
    {
        if (IWorldHttpClient::Get() == this)
        {
            IWorldHttpClient::Unregister(this);
        }
    }

    bool NetClientSystemComponent::Login(
        const AZStd::string& authEndpoint,
        const AZStd::string& username,
        const AZStd::string& password,
        AuthSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return LoginRequest(authEndpoint, username, password, outResponse, outError);
    }

    bool NetClientSystemComponent::ListRealms(
        const AZStd::string& realmEndpoint,
        AZStd::vector<RealmDescriptor>& outRealms,
        AZStd::string& outError)
    {
        return ListRealmsRequest(realmEndpoint, outRealms, outError);
    }

    bool NetClientSystemComponent::ListCharacters(
        const AZStd::string& characterEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        AZStd::vector<CharacterSummary>& outCharacters,
        AZStd::string& outError)
    {
        return ListCharactersRequest(characterEndpoint, accessToken, realmId, outCharacters, outError);
    }

    bool NetClientSystemComponent::CreateCharacter(
        const AZStd::string& characterEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        const AZStd::string& displayName,
        const AZStd::string& archetypeId,
        CharacterSummary& outCharacter,
        AZStd::string& outError)
    {
        return CreateCharacterRequest(characterEndpoint, accessToken, realmId, displayName, archetypeId, outCharacter, outError);
    }

    bool NetClientSystemComponent::CreateJoinTicket(
        const AZStd::string& worldEndpoint,
        const AZStd::string& accessToken,
        const AZStd::string& realmId,
        const AZStd::string& characterId,
        WorldJoinTicketResponse& outTicket,
        AZStd::string& outError)
    {
        return CreateJoinTicketRequest(worldEndpoint, accessToken, realmId, characterId, outTicket, outError);
    }

    bool NetClientSystemComponent::Connect(
        const AZStd::string& worldEndpoint,
        const AZStd::string& ticketId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ConnectRequest(worldEndpoint, ticketId, outResponse, outError);
    }

    bool NetClientSystemComponent::Bootstrap(
        const AZStd::string& worldEndpoint,
        WorldBootstrapResponse& outResponse,
        AZStd::string& outError)
    {
        return BootstrapRequest(worldEndpoint, outResponse, outError);
    }

    bool NetClientSystemComponent::Move(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        double deltaX,
        double deltaY,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return MoveRequest(worldEndpoint, worldSessionToken, deltaX, deltaY, outResponse, outError);
    }

    bool NetClientSystemComponent::Disconnect(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        AZStd::string& outError)
    {
        return DisconnectRequest(worldEndpoint, worldSessionToken, outError);
    }

    bool NetClientSystemComponent::State(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& cursor,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return StateRequest(worldEndpoint, worldSessionToken, cursor, outResponse, outError);
    }

    bool NetClientSystemComponent::SocialState(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& afterMessageId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return SocialStateRequest(worldEndpoint, worldSessionToken, afterMessageId, outResponse, outError);
    }

    bool NetClientSystemComponent::SendChat(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& channel,
        const AZStd::string& targetName,
        const AZStd::string& messageText,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return SendChatRequest(worldEndpoint, worldSessionToken, channel, targetName, messageText, outResponse, outError);
    }

    bool NetClientSystemComponent::AddFriend(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& name,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return FriendRequest(worldEndpoint, L"/v1/world/friends/add", worldSessionToken, name, outResponse, outError);
    }

    bool NetClientSystemComponent::RemoveFriend(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& name,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return FriendRequest(worldEndpoint, L"/v1/world/friends/remove", worldSessionToken, name, outResponse, outError);
    }

    bool NetClientSystemComponent::InviteParty(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        const AZStd::string& targetCharacterId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return InvitePartyRequest(worldEndpoint, worldSessionToken, targetName, targetCharacterId, outResponse, outError);
    }

    bool NetClientSystemComponent::AcceptPartyInvite(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyInviteActionRequest(worldEndpoint, L"/v1/world/party/accept", worldSessionToken, inviteId, outResponse, outError);
    }

    bool NetClientSystemComponent::DeclinePartyInvite(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyInviteActionRequest(worldEndpoint, L"/v1/world/party/decline", worldSessionToken, inviteId, outResponse, outError);
    }

    bool NetClientSystemComponent::LeaveParty(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyActionRequest(worldEndpoint, L"/v1/world/party/leave", worldSessionToken, outResponse, outError);
    }

    bool NetClientSystemComponent::DisbandParty(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyActionRequest(worldEndpoint, L"/v1/world/party/disband", worldSessionToken, outResponse, outError);
    }

    bool NetClientSystemComponent::CreateGuild(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& guildName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildCreateRequest(worldEndpoint, worldSessionToken, guildName, outResponse, outError);
    }

    bool NetClientSystemComponent::InviteGuild(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildNameActionRequest(worldEndpoint, L"/v1/world/guild/invite", worldSessionToken, targetName, outResponse, outError);
    }

    bool NetClientSystemComponent::AcceptGuildInvite(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildInviteActionRequest(worldEndpoint, L"/v1/world/guild/accept", worldSessionToken, inviteId, outResponse, outError);
    }

    bool NetClientSystemComponent::DeclineGuildInvite(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& inviteId,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildInviteActionRequest(worldEndpoint, L"/v1/world/guild/decline", worldSessionToken, inviteId, outResponse, outError);
    }

    bool NetClientSystemComponent::LeaveGuild(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyActionRequest(worldEndpoint, L"/v1/world/guild/leave", worldSessionToken, outResponse, outError);
    }

    bool NetClientSystemComponent::DisbandGuild(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return PartyActionRequest(worldEndpoint, L"/v1/world/guild/disband", worldSessionToken, outResponse, outError);
    }

    bool NetClientSystemComponent::PromoteGuildMember(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildNameActionRequest(worldEndpoint, L"/v1/world/guild/promote", worldSessionToken, targetName, outResponse, outError);
    }

    bool NetClientSystemComponent::DemoteGuildMember(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildNameActionRequest(worldEndpoint, L"/v1/world/guild/demote", worldSessionToken, targetName, outResponse, outError);
    }

    bool NetClientSystemComponent::RemoveGuildMember(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetName,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildNameActionRequest(worldEndpoint, L"/v1/world/guild/remove", worldSessionToken, targetName, outResponse, outError);
    }

    bool NetClientSystemComponent::SetGuildMessageOfTheDay(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& messageOfTheDay,
        SocialStateResponse& outResponse,
        AZStd::string& outError)
    {
        return GuildMOTDRequest(worldEndpoint, worldSessionToken, messageOfTheDay, outResponse, outError);
    }

    bool NetClientSystemComponent::SetTarget(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return SetTargetRequest(worldEndpoint, worldSessionToken, targetId, outResponse, outError);
    }

    bool NetClientSystemComponent::AcceptQuest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return AcceptQuestRequest(worldEndpoint, worldSessionToken, questId, outResponse, outError);
    }

    bool NetClientSystemComponent::EnterDungeon(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& dungeonId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return EnterDungeonRequest(worldEndpoint, worldSessionToken, dungeonId, outResponse, outError);
    }

    bool NetClientSystemComponent::ExitDungeon(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ExitDungeonRequest(worldEndpoint, worldSessionToken, outResponse, outError);
    }

    bool NetClientSystemComponent::TrackQuest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& questId,
        bool tracked,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return TrackQuestRequest(worldEndpoint, worldSessionToken, questId, tracked, outResponse, outError);
    }

    bool NetClientSystemComponent::SetAutoAttack(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        bool enabled,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return SetAutoAttackRequest(worldEndpoint, worldSessionToken, enabled, outResponse, outError);
    }

    bool NetClientSystemComponent::ActivateAbility(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ActivateAbilityRequest(worldEndpoint, worldSessionToken, abilityId, outResponse, outError);
    }

    bool NetClientSystemComponent::RequestDuel(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& targetCharacterId,
        const AZStd::string& targetName,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return RequestDuelRequest(worldEndpoint, worldSessionToken, targetCharacterId, targetName, outResponse, outError);
    }

    bool NetClientSystemComponent::AcceptDuel(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return DuelActionRequest(worldEndpoint, L"/v1/world/duel/accept", worldSessionToken, duelId, outResponse, outError);
    }

    bool NetClientSystemComponent::DeclineDuel(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return DuelActionRequest(worldEndpoint, L"/v1/world/duel/decline", worldSessionToken, duelId, outResponse, outError);
    }

    bool NetClientSystemComponent::CancelDuel(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return DuelActionRequest(worldEndpoint, L"/v1/world/duel/cancel", worldSessionToken, duelId, outResponse, outError);
    }

    bool NetClientSystemComponent::SurrenderDuel(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& duelId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return DuelActionRequest(worldEndpoint, L"/v1/world/duel/surrender", worldSessionToken, duelId, outResponse, outError);
    }

    bool NetClientSystemComponent::LearnTrainerAbility(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& trainerId,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return LearnTrainerAbilityRequest(
            worldEndpoint,
            worldSessionToken,
            trainerId,
            abilityId,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::SelectTalent(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& talentId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return SelectTalentRequest(worldEndpoint, worldSessionToken, talentId, outResponse, outError);
    }

    bool NetClientSystemComponent::AssignActionBarSlot(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        const AZStd::string& abilityId,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return AssignActionBarSlotRequest(worldEndpoint, worldSessionToken, slotIndex, abilityId, outResponse, outError);
    }

    bool NetClientSystemComponent::ClearActionBarSlot(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ClearActionBarSlotRequest(worldEndpoint, worldSessionToken, slotIndex, outResponse, outError);
    }

    bool NetClientSystemComponent::MoveActionBarSlot(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return MoveActionBarSlotRequest(
            worldEndpoint,
            worldSessionToken,
            fromSlotIndex,
            toSlotIndex,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::MoveInventorySlot(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int fromSlotIndex,
        int toSlotIndex,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return MoveInventorySlotRequest(
            worldEndpoint,
            worldSessionToken,
            fromSlotIndex,
            toSlotIndex,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::BrowseAuctions(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& search,
        const AZStd::string& itemType,
        const AZStd::string& sort,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        return AuctionStateRequest(worldEndpoint, worldSessionToken, search, itemType, sort, outResponse, outError);
    }

    bool NetClientSystemComponent::ListAuctionItem(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        int slotIndex,
        int stackCount,
        int buyoutCopper,
        AZ::s64 durationSeconds,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        return ListAuctionItemRequest(
            worldEndpoint,
            worldSessionToken,
            slotIndex,
            stackCount,
            buyoutCopper,
            durationSeconds,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::BuyoutAuction(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& auctionId,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        return AuctionIdActionRequest(
            worldEndpoint,
            L"/v1/world/auction/buyout",
            worldSessionToken,
            auctionId,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::CancelAuction(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& auctionId,
        AuctionStateResponse& outResponse,
        AZStd::string& outError)
    {
        return AuctionIdActionRequest(
            worldEndpoint,
            L"/v1/world/auction/cancel",
            worldSessionToken,
            auctionId,
            outResponse,
            outError);
    }

    bool NetClientSystemComponent::Reconnect(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ReconnectRequest(worldEndpoint, worldSessionToken, outResponse, outError);
    }
}

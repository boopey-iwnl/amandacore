#include <NetClient/NetClientSystemComponent.h>

#include <AzCore/Debug/Trace.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <NetClient/WorldHttpClient.h>

namespace NetClient
{
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

    bool LearnTrainerAbilityRequest(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        const AZStd::string& trainerId,
        const AZStd::string& abilityId,
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
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return StateRequest(worldEndpoint, worldSessionToken, outResponse, outError);
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

    bool NetClientSystemComponent::Reconnect(
        const AZStd::string& worldEndpoint,
        const AZStd::string& worldSessionToken,
        WorldSessionResponse& outResponse,
        AZStd::string& outError)
    {
        return ReconnectRequest(worldEndpoint, worldSessionToken, outResponse, outError);
    }
}

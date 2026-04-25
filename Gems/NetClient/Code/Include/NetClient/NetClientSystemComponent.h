#pragma once

#include <AzCore/Component/Component.h>
#include <NetClient/WorldHttpClient.h>

namespace NetClient
{
    class NetClientSystemComponent final
        : public AZ::Component
        , public IWorldHttpClient
    {
    public:
        AZ_COMPONENT(NetClientSystemComponent, "{148F8E93-02D1-4A32-BCCA-1EEA838D72C2}");

        NetClientSystemComponent() = default;
        ~NetClientSystemComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        bool Connect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& ticketId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool Bootstrap(
            const AZStd::string& worldEndpoint,
            WorldBootstrapResponse& outResponse,
            AZStd::string& outError) override;

        bool Move(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            double deltaX,
            double deltaY,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool Disconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            AZStd::string& outError) override;

        bool State(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SetTarget(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& targetId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool AcceptQuest(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& questId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool SetAutoAttack(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            bool enabled,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ActivateAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool LearnTrainerAbility(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            const AZStd::string& trainerId,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool AssignActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            const AZStd::string& abilityId,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool MoveActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool ClearActionBarSlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int slotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool MoveInventorySlot(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            int fromSlotIndex,
            int toSlotIndex,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;

        bool Reconnect(
            const AZStd::string& worldEndpoint,
            const AZStd::string& worldSessionToken,
            WorldSessionResponse& outResponse,
            AZStd::string& outError) override;
    };
}

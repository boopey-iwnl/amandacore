#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <GameCore/GameCoreInterface.h>

namespace GameCore
{
    class GameCoreSystemComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public AzFramework::LevelSystemLifecycleNotificationBus::Handler
        , public IGameCoreRequests
    {
    public:
        AZ_COMPONENT(GameCoreSystemComponent, "{72D82A3D-0F0F-4F93-8E47-91D7446A218A}");

        GameCoreSystemComponent() = default;
        ~GameCoreSystemComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;

        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;
        void OnLoadingComplete(const char* levelName) override;

        const ClientLaunchOptions& GetLaunchOptions() const override;
        const ClientWorldState& GetClientWorldState() const override;
        const ClientCameraState& GetCameraState() const override;
        bool SubmitMove(double deltaX, double deltaY) override;
        bool SetTarget(const AZStd::string& targetId) override;
        bool InteractWithEntity(const AZStd::string& entityId) override;
        bool AcceptQuest(const AZStd::string& questId) override;
        bool TrackQuest(const AZStd::string& questId, bool tracked) override;
        bool SetAutoAttack(bool enabled) override;
        bool ActivateAbility(const AZStd::string& abilityId) override;
        bool LearnTrainerAbility(const AZStd::string& trainerId, const AZStd::string& abilityId) override;
        bool AssignActionBarSlot(int slotIndex, const AZStd::string& abilityId) override;
        bool MoveActionBarSlot(int fromSlotIndex, int toSlotIndex) override;
        bool ClearActionBarSlot(int slotIndex) override;
        bool MoveInventorySlot(int fromSlotIndex, int toSlotIndex) override;
        bool DisconnectWorld() override;
        bool ReconnectWorld() override;
        void SetCameraState(const ClientCameraState& cameraState) override;

    private:
        void ParseLaunchOptions();
        void MarkLevelReady(const char* levelName);
        void AttemptInitialWorldConnect();
        bool ApplyWorldSessionResponse(NetClient::WorldSessionResponse&& response, const char* source);
        void EnsureAbilityPresentationDefaults(NetClient::WorldSessionResponse& session, const char* source);
        bool PollWorldState();
        void LogCombatStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogAbilityStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogQuestStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);
        void LogTrainerStateIfChanged(const NetClient::WorldSessionResponse& previousSession, const char* source);

        ClientLaunchOptions m_launchOptions;
        ClientWorldState m_worldState;
        ClientCameraState m_cameraState;
        bool m_levelReady = false;
        bool m_worldConnectStartLogged = false;
        bool m_levelReadyLogged = false;
        float m_statePollAccumulator = 0.0f;
    };
}

#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzCore/std/containers/unordered_map.h>
#include <AzCore/std/string/string.h>

namespace AZ
{
    class Entity;
}

namespace NpcAi
{
    class NpcAiSystemComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
    {
    public:
        AZ_COMPONENT(NpcAiSystemComponent, "{3258852B-A437-444A-B586-59766D340AB7}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;

    private:
        struct MobProxyState
        {
            AZ::Entity* m_entity = nullptr;
            bool m_lastAlive = false;
        };

        void DestroyProxy(const AZStd::string& mobId);

        AZStd::unordered_map<AZStd::string, MobProxyState> m_mobProxies;
        bool m_loggedEncounterVisible = false;
        bool m_loggedEncounterValidationPassed = false;
        bool m_loggedZoneLandmarksReady = false;
        bool m_loggedTrainerNpcVisible = false;
        bool m_loggedQuestGiverNpcVisible = false;
    };
}

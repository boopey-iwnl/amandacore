#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>

namespace NpcAi
{
    class MobCombatStateComponent;

    class MobPresentationComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
    {
    public:
        AZ_COMPONENT(MobPresentationComponent, "{52A8F6E7-2E33-47D2-85E5-13FB6FC1C711}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;

    private:
        MobCombatStateComponent* m_stateComponent = nullptr;
    };
} // namespace NpcAi

#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/Input/Events/InputChannelEventListener.h>

namespace CombatRules
{
    class PlayerCombatComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public AzFramework::InputChannelEventListener
    {
    public:
        AZ_COMPONENT(PlayerCombatComponent, "{F41ED8C9-8BDE-4F0F-81E5-8C594B75F869}");

        PlayerCombatComponent();
        ~PlayerCombatComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;

    protected:
        bool OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel) override;

    private:
        bool m_loggedReady = false;
        bool m_loggedHelp = false;
        bool m_lastAutoAttackState = false;
        AZStd::string m_lastCastingAbilityId;
    };
} // namespace CombatRules

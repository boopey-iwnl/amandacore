#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/Input/Events/InputChannelEventListener.h>

namespace CombatRules
{
    class PlayerTargetingComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public AzFramework::InputChannelEventListener
    {
    public:
        AZ_COMPONENT(PlayerTargetingComponent, "{A3F2AE2F-90AA-428E-BAA1-DA6091D1552B}");

        PlayerTargetingComponent();
        ~PlayerTargetingComponent() override = default;

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
        AZStd::string FindNextHostileTarget() const;
        AZStd::string FindClickedFriendlyNpcTarget() const;
        AZStd::string FindClickedTarget(bool friendlyOnly) const;

        bool m_loggedReady = false;
        bool m_loggedHelp = false;
        AZStd::string m_lastLoggedTargetId;
    };
} // namespace CombatRules

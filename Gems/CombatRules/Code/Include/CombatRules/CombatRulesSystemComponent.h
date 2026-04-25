#pragma once

#include <AzCore/Component/Component.h>

namespace CombatRules
{
    class CombatRulesSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(CombatRulesSystemComponent, "{21FBAA0E-0D12-4B41-A693-D909832C253D}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}

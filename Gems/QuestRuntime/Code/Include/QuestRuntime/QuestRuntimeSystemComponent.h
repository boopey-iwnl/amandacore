#pragma once

#include <AzCore/Component/Component.h>

namespace QuestRuntime
{
    class QuestRuntimeSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(QuestRuntimeSystemComponent, "{033FC814-480A-4AFD-9EA0-804EAA57A9DD}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}

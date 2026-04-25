#pragma once

#include <AzCore/Component/Component.h>

namespace StatsProgression
{
    class StatsProgressionSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(StatsProgressionSystemComponent, "{46E7D7BA-B190-4B04-BBE0-86D76225882D}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}

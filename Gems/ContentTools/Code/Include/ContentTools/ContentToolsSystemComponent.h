#pragma once

#include <AzCore/Component/Component.h>

namespace ContentTools
{
    class ContentToolsSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(ContentToolsSystemComponent, "{C4FEA349-22AD-47DA-8F0D-A521D4DE36F6}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}

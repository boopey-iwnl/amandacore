#pragma once

#include <AzCore/Component/Component.h>

namespace Persistence
{
    class PersistenceSystemComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(PersistenceSystemComponent, "{DF4EA0BD-5302-49B6-B8A5-779E457721D0}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
    };
}

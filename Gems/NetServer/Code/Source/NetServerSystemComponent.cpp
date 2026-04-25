#include <NetServer/NetServerSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace NetServer
{
    void NetServerSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<NetServerSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void NetServerSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("NetServerService"));
    }

    void NetServerSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("NetServerService"));
    }

    void NetServerSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("PersistenceService"));
    }

    void NetServerSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void NetServerSystemComponent::Activate()
    {
    }

    void NetServerSystemComponent::Deactivate()
    {
    }
}

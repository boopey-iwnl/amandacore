#include <Persistence/PersistenceSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace Persistence
{
    void PersistenceSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<PersistenceSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void PersistenceSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("PersistenceService"));
    }

    void PersistenceSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("PersistenceService"));
    }

    void PersistenceSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void PersistenceSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void PersistenceSystemComponent::Activate()
    {
    }

    void PersistenceSystemComponent::Deactivate()
    {
    }
}

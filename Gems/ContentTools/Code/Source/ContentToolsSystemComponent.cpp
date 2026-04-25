#include <ContentTools/ContentToolsSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace ContentTools
{
    void ContentToolsSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<ContentToolsSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void ContentToolsSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("ContentToolsService"));
    }

    void ContentToolsSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("ContentToolsService"));
    }

    void ContentToolsSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
        required.push_back(AZ_CRC_CE("PersistenceService"));
    }

    void ContentToolsSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void ContentToolsSystemComponent::Activate()
    {
    }

    void ContentToolsSystemComponent::Deactivate()
    {
    }
}

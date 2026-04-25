#include <QuestRuntime/QuestRuntimeSystemComponent.h>

#include <AzCore/Serialization/SerializeContext.h>

namespace QuestRuntime
{
    void QuestRuntimeSystemComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<QuestRuntimeSystemComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void QuestRuntimeSystemComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("QuestRuntimeService"));
    }

    void QuestRuntimeSystemComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("QuestRuntimeService"));
    }

    void QuestRuntimeSystemComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("GameCoreService"));
    }

    void QuestRuntimeSystemComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void QuestRuntimeSystemComponent::Activate()
    {
    }

    void QuestRuntimeSystemComponent::Deactivate()
    {
    }
}

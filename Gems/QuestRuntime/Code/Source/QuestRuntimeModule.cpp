#include <QuestRuntime/QuestRuntimeSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace QuestRuntime
{
    class QuestRuntimeModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(QuestRuntimeModule, "{8668C682-B03D-4775-84C5-EB9042550B3A}", AZ::Module);
        AZ_CLASS_ALLOCATOR(QuestRuntimeModule, AZ::SystemAllocator);

        QuestRuntimeModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                QuestRuntimeSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<QuestRuntimeSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_QuestRuntime, QuestRuntime::QuestRuntimeModule)

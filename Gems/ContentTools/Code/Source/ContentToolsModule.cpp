#include <ContentTools/ContentToolsSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace ContentTools
{
    class ContentToolsModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(ContentToolsModule, "{F201EABC-4D48-4D69-BC75-BD6FEFF7CC8A}", AZ::Module);
        AZ_CLASS_ALLOCATOR(ContentToolsModule, AZ::SystemAllocator);

        ContentToolsModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                ContentToolsSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<ContentToolsSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_ContentTools, ContentTools::ContentToolsModule)

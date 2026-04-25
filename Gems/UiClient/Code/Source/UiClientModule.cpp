#include <UiClient/UiClientSystemComponent.h>

#include <AzCore/Memory/SystemAllocator.h>
#include <AzCore/Module/Module.h>

namespace UiClient
{
    class UiClientModule final
        : public AZ::Module
    {
    public:
        AZ_RTTI(UiClientModule, "{6A9B5042-3A86-4FAE-BE91-F1A4D08C5F80}", AZ::Module);
        AZ_CLASS_ALLOCATOR(UiClientModule, AZ::SystemAllocator);

        UiClientModule()
        {
            m_descriptors.insert(m_descriptors.end(), {
                UiClientSystemComponent::CreateDescriptor()
            });
        }

        AZ::ComponentTypeList GetRequiredSystemComponents() const override
        {
            return AZ::ComponentTypeList{ azrtti_typeid<UiClientSystemComponent>() };
        }
    };
}

AZ_DECLARE_MODULE_CLASS(Gem_UiClient, UiClient::UiClientModule)

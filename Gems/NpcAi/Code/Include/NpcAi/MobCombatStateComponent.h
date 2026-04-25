#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/std/string/string.h>

namespace NpcAi
{
    class MobCombatStateComponent final
        : public AZ::Component
    {
    public:
        AZ_COMPONENT(MobCombatStateComponent, "{6DCD6B80-43D4-4D4A-B66F-B01691AD12B9}");

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override {}
        void Deactivate() override {}

        void SetMobId(const AZStd::string& value) { m_mobId = value; }
        void SetDisplayName(const AZStd::string& value) { m_displayName = value; }
        void SetHealth(double value) { m_health = value; }
        void SetMaxHealth(double value) { m_maxHealth = value; }
        void SetAlive(bool value) { m_alive = value; }
        void SetTargetable(bool value) { m_targetable = value; }
        void SetAiState(const AZStd::string& value) { m_aiState = value; }

        const AZStd::string& GetMobId() const { return m_mobId; }
        const AZStd::string& GetDisplayName() const { return m_displayName; }
        double GetHealth() const { return m_health; }
        double GetMaxHealth() const { return m_maxHealth; }
        bool IsAlive() const { return m_alive; }
        bool IsTargetable() const { return m_targetable; }
        const AZStd::string& GetAiState() const { return m_aiState; }

    private:
        AZStd::string m_mobId;
        AZStd::string m_displayName;
        AZStd::string m_aiState;
        double m_health = 0.0;
        double m_maxHealth = 0.0;
        bool m_alive = false;
        bool m_targetable = false;
    };
} // namespace NpcAi

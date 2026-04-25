#include <NpcAi/MobPresentationComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AzCore/Component/Entity.h>
#include <AzCore/Component/TransformBus.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <GameCore/GameCoreInterface.h>
#include <GameCore/MobProxyGeometry.h>
#include <NpcAi/MobCombatStateComponent.h>

namespace NpcAi
{
    namespace
    {
        constexpr float SelectedRingRadius = 1.45f;
        constexpr float SelectedRingSphereRadius = 0.12f;
        constexpr int SelectedRingSegments = 18;
        constexpr float SelectedMarkerTopOffset = 2.75f;
        constexpr float SelectedMarkerStep = 0.28f;
        constexpr int SelectedMarkerSteps = 5;

        AZ::Color GetInstanceAccentColor(int ordinal)
        {
            switch (ordinal)
            {
            case 1:
                return AZ::Color(0.35f, 0.85f, 1.0f, 1.0f);
            case 2:
                return AZ::Color(0.45f, 1.0f, 0.45f, 1.0f);
            case 3:
                return AZ::Color(0.95f, 0.55f, 1.0f, 1.0f);
            default:
                return AZ::Color(1.0f, 1.0f, 1.0f, 1.0f);
            }
        }
    }

    void MobPresentationComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<MobPresentationComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void MobPresentationComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("MobPresentationService"));
    }

    void MobPresentationComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("MobPresentationService"));
    }

    void MobPresentationComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("TransformService"));
        required.push_back(AZ_CRC_CE("MobCombatStateService"));
    }

    void MobPresentationComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void MobPresentationComponent::Activate()
    {
        if (GetEntity())
        {
            m_stateComponent = GetEntity()->FindComponent<MobCombatStateComponent>();
        }
        AZ::TickBus::Handler::BusConnect();
    }

    void MobPresentationComponent::Deactivate()
    {
        AZ::TickBus::Handler::BusDisconnect();
        m_stateComponent = nullptr;
    }

    void MobPresentationComponent::OnTick(float, AZ::ScriptTimePoint)
    {
        if (!m_stateComponent)
        {
            return;
        }

        AZ::RPI::Scene* scene = AZ::RPI::Scene::GetSceneForEntityId(GetEntityId());
        if (!scene)
        {
            return;
        }

        auto auxGeom = AZ::RPI::AuxGeomFeatureProcessorInterface::GetDrawQueueForScene(scene);
        if (!auxGeom)
        {
            return;
        }

        AZ::Vector3 position = AZ::Vector3::CreateZero();
        AZ::TransformBus::EventResult(position, GetEntityId(), &AZ::TransformBus::Events::GetWorldTranslation);

        const int mobOrdinal = GameCore::MobProxyGeometry::GetMobOrdinal(m_stateComponent->GetMobId());
        const bool isTrainerNpc = m_stateComponent->GetAiState() == "trainer";
        const bool isQuestGiverNpc = m_stateComponent->GetAiState() == "quest_giver";
        const bool isServiceNpc = m_stateComponent->GetAiState() == "profession_trainer" ||
            m_stateComponent->GetAiState() == "quest_object";
        const AZ::Color instanceAccent = isTrainerNpc
            ? AZ::Color(0.95f, 0.72f, 0.22f, 1.0f)
            : (isQuestGiverNpc
                ? AZ::Color(0.30f, 0.95f, 0.70f, 1.0f)
                : (isServiceNpc ? AZ::Color(0.68f, 0.82f, 1.0f, 1.0f) : GetInstanceAccentColor(mobOrdinal)));
        AZ::Color color = m_stateComponent->IsAlive()
            ? (isTrainerNpc
                ? AZ::Color(0.18f, 0.56f, 0.92f, 1.0f)
                : (isQuestGiverNpc
                    ? AZ::Color(0.20f, 0.70f, 0.38f, 1.0f)
                    : (isServiceNpc ? AZ::Color(0.34f, 0.46f, 0.72f, 1.0f) : AZ::Color(0.85f, 0.25f, 0.15f, 1.0f))))
            : AZ::Color(0.4f, 0.4f, 0.4f, 1.0f);
        const bool isSelected = [&]() -> bool
        {
            auto* gameCore = GameCore::IGameCoreRequests::Get();
            return gameCore && gameCore->GetClientWorldState().m_session.m_currentTargetId == m_stateComponent->GetMobId();
        }();
        if (isSelected)
        {
            color = AZ::Color(1.0f, 0.85f, 0.2f, 1.0f);
        }

        GameCore::MobProxyGeometry::VisitWorldProxySpheres(
            m_stateComponent->GetMobId(),
            position,
            m_stateComponent->IsAlive(),
            [auxGeom, &color, &instanceAccent](const AZ::Vector3& sphereCenter, float sphereRadius)
            {
                const bool isAdornmentSphere = sphereRadius <= GameCore::MobProxyGeometry::BaseRingSphereRadius ||
                    sphereRadius == GameCore::MobProxyGeometry::InstancePipRadius;
                auxGeom->DrawSphere(sphereCenter, sphereRadius, isAdornmentSphere ? instanceAccent : color);
            });

        if (m_stateComponent->IsAlive() && mobOrdinal > 0)
        {
            // Instance pips are already drawn via shared proxy geometry.
        }

        if ((isTrainerNpc || isQuestGiverNpc || isServiceNpc) && m_stateComponent->IsAlive())
        {
            const AZ::Color trainerPromptColor = isTrainerNpc
                ? AZ::Color(0.95f, 0.78f, 0.28f, 1.0f)
                : (isQuestGiverNpc ? AZ::Color(0.35f, 1.0f, 0.72f, 1.0f) : AZ::Color(0.62f, 0.78f, 1.0f, 1.0f));
            for (int markerIndex = 0; markerIndex < SelectedMarkerSteps; ++markerIndex)
            {
                const float heightOffset = SelectedMarkerTopOffset + (SelectedMarkerStep * static_cast<float>(markerIndex));
                auxGeom->DrawSphere(
                    position + AZ::Vector3(0.0f, 0.0f, heightOffset),
                    SelectedRingSphereRadius,
                    trainerPromptColor);
            }
        }

        if (isSelected && m_stateComponent->IsAlive())
        {
            const AZ::Color selectedAccent = AZ::Color(1.0f, 0.95f, 0.25f, 1.0f);
            for (int segmentIndex = 0; segmentIndex < SelectedRingSegments; ++segmentIndex)
            {
                const float angleRadians = (AZ::Constants::TwoPi / static_cast<float>(SelectedRingSegments)) *
                    static_cast<float>(segmentIndex);
                const AZ::Vector3 ringOffset(
                    AZStd::cos(angleRadians) * SelectedRingRadius,
                    AZStd::sin(angleRadians) * SelectedRingRadius,
                    0.08f);
                auxGeom->DrawSphere(position + ringOffset, SelectedRingSphereRadius, selectedAccent);
            }

            for (int markerIndex = 0; markerIndex < SelectedMarkerSteps; ++markerIndex)
            {
                const float heightOffset = SelectedMarkerTopOffset + (SelectedMarkerStep * static_cast<float>(markerIndex));
                auxGeom->DrawSphere(
                    position + AZ::Vector3(0.0f, 0.0f, heightOffset),
                    SelectedRingSphereRadius,
                    selectedAccent);
            }
        }
    }
} // namespace NpcAi

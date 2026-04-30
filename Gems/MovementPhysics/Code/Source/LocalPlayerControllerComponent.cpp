#include <MovementPhysics/LocalPlayerControllerComponent.h>

#include <Atom/RPI.Public/AuxGeom/AuxGeomDraw.h>
#include <Atom/RPI.Public/AuxGeom/AuxGeomFeatureProcessorInterface.h>
#include <Atom/RPI.Public/Scene.h>
#include <AtomLyIntegration/CommonFeatures/Material/MaterialComponentBus.h>
#include <AtomLyIntegration/CommonFeatures/Material/MaterialComponentConstants.h>
#include <AtomLyIntegration/CommonFeatures/Mesh/MeshComponentBus.h>
#include <AtomLyIntegration/CommonFeatures/Mesh/MeshComponentConstants.h>
#include <AzCore/Asset/AssetManagerBus.h>
#include <AzCore/Component/ComponentApplicationBus.h>
#include <AzCore/Component/TransformBus.h>
#include <AzCore/Component/Entity.h>
#include <AzCore/Math/Aabb.h>
#include <AzCore/Math/Color.h>
#include <AzCore/Math/MathUtils.h>
#include <AzCore/Math/Transform.h>
#include <AzCore/Math/Vector2.h>
#include <AzCore/Serialization/SerializeContext.h>
#include <AzFramework/API/ApplicationAPI.h>
#include <AzFramework/Components/NonUniformScaleComponent.h>
#include <AzFramework/Components/TransformComponent.h>
#include <AzFramework/Entity/GameEntityContextBus.h>
#include <AzFramework/Input/Channels/InputChannel.h>
#include <AzFramework/Input/Devices/Keyboard/InputDeviceKeyboard.h>
#include <AzFramework/Input/Devices/Mouse/InputDeviceMouse.h>
#include <AzFramework/Physics/CharacterBus.h>
#include <AzFramework/Physics/SystemBus.h>
#include <AzFramework/Viewport/ViewportBus.h>
#include <Atom/RPI.Public/ViewportContext.h>
#include <Atom/RPI.Public/ViewportContextBus.h>
#include <GameCore/GameCoreInterface.h>
#include <PhysX/CharacterGameplayBus.h>
#include <imgui/imgui.h>

namespace MovementPhysics
{
    namespace
    {
        constexpr float ValidationFloorZ = 0.0f;
        constexpr float ValidationFloorExtent = 460.0f;
        constexpr float ValidationFloorExtentY = 270.0f;
        constexpr float ValidationMarkerZ = 0.08f;
        constexpr float ValidationSpawnMarkerRadius = 0.18f;
        constexpr float ValidationSpawnX = 232.0f;
        constexpr float ValidationSpawnY = 130.0f;
        constexpr float EncounterAnchorX = 380.0f;
        constexpr float EncounterAnchorY = 231.0f;
        constexpr float MoveSpeedUnitsPerSecond = 6.0f;
        constexpr float BackpedalSpeedFactor = 0.62f;
        constexpr float SubmitIntervalSeconds = 0.18f;
        constexpr float CorrectionSnapDistance = 2.25f;
        constexpr float CorrectionBlendRate = 2.85f;
        constexpr float CorrectionDeadZoneDistance = 0.16f;
        constexpr float CorrectionEpsilon = 0.002f;
        constexpr float CharacterBaseSnapZ = 0.05f;
        constexpr float AvatarTurnRate = 2.75f;
        constexpr float CameraEncounterAnchorX = 232.0f;
        constexpr float CameraEncounterAnchorY = 130.0f;
        constexpr float CameraPivotOffsetZ = 1.42f;
        constexpr float CameraLookAheadDistance = 0.82f;
        constexpr float CameraLookLiftZ = 0.16f;
        constexpr float CameraYawSensitivity = 0.01f;
        constexpr float CameraPitchSensitivity = 0.01f;
        constexpr float CameraDefaultFollowDistance = 4.65f;
        constexpr float CameraMinFollowDistance = 4.2f;
        constexpr float CameraMaxFollowDistance = 6.0f;
        constexpr float CameraDefaultPitchRadians = -0.44f;
        constexpr float CameraMinPitchRadians = -0.95f;
        constexpr float CameraMaxPitchRadians = -0.26f;
        constexpr float CameraFloorFallbackZ = ValidationFloorZ + 0.30f;
        constexpr float AvatarFootHeight = 0.08f;
        constexpr float AvatarAnkleHeight = 0.30f;
        constexpr float AvatarKneeHeight = 0.70f;
        constexpr float AvatarHipHeight = 1.00f;
        constexpr float AvatarWaistHeight = 1.16f;
        constexpr float AvatarChestHeight = 1.40f;
        constexpr float AvatarShoulderHeight = 1.54f;
        constexpr float AvatarHeadHeight = 1.82f;
        constexpr float AvatarFootSpacing = 0.18f;
        constexpr float AvatarHipSpacing = 0.12f;
        constexpr float AvatarShoulderOffset = 0.27f;
        constexpr float AvatarPelvisRadius = 0.18f;
        constexpr float AvatarTorsoRadius = 0.23f;
        constexpr float AvatarChestRadius = 0.28f;
        constexpr float AvatarHeadRadius = 0.17f;
        constexpr float AvatarLimbRadius = 0.10f;
        constexpr float AvatarArmRadius = 0.09f;
        constexpr float AvatarSelectionRingRadius = 0.68f;
        constexpr float AvatarSelectionRingSphereRadius = 0.06f;
        constexpr int AvatarSelectionRingSegments = 12;
        constexpr float AvatarStrideRate = 8.5f;
        constexpr float AvatarStrideReach = 0.22f;
        constexpr float AvatarIdleTurnRate = 4.6f;
        constexpr float AvatarMoveTurnRate = 10.0f;
        constexpr float AvatarStrideBlendRate = 7.5f;
        constexpr float PoseLogDistanceThreshold = 0.75f;
        constexpr const char* StonewakeGroundEntityName = "Ground";
        constexpr const char* StonewakeBaseGroundModelAssetPath = "objects/groudplane/groundplane_512x512m.fbx.azmodel";
        constexpr const char* StonewakeSurfacePlaneModelAssetPath = "objects/shaderball/ground_plane_4x4m.fbx.azmodel";
        constexpr const char* StonewakeGroundMaterialAssetPaths[] = {
            "content/art/materials/mat_stonewake_grass_lush.azmaterial",
            "content/art/materials/mat_stonewake_grass_worn.azmaterial",
            "content/art/materials/mat_stonewake_rocky_ground.azmaterial",
        };

        bool IsInsideWestApproachObstacle(const AZ::Vector3& position)
        {
            return position.GetX() >= 355.0f && position.GetX() <= 367.0f &&
                position.GetY() >= 152.0f && position.GetY() <= 166.0f;
        }

        AZ::Vector3 ResolveMovementCollision(const AZ::Vector3& currentPosition, const AZ::Vector3& requestedPosition)
        {
            AZ::Vector3 clampedPosition = requestedPosition;
            clampedPosition.SetX(AZ::GetClamp(clampedPosition.GetX(), 0.0f, ValidationFloorExtent));
            clampedPosition.SetY(AZ::GetClamp(clampedPosition.GetY(), 0.0f, ValidationFloorExtentY));
            return IsInsideWestApproachObstacle(clampedPosition) ? currentPosition : clampedPosition;
        }

        float WrapAngleRadians(float angleRadians)
        {
            while (angleRadians > AZ::Constants::Pi)
            {
                angleRadians -= AZ::Constants::TwoPi;
            }
            while (angleRadians < -AZ::Constants::Pi)
            {
                angleRadians += AZ::Constants::TwoPi;
            }
            return angleRadians;
        }

        float StepAngleTowards(float currentRadians, float targetRadians, float maxStepRadians)
        {
            const float deltaRadians = WrapAngleRadians(targetRadians - currentRadians);
            return WrapAngleRadians(currentRadians + AZ::GetClamp(deltaRadians, -maxStepRadians, maxStepRadians));
        }

        AZ::Vector3 ForwardFromYaw(float yawRadians)
        {
            return AZ::Vector3(AZStd::cos(yawRadians), AZStd::sin(yawRadians), 0.0f);
        }

        AZ::Vector3 RightFromYaw(float yawRadians)
        {
            return AZ::Vector3(-AZStd::sin(yawRadians), AZStd::cos(yawRadians), 0.0f);
        }

        AZ::Vector3 BuildCameraForwardFromYaw(float yawRadians)
        {
            return AZ::Vector3(AZStd::cos(yawRadians), AZStd::sin(yawRadians), 0.0f);
        }
    } // namespace

    LocalPlayerControllerComponent::LocalPlayerControllerComponent()
        : AzFramework::InputChannelEventListener(AzFramework::InputChannelEventListener::GetPriorityFirst())
    {
    }

    void LocalPlayerControllerComponent::Reflect(AZ::ReflectContext* context)
    {
        if (auto* serializeContext = azrtti_cast<AZ::SerializeContext*>(context))
        {
            serializeContext->Class<LocalPlayerControllerComponent, AZ::Component>()
                ->Version(0);
        }
    }

    void LocalPlayerControllerComponent::GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided)
    {
        provided.push_back(AZ_CRC_CE("LocalPlayerControllerService"));
    }

    void LocalPlayerControllerComponent::GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible)
    {
        incompatible.push_back(AZ_CRC_CE("LocalPlayerControllerService"));
    }

    void LocalPlayerControllerComponent::GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required)
    {
        required.push_back(AZ_CRC_CE("TransformService"));
    }

    void LocalPlayerControllerComponent::GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType&)
    {
    }

    void LocalPlayerControllerComponent::Activate()
    {
        AzFramework::InputChannelEventListener::Connect();
        m_inputListening = true;
        AZ::TickBus::Handler::BusConnect();
    }

    void LocalPlayerControllerComponent::Deactivate()
    {
        DestroyStonewakeMaterialSurfaces();
        AZ::TickBus::Handler::BusDisconnect();
        AzFramework::InputChannelEventListener::Disconnect();
    }

    void LocalPlayerControllerComponent::OnTick(float deltaTime, AZ::ScriptTimePoint)
    {
        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        InitializeFromWorldState();
        UpdateGroundingState();

        if (m_requestDisconnect)
        {
            AZ_Printf("amandacore", "client.disconnect_requested");
            gameCore->DisconnectWorld();
            m_requestDisconnect = false;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            m_cachedFinalPoseValid = false;
            return;
        }

        if (m_requestReconnect)
        {
            AZ_Printf("amandacore", "client.reconnect_requested");
            if (gameCore->ReconnectWorld())
            {
                const auto& session = gameCore->GetClientWorldState().m_session;
                ApplyWorldPosition(
                    static_cast<float>(session.m_position.m_x),
                    static_cast<float>(session.m_position.m_y),
                    static_cast<float>(session.m_position.m_z));
                AZ_Printf(
                    "amandacore",
                    "client.reconnect_completed token=%s position=(%.3f, %.3f, %.3f)",
                    session.m_worldSessionToken.c_str(),
                    session.m_position.m_x,
                    session.m_position.m_y,
                    session.m_position.m_z);
                m_pendingCameraReset = true;
                m_avatarStridePhase = 0.0f;
                m_avatarStrideBlend = 0.0f;
            }
            m_requestReconnect = false;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            return;
        }

        if (m_requestQuit)
        {
            AZ_Printf("amandacore", "client.quit_requested");
            AzFramework::ApplicationRequests::Bus::Broadcast(
                &AzFramework::ApplicationRequests::Bus::Events::ExitMainLoop);
            m_requestQuit = false;
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        if (worldState.m_session.m_worldSessionToken != m_lastWorldSessionToken)
        {
            m_lastWorldSessionToken = worldState.m_session.m_worldSessionToken;
            m_pendingServerDelta = AZ::Vector2::CreateZero();
            m_pendingServerCorrection = AZ::Vector2::CreateZero();
            ApplyWorldPosition(
                static_cast<float>(worldState.m_session.m_position.m_x),
                static_cast<float>(worldState.m_session.m_position.m_y),
                static_cast<float>(worldState.m_session.m_position.m_z));
            m_pendingCameraReset = true;
            m_avatarStridePhase = 0.0f;
            m_avatarStrideBlend = 0.0f;
            m_stonewakeGroundMaterialApplied = false;
            m_loggedStonewakeGroundMaterialMissing = false;
            DestroyStonewakeMaterialSurfaces();
        }

        const bool orbiting = m_cameraOrbitModeActive;
        const bool dualMouseMoveActive = orbiting && m_leftMouseHeld;
        float cameraYawRadians = orbiting
            ? WrapAngleRadians(m_avatarFacingRadians + m_cameraOrbitYawRadians)
            : m_avatarFacingRadians;
        m_cameraYawRadians = cameraYawRadians;

        const float turnInput = (m_strafeLeft ? 1.0f : 0.0f) - (m_strafeRight ? 1.0f : 0.0f);
        AZ::Vector2 movementIntent = AZ::Vector2::CreateZero();
        if (m_moveForward)
        {
            movementIntent += AZ::Vector2(1.0f, 0.0f);
        }
        if (m_moveBackward)
        {
            movementIntent += AZ::Vector2(-1.0f, 0.0f);
        }
        if (dualMouseMoveActive)
        {
            movementIntent += AZ::Vector2(1.0f, 0.0f);
        }

        if (orbiting)
        {
            if (m_strafeLeft)
            {
                movementIntent += AZ::Vector2(0.0f, 1.0f);
            }
            if (m_strafeRight)
            {
                movementIntent += AZ::Vector2(0.0f, -1.0f);
            }

            if (movementIntent.GetLengthSq() > 0.0f)
            {
                m_avatarFacingRadians = cameraYawRadians;
                m_cameraOrbitYawRadians = 0.0f;
                cameraYawRadians = m_avatarFacingRadians;
                m_cameraYawRadians = cameraYawRadians;
                m_chaseLockActive = true;
            }
            else
            {
                m_chaseLockActive = false;
            }
        }
        else if (AZ::GetAbs(turnInput) > 0.001f)
        {
            m_avatarFacingRadians = WrapAngleRadians(m_avatarFacingRadians + (turnInput * AvatarTurnRate * deltaTime));
            cameraYawRadians = m_avatarFacingRadians;
            m_cameraYawRadians = cameraYawRadians;
            m_chaseLockActive = true;
        }
        else if (!orbiting)
        {
            m_chaseLockActive = true;
        }

        if (!orbiting)
        {
            m_cameraOrbitYawRadians = 0.0f;
            cameraYawRadians = m_avatarFacingRadians;
            m_cameraYawRadians = cameraYawRadians;
        }

        const AZ::Vector3 groundedPosition = GetPresentationPosition();
        AZ::Vector3 currentPosition = groundedPosition;
        if (m_cachedFinalPoseValid)
        {
            currentPosition.SetX(m_cachedFinalPresentationPosition.GetX());
            currentPosition.SetY(m_cachedFinalPresentationPosition.GetY());
        }

        AZ::Vector3 finalPresentationPosition = currentPosition;
        AZ::Vector2 visualPlanarDelta = AZ::Vector2::CreateZero();

        if (movementIntent.GetLengthSq() > 0.0f)
        {
            if (!m_loggedFirstMovementInput)
            {
                m_loggedFirstMovementInput = true;
                AZ_Printf("amandacore", "client.first_movement_input_received");
            }

            AZ::Vector2 move2d = AZ::Vector2::CreateZero();
            if (orbiting)
            {
                movementIntent.Normalize();
                const AZ::Vector2 forward(AZStd::cos(cameraYawRadians), AZStd::sin(cameraYawRadians));
                const AZ::Vector2 right(-AZStd::sin(cameraYawRadians), AZStd::cos(cameraYawRadians));
                move2d = ((forward * movementIntent.GetX()) + (right * movementIntent.GetY())) *
                    (MoveSpeedUnitsPerSecond * deltaTime);
            }
            else
            {
                const float moveIntent = movementIntent.GetX();
                const AZ::Vector2 facingForward(AZStd::cos(m_avatarFacingRadians), AZStd::sin(m_avatarFacingRadians));
                const float moveSpeed = moveIntent > 0.0f ? MoveSpeedUnitsPerSecond : (MoveSpeedUnitsPerSecond * BackpedalSpeedFactor);
                move2d = facingForward * (moveIntent * moveSpeed * deltaTime);
            }

            AZ::Vector3 requestedPosition = currentPosition + AZ::Vector3(move2d.GetX(), move2d.GetY(), 0.0f);
            requestedPosition = ResolveMovementCollision(currentPosition, requestedPosition);
            const AZ::Vector3 appliedDelta = requestedPosition - currentPosition;
            visualPlanarDelta = AZ::Vector2(appliedDelta.GetX(), appliedDelta.GetY());
            m_pendingServerDelta += visualPlanarDelta;
            finalPresentationPosition = requestedPosition;
        }

        if (m_pendingServerCorrection.GetLengthSq() > CorrectionEpsilon * CorrectionEpsilon)
        {
            const float correctionBlend = AZ::GetClamp(deltaTime * CorrectionBlendRate, 0.0f, 1.0f);
            const AZ::Vector2 correctionStep = m_pendingServerCorrection * correctionBlend;
            finalPresentationPosition.SetX(finalPresentationPosition.GetX() + correctionStep.GetX());
            finalPresentationPosition.SetY(finalPresentationPosition.GetY() + correctionStep.GetY());
            m_pendingServerCorrection -= correctionStep;
            if (m_pendingServerCorrection.GetLengthSq() <= CorrectionEpsilon * CorrectionEpsilon)
            {
                m_pendingServerCorrection = AZ::Vector2::CreateZero();
            }
        }

        m_submitAccumulator += deltaTime;
        if (m_submitAccumulator >= SubmitIntervalSeconds && m_pendingServerDelta.GetLengthSq() >= 0.0001f)
        {
            m_submitAccumulator = AZStd::fmod(m_submitAccumulator, SubmitIntervalSeconds);
            const AZ::Vector2 submittedDelta = m_pendingServerDelta;
            m_pendingServerDelta = AZ::Vector2::CreateZero();

            if (gameCore->SubmitMove(submittedDelta.GetX(), submittedDelta.GetY()))
            {
                const auto& authoritativeSession = gameCore->GetClientWorldState().m_session;
                const AZ::Vector2 authoritativePosition(
                    static_cast<float>(authoritativeSession.m_position.m_x),
                    static_cast<float>(authoritativeSession.m_position.m_y));

                const AZ::Vector3 locallyPredictedPosition = finalPresentationPosition;
                const AZ::Vector2 localPlanarPosition(locallyPredictedPosition.GetX(), locallyPredictedPosition.GetY());
                const AZ::Vector2 correctionVector = authoritativePosition - localPlanarPosition;
                const float correctionDistance = correctionVector.GetLength();
                if (correctionDistance > CorrectionSnapDistance)
                {
                    finalPresentationPosition.SetX(authoritativePosition.GetX());
                    finalPresentationPosition.SetY(authoritativePosition.GetY());
                    finalPresentationPosition.SetZ(locallyPredictedPosition.GetZ());
                    m_pendingServerCorrection = AZ::Vector2::CreateZero();
                    SetCharacterBasePosition(finalPresentationPosition);
                    AZ_Printf(
                        "amandacore",
                        "client.planar_reconciliation_applied localXY=(%.3f, %.3f) authoritativeXY=(%.3f, %.3f) mode=snap",
                        localPlanarPosition.GetX(),
                        localPlanarPosition.GetY(),
                        authoritativePosition.GetX(),
                        authoritativePosition.GetY());
                }
                else if (correctionDistance > CorrectionDeadZoneDistance)
                {
                    m_pendingServerCorrection = correctionVector;
                    AZ_Printf(
                        "amandacore",
                        "client.planar_reconciliation_applied localXY=(%.3f, %.3f) authoritativeXY=(%.3f, %.3f) mode=queued_blend",
                        localPlanarPosition.GetX(),
                        localPlanarPosition.GetY(),
                        authoritativePosition.GetX(),
                        authoritativePosition.GetY());
                }
                else
                {
                    m_pendingServerCorrection = AZ::Vector2::CreateZero();
                }
            }
        }

        m_cachedFinalPresentationPosition = finalPresentationPosition;
        m_cachedFinalAvatarFacingRadians = m_avatarFacingRadians;
        m_cachedFinalPoseValid = true;

        if (visualPlanarDelta.GetLengthSq() > 0.0001f)
        {
            if (!m_loggedMovementTranslationApplied)
            {
                m_loggedMovementTranslationApplied = true;
                AZ_Printf(
                    "amandacore",
                    "client.movement_translation_applied delta=(%.3f, %.3f, %.3f)",
                    visualPlanarDelta.GetX(),
                    visualPlanarDelta.GetY(),
                    0.0f);
            }
        }
        else
        {
            m_loggedMovementTranslationApplied = false;
        }

        const float finalBodyPoseDistance = m_loggedFinalBodyPose
            ? (m_cachedFinalPresentationPosition - m_lastLoggedFinalBodyPose).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedFinalBodyPose || finalBodyPoseDistance >= PoseLogDistanceThreshold)
        {
            m_loggedFinalBodyPose = true;
            m_lastLoggedFinalBodyPose = m_cachedFinalPresentationPosition;
            AZ_Printf(
                "amandacore",
                "client.final_body_pose pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                m_cachedFinalPresentationPosition.GetX(),
                m_cachedFinalPresentationPosition.GetY(),
                m_cachedFinalPresentationPosition.GetZ(),
                m_cachedFinalAvatarFacingRadians);
        }

        UpdateAvatarPresentation(deltaTime, visualPlanarDelta);
        SetCharacterBasePosition(m_cachedFinalPresentationPosition);
        SyncEntityTransformToCharacterBase(m_cachedFinalPresentationPosition);
        UpdateCameraComponent();
        ApplyStonewakeGroundMaterial();
        SpawnStonewakeMaterialSurfaces();
        DrawValidationArena();
        DrawLocalPlayerProxy();
    }

    bool LocalPlayerControllerComponent::OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel)
    {
        const auto& channelId = inputChannel.GetInputChannelId();
        const bool active = inputChannel.IsActive();
        const bool imguiWantsMouse = ImGui::GetIO().WantCaptureMouse || ImGui::IsAnyItemActive();

        if (ImGui::GetIO().WantTextInput)
        {
            if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericW)
            {
                m_moveForward = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericS)
            {
                m_moveBackward = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericA)
            {
                m_strafeLeft = false;
            }
            else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericD)
            {
                m_strafeRight = false;
            }
            return false;
        }

        if (channelId == AzFramework::InputDeviceMouse::Button::Left)
        {
            if (imguiWantsMouse)
            {
                m_leftMouseHeld = false;
                return false;
            }

            m_leftMouseHeld = active;
            return false;
        }

        if (channelId == AzFramework::InputDeviceMouse::Button::Right)
        {
            if (inputChannel.IsStateBegan())
            {
                if (imguiWantsMouse)
                {
                    m_cameraOrbitModeActive = false;
                    m_chaseLockActive = true;
                    return false;
                }

                m_cameraOrbitModeActive = true;
                m_chaseLockActive = false;
                return true;
            }

            if (inputChannel.IsStateEnded() || !active || imguiWantsMouse)
            {
                const bool wasOrbiting = m_cameraOrbitModeActive;
                m_cameraOrbitModeActive = false;
                if (wasOrbiting)
                {
                    m_cameraOrbitYawRadians = 0.0f;
                    m_cameraPitchRadians = CameraDefaultPitchRadians;
                    m_cameraYawRadians = m_avatarFacingRadians;
                    m_chaseLockActive = true;
                }
                return wasOrbiting;
            }

            return m_cameraOrbitModeActive;
        }

        if (channelId == AzFramework::InputDeviceMouse::Movement::X)
        {
            if (imguiWantsMouse)
            {
                return false;
            }

            if (!m_cameraOrbitModeActive)
            {
                if (auto* gameCore = GameCore::IGameCoreRequests::Get())
                {
                    return gameCore->GetClientWorldState().m_worldConnected;
                }
                return false;
            }

            m_cameraOrbitYawRadians = WrapAngleRadians(m_cameraOrbitYawRadians - (inputChannel.GetValue() * CameraYawSensitivity));
            m_cameraYawRadians = WrapAngleRadians(m_avatarFacingRadians + m_cameraOrbitYawRadians);
            return true;
        }

        if (channelId == AzFramework::InputDeviceMouse::Movement::Y)
        {
            if (imguiWantsMouse)
            {
                return false;
            }

            if (!m_cameraOrbitModeActive)
            {
                if (auto* gameCore = GameCore::IGameCoreRequests::Get())
                {
                    return gameCore->GetClientWorldState().m_worldConnected;
                }
                return false;
            }

            m_cameraPitchRadians = AZ::GetClamp(
                m_cameraPitchRadians - (inputChannel.GetValue() * CameraPitchSensitivity),
                CameraMinPitchRadians,
                CameraMaxPitchRadians);
            return true;
        }

        if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericW)
        {
            m_moveForward = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericS)
        {
            m_moveBackward = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericA)
        {
            m_strafeLeft = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericD)
        {
            m_strafeRight = active;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericX && inputChannel.IsStateBegan())
        {
            m_requestDisconnect = true;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericR && inputChannel.IsStateBegan())
        {
            m_requestReconnect = true;
        }
        else if (channelId == AzFramework::InputDeviceKeyboard::Key::AlphanumericQ && inputChannel.IsStateBegan())
        {
            m_requestQuit = true;
        }

        return false;
    }

    void LocalPlayerControllerComponent::InitializeFromWorldState()
    {
        if (m_initialized)
        {
            return;
        }

        auto* gameCore = GameCore::IGameCoreRequests::Get();
        if (!gameCore)
        {
            return;
        }

        const auto& worldState = gameCore->GetClientWorldState();
        if (!worldState.m_worldConnected)
        {
            return;
        }

        ApplyWorldPosition(
            static_cast<float>(worldState.m_session.m_position.m_x),
            static_cast<float>(worldState.m_session.m_position.m_y),
            static_cast<float>(worldState.m_session.m_position.m_z));
        if (!m_loggedInputReady)
        {
            m_loggedInputReady = true;
            AZ_Printf(
                "amandacore",
                "client.input_ready entity=LocalPlayer listening=%s priority=%d focusHint=click_game_window_if_input_is_still_inactive",
                m_inputListening ? "true" : "false",
                GetPriority());
        }
        if (!m_loggedInputHelp)
        {
            m_loggedInputHelp = true;
            AZ_Printf(
                "amandacore",
                "client.input_help move=W/S turn=A/D camera=RMB target=Tab/LMB dualMove=LMB+RMB combat=F/1/2 disconnect=X reconnect=R quit=Q");
        }
        if (!m_loggedLocomotionMode)
        {
            m_loggedLocomotionMode = true;
            AZ_Printf(
                "amandacore",
                "client.locomotion_mode_applied mode=avatar_chase turn=A/D orbitMove=RMB+WASD freeFly=false");
        }
        if (!m_loggedStableLocomotionMode)
        {
            m_loggedStableLocomotionMode = true;
            AZ_Printf("amandacore", "client.locomotion_mode_active mode=avatar_chase_stable");
        }
        if (!m_loggedGrounded)
        {
            m_loggedGrounded = true;
            AZ_Printf(
                "amandacore",
                "client.player_grounded spawn=(%.3f, %.3f, %.3f) floorZ=%.3f",
                static_cast<float>(worldState.m_session.m_position.m_x),
                static_cast<float>(worldState.m_session.m_position.m_y),
                ValidationFloorZ,
                ValidationFloorZ);
        }
        m_initialized = true;
    }

    void LocalPlayerControllerComponent::ResetCameraToEncounterFrame(const AZ::Vector3& playerPosition)
    {
        const AZ::Vector2 encounterDirection(
            CameraEncounterAnchorX - playerPosition.GetX(),
            CameraEncounterAnchorY - playerPosition.GetY());
        if (encounterDirection.GetLengthSq() > 0.001f)
        {
            m_avatarFacingRadians = WrapAngleRadians(AZStd::atan2(encounterDirection.GetY(), encounterDirection.GetX()));
        }

        m_cameraOrbitYawRadians = 0.0f;
        m_cameraYawRadians = m_avatarFacingRadians;
        m_cameraPitchRadians = CameraDefaultPitchRadians;
        m_cameraFollowDistance = CameraDefaultFollowDistance;
        m_cameraOrbitModeActive = false;
        m_chaseLockActive = true;
        m_loggedCameraSource = false;

        AZ_Printf(
            "amandacore",
            "client.camera_spawn_frame_locked yawRadians=%.3f pitchRadians=%.3f followDistance=%.2f",
            m_cameraYawRadians,
            m_cameraPitchRadians,
            m_cameraFollowDistance);
    }

    void LocalPlayerControllerComponent::UpdateCameraComponent()
    {
        if (m_defaultViewportId == AzFramework::InvalidViewportId)
        {
            if (auto* viewportRequests = AZ::RPI::ViewportContextRequests::Get())
            {
                if (AZ::RPI::ViewportContextPtr viewportContext = viewportRequests->GetDefaultViewportContext())
                {
                    m_defaultViewportId = viewportContext->GetId();
                }
            }
        }

        if (m_defaultViewportId == AzFramework::InvalidViewportId)
        {
            return;
        }

        if (!m_loggedCameraAttachment)
        {
            m_loggedCameraAttachment = true;
            AZ_Printf("amandacore", "client.camera_attached entity=LocalPlayer attached=true freeFly=false");
        }

        if (!m_loggedCameraMode)
        {
            m_loggedCameraMode = true;
            AZ_Printf("amandacore", "client.camera_mode_applied mode=third_person_chase");
        }

        const AZ::Vector3 playerPosition = m_cachedFinalPoseValid ? m_cachedFinalPresentationPosition : GetPresentationPosition();
        if (m_pendingCameraReset)
        {
            ResetCameraToEncounterFrame(playerPosition);
            m_pendingCameraReset = false;
        }

        AZ::Transform playerWorldTransform = AZ::Transform::CreateIdentity();
        playerWorldTransform.SetTranslation(playerPosition);
        playerWorldTransform.SetRotation(AZ::Quaternion::CreateRotationZ(m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians));

        m_cameraFollowDistance = AZ::GetClamp(m_cameraFollowDistance, CameraMinFollowDistance, CameraMaxFollowDistance);
        const AZ::Vector3 localForward = BuildCameraForwardFromYaw(m_cameraOrbitYawRadians);
        const AZ::Vector3 pivotLocal(0.0f, 0.0f, CameraPivotOffsetZ);
        const AZ::Vector3 targetLocal =
            pivotLocal + (localForward * CameraLookAheadDistance) + AZ::Vector3(0.0f, 0.0f, CameraLookLiftZ);
        const float horizontalDistance = AZStd::cos(-m_cameraPitchRadians) * m_cameraFollowDistance;
        const float verticalDistance = AZStd::sin(-m_cameraPitchRadians) * m_cameraFollowDistance;
        const AZ::Vector3 desiredCameraLocalPosition =
            pivotLocal - (localForward * horizontalDistance) + AZ::Vector3(0.0f, 0.0f, 0.50f + verticalDistance);

        const AZ::Vector3 anchorWorldPosition = playerWorldTransform.TransformPoint(pivotLocal);
        AZ::Vector3 resolvedCameraWorldPosition = playerWorldTransform.TransformPoint(desiredCameraLocalPosition);
        const AZ::Vector3 desiredCameraWorldPosition = resolvedCameraWorldPosition;
        const AZ::Vector3 cameraDelta = desiredCameraWorldPosition - anchorWorldPosition;
        const float desiredDistance = cameraDelta.GetLength();

        bool sceneQueryResolved = false;
        if (!m_loggedCameraCollisionDisabled)
        {
            m_loggedCameraCollisionDisabled = true;
            AZ_Printf(
                "amandacore",
                "client.camera_scene_query_mode=disabled_for_0_2 fixedFollowDistance=%.3f",
                m_cameraFollowDistance);
        }

        if (!sceneQueryResolved && resolvedCameraWorldPosition.GetZ() < CameraFloorFallbackZ)
        {
            const float candidateZ = resolvedCameraWorldPosition.GetZ();
            resolvedCameraWorldPosition.SetZ(CameraFloorFallbackZ);
            AZ_Printf(
                "amandacore",
                "client.camera_floor_clamp_applied candidateZ=%.3f resolvedZ=%.3f floorZ=%.3f",
                candidateZ,
                resolvedCameraWorldPosition.GetZ(),
                ValidationFloorZ);
        }

        const AZ::Transform inversePlayerWorld = playerWorldTransform.GetInverse();
        const AZ::Vector3 resolvedCameraLocalPosition = inversePlayerWorld.TransformPoint(resolvedCameraWorldPosition);
        const AZ::Transform cameraLocalTransform = AZ::Transform::CreateLookAt(
            resolvedCameraLocalPosition,
            targetLocal,
            AZ::Transform::Axis::YPositive);
        const AZ::Transform finalCameraWorldTransform = playerWorldTransform * cameraLocalTransform;
        AzFramework::ViewportRequestBus::Event(
            m_defaultViewportId,
            &AzFramework::ViewportRequestBus::Events::SetCameraTransform,
            finalCameraWorldTransform);

        if (auto* gameCore = GameCore::IGameCoreRequests::Get())
        {
            GameCore::ClientCameraState cameraState;
            cameraState.m_ready = true;
            cameraState.m_worldTransform = finalCameraWorldTransform;
            cameraState.m_verticalFovDegrees = 60.0f;
            gameCore->SetCameraState(cameraState);
        }

        if (!m_loggedCameraReady)
        {
            m_loggedCameraReady = true;
            AZ_Printf(
                "amandacore",
                "client.camera_ready entity=LocalPlayer viewportId=%d active=true attached=true yawRadians=%.3f",
                m_defaultViewportId,
                m_cameraYawRadians);
        }

        if (!m_loggedCameraEntity)
        {
            m_loggedCameraEntity = true;
            AZ_Printf(
                "amandacore",
                "client.active_camera_entity name=DefaultViewport id=%d parent=LocalPlayer",
                m_defaultViewportId);
        }

        if (!m_loggedCameraAnchor)
        {
            m_loggedCameraAnchor = true;
            const AZStd::string anchorEntityId = GetEntityId().ToString();
            AZ_Printf(
                "amandacore",
                "client.camera_anchor_entity name=LocalPlayer id=%s",
                anchorEntityId.c_str());
        }

        if (!m_loggedDetachedCameraDisabled)
        {
            m_loggedDetachedCameraDisabled = true;
            AZ_Printf("amandacore", "client.detached_camera_path_active=false");
        }

        if (!m_loggedViewportCameraOwner)
        {
            m_loggedViewportCameraOwner = true;
            AZ_Printf("amandacore", "client.viewport_camera_owned_by=LocalPlayer");
        }

        if (!m_loggedCameraTransformWriter)
        {
            m_loggedCameraTransformWriter = true;
            AZ_Printf("amandacore", "client.camera_transform_writer=LocalPlayer");
        }

        if (!m_loggedCameraSource)
        {
            m_loggedCameraSource = true;
            if (!sceneQueryResolved)
            {
                AZ_Printf(
                    "amandacore",
                    "client.camera_scene_query_resolved hit=false resolvedDistance=%.3f",
                    desiredDistance);
            }
            AZ_Printf(
                "amandacore",
                "client.avatar_root_transform pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                playerPosition.GetX(),
                playerPosition.GetY(),
                playerPosition.GetZ(),
                m_avatarFacingRadians);
            AZ_Printf(
                "amandacore",
                "client.camera_source_transform anchorPos=(%.3f, %.3f, %.3f) anchorYaw=%.3f",
                anchorWorldPosition.GetX(),
                anchorWorldPosition.GetY(),
                anchorWorldPosition.GetZ(),
                m_cameraYawRadians);
        }

        const float cameraSourceDistance = m_loggedCameraSourceFollow
            ? (anchorWorldPosition - m_lastLoggedCameraSourcePosition).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedCameraSourceFollow || cameraSourceDistance >= PoseLogDistanceThreshold)
        {
            m_loggedCameraSourceFollow = true;
            m_lastLoggedCameraSourcePosition = anchorWorldPosition;
            AZ_Printf(
                "amandacore",
                "client.camera_source_transform pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                anchorWorldPosition.GetX(),
                anchorWorldPosition.GetY(),
                anchorWorldPosition.GetZ(),
                m_cameraYawRadians);
        }

        if (!m_loggedMouseLookState || m_lastLoggedMouseLookState != m_cameraOrbitModeActive)
        {
            m_loggedMouseLookState = true;
            m_lastLoggedMouseLookState = m_cameraOrbitModeActive;
            AZ_Printf(
                "amandacore",
                "client.mouse_look_active=%s",
                m_cameraOrbitModeActive ? "true" : "false");
        }

        if (!m_loggedOrbitState || m_lastLoggedOrbitState != m_cameraOrbitModeActive)
        {
            m_loggedOrbitState = true;
            m_lastLoggedOrbitState = m_cameraOrbitModeActive;
            AZ_Printf(
                "amandacore",
                "client.orbit_mode_active=%s",
                m_cameraOrbitModeActive ? "true" : "false");
        }

        if (!m_loggedChaseLockState || m_lastLoggedChaseLockState != m_chaseLockActive)
        {
            m_loggedChaseLockState = true;
            m_lastLoggedChaseLockState = m_chaseLockActive;
            AZ_Printf(
                "amandacore",
                "client.chase_lock_active=%s",
                m_chaseLockActive ? "true" : "false");
        }
    }

    void LocalPlayerControllerComponent::UpdateGroundingState()
    {
        bool onGround = false;
        PhysX::CharacterGameplayRequestBus::EventResult(
            onGround,
            GetEntityId(),
            &PhysX::CharacterGameplayRequestBus::Events::IsOnGround);

        if (onGround && !m_loggedGroundMovementReady)
        {
            m_loggedGroundMovementReady = true;
            AZ_Printf("amandacore", "client.grounded_movement_ready onGround=true");
        }
    }

    void LocalPlayerControllerComponent::UpdateAvatarPresentation(
        float deltaTime,
        const AZ::Vector2& planarDelta)
    {
        const float deltaTimeSafe = deltaTime > 0.0001f ? deltaTime : 0.0001f;
        const float planarSpeed = planarDelta.GetLength() / deltaTimeSafe;
        const float strideTarget = planarSpeed > 0.05f ? 1.0f : 0.0f;
        m_avatarStrideBlend += (strideTarget - m_avatarStrideBlend) *
            AZ::GetClamp(deltaTime * AvatarStrideBlendRate, 0.0f, 1.0f);

        if (planarSpeed > 0.05f)
        {
            const float strideScale = AZ::GetClamp(planarSpeed / MoveSpeedUnitsPerSecond, 0.55f, 1.35f);
            m_avatarStridePhase = WrapAngleRadians(m_avatarStridePhase + (deltaTime * AvatarStrideRate * strideScale));
            return;
        }

    }

    AZ::Vector3 LocalPlayerControllerComponent::GetPresentationPosition() const
    {
        AZ::Vector3 basePosition = AZ::Vector3::CreateZero();
        Physics::CharacterRequestBus::EventResult(
            basePosition,
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::GetBasePosition);

        if (basePosition.GetLengthSq() > 0.0f)
        {
            return basePosition;
        }

        AZ::Vector3 transformPosition = AZ::Vector3::CreateZero();
        AZ::TransformBus::EventResult(transformPosition, GetEntityId(), &AZ::TransformBus::Events::GetWorldTranslation);
        return transformPosition;
    }

    void LocalPlayerControllerComponent::SyncEntityTransformToCharacterBase(const AZ::Vector3& basePosition)
    {
        AZ::Transform worldTransform = AZ::Transform::CreateIdentity();
        worldTransform.SetTranslation(basePosition);
        worldTransform.SetRotation(AZ::Quaternion::CreateRotationZ(m_avatarFacingRadians));
        AZ::TransformBus::Event(GetEntityId(), &AZ::TransformBus::Events::SetWorldTM, worldTransform);

        const float entitySyncDistance = m_loggedEntitySync
            ? (basePosition - m_lastLoggedEntitySyncPosition).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedEntitySync || entitySyncDistance >= PoseLogDistanceThreshold)
        {
            m_loggedEntitySync = true;
            m_lastLoggedEntitySyncPosition = basePosition;
            AZ_Printf(
                "amandacore",
                "client.entity_sync_applied pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                basePosition.GetX(),
                basePosition.GetY(),
                basePosition.GetZ(),
                m_avatarFacingRadians);
        }
    }

    void LocalPlayerControllerComponent::SetCharacterBasePosition(const AZ::Vector3& basePosition)
    {
        Physics::CharacterRequestBus::Event(
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::SetBasePosition,
            basePosition);
    }

    void LocalPlayerControllerComponent::ApplyWorldPosition(float x, float y, float z)
    {
        AZ::Vector3 currentBasePosition = AZ::Vector3::CreateZero();
        Physics::CharacterRequestBus::EventResult(
            currentBasePosition,
            GetEntityId(),
            &Physics::CharacterRequestBus::Events::GetBasePosition);

        const AZ::Vector3 requestedPosition(
            x,
            y,
            currentBasePosition.GetZ() == 0.0f ? CharacterBaseSnapZ : currentBasePosition.GetZ());
        if (AZ::GetAbs(z - ValidationFloorZ) > 0.001f)
        {
            AZ_Printf(
                "amandacore",
                "client.ground_snap_applied fromZ=%.3f toZ=%.3f",
                z,
                ValidationFloorZ);
        }

        SetCharacterBasePosition(requestedPosition);
        m_cachedFinalPresentationPosition = requestedPosition;
        m_cachedFinalAvatarFacingRadians = m_avatarFacingRadians;
        m_cachedFinalPoseValid = true;
    }

    void LocalPlayerControllerComponent::ApplyStonewakeGroundMaterial()
    {
        if (m_stonewakeGroundMaterialApplied)
        {
            return;
        }

        AZ::Data::AssetId materialAssetId;
        const char* resolvedMaterialPath = nullptr;
        for (const char* candidatePath : StonewakeGroundMaterialAssetPaths)
        {
            AZ::Data::AssetCatalogRequestBus::BroadcastResult(
                materialAssetId,
                &AZ::Data::AssetCatalogRequestBus::Events::GetAssetIdByPath,
                candidatePath,
                AZ::Data::AssetType{},
                false);
            if (materialAssetId.IsValid())
            {
                resolvedMaterialPath = candidatePath;
                break;
            }
        }

        AZ::EntityId groundEntityId;
        AZ::ComponentApplicationBus::Broadcast(
            &AZ::ComponentApplicationRequests::EnumerateEntities,
            [&groundEntityId](AZ::Entity* entity)
            {
                if (!groundEntityId.IsValid() && entity && entity->GetName() == StonewakeGroundEntityName)
                {
                    groundEntityId = entity->GetId();
                }
            });

        if (!materialAssetId.IsValid() || !groundEntityId.IsValid())
        {
            if (!m_loggedStonewakeGroundMaterialMissing)
            {
                m_loggedStonewakeGroundMaterialMissing = true;
                AZ_Warning(
                    "amandacore",
                    false,
                    "client.stonewake_ground_material_missing materialResolved=%s groundEntityResolved=%s",
                    materialAssetId.IsValid() ? "true" : "false",
                    groundEntityId.IsValid() ? "true" : "false");
            }
            return;
        }

        AZ::Render::MaterialComponentRequestBus::Event(
            groundEntityId,
            &AZ::Render::MaterialComponentRequestBus::Events::SetMaterialAssetIdOnDefaultSlot,
            materialAssetId);

        m_stonewakeGroundMaterialApplied = true;
        AZ_Printf(
            "amandacore",
            "client.stonewake_ground_material_applied entity=%s material=%s",
            StonewakeGroundEntityName,
            resolvedMaterialPath ? resolvedMaterialPath : "unknown");
    }

    void LocalPlayerControllerComponent::SpawnStonewakeMaterialSurfaces()
    {
        if (m_stonewakeMaterialSurfacesSpawned)
        {
            return;
        }

        auto resolveAssetIdByPath = [](const char* assetPath)
        {
            AZ::Data::AssetId assetId;
            AZ::Data::AssetCatalogRequestBus::BroadcastResult(
                assetId,
                &AZ::Data::AssetCatalogRequestBus::Events::GetAssetIdByPath,
                assetPath,
                AZ::Data::AssetType{},
                false);
            return assetId;
        };

        struct StonewakeMaterialSurfaceDefinition
        {
            const char* m_name;
            const char* m_modelAssetPath;
            const char* m_materialAssetPath;
            AZ::Vector3 m_position;
            AZ::Vector3 m_nonUniformScale;
            AZ::Quaternion m_rotation;
        };

        const AZ::Quaternion flatRotation = AZ::Quaternion::CreateIdentity();
        const AZ::Quaternion northWallRotation = AZ::Quaternion::CreateRotationX(AZ::Constants::Pi * 0.5f);
        const AZ::Quaternion eastWallRotation =
            AZ::Quaternion::CreateRotationY(AZ::Constants::Pi * -0.5f) * AZ::Quaternion::CreateRotationZ(AZ::Constants::Pi * 0.5f);
        const AZ::Quaternion roofRotation = AZ::Quaternion::CreateRotationY(-0.18f);

        const StonewakeMaterialSurfaceDefinition surfaceDefinitions[] = {
            {
                "Stonewake_TexturedTerrain_Base",
                StonewakeBaseGroundModelAssetPath,
                "content/art/materials/mat_stonewake_grass_lush.azmaterial",
                AZ::Vector3(230.0f, 135.0f, 0.018f),
                AZ::Vector3::CreateOne(),
                flatRotation,
            },
            {
                "Stonewake_Hearthwatch_CobbleYard",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_cobble_path.azmaterial",
                AZ::Vector3(232.0f, 130.0f, 0.205f),
                AZ::Vector3(24.0f, 15.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_WornGrass_WestApproach",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_grass_worn.azmaterial",
                AZ::Vector3(197.0f, 74.0f, 0.185f),
                AZ::Vector3(18.0f, 12.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_RockyGround_Ridge",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_rocky_ground.azmaterial",
                AZ::Vector3(361.0f, 157.0f, 0.185f),
                AZ::Vector3(22.0f, 13.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_MossGrass_EastVale",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_foliage_mossy_grass.azmaterial",
                AZ::Vector3(380.0f, 231.0f, 0.185f),
                AZ::Vector3(22.0f, 14.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_DirtRoad_Hearthwatch",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_dirt_path.azmaterial",
                AZ::Vector3(232.0f, 130.0f, 0.225f),
                AZ::Vector3(26.0f, 1.9f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_DirtRoad_Training",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_dirt_path.azmaterial",
                AZ::Vector3(197.0f, 74.0f, 0.225f),
                AZ::Vector3(20.0f, 1.85f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_DirtRoad_CentralVale",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_dirt_path.azmaterial",
                AZ::Vector3(313.0f, 81.0f, 0.225f),
                AZ::Vector3(20.0f, 2.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_DirtRoad_EastRise",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_dirt_path.azmaterial",
                AZ::Vector3(375.0f, 77.0f, 0.225f),
                AZ::Vector3(11.0f, 2.1f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_TrainingRing_Dirt",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_dirt_path.azmaterial",
                AZ::Vector3(268.0f, 145.0f, 0.235f),
                AZ::Vector3(4.8f, 4.8f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_FarmSoil_Valefurrow",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_valefurrow_farm_soil.azmaterial",
                AZ::Vector3(197.0f, 74.0f, 0.205f),
                AZ::Vector3(17.0f, 9.0f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_StreamWater_Crossing",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stream_water_placeholder.azmaterial",
                AZ::Vector3(313.0f, 81.0f, 0.240f),
                AZ::Vector3(19.0f, 1.65f, 1.0f),
                flatRotation,
            },
            {
                "Stonewake_ShoreMud_Crossing",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_stonewake_mud_dark.azmaterial",
                AZ::Vector3(313.0f, 81.0f, 0.200f),
                AZ::Vector3(20.0f, 3.2f, 1.0f),
                flatRotation,
            },
            {
                "Hearthwatch_CommandFloor_Wood",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_village_wood.azmaterial",
                AZ::Vector3(232.0f, 130.0f, 0.265f),
                AZ::Vector3(3.4f, 2.2f, 1.0f),
                flatRotation,
            },
            {
                "Hearthwatch_RoomFloor_Cobble",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_cobble_path.azmaterial",
                AZ::Vector3(222.0f, 121.0f, 0.265f),
                AZ::Vector3(2.8f, 2.1f, 1.0f),
                flatRotation,
            },
            {
                "Hearthwatch_WhitePlaster_Wall",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_village_plaster.azmaterial",
                AZ::Vector3(242.0f, 137.0f, 1.45f),
                AZ::Vector3(3.2f, 0.70f, 1.0f),
                northWallRotation,
            },
            {
                "Hearthwatch_CutStone_Wall",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_cut_stone.azmaterial",
                AZ::Vector3(254.0f, 130.0f, 1.35f),
                AZ::Vector3(2.8f, 0.65f, 1.0f),
                eastWallRotation,
            },
            {
                "Hearthwatch_ThatchRoof_Patch",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_thatch_roof.azmaterial",
                AZ::Vector3(232.0f, 130.0f, 2.45f),
                AZ::Vector3(3.8f, 2.5f, 1.0f),
                roofRotation,
            },
            {
                "Hearthwatch_ShingleRoof_Patch",
                StonewakeSurfacePlaneModelAssetPath,
                "content/art/materials/mat_hearthwatch_wood_shingles.azmaterial",
                AZ::Vector3(222.0f, 121.0f, 2.20f),
                AZ::Vector3(3.1f, 2.3f, 1.0f),
                roofRotation,
            },
        };

        unsigned int spawnedCount = 0;
        for (const StonewakeMaterialSurfaceDefinition& surfaceDefinition : surfaceDefinitions)
        {
            const AZ::Data::AssetId modelAssetId = resolveAssetIdByPath(surfaceDefinition.m_modelAssetPath);
            const AZ::Data::AssetId materialAssetId = resolveAssetIdByPath(surfaceDefinition.m_materialAssetPath);
            if (!modelAssetId.IsValid() || !materialAssetId.IsValid())
            {
                if (!m_loggedStonewakeMaterialSurfaceMissing)
                {
                    m_loggedStonewakeMaterialSurfaceMissing = true;
                    AZ_Warning(
                        "amandacore",
                        false,
                        "client.stonewake_textured_surface_missing modelResolved=%s materialResolved=%s model=%s material=%s",
                        modelAssetId.IsValid() ? "true" : "false",
                        materialAssetId.IsValid() ? "true" : "false",
                        surfaceDefinition.m_modelAssetPath,
                        surfaceDefinition.m_materialAssetPath);
                }
                continue;
            }

            AZ::Entity* entity = nullptr;
            AzFramework::GameEntityContextRequestBus::BroadcastResult(
                entity,
                &AzFramework::GameEntityContextRequestBus::Events::CreateGameEntity,
                surfaceDefinition.m_name);
            if (!entity)
            {
                AZ_Warning("amandacore", false, "client.stonewake_textured_surface_entity_create_failed name=%s", surfaceDefinition.m_name);
                continue;
            }

            auto* transformComponent = entity->CreateComponent<AzFramework::TransformComponent>();
            auto* nonUniformScaleComponent = entity->CreateComponent<AzFramework::NonUniformScaleComponent>();
            AZ::Component* meshComponent = entity->CreateComponent(AZ::Render::MeshComponentTypeId);
            AZ::Component* materialComponent = entity->CreateComponent(AZ::Render::MaterialComponentTypeId);
            if (!transformComponent || !nonUniformScaleComponent || !meshComponent || !materialComponent)
            {
                AZ_Warning(
                    "amandacore",
                    false,
                    "client.stonewake_textured_surface_component_missing name=%s transform=%s scale=%s mesh=%s material=%s",
                    surfaceDefinition.m_name,
                    transformComponent ? "true" : "false",
                    nonUniformScaleComponent ? "true" : "false",
                    meshComponent ? "true" : "false",
                    materialComponent ? "true" : "false");
                delete entity;
                continue;
            }

            nonUniformScaleComponent->SetScale(surfaceDefinition.m_nonUniformScale);
            entity->Init();
            AzFramework::GameEntityContextRequestBus::Broadcast(
                &AzFramework::GameEntityContextRequestBus::Events::AddGameEntity,
                entity);
            AzFramework::GameEntityContextRequestBus::Broadcast(
                &AzFramework::GameEntityContextRequestBus::Events::ActivateGameEntity,
                entity->GetId());

            const AZ::Transform worldTransform = AZ::Transform::CreateFromQuaternionAndTranslation(
                surfaceDefinition.m_rotation,
                surfaceDefinition.m_position);
            AZ::TransformBus::Event(
                entity->GetId(),
                &AZ::TransformBus::Events::SetWorldTM,
                worldTransform);
            AZ::NonUniformScaleRequestBus::Event(
                entity->GetId(),
                &AZ::NonUniformScaleRequestBus::Events::SetScale,
                surfaceDefinition.m_nonUniformScale);
            AZ::Render::MeshComponentRequestBus::Event(
                entity->GetId(),
                &AZ::Render::MeshComponentRequestBus::Events::SetModelAssetPath,
                AZStd::string(surfaceDefinition.m_modelAssetPath));
            AZ::Render::MaterialComponentRequestBus::Event(
                entity->GetId(),
                &AZ::Render::MaterialComponentRequestBus::Events::SetMaterialAssetIdOnDefaultSlot,
                materialAssetId);

            m_stonewakeMaterialSurfaceEntityIds.push_back(entity->GetId());
            ++spawnedCount;
            AZ_Printf(
                "amandacore",
                "client.stonewake_textured_surface_spawned name=%s model=%s material=%s",
                surfaceDefinition.m_name,
                surfaceDefinition.m_modelAssetPath,
                surfaceDefinition.m_materialAssetPath);
        }

        if (spawnedCount > 0)
        {
            m_stonewakeMaterialSurfacesSpawned = true;
            AZ_Printf(
                "amandacore",
                "client.stonewake_textured_surface_coverage count=%u source=curated_png_materials scope=stonewake_0_2",
                spawnedCount);
        }
    }

    void LocalPlayerControllerComponent::DestroyStonewakeMaterialSurfaces()
    {
        for (const AZ::EntityId& entityId : m_stonewakeMaterialSurfaceEntityIds)
        {
            if (entityId.IsValid())
            {
                AzFramework::GameEntityContextRequestBus::Broadcast(
                    &AzFramework::GameEntityContextRequestBus::Events::DestroyGameEntity,
                    entityId);
            }
        }

        m_stonewakeMaterialSurfaceEntityIds.clear();
        m_stonewakeMaterialSurfacesSpawned = false;
        m_loggedStonewakeMaterialSurfaceMissing = false;
    }

    void LocalPlayerControllerComponent::DrawValidationArena()
    {
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

        if (!m_loggedValidationFloor)
        {
            m_loggedValidationFloor = true;
            AZ_Printf(
                "amandacore",
                "client.world_material_coverage_visible center=(%.1f, %.1f, %.1f) extent=(%.1f, %.1f) spawn=(%.1f, %.1f, %.1f) material=stonewake_grass_lush overlays=stonewake_0_2 interiors=4",
                ValidationFloorExtent * 0.5f,
                ValidationFloorExtentY * 0.5f,
                ValidationFloorZ,
                ValidationFloorExtent,
                ValidationFloorExtentY,
                ValidationSpawnX,
                ValidationSpawnY,
                ValidationFloorZ);
        }

        const AZ::Color commandColor(0.28f, 0.74f, 0.78f, 1.0f);
        const AZ::Color pathColor(0.78f, 0.60f, 0.28f, 1.0f);
        const AZ::Color obstacleColor(0.38f, 0.39f, 0.43f, 1.0f);
        const AZ::Color encounterColor(0.90f, 0.42f, 0.22f, 1.0f);
        const AZ::Color groundTileLight(0.31f, 0.43f, 0.25f, 1.0f);
        const AZ::Color groundTileDark(0.22f, 0.32f, 0.21f, 1.0f);
        const AZ::Color hearthwatchYardColor(0.42f, 0.39f, 0.31f, 1.0f);
        const AZ::Color hearthwatchCobbleColor(0.52f, 0.49f, 0.39f, 1.0f);
        const AZ::Color mossColor(0.20f, 0.41f, 0.23f, 1.0f);
        const AZ::Color rockyGroundColor(0.34f, 0.36f, 0.34f, 1.0f);
        const AZ::Color ridgeColor(0.30f, 0.31f, 0.34f, 1.0f);
        const AZ::Color horizonColor(0.23f, 0.34f, 0.44f, 1.0f);
        const AZ::Color roadEdgeColor(0.30f, 0.24f, 0.16f, 1.0f);
        const AZ::Color roadColor(0.66f, 0.49f, 0.27f, 1.0f);
        const AZ::Color roadPebbleColor(0.78f, 0.70f, 0.53f, 1.0f);
        const AZ::Color fieldColor(0.50f, 0.43f, 0.22f, 1.0f);
        const AZ::Color cropColor(0.74f, 0.62f, 0.25f, 1.0f);
        const AZ::Color plasterColor(0.78f, 0.70f, 0.58f, 1.0f);
        const AZ::Color woodColor(0.42f, 0.28f, 0.16f, 1.0f);
        const AZ::Color roofColor(0.50f, 0.40f, 0.20f, 1.0f);
        const AZ::Color cutStoneColor(0.42f, 0.43f, 0.40f, 1.0f);
        const AZ::Color trunkColor(0.33f, 0.22f, 0.13f, 1.0f);
        const AZ::Color canopyColor(0.16f, 0.43f, 0.23f, 1.0f);
        const AZ::Color trainingRingColor(0.70f, 0.55f, 0.28f, 1.0f);
        const AZ::Color waterColor(0.12f, 0.39f, 0.57f, 1.0f);
        const AZ::Color wetShoreColor(0.45f, 0.39f, 0.30f, 1.0f);
        const AZ::Color runeColor(0.28f, 0.74f, 0.92f, 1.0f);

        struct SurfacePatch
        {
            AZ::Vector3 m_center;
            AZ::Vector3 m_halfExtents;
            AZ::Color m_color;
        };
        const SurfacePatch surfacePatches[] = {
            {AZ::Vector3(232.0f, 130.0f, 0.035f), AZ::Vector3(42.0f, 25.0f, 0.035f), hearthwatchYardColor},
            {AZ::Vector3(197.0f, 74.0f, 0.030f), AZ::Vector3(34.0f, 22.0f, 0.030f), groundTileLight},
            {AZ::Vector3(313.0f, 81.0f, 0.030f), AZ::Vector3(36.0f, 18.0f, 0.030f), mossColor},
            {AZ::Vector3(361.0f, 157.0f, 0.030f), AZ::Vector3(44.0f, 24.0f, 0.030f), rockyGroundColor},
            {AZ::Vector3(380.0f, 231.0f, 0.030f), AZ::Vector3(38.0f, 22.0f, 0.030f), groundTileDark},
            {AZ::Vector3(358.0f, 39.0f, 0.030f), AZ::Vector3(34.0f, 18.0f, 0.030f), mossColor},
        };
        for (const SurfacePatch& patch : surfacePatches)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(patch.m_center, patch.m_halfExtents),
                patch.m_color,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        for (int cobbleIndex = 0; cobbleIndex < 30; ++cobbleIndex)
        {
            const float centerX = 200.0f + (static_cast<float>(cobbleIndex % 10) * 7.2f);
            const float centerY = 112.0f + (static_cast<float>(cobbleIndex / 10) * 10.0f);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(centerX, centerY, 0.090f),
                    AZ::Vector3(1.35f, 0.16f, 0.025f)),
                hearthwatchCobbleColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        for (int grassBreakIndex = 0; grassBreakIndex < 26; ++grassBreakIndex)
        {
            const float centerX = 140.0f + (static_cast<float>((grassBreakIndex * 37) % 270));
            const float centerY = 42.0f + (static_cast<float>((grassBreakIndex * 29) % 205));
            const AZ::Color breakColor = (grassBreakIndex % 3) == 0 ? mossColor : groundTileLight;
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(centerX, centerY, 0.075f),
                    AZ::Vector3(2.4f, 0.10f, 0.025f)),
                breakColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 streamCenters[] = {
            AZ::Vector3(286.0f, 105.0f, 0.04f),
            AZ::Vector3(313.0f, 81.0f, 0.04f),
            AZ::Vector3(336.0f, 72.0f, 0.04f),
            AZ::Vector3(366.0f, 64.0f, 0.04f),
        };
        const AZ::Vector3 streamExtents[] = {
            AZ::Vector3(30.0f, 2.8f, 0.04f),
            AZ::Vector3(34.0f, 3.1f, 0.04f),
            AZ::Vector3(38.0f, 3.3f, 0.04f),
            AZ::Vector3(30.0f, 3.5f, 0.04f),
        };
        for (size_t streamIndex = 0; streamIndex < AZ_ARRAY_SIZE(streamCenters); ++streamIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(streamCenters[streamIndex], streamExtents[streamIndex]),
                wetShoreColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    streamCenters[streamIndex] + AZ::Vector3(0.0f, 0.0f, 0.035f),
                    streamExtents[streamIndex] - AZ::Vector3(0.0f, 1.2f, 0.0f)),
                waterColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 roadCenters[] = {
            AZ::Vector3(232.0f, 130.0f, 0.10f),
            AZ::Vector3(197.0f, 74.0f, 0.10f),
            AZ::Vector3(260.0f, 70.0f, 0.10f),
            AZ::Vector3(313.0f, 81.0f, 0.10f),
            AZ::Vector3(375.0f, 77.0f, 0.10f),
            AZ::Vector3(361.0f, 157.0f, 0.10f),
            AZ::Vector3(330.0f, 197.0f, 0.10f),
            AZ::Vector3(380.0f, 231.0f, 0.10f),
        };
        const AZ::Vector3 roadExtents[] = {
            AZ::Vector3(18.0f, 3.2f, 0.06f),
            AZ::Vector3(26.0f, 3.3f, 0.06f),
            AZ::Vector3(32.0f, 3.5f, 0.06f),
            AZ::Vector3(36.0f, 3.7f, 0.06f),
            AZ::Vector3(42.0f, 3.9f, 0.06f),
            AZ::Vector3(40.0f, 4.0f, 0.06f),
            AZ::Vector3(36.0f, 4.1f, 0.06f),
            AZ::Vector3(22.0f, 4.2f, 0.06f),
        };
        for (size_t roadIndex = 0; roadIndex < AZ_ARRAY_SIZE(roadCenters); ++roadIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    roadCenters[roadIndex] - AZ::Vector3(0.0f, 0.0f, 0.025f),
                    roadExtents[roadIndex] + AZ::Vector3(0.0f, 0.85f, 0.02f)),
                roadEdgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(roadCenters[roadIndex], roadExtents[roadIndex]),
                roadColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    roadCenters[roadIndex] + AZ::Vector3(0.0f, 1.55f, 0.070f),
                    AZ::Vector3(roadExtents[roadIndex].GetX() * 0.82f, 0.08f, 0.025f)),
                roadEdgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    roadCenters[roadIndex] + AZ::Vector3(0.0f, -1.55f, 0.070f),
                    AZ::Vector3(roadExtents[roadIndex].GetX() * 0.82f, 0.08f, 0.025f)),
                roadEdgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

            for (int pebbleIndex = 0; pebbleIndex < 7; ++pebbleIndex)
            {
                const float offsetX = (static_cast<float>(pebbleIndex) - 3.0f) * (roadExtents[roadIndex].GetX() * 0.24f);
                const float offsetY = (pebbleIndex % 2 == 0 ? 1.20f : -1.20f);
                auxGeom->DrawSphere(
                    roadCenters[roadIndex] + AZ::Vector3(offsetX, offsetY, 0.08f),
                    0.22f,
                    roadPebbleColor);
            }
        }

        const AZ::Vector3 fieldCenters[] = {
            AZ::Vector3(175.0f, 62.0f, 0.02f),
            AZ::Vector3(188.0f, 68.0f, 0.02f),
            AZ::Vector3(202.0f, 74.0f, 0.02f),
            AZ::Vector3(216.0f, 80.0f, 0.02f),
            AZ::Vector3(198.0f, 92.0f, 0.02f),
        };
        for (const AZ::Vector3& fieldCenter : fieldCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(fieldCenter, AZ::Vector3(9.0f, 0.28f, 0.04f)),
                fieldColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

            for (int cropIndex = 0; cropIndex < 6; ++cropIndex)
            {
                auxGeom->DrawAabb(
                    AZ::Aabb::CreateCenterHalfExtents(
                        fieldCenter + AZ::Vector3((static_cast<float>(cropIndex) - 2.5f) * 2.6f, 0.0f, 0.22f),
                        AZ::Vector3(0.10f, 0.16f, 0.22f)),
                    cropColor,
                    AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            }
        }

        auto drawOpenRoom = [&auxGeom, &woodColor, &roofColor](
                                const AZ::Vector3& center,
                                const AZ::Vector3& halfExtents,
                                const AZ::Color& wallColor,
                                const AZ::Color& floorColor)
        {
            const float floorZ = 0.105f;
            const float wallThickness = 0.16f;
            const float wallHeight = 1.05f;
            const float wallCenterZ = floorZ + (wallHeight * 0.5f);
            const float doorwayHalfWidth = AZ::GetMin(halfExtents.GetX() * 0.34f, 1.15f);
            const float frontSegmentHalfX = AZ::GetMax((halfExtents.GetX() - doorwayHalfWidth) * 0.5f, 0.15f);

            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX(), center.GetY(), floorZ),
                    AZ::Vector3(halfExtents.GetX(), halfExtents.GetY(), 0.055f)),
                floorColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() - halfExtents.GetX(), center.GetY(), wallCenterZ),
                    AZ::Vector3(wallThickness, halfExtents.GetY(), wallHeight * 0.5f)),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() + halfExtents.GetX(), center.GetY(), wallCenterZ),
                    AZ::Vector3(wallThickness, halfExtents.GetY(), wallHeight * 0.5f)),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX(), center.GetY() + halfExtents.GetY(), wallCenterZ),
                    AZ::Vector3(halfExtents.GetX() + wallThickness, wallThickness, wallHeight * 0.5f)),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() - doorwayHalfWidth - frontSegmentHalfX, center.GetY() - halfExtents.GetY(), wallCenterZ),
                    AZ::Vector3(frontSegmentHalfX, wallThickness, wallHeight * 0.5f)),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() + doorwayHalfWidth + frontSegmentHalfX, center.GetY() - halfExtents.GetY(), wallCenterZ),
                    AZ::Vector3(frontSegmentHalfX, wallThickness, wallHeight * 0.5f)),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX(), center.GetY() + (halfExtents.GetY() * 0.40f), wallCenterZ + (wallHeight * 0.58f)),
                    AZ::Vector3(halfExtents.GetX() + 0.28f, halfExtents.GetY() * 0.35f, 0.12f)),
                roofColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);

            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX(), center.GetY() + 0.16f, floorZ + 0.24f),
                    AZ::Vector3(0.75f, 0.22f, 0.12f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() - (halfExtents.GetX() * 0.42f), center.GetY() - 0.34f, floorZ + 0.18f),
                    AZ::Vector3(0.18f, 0.72f, 0.10f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(center.GetX() + (halfExtents.GetX() * 0.42f), center.GetY() - 0.34f, floorZ + 0.18f),
                    AZ::Vector3(0.18f, 0.72f, 0.10f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        };

        const AZ::Vector3 buildingCenters[] = {
            AZ::Vector3(218.0f, 120.0f, 1.0f),
            AZ::Vector3(232.0f, 118.0f, 0.9f),
            AZ::Vector3(220.0f, 142.0f, 0.85f),
            AZ::Vector3(244.0f, 142.0f, 0.85f),
            AZ::Vector3(268.0f, 145.0f, 0.65f),
            AZ::Vector3(197.0f, 74.0f, 1.1f),
            AZ::Vector3(313.0f, 81.0f, 1.2f),
            AZ::Vector3(380.0f, 231.0f, 2.6f),
            AZ::Vector3(358.0f, 39.0f, 1.0f),
            AZ::Vector3(375.0f, 77.0f, 1.7f),
        };
        const AZ::Vector3 buildingExtents[] = {
            AZ::Vector3(2.8f, 2.0f, 1.0f),
            AZ::Vector3(2.0f, 1.8f, 0.9f),
            AZ::Vector3(2.4f, 1.6f, 0.85f),
            AZ::Vector3(3.0f, 1.8f, 0.85f),
            AZ::Vector3(7.0f, 0.35f, 0.65f),
            AZ::Vector3(6.0f, 0.9f, 1.1f),
            AZ::Vector3(2.4f, 2.4f, 1.2f),
            AZ::Vector3(4.4f, 4.4f, 2.6f),
            AZ::Vector3(3.8f, 1.6f, 1.0f),
            AZ::Vector3(6.5f, 1.2f, 1.7f),
        };
        for (size_t buildingIndex = 0; buildingIndex < AZ_ARRAY_SIZE(buildingCenters); ++buildingIndex)
        {
            const AZ::Color wallColor = buildingIndex == 6 || buildingIndex == 7
                ? cutStoneColor
                : (buildingIndex == 4 || buildingIndex == 8 || buildingIndex == 9 ? woodColor : plasterColor);
            if (buildingIndex < 4)
            {
                drawOpenRoom(
                    buildingCenters[buildingIndex],
                    AZ::Vector3(
                        buildingExtents[buildingIndex].GetX() + 0.35f,
                        buildingExtents[buildingIndex].GetY() + 0.30f,
                        0.0f),
                    wallColor,
                    hearthwatchCobbleColor);
                continue;
            }

            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(buildingCenters[buildingIndex], buildingExtents[buildingIndex]),
                wallColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    buildingCenters[buildingIndex] + AZ::Vector3(0.0f, 0.0f, buildingExtents[buildingIndex].GetZ() + 0.18f),
                    AZ::Vector3(buildingExtents[buildingIndex].GetX() + 0.35f, buildingExtents[buildingIndex].GetY() + 0.35f, 0.18f)),
                buildingIndex == 6 || buildingIndex == 7 ? ridgeColor : roofColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 horizonCenters[] = {
            AZ::Vector3(ValidationFloorExtent * 0.5f, -5.0f, 3.0f),
            AZ::Vector3(ValidationFloorExtent * 0.5f, ValidationFloorExtentY + 5.0f, 3.0f),
            AZ::Vector3(-5.0f, ValidationFloorExtentY * 0.5f, 3.0f),
            AZ::Vector3(ValidationFloorExtent + 5.0f, ValidationFloorExtentY * 0.5f, 3.0f),
        };
        const AZ::Vector3 horizonHalfExtents[] = {
            AZ::Vector3(ValidationFloorExtent * 0.5f, 0.5f, 3.0f),
            AZ::Vector3(ValidationFloorExtent * 0.5f, 0.5f, 3.0f),
            AZ::Vector3(0.5f, ValidationFloorExtentY * 0.5f, 3.0f),
            AZ::Vector3(0.5f, ValidationFloorExtentY * 0.5f, 3.0f),
        };
        for (size_t horizonIndex = 0; horizonIndex < AZ_ARRAY_SIZE(horizonCenters); ++horizonIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(horizonCenters[horizonIndex], horizonHalfExtents[horizonIndex]),
                horizonColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        const AZ::Vector3 ridgeCenters[] = {
            AZ::Vector3(120.0f, 48.0f, 1.2f),
            AZ::Vector3(150.0f, 70.0f, 1.0f),
            AZ::Vector3(176.0f, 54.0f, 1.3f),
            AZ::Vector3(220.0f, 170.0f, 1.1f),
            AZ::Vector3(250.0f, 188.0f, 1.2f),
            AZ::Vector3(292.0f, 160.0f, 1.4f),
            AZ::Vector3(338.0f, 150.0f, 1.1f),
            AZ::Vector3(361.0f, 157.0f, 1.4f),
            AZ::Vector3(388.0f, 166.0f, 1.2f),
            AZ::Vector3(370.0f, 228.0f, 1.3f),
            AZ::Vector3(405.0f, 235.0f, 1.1f),
            AZ::Vector3(420.0f, 94.0f, 1.3f),
        };
        for (const AZ::Vector3& ridgeCenter : ridgeCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    ridgeCenter,
                    AZ::Vector3(1.8f, 1.4f, 1.1f)),
                ridgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        const AZ::Vector3 treeCenters[] = {
            AZ::Vector3(180.0f, 118.0f, 0.0f),
            AZ::Vector3(198.0f, 112.0f, 0.0f),
            AZ::Vector3(250.0f, 155.0f, 0.0f),
            AZ::Vector3(286.0f, 134.0f, 0.0f),
            AZ::Vector3(306.0f, 96.0f, 0.0f),
            AZ::Vector3(332.0f, 112.0f, 0.0f),
            AZ::Vector3(348.0f, 196.0f, 0.0f),
            AZ::Vector3(365.0f, 162.0f, 0.0f),
            AZ::Vector3(396.0f, 244.0f, 0.0f),
        };
        for (const AZ::Vector3& treeBase : treeCenters)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    treeBase + AZ::Vector3(0.0f, 0.0f, 0.65f),
                    AZ::Vector3(0.18f, 0.18f, 0.65f)),
                trunkColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(0.0f, 0.0f, 1.75f), 0.82f, canopyColor);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(0.42f, 0.12f, 1.55f), 0.55f, canopyColor);
            auxGeom->DrawSphere(treeBase + AZ::Vector3(-0.38f, -0.16f, 1.55f), 0.55f, canopyColor);
        }

        for (int segmentIndex = 0; segmentIndex < 18; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 18.0f) * static_cast<float>(segmentIndex);
            auxGeom->DrawSphere(
                AZ::Vector3(
                    268.0f + (AZStd::cos(angleRadians) * 9.0f),
                    145.0f + (AZStd::sin(angleRadians) * 7.0f),
                    ValidationMarkerZ),
                0.14f,
                trainingRingColor);
        }

        for (int segmentIndex = 0; segmentIndex < 10; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 10.0f) * static_cast<float>(segmentIndex);
            const AZ::Vector3 ringOffset(
                AZStd::cos(angleRadians) * 2.5f,
                AZStd::sin(angleRadians) * 2.5f,
                ValidationMarkerZ);
            auxGeom->DrawSphere(
                AZ::Vector3(ValidationSpawnX, ValidationSpawnY, 0.0f) + ringOffset,
                ValidationSpawnMarkerRadius,
                commandColor);
        }

        const AZ::Vector3 commandPostColumns[] = {
            AZ::Vector3(228.0f, 126.0f, ValidationMarkerZ),
            AZ::Vector3(236.0f, 126.0f, ValidationMarkerZ),
            AZ::Vector3(228.0f, 134.0f, ValidationMarkerZ),
            AZ::Vector3(236.0f, 134.0f, ValidationMarkerZ)};
        for (const AZ::Vector3& columnBase : commandPostColumns)
        {
            for (int segmentIndex = 0; segmentIndex < 4; ++segmentIndex)
            {
                auxGeom->DrawSphere(
                    columnBase + AZ::Vector3(0.0f, 0.0f, 0.28f + (segmentIndex * 0.24f)),
                    0.12f,
                    commandColor);
            }
        }

        const AZ::Vector3 trailMarkers[] = {
            AZ::Vector3(232.0f, 130.0f, ValidationMarkerZ),
            AZ::Vector3(197.0f, 74.0f, ValidationMarkerZ),
            AZ::Vector3(260.0f, 70.0f, ValidationMarkerZ),
            AZ::Vector3(313.0f, 81.0f, ValidationMarkerZ),
            AZ::Vector3(375.0f, 77.0f, ValidationMarkerZ),
            AZ::Vector3(358.0f, 39.0f, ValidationMarkerZ),
            AZ::Vector3(361.0f, 157.0f, ValidationMarkerZ),
            AZ::Vector3(380.0f, 231.0f, ValidationMarkerZ)};
        for (const AZ::Vector3& marker : trailMarkers)
        {
            auxGeom->DrawSphere(marker + AZ::Vector3(0.0f, 0.0f, 0.08f), 0.26f, pathColor);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    marker + AZ::Vector3(0.0f, 0.0f, 0.34f),
                    AZ::Vector3(0.07f, 0.07f, 0.34f)),
                roadPebbleColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        struct LandmarkMarker
        {
            AZ::Vector3 m_position;
            AZ::Color m_color;
            float m_height;
        };
        const LandmarkMarker landmarkMarkers[] = {
            {AZ::Vector3(232.0f, 130.0f, 0.0f), commandColor, 2.0f},
            {AZ::Vector3(197.0f, 74.0f, 0.0f), cropColor, 1.8f},
            {AZ::Vector3(313.0f, 81.0f, 0.0f), runeColor, 2.0f},
            {AZ::Vector3(361.0f, 157.0f, 0.0f), cutStoneColor, 2.2f},
            {AZ::Vector3(380.0f, 231.0f, 0.0f), encounterColor, 2.1f},
            {AZ::Vector3(375.0f, 77.0f, 0.0f), waterColor, 1.9f},
        };
        for (const LandmarkMarker& landmark : landmarkMarkers)
        {
            const AZ::Vector3 base = landmark.m_position;
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    base + AZ::Vector3(0.0f, 0.0f, 0.08f),
                    AZ::Vector3(1.25f, 1.25f, 0.08f)),
                roadEdgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    base + AZ::Vector3(0.0f, 0.0f, landmark.m_height * 0.5f),
                    AZ::Vector3(0.10f, 0.10f, landmark.m_height * 0.5f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    base + AZ::Vector3(0.0f, 0.0f, landmark.m_height + 0.18f),
                    AZ::Vector3(0.95f, 0.10f, 0.32f)),
                landmark.m_color,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
            auxGeom->DrawSphere(
                base + AZ::Vector3(0.0f, 0.0f, landmark.m_height + 0.62f),
                0.22f,
                landmark.m_color);
        }

        const AZ::Vector3 boulderCluster[] = {
            AZ::Vector3(355.0f, 152.0f, 0.30f),
            AZ::Vector3(360.0f, 158.0f, 0.55f),
            AZ::Vector3(367.0f, 164.0f, 0.42f),
            AZ::Vector3(352.0f, 166.0f, 0.38f)};
        for (const AZ::Vector3& boulder : boulderCluster)
        {
            auxGeom->DrawSphere(boulder, 0.75f, obstacleColor);
        }

        const AZ::Vector3 standingStones[] = {
            AZ::Vector3(286.0f, 154.0f, 1.0f),
            AZ::Vector3(291.0f, 153.0f, 1.4f),
            AZ::Vector3(295.0f, 157.0f, 1.0f),
            AZ::Vector3(293.0f, 163.0f, 1.2f),
            AZ::Vector3(287.0f, 162.0f, 0.9f),
        };
        for (const AZ::Vector3& stone : standingStones)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(stone, AZ::Vector3(0.42f, 0.28f, stone.GetZ())),
                cutStoneColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
            auxGeom->DrawSphere(stone + AZ::Vector3(0.0f, 0.0f, stone.GetZ() + 0.22f), 0.16f, runeColor);
        }

        const AZ::Vector3 quarryBlocks[] = {
            AZ::Vector3(350.0f, 152.0f, 0.45f),
            AZ::Vector3(358.0f, 157.0f, 0.55f),
            AZ::Vector3(368.0f, 162.0f, 0.50f),
            AZ::Vector3(376.0f, 167.0f, 0.38f),
        };
        for (const AZ::Vector3& block : quarryBlocks)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(block, AZ::Vector3(2.6f, 1.1f, block.GetZ())),
                ridgeColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        const AZ::Vector3 dockPlanks[] = {
            AZ::Vector3(362.0f, 45.0f, 0.18f),
            AZ::Vector3(370.0f, 46.0f, 0.18f),
            AZ::Vector3(378.0f, 47.0f, 0.18f),
            AZ::Vector3(386.0f, 48.0f, 0.18f),
        };
        for (const AZ::Vector3& plank : dockPlanks)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(plank, AZ::Vector3(3.6f, 0.42f, 0.08f)),
                woodColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Solid);
        }

        for (int tierIndex = 0; tierIndex < 4; ++tierIndex)
        {
            auxGeom->DrawAabb(
                AZ::Aabb::CreateCenterHalfExtents(
                    AZ::Vector3(380.0f, 231.0f, 0.45f + (tierIndex * 0.62f)),
                    AZ::Vector3(1.15f - (tierIndex * 0.10f), 1.15f - (tierIndex * 0.10f), 0.30f)),
                cutStoneColor,
                AZ::RPI::AuxGeomDraw::DrawStyle::Shaded);
        }

        for (int segmentIndex = 0; segmentIndex < 12; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / 12.0f) * static_cast<float>(segmentIndex);
            auxGeom->DrawSphere(
                AZ::Vector3(
                    EncounterAnchorX + (AZStd::cos(angleRadians) * 6.1f),
                    EncounterAnchorY + (AZStd::sin(angleRadians) * 4.8f),
                    ValidationMarkerZ),
                0.15f,
                encounterColor);
        }
    }

    void LocalPlayerControllerComponent::DrawLocalPlayerProxy()
    {
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

        const AZ::Vector3 position = m_cachedFinalPoseValid ? m_cachedFinalPresentationPosition : GetPresentationPosition();

        const float avatarVisiblePoseDistance = m_loggedAvatarVisiblePose
            ? (position - m_lastLoggedAvatarVisiblePose).GetLength()
            : PoseLogDistanceThreshold;
        if (!m_loggedAvatarVisiblePose || avatarVisiblePoseDistance >= PoseLogDistanceThreshold)
        {
            m_loggedAvatarVisiblePose = true;
            m_lastLoggedAvatarVisiblePose = position;
            AZ_Printf(
                "amandacore",
                "client.avatar_visible_pose pos=(%.3f, %.3f, %.3f) yaw=%.3f",
                position.GetX(),
                position.GetY(),
                position.GetZ(),
                m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians);
        }

        const float facingRadians = m_cachedFinalPoseValid ? m_cachedFinalAvatarFacingRadians : m_avatarFacingRadians;
        const AZ::Vector3 forward = ForwardFromYaw(facingRadians);
        const AZ::Vector3 right = RightFromYaw(facingRadians);
        const float strideSwing = AZStd::sin(m_avatarStridePhase) * AvatarStrideReach * m_avatarStrideBlend;
        const float oppositeStrideSwing = -strideSwing;
        const float armSwing = strideSwing * 0.8f;
        const float leanForward = 0.05f * m_avatarStrideBlend;

        const AZ::Color clothColor(0.19f, 0.34f, 0.59f, 1.0f);
        const AZ::Color trimColor(0.85f, 0.71f, 0.31f, 1.0f);
        const AZ::Color skinColor(0.92f, 0.90f, 0.84f, 1.0f);
        const AZ::Color shadowColor(0.10f, 0.16f, 0.22f, 1.0f);
        const AZ::Color ringColor(0.88f, 0.91f, 0.94f, 1.0f);

        const AZ::Vector3 pelvis = position + AZ::Vector3(0.0f, 0.0f, AvatarHipHeight);
        const AZ::Vector3 waist = position + (forward * leanForward) + AZ::Vector3(0.0f, 0.0f, AvatarWaistHeight);
        const AZ::Vector3 chest = position + (forward * (0.08f + leanForward)) + AZ::Vector3(0.0f, 0.0f, AvatarChestHeight);
        const AZ::Vector3 head = chest + (forward * 0.08f) + AZ::Vector3(0.0f, 0.0f, AvatarHeadHeight - AvatarChestHeight);

        auxGeom->DrawSphere(pelvis, AvatarPelvisRadius, clothColor);
        auxGeom->DrawSphere(waist, AvatarTorsoRadius, clothColor);
        auxGeom->DrawSphere(chest, AvatarChestRadius, clothColor);
        auxGeom->DrawSphere(chest - (forward * 0.18f), 0.18f, shadowColor);
        auxGeom->DrawSphere(head, AvatarHeadRadius, skinColor);
        auxGeom->DrawSphere(head + (forward * 0.14f), 0.05f, trimColor);

        const AZ::Vector3 leftHip = pelvis + (right * AvatarHipSpacing);
        const AZ::Vector3 rightHip = pelvis - (right * AvatarHipSpacing);
        const AZ::Vector3 leftKnee = position + (right * AvatarHipSpacing) + (forward * (-strideSwing * 0.20f)) + AZ::Vector3(0.0f, 0.0f, AvatarKneeHeight);
        const AZ::Vector3 rightKnee = position - (right * AvatarHipSpacing) + (forward * (-oppositeStrideSwing * 0.20f)) + AZ::Vector3(0.0f, 0.0f, AvatarKneeHeight);
        const AZ::Vector3 leftAnkle = position + (right * AvatarFootSpacing) + (forward * (strideSwing * 0.35f)) + AZ::Vector3(0.0f, 0.0f, AvatarAnkleHeight);
        const AZ::Vector3 rightAnkle = position - (right * AvatarFootSpacing) + (forward * (oppositeStrideSwing * 0.35f)) + AZ::Vector3(0.0f, 0.0f, AvatarAnkleHeight);
        const AZ::Vector3 leftFoot = position + (right * AvatarFootSpacing) + (forward * (strideSwing * 0.45f)) + AZ::Vector3(0.0f, 0.0f, AvatarFootHeight);
        const AZ::Vector3 rightFoot = position - (right * AvatarFootSpacing) + (forward * (oppositeStrideSwing * 0.45f)) + AZ::Vector3(0.0f, 0.0f, AvatarFootHeight);

        auxGeom->DrawSphere(leftHip, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(rightHip, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(leftKnee, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(rightKnee, AvatarLimbRadius, clothColor);
        auxGeom->DrawSphere(leftAnkle, AvatarLimbRadius * 0.92f, trimColor);
        auxGeom->DrawSphere(rightAnkle, AvatarLimbRadius * 0.92f, trimColor);
        auxGeom->DrawSphere(leftFoot, AvatarLimbRadius * 0.85f, trimColor);
        auxGeom->DrawSphere(rightFoot, AvatarLimbRadius * 0.85f, trimColor);

        const AZ::Vector3 leftShoulder = chest + (right * AvatarShoulderOffset) + AZ::Vector3(0.0f, 0.0f, AvatarShoulderHeight - AvatarChestHeight);
        const AZ::Vector3 rightShoulder = chest - (right * AvatarShoulderOffset) + AZ::Vector3(0.0f, 0.0f, AvatarShoulderHeight - AvatarChestHeight);
        const AZ::Vector3 leftElbow = chest + (right * (AvatarShoulderOffset + 0.10f)) + (forward * (-armSwing * 0.24f)) + AZ::Vector3(0.0f, 0.0f, 1.16f);
        const AZ::Vector3 rightElbow = chest - (right * (AvatarShoulderOffset + 0.10f)) + (forward * (armSwing * 0.24f)) + AZ::Vector3(0.0f, 0.0f, 1.16f);
        const AZ::Vector3 leftHand = chest + (right * (AvatarShoulderOffset + 0.14f)) + (forward * (-armSwing * 0.34f)) + AZ::Vector3(0.0f, 0.0f, 0.84f);
        const AZ::Vector3 rightHand = chest - (right * (AvatarShoulderOffset + 0.14f)) + (forward * (armSwing * 0.34f)) + AZ::Vector3(0.0f, 0.0f, 0.84f);

        auxGeom->DrawSphere(leftShoulder, AvatarArmRadius, trimColor);
        auxGeom->DrawSphere(rightShoulder, AvatarArmRadius, trimColor);
        auxGeom->DrawSphere(leftElbow, AvatarArmRadius, clothColor);
        auxGeom->DrawSphere(rightElbow, AvatarArmRadius, clothColor);
        auxGeom->DrawSphere(leftHand, AvatarArmRadius * 0.85f, skinColor);
        auxGeom->DrawSphere(rightHand, AvatarArmRadius * 0.85f, skinColor);

        for (int segmentIndex = 0; segmentIndex < AvatarSelectionRingSegments; ++segmentIndex)
        {
            const float angleRadians = (AZ::Constants::TwoPi / static_cast<float>(AvatarSelectionRingSegments)) *
                static_cast<float>(segmentIndex);
            const AZ::Vector3 ringOffset(
                AZStd::cos(angleRadians) * AvatarSelectionRingRadius,
                AZStd::sin(angleRadians) * AvatarSelectionRingRadius,
                0.06f);
            auxGeom->DrawSphere(position + ringOffset, AvatarSelectionRingSphereRadius, ringColor);
        }
    }
} // namespace MovementPhysics

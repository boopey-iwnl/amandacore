#pragma once

#include <AzCore/Component/Component.h>
#include <AzCore/Component/TickBus.h>
#include <AzCore/Math/Vector2.h>
#include <AzCore/Math/Vector3.h>
#include <AzCore/std/string/string.h>
#include <AzFramework/Input/Events/InputChannelEventListener.h>
#include <AzFramework/Viewport/ViewportId.h>

namespace MovementPhysics
{
    class LocalPlayerControllerComponent final
        : public AZ::Component
        , public AZ::TickBus::Handler
        , public AzFramework::InputChannelEventListener
    {
    public:
        AZ_COMPONENT(LocalPlayerControllerComponent, "{65557B61-94AB-456F-B0F0-78C162C55E1B}");

        LocalPlayerControllerComponent();
        ~LocalPlayerControllerComponent() override = default;

        static void Reflect(AZ::ReflectContext* context);
        static void GetProvidedServices(AZ::ComponentDescriptor::DependencyArrayType& provided);
        static void GetIncompatibleServices(AZ::ComponentDescriptor::DependencyArrayType& incompatible);
        static void GetRequiredServices(AZ::ComponentDescriptor::DependencyArrayType& required);
        static void GetDependentServices(AZ::ComponentDescriptor::DependencyArrayType& dependent);

        void Activate() override;
        void Deactivate() override;
        void OnTick(float deltaTime, AZ::ScriptTimePoint time) override;

    protected:
        bool OnInputChannelEventFiltered(const AzFramework::InputChannel& inputChannel) override;

    private:
        void InitializeFromWorldState();
        void UpdateCameraComponent();
        void ResetCameraToEncounterFrame(const AZ::Vector3& playerPosition);
        void UpdateGroundingState();
        void UpdateAvatarPresentation(float deltaTime, const AZ::Vector2& planarDelta);
        AZ::Vector3 GetPresentationPosition() const;
        void SyncEntityTransformToCharacterBase(const AZ::Vector3& basePosition);
        void SetCharacterBasePosition(const AZ::Vector3& basePosition);
        void ApplyWorldPosition(float x, float y, float z);
        void DrawValidationArena();
        void DrawLocalPlayerProxy();

        bool m_initialized = false;
        bool m_leftMouseHeld = false;
        bool m_moveForward = false;
        bool m_moveBackward = false;
        bool m_strafeLeft = false;
        bool m_strafeRight = false;
        bool m_requestDisconnect = false;
        bool m_requestReconnect = false;
        bool m_requestQuit = false;
        bool m_inputListening = false;
        bool m_loggedInputHelp = false;
        bool m_loggedInputReady = false;
        bool m_loggedFirstMovementInput = false;
        bool m_loggedLocomotionMode = false;
        bool m_loggedCameraAttachment = false;
        bool m_loggedCameraReady = false;
        bool m_loggedCameraMode = false;
        bool m_loggedCameraEntity = false;
        bool m_loggedCameraAnchor = false;
        bool m_loggedCameraSource = false;
        bool m_loggedDetachedCameraDisabled = false;
        bool m_loggedViewportCameraOwner = false;
        bool m_loggedCameraTransformWriter = false;
        bool m_loggedOrbitState = false;
        bool m_lastLoggedOrbitState = false;
        bool m_loggedChaseLockState = false;
        bool m_lastLoggedChaseLockState = true;
        bool m_loggedGrounded = false;
        bool m_loggedGroundMovementReady = false;
        bool m_loggedValidationFloor = false;
        bool m_loggedMouseLookState = false;
        bool m_lastLoggedMouseLookState = false;
        bool m_loggedStableLocomotionMode = false;
        bool m_loggedFinalBodyPose = false;
        bool m_loggedAvatarVisiblePose = false;
        bool m_loggedEntitySync = false;
        bool m_loggedMovementTranslationApplied = false;
        bool m_loggedCameraSourceFollow = false;
        bool m_pendingCameraReset = false;
        bool m_cameraOrbitModeActive = false;
        bool m_chaseLockActive = true;
        float m_submitAccumulator = 0.0f;
        float m_avatarFacingRadians = 0.72f;
        float m_avatarStridePhase = 0.0f;
        float m_avatarStrideBlend = 0.0f;
        float m_cameraOrbitYawRadians = 0.0f;
        float m_cameraYawRadians = 0.72f;
        float m_cameraPitchRadians = -0.44f;
        float m_cameraFollowDistance = 4.65f;
        AZ::Vector3 m_cachedFinalPresentationPosition = AZ::Vector3::CreateZero();
        float m_cachedFinalAvatarFacingRadians = 0.72f;
        bool m_cachedFinalPoseValid = false;
        AZ::Vector3 m_lastLoggedFinalBodyPose = AZ::Vector3::CreateZero();
        AZ::Vector3 m_lastLoggedAvatarVisiblePose = AZ::Vector3::CreateZero();
        AZ::Vector3 m_lastLoggedEntitySyncPosition = AZ::Vector3::CreateZero();
        AZ::Vector3 m_lastLoggedCameraSourcePosition = AZ::Vector3::CreateZero();
        AZ::Vector2 m_pendingServerDelta = AZ::Vector2::CreateZero();
        AZ::Vector2 m_pendingServerCorrection = AZ::Vector2::CreateZero();
        AZStd::string m_lastWorldSessionToken;
        AzFramework::ViewportId m_defaultViewportId = AzFramework::InvalidViewportId;
    };
} // namespace MovementPhysics

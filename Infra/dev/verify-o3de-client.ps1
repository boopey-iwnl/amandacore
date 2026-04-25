$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$logsRoot = Join-Path $PSScriptRoot "logs"
$gameLauncherPath = Join-Path $repoRoot "build\\o3de-windows\\bin\\profile\\amandacore.GameLauncher.exe"
$assetProcessorLog = Get-ChildItem -Path $logsRoot -Filter "o3de-assetprocessor-*.log" -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending |
    Select-Object -First 1

function Resolve-ExistingPath([string[]]$candidatePaths, [string]$commandName) {
    foreach ($candidate in $candidatePaths) {
        if (![string]::IsNullOrWhiteSpace($candidate) -and (Test-Path $candidate)) {
            return (Resolve-Path $candidate).Path
        }
    }

    $command = Get-Command $commandName -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    throw "Required tool '$commandName' was not found."
}

$cmake = Resolve-ExistingPath @("C:\\Program Files\\CMake\\bin\\cmake.exe") "cmake"
$o3de = Resolve-ExistingPath @("C:\\O3DE\\25.10.2\\scripts\\o3de.bat") "o3de"
$assetProcessorBatch = Resolve-ExistingPath @("C:\\O3DE\\25.10.2\\bin\\Windows\\profile\\Default\\AssetProcessorBatch.exe") "AssetProcessorBatch"

if (!(Test-Path $gameLauncherPath)) {
    throw "GameLauncher was not found at $gameLauncherPath"
}

if (-not $assetProcessorLog) {
    throw "No captured AssetProcessorBatch log was found in $logsRoot"
}

$assetProcessorContents = Get-Content -Path $assetProcessorLog.FullName -Raw
if ($assetProcessorContents -match "failed with exit code" -or $assetProcessorContents -match "AssetProcessorBatch failed") {
    throw "Latest AssetProcessorBatch run indicates failure. Log: $($assetProcessorLog.FullName)"
}

$launcherSettingsPath = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\\launcher-settings.json"
$resolvedLauncherPath = if (Test-Path $launcherSettingsPath) {
    try {
        $settings = Get-Content -Path $launcherSettingsPath -Raw | ConvertFrom-Json
        if ($settings.ClientExecutablePath) {
            $settings.ClientExecutablePath
        }
        else {
            $gameLauncherPath
        }
    }
    catch {
        $gameLauncherPath
    }
}
else {
    $gameLauncherPath
}

if ($resolvedLauncherPath -ne $gameLauncherPath) {
    throw "Launcher is not currently resolved to the O3DE GameLauncher path. Resolved: $resolvedLauncherPath"
}

$runtimeLog = $null
$gameLogPath = Join-Path $repoRoot "user\\log\\Game.log"
if (Test-Path $gameLogPath) {
    $runtimeLog = Get-Item $gameLogPath
}

if (-not $runtimeLog) {
    $candidateClientLogs = Get-ChildItem -Path (Join-Path $repoRoot "user\\log") -Filter "*.log" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending

    $runtimeLog = $candidateClientLogs |
        Where-Object {
            $content = Get-Content -Path $_.FullName -Raw -ErrorAction SilentlyContinue
            $content -match "client\\.world_connect_started" -or $content -match "client\\.level_ready" -or $content -match "client\\.world_connected"
        } |
        Select-Object -First 1
}

if (-not $runtimeLog) {
    Write-Host "Toolchain resolved:"
    Write-Host "  cmake: $cmake"
    Write-Host "  o3de: $o3de"
    Write-Host "  AssetProcessorBatch: $assetProcessorBatch"
    Write-Host "GameLauncher path: $gameLauncherPath"
    Write-Host "Latest AssetProcessorBatch log: $($assetProcessorLog.FullName)"
    Write-Host ""
    Write-Host "No O3DE runtime log with client world events was found yet."
    Write-Host "Manual runtime verification is required:"
    Write-Host "  1. Start the local stack and launcher."
    Write-Host "  2. Join the world through the launcher and confirm the O3DE GameLauncher is selected."
    Write-Host "  3. Confirm TestZone01 loads, the player remains grounded, and the west-approach landmarks render."
    Write-Host "  4. Accept field orders at the command post, defeat all three Ember Hounds, and confirm the reward grants once."
    Write-Host "  5. Disconnect/reconnect and restart the stack to confirm position and quest persistence."
    return
}

$runtimeContent = Get-Content -Path $runtimeLog.FullName -Raw
$connectStartedIndex = $runtimeContent.IndexOf("client.world_connect_started")
$levelReadyIndex = $runtimeContent.IndexOf("client.level_ready")
$worldConnectedIndex = $runtimeContent.IndexOf("client.world_connected")
$playerSpawnedIndex = $runtimeContent.IndexOf("client.player_spawned")

if ($connectStartedIndex -lt 0 -or $levelReadyIndex -lt 0 -or $worldConnectedIndex -lt 0) {
    throw "Runtime log is missing one or more required events. Log: $($runtimeLog.FullName)"
}

if (!($connectStartedIndex -lt $levelReadyIndex -and $levelReadyIndex -lt $worldConnectedIndex)) {
    throw "Runtime log events are out of order. Expected client.world_connect_started -> client.level_ready -> client.world_connected. Log: $($runtimeLog.FullName)"
}

if ($playerSpawnedIndex -lt 0 -or $worldConnectedIndex -gt $playerSpawnedIndex) {
    throw "client.player_spawned did not occur after client.world_connected. Log: $($runtimeLog.FullName)"
}

Write-Host "Playable slice O3DE smoke verification passed."
Write-Host "Toolchain resolved:"
Write-Host "  cmake: $cmake"
Write-Host "  o3de: $o3de"
Write-Host "  AssetProcessorBatch: $assetProcessorBatch"
Write-Host "GameLauncher path: $gameLauncherPath"
Write-Host "Latest AssetProcessorBatch log: $($assetProcessorLog.FullName)"
Write-Host "Runtime log: $($runtimeLog.FullName)"
Write-Host ""
Write-Host "Verified runtime phases:"
Write-Host "  1. GameLauncher starts and TestZone01 loads."
Write-Host "  2. World bootstrap/connect runs after level-ready."
Write-Host "  3. Player spawn occurs after successful world connect."

param(
    [switch]$BuildFirst,
    [switch]$StartLauncher = $false
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$servicesRoot = Join-Path $repoRoot "Services"
$logsRoot = Join-Path $PSScriptRoot "logs"
$processManifest = Join-Path $PSScriptRoot "local-processes.json"
$versionManifestScript = Join-Path $PSScriptRoot "write-version-manifest.ps1"
$versionManifestPath = Join-Path $PSScriptRoot "version-manifest.json"
$secretPath = Join-Path $repoRoot ".secrets\\amandacore.dev.env"
$secretExamplePath = Join-Path $repoRoot "Docs\\Config\\amandacore.dev.env.example"
$launcherExe = Join-Path $repoRoot "Client\\Launcher\\AmandaCore.Launcher\\bin\\Debug\\net8.0-windows\\AmandaCore.Launcher.exe"
$localStateRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore"
$storePath = Join-Path $localStateRoot "platform-state.json"

if (-not $PSBoundParameters.ContainsKey("BuildFirst")) {
    $BuildFirst = $true
}

& (Join-Path $PSScriptRoot "stop-local.ps1") | Out-Null

if ($BuildFirst) {
    & (Join-Path $PSScriptRoot "build-local.ps1")
}

if (!(Test-Path $versionManifestPath)) {
    & $versionManifestScript -OutputPath $versionManifestPath
}

$versionManifest = Get-Content -Path $versionManifestPath -Raw | ConvertFrom-Json

if (!(Test-Path $secretPath)) {
    Copy-Item $secretExamplePath $secretPath
}

New-Item -ItemType Directory -Force -Path $logsRoot | Out-Null
New-Item -ItemType Directory -Force -Path $localStateRoot | Out-Null

$env:AMANDACORE_STORE_PATH = $storePath
$env:AMANDACORE_LOCAL_SEED_FILE = $secretPath
$env:AMANDACORE_BUILD_ID = [string]$versionManifest.buildId
$env:AMANDACORE_BUILD_CHANNEL = [string]$versionManifest.channel
$env:AMANDACORE_DISPLAY_VERSION = [string]$versionManifest.displayVersion
$env:AMANDACORE_BUILD_GENERATED_AT_UTC = [string]$versionManifest.generatedAtUtc
$env:AMANDACORE_CLIENT_VERSION = [string]$versionManifest.clientVersion
$env:AMANDACORE_SERVER_VERSION = [string]$versionManifest.serverVersion
$env:AMANDACORE_CONTENT_VERSION = [string]$versionManifest.contentVersion
$env:AMANDACORE_PROTOCOL_VERSION = [string]$versionManifest.protocolVersion
$env:AMANDACORE_API_VERSION = [string]$versionManifest.apiVersion
$env:AMANDACORE_WORLD_ENDPOINT = if ([string]::IsNullOrWhiteSpace([string]$versionManifest.worldEndpointHint)) { "http://localhost:8085" } else { [string]$versionManifest.worldEndpointHint }
$env:AMANDACORE_REPO_ROOT = $repoRoot

$serviceDefinitions = @(
    @{ Name = "auth-service"; Port = "8081" },
    @{ Name = "account-service"; Port = "8082" },
    @{ Name = "realm-service"; Port = "8083" },
    @{ Name = "character-service"; Port = "8084" },
    @{ Name = "world-service"; Port = "8085" },
    @{ Name = "admin-service"; Port = "8086" }
)

function Wait-ServiceReady($serviceName, $port, $logPath) {
    $deadline = (Get-Date).AddSeconds(20)
    $healthUrl = "http://localhost:$port/health"

    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -Uri $healthUrl -UseBasicParsing -TimeoutSec 2
            if ($response.StatusCode -eq 200) {
                return
            }
        }
        catch {
            Start-Sleep -Milliseconds 500
        }
    }

    if (Test-Path $logPath) {
        $tail = Get-Content $logPath -Tail 20 | Out-String
        throw "$serviceName did not become healthy on port $port.`n$tail"
    }

    throw "$serviceName did not become healthy on port $port."
}

$processes = @()
foreach ($service in $serviceDefinitions) {
    $exePath = Join-Path $servicesRoot "bin\\$($service.Name).exe"
    $logPath = Join-Path $logsRoot "$($service.Name).log"
    $command = "[System.Environment]::SetEnvironmentVariable('AMANDACORE_SERVICE_PORT','$($service.Port)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_STORE_PATH','$($env:AMANDACORE_STORE_PATH)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_LOCAL_SEED_FILE','$secretPath','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_BUILD_ID','$($env:AMANDACORE_BUILD_ID)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_BUILD_CHANNEL','$($env:AMANDACORE_BUILD_CHANNEL)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_DISPLAY_VERSION','$($env:AMANDACORE_DISPLAY_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_BUILD_GENERATED_AT_UTC','$($env:AMANDACORE_BUILD_GENERATED_AT_UTC)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_CLIENT_VERSION','$($env:AMANDACORE_CLIENT_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_SERVER_VERSION','$($env:AMANDACORE_SERVER_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_CONTENT_VERSION','$($env:AMANDACORE_CONTENT_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_PROTOCOL_VERSION','$($env:AMANDACORE_PROTOCOL_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_API_VERSION','$($env:AMANDACORE_API_VERSION)','Process'); [System.Environment]::SetEnvironmentVariable('AMANDACORE_WORLD_ENDPOINT','$($env:AMANDACORE_WORLD_ENDPOINT)','Process'); & '$exePath' *>> '$logPath'"

    $process = Start-Process -FilePath "powershell.exe" -ArgumentList "-NoLogo", "-NoProfile", "-Command", $command -PassThru -WindowStyle Hidden
    $processes += [pscustomobject]@{
        Name = $service.Name
        Id = $process.Id
        Port = $service.Port
        Log = $logPath
    }
}

$processes | ConvertTo-Json | Set-Content $processManifest

foreach ($process in $processes) {
    Wait-ServiceReady $process.Name $process.Port $process.Log
}

if ($StartLauncher -and (Test-Path $launcherExe)) {
    Start-Process -FilePath $launcherExe | Out-Null
}

Write-Host "Local amandacore stack started."
Write-Host "Build ID: $($env:AMANDACORE_BUILD_ID)"
Write-Host "Version manifest: $versionManifestPath"
Write-Host "Process manifest: $processManifest"
Write-Host "State store: $storePath"

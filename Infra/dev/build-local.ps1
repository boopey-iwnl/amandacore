$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$servicesRoot = Join-Path $repoRoot "Services"
$serviceOutput = Join-Path $servicesRoot "bin"
$versionManifestScript = Join-Path $PSScriptRoot "write-version-manifest.ps1"
$versionManifestPath = Join-Path $PSScriptRoot "version-manifest.json"

function Resolve-Tool($preferredPath, $commandName) {
    if (Test-Path $preferredPath) {
        return $preferredPath
    }

    $command = Get-Command $commandName -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    throw "Required tool '$commandName' was not found."
}

$go = Resolve-Tool "C:\Program Files\Go\bin\go.exe" "go"

& $versionManifestScript -OutputPath $versionManifestPath
$versionManifest = Get-Content -Path $versionManifestPath -Raw | ConvertFrom-Json
$env:AMANDACORE_BUILD_ID = [string]$versionManifest.buildId
$env:AMANDACORE_BUILD_CHANNEL = [string]$versionManifest.channel
$env:AMANDACORE_DISPLAY_VERSION = [string]$versionManifest.displayVersion
$env:AMANDACORE_BUILD_GENERATED_AT_UTC = [string]$versionManifest.generatedAtUtc
$env:AMANDACORE_CLIENT_VERSION = [string]$versionManifest.clientVersion
$env:AMANDACORE_SERVER_VERSION = [string]$versionManifest.serverVersion
$env:AMANDACORE_CONTENT_VERSION = [string]$versionManifest.contentVersion
$env:AMANDACORE_PROTOCOL_VERSION = [string]$versionManifest.protocolVersion
$env:AMANDACORE_API_VERSION = [string]$versionManifest.apiVersion

New-Item -ItemType Directory -Force -Path $serviceOutput | Out-Null

Push-Location $servicesRoot
& $go test ./...
foreach ($service in @("auth-service", "account-service", "realm-service", "character-service", "world-service", "admin-service")) {
    & $go build -o (Join-Path $serviceOutput "$service.exe") "./cmd/$service"
}
Pop-Location

& (Join-Path $PSScriptRoot "build-playable-client.ps1")

Write-Host "Build completed for services, launcher, world client, and available O3DE GameLauncher targets."
Write-Host "Build manifest: $versionManifestPath"

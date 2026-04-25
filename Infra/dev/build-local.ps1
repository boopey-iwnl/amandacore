$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$servicesRoot = Join-Path $repoRoot "Services"
$serviceOutput = Join-Path $servicesRoot "bin"

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

New-Item -ItemType Directory -Force -Path $serviceOutput | Out-Null

Push-Location $servicesRoot
& $go test ./...
foreach ($service in @("auth-service", "account-service", "realm-service", "character-service", "world-service", "admin-service")) {
    & $go build -o (Join-Path $serviceOutput "$service.exe") "./cmd/$service"
}
Pop-Location

& (Join-Path $PSScriptRoot "build-playable-client.ps1")

Write-Host "Build completed for services, launcher, world client, and available O3DE GameLauncher targets."

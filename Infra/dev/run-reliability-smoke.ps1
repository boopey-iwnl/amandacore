param(
    [int]$Clients = 2,
    [string]$Duration = "30s",
    [int]$DurationMinutes = 1,
    [switch]$RequireRunningStack
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$scanScript = Join-Path $repoRoot "Infra\qa\Scan-ForbiddenArtifacts.ps1"
$worldLoadsim = Join-Path $PSScriptRoot "run-world-loadsim.ps1"
$stackLoadTest = Join-Path $PSScriptRoot "run-load-test.ps1"

& $scanScript
if ($LASTEXITCODE -ne 0) {
    throw "Forbidden artifact scan failed."
}

& $worldLoadsim -Scenario reconnect-pressure -Clients $Clients -Duration $Duration -CommandRate 2
if ($LASTEXITCODE -ne 0) {
    throw "Reconnect-pressure world loadsim failed."
}

if ($RequireRunningStack) {
    & $stackLoadTest -Clients $Clients -Scenario mixed -DurationMinutes $DurationMinutes -StepInterval "250ms"
    if ($LASTEXITCODE -ne 0) {
        throw "Mixed local-stack load test failed."
    }
}

Write-Host "Reliability smoke completed."

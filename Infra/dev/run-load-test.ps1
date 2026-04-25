param(
    [int]$Clients = 2,
    [ValidateSet("idle", "move", "combat", "reconnect", "mixed")]
    [string]$Scenario = "mixed",
    [int]$DurationMinutes = 5,
    [string]$StepInterval = "250ms",
    [string]$OutputRoot = "",
    [string]$AuthEndpoint = "http://localhost:8081",
    [string]$RealmEndpoint = "http://localhost:8083",
    [string]$CharacterEndpoint = "http://localhost:8084",
    [string]$WorldEndpoint = "http://localhost:8085"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$servicesRoot = Join-Path $repoRoot "Services"
if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path $PSScriptRoot "load-tests"
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null

function Assert-ServiceReady {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url
    )

    try {
        $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -ne 200) {
            throw "Unexpected status $($response.StatusCode)"
        }
    }
    catch {
        throw "Local services do not appear ready at $Url. Start the stack with Infra/dev/start-local.ps1 first. $($_.Exception.Message)"
    }
}

Assert-ServiceReady "$WorldEndpoint/health"

Push-Location $servicesRoot
try {
    go run ./cmd/loadtest-client `
        --clients $Clients `
        --scenario $Scenario `
        --duration "$($DurationMinutes)m" `
        --step-interval $StepInterval `
        --out $OutputRoot `
        --auth-endpoint $AuthEndpoint `
        --realm-endpoint $RealmEndpoint `
        --character-endpoint $CharacterEndpoint `
        --world-endpoint $WorldEndpoint
}
finally {
    Pop-Location
}

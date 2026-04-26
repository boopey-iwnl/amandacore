param(
    [int]$Clients = 100,
    [int]$DurationSeconds = 60,
    [double]$CommandsPerSecond = 5,
    [double]$ReconnectPercentage = 0,
    [string]$Realm = "loadsim-realm",
    [string]$Zone = "loadsim-zone"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$servicesRoot = Join-Path $repoRoot "Services"

Push-Location $servicesRoot
try {
    go run ./cmd/loadsim `
        --clients $Clients `
        --duration "$($DurationSeconds)s" `
        --cmd-rate $CommandsPerSecond `
        --reconnect-percent $ReconnectPercentage `
        --realm $Realm `
        --zone $Zone
}
finally {
    Pop-Location
}

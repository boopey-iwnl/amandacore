param(
    [string]$Source = "",
    [string]$SQLitePath = "",
    [switch]$Apply,
    [switch]$IncludeExpiredTickets,
    [switch]$Json,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"

if ($SelfTest) {
    if (-not (Test-Path (Join-Path $servicesRoot "cmd\statecutover\main.go"))) {
        throw "statecutover command is missing."
    }
    Write-Host "Import-LegacyState self-test passed."
    exit 0
}

if ([string]::IsNullOrWhiteSpace($Source)) {
    $Source = Join-Path $repoRoot "Infra\dev\platform-state.json"
}

$argsList = @("--source", $Source)
if ($SQLitePath -ne "") {
    $argsList += @("--sqlite", $SQLitePath)
}
if ($Apply) {
    $argsList += "--apply"
}
if ($IncludeExpiredTickets) {
    $argsList += "--include-expired-tickets"
}
if ($Json) {
    $argsList += "--json"
}

Push-Location $servicesRoot
try {
    go run ./cmd/statecutover @argsList
}
finally {
    Pop-Location
}

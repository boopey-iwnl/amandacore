param(
    [string]$Store = "",
    [switch]$DryRun,
    [switch]$Status,
    [switch]$Json
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"

$argsList = @()
if ($Store -ne "") {
    $argsList += @("--store", $Store)
}
if ($DryRun) {
    $argsList += "--dry-run"
}
if ($Status) {
    $argsList += "--status"
}
if ($Json) {
    $argsList += "--json"
}

Push-Location $servicesRoot
try {
    go run ./cmd/dbmigrate @argsList
} finally {
    Pop-Location
}

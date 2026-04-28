param(
    [ValidateSet("", "file", "sqlite")]
    [string]$Backend = "",
    [string]$Store = "",
    [string]$SQLitePath = "",
    [switch]$DryRun,
    [switch]$Status,
    [switch]$Check,
    [switch]$Json
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"

$argsList = @()
if ($Backend -ne "") {
    $argsList += @("--backend", $Backend)
}
if ($Store -ne "") {
    $argsList += @("--store", $Store)
}
if ($SQLitePath -ne "") {
    $argsList += @("--sqlite", $SQLitePath)
}
if ($DryRun) {
    $argsList += "--dry-run"
}
if ($Status) {
    $argsList += "--status"
}
if ($Check) {
    $argsList += "--check"
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

param(
    [ValidateSet("file", "sqlite")]
    [string]$Backend = "file",
    [string]$Store = "",
    [string]$SQLitePath = "",
    [switch]$Apply,
    [switch]$Status,
    [switch]$Json,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$migrationScript = Join-Path $repoRoot "Infra\dev\run-db-migrations.ps1"

if ($SelfTest) {
    if (-not (Test-Path $migrationScript)) {
        throw "Migration runner is missing: $migrationScript"
    }
    Write-Host "Check-Migrations self-test passed."
    exit 0
}

$argsList = @("-Backend", $Backend)
if ($Store -ne "") {
    $argsList += @("-Store", $Store)
}
if ($SQLitePath -ne "") {
    $argsList += @("-SQLitePath", $SQLitePath)
}
if ($Status) {
    $argsList += "-Status"
}
elseif (-not $Apply) {
    $argsList += "-Check"
}
if ($Json) {
    $argsList += "-Json"
}

& powershell.exe -NoProfile -ExecutionPolicy Bypass -File $migrationScript @argsList
if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

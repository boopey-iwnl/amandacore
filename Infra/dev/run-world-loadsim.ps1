param(
    [string]$Scenario = "multizone-pressure",
    [int]$Clients = 5,
    [string]$Duration = "10s",
    [double]$CommandRate = 2,
    [string]$Content = "Content/Packs/dawnwake_isles/package.json",
    [int]$Seed = 42,
    [int]$Shards = 1,
    [string]$AssignmentPolicy = "static"
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"
$contentPath = Join-Path $repoRoot $Content

Push-Location $servicesRoot
try {
    go run ./cmd/loadsim `
        --scenario $Scenario `
        --clients $Clients `
        --duration $Duration `
        --cmd-rate $CommandRate `
        --content $contentPath `
        --seed $Seed `
        --shards $Shards `
        --assignment-policy $AssignmentPolicy
} finally {
    Pop-Location
}

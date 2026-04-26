param(
    [string]$OutputPath = (Join-Path $PSScriptRoot "version-manifest.json"),
    [string]$Channel = "development",
    [string]$WorldEndpoint = "http://127.0.0.1:8085",
    [string]$ProtocolVersion = "local-dev-1",
    [string]$ApiVersion = "local-api-1"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$projectJsonPath = Join-Path $repoRoot "project.json"

function Invoke-GitValue {
    param(
        [string[]]$Arguments,
        [string]$Fallback
    )

    $git = Get-Command "git" -ErrorAction SilentlyContinue
    if (-not $git) {
        return $Fallback
    }

    Push-Location $repoRoot
    try {
        $output = & $git.Source @Arguments 2>$null
        if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($output)) {
            return $Fallback
        }

        return (($output | Out-String).Trim())
    }
    finally {
        Pop-Location
    }
}

if (!(Test-Path $projectJsonPath)) {
    throw "project.json was not found at $projectJsonPath"
}

$project = Get-Content -Path $projectJsonPath -Raw | ConvertFrom-Json
$projectVersion = [string]$project.version
if ([string]::IsNullOrWhiteSpace($projectVersion)) {
    $projectVersion = "0.0.0"
}

$gitBranch = Invoke-GitValue -Arguments @("rev-parse", "--abbrev-ref", "HEAD") -Fallback "nogit"
$gitCommit = Invoke-GitValue -Arguments @("rev-parse", "--short=12", "HEAD") -Fallback "nogit"
$gitStatus = Invoke-GitValue -Arguments @("status", "--porcelain") -Fallback ""
$dirtySuffix = if ([string]::IsNullOrWhiteSpace($gitStatus)) { "" } else { ".dirty" }
$safeBranch = [regex]::Replace($gitBranch, "[^A-Za-z0-9._-]", "-")
$buildStamp = (Get-Date).ToUniversalTime().ToString("yyyyMMddHHmmss")
$generatedAtUtc = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$buildId = "amandacore-local-$projectVersion+$safeBranch.$gitCommit$dirtySuffix.$buildStamp"

$o3deWindowsExe = Join-Path $repoRoot "build\windows\bin\profile\amandacore.GameLauncher.exe"
$o3deAltExe = Join-Path $repoRoot "build\o3de-windows\bin\profile\amandacore.GameLauncher.exe"
$fallbackWorldClient = Join-Path $repoRoot "Client\Game\AmandaCore.WorldClient\bin\Debug\net8.0\AmandaCore.WorldClient.exe"

function New-ClientCandidate {
    param(
        [string]$Label,
        [string]$Path
    )

    $exists = Test-Path $Path
    [ordered]@{
        label            = $Label
        path             = $Path
        exists           = $exists
        lastWriteTimeUtc = if ($exists) { (Get-Item $Path).LastWriteTimeUtc.ToString("yyyy-MM-ddTHH:mm:ssZ") } else { $null }
    }
}

$manifest = [ordered]@{
    schemaVersion              = 1
    project                    = "amandacore"
    channel                    = $Channel
    buildId                    = $buildId
    displayVersion             = "$projectVersion-$Channel"
    generatedAtUtc             = $generatedAtUtc
    gitBranch                  = $gitBranch
    gitCommit                  = $gitCommit
    gitDirty                   = -not [string]::IsNullOrWhiteSpace($gitStatus)
    clientVersion              = $projectVersion
    launcherVersion            = $projectVersion
    serverVersion              = $projectVersion
    contentVersion             = $projectVersion
    protocolVersion            = $ProtocolVersion
    apiVersion                 = $ApiVersion
    worldEndpointHint          = $WorldEndpoint
    requiredServices           = @(
        "auth-service",
        "account-service",
        "realm-service",
        "character-service",
        "world-service",
        "admin-service"
    )
    compatibleClientVersions   = @($projectVersion)
    compatibleServerVersions   = @($projectVersion)
    compatibleProtocolVersions = @($ProtocolVersion)
    clientExecutableCandidates = @(
        (New-ClientCandidate -Label "o3de-build-windows" -Path $o3deWindowsExe),
        (New-ClientCandidate -Label "o3de-build-o3de-windows" -Path $o3deAltExe),
        (New-ClientCandidate -Label "fallback-dotnet" -Path $fallbackWorldClient)
    )
    files                      = @()
}

$outputDirectory = Split-Path -Parent $OutputPath
if (![string]::IsNullOrWhiteSpace($outputDirectory)) {
    New-Item -ItemType Directory -Force -Path $outputDirectory | Out-Null
}

$manifest | ConvertTo-Json -Depth 8 | Set-Content -Path $OutputPath -Encoding UTF8
Write-Host "Version manifest written: $OutputPath"
Write-Host "Build ID: $buildId"

param(
    [string]$OutputRoot = "",
    [int]$LogTailLines = 1200,
    [int]$MaxLogBytes = 5242880,
    [switch]$NoArchive,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"

function ConvertTo-RedactedText {
    param(
        [AllowNull()]
        [string]$Text
    )

    if ($null -eq $Text) {
        return ""
    }

    $redacted = $Text
    $keyValuePattern = '(?i)("?((access|refresh|worldSession|reset)?token|ticketId|sessionId|password|passwordHash|adminSeedPassword|AMANDACORE_ADMIN_SEED_PASSWORD|AMANDACORE_DEV_SECRET|POSTGRES_PASSWORD)"?\s*[:=]\s*)("[^"]*"|[^\s,}\]]+)'
    $keyValueEvaluator = [System.Text.RegularExpressions.MatchEvaluator]{
        param([System.Text.RegularExpressions.Match]$Match)

        $prefix = $Match.Groups[1].Value
        $value = $Match.Groups[4].Value
        if ($value.StartsWith('"')) {
            return $prefix + '"<redacted>"'
        }

        return $prefix + '<redacted>'
    }

    $redacted = [System.Text.RegularExpressions.Regex]::Replace($redacted, $keyValuePattern, $keyValueEvaluator)
    $redacted = [System.Text.RegularExpressions.Regex]::Replace($redacted, '(?i)(Authorization:\s*Bearer\s+)[^\s]+', '$1<redacted>')
    $redacted = [System.Text.RegularExpressions.Regex]::Replace($redacted, '(?i)(--join-ticket\s+)[^\s]+', '$1<redacted>')
    $redacted = [System.Text.RegularExpressions.Regex]::Replace($redacted, '(?i)\b(ticket|sess|world|reset)_[a-f0-9]{16,}\b', '<redacted-id>')

    $pathReplacements = @(
        @{ Value = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData); Replacement = "%LOCALAPPDATA%" },
        @{ Value = [Environment]::GetFolderPath([Environment+SpecialFolder]::UserProfile); Replacement = "%USERPROFILE%" }
    )
    foreach ($pathReplacement in $pathReplacements) {
        $pathValue = $pathReplacement.Value
        if ([string]::IsNullOrWhiteSpace($pathValue)) {
            continue
        }

        foreach ($candidate in @($pathValue, $pathValue.Replace("\", "\\"))) {
            $redacted = [System.Text.RegularExpressions.Regex]::Replace(
                $redacted,
                [System.Text.RegularExpressions.Regex]::Escape($candidate),
                $pathReplacement.Replacement,
                [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)
        }
    }

    return $redacted
}

function Invoke-SelfTest {
    $sample = @'
Authorization: Bearer abcdef012345
--join-ticket ticket_1234567890abcdef1234567890abcdef
"accessToken": "token_value"
"refreshToken": "refresh_value"
"passwordHash": "argon_hash"
AMANDACORE_ADMIN_SEED_PASSWORD=secret_value
POSTGRES_PASSWORD=postgres_secret
world_1234567890abcdef1234567890abcdef
'@

    $userProfile = [Environment]::GetFolderPath([Environment+SpecialFolder]::UserProfile)
    $localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
    $sample += "`r`nrepoRoot=$userProfile\OneDrive\Desktop\Code Project"
    $sample += "`r`nstatePath=$localAppData\amandacore\platform-state.json"

    $redacted = ConvertTo-RedactedText $sample
    $forbidden = @(
        "abcdef012345",
        "ticket_1234567890abcdef1234567890abcdef",
        "token_value",
        "refresh_value",
        "argon_hash",
        "secret_value",
        "postgres_secret",
        "world_1234567890abcdef1234567890abcdef",
        $userProfile,
        $localAppData
    )

    foreach ($value in $forbidden) {
        if ($redacted.Contains($value)) {
            throw "Redaction self-test failed. Leaked value: $value"
        }
    }

    if (-not $redacted.Contains("<redacted")) {
        throw "Redaction self-test failed. No redaction marker found."
    }

    Write-Host "Diagnostic redaction self-test passed."
}

if ($SelfTest) {
    Invoke-SelfTest
    return
}

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\diagnostics"
}

$timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
$bundleName = "amandacore-diagnostics-$timestamp"
$bundleRoot = Join-Path $OutputRoot $bundleName
$archivePath = "$bundleRoot.zip"

function New-BundleDirectory {
    param([string]$Path)
    New-Item -ItemType Directory -Force -Path $Path | Out-Null
}

function Write-JsonFile {
    param(
        [Parameter(Mandatory = $true)]
        [object]$Value,
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [int]$Depth = 8
    )

    New-BundleDirectory (Split-Path -Parent $Path)
    $json = $Value | ConvertTo-Json -Depth $Depth
    Set-Content -Path $Path -Value (ConvertTo-RedactedText $json) -Encoding UTF8
}

function Write-TextFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value,
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    New-BundleDirectory (Split-Path -Parent $Path)
    Set-Content -Path $Path -Value $Value -Encoding UTF8
}

function Read-JsonSafe {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        return $null
    }

    try {
        return Get-Content -Path $Path -Raw | ConvertFrom-Json
    }
    catch {
        return $null
    }
}

function Copy-RedactedTextFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$SourcePath,
        [Parameter(Mandatory = $true)]
        [string]$DestinationPath,
        [switch]$TailIfLarge
    )

    if (-not (Test-Path $SourcePath)) {
        return $false
    }

    New-BundleDirectory (Split-Path -Parent $DestinationPath)
    $item = Get-Item -Path $SourcePath
    if ($TailIfLarge -and $item.Length -gt $MaxLogBytes) {
        $content = Get-Content -Path $SourcePath -Tail $LogTailLines -ErrorAction SilentlyContinue | Out-String
        $content = "[truncated to last $LogTailLines lines because source was $($item.Length) bytes]`r`n$content"
    }
    else {
        $content = Get-Content -Path $SourcePath -Raw -ErrorAction SilentlyContinue
    }

    Write-TextFile (ConvertTo-RedactedText $content) $DestinationPath
    return $true
}

function Get-RelativePath {
    param(
        [string]$Root,
        [string]$Path
    )

    $rootFull = [System.IO.Path]::GetFullPath($Root).TrimEnd('\', '/')
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if ($pathFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $pathFull.Substring($rootFull.Length).TrimStart('\', '/')
    }

    return Split-Path -Leaf $Path
}

function Invoke-ToolText {
    param(
        [string]$Command,
        [string[]]$Arguments
    )

    try {
        $output = & $Command @Arguments 2>&1
        return [pscustomobject]@{
            ExitCode = $LASTEXITCODE
            Output = (($output | Out-String).Trim())
        }
    }
    catch {
        return [pscustomobject]@{
            ExitCode = -1
            Output = $_.Exception.Message
        }
    }
}

function Get-ServiceHealthSummary {
    $definitions = @(
        @{ Name = "auth-service"; Port = 8081 },
        @{ Name = "account-service"; Port = 8082 },
        @{ Name = "realm-service"; Port = 8083 },
        @{ Name = "character-service"; Port = 8084 },
        @{ Name = "world-service"; Port = 8085 },
        @{ Name = "admin-service"; Port = 8086 }
    )

    $results = @()
    foreach ($service in $definitions) {
        $uri = "http://localhost:$($service.Port)/health"
        try {
            $response = Invoke-WebRequest -Uri $uri -UseBasicParsing -TimeoutSec 2
            $results += [pscustomobject]@{
                name = $service.Name
                port = $service.Port
                healthy = ($response.StatusCode -eq 200)
                statusCode = $response.StatusCode
                error = ""
            }
        }
        catch {
            $results += [pscustomobject]@{
                name = $service.Name
                port = $service.Port
                healthy = $false
                statusCode = 0
                error = $_.Exception.Message
            }
        }
    }

    return $results
}

function Get-PatchManifestSummary {
    try {
        return [pscustomobject]@{
            available = $true
            manifest = Invoke-RestMethod -Uri "http://localhost:8083/v1/patch/manifest" -Method Get -TimeoutSec 2
            error = ""
        }
    }
    catch {
        return [pscustomobject]@{
            available = $false
            manifest = $null
            error = $_.Exception.Message
        }
    }
}

function Get-SafeStateSummary {
    param([string]$StatePath)

    $state = Read-JsonSafe $StatePath
    if ($null -eq $state) {
        return [pscustomobject]@{
            statePath = $StatePath
            present = (Test-Path $StatePath)
            readable = $false
        }
    }

    $accounts = @($state.accounts.PSObject.Properties)
    $sessions = @($state.sessions.PSObject.Properties)
    $tickets = @($state.worldJoinTickets.PSObject.Properties)
    $characters = @()
    foreach ($property in @($state.characters.PSObject.Properties)) {
        $character = $property.Value
        $characters += [pscustomobject]@{
            id = $character.id
            displayName = $character.displayName
            raceId = $character.raceId
            classId = $character.classId
            archetypeId = $character.archetypeId
            level = $character.level
            zoneId = $character.zoneId
            position = [pscustomobject]@{
                x = $character.positionX
                y = $character.positionY
                z = $character.positionZ
            }
            questCount = @($character.quests.PSObject.Properties).Count
            lastSeenAt = $character.lastSeenAt
        }
    }

    return [pscustomobject]@{
        statePath = $StatePath
        present = $true
        readable = $true
        accountCount = $accounts.Count
        characterCount = $characters.Count
        sessionCount = $sessions.Count
        worldJoinTicketCount = $tickets.Count
        characters = $characters
        buildManifest = $state.buildManifest
    }
}

New-BundleDirectory $bundleRoot

$projectPath = Join-Path $repoRoot "project.json"
$project = Read-JsonSafe $projectPath
$projectVersion = if ($project -and $project.version) { $project.version } else { "unknown" }
$buildId = if (-not [string]::IsNullOrWhiteSpace($env:AMANDACORE_BUILD_ID)) {
    $env:AMANDACORE_BUILD_ID
}
else {
    "amandacore-local-$projectVersion"
}

$gitSummary = [pscustomobject]@{
    available = $false
    branch = ""
    head = ""
    statusShort = ""
}
$git = Get-Command git -ErrorAction SilentlyContinue
if ($git) {
    Push-Location $repoRoot
    try {
        $branch = Invoke-ToolText $git.Source @("branch", "--show-current")
        $head = Invoke-ToolText $git.Source @("rev-parse", "HEAD")
        $status = Invoke-ToolText $git.Source @("status", "--short")
        $gitSummary = [pscustomobject]@{
            available = $true
            branch = $branch.Output
            head = $head.Output
            statusShort = $status.Output
        }
    }
    finally {
        Pop-Location
    }
}

$localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
$statePath = Join-Path $localAppData "amandacore\platform-state.json"
$launcherSettingsPath = Join-Path $localAppData "amandacore\launcher-settings.json"
$launcherSessionPath = Join-Path $localAppData "amandacore\launcher-session.json"
$serviceLogsRoot = Join-Path $repoRoot "Infra\dev\logs"
$userLogRoot = Join-Path $repoRoot "user\log"
$processManifestPath = Join-Path $repoRoot "Infra\dev\local-processes.json"
$contentManifestPath = Join-Path $repoRoot "Content\GameData\ZoneSlice\manifest.json"
$qaDocsRoot = Join-Path $repoRoot "Docs\QA"

$health = Get-ServiceHealthSummary
$patchManifest = Get-PatchManifestSummary
$safeState = Get-SafeStateSummary $statePath

$contentManifestHash = $null
if (Test-Path $contentManifestPath) {
    $contentManifestHash = Get-FileHash -Path $contentManifestPath -Algorithm SHA256
}

$osInfo = $null
try {
    $os = Get-CimInstance Win32_OperatingSystem -ErrorAction Stop
    $osInfo = [pscustomobject]@{
        caption = $os.Caption
        version = $os.Version
        architecture = $os.OSArchitecture
        lastBootUpTime = $os.LastBootUpTime
    }
}
catch {
    $osInfo = [pscustomobject]@{
        caption = [Environment]::OSVersion.VersionString
        version = [Environment]::OSVersion.Version.ToString()
        architecture = ""
        lastBootUpTime = $null
    }
}

$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    buildId = $buildId
    projectVersion = $projectVersion
    repoRoot = $repoRoot
    git = $gitSummary
    os = $osInfo
    powershellVersion = $PSVersionTable.PSVersion.ToString()
    serviceHealth = $health
    patchManifestAvailable = $patchManifest.available
    patchManifestError = $patchManifest.error
    contentManifest = [pscustomobject]@{
        path = $contentManifestPath
        present = (Test-Path $contentManifestPath)
        sha256 = if ($contentManifestHash) { $contentManifestHash.Hash } else { "" }
    }
    paths = [pscustomobject]@{
        serviceLogs = $serviceLogsRoot
        clientLogs = $userLogRoot
        localState = $statePath
        launcherSettings = $launcherSettingsPath
        launcherSessionPresent = (Test-Path $launcherSessionPath)
        processManifest = $processManifestPath
    }
    collectionPolicy = [pscustomobject]@{
        fullLogsUpToBytes = $MaxLogBytes
        tailLinesForLargeLogs = $LogTailLines
        rawSecretsIncluded = $false
        rawLauncherSessionIncluded = $false
        rawPlatformStateIncluded = $false
    }
}

Write-JsonFile $summary (Join-Path $bundleRoot "diagnostic-summary.json")
Write-JsonFile $safeState (Join-Path $bundleRoot "safe-state-summary.json") 10
if ($patchManifest.available) {
    Write-JsonFile $patchManifest.manifest (Join-Path $bundleRoot "build-manifest-live.json") 8
}
else {
    Write-TextFile $patchManifest.error (Join-Path $bundleRoot "build-manifest-live-error.txt")
}

Copy-RedactedTextFile $projectPath (Join-Path $bundleRoot "manifests\project.json") | Out-Null
Copy-RedactedTextFile $contentManifestPath (Join-Path $bundleRoot "manifests\content-manifest.json") | Out-Null
Copy-RedactedTextFile $launcherSettingsPath (Join-Path $bundleRoot "config\launcher-settings.redacted.json") | Out-Null
Copy-RedactedTextFile $processManifestPath (Join-Path $bundleRoot "config\local-processes.redacted.json") | Out-Null

$secretStatus = [pscustomobject]@{
    localSeedFilePresent = (Test-Path (Join-Path $repoRoot ".secrets\amandacore.dev.env"))
    localSeedExamplePresent = (Test-Path (Join-Path $repoRoot "Docs\Config\amandacore.dev.env.example"))
    note = "Secret files are intentionally not included."
}
Write-JsonFile $secretStatus (Join-Path $bundleRoot "config\secret-file-status.json")

$copiedLogs = @()
if (Test-Path $serviceLogsRoot) {
    $serviceLogFiles = @()
    foreach ($serviceLogName in @(
        "auth-service.log",
        "account-service.log",
        "realm-service.log",
        "character-service.log",
        "world-service.log",
        "admin-service.log"
    )) {
        $serviceLogPath = Join-Path $serviceLogsRoot $serviceLogName
        if (Test-Path $serviceLogPath) {
            $serviceLogFiles += Get-Item $serviceLogPath
        }
    }

    $serviceLogFiles += Get-ChildItem -Path $serviceLogsRoot -File -Filter "o3de-assetprocessor-*.log" -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 3

    foreach ($file in $serviceLogFiles | Sort-Object FullName -Unique) {
        $relative = Get-RelativePath $serviceLogsRoot $file.FullName
        $destination = Join-Path $bundleRoot (Join-Path "logs\services" $relative)
        if (Copy-RedactedTextFile $file.FullName $destination -TailIfLarge) {
            $copiedLogs += [pscustomobject]@{ source = $file.FullName; destination = $destination; bytes = $file.Length }
        }
    }
}

if (Test-Path $userLogRoot) {
    $clientLogFiles = @()
    foreach ($clientLogName in @(
        "Game.log",
        "Game(1).log",
        "AP_GUI.log",
        "AP_Batch.log",
        "PakMissingAssets.log",
        "Editor.log"
    )) {
        $clientLogPath = Join-Path $userLogRoot $clientLogName
        if (Test-Path $clientLogPath) {
            $clientLogFiles += Get-Item $clientLogPath
        }
    }

    $jobLogsRoot = Join-Path $userLogRoot "JobLogs"
    if (Test-Path $jobLogsRoot) {
        $clientLogFiles += Get-ChildItem -Path $jobLogsRoot -Recurse -File -Filter "*.log" -ErrorAction SilentlyContinue |
            Sort-Object LastWriteTime -Descending |
            Select-Object -First 20
    }

    foreach ($file in $clientLogFiles | Sort-Object FullName -Unique) {
        $relative = Get-RelativePath $userLogRoot $file.FullName
        $destination = Join-Path $bundleRoot (Join-Path "logs\client" $relative)
        if (Copy-RedactedTextFile $file.FullName $destination -TailIfLarge) {
            $copiedLogs += [pscustomobject]@{ source = $file.FullName; destination = $destination; bytes = $file.Length }
        }
    }
}

Write-JsonFile $copiedLogs (Join-Path $bundleRoot "logs\copied-log-index.json") 6

$crashCandidates = @()
foreach ($root in @($serviceLogsRoot, $userLogRoot)) {
    if (-not (Test-Path $root)) {
        continue
    }

    $crashCandidates += Get-ChildItem -Path $root -Recurse -File -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '(?i)(crash|error|exception|fatal)' } |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 20
}

$crashIndex = @()
foreach ($file in $crashCandidates) {
    $crashIndex += [pscustomobject]@{
        path = $file.FullName
        name = $file.Name
        bytes = $file.Length
        lastWriteTime = $file.LastWriteTimeUtc.ToString("o")
        copied = ($file.Extension -match '(?i)\.(log|txt|json)$')
    }

    if ($file.Extension -match '(?i)\.(log|txt|json)$') {
        $destination = Join-Path $bundleRoot (Join-Path "crash-errors" $file.Name)
        Copy-RedactedTextFile $file.FullName $destination -TailIfLarge | Out-Null
    }
}
Write-JsonFile $crashIndex (Join-Path $bundleRoot "crash-error-index.json") 6

$docsDestination = Join-Path $bundleRoot "tester-docs"
foreach ($relativeDoc in @(
    "Alpha01FeatureFreeze.md",
    "bug-report-template.md",
    "KnownIssues.md",
    "ReleaseNotes.md",
    "TestFocus.md",
    "TesterInstructions.md",
    "PlaytestOperations.md",
    "checklists\closed-alpha-route.md"
)) {
    $source = Join-Path $qaDocsRoot $relativeDoc
    if (Test-Path $source) {
        Copy-RedactedTextFile $source (Join-Path $docsDestination $relativeDoc) | Out-Null
    }
}

$scriptsDestination = Join-Path $bundleRoot "qa-scripts"
foreach ($scriptName in @(
    "..\dev\package-alpha.ps1",
    "Collect-Diagnostics.ps1",
    "Smoke-Test.ps1",
    "Seed-TestAccount.ps1",
    "Reset-LocalTestState.ps1"
)) {
    $source = Join-Path $PSScriptRoot $scriptName
    if (Test-Path $source) {
        Copy-RedactedTextFile $source (Join-Path $scriptsDestination (Split-Path -Leaf $scriptName)) | Out-Null
    }
}

$testerNotes = @'
# Tester Notes

Build ID:
Tester alias:
Date:

## What I tested


## What happened


## Reproduction notes


## Attachments included

- Diagnostic bundle:
- Screenshots/videos:
- Completed checklist:
'@
Write-TextFile $testerNotes (Join-Path $bundleRoot "tester-notes-template.md")

if (-not $NoArchive) {
    if (Test-Path $archivePath) {
        Remove-Item -Path $archivePath -Force
    }

    Compress-Archive -Path (Join-Path $bundleRoot "*") -DestinationPath $archivePath -Force
    Write-Host "Diagnostic bundle created: $archivePath"
}
else {
    Write-Host "Diagnostic bundle folder created: $bundleRoot"
}

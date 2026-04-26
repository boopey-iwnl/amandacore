param(
    [string]$PackageRoot = "",
    [switch]$RequireServices,
    [switch]$RunO3deLevelSmoke,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$results = @()

function Add-Result {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Detail = "",
        [string]$Severity = "error"
    )

    $script:results += [pscustomobject]@{
        name = $Name
        passed = $Passed
        severity = $Severity
        detail = $Detail
    }
}

function Test-PathRequired {
    param(
        [string]$Name,
        [string]$Path
    )

    Add-Result -Name $Name -Passed (Test-Path $Path) -Detail $Path
}

function Test-JsonFile {
    param(
        [string]$Name,
        [string]$Path
    )

    if (-not (Test-Path $Path)) {
        Add-Result -Name $Name -Passed $false -Detail "$Path is missing."
        return
    }

    try {
        Get-Content -Path $Path -Raw | ConvertFrom-Json | Out-Null
        Add-Result -Name $Name -Passed $true -Detail $Path
    }
    catch {
        Add-Result -Name $Name -Passed $false -Detail $_.Exception.Message
    }
}

function Test-TextFileDoesNotMatch {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Pattern,
        [string]$FailureDetail
    )

    if (-not (Test-Path $Path)) {
        Add-Result -Name $Name -Passed $false -Detail "$Path is missing."
        return
    }

    $content = Get-Content -Path $Path -Raw
    Add-Result -Name $Name -Passed ($content -notmatch $Pattern) -Detail $(if ($content -match $Pattern) { $FailureDetail } else { $Path })
}

function Get-RelativePackagePath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Root,
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $rootFull = [System.IO.Path]::GetFullPath($Root).TrimEnd('\', '/')
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if ($pathFull.StartsWith($rootFull + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $pathFull.Substring($rootFull.Length + 1)
    }

    return Split-Path -Leaf $Path
}

function Test-PackageDoesNotContain {
    param(
        [string]$Name,
        [string]$PackageRoot,
        [string[]]$RelativePatterns
    )

    $violations = @()
    foreach ($item in Get-ChildItem -Path $PackageRoot -Recurse -Force -ErrorAction SilentlyContinue) {
        $relative = (Get-RelativePackagePath -Root $PackageRoot -Path $item.FullName).Replace("\", "/")
        foreach ($pattern in $RelativePatterns) {
            if ($relative -match $pattern) {
                $violations += $relative
                break
            }
        }
    }

    Add-Result -Name $Name -Passed ($violations.Count -eq 0) -Detail $(if ($violations.Count -eq 0) { $PackageRoot } else { ($violations | Select-Object -First 20) -join "; " })
}

function Test-BinaryFileContainsAscii {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Needle
    )

    if (-not (Test-Path $Path)) {
        Add-Result -Name $Name -Passed $false -Detail "$Path is missing."
        return
    }

    $haystack = [System.IO.File]::ReadAllBytes($Path)
    $needleBytes = [System.Text.Encoding]::ASCII.GetBytes($Needle)
    $found = $false
    for ($i = 0; $i -le $haystack.Length - $needleBytes.Length; $i++) {
        $matches = $true
        for ($j = 0; $j -lt $needleBytes.Length; $j++) {
            if ($haystack[$i + $j] -ne $needleBytes[$j]) {
                $matches = $false
                break
            }
        }

        if ($matches) {
            $found = $true
            break
        }
    }

    Add-Result -Name $Name -Passed $found -Detail $(if ($found) { $Path } else { "$Needle was not found in $Path" })
}

function Test-PackagedO3deBootstrapOffline {
    param([string]$PackageRoot)

    $cacheRoot = Join-Path $PackageRoot "Cache\pc"
    $bootstrapFiles = @(Get-ChildItem -Path $cacheRoot -Filter "bootstrap*.setreg" -File -ErrorAction SilentlyContinue)
    if ($bootstrapFiles.Count -eq 0) {
        Add-Result -Name "package O3DE bootstrap files exist" -Passed $false -Detail $cacheRoot
        return
    }

    Add-Result -Name "package O3DE bootstrap files exist" -Passed $true -Detail (($bootstrapFiles | ForEach-Object { $_.Name }) -join "; ")

    $violations = @()
    foreach ($file in $bootstrapFiles) {
        $content = Get-Content -Path $file.FullName -Raw
        if (
            $content -match '"connect_to_remote"\s*:\s*1' -or
            $content -match '"wait_for_connect"\s*:\s*1' -or
            $content -match '"connect_ap_timeout"\s*:\s*[1-9]\d*' -or
            $content -match '"launch_ap_timeout"\s*:\s*[1-9]\d*' -or
            $content -match '"wait_ap_ready_timeout"\s*:\s*[1-9]\d*'
        ) {
            $violations += $file.Name
        }
    }

    Add-Result -Name "package O3DE bootstrap uses packaged cache" -Passed ($violations.Count -eq 0) -Detail $(if ($violations.Count -eq 0) { $cacheRoot } else { ($violations -join "; ") })

    $bomFiles = @()
    foreach ($file in $bootstrapFiles) {
        $prefix = @(Get-Content -Path $file.FullName -Encoding Byte -TotalCount 3)
        if ($prefix.Count -eq 3 -and $prefix[0] -eq 0xEF -and $prefix[1] -eq 0xBB -and $prefix[2] -eq 0xBF) {
            $bomFiles += $file.Name
        }
    }

    Add-Result -Name "package O3DE bootstrap has no UTF8 BOM" -Passed ($bomFiles.Count -eq 0) -Detail $(if ($bomFiles.Count -eq 0) { $cacheRoot } else { ($bomFiles -join "; ") })
}

function Stop-PackageRuntimeProcess {
    param(
        [System.Diagnostics.Process]$Process,
        [datetime]$StartedAt
    )

    if ($Process -and -not $Process.HasExited) {
        Stop-Process -Id $Process.Id -Force -ErrorAction SilentlyContinue
    }

    foreach ($name in @("amandacore.GameLauncher", "GameLauncher", "AssetProcessor", "AssetProcessorBatch")) {
        Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
            try {
                if ($_.StartTime -ge $StartedAt.AddSeconds(-2)) {
                    Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
                }
            }
            catch {
            }
        }
    }
}

function Invoke-PackageO3deLevelSmoke {
    param([string]$PackageRoot)

    $launcherCandidates = @(
        (Join-Path $PackageRoot "build\o3de-windows\bin\profile\amandacore.GameLauncher.exe"),
        (Join-Path $PackageRoot "build\windows\bin\profile\amandacore.GameLauncher.exe")
    )
    $gameLauncherPath = $launcherCandidates | Where-Object { Test-Path $_ } | Select-Object -First 1
    if (-not $gameLauncherPath) {
        Add-Result -Name "package O3DE GameLauncher exists" -Passed $false -Detail ($launcherCandidates -join "; ")
        return
    }

    Add-Result -Name "package O3DE GameLauncher exists" -Passed $true -Detail $gameLauncherPath

    $userRoot = Join-Path $PackageRoot "user"
    Remove-Item -Path $userRoot -Recurse -Force -ErrorAction SilentlyContinue
    $gameLog = Join-Path $userRoot "log\Game.log"
    $startTime = Get-Date
    $process = $null
    $failurePattern = "(?i)(Requested level not found|Startup level asset is not registered|Bootstrap zone mapping did not match|missing AssetCatalog|AssetCatalog.*missing|project path not found|Unable to find product asset|Failed to load RPI system asset|Could not compile asset)"
    $failureMatch = ""
    $levelReady = $false
    $assetCatalogPath = Join-Path $PackageRoot "Cache\pc\assetcatalog.xml"
    $assetCatalogHashBefore = ""
    if (Test-Path $assetCatalogPath) {
        $assetCatalogHashBefore = (Get-FileHash -Path $assetCatalogPath -Algorithm SHA256).Hash
    }

    try {
        $escapedPackageRoot = $PackageRoot.Replace('"', '\"')
        $cacheRoot = (Join-Path $PackageRoot "Cache").Replace('"', '\"')
        $userRootArg = (Join-Path $PackageRoot "user").Replace('"', '\"')
        $logRootArg = (Join-Path $PackageRoot "user\log").Replace('"', '\"')
        $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
        $startInfo.FileName = $gameLauncherPath
        $startInfo.WorkingDirectory = $PackageRoot
        $startInfo.Arguments = "--project-path `"$escapedPackageRoot`" --project-cache-path `"$cacheRoot`" --project-user-path `"$userRootArg`" --project-log-path `"$logRootArg`""
        $startInfo.UseShellExecute = $false
        $startInfo.CreateNoWindow = $true
        $process = [System.Diagnostics.Process]::Start($startInfo)

        $deadline = (Get-Date).AddSeconds(90)
        while ((Get-Date) -lt $deadline) {
            Start-Sleep -Milliseconds 500
            if (Test-Path $gameLog) {
                $content = Get-Content -Path $gameLog -Raw -ErrorAction SilentlyContinue
                if ($content -match $failurePattern) {
                    $failureMatch = $Matches[1]
                    break
                }

                if ($content -match "client\.level_ready") {
                    $levelReady = $true
                    break
                }
            }

            if ($process.HasExited -and -not (Test-Path $gameLog)) {
                break
            }
        }
    }
    catch {
        Add-Result -Name "package O3DE runtime level smoke" -Passed $false -Detail $_.Exception.Message
        return
    }
    finally {
        Stop-PackageRuntimeProcess -Process $process -StartedAt $startTime
    }

    $detail = if (Test-Path $gameLog) {
        if ($failureMatch) {
            "$failureMatch in $gameLog"
        }
        else {
            $gameLog
        }
    }
    else {
        "Game.log was not created at $gameLog"
    }

    $assetCatalogHashAfter = ""
    $assetCatalogHasLevel = $false
    if (Test-Path $assetCatalogPath) {
        $assetCatalogHashAfter = (Get-FileHash -Path $assetCatalogPath -Algorithm SHA256).Hash
        $assetCatalogHasLevel = [bool](Select-String -Path $assetCatalogPath -SimpleMatch -Pattern "levels/testzone01/testzone01.spawnable" -Quiet -ErrorAction SilentlyContinue)
    }

    $assetCatalogPreserved = (
        -not [string]::IsNullOrWhiteSpace($assetCatalogHashBefore) -and
        $assetCatalogHashBefore -eq $assetCatalogHashAfter -and
        $assetCatalogHasLevel
    )
    Add-Result -Name "package O3DE asset catalog preserved during runtime smoke" -Passed $assetCatalogPreserved -Detail $(if ($assetCatalogPreserved) { $assetCatalogHashAfter } else { "before=$assetCatalogHashBefore after=$assetCatalogHashAfter hasTestzone=$assetCatalogHasLevel" })

    $assetProcessorLog = Join-Path $userRoot "log\AP_GUI.log"
    $assetProcessorRan = (Test-Path $assetProcessorLog) -and ((Get-Item $assetProcessorLog).Length -gt 0)
    if ($assetProcessorRan) {
        Add-Result -Name "package O3DE AssetProcessor background log" -Passed $false -Severity "warning" -Detail $assetProcessorLog
    }

    Add-Result -Name "package O3DE runtime level smoke" -Passed ($levelReady -and [string]::IsNullOrWhiteSpace($failureMatch)) -Detail $detail
}

function Test-ServiceHealth {
    param(
        [string]$Name,
        [int]$Port
    )

    try {
        $response = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/health" -UseBasicParsing -TimeoutSec 2
        Add-Result -Name "$Name health" -Passed ($response.StatusCode -eq 200) -Detail "HTTP $($response.StatusCode)"
    }
    catch {
        Add-Result -Name "$Name health" -Passed (-not $RequireServices) -Detail $_.Exception.Message -Severity $(if ($RequireServices) { "error" } else { "warning" })
    }
}

function Invoke-ScriptSelfTest {
    param(
        [string]$Name,
        [string]$ScriptPath
    )

    try {
        & powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File $ScriptPath -SelfTest | Out-Null
        Add-Result -Name "$Name self-test" -Passed $true -Detail $ScriptPath
    }
    catch {
        Add-Result -Name "$Name self-test" -Passed $false -Detail $_.Exception.Message
    }
}

Test-PathRequired -Name "QA bug report template" -Path (Join-Path $repoRoot "Docs\QA\bug-report-template.md")
Test-PathRequired -Name "QA route checklist" -Path (Join-Path $repoRoot "Docs\QA\checklists\closed-alpha-route.md")
Test-PathRequired -Name "Alpha 0.1 feature freeze" -Path (Join-Path $repoRoot "Docs\QA\Alpha01FeatureFreeze.md")
Test-PathRequired -Name "Known issues" -Path (Join-Path $repoRoot "Docs\QA\KnownIssues.md")
Test-PathRequired -Name "Release notes" -Path (Join-Path $repoRoot "Docs\QA\ReleaseNotes.md")
Test-PathRequired -Name "Tester instructions" -Path (Join-Path $repoRoot "Docs\QA\TesterInstructions.md")
Test-PathRequired -Name "Playtest operations" -Path (Join-Path $repoRoot "Docs\QA\PlaytestOperations.md")
Test-JsonFile -Name "Content manifest" -Path (Join-Path $repoRoot "Content\GameData\ZoneSlice\manifest.json")
Test-PathRequired -Name "Alpha package script" -Path (Join-Path $repoRoot "Infra\dev\package-alpha.ps1")

foreach ($script in @(
    @{ Name = "diagnostics"; Path = Join-Path $PSScriptRoot "Collect-Diagnostics.ps1" },
    @{ Name = "seed account"; Path = Join-Path $PSScriptRoot "Seed-TestAccount.ps1" },
    @{ Name = "reset state"; Path = Join-Path $PSScriptRoot "Reset-LocalTestState.ps1" }
)) {
    Test-PathRequired -Name "$($script.Name) script" -Path $script.Path
    Invoke-ScriptSelfTest -Name $script.Name -ScriptPath $script.Path
}

foreach ($service in @(
    @{ Name = "auth-service"; Port = 8081 },
    @{ Name = "account-service"; Port = 8082 },
    @{ Name = "realm-service"; Port = 8083 },
    @{ Name = "character-service"; Port = 8084 },
    @{ Name = "world-service"; Port = 8085 },
    @{ Name = "admin-service"; Port = 8086 }
)) {
    Test-ServiceHealth -Name $service.Name -Port $service.Port
}

if (-not [string]::IsNullOrWhiteSpace($PackageRoot)) {
    Test-PathRequired -Name "package root" -Path $PackageRoot
    foreach ($relativePath in @(
        "Docs\QA\bug-report-template.md",
        "Docs\QA\checklists\closed-alpha-route.md",
        "Docs\QA\KnownIssues.md",
        "Docs\QA\Alpha01FeatureFreeze.md",
        "Docs\QA\ReleaseNotes.md",
        "Docs\QA\ReleaseNotes-Alpha-0.15.md",
        "Docs\QA\TesterInstructions.md",
        "Content\Art\Icons\Abilities\ability_auto_attack.png",
        "Content\Art\Icons\UI\icon_missing.png",
        "Content\Art\Materials\mat_stonewake_grass_lush.material",
        "Content\Packs\dev_foundation\package.json",
        "release-package-manifest.json",
        "Infra\dev\package-alpha.ps1",
        "Infra\dev\version-manifest.json",
        "Infra\qa\Collect-Diagnostics.ps1",
        "Infra\qa\Smoke-Test.ps1"
    )) {
        Test-PathRequired -Name "package file $relativePath" -Path (Join-Path $PackageRoot $relativePath)
    }
    Test-JsonFile -Name "package version manifest" -Path (Join-Path $PackageRoot "Infra\dev\version-manifest.json")
    Test-JsonFile -Name "package release manifest" -Path (Join-Path $PackageRoot "release-package-manifest.json")
    Test-TextFileDoesNotMatch -Name "package manifest has no local absolute paths" -Path (Join-Path $PackageRoot "Infra\dev\version-manifest.json") -Pattern "[A-Za-z]:\\" -FailureDetail "version-manifest.json contains a local absolute Windows path."
    Test-PathRequired -Name "package testzone01 spawnable" -Path (Join-Path $PackageRoot "Cache\pc\levels\testzone01\testzone01.spawnable")
    Test-BinaryFileContainsAscii -Name "package asset catalog references testzone01" -Path (Join-Path $PackageRoot "Cache\pc\assetcatalog.xml") -Needle "levels/testzone01/testzone01.spawnable"
    Test-PackagedO3deBootstrapOffline -PackageRoot $PackageRoot
    Test-PackageDoesNotContain -Name "package excludes local/secrets/build junk" -PackageRoot $PackageRoot -RelativePatterns @(
        '(^|/)\.git(/|$)',
        '(^|/)\.secrets(/|$)',
        '(^|/)\.vs(/|$)',
        '^Client/Portal/',
        '^Cache/pc/(client|dist|infra)/',
        '^Infra/dev/logs/',
        '^user/',
        '(^|/)local-processes\.json$',
        '(^|/)platform-state\.json$',
        '(?i)(^|/)(m2_|milestone2_|milestone3_)[^/]*',
        '(?i)(^|/)milestone.*_runtime_ticket\.txt$',
        '(?i)\.(log|tmp)$',
        '(?i)^(?!Content/Art/).*\.(png|jpg|jpeg)$'
    )

    if ($RunO3deLevelSmoke) {
        Invoke-PackageO3deLevelSmoke -PackageRoot $PackageRoot
    }
}

$failed = @($results | Where-Object { -not $_.passed -and $_.severity -eq "error" })
$warnings = @($results | Where-Object { -not $_.passed -and $_.severity -eq "warning" })
$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    packageRoot = $PackageRoot
    requireServices = [bool]$RequireServices
    passed = $failed.Count -eq 0
    errorCount = $failed.Count
    warningCount = $warnings.Count
    results = $results
}

$summary | ConvertTo-Json -Depth 8
if ($failed.Count -gt 0) {
    exit 1
}

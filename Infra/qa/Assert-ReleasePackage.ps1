param(
    [string]$PackageRoot = "",
    [string]$ArchivePath = "",
    [string]$ReleaseNotesPath = "",
    [switch]$RunSmoke,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$smokeScript = Join-Path $repoRoot "Infra\qa\Smoke-Test.ps1"
$findings = @()
$tempExtractRoot = ""

function Add-Finding {
    param([string]$Name, [bool]$Passed, [string]$Detail = "")
    $script:findings += [pscustomobject]@{
        name = $Name
        passed = $Passed
        detail = $Detail
    }
}

function Get-RelativePath {
    param([string]$Root, [string]$Path)
    $rootFull = [System.IO.Path]::GetFullPath($Root).TrimEnd('\', '/')
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if ($pathFull.StartsWith($rootFull + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $pathFull.Substring($rootFull.Length + 1).Replace('\', '/')
    }
    return (Split-Path -Leaf $Path)
}

function Test-PackageText {
    param([string]$Root)
    $violations = @()
    $allFilePatterns = @(
        '(?i)C:[\\/]Users[\\/]forwo[\\/]Downloads[\\/]textures',
        '(?i)[A-Z]:[\\/].*(side[-_ ]?worktree|local[-_ ]?cache|local[-_ ]?log)',
        '-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----',
        '(?i)(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|xox[baprs]-[A-Za-z0-9-]{20,}'
    )
    $nonCachePatterns = @(
        '(?i)[A-Z]:[\\/](Users|Documents and Settings)[\\/]'
    )
    $binaryExtensions = @(".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".dds", ".tga", ".bmp", ".wav", ".mp3", ".ogg", ".fbx", ".pak", ".dll", ".exe", ".pdb")
    foreach ($file in Get-ChildItem -Path $Root -Recurse -File -Force -ErrorAction SilentlyContinue) {
        if ($binaryExtensions -contains $file.Extension.ToLowerInvariant() -or $file.Length -gt 2MB) {
            continue
        }
        try {
            $content = Get-Content -Path $file.FullName -Raw -ErrorAction Stop
        }
        catch {
            continue
        }
        $relative = Get-RelativePath -Root $Root -Path $file.FullName
        foreach ($pattern in $allFilePatterns) {
            if ($content -match $pattern) {
                $violations += $relative
                break
            }
        }
        if ($relative -notmatch '^Cache/pc/') {
            foreach ($pattern in $nonCachePatterns) {
                if ($content -match $pattern) {
                    $violations += $relative
                    break
                }
            }
        }
    }
    Add-Finding -Name "package text has no secrets or local paths" -Passed ($violations.Count -eq 0) -Detail (($violations | Select-Object -First 20) -join "; ")
}

function Test-PackagePathExclusions {
    param([string]$Root)
    $patterns = @(
        '(^|/)\.git(/|$)',
        '(^|/)\.secrets(/|$)',
        '(^|/)(logs?|diagnostics?|screenshots?)(/|$)',
        '(^|/)(tmp|temp)(/|$)',
        '(?i)(^|/)(runtime[-_]?ticket|platform-state\.json|local-processes\.json)$',
        '(?i)\.(zip|7z|rar|tar|tgz|gz|log|tmp|db|sqlite|sqlite3)$'
    )
    $violations = @()
    foreach ($item in Get-ChildItem -Path $Root -Recurse -Force -ErrorAction SilentlyContinue) {
        $relative = Get-RelativePath -Root $Root -Path $item.FullName
        foreach ($pattern in $patterns) {
            if ($relative -match $pattern) {
                $violations += $relative
                break
            }
        }
    }
    Add-Finding -Name "package excludes forbidden artifacts" -Passed ($violations.Count -eq 0) -Detail (($violations | Select-Object -First 20) -join "; ")
}

if ($SelfTest) {
    $sampleRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("AmandaCoreAssertPackageSelfTest-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $sampleRoot | Out-Null
    Set-Content -Path (Join-Path $sampleRoot "release-package-manifest.json") -Value '{"schemaVersion":1,"sourceCommit":"abc","sourceBranch":"codex/test","buildLabel":"test","assetDigest":{"algorithm":"SHA256","files":[]}}' -Encoding UTF8
    Test-PackagePathExclusions -Root $sampleRoot
    Test-PackageText -Root $sampleRoot
    Remove-Item -LiteralPath $sampleRoot -Recurse -Force
    if (@($findings | Where-Object { -not $_.passed }).Count -gt 0) {
        throw "Assert-ReleasePackage self-test failed."
    }
    Write-Host "Assert-ReleasePackage self-test passed."
    exit 0
}

try {
    if ($ArchivePath -ne "") {
        if (-not (Test-Path $ArchivePath)) {
            throw "Archive is missing: $ArchivePath"
        }
        $tempExtractRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("AmandaCorePackageAssert-" + [guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Force -Path $tempExtractRoot | Out-Null
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory((Resolve-Path $ArchivePath), $tempExtractRoot)
        $PackageRoot = $tempExtractRoot
    }

    if ([string]::IsNullOrWhiteSpace($PackageRoot) -or -not (Test-Path $PackageRoot)) {
        throw "PackageRoot or ArchivePath is required."
    }

    $manifestPath = Join-Path $PackageRoot "release-package-manifest.json"
    Add-Finding -Name "release package manifest exists" -Passed (Test-Path $manifestPath) -Detail $manifestPath
    if (Test-Path $manifestPath) {
        $manifest = Get-Content -Path $manifestPath -Raw | ConvertFrom-Json
        Add-Finding -Name "manifest has source commit" -Passed (-not [string]::IsNullOrWhiteSpace($manifest.sourceCommit)) -Detail $manifest.sourceCommit
        Add-Finding -Name "manifest has source branch" -Passed (-not [string]::IsNullOrWhiteSpace($manifest.sourceBranch)) -Detail $manifest.sourceBranch
        Add-Finding -Name "manifest has build label" -Passed (-not [string]::IsNullOrWhiteSpace($manifest.buildLabel)) -Detail $manifest.buildLabel
        Add-Finding -Name "manifest has asset digest" -Passed ($null -ne $manifest.assetDigest -and $manifest.assetDigest.files.Count -gt 0) -Detail "files=$($manifest.assetDigest.files.Count)"
    }

    if ($ReleaseNotesPath -ne "") {
        Add-Finding -Name "release notes path exists" -Passed (Test-Path $ReleaseNotesPath) -Detail $ReleaseNotesPath
    }

    foreach ($relative in @(
        "Infra\dev\version-manifest.json",
        "Infra\qa\Smoke-Test.ps1",
        "Content\Art\Icons\UI\icon_missing.png",
        "Content\Art\Icons\Abilities\ability_auto_attack.png",
        "Content\Art\Materials\mat_stonewake_grass_lush.material",
        "Content\Packs\dev_foundation\package.json"
    )) {
        Add-Finding -Name "package contains $relative" -Passed (Test-Path (Join-Path $PackageRoot $relative)) -Detail $relative
    }

    Test-PackagePathExclusions -Root $PackageRoot
    Test-PackageText -Root $PackageRoot

    if ($RunSmoke) {
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $smokeScript -PackageRoot $PackageRoot
        Add-Finding -Name "package smoke script" -Passed ($LASTEXITCODE -eq 0) -Detail $smokeScript
    }
}
finally {
    if ($tempExtractRoot -ne "" -and (Test-Path $tempExtractRoot)) {
        Remove-Item -LiteralPath $tempExtractRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}

$failed = @($findings | Where-Object { -not $_.passed })
$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    packageRoot = $PackageRoot
    archivePath = $ArchivePath
    passed = $failed.Count -eq 0
    errorCount = $failed.Count
    results = $findings
}
$summary | ConvertTo-Json -Depth 8
if ($failed.Count -gt 0) {
    exit 1
}

param(
    [string]$OutputRoot = "",
    [string]$PackageName = "",
    [string]$Channel = "alpha-0.1-rc",
    [switch]$SkipBuild,
    [switch]$SkipArchive,
    [switch]$SkipSmoke,
    [switch]$AllowDirty
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$buildScript = Join-Path $PSScriptRoot "build-local.ps1"
$versionManifestScript = Join-Path $PSScriptRoot "write-version-manifest.ps1"
$versionManifestPath = Join-Path $PSScriptRoot "version-manifest.json"
$smokeScript = Join-Path $repoRoot "Infra\qa\Smoke-Test.ps1"

if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path ([System.IO.Path]::GetTempPath()) "AmandaCore\$Channel"
}

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE`: $FilePath $($Arguments -join ' ')"
    }
}

function Get-GitValue {
    param(
        [string[]]$Arguments,
        [string]$Fallback = ""
    )

    $git = Get-Command "git" -ErrorAction SilentlyContinue
    if (-not $git) {
        return $Fallback
    }

    Push-Location $repoRoot
    try {
        $output = & $git.Source @Arguments 2>$null
        if ($LASTEXITCODE -ne 0) {
            return $Fallback
        }
        return (($output | Out-String).Trim())
    }
    finally {
        Pop-Location
    }
}

function Assert-ChildPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Parent,
        [Parameter(Mandatory = $true)]
        [string]$Child
    )

    $parentFull = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\', '/')
    $childFull = [System.IO.Path]::GetFullPath($Child).TrimEnd('\', '/')
    if (-not $childFull.StartsWith($parentFull + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to operate outside package output root. Parent=$parentFull Child=$childFull"
    }
}

function ConvertTo-PackageSafeName {
    param([string]$Value)
    $safe = [regex]::Replace($Value.ToLowerInvariant(), "[^a-z0-9._-]+", "-").Trim("-")
    if ([string]::IsNullOrWhiteSpace($safe)) {
        return "amandacore-release-candidate"
    }
    return $safe
}

function Test-ExcludedPackagePath {
    param([string]$RelativePath)

    $path = $RelativePath.Replace("\", "/")
    $fileName = [System.IO.Path]::GetFileName($path)

    if ($path -match '(^|/)(\.git|\.secrets|\.vs|logs|user)(/|$)') { return $true }
    if ($path -match '(^|/)(Cache|cache)(/|$)' -and $path -notmatch '^Cache($|/pc($|/))') { return $true }
    if ($path -match '^Cache/pc/(client|dist|infra)(/|$)') { return $true }
    if ($path -match '^Client/Portal/') { return $true }
    if ($path -match '^Infra/dev/(local-processes\.json|platform-state\.json|logs/)') { return $true }
    if ($path -match '(^|/)dist(/|$)') { return $true }
    if ($fileName -match '(?i)\.(log|tmp|png|jpg|jpeg|pdb|ilk|lock)$') { return $true }
    if ($fileName -match '(?i)^(m2_|milestone2_|milestone3_)') { return $true }
    if ($fileName -match '(?i)(^required-go-test-output\.txt$|^combat-test-output\.txt$|^worlds-compile-output.*\.txt$|^e2e-.*\.(txt|json|log)$|^milestone.*_runtime_ticket\.txt$|^imgui\.ini$)') { return $true }

    return $false
}

function Copy-PackageFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$SourcePath,
        [Parameter(Mandatory = $true)]
        [string]$RelativePath,
        [switch]$ScrubRepoPath
    )

    if (-not (Test-Path $SourcePath)) {
        return $false
    }

    $destination = Join-Path $script:stagingRoot $RelativePath
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $destination) | Out-Null

    if ($ScrubRepoPath) {
        $content = Get-Content -Path $SourcePath -Raw
        $content = $content.Replace($repoRoot, "%AMANDACORE_PACKAGE_ROOT%")
        $content = $content.Replace($repoRoot.Replace("\", "\\"), "%AMANDACORE_PACKAGE_ROOT%")
        Set-Content -Path $destination -Value $content -Encoding UTF8
    }
    else {
        Copy-Item -Path $SourcePath -Destination $destination -Force
    }

    return $true
}

function Get-PackageRelativePath {
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

function Copy-PackageDirectory {
    param(
        [Parameter(Mandatory = $true)]
        [string]$SourceDirectory,
        [Parameter(Mandatory = $true)]
        [string]$RelativeDirectory
    )

    if (-not (Test-Path $SourceDirectory)) {
        return 0
    }

    $copied = 0
    Get-ChildItem -Path $SourceDirectory -Recurse -File -Force | ForEach-Object {
        $relativeFile = Get-PackageRelativePath -Root $SourceDirectory -Path $_.FullName
        $packageRelative = Join-Path $RelativeDirectory $relativeFile
        if (-not (Test-ExcludedPackagePath $packageRelative)) {
            if (Copy-PackageFile -SourcePath $_.FullName -RelativePath $packageRelative) {
                $copied++
            }
        }
    }
    return $copied
}

function Assert-PackageSafety {
    param([string]$PackageRoot)

    $forbidden = @()
    Get-ChildItem -Path $PackageRoot -Recurse -Force | ForEach-Object {
        $relative = Get-PackageRelativePath -Root $PackageRoot -Path $_.FullName
        if (Test-ExcludedPackagePath $relative) {
            $forbidden += $relative
        }
    }

    if ($forbidden.Count -gt 0) {
        throw "Package contains excluded files or folders: $($forbidden -join ', ')"
    }

    $manifestPath = Join-Path $PackageRoot "Infra\dev\version-manifest.json"
    if (-not (Test-Path $manifestPath)) {
        throw "Package version manifest is missing: $manifestPath"
    }

    $manifestText = Get-Content -Path $manifestPath -Raw
    if ($manifestText -match '[A-Za-z]:\\') {
        throw "Package version manifest contains a local absolute Windows path."
    }
}

function Set-PackagedO3deBootstrapOffline {
    param([string]$PackageRoot)

    $cacheRoot = Join-Path $PackageRoot "Cache\pc"
    if (-not (Test-Path $cacheRoot)) {
        return @()
    }

    $updated = @()
    Get-ChildItem -Path $cacheRoot -Filter "bootstrap*.setreg" -File -ErrorAction SilentlyContinue | ForEach-Object {
        $content = Get-Content -Path $_.FullName -Raw
        $newContent = $content
        $newContent = [regex]::Replace($newContent, '"connect_to_remote"\s*:\s*1', '"connect_to_remote":0')
        $newContent = [regex]::Replace($newContent, '"wait_for_connect"\s*:\s*1', '"wait_for_connect":0')
        $newContent = [regex]::Replace($newContent, '"connect_ap_timeout"\s*:\s*\d+', '"connect_ap_timeout":0')
        $newContent = [regex]::Replace($newContent, '"launch_ap_timeout"\s*:\s*\d+', '"launch_ap_timeout":0')
        $newContent = [regex]::Replace($newContent, '"wait_ap_ready_timeout"\s*:\s*\d+', '"wait_ap_ready_timeout":0')

        if ($newContent -ne $content) {
            Set-Content -Path $_.FullName -Value $newContent -Encoding ASCII -NoNewline
            $updated += (Get-PackageRelativePath -Root $PackageRoot -Path $_.FullName)
        }
    }

    return $updated
}

if (-not $SkipBuild) {
    & $buildScript
}

& $versionManifestScript -OutputPath $versionManifestPath -Channel $Channel
$versionManifest = Get-Content -Path $versionManifestPath -Raw | ConvertFrom-Json

$gitStatus = Get-GitValue -Arguments @("status", "--porcelain") -Fallback ""
if (-not $AllowDirty -and -not [string]::IsNullOrWhiteSpace($gitStatus)) {
    throw "Refusing to create a release package from a dirty worktree. Commit or rerun with -AllowDirty for local validation."
}

$sourceBranch = Get-GitValue -Arguments @("branch", "--show-current") -Fallback "nogit"
$sourceCommit = Get-GitValue -Arguments @("rev-parse", "--short=12", "HEAD") -Fallback "nogit"
if ([string]::IsNullOrWhiteSpace($PackageName)) {
    $PackageName = ConvertTo-PackageSafeName ("amandacore-" + $Channel + "-" + $sourceCommit)
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$script:stagingRoot = Join-Path $OutputRoot $PackageName
Assert-ChildPath -Parent $OutputRoot -Child $script:stagingRoot
if (Test-Path $script:stagingRoot) {
    Remove-Item -LiteralPath $script:stagingRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $script:stagingRoot | Out-Null

$releaseFilesCopied = 0
foreach ($relativePath in @("project.json")) {
    $sourcePath = Join-Path $repoRoot $relativePath
    if (-not (Test-ExcludedPackagePath $relativePath)) {
        if (Copy-PackageFile -SourcePath $sourcePath -RelativePath $relativePath) {
            $releaseFilesCopied++
        }
    }
}

$releaseDirectories = @(
    @{ Source = Join-Path $repoRoot "Docs\QA"; Relative = "Docs\QA" },
    @{ Source = Join-Path $repoRoot "Docs\Config"; Relative = "Docs\Config" },
    @{ Source = Join-Path $repoRoot "Content\GameData"; Relative = "Content\GameData" },
    @{ Source = Join-Path $repoRoot "Content\Schemas"; Relative = "Content\Schemas" },
    @{ Source = Join-Path $repoRoot "Infra\dev"; Relative = "Infra\dev" },
    @{ Source = Join-Path $repoRoot "Infra\qa"; Relative = "Infra\qa" },
    @{ Source = Join-Path $repoRoot "Registry"; Relative = "Registry" }
)

foreach ($releaseDirectory in $releaseDirectories) {
    $releaseFilesCopied += Copy-PackageDirectory -SourceDirectory $releaseDirectory.Source -RelativeDirectory $releaseDirectory.Relative
}

Copy-PackageFile -SourcePath $versionManifestPath -RelativePath "Infra\dev\version-manifest.json" -ScrubRepoPath | Out-Null

$runtimePaths = @(
    @{ Source = Join-Path $repoRoot "Services\bin"; Relative = "Services\bin" },
    @{ Source = Join-Path $repoRoot "Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows"; Relative = "Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows" },
    @{ Source = Join-Path $repoRoot "Client\Game\AmandaCore.WorldClient\bin\Debug\net8.0"; Relative = "Client\Game\AmandaCore.WorldClient\bin\Debug\net8.0" },
    @{ Source = Join-Path $repoRoot "build\windows\bin\profile"; Relative = "build\windows\bin\profile" },
    @{ Source = Join-Path $repoRoot "build\o3de-windows\bin\profile"; Relative = "build\o3de-windows\bin\profile" },
    @{ Source = Join-Path $repoRoot "Cache\pc"; Relative = "Cache\pc" }
)

$runtimeSummary = @()
foreach ($runtimePath in $runtimePaths) {
    $count = Copy-PackageDirectory -SourceDirectory $runtimePath.Source -RelativeDirectory $runtimePath.Relative
    $runtimeSummary += [pscustomobject]@{
        relativePath = $runtimePath.Relative
        present = (Test-Path $runtimePath.Source)
        filesCopied = $count
    }
}

$packagedBootstrapFiles = Set-PackagedO3deBootstrapOffline -PackageRoot $script:stagingRoot

$packageManifest = [ordered]@{
    schemaVersion = 1
    packageName = $PackageName
    packageKind = "$Channel-release-candidate"
    channel = $Channel
    createdAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    sourceBranch = $sourceBranch
    sourceCommit = $sourceCommit
    sourceDirty = -not [string]::IsNullOrWhiteSpace($gitStatus)
    buildId = [string]$versionManifest.buildId
    displayVersion = [string]$versionManifest.displayVersion
    releaseFilesCopied = $releaseFilesCopied
    sourceFilesCopied = 0
    runtimePaths = $runtimeSummary
    packagedBootstrapFiles = $packagedBootstrapFiles
    excludedAreas = @(
        ".git",
        ".secrets",
        "Client/Portal",
        "logs",
        "local state",
        "screenshots",
        "transient test output"
    )
}
$packageManifest | ConvertTo-Json -Depth 8 | Set-Content -Path (Join-Path $script:stagingRoot "release-package-manifest.json") -Encoding UTF8

Assert-PackageSafety -PackageRoot $script:stagingRoot

$archivePath = Join-Path $OutputRoot "$PackageName.zip"
$hashPath = "$archivePath.sha256"
if (-not $SkipArchive) {
    if (Test-Path $archivePath) {
        Remove-Item -LiteralPath $archivePath -Force
    }
    Compress-Archive -Path (Join-Path $script:stagingRoot "*") -DestinationPath $archivePath -Force
    $hash = Get-FileHash -Path $archivePath -Algorithm SHA256
    "{0}  {1}" -f $hash.Hash.ToLowerInvariant(), (Split-Path -Leaf $archivePath) | Set-Content -Path $hashPath -Encoding ASCII
}

if (-not $SkipSmoke) {
    $smokePackageRoot = $script:stagingRoot
    $smokeExtractRoot = ""
    if (-not $SkipArchive) {
        $smokeExtractRoot = Join-Path $OutputRoot "$PackageName-smoke-extract"
        Assert-ChildPath -Parent $OutputRoot -Child $smokeExtractRoot
        Remove-Item -LiteralPath $smokeExtractRoot -Recurse -Force -ErrorAction SilentlyContinue
        Expand-Archive -Path $archivePath -DestinationPath $smokeExtractRoot -Force
        $smokePackageRoot = $smokeExtractRoot
    }

    $smokeSucceeded = $false
    try {
        & $smokeScript -PackageRoot $smokePackageRoot -RunO3deLevelSmoke
        if ($LASTEXITCODE -ne 0) {
            throw "Package smoke test failed."
        }
        $smokeSucceeded = $true
    }
    finally {
        if ($smokeSucceeded -and -not [string]::IsNullOrWhiteSpace($smokeExtractRoot)) {
            Remove-Item -LiteralPath $smokeExtractRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

[pscustomobject]@{
    packageRoot = $script:stagingRoot
    archivePath = if ($SkipArchive) { "" } else { $archivePath }
    sha256Path = if ($SkipArchive) { "" } else { $hashPath }
    buildId = [string]$versionManifest.buildId
    sourceBranch = $sourceBranch
    sourceCommit = $sourceCommit
} | ConvertTo-Json -Depth 6

param(
    [string]$Root = "",
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function Add-Result {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Detail = ""
    )

    $script:results += [pscustomobject]@{
        name = $Name
        passed = $Passed
        detail = $Detail
    }
}

function Get-RelativePath {
    param([string]$RootPath, [string]$Path)

    $rootFull = [System.IO.Path]::GetFullPath($RootPath).TrimEnd('\', '/')
    $pathFull = [System.IO.Path]::GetFullPath($Path)
    if ($pathFull.StartsWith($rootFull + [System.IO.Path]::DirectorySeparatorChar, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $pathFull.Substring($rootFull.Length + 1).Replace('\', '/')
    }
    return $Path.Replace('\', '/')
}

function Get-RepoFiles {
    param([string]$RootPath)

    $git = Get-Command git -ErrorAction SilentlyContinue
    if ($git -and (Test-Path (Join-Path $RootPath ".git"))) {
        Push-Location $RootPath
        try {
            $tracked = & $git.Source ls-files
            if ($LASTEXITCODE -eq 0 -and $tracked.Count -gt 0) {
                return @($tracked | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
            }
        }
        finally {
            Pop-Location
        }
    }

    return @(Get-ChildItem -Path $RootPath -Recurse -File -Force | Where-Object {
        $relative = Get-RelativePath -RootPath $RootPath -Path $_.FullName
        $relative -notmatch '(^|/)(\.git|Cache|cache|build|bin|obj|dist|logs|user)(/|$)'
    } | ForEach-Object { Get-RelativePath -RootPath $RootPath -Path $_.FullName })
}

function Add-ContentArtPathValues {
    param(
        [object]$Value,
        [System.Collections.Generic.List[string]]$Output
    )

    if ($null -eq $Value) {
        return
    }

    if ($Value -is [string]) {
        if ($Value.StartsWith("Content/Art/") -or $Value.StartsWith("Content\Art\")) {
            $Output.Add($Value.Replace('\', '/')) | Out-Null
        }
        return
    }

    if ($Value -is [System.Collections.IEnumerable]) {
        foreach ($item in $Value) {
            Add-ContentArtPathValues -Value $item -Output $Output
        }
        return
    }

    if ($Value -is [pscustomobject]) {
        foreach ($property in $Value.PSObject.Properties) {
            Add-ContentArtPathValues -Value $property.Value -Output $Output
        }
    }
}

function Test-RequiredDocs {
    param([string]$RootPath)

    $required = @(
        "Docs/UI/UIRoadmapIntegrationMilestone10.md",
        "Docs/UI/IntegratedUiContract.md",
        "Docs/UI/PanelStackingAndInputContract.md",
        "Docs/UI/UiReleaseCandidateChecklist.md",
        "Docs/Runbooks/UiManualSmokeTest.md",
        "Docs/QA/UiReleaseCandidateChecklist.md"
    )

    $missing = @()
    foreach ($relative in $required) {
        if (-not (Test-Path (Join-Path $RootPath $relative))) {
            $missing += $relative
        }
    }
    Add-Result -Name "required UI M10 docs exist" -Passed ($missing.Count -eq 0) -Detail (($missing | Select-Object -First 20) -join "; ")
}

function Test-NoAddonDirectories {
    param([string]$RootPath)

    $violations = @(Get-ChildItem -Path $RootPath -Recurse -Directory -Force -ErrorAction SilentlyContinue | Where-Object {
        $_.Name -ieq "AddOns" -or $_.Name -ieq "Addons"
    } | ForEach-Object { Get-RelativePath -RootPath $RootPath -Path $_.FullName })
    Add-Result -Name "no AddOns directories" -Passed ($violations.Count -eq 0) -Detail (($violations | Select-Object -First 20) -join "; ")
}

function Test-NoLuaScripts {
    param([string[]]$Files)

    $violations = @($Files | Where-Object { $_ -match '(?i)\.lua$' })
    Add-Result -Name "no tracked Lua UI/addon scripts" -Passed ($violations.Count -eq 0) -Detail (($violations | Select-Object -First 20) -join "; ")
}

function Test-NoDownloadsTexturePath {
    param([string]$RootPath, [string[]]$Files)

    $binaryExtensions = @(".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".dds", ".tga", ".bmp", ".wav", ".mp3", ".ogg", ".fbx", ".pak", ".dll", ".exe", ".pdb")
    $violations = @()
    foreach ($relative in $Files) {
        $normalized = $relative.Replace('\', '/')
        if ($normalized -match '^Docs/' -and $normalized -notmatch '^Docs/Config/') {
            continue
        }
        $full = Join-Path $RootPath $relative
        if (-not (Test-Path $full -PathType Leaf)) {
            continue
        }
        $extension = [System.IO.Path]::GetExtension($full).ToLowerInvariant()
        if ($binaryExtensions -contains $extension) {
            continue
        }
        $item = Get-Item $full
        if ($item.Length -gt 2MB) {
            continue
        }
        try {
            $content = Get-Content -Path $full -Raw -ErrorAction Stop
        }
        catch {
            continue
        }
        if ($content -match '(?i)C:[\\/]Users[\\/]forwo[\\/]Downloads[\\/]textures' -or
            $content -match '(?i)Downloads[\\/]textures') {
            $violations += $normalized
        }
    }
    Add-Result -Name "no Downloads texture runtime paths" -Passed ($violations.Count -eq 0) -Detail (($violations | Select-Object -First 20) -join "; ")
}

function Test-ContentArtManifestPaths {
    param([string]$RootPath)

    $manifestRoots = @(
        "Content/Art/Manifests",
        "Content/GameData/Maps"
    )
    $missing = @()
    foreach ($manifestRoot in $manifestRoots) {
        $fullRoot = Join-Path $RootPath $manifestRoot
        if (-not (Test-Path $fullRoot)) {
            continue
        }
        foreach ($file in Get-ChildItem -Path $fullRoot -Recurse -File -Filter *.json) {
            $paths = [System.Collections.Generic.List[string]]::new()
            try {
                $json = Get-Content -Path $file.FullName -Raw | ConvertFrom-Json
                Add-ContentArtPathValues -Value $json -Output $paths
            }
            catch {
                $missing += "$(Get-RelativePath -RootPath $RootPath -Path $file.FullName): invalid JSON: $($_.Exception.Message)"
                continue
            }
            foreach ($path in ($paths | Sort-Object -Unique)) {
                if (-not (Test-Path (Join-Path $RootPath $path))) {
                    $missing += "$(Get-RelativePath -RootPath $RootPath -Path $file.FullName): $path"
                }
            }
        }
    }

    Add-Result -Name "repo-side Content/Art manifest paths resolve" -Passed ($missing.Count -eq 0) -Detail (($missing | Select-Object -First 30) -join "; ")
}

if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
}
else {
    $Root = Resolve-Path $Root
}
$repoRoot = [System.IO.Path]::GetFullPath($Root)

if ($SelfTest) {
    $sampleRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("AmandaCoreUiSmokeSelfTest-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $sampleRoot | Out-Null
    foreach ($relative in @(
        "Docs/UI/UIRoadmapIntegrationMilestone10.md",
        "Docs/UI/IntegratedUiContract.md",
        "Docs/UI/PanelStackingAndInputContract.md",
        "Docs/UI/UiReleaseCandidateChecklist.md",
        "Docs/Runbooks/UiManualSmokeTest.md",
        "Docs/QA/UiReleaseCandidateChecklist.md",
        "Content/Art/Icons/UI/icon_missing.png",
        "Content/Art/Manifests/test-ui-manifest.json"
    )) {
        $path = Join-Path $sampleRoot $relative
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $path) | Out-Null
        Set-Content -Path $path -Value "self-test" -Encoding UTF8
    }
    Set-Content -Path (Join-Path $sampleRoot "Content/Art/Manifests/test-ui-manifest.json") -Value '{"path":"Content/Art/Icons/UI/icon_missing.png"}' -Encoding UTF8
    $repoRoot = $sampleRoot
}

$script:results = @()
$files = @(Get-RepoFiles -RootPath $repoRoot)
Test-RequiredDocs -RootPath $repoRoot
Test-NoAddonDirectories -RootPath $repoRoot
Test-NoLuaScripts -Files $files
Test-NoDownloadsTexturePath -RootPath $repoRoot -Files $files
Test-ContentArtManifestPaths -RootPath $repoRoot

$failed = @($script:results | Where-Object { -not $_.passed })
$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    root = $repoRoot
    passed = $failed.Count -eq 0
    errorCount = $failed.Count
    results = $script:results
}
$summary | ConvertTo-Json -Depth 8

if ($SelfTest -and (Test-Path $sampleRoot)) {
    Remove-Item -LiteralPath $sampleRoot -Recurse -Force -ErrorAction SilentlyContinue
}

if ($failed.Count -gt 0) {
    exit 1
}

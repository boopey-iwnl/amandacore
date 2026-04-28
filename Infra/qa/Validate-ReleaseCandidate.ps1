param(
    [switch]$SkipO3DE,
    [switch]$RunPackageSmoke,
    [switch]$RunSoak,
    [int]$SoakUsers = 2,
    [int]$SoakDurationMinutes = 1,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"
$results = @()

function Add-Result {
    param([string]$Name, [bool]$Passed, [string]$Detail = "")
    $script:results += [pscustomobject]@{
        name = $Name
        passed = $Passed
        detail = $Detail
    }
}

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Block
    )
    try {
        & $Block
        Add-Result -Name $Name -Passed $true
    }
    catch {
        Add-Result -Name $Name -Passed $false -Detail $_.Exception.Message
    }
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

if ($SelfTest) {
    foreach ($relative in @(
        "Infra\qa\Scan-ForbiddenArtifacts.ps1",
        "Infra\dev\build-local.ps1",
        "Infra\qa\Assert-ReleasePackage.ps1",
        "Infra\qa\Run-ScaleSoak.ps1"
    )) {
        if (-not (Test-Path (Join-Path $repoRoot $relative))) {
            throw "Missing required script: $relative"
        }
    }
    Write-Host "Validate-ReleaseCandidate self-test passed."
    exit 0
}

Push-Location $repoRoot
try {
    $branch = (& git branch --show-current).Trim()
    $commit = (& git rev-parse HEAD).Trim()
    $status = (& git status --porcelain | Out-String).Trim()
    Add-Result -Name "git branch resolved" -Passed (-not [string]::IsNullOrWhiteSpace($branch)) -Detail $branch
    Add-Result -Name "git commit resolved" -Passed (-not [string]::IsNullOrWhiteSpace($commit)) -Detail $commit
    Add-Result -Name "worktree clean" -Passed ([string]::IsNullOrWhiteSpace($status)) -Detail $(if ($status) { $status } else { "clean" })
}
finally {
    Pop-Location
}

if (@($results | Where-Object { -not $_.passed }).Count -eq 0) {
    Invoke-Step -Name "Go tests" -Block {
        Push-Location $servicesRoot
        try {
            Invoke-Native -FilePath "go" -Arguments @("test", "./...", "-count=1", "-timeout", "15m")
        }
        finally {
            Pop-Location
        }
    }

    Invoke-Step -Name "contract docs present" -Block {
        foreach ($relative in @(
            "Docs\Contracts\http-api-v1.json",
            "Docs\Contracts\replication-v1.json",
            "Docs\Architecture\ReliabilitySecurityCI.md",
            "Docs\Runbooks\ReleaseGateChecklist.md"
        )) {
            if (-not (Test-Path (Join-Path $repoRoot $relative))) {
                throw "Missing $relative"
            }
        }
    }

    Invoke-Step -Name "forbidden artifact scan" -Block {
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\qa\Scan-ForbiddenArtifacts.ps1")
        if ($LASTEXITCODE -ne 0) {
            throw "Forbidden artifact scan failed."
        }
    }

    Invoke-Step -Name "local build" -Block {
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\dev\build-local.ps1")
        if ($LASTEXITCODE -ne 0) {
            throw "Local build failed."
        }
    }

    if (-not $SkipO3DE) {
        Invoke-Step -Name "O3DE client build" -Block {
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\dev\build-o3de-client.ps1")
            if ($LASTEXITCODE -ne 0) {
                throw "O3DE build failed."
            }
        }
        Invoke-Step -Name "O3DE client verify" -Block {
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\dev\verify-o3de-client.ps1")
            if ($LASTEXITCODE -ne 0) {
                throw "O3DE verify failed."
            }
        }
    }
    else {
        Add-Result -Name "O3DE client build" -Passed $true -Detail "skipped"
        Add-Result -Name "O3DE client verify" -Passed $true -Detail "skipped"
    }

    if ($RunPackageSmoke) {
        Invoke-Step -Name "package smoke" -Block {
            $outputRoot = Join-Path ([System.IO.Path]::GetTempPath()) "AmandaCore\rc-validation-package"
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\dev\package-alpha.ps1") `
                -OutputRoot $outputRoot `
                -Channel "alpha-rc-validation" `
                -BuildLabel "alpha-rc-validation" `
                -ReleaseNotesPath (Join-Path $repoRoot "Docs\QA\ReleaseNotes.md")
            if ($LASTEXITCODE -ne 0) {
                throw "Package smoke failed."
            }
        }
    }
    else {
        Add-Result -Name "package smoke" -Passed $true -Detail "skipped"
    }

    if ($RunSoak) {
        Invoke-Step -Name "scale soak" -Block {
            & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot "Infra\qa\Run-ScaleSoak.ps1") `
                -Mode runtime `
                -Users $SoakUsers `
                -Duration "$($SoakDurationMinutes)m"
            if ($LASTEXITCODE -ne 0) {
                throw "Scale soak failed."
            }
        }
    }
    else {
        Add-Result -Name "scale soak" -Passed $true -Detail "skipped"
    }
}

$failed = @($results | Where-Object { -not $_.passed })
$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    branch = $branch
    commit = $commit
    passed = $failed.Count -eq 0
    errorCount = $failed.Count
    results = $results
}
$summary | ConvertTo-Json -Depth 8
if ($failed.Count -gt 0) {
    exit 1
}

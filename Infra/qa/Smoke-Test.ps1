param(
    [string]$PackageRoot = "",
    [switch]$RequireServices,
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

function Test-ServiceHealth {
    param(
        [string]$Name,
        [int]$Port
    )

    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port/health" -UseBasicParsing -TimeoutSec 2
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
Test-PathRequired -Name "Known issues" -Path (Join-Path $repoRoot "Docs\QA\KnownIssues.md")
Test-PathRequired -Name "Release notes" -Path (Join-Path $repoRoot "Docs\QA\ReleaseNotes.md")
Test-PathRequired -Name "Tester instructions" -Path (Join-Path $repoRoot "Docs\QA\TesterInstructions.md")
Test-PathRequired -Name "Playtest operations" -Path (Join-Path $repoRoot "Docs\QA\PlaytestOperations.md")
Test-JsonFile -Name "Content manifest" -Path (Join-Path $repoRoot "Content\GameData\ZoneSlice\manifest.json")

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
        "Docs\QA\ReleaseNotes.md",
        "Docs\QA\TesterInstructions.md",
        "Infra\qa\Collect-Diagnostics.ps1",
        "Infra\qa\Smoke-Test.ps1"
    )) {
        Test-PathRequired -Name "package file $relativePath" -Path (Join-Path $PackageRoot $relativePath)
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

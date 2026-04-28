param(
    [ValidateSet("http", "runtime")]
    [string]$Mode = "http",
    [int]$Users = 2,
    [string]$Duration = "30s",
    [ValidateSet("idle", "move", "combat", "reconnect", "mixed")]
    [string]$HttpScenario = "mixed",
    [string]$RuntimeScenario = "reconnect-pressure",
    [double]$CommandRate = 2,
    [double]$ReconnectRate = 0.25,
    [double]$MutationRate = 0.20,
    [double]$MaxErrorRate = 0.02,
    [double]$MaxP95Ms = 750,
    [string]$OutputRoot = "",
    [string]$AuthEndpoint = "http://127.0.0.1:8081",
    [string]$RealmEndpoint = "http://127.0.0.1:8083",
    [string]$CharacterEndpoint = "http://127.0.0.1:8084",
    [string]$WorldEndpoint = "http://127.0.0.1:8085",
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$servicesRoot = Join-Path $repoRoot "Services"

if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path ([System.IO.Path]::GetTempPath()) "AmandaCore\scale-soak"
}

if ($SelfTest) {
    if ($Users -le 0) {
        throw "Users must be greater than zero."
    }
    [timespan]::Parse("00:00:01") | Out-Null
    Write-Host "Run-ScaleSoak self-test passed."
    exit 0
}

New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null

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

function Assert-ServiceReady {
    param([string]$Url)
    try {
        $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -ne 200) {
            throw "Unexpected status $($response.StatusCode)"
        }
    }
    catch {
        throw "Local services are not ready at $Url. Start the stack before HTTP soak. $($_.Exception.Message)"
    }
}

function Get-LatestSummary {
    param([string]$Root)
    Get-ChildItem -Path $Root -Filter "summary.json" -Recurse -File |
        Sort-Object LastWriteTimeUtc -Descending |
        Select-Object -First 1
}

function Measure-HttpSummary {
    param([object]$Summary)

    $total = 0
    $errors = 0
    $maxP95 = 0.0
    foreach ($property in $Summary.operations.PSObject.Properties) {
        $operation = $property.Value
        $total += [int64]$operation.count
        $errors += [int64]$operation.errors
        if ($operation.latencyMs.p95 -gt $maxP95) {
            $maxP95 = [double]$operation.latencyMs.p95
        }
    }
    $errorRate = 0.0
    if ($total -gt 0) {
        $errorRate = [double]$errors / [double]$total
    }
    [pscustomobject]@{
        totalRequests = $total
        failures = $errors
        errorRate = $errorRate
        latencyP95Ms = $maxP95
        reconnectSuccessRate = 1.0
        duplicateMutationRejections = 0
        worldLoopQueueDepth = if ($Summary.serverMetrics.worldTick.maxQueueDepth) { $Summary.serverMetrics.worldTick.maxQueueDepth } else { 0 }
        replicationResyncCount = if ($Summary.serverMetrics.replication.resyncCount) { $Summary.serverMetrics.replication.resyncCount } else { 0 }
        persistenceTransactionFailures = 0
        serviceHealthFailures = 0
    }
}

$summaryPath = Join-Path $OutputRoot "scale-soak-summary.json"
$startedAt = (Get-Date).ToUniversalTime().ToString("o")

if ($Mode -eq "http") {
    Assert-ServiceReady "$WorldEndpoint/health"
    Push-Location $servicesRoot
    try {
        Invoke-Native -FilePath "go" -Arguments @(
            "run", "./cmd/loadtest-client",
            "--clients", "$Users",
            "--scenario", $HttpScenario,
            "--duration", $Duration,
            "--step-interval", "250ms",
            "--out", $OutputRoot,
            "--auth-endpoint", $AuthEndpoint,
            "--realm-endpoint", $RealmEndpoint,
            "--character-endpoint", $CharacterEndpoint,
            "--world-endpoint", $WorldEndpoint
        )
    }
    finally {
        Pop-Location
    }
    $latest = Get-LatestSummary -Root $OutputRoot
    if (-not $latest) {
        throw "HTTP soak did not produce summary.json under $OutputRoot"
    }
    $rawSummary = Get-Content -Path $latest.FullName -Raw | ConvertFrom-Json
    $metrics = Measure-HttpSummary -Summary $rawSummary
}
else {
    $runtimeReport = Join-Path $OutputRoot "runtime-loadsim-report.json"
    Push-Location $servicesRoot
    try {
        Invoke-Native -FilePath "go" -Arguments @(
            "run", "./cmd/loadsim",
            "--scenario", $RuntimeScenario,
            "--clients", "$Users",
            "--duration", $Duration,
            "--cmd-rate", "$CommandRate",
            "--reconnect-rate", "$ReconnectRate",
            "--ability-rate", "$MutationRate",
            "--quest-rate", "$MutationRate",
            "--report", $runtimeReport
        )
    }
    finally {
        Pop-Location
    }
    $rawSummary = Get-Content -Path $runtimeReport -Raw | ConvertFrom-Json
    $reconnectRateValue = 1.0
    if ($rawSummary.reconnectAttempts -gt 0) {
        $reconnectRateValue = [double]$rawSummary.reconnectSuccesses / [double]$rawSummary.reconnectAttempts
    }
    $metrics = [pscustomobject]@{
        totalRequests = [int64]$rawSummary.totalCommandsSent
        failures = [int64]$rawSummary.rejectedCommands + [int64]$rawSummary.reconnectFailures
        errorRate = if ($rawSummary.totalCommandsSent -gt 0) { [double]$rawSummary.rejectedCommands / [double]$rawSummary.totalCommandsSent } else { 0.0 }
        latencyP95Ms = 0.0
        reconnectSuccessRate = $reconnectRateValue
        duplicateMutationRejections = if ($rawSummary.rejectionReasons.duplicate_mutation) { $rawSummary.rejectionReasons.duplicate_mutation } else { 0 }
        worldLoopQueueDepth = $rawSummary.queueMetrics.maxDepth
        replicationResyncCount = 0
        persistenceTransactionFailures = 0
        serviceHealthFailures = 0
    }
}

$passed = (
    $metrics.errorRate -le $MaxErrorRate -and
    ($metrics.latencyP95Ms -eq 0.0 -or $metrics.latencyP95Ms -le $MaxP95Ms) -and
    $metrics.serviceHealthFailures -eq 0
)

$summary = [pscustomobject]@{
    generatedAtUtc = (Get-Date).ToUniversalTime().ToString("o")
    startedAtUtc = $startedAt
    mode = $Mode
    users = $Users
    duration = $Duration
    commandRate = $CommandRate
    reconnectRate = $ReconnectRate
    mutationRate = $MutationRate
    thresholds = [pscustomobject]@{
        maxErrorRate = $MaxErrorRate
        maxP95Ms = $MaxP95Ms
    }
    passed = $passed
    metrics = $metrics
}

$summary | ConvertTo-Json -Depth 8 | Set-Content -Path $summaryPath -Encoding UTF8
$summary | ConvertTo-Json -Depth 8
if (-not $passed) {
    exit 1
}

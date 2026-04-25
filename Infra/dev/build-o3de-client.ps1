$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$logsRoot = Join-Path $PSScriptRoot "logs"
$buildRoot = Join-Path $repoRoot "build\\o3de-windows"
$gameLauncherTarget = "amandacore.GameLauncher"
$assetProcessorLog = Join-Path $logsRoot ("o3de-assetprocessor-{0}.log" -f (Get-Date -Format "yyyyMMdd-HHmmss"))

New-Item -ItemType Directory -Force -Path $logsRoot | Out-Null

function Resolve-ExistingPath([string[]]$candidatePaths, [string]$commandName) {
    foreach ($candidate in $candidatePaths) {
        if (![string]::IsNullOrWhiteSpace($candidate) -and (Test-Path $candidate)) {
            return (Resolve-Path $candidate).Path
        }
    }

    $command = Get-Command $commandName -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    throw "Required tool '$commandName' was not found."
}

function Invoke-CmdChain {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command,
        [string]$LogPath
    )

    if ($LogPath) {
        & cmd.exe /c $Command *>&1 | Tee-Object -FilePath $LogPath
        $exitCode = $LASTEXITCODE
    }
    else {
        & cmd.exe /c $Command
        $exitCode = $LASTEXITCODE
    }

    if ($exitCode -ne 0) {
        if ($LogPath) {
            throw "Command failed with exit code $exitCode. Log: $LogPath"
        }

        throw "Command failed with exit code $exitCode."
    }
}

function Invoke-O3de {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    & $script:o3de @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "o3de command failed: $($Arguments -join ' ')"
    }
}

function Get-RegistrationOutput {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $output = & $script:o3de @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "o3de command failed: $($Arguments -join ' ')"
    }

    return ($output | Out-String)
}

$cmake = Resolve-ExistingPath @(
    "C:\\Program Files\\CMake\\bin\\cmake.exe"
) "cmake"

$vcVars = Resolve-ExistingPath @(
    "C:\\Program Files\\Microsoft Visual Studio\\2022\\Community\\VC\\Auxiliary\\Build\\vcvars64.bat",
    "C:\\Program Files\\Microsoft Visual Studio\\2022\\BuildTools\\VC\\Auxiliary\\Build\\vcvars64.bat"
) "vcvars64.bat"

$o3de = Resolve-ExistingPath @(
    "C:\\O3DE\\25.10.2\\scripts\\o3de.bat"
) "o3de"

$assetProcessorBatch = Resolve-ExistingPath @(
    "C:\\O3DE\\25.10.2\\bin\\Windows\\profile\\Default\\AssetProcessorBatch.exe"
) "AssetProcessorBatch"

$engineRoot = Split-Path -Parent (Split-Path -Parent $o3de)
$engineJsonPath = Join-Path $engineRoot "engine.json"
if (!(Test-Path $engineJsonPath)) {
    throw "Unable to locate engine.json at $engineJsonPath"
}

$engineJson = Get-Content -Path $engineJsonPath -Raw | ConvertFrom-Json
$engineName = [string]$engineJson.engine_name
if ([string]::IsNullOrWhiteSpace($engineName)) {
    throw "engine.json did not contain engine_name."
}

Write-Host "Resolved toolchain:"
Write-Host "  cmake: $cmake"
Write-Host "  vcvars64: $vcVars"
Write-Host "  o3de: $o3de"
Write-Host "  AssetProcessorBatch: $assetProcessorBatch"
Write-Host "  engine root: $engineRoot"

$toolCheckCommand = "call `"$vcVars`" >nul && where cl && where rc && where mt"
Invoke-CmdChain -Command $toolCheckCommand

$registeredEngines = Get-RegistrationOutput -Arguments @("register-show", "-e")
if ($registeredEngines -notmatch [regex]::Escape($engineRoot.Replace('\', '/'))) {
    Write-Host "Registering O3DE engine..."
    Push-Location $engineRoot
    try {
        Invoke-O3de -Arguments @("register", "--this-engine")
    }
    finally {
        Pop-Location
    }
}

$registeredProjects = Get-RegistrationOutput -Arguments @("register-show", "-p")
if ($registeredProjects -notmatch [regex]::Escape($repoRoot.Replace('\', '/'))) {
    Write-Host "Registering project..."
    Invoke-O3de -Arguments @("register", "--project-path", $repoRoot, "--force")
}

$projectEngineName = (Get-RegistrationOutput -Arguments @("register-show", "-pen", "--project-path", $repoRoot)) -replace '(?s)^.*?Engine Name:\s*', '' -replace '\[INFO\].*$', ''
$projectEngineName = $projectEngineName.Trim()
if ($projectEngineName -ne $engineName) {
    Write-Host "Repairing project engine association..."
    Invoke-O3de -Arguments @("edit-project-properties", "--project-path", $repoRoot, "--user", "--engine-path", $engineRoot)
    $projectEngineName = (Get-RegistrationOutput -Arguments @("register-show", "-pen", "--project-path", $repoRoot)) -replace '(?s)^.*?Engine Name:\s*', '' -replace '\[INFO\].*$', ''
    $projectEngineName = $projectEngineName.Trim()
    if ($projectEngineName -ne $engineName) {
        throw "Project engine association is invalid. Expected '$engineName', found '$projectEngineName'."
    }
}

Write-Host "O3DE registration checks passed."

$configureCommand = "call `"$vcVars`" >nul && `"$cmake`" -S `"$repoRoot`" -B `"$buildRoot`" -G `"Visual Studio 17 2022`" -A x64"
Invoke-CmdChain -Command $configureCommand

$buildCommand = "call `"$vcVars`" >nul && `"$cmake`" --build `"$buildRoot`" --target $gameLauncherTarget --config profile -- /m"
Invoke-CmdChain -Command $buildCommand

Write-Host "Running AssetProcessorBatch..."
& $assetProcessorBatch --project-path $repoRoot --platforms=pc *>&1 | Tee-Object -FilePath $assetProcessorLog
$assetProcessorExitCode = $LASTEXITCODE
Write-Host "AssetProcessorBatch log: $assetProcessorLog"
if ($assetProcessorExitCode -ne 0) {
    throw "AssetProcessorBatch failed with exit code $assetProcessorExitCode. Log: $assetProcessorLog"
}

Write-Host "O3DE client build completed successfully."

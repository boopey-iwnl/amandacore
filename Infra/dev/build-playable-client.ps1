$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$launcherProject = Join-Path $repoRoot "Client\\Launcher\\AmandaCore.Launcher\\AmandaCore.Launcher.csproj"
$worldClientProject = Join-Path $repoRoot "Client\\Game\\AmandaCore.WorldClient\\AmandaCore.WorldClient.csproj"
$versionManifestScript = Join-Path $PSScriptRoot "write-version-manifest.ps1"
$versionManifestPath = Join-Path $PSScriptRoot "version-manifest.json"
$o3deBuildRoots = @(
    (Join-Path $repoRoot "build\\windows"),
    (Join-Path $repoRoot "build\\o3de-windows")
)

function Resolve-Tool($preferredPath, $commandName) {
    if (Test-Path $preferredPath) {
        return $preferredPath
    }

    $command = Get-Command $commandName -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    throw "Required tool '$commandName' was not found."
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

$dotnet = Resolve-Tool "C:\Program Files\dotnet\dotnet.exe" "dotnet"
$cmake = Resolve-Tool "C:\Program Files\CMake\bin\cmake.exe" "cmake"

if (!(Test-Path $versionManifestPath)) {
    & $versionManifestScript -OutputPath $versionManifestPath
}

Invoke-Native $dotnet "build" $launcherProject
Invoke-Native $dotnet "build" $worldClientProject

foreach ($o3deBuildRoot in $o3deBuildRoots) {
    if (Test-Path $o3deBuildRoot) {
        Invoke-Native $cmake "--build" $o3deBuildRoot "--config" "profile" "--target" "amandacore.GameLauncher" "--" "/m:1" "/p:CL_MPCount=1"
    }
}

Write-Host "Playable client build completed for launcher, world client, and available O3DE GameLauncher targets."
Write-Host "Build manifest: $versionManifestPath"

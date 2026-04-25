$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$launcherProject = Join-Path $repoRoot "Client\\Launcher\\AmandaCore.Launcher\\AmandaCore.Launcher.csproj"
$worldClientProject = Join-Path $repoRoot "Client\\Game\\AmandaCore.WorldClient\\AmandaCore.WorldClient.csproj"
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

$dotnet = Resolve-Tool "C:\Program Files\dotnet\dotnet.exe" "dotnet"
$cmake = Resolve-Tool "C:\Program Files\CMake\bin\cmake.exe" "cmake"

& $dotnet build $launcherProject
& $dotnet build $worldClientProject

foreach ($o3deBuildRoot in $o3deBuildRoots) {
    if (Test-Path $o3deBuildRoot) {
        & $cmake --build $o3deBuildRoot --config profile --target amandacore.GameLauncher
    }
}

Write-Host "Playable client build completed for launcher, world client, and available O3DE GameLauncher targets."

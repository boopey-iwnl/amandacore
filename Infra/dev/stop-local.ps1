$ErrorActionPreference = "Stop"

$processManifest = Join-Path $PSScriptRoot "local-processes.json"
if (Test-Path $processManifest) {
    $processes = Get-Content $processManifest | ConvertFrom-Json
    foreach ($process in $processes) {
        try {
            Stop-Process -Id $process.Id -Force -ErrorAction Stop
            Write-Host "Stopped $($process.Name) [$($process.Id)]"
        }
        catch {
            Write-Host "Process $($process.Name) [$($process.Id)] was already stopped."
        }
    }

    Remove-Item $processManifest -Force
}
else {
    Write-Host "No local process manifest found."
}

$localProcessNames = @(
    "auth-service",
    "account-service",
    "realm-service",
    "character-service",
    "world-service",
    "admin-service",
    "AmandaCore.Launcher",
    "amandacore.GameLauncher",
    "GameLauncher",
    "AssetProcessor",
    "AssetProcessorBatch"
)

foreach ($name in $localProcessNames) {
    Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction Stop
            Write-Host "Stopped lingering $name [$($_.Id)]"
        }
        catch {
            Write-Host "Lingering $name [$($_.Id)] was already stopped."
        }
    }
}

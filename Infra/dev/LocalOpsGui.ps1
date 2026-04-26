param()

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

[System.Windows.Forms.Application]::EnableVisualStyles()

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$startScript = Join-Path $PSScriptRoot "start-local.ps1"
$stopScript = Join-Path $PSScriptRoot "stop-local.ps1"
$buildClientScript = Join-Path $PSScriptRoot "build-playable-client.ps1"
$diagnosticsScript = Join-Path $repoRoot "Infra\qa\Collect-Diagnostics.ps1"
$resetStateScript = Join-Path $repoRoot "Infra\qa\Reset-LocalTestState.ps1"
$qaDocsRoot = Join-Path $repoRoot "Docs\QA"
$adminPortalPath = Join-Path $repoRoot "Client\Portal\admin-portal.html"
$diagnosticsOutputRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\diagnostics"
$processManifest = Join-Path $PSScriptRoot "local-processes.json"
$versionManifestPath = Join-Path $PSScriptRoot "version-manifest.json"
$serviceLogsRoot = Join-Path $PSScriptRoot "logs"
$userLogRoot = Join-Path $repoRoot "user\log"
$gameLogPath = Join-Path $userLogRoot "Game.log"
$launcherExe = Join-Path $repoRoot "Client\Launcher\AmandaCore.Launcher\bin\Debug\net8.0-windows\AmandaCore.Launcher.exe"
$desktopShortcut = Join-Path ([Environment]::GetFolderPath("Desktop")) "Local Playable Slice Controls.lnk"
$o3deWindowsExe = Join-Path $repoRoot "build\windows\bin\profile\amandacore.GameLauncher.exe"
$o3deAltExe = Join-Path $repoRoot "build\o3de-windows\bin\profile\amandacore.GameLauncher.exe"
$stateStore = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\platform-state.json"
$serviceNames = @(
    "auth-service",
    "account-service",
    "realm-service",
    "character-service",
    "world-service",
    "admin-service"
)
$serviceDefinitions = @(
    @{ Name = "auth-service"; Port = "8081" },
    @{ Name = "account-service"; Port = "8082" },
    @{ Name = "realm-service"; Port = "8083" },
    @{ Name = "character-service"; Port = "8084" },
    @{ Name = "world-service"; Port = "8085" },
    @{ Name = "admin-service"; Port = "8086" }
)

function Invoke-BackgroundPowerShell {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    Start-Process -FilePath "powershell.exe" `
        -ArgumentList @("-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", $Command) `
        -WindowStyle Hidden `
        -PassThru | Out-Null
}

function Invoke-WaitingPowerShell {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Command
    )

    $process = Start-Process -FilePath "powershell.exe" `
        -ArgumentList @("-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", $Command) `
        -WindowStyle Hidden `
        -PassThru `
        -Wait

    return $process.ExitCode
}

function Stop-LocalProcessesFallback {
    $namesToStop = @(
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

    $stopped = @()
    foreach ($name in $namesToStop) {
        $matching = @(Get-Process -Name $name -ErrorAction SilentlyContinue)
        foreach ($process in $matching) {
            try {
                Stop-Process -Id $process.Id -Force -ErrorAction Stop
                $stopped += "$name [$($process.Id)]"
            }
            catch {
            }
        }
    }

    if (Test-Path $processManifest) {
        Remove-Item $processManifest -Force -ErrorAction SilentlyContinue
    }

    return $stopped
}

function Get-StackSnapshot {
    $serviceStatuses = @(Get-ServiceStatuses)
    $runningServices = @($serviceStatuses | Where-Object { $_.ProcessRunning } | ForEach-Object { $_.Name })
    $healthyServices = @($serviceStatuses | Where-Object { $_.HealthStatus -eq "Healthy" } | ForEach-Object { $_.Name })
    $failedServices = @($serviceStatuses | Where-Object { $_.HealthStatus -eq "Failed" -or ($_.ProcessRunning -and $_.HealthStatus -eq "Unhealthy") })

    $launcherProcesses = @(Get-Process -Name "AmandaCore.Launcher" -ErrorAction SilentlyContinue)
    $gameProcesses = @(
        Get-Process -Name "amandacore.GameLauncher" -ErrorAction SilentlyContinue
        Get-Process -Name "GameLauncher" -ErrorAction SilentlyContinue
    ) | Where-Object { $_ }

    $manifestPresent = Test-Path $processManifest
    $stackStatus = if ($healthyServices.Count -eq $serviceNames.Count) {
        "Running"
    }
    elseif ($failedServices.Count -gt 0) {
        "Failed"
    }
    elseif ($runningServices.Count -gt 0 -or $manifestPresent) {
        "Partial"
    }
    else {
        "Stopped"
    }

    [pscustomobject]@{
        StackStatus     = $stackStatus
        RunningServices = $runningServices
        HealthyServices = $healthyServices
        ServiceStatuses = $serviceStatuses
        ManifestPresent = $manifestPresent
        LauncherRunning = $launcherProcesses.Count -gt 0
        LauncherCount   = $launcherProcesses.Count
        GameRunning     = $gameProcesses.Count -gt 0
        GameCount       = $gameProcesses.Count
    }
}

function Get-ServiceStatuses {
    foreach ($service in $serviceDefinitions) {
        $serviceName = [string]$service.Name
        $port = [string]$service.Port
        $processes = @(Get-Process -Name $serviceName -ErrorAction SilentlyContinue)
        $healthUrl = "http://127.0.0.1:$port/health"
        $healthStatus = if ($processes.Count -gt 0) { "Starting" } else { "Stopped" }
        $lastError = ""

        if ($processes.Count -gt 0) {
            try {
                $response = Invoke-WebRequest -Uri $healthUrl -UseBasicParsing -TimeoutSec 1
                if ($response.StatusCode -eq 200) {
                    $healthStatus = "Healthy"
                }
                else {
                    $healthStatus = "Unhealthy"
                    $lastError = "HTTP $($response.StatusCode)"
                }
            }
            catch {
                $healthStatus = "Unhealthy"
                $lastError = $_.Exception.Message
            }
        }

        [pscustomobject]@{
            Name           = $serviceName
            Port           = $port
            ProcessRunning = $processes.Count -gt 0
            ProcessIds     = @($processes | ForEach-Object { $_.Id })
            HealthStatus   = $healthStatus
            LastError      = $lastError
            LogPath        = Join-Path $serviceLogsRoot "$serviceName.log"
        }
    }
}

function Format-ServiceStatusSummary {
    param(
        [Parameter(Mandatory = $true)]
        [object[]]$ServiceStatuses
    )

    if ($ServiceStatuses.Count -eq 0) {
        return "No service status available."
    }

    return ($ServiceStatuses | ForEach-Object {
        $pidText = if ($_.ProcessIds.Count -gt 0) { " pid=" + ($_.ProcessIds -join ",") } else { "" }
        $errorText = if (![string]::IsNullOrWhiteSpace($_.LastError)) { " - " + $_.LastError } else { "" }
        "$($_.Name): $($_.HealthStatus) :$($_.Port)$pidText$errorText"
    }) -join [Environment]::NewLine
}

function Get-FailedServiceLogTail {
    param(
        [Parameter(Mandatory = $true)]
        [object[]]$ServiceStatuses
    )

    $failed = @($ServiceStatuses | Where-Object { $_.HealthStatus -eq "Failed" -or ($_.ProcessRunning -and $_.HealthStatus -eq "Unhealthy") })
    if ($failed.Count -eq 0) {
        return ""
    }

    $sections = @()
    foreach ($service in $failed) {
        if (!(Test-Path $service.LogPath)) {
            continue
        }

        $tail = Get-Content -Path $service.LogPath -Tail 12 -ErrorAction SilentlyContinue | Out-String
        if (![string]::IsNullOrWhiteSpace($tail)) {
            $sections += "$($service.Name) log tail:`n$tail"
        }
    }

    return ($sections -join "`n")
}

function Get-VersionManifestSummary {
    if (!(Test-Path $versionManifestPath)) {
        return "Not present"
    }

    try {
        $manifest = Get-Content -Path $versionManifestPath -Raw | ConvertFrom-Json
        $buildId = [string]$manifest.buildId
        $clientVersion = [string]$manifest.clientVersion
        $protocolVersion = [string]$manifest.protocolVersion
        if ([string]::IsNullOrWhiteSpace($buildId)) {
            return "Invalid manifest"
        }

        $shortBuildId = if ($buildId.Length -gt 52) {
            $buildId.Substring(0, 44) + "..." + $buildId.Substring($buildId.Length - 5)
        }
        else {
            $buildId
        }

        return "$shortBuildId | client $clientVersion | protocol $protocolVersion"
    }
    catch {
        return "Unreadable: $($_.Exception.Message)"
    }
}

function Get-LatestDiagnosticBundle {
    if (-not (Test-Path $diagnosticsOutputRoot)) {
        return ""
    }

    $latest = Get-ChildItem -Path $diagnosticsOutputRoot -Filter "amandacore-diagnostics-*.zip" -File -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -First 1
    if ($latest) {
        return $latest.FullName
    }

    return ""
}

function Get-RepoRevisionSummary {
    try {
        $branch = (& git -C $repoRoot branch --show-current 2>$null).Trim()
        $commit = (& git -C $repoRoot rev-parse --short HEAD 2>$null).Trim()
        $upstream = (& git -C $repoRoot rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>$null).Trim()
        $dirty = (& git -C $repoRoot status --porcelain 2>$null)
        $dirtyText = if ($dirty) { "dirty" } else { "clean" }
        if ([string]::IsNullOrWhiteSpace($branch) -or [string]::IsNullOrWhiteSpace($commit)) {
            return "Unavailable"
        }

        if ([string]::IsNullOrWhiteSpace($upstream)) {
            return "$branch @ $commit ($dirtyText)"
        }

        return "$branch @ $commit tracking $upstream ($dirtyText)"
    }
    catch {
        return "Unavailable: $($_.Exception.Message)"
    }
}

function Set-StatusMessage {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message,
        [System.Drawing.Color]$Color = [System.Drawing.Color]::FromArgb(44, 62, 80)
    )

    $statusLabel.ForeColor = $Color
    $statusLabel.Text = $Message
}

function Refresh-Status {
    $snapshot = Get-StackSnapshot

    $stackValueLabel.Text = $snapshot.StackStatus
    $stackValueLabel.ForeColor = switch ($snapshot.StackStatus) {
        "Running" { [System.Drawing.Color]::ForestGreen }
        "Partial" { [System.Drawing.Color]::DarkOrange }
        "Failed" { [System.Drawing.Color]::Firebrick }
        default { [System.Drawing.Color]::Firebrick }
    }

    $launcherValueLabel.Text = if ($snapshot.LauncherRunning) { "Running ($($snapshot.LauncherCount))" } else { "Stopped" }
    $launcherValueLabel.ForeColor = if ($snapshot.LauncherRunning) {
        [System.Drawing.Color]::ForestGreen
    }
    else {
        [System.Drawing.Color]::Firebrick
    }

    $gameValueLabel.Text = if ($snapshot.GameRunning) { "Running ($($snapshot.GameCount))" } else { "Stopped" }
    $gameValueLabel.ForeColor = if ($snapshot.GameRunning) {
        [System.Drawing.Color]::ForestGreen
    }
    else {
        [System.Drawing.Color]::Firebrick
    }

    $manifestValueLabel.Text = if ($snapshot.ManifestPresent) { $processManifest } else { "Not present" }
    $buildValueLabel.Text = Get-VersionManifestSummary
    $servicesValueLabel.Text = Format-ServiceStatusSummary -ServiceStatuses $snapshot.ServiceStatuses
    $failedLogTail = Get-FailedServiceLogTail -ServiceStatuses $snapshot.ServiceStatuses
    if ([string]::IsNullOrWhiteSpace($failedLogTail)) {
        $failureTextBox.Text = "No service failures detected."
    }
    else {
        $failureTextBox.Text = $failedLogTail
    }

    $pathBox.Lines = @(
        "Repo root: $repoRoot",
        "Repo revision: $(Get-RepoRevisionSummary)",
        "Service logs: $serviceLogsRoot",
        "Game client log: $gameLogPath",
        "User log folder: $userLogRoot",
        "Local state store: $stateStore",
        "Diagnostic bundles: $diagnosticsOutputRoot",
        "QA docs: $qaDocsRoot",
        "Admin portal: $adminPortalPath",
        "Process manifest: $processManifest",
        "Version manifest: $versionManifestPath",
        "Desktop shortcut: $desktopShortcut",
        "Launcher executable: $launcherExe",
        "O3DE client (windows): $o3deWindowsExe",
        "O3DE client (o3de-windows): $o3deAltExe"
    )
}

$form = New-Object System.Windows.Forms.Form
$form.Text = "amandacore Local Ops"
$form.StartPosition = "CenterScreen"
$form.Size = New-Object System.Drawing.Size(940, 760)
$form.MinimumSize = New-Object System.Drawing.Size(940, 760)
$form.MaximizeBox = $false

$titleLabel = New-Object System.Windows.Forms.Label
$titleLabel.Text = "Local Playable Slice Controls"
$titleLabel.Location = New-Object System.Drawing.Point(20, 18)
$titleLabel.Size = New-Object System.Drawing.Size(360, 24)
$titleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 12, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($titleLabel)

$subtitleLabel = New-Object System.Windows.Forms.Label
$subtitleLabel.Text = "Use Build + Launch Test for the human visual pass. Services bind to local loopback by default."
$subtitleLabel.Location = New-Object System.Drawing.Point(20, 46)
$subtitleLabel.Size = New-Object System.Drawing.Size(690, 32)
$subtitleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($subtitleLabel)

$startButton = New-Object System.Windows.Forms.Button
$startButton.Text = "Build + Launch Test"
$startButton.Location = New-Object System.Drawing.Point(20, 92)
$startButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($startButton)

$startServicesButton = New-Object System.Windows.Forms.Button
$startServicesButton.Text = "Start Services"
$startServicesButton.Location = New-Object System.Drawing.Point(192, 92)
$startServicesButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($startServicesButton)

$stopButton = New-Object System.Windows.Forms.Button
$stopButton.Text = "Stop Local Stack"
$stopButton.Location = New-Object System.Drawing.Point(336, 92)
$stopButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($stopButton)

$launcherButton = New-Object System.Windows.Forms.Button
$launcherButton.Text = "Open Launcher"
$launcherButton.Location = New-Object System.Drawing.Point(480, 92)
$launcherButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($launcherButton)

$logsButton = New-Object System.Windows.Forms.Button
$logsButton.Text = "Open Logs Folder"
$logsButton.Location = New-Object System.Drawing.Point(624, 92)
$logsButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($logsButton)

$refreshButton = New-Object System.Windows.Forms.Button
$refreshButton.Text = "Refresh Status"
$refreshButton.Location = New-Object System.Drawing.Point(768, 92)
$refreshButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($refreshButton)

$diagnosticsButton = New-Object System.Windows.Forms.Button
$diagnosticsButton.Text = "Collect Diagnostics"
$diagnosticsButton.Location = New-Object System.Drawing.Point(20, 134)
$diagnosticsButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($diagnosticsButton)

$qaDocsButton = New-Object System.Windows.Forms.Button
$qaDocsButton.Text = "Open QA Docs"
$qaDocsButton.Location = New-Object System.Drawing.Point(192, 134)
$qaDocsButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($qaDocsButton)

$resetStateButton = New-Object System.Windows.Forms.Button
$resetStateButton.Text = "Reset Test State"
$resetStateButton.Location = New-Object System.Drawing.Point(336, 134)
$resetStateButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($resetStateButton)

$adminPortalButton = New-Object System.Windows.Forms.Button
$adminPortalButton.Text = "Admin Panel"
$adminPortalButton.Location = New-Object System.Drawing.Point(480, 134)
$adminPortalButton.Size = New-Object System.Drawing.Size(132, 34)
$form.Controls.Add($adminPortalButton)

$statusLabel = New-Object System.Windows.Forms.Label
$statusLabel.Text = "Ready."
$statusLabel.Location = New-Object System.Drawing.Point(20, 180)
$statusLabel.Size = New-Object System.Drawing.Size(888, 28)
$statusLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($statusLabel)

$statusGroup = New-Object System.Windows.Forms.GroupBox
$statusGroup.Text = "Current Status"
$statusGroup.Location = New-Object System.Drawing.Point(20, 220)
$statusGroup.Size = New-Object System.Drawing.Size(888, 158)
$form.Controls.Add($statusGroup)

function Add-StatusRow {
    param(
        [string]$Label,
        [int]$Y
    )

    $labelControl = New-Object System.Windows.Forms.Label
    $labelControl.Text = $Label
    $labelControl.Location = New-Object System.Drawing.Point(16, $Y)
    $labelControl.Size = New-Object System.Drawing.Size(145, 22)
    $statusGroup.Controls.Add($labelControl)

    $valueControl = New-Object System.Windows.Forms.Label
    $valueControl.Location = New-Object System.Drawing.Point(168, $Y)
    $valueControl.Size = New-Object System.Drawing.Size(695, 22)
    $valueControl.Font = New-Object System.Drawing.Font("Segoe UI", 9, [System.Drawing.FontStyle]::Bold)
    $statusGroup.Controls.Add($valueControl)
    return $valueControl
}

$stackValueLabel = Add-StatusRow -Label "Stack" -Y 26
$launcherValueLabel = Add-StatusRow -Label "Launcher" -Y 50
$gameValueLabel = Add-StatusRow -Label "Game Client" -Y 74
$buildValueLabel = Add-StatusRow -Label "Build" -Y 98
$manifestValueLabel = Add-StatusRow -Label "Process Manifest" -Y 122

$servicesLabel = New-Object System.Windows.Forms.Label
$servicesLabel.Text = "Running Services"
$servicesLabel.Location = New-Object System.Drawing.Point(20, 392)
$servicesLabel.Size = New-Object System.Drawing.Size(140, 22)
$form.Controls.Add($servicesLabel)

$servicesValueLabel = New-Object System.Windows.Forms.Label
$servicesValueLabel.Location = New-Object System.Drawing.Point(162, 392)
$servicesValueLabel.Size = New-Object System.Drawing.Size(746, 118)
$servicesValueLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($servicesValueLabel)

$failureLabel = New-Object System.Windows.Forms.Label
$failureLabel.Text = "Failure Details"
$failureLabel.Location = New-Object System.Drawing.Point(20, 518)
$failureLabel.Size = New-Object System.Drawing.Size(140, 22)
$form.Controls.Add($failureLabel)

$failureTextBox = New-Object System.Windows.Forms.TextBox
$failureTextBox.Location = New-Object System.Drawing.Point(162, 518)
$failureTextBox.Size = New-Object System.Drawing.Size(746, 74)
$failureTextBox.Multiline = $true
$failureTextBox.ReadOnly = $true
$failureTextBox.ScrollBars = "Vertical"
$failureTextBox.Font = New-Object System.Drawing.Font("Consolas", 9)
$form.Controls.Add($failureTextBox)

$pathsLabel = New-Object System.Windows.Forms.Label
$pathsLabel.Text = "Main Paths"
$pathsLabel.Location = New-Object System.Drawing.Point(20, 600)
$pathsLabel.Size = New-Object System.Drawing.Size(120, 22)
$form.Controls.Add($pathsLabel)

$pathBox = New-Object System.Windows.Forms.TextBox
$pathBox.Location = New-Object System.Drawing.Point(20, 626)
$pathBox.Size = New-Object System.Drawing.Size(888, 74)
$pathBox.Multiline = $true
$pathBox.ReadOnly = $true
$pathBox.ScrollBars = "Vertical"
$pathBox.Font = New-Object System.Drawing.Font("Consolas", 9)
$form.Controls.Add($pathBox)

$startButton.Add_Click({
    Set-StatusMessage -Message "Stopping old processes, building latest binaries, starting services, and opening launcher..." -Color ([System.Drawing.Color]::DodgerBlue)
    $command = "& '$startScript' -BuildFirst -StartLauncher"
    Invoke-BackgroundPowerShell -Command $command
    Start-Sleep -Milliseconds 500
    Refresh-Status
})

$startServicesButton.Add_Click({
    Set-StatusMessage -Message "Starting local services without rebuilding..." -Color ([System.Drawing.Color]::DodgerBlue)
    $command = "& '$startScript' -BuildFirst:`$false"
    Invoke-BackgroundPowerShell -Command $command
    Start-Sleep -Milliseconds 500
    Refresh-Status
})

$stopButton.Add_Click({
    $snapshot = Get-StackSnapshot
    $managedProcessesRunning = $snapshot.StackStatus -ne "Stopped" -or $snapshot.LauncherRunning -or $snapshot.GameRunning
    if (-not $managedProcessesRunning) {
        Set-StatusMessage -Message "Local stack and client are already stopped." -Color ([System.Drawing.Color]::DarkOrange)
        Refresh-Status
        return
    }

    Set-StatusMessage -Message "Stopping local stack..." -Color ([System.Drawing.Color]::DodgerBlue)
    $exitCode = 0
    if ($snapshot.StackStatus -ne "Stopped" -or $snapshot.ManifestPresent) {
        $command = "& '$stopScript'"
        $exitCode = Invoke-WaitingPowerShell -Command $command
    }
    $stoppedFallback = Stop-LocalProcessesFallback
    Start-Sleep -Milliseconds 250
    Refresh-Status

    $postSnapshot = Get-StackSnapshot
    $everythingStopped = $postSnapshot.StackStatus -eq "Stopped" -and -not $postSnapshot.LauncherRunning -and -not $postSnapshot.GameRunning
    if ($everythingStopped) {
        if ($stoppedFallback.Count -gt 0) {
            Set-StatusMessage -Message "Local stack and client stopped. Fallback closed: $($stoppedFallback -join ', ')" -Color ([System.Drawing.Color]::ForestGreen)
        }
        else {
            Set-StatusMessage -Message "Local stack and client stopped." -Color ([System.Drawing.Color]::ForestGreen)
        }
        return
    }

    $details = if ($stoppedFallback.Count -gt 0) {
        " Fallback closed: $($stoppedFallback -join ', ')"
    }
    else {
        ""
    }
    Set-StatusMessage -Message "Stop command finished with exit code $exitCode, but stack is still not fully stopped.$details" -Color ([System.Drawing.Color]::Firebrick)
})

$launcherButton.Add_Click({
    $snapshot = Get-StackSnapshot
    if ($snapshot.LauncherRunning) {
        Set-StatusMessage -Message "Launcher is already open." -Color ([System.Drawing.Color]::DarkOrange)
        Refresh-Status
        return
    }

    Set-StatusMessage -Message "Building latest launcher and playable client..." -Color ([System.Drawing.Color]::DodgerBlue)
    $buildExitCode = Invoke-WaitingPowerShell -Command "& '$buildClientScript'"
    if ($buildExitCode -ne 0) {
        Set-StatusMessage -Message "Launcher build failed with exit code $buildExitCode." -Color ([System.Drawing.Color]::Firebrick)
        Refresh-Status
        return
    }

    if (-not (Test-Path $launcherExe)) {
        Set-StatusMessage -Message "Launcher executable not found at $launcherExe" -Color ([System.Drawing.Color]::Firebrick)
        Refresh-Status
        return
    }

    Start-Process -FilePath $launcherExe | Out-Null
    Set-StatusMessage -Message "Launcher opened." -Color ([System.Drawing.Color]::ForestGreen)
    Start-Sleep -Milliseconds 300
    Refresh-Status
})

$logsButton.Add_Click({
    if (Test-Path $serviceLogsRoot) {
        Start-Process -FilePath "explorer.exe" -ArgumentList $serviceLogsRoot | Out-Null
    }

    if (Test-Path $userLogRoot) {
        Start-Process -FilePath "explorer.exe" -ArgumentList $userLogRoot | Out-Null
    }

    Set-StatusMessage -Message "Opened the service logs and user log folders." -Color ([System.Drawing.Color]::ForestGreen)
    Refresh-Status
})

$diagnosticsButton.Add_Click({
    if (-not (Test-Path $diagnosticsScript)) {
        Set-StatusMessage -Message "Diagnostic script not found at $diagnosticsScript" -Color ([System.Drawing.Color]::Firebrick)
        return
    }

    Set-StatusMessage -Message "Collecting diagnostic bundle..." -Color ([System.Drawing.Color]::DodgerBlue)
    $exitCode = Invoke-WaitingPowerShell -Command "& '$diagnosticsScript'"
    $bundlePath = Get-LatestDiagnosticBundle
    if ($exitCode -eq 0 -and -not [string]::IsNullOrWhiteSpace($bundlePath)) {
        Start-Process -FilePath "explorer.exe" -ArgumentList "/select,`"$bundlePath`"" | Out-Null
        Set-StatusMessage -Message "Diagnostic bundle created: $bundlePath" -Color ([System.Drawing.Color]::ForestGreen)
    }
    elseif ($exitCode -eq 0) {
        Set-StatusMessage -Message "Diagnostics completed, but no bundle zip was found under $diagnosticsOutputRoot." -Color ([System.Drawing.Color]::DarkOrange)
    }
    else {
        Set-StatusMessage -Message "Diagnostics failed with exit code $exitCode." -Color ([System.Drawing.Color]::Firebrick)
    }
    Refresh-Status
})

$qaDocsButton.Add_Click({
    if (Test-Path $qaDocsRoot) {
        Start-Process -FilePath "explorer.exe" -ArgumentList $qaDocsRoot | Out-Null
        Set-StatusMessage -Message "Opened QA docs." -Color ([System.Drawing.Color]::ForestGreen)
    }
    else {
        Set-StatusMessage -Message "QA docs folder not found at $qaDocsRoot" -Color ([System.Drawing.Color]::Firebrick)
    }
})

$resetStateButton.Add_Click({
    $answer = [System.Windows.Forms.MessageBox]::Show(
        "This backs up and resets all local account/character test state. Logs and diagnostic bundles are preserved. Stop the local stack first. Continue?",
        "Reset Local Test State",
        [System.Windows.Forms.MessageBoxButtons]::YesNo,
        [System.Windows.Forms.MessageBoxIcon]::Warning)
    if ($answer -ne [System.Windows.Forms.DialogResult]::Yes) {
        Set-StatusMessage -Message "Reset canceled." -Color ([System.Drawing.Color]::DarkOrange)
        return
    }

    Set-StatusMessage -Message "Resetting local test state..." -Color ([System.Drawing.Color]::DodgerBlue)
    $exitCode = Invoke-WaitingPowerShell -Command "& '$resetStateScript' -All -ConfirmReset"
    if ($exitCode -eq 0) {
        Set-StatusMessage -Message "Local test state reset. Logs were preserved." -Color ([System.Drawing.Color]::ForestGreen)
    }
    else {
        Set-StatusMessage -Message "Reset failed with exit code $exitCode. Stop services first or use the script manually with -Force." -Color ([System.Drawing.Color]::Firebrick)
    }
    Refresh-Status
})

$adminPortalButton.Add_Click({
    if (Test-Path $adminPortalPath) {
        Start-Process -FilePath $adminPortalPath | Out-Null
        Set-StatusMessage -Message "Opened local admin panel." -Color ([System.Drawing.Color]::ForestGreen)
    }
    else {
        Set-StatusMessage -Message "Admin portal not found at $adminPortalPath" -Color ([System.Drawing.Color]::Firebrick)
    }
})

$refreshButton.Add_Click({
    Refresh-Status
    Set-StatusMessage -Message "Status refreshed." -Color ([System.Drawing.Color]::FromArgb(44, 62, 80))
})

$refreshTimer = New-Object System.Windows.Forms.Timer
$refreshTimer.Interval = 2000
$refreshTimer.Add_Tick({ Refresh-Status })
$refreshTimer.Start()

$form.Add_Shown({
    Refresh-Status
})

[void]$form.ShowDialog()

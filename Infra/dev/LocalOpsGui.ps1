param()

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

[System.Windows.Forms.Application]::EnableVisualStyles()

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$startScript = Join-Path $PSScriptRoot "start-local.ps1"
$stopScript = Join-Path $PSScriptRoot "stop-local.ps1"
$buildClientScript = Join-Path $PSScriptRoot "build-playable-client.ps1"
$processManifest = Join-Path $PSScriptRoot "local-processes.json"
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
    $runningServices = @()
    foreach ($serviceName in $serviceNames) {
        $serviceProcesses = @(Get-Process -Name $serviceName -ErrorAction SilentlyContinue)
        if ($serviceProcesses.Count -gt 0) {
            $runningServices += $serviceName
        }
    }

    $launcherProcesses = @(Get-Process -Name "AmandaCore.Launcher" -ErrorAction SilentlyContinue)
    $gameProcesses = @(
        Get-Process -Name "amandacore.GameLauncher" -ErrorAction SilentlyContinue
        Get-Process -Name "GameLauncher" -ErrorAction SilentlyContinue
    ) | Where-Object { $_ }

    $manifestPresent = Test-Path $processManifest
    $stackStatus = if ($runningServices.Count -eq $serviceNames.Count) {
        "Running"
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
        ManifestPresent = $manifestPresent
        LauncherRunning = $launcherProcesses.Count -gt 0
        LauncherCount   = $launcherProcesses.Count
        GameRunning     = $gameProcesses.Count -gt 0
        GameCount       = $gameProcesses.Count
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
    $servicesValueLabel.Text = if ($snapshot.RunningServices.Count -gt 0) {
        $snapshot.RunningServices -join ", "
    }
    else {
        "None"
    }

    $pathBox.Lines = @(
        "Service logs: $serviceLogsRoot",
        "Game client log: $gameLogPath",
        "User log folder: $userLogRoot",
        "Local state store: $stateStore",
        "Process manifest: $processManifest",
        "Desktop shortcut: $desktopShortcut",
        "Launcher executable: $launcherExe",
        "O3DE client (windows): $o3deWindowsExe",
        "O3DE client (o3de-windows): $o3deAltExe"
    )
}

$form = New-Object System.Windows.Forms.Form
$form.Text = "amandacore Local Ops"
$form.StartPosition = "CenterScreen"
$form.Size = New-Object System.Drawing.Size(760, 510)
$form.MinimumSize = New-Object System.Drawing.Size(760, 510)
$form.MaximizeBox = $false

$titleLabel = New-Object System.Windows.Forms.Label
$titleLabel.Text = "Local Playable Slice Controls"
$titleLabel.Location = New-Object System.Drawing.Point(20, 18)
$titleLabel.Size = New-Object System.Drawing.Size(360, 24)
$titleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 12, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($titleLabel)

$subtitleLabel = New-Object System.Windows.Forms.Label
$subtitleLabel.Text = "Wraps the existing Infra/dev scripts for starting services, stopping services, and opening the launcher."
$subtitleLabel.Location = New-Object System.Drawing.Point(20, 46)
$subtitleLabel.Size = New-Object System.Drawing.Size(690, 32)
$subtitleLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($subtitleLabel)

$startButton = New-Object System.Windows.Forms.Button
$startButton.Text = "Build + Restart Stack"
$startButton.Location = New-Object System.Drawing.Point(20, 92)
$startButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($startButton)

$stopButton = New-Object System.Windows.Forms.Button
$stopButton.Text = "Stop Local Stack"
$stopButton.Location = New-Object System.Drawing.Point(192, 92)
$stopButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($stopButton)

$launcherButton = New-Object System.Windows.Forms.Button
$launcherButton.Text = "Open Launcher"
$launcherButton.Location = New-Object System.Drawing.Point(364, 92)
$launcherButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($launcherButton)

$logsButton = New-Object System.Windows.Forms.Button
$logsButton.Text = "Open Logs Folder"
$logsButton.Location = New-Object System.Drawing.Point(536, 92)
$logsButton.Size = New-Object System.Drawing.Size(160, 34)
$form.Controls.Add($logsButton)

$refreshButton = New-Object System.Windows.Forms.Button
$refreshButton.Text = "Refresh Status"
$refreshButton.Location = New-Object System.Drawing.Point(536, 134)
$refreshButton.Size = New-Object System.Drawing.Size(160, 30)
$form.Controls.Add($refreshButton)

$statusLabel = New-Object System.Windows.Forms.Label
$statusLabel.Text = "Ready."
$statusLabel.Location = New-Object System.Drawing.Point(20, 138)
$statusLabel.Size = New-Object System.Drawing.Size(500, 28)
$statusLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9, [System.Drawing.FontStyle]::Bold)
$form.Controls.Add($statusLabel)

$statusGroup = New-Object System.Windows.Forms.GroupBox
$statusGroup.Text = "Current Status"
$statusGroup.Location = New-Object System.Drawing.Point(20, 180)
$statusGroup.Size = New-Object System.Drawing.Size(696, 132)
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
    $valueControl.Size = New-Object System.Drawing.Size(505, 22)
    $valueControl.Font = New-Object System.Drawing.Font("Segoe UI", 9, [System.Drawing.FontStyle]::Bold)
    $statusGroup.Controls.Add($valueControl)
    return $valueControl
}

$stackValueLabel = Add-StatusRow -Label "Stack" -Y 26
$launcherValueLabel = Add-StatusRow -Label "Launcher" -Y 50
$gameValueLabel = Add-StatusRow -Label "Game Client" -Y 74
$manifestValueLabel = Add-StatusRow -Label "Process Manifest" -Y 98

$servicesLabel = New-Object System.Windows.Forms.Label
$servicesLabel.Text = "Running Services"
$servicesLabel.Location = New-Object System.Drawing.Point(20, 326)
$servicesLabel.Size = New-Object System.Drawing.Size(140, 22)
$form.Controls.Add($servicesLabel)

$servicesValueLabel = New-Object System.Windows.Forms.Label
$servicesValueLabel.Location = New-Object System.Drawing.Point(162, 326)
$servicesValueLabel.Size = New-Object System.Drawing.Size(554, 36)
$servicesValueLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9)
$form.Controls.Add($servicesValueLabel)

$pathsLabel = New-Object System.Windows.Forms.Label
$pathsLabel.Text = "Main Paths"
$pathsLabel.Location = New-Object System.Drawing.Point(20, 364)
$pathsLabel.Size = New-Object System.Drawing.Size(120, 22)
$form.Controls.Add($pathsLabel)

$pathBox = New-Object System.Windows.Forms.TextBox
$pathBox.Location = New-Object System.Drawing.Point(20, 390)
$pathBox.Size = New-Object System.Drawing.Size(696, 68)
$pathBox.Multiline = $true
$pathBox.ReadOnly = $true
$pathBox.ScrollBars = "Vertical"
$pathBox.Font = New-Object System.Drawing.Font("Consolas", 9)
$form.Controls.Add($pathBox)

$startButton.Add_Click({
    Set-StatusMessage -Message "Stopping old processes, building latest binaries, and starting local stack..." -Color ([System.Drawing.Color]::DodgerBlue)
    $command = "& '$startScript' -BuildFirst"
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

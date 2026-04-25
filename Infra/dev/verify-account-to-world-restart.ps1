$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$worldClientExe = Join-Path $repoRoot "Client\\Game\\AmandaCore.WorldClient\\bin\\Debug\\net8.0\\AmandaCore.WorldClient.exe"
$startLocalScript = Join-Path $PSScriptRoot "start-local.ps1"
$stopLocalScript = Join-Path $PSScriptRoot "stop-local.ps1"

function Invoke-JsonPost($url, $body, $token) {
    $headers = @{ "Content-Type" = "application/json" }
    if ($token) {
        $headers["Authorization"] = "Bearer $token"
    }

    return Invoke-RestMethod -Uri $url -Method Post -Headers $headers -Body ($body | ConvertTo-Json)
}

function Invoke-JsonGet($url, $token) {
    $headers = @{}
    if ($token) {
        $headers["Authorization"] = "Bearer $token"
    }

    return Invoke-RestMethod -Uri $url -Method Get -Headers $headers
}

if (!(Test-Path $worldClientExe)) {
    throw "World client executable not found at $worldClientExe"
}

$username = "restart_" + [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$password = "restart_" + [Guid]::NewGuid().ToString("N")

Invoke-JsonPost "http://localhost:8081/v1/accounts/register" @{ username = $username; password = $password } $null | Out-Null
$login = Invoke-JsonPost "http://localhost:8081/v1/auth/login" @{ username = $username; password = $password } $null
$token = $login.accessToken

$realms = Invoke-JsonGet "http://localhost:8083/v1/realms" $null
$realmId = $realms.realms[0].id

$character = Invoke-JsonPost "http://localhost:8084/v1/characters" @{ realmId = $realmId; displayName = "Restart$($username.Substring($username.Length - 4))"; archetypeId = "wayfarer_warden" } $token
$ticket = Invoke-JsonPost "http://localhost:8085/v1/world/join-ticket" @{ realmId = $realmId; characterId = $character.id } $token

$process = Start-Process -FilePath $worldClientExe -ArgumentList "--join-ticket", $ticket.ticketId, "--world-endpoint", "http://localhost:8085", "--auto-demo" -PassThru -Wait
if ($process.ExitCode -ne 0) {
    throw "World client exited with code $($process.ExitCode)"
}

& $stopLocalScript | Out-Null
& $startLocalScript -BuildFirst:$false | Out-Null

$loginAfterRestart = Invoke-JsonPost "http://localhost:8081/v1/auth/login" @{ username = $username; password = $password } $null
$tokenAfterRestart = $loginAfterRestart.accessToken

$charactersAfterRestart = Invoke-JsonGet ("http://localhost:8084/v1/characters?realmId=" + $realmId) $tokenAfterRestart
$restoredCharacter = $charactersAfterRestart.characters | Select-Object -First 1
$ticketAfterRestart = Invoke-JsonPost "http://localhost:8085/v1/world/join-ticket" @{ realmId = $realmId; characterId = $restoredCharacter.id } $tokenAfterRestart
$connectAfterRestart = Invoke-JsonPost "http://localhost:8085/v1/world/connect" @{ ticketId = $ticketAfterRestart.ticketId } $null

Write-Host "Restart persistence verified."
Write-Host "User: $username"
Write-Host "Character: $($restoredCharacter.displayName)"
Write-Host ("Restored position: ({0}, {1}, {2})" -f $connectAfterRestart.position.x, $connectAfterRestart.position.y, $connectAfterRestart.position.z)

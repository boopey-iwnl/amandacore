$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$worldClientExe = Join-Path $repoRoot "Client\\Game\\AmandaCore.WorldClient\\bin\\Debug\\net8.0\\AmandaCore.WorldClient.exe"

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

$username = "proof_" + [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$password = "proof_" + [Guid]::NewGuid().ToString("N")

Invoke-JsonPost "http://127.0.0.1:8081/v1/accounts/register" @{ username = $username; password = $password } $null | Out-Null
$login = Invoke-JsonPost "http://127.0.0.1:8081/v1/auth/login" @{ username = $username; password = $password } $null
$token = $login.accessToken

$realms = Invoke-JsonGet "http://127.0.0.1:8083/v1/realms" $null
$realmId = $realms.realms[0].id

$character = Invoke-JsonPost "http://127.0.0.1:8084/v1/characters" @{ realmId = $realmId; displayName = "Runner$($username.Substring($username.Length - 4))"; archetypeId = "wayfarer_warden" } $token
$ticket = Invoke-JsonPost "http://127.0.0.1:8085/v1/world/join-ticket" @{ realmId = $realmId; characterId = $character.id } $token

if (!(Test-Path $worldClientExe)) {
    throw "World client executable not found at $worldClientExe"
}

$process = Start-Process -FilePath $worldClientExe -ArgumentList "--join-ticket", $ticket.ticketId, "--world-endpoint", "http://127.0.0.1:8085", "--auto-demo" -PassThru -Wait
if ($process.ExitCode -ne 0) {
    throw "World client exited with code $($process.ExitCode)"
}

$loginAfterClient = Invoke-JsonPost "http://127.0.0.1:8081/v1/auth/login" @{ username = $username; password = $password } $null
$tokenAfterClient = $loginAfterClient.accessToken
$ticketAfterClient = Invoke-JsonPost "http://127.0.0.1:8085/v1/world/join-ticket" @{ realmId = $realmId; characterId = $character.id } $tokenAfterClient
$connectAfterClient = Invoke-JsonPost "http://127.0.0.1:8085/v1/world/connect" @{ ticketId = $ticketAfterClient.ticketId } $null
$state = Invoke-JsonGet ("http://127.0.0.1:8085/v1/world/state?worldSessionToken=" + $connectAfterClient.worldSessionToken) $null

Write-Host "Golden path verified."
Write-Host "User: $username"
Write-Host "Realm: $realmId"
Write-Host "Character: $($character.displayName)"
Write-Host ("Position after reconnect: ({0}, {1}, {2})" -f $state.position.x, $state.position.y, $state.position.z)

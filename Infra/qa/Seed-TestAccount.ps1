param(
    [string]$Username = "alpha_tester",
    [string]$Password = "AlphaTest!123",
    [string]$CharacterName = "Alphaone",
    [string]$AuthBaseUrl = "http://127.0.0.1:8081",
    [string]$RealmBaseUrl = "http://127.0.0.1:8083",
    [string]$CharacterBaseUrl = "http://127.0.0.1:8084",
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Method,
        [Parameter(Mandatory = $true)]
        [string]$Uri,
        [AllowNull()]
        [object]$Body,
        [string]$BearerToken = ""
    )

    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($BearerToken)) {
        $headers["Authorization"] = "Bearer $BearerToken"
    }

    $parameters = @{
        Method = $Method
        Uri = $Uri
        Headers = $headers
        TimeoutSec = 10
    }

    if ($null -ne $Body) {
        $parameters["ContentType"] = "application/json"
        $parameters["Body"] = ($Body | ConvertTo-Json -Depth 8)
    }

    return Invoke-RestMethod @parameters
}

function Test-ServiceHealth {
    param([string]$BaseUrl)

    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl/health" -UseBasicParsing -TimeoutSec 3
        return $response.StatusCode -eq 200
    }
    catch {
        return $false
    }
}

if ($SelfTest) {
    foreach ($required in @("Invoke-JsonRequest", "Test-ServiceHealth")) {
        if (-not (Get-Command $required -ErrorAction SilentlyContinue)) {
            throw "Missing helper: $required"
        }
    }

    Write-Host "Seed test account script self-test passed."
    return
}

if ([string]::IsNullOrWhiteSpace($Username) -or [string]::IsNullOrWhiteSpace($Password) -or [string]::IsNullOrWhiteSpace($CharacterName)) {
    throw "Username, Password, and CharacterName are required."
}

foreach ($service in @(
    @{ Name = "auth-service"; BaseUrl = $AuthBaseUrl },
    @{ Name = "realm-service"; BaseUrl = $RealmBaseUrl },
    @{ Name = "character-service"; BaseUrl = $CharacterBaseUrl }
)) {
    if (-not (Test-ServiceHealth $service.BaseUrl)) {
        throw "$($service.Name) is not healthy at $($service.BaseUrl). Start the local stack first."
    }
}

$registered = $false
try {
    Invoke-JsonRequest -Method "Post" -Uri "$AuthBaseUrl/v1/accounts/register" -Body @{
        username = $Username
        password = $Password
    } | Out-Null
    $registered = $true
}
catch {
    $message = $_.Exception.Message
    if ($message -notmatch "(?i)(409|already exists|account already exists)") {
        throw
    }
}

$login = Invoke-JsonRequest -Method "Post" -Uri "$AuthBaseUrl/v1/auth/login" -Body @{
    username = $Username
    password = $Password
}

$realms = Invoke-JsonRequest -Method "Get" -Uri "$RealmBaseUrl/v1/realms" -Body $null
if ($null -eq $realms.realms -or @($realms.realms).Count -eq 0) {
    throw "No realms are available from $RealmBaseUrl."
}

$realm = @($realms.realms)[0]
$characters = Invoke-JsonRequest -Method "Get" -Uri "$CharacterBaseUrl/v1/characters?realmId=$([uri]::EscapeDataString($realm.id))" -Body $null -BearerToken $login.accessToken
$existingCharacter = @($characters.characters) | Where-Object { $_.displayName -ieq $CharacterName } | Select-Object -First 1
$created = $false
if ($null -eq $existingCharacter) {
    $existingCharacter = Invoke-JsonRequest -Method "Post" -Uri "$CharacterBaseUrl/v1/characters" -Body @{
        realmId = $realm.id
        displayName = $CharacterName
        raceId = "human"
        classId = "warrior"
        archetypeId = "wayfarer_warden"
    } -BearerToken $login.accessToken
    $created = $true
}

[pscustomobject]@{
    username = $Username
    registered = $registered
    realmId = $realm.id
    realmName = $realm.displayName
    characterId = $existingCharacter.id
    characterName = $existingCharacter.displayName
    characterCreated = $created
    note = "Password is not written to disk by this script. Share test credentials through the approved tester channel."
} | ConvertTo-Json -Depth 6

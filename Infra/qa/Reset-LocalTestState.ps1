param(
    [string]$StatePath = "",
    [string]$BackupRoot = "",
    [string]$AccountUsername = "",
    [string]$CharacterName = "",
    [switch]$All,
    [switch]$ConfirmReset,
    [switch]$Force,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($StatePath)) {
    $StatePath = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\platform-state.json"
}

if ([string]::IsNullOrWhiteSpace($BackupRoot)) {
    $BackupRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "amandacore\state-backups"
}

function Get-RunningLocalServiceNames {
    $names = @("auth-service", "account-service", "realm-service", "character-service", "world-service", "admin-service")
    return @($names | Where-Object { @(Get-Process -Name $_ -ErrorAction SilentlyContinue).Count -gt 0 })
}

function Remove-JsonProperty {
    param(
        [Parameter(Mandatory = $true)]
        [psobject]$Object,
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    $Object.PSObject.Properties.Remove($Name)
}

function Get-JsonMapProperties {
    param([AllowNull()][object]$Map)

    if ($null -eq $Map) {
        return @()
    }

    return @($Map.PSObject.Properties)
}

function Reset-StateFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$TargetPath,
        [Parameter(Mandatory = $true)]
        [string]$TargetBackupRoot
    )

    if (-not (Test-Path $TargetPath)) {
        return [pscustomobject]@{
            statePath = $TargetPath
            statePresent = $false
            changed = $false
            backupPath = ""
            removedAccounts = @()
            removedCharacters = @()
            removedSessions = 0
            removedWorldJoinTickets = 0
            mode = "missing"
        }
    }

    New-Item -ItemType Directory -Force -Path $TargetBackupRoot | Out-Null
    $timestamp = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
    $backupPath = Join-Path $TargetBackupRoot ("platform-state-$timestamp.json")
    Copy-Item -Path $TargetPath -Destination $backupPath -Force

    if ($All) {
        Remove-Item -Path $TargetPath -Force
        $lockPath = "$TargetPath.lock"
        if (Test-Path $lockPath) {
            Remove-Item -Path $lockPath -Force
        }

        return [pscustomobject]@{
            statePath = $TargetPath
            statePresent = $true
            changed = $true
            backupPath = $backupPath
            removedAccounts = @("*")
            removedCharacters = @("*")
            removedSessions = -1
            removedWorldJoinTickets = -1
            mode = "all"
        }
    }

    $state = Get-Content -Path $TargetPath -Raw | ConvertFrom-Json
    $removedAccountIds = @()
    $removedCharacterIds = @()

    if (-not [string]::IsNullOrWhiteSpace($AccountUsername)) {
        foreach ($accountProperty in Get-JsonMapProperties $state.accounts) {
            $account = $accountProperty.Value
            if ([string]$account.username -ieq $AccountUsername) {
                $removedAccountIds += $account.id
                Remove-JsonProperty -Object $state.accounts -Name $accountProperty.Name
            }
        }
    }

    foreach ($characterProperty in Get-JsonMapProperties $state.characters) {
        $character = $characterProperty.Value
        $matchesAccount = $removedAccountIds -contains [string]$character.accountId
        $matchesCharacter = -not [string]::IsNullOrWhiteSpace($CharacterName) -and ([string]$character.displayName -ieq $CharacterName)
        if ($matchesAccount -or $matchesCharacter) {
            $removedCharacterIds += $character.id
            Remove-JsonProperty -Object $state.characters -Name $characterProperty.Name
        }
    }

    $removedSessions = 0
    foreach ($sessionProperty in Get-JsonMapProperties $state.sessions) {
        if ($removedAccountIds -contains [string]$sessionProperty.Value.accountId) {
            Remove-JsonProperty -Object $state.sessions -Name $sessionProperty.Name
            $removedSessions++
        }
    }

    $removedTickets = 0
    foreach ($ticketProperty in Get-JsonMapProperties $state.worldJoinTickets) {
        $ticket = $ticketProperty.Value
        if (($removedAccountIds -contains [string]$ticket.accountId) -or ($removedCharacterIds -contains [string]$ticket.characterId)) {
            Remove-JsonProperty -Object $state.worldJoinTickets -Name $ticketProperty.Name
            $removedTickets++
        }
    }

    foreach ($resetProperty in Get-JsonMapProperties $state.passwordReset) {
        if ($removedAccountIds -contains [string]$resetProperty.Value.accountId) {
            Remove-JsonProperty -Object $state.passwordReset -Name $resetProperty.Name
        }
    }

    foreach ($friendProperty in Get-JsonMapProperties $state.friends) {
        $friend = $friendProperty.Value
        if (($removedCharacterIds -contains [string]$friend.ownerCharacterId) -or ($removedCharacterIds -contains [string]$friend.friendCharacterId)) {
            Remove-JsonProperty -Object $state.friends -Name $friendProperty.Name
        }
    }

    foreach ($partyProperty in Get-JsonMapProperties $state.parties) {
        $party = $partyProperty.Value
        $members = @($party.memberCharacterIds)
        if (($removedCharacterIds -contains [string]$party.leaderCharacterId) -or (@($members | Where-Object { $removedCharacterIds -contains [string]$_ }).Count -gt 0)) {
            Remove-JsonProperty -Object $state.parties -Name $partyProperty.Name
        }
    }

    $changed = $removedAccountIds.Count -gt 0 -or $removedCharacterIds.Count -gt 0 -or $removedSessions -gt 0 -or $removedTickets -gt 0
    if ($changed) {
        $state | ConvertTo-Json -Depth 30 | Set-Content -Path $TargetPath -Encoding UTF8
    }

    return [pscustomobject]@{
        statePath = $TargetPath
        statePresent = $true
        changed = $changed
        backupPath = $backupPath
        removedAccounts = $removedAccountIds
        removedCharacters = $removedCharacterIds
        removedSessions = $removedSessions
        removedWorldJoinTickets = $removedTickets
        mode = "selective"
    }
}

if ($SelfTest) {
    $tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("amandacore-reset-selftest-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null
    $samplePath = Join-Path $tempRoot "platform-state.json"
    @'
{
  "accounts": {
    "acct_keep": { "id": "acct_keep", "username": "keeper" },
    "acct_drop": { "id": "acct_drop", "username": "alpha_tester" }
  },
  "realms": {},
  "characters": {
    "char_keep": { "id": "char_keep", "accountId": "acct_keep", "displayName": "Keeper" },
    "char_drop": { "id": "char_drop", "accountId": "acct_drop", "displayName": "Alphaone" }
  },
  "sessions": { "sess_drop": { "id": "sess_drop", "accountId": "acct_drop" } },
  "worldJoinTickets": { "ticket_drop": { "ticketId": "ticket_drop", "accountId": "acct_drop", "characterId": "char_drop" } },
  "passwordReset": {},
  "friends": {},
  "parties": {},
  "buildManifest": {}
}
'@ | Set-Content -Path $samplePath -Encoding UTF8
    $script:AccountUsername = "alpha_tester"
    $result = Reset-StateFile -TargetPath $samplePath -TargetBackupRoot (Join-Path $tempRoot "backups")
    $state = Get-Content -Path $samplePath -Raw | ConvertFrom-Json
    if ($null -ne $state.accounts.acct_drop -or $null -eq $state.accounts.acct_keep) {
        throw "Reset self-test failed account filtering."
    }
    if (-not $result.changed -or $result.removedAccounts.Count -ne 1 -or -not (Test-Path $result.backupPath)) {
        throw "Reset self-test failed result summary."
    }
    Remove-Item -Path $tempRoot -Recurse -Force
    Write-Host "Reset local test state script self-test passed."
    return
}

if (-not $ConfirmReset) {
    throw "Refusing to reset local test state without -ConfirmReset. Use -All for full local state reset or provide -AccountUsername/-CharacterName for a selective reset."
}

if (-not $All -and [string]::IsNullOrWhiteSpace($AccountUsername) -and [string]::IsNullOrWhiteSpace($CharacterName)) {
    throw "Specify -All, -AccountUsername, or -CharacterName."
}

$runningServices = Get-RunningLocalServiceNames
if ($runningServices.Count -gt 0 -and -not $Force) {
    throw "Stop local services before resetting state, or rerun with -Force. Running: $($runningServices -join ', ')"
}

Reset-StateFile -TargetPath $StatePath -TargetBackupRoot $BackupRoot | ConvertTo-Json -Depth 8

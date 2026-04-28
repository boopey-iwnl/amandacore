param(
    [string]$Root = "",
    [switch]$IncludeUntracked,
    [switch]$TreatWarningsAsErrors,
    [switch]$SelfTest
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

function New-Finding {
    param(
        [string]$Severity,
        [string]$Path,
        [string]$Rule,
        [string]$Message
    )
    [pscustomobject]@{
        Severity = $Severity
        Path     = $Path
        Rule     = $Rule
        Message  = $Message
    }
}

if ($SelfTest) {
    $privateKeyPattern = '-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----'
    $privateKeySample = "-----BEGIN " + "PRIVATE KEY-----"
    if ($privateKeySample -notmatch $privateKeyPattern) {
        throw "private key self-test pattern failed"
    }
    $localPathSample = "C:" + "\Users\example\Downloads\asset.png"
    if ($localPathSample -notmatch '(?i)[A-Z]:[\\/](Users|Documents and Settings)[\\/]') {
        throw "local path self-test pattern failed"
    }
    Write-Host "Scan-ForbiddenArtifacts self-test passed."
    exit 0
}

if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
}
else {
    $Root = Resolve-Path $Root
}

$repoRoot = [System.IO.Path]::GetFullPath($Root)
$findings = New-Object System.Collections.Generic.List[object]

function Add-Finding {
    param(
        [string]$Severity,
        [string]$Path,
        [string]$Rule,
        [string]$Message
    )
    $findings.Add((New-Finding -Severity $Severity -Path $Path -Rule $Rule -Message $Message)) | Out-Null
}

function Get-RepoRelativePath {
    param([string]$Path)
    $full = [System.IO.Path]::GetFullPath($Path)
    if ($full.StartsWith($repoRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $full.Substring($repoRoot.Length).TrimStart('\', '/').Replace('\', '/')
    }
    return $Path.Replace('\', '/')
}

function Get-GitFiles {
    $git = Get-Command git -ErrorAction SilentlyContinue
    if (-not $git) {
        return @()
    }

    Push-Location $repoRoot
    try {
        $tracked = & $git.Source ls-files
        if ($LASTEXITCODE -ne 0) {
            return @()
        }
        $files = @($tracked)
        if ($IncludeUntracked) {
            $untracked = & $git.Source ls-files --others --exclude-standard
            if ($LASTEXITCODE -eq 0) {
                $files += @($untracked)
            }
        }
        return $files | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Sort-Object -Unique
    }
    finally {
        Pop-Location
    }
}

function Get-FallbackFiles {
    $excludedDirs = @(".git", ".secrets", "Cache", "cache", "build", "bin", "obj", "dist", "logs", "user")
    Get-ChildItem -Path $repoRoot -Recurse -File -Force | Where-Object {
        $relative = Get-RepoRelativePath $_.FullName
        foreach ($dir in $excludedDirs) {
            if ($relative -match "(^|/)$([regex]::Escape($dir))(/|$)") {
                return $false
            }
        }
        return $true
    } | ForEach-Object { Get-RepoRelativePath $_.FullName }
}

$files = @(Get-GitFiles)
if ($files.Count -eq 0) {
    $files = @(Get-FallbackFiles)
}

$pathRules = @(
    @{ Severity = "error"; Rule = "secrets-directory"; Pattern = '(^|/)\.secrets(/|$)'; Message = "Do not commit .secrets content." },
    @{ Severity = "error"; Rule = "env-file"; Pattern = '(^|/)(\.env|[^/]+\.env)$'; Message = "Do not commit environment files." },
    @{ Severity = "error"; Rule = "logs"; Pattern = '(^|/)(logs?|diagnostics?)(/|$)|(?i)\.(log|trace)$'; Message = "Do not commit logs or diagnostics." },
    @{ Severity = "error"; Rule = "archives"; Pattern = '(?i)\.(zip|7z|rar|tar|tgz|gz)$'; Message = "Do not commit release archives or nested packages." },
    @{ Severity = "error"; Rule = "build-output"; Pattern = '(^|/)(Cache|cache|build|bin|obj|dist|out)(/|$)'; Message = "Do not commit build output or caches." },
    @{ Severity = "error"; Rule = "temporary-files"; Pattern = '(^|/)(tmp|temp)(/|$)|(?i)\.(tmp|temp|lock)$'; Message = "Do not commit temporary files." },
    @{ Severity = "error"; Rule = "runtime-state"; Pattern = '(?i)(runtime[-_]?ticket|platform-state\.json|local-processes\.json|version-manifest\.json)$'; Message = "Do not commit runtime tickets or local process/state manifests." },
    @{ Severity = "error"; Rule = "local-database"; Pattern = '(?i)\.(db|sqlite|sqlite3)$'; Message = "Do not commit local database files." },
    @{ Severity = "error"; Rule = "private-key-file"; Pattern = '(?i)\.(pem|pfx|p12|key)$'; Message = "Do not commit private key material." }
)

$binaryExtensions = @(
    ".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".dds", ".tga", ".bmp",
    ".wav", ".mp3", ".ogg", ".fbx", ".pak", ".dll", ".exe", ".pdb"
)

foreach ($relative in $files) {
    $normalized = $relative.Replace('\', '/')
    foreach ($rule in $pathRules) {
        if ($normalized -match $rule.Pattern) {
            $severity = $rule.Severity
            if ($normalized -eq "Infra/dev/platform-state.json.lock") {
                $severity = "warning"
            }
            Add-Finding -Severity $severity -Path $normalized -Rule $rule.Rule -Message $rule.Message
        }
    }

    $fullPath = Join-Path $repoRoot $relative
    if (-not (Test-Path $fullPath -PathType Leaf)) {
        continue
    }
    $extension = [System.IO.Path]::GetExtension($fullPath).ToLowerInvariant()
    if ($binaryExtensions -contains $extension) {
        continue
    }
    $item = Get-Item $fullPath
    if ($item.Length -gt 2MB) {
        continue
    }

    try {
        $content = Get-Content -Path $fullPath -Raw -ErrorAction Stop
    }
    catch {
        continue
    }

    if ($content -match '-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----') {
        Add-Finding -Severity "error" -Path $normalized -Rule "private-key-content" -Message "Private key block detected."
    }
    if ($content -match '(?i)(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}|AKIA[0-9A-Z]{16}|xox[baprs]-[A-Za-z0-9-]{20,}') {
        Add-Finding -Severity "error" -Path $normalized -Rule "high-confidence-token" -Message "High-confidence token-like string detected."
    }
    if ($content -match '(?i)[A-Z]:[\\/](Users|Documents and Settings)[\\/]') {
        $severity = "error"
        if ($normalized -match '^(Docs|AGENTS\.md)(/|$)') {
            $severity = "warning"
        }
        Add-Finding -Severity $severity -Path $normalized -Rule "machine-local-path" -Message "Machine-local Windows path detected."
    }
    if ($content -match '(?i)C:[\\/]Users[\\/]forwo[\\/]Downloads[\\/]textures') {
        Add-Finding -Severity "error" -Path $normalized -Rule "downloads-texture-path" -Message "Runtime dependency on local Downloads texture path detected."
    }
}

$errors = @($findings | Where-Object { $_.Severity -eq "error" })
$warnings = @($findings | Where-Object { $_.Severity -eq "warning" })

foreach ($finding in ($findings | Sort-Object Severity, Path, Rule)) {
    Write-Host ("[{0}] {1}: {2} - {3}" -f $finding.Severity.ToUpperInvariant(), $finding.Path, $finding.Rule, $finding.Message)
}

Write-Host ("Forbidden artifact scan complete. Files scanned: {0}. Errors: {1}. Warnings: {2}." -f $files.Count, $errors.Count, $warnings.Count)

if ($errors.Count -gt 0 -or ($TreatWarningsAsErrors -and $warnings.Count -gt 0)) {
    exit 1
}

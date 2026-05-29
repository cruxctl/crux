param(
    [string]$Version = $(if ($env:CRUX_VERSION) { $env:CRUX_VERSION } else { "latest" }),
    [string]$Repo = $(if ($env:CRUX_REPO) { $env:CRUX_REPO } else { "github.com/cruxctl/crux" }),
    [string]$BinDir = $(if ($env:CRUX_BIN_DIR) { $env:CRUX_BIN_DIR } else { Join-Path $env:LOCALAPPDATA "Crux\bin" }),
    [string]$ConfigDir = $(if ($env:CRUX_CONFIG_DIR) { $env:CRUX_CONFIG_DIR } else { Join-Path $env:APPDATA "Crux" }),
    [string]$CruxdInstallUrl = $(if ($env:CRUXD_INSTALL_URL) { $env:CRUXD_INSTALL_URL } else { "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1" }),
    [string]$CruxdInstallRef = $(if ($env:CRUXD_INSTALL_REF) { $env:CRUXD_INSTALL_REF } else { "main" }),
    [switch]$Force,
    [switch]$SkipCruxd,
    [switch]$NoStart
)

$ErrorActionPreference = "Stop"

function Require-Command($Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "required command not found: $Name"
    }
}

function Test-Sha($Value) {
    return $Value -match "^[0-9a-fA-F]{40}$"
}

function Resolve-GitHubRef($Ref) {
    if (Test-Sha $Ref) {
        return $Ref
    }

    if (Get-Command git -ErrorAction SilentlyContinue) {
        $Lines = git ls-remote https://github.com/cruxctl/cruxd.git "refs/heads/$Ref" "refs/tags/$Ref"
        if ($LASTEXITCODE -eq 0 -and $Lines) {
            $First = @($Lines)[0]
            $Sha = ($First -split "\s+")[0]
            if (Test-Sha $Sha) {
                return $Sha
            }
        }
    }

    $Headers = @{
        "Accept" = "application/vnd.github+json"
        "User-Agent" = "crux-installer"
    }
    $Response = Invoke-RestMethod -Uri "https://api.github.com/repos/cruxctl/cruxd/commits/$Ref" -Headers $Headers
    if (-not (Test-Sha $Response.sha)) {
        throw "could not resolve cruxd installer ref: $Ref"
    }
    return $Response.sha
}

function Resolve-CruxdInstallUrl {
    $DefaultUrl = "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1"
    if ($CruxdInstallUrl -ne $DefaultUrl) {
        return $CruxdInstallUrl
    }

    $Sha = Resolve-GitHubRef $CruxdInstallRef
    return "https://raw.githubusercontent.com/cruxctl/cruxd/$Sha/scripts/install-cruxd.ps1"
}

Require-Command go
New-Item -ItemType Directory -Force -Path $BinDir, $ConfigDir | Out-Null

if (-not $SkipCruxd) {
    $ScriptPath = Join-Path ([System.IO.Path]::GetTempPath()) ("install-cruxd-" + [System.Guid]::NewGuid().ToString("N") + ".ps1")
    $ResolvedCruxdInstallUrl = Resolve-CruxdInstallUrl
    Invoke-WebRequest -Uri $ResolvedCruxdInstallUrl -UseBasicParsing -OutFile $ScriptPath
    try {
        $Args = @("-Version", $(if ($env:CRUXD_VERSION) { $env:CRUXD_VERSION } else { $Version }))
        if ($Force) { $Args += "-Force" }
        if ($NoStart) { $Args += "-NoStart" }
        & powershell -NoProfile -ExecutionPolicy Bypass -File $ScriptPath @Args
    } finally {
        Remove-Item -Force -ErrorAction SilentlyContinue $ScriptPath
    }
}

$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("crux-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $Tmp | Out-Null
try {
    Write-Host "Installing crux from $Repo@$Version"
    $env:GOBIN = $Tmp
    if (-not $env:GOPROXY) { $env:GOPROXY = "direct" }
    go install "$Repo/cmd/crux@$Version"
    Copy-Item -Force (Join-Path $Tmp "crux.exe") (Join-Path $BinDir "crux.exe")
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $Tmp
}

$ConfigPath = Join-Path $ConfigDir "config.yaml"
if (-not (Test-Path $ConfigPath)) {
@"
currentContext: local
contexts:
  local:
    serverUrl: http://127.0.0.1:4357
    namespace: default
"@ | Set-Content -NoNewline -Encoding UTF8 $ConfigPath
}

Write-Host "crux installed at $(Join-Path $BinDir "crux.exe")"

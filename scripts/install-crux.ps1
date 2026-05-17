param(
    [string]$Version = $(if ($env:CRUX_VERSION) { $env:CRUX_VERSION } else { "latest" }),
    [string]$Repo = $(if ($env:CRUX_REPO) { $env:CRUX_REPO } else { "github.com/cruxctl/crux" }),
    [string]$BinDir = $(if ($env:CRUX_BIN_DIR) { $env:CRUX_BIN_DIR } else { Join-Path $env:LOCALAPPDATA "Crux\bin" }),
    [string]$ConfigDir = $(if ($env:CRUX_CONFIG_DIR) { $env:CRUX_CONFIG_DIR } else { Join-Path $env:APPDATA "Crux" }),
    [string]$CruxdInstallUrl = $(if ($env:CRUXD_INSTALL_URL) { $env:CRUXD_INSTALL_URL } else { "https://raw.githubusercontent.com/cruxctl/cruxd/main/scripts/install-cruxd.ps1" }),
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

Require-Command go
New-Item -ItemType Directory -Force -Path $BinDir, $ConfigDir | Out-Null

if (-not $SkipCruxd) {
    $ScriptPath = Join-Path ([System.IO.Path]::GetTempPath()) ("install-cruxd-" + [System.Guid]::NewGuid().ToString("N") + ".ps1")
    Invoke-WebRequest -Uri $CruxdInstallUrl -UseBasicParsing -OutFile $ScriptPath
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
    serverUrl: http://127.0.0.1:7700
    namespace: default
"@ | Set-Content -NoNewline -Encoding UTF8 $ConfigPath
}

Write-Host "crux installed at $(Join-Path $BinDir "crux.exe")"


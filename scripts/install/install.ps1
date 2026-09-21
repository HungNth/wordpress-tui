# WPTUI installer for Windows
# Installs the latest stable release into the user-local application directory.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $false)]
    [string]$InstallDir
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$RepoOwner = "HungNth"
$RepoName = "wordpress-tui"
$AppName = "wptui"
$ExeName = "$AppName.exe"

$BaseDownloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download"

function Write-Info {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Fatal {
    param([string]$Message)
    Write-Error "error: $Message"
    exit 1
}
# 1. Architecture detection
# Query CIM Win32_Processor Architecture first (9 = x64/AMD64, 12 = ARM64)
# This reliably reflects the underlying processor architecture even under 32-bit PowerShell or emulation.
$DetectedArch = $null
try {
    $ProcArch = (Get-CimInstance Win32_Processor -ErrorAction Stop | Select-Object -First 1).Architecture
    if ($ProcArch -eq 9) {
        $DetectedArch = "amd64"
    } elseif ($ProcArch -eq 12) {
        $DetectedArch = "arm64"
    }
} catch {
    # Fallback for environments where CIM is unavailable
    $EnvArch = $env:PROCESSOR_ARCHITEW6432
    if (-not $EnvArch) {
        $EnvArch = $env:PROCESSOR_ARCHITECTURE
    }
    if ($EnvArch -ieq "AMD64") {
        $DetectedArch = "amd64"
    } elseif ($EnvArch -ieq "ARM64") {
        $DetectedArch = "arm64"
    } else {
        $DetectedArch = $EnvArch
    }
}

if ($DetectedArch -eq "amd64") {
    $TargetArch = "amd64"
} elseif ($DetectedArch -eq "arm64") {
    Write-Fatal "Windows ARM64 is not currently supported. WPTUI publishes 64-bit x86 (amd64) binaries only."
} else {
    Write-Fatal "Unsupported Windows architecture '$DetectedArch'. WPTUI supports 64-bit x86 (amd64) on Windows."
}

# 2. Determine target install directory
if (-not $InstallDir) {
    $LocalApp = $env:LOCALAPPDATA
    if (-not $LocalApp) {
        $LocalApp = Join-Path $env:USERPROFILE "AppData\Local"
    }
    $InstallDir = Join-Path $LocalApp "Programs\$AppName"
}

$TargetExe = Join-Path $InstallDir $ExeName

# 3. Setup temporary staging directory
$TempRoot = [System.IO.Path]::GetTempPath()
$StagingDir = Join-Path $TempRoot ("wptui-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $StagingDir -Force | Out-Null

try {
    # 4. Fetch checksums manifest using -UseBasicParsing for PS 5.1 compatibility
    $ChecksumsFile = Join-Path $StagingDir "checksums.txt"
    Write-Info "Fetching release checksums..."
    try {
        Invoke-WebRequest -Uri "$BaseDownloadUrl/checksums.txt" -OutFile $ChecksumsFile -UseBasicParsing
    } catch {
        Write-Fatal "Failed to download checksums from '$BaseDownloadUrl/checksums.txt': $_"
    }

    # 5. Parse manifest for matching windows_amd64 asset
    $AssetRegex = '^wptui_[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?_windows_amd64\.zip$'
    $MatchingLines = @()
    foreach ($Line in Get-Content -Path $ChecksumsFile) {
        $Trimmed = $Line.Trim()
        if (-not $Trimmed) { continue }
        $Parts = $Trimmed -split '\s+'
        if ($Parts.Length -ge 2 -and $Parts[1] -match $AssetRegex) {
            $MatchingLines += ,@($Parts[0], $Parts[1])
        }
    }

    if ($MatchingLines.Count -eq 0) {
        Write-Fatal "No asset matching Windows amd64 pattern found in checksums.txt."
    }
    if ($MatchingLines.Count -gt 1) {
        Write-Fatal "Multiple assets matching Windows amd64 pattern found in checksums.txt."
    }

    $ExpectedHash = $MatchingLines[0][0]
    $AssetName = $MatchingLines[0][1]

    if ($ExpectedHash -notmatch '^[0-9a-fA-F]{64}$') {
        Write-Fatal "Malformed SHA-256 digest in checksums.txt: '$ExpectedHash'."
    }

    # 6. Download asset archive
    $ArchivePath = Join-Path $StagingDir $AssetName
    Write-Info "Downloading $AssetName..."
    try {
        Invoke-WebRequest -Uri "$BaseDownloadUrl/$AssetName" -OutFile $ArchivePath -UseBasicParsing
    } catch {
        Write-Fatal "Failed to download release archive '$AssetName' from '$BaseDownloadUrl': $_"
    }

    # 7. Verify SHA-256 checksum using .NET Cryptography (guaranteed in all PS 5.1+ environments)
    Write-Info "Verifying SHA-256 checksum..."
    $Sha256 = [System.Security.Cryptography.SHA256]::Create()
    $Stream = [System.IO.File]::OpenRead($ArchivePath)
    try {
        $HashBytes = $Sha256.ComputeHash($Stream)
    } finally {
        $Stream.Dispose()
        $Sha256.Dispose()
    }
    $FileHash = -join ($HashBytes | ForEach-Object { "{0:x2}" -f $_ })
    if ($FileHash -ine $ExpectedHash) {
        Write-Fatal "Checksum mismatch for '$AssetName'.`nExpected:   $ExpectedHash`nCalculated: $FileHash"
    }

    # 8. Extract archive and verify exactly 1 member named wptui.exe
    $ExtractDir = Join-Path $StagingDir "extracted"
    New-Item -ItemType Directory -Path $ExtractDir -Force | Out-Null
    
    # Use .NET ZipFile to inspect entries without external tool dependency
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $ZipArchive = [System.IO.Compression.ZipFile]::OpenRead($ArchivePath)
    try {
        if ($ZipArchive.Entries.Count -ne 1) {
            Write-Fatal "Archive '$AssetName' must contain exactly one member '$ExeName', found $($ZipArchive.Entries.Count) entries."
        }
        if ($ZipArchive.Entries[0].FullName -ne $ExeName) {
            Write-Fatal "Archive '$AssetName' member is '$($ZipArchive.Entries[0].FullName)', expected '$ExeName'."
        }
    } finally {
        $ZipArchive.Dispose()
    }

    [System.IO.Compression.ZipFile]::ExtractToDirectory($ArchivePath, $ExtractDir)

    $StageExe = Join-Path $ExtractDir $ExeName
    if (-not (Test-Path -Path $StageExe -PathType Leaf)) {
        Write-Fatal "Extracted executable '$StageExe' not found."
    }

    # 9. Create target directory and atomically replace executable
    if (-not (Test-Path -Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $TempTarget = Join-Path $InstallDir ("$AppName-" + [System.Guid]::NewGuid().ToString("N") + ".tmp")
    Copy-Item -Path $StageExe -Destination $TempTarget -Force

    try {
        Move-Item -Path $TempTarget -Destination $TargetExe -Force
    } catch {
        Remove-Item -Path $TempTarget -Force -ErrorAction SilentlyContinue
        Write-Fatal "Failed to replace '$TargetExe'. If WPTUI is currently running, please close it and retry the installer: $_"
    }

    Write-Info "Installed $AppName to $TargetExe"

    # 10. User PATH update
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
    $NeedsPathUpdate = $true
    if ($UserPath) {
        $ExistingDirs = $UserPath -split ';' | ForEach-Object { $_.Trim().TrimEnd('\/') }
        $NormalizedInstallDir = $InstallDir.Trim().TrimEnd('\/')
        foreach ($d in $ExistingDirs) {
            if ($d -ieq $NormalizedInstallDir) {
                $NeedsPathUpdate = $false
                break
            }
        }
    }

    if ($NeedsPathUpdate) {
        $NewUserPath = if ($UserPath -and $UserPath.Trim().Length -gt 0) {
            $UserPath.TrimEnd(';') + ";$InstallDir"
        } else {
            $InstallDir
        }
        [Environment]::SetEnvironmentVariable("PATH", $NewUserPath, [EnvironmentVariableTarget]::User)
        Write-Host ""
        Write-Info "Added '$InstallDir' to your User PATH."
        Write-Info "Please restart any open terminal windows to apply the PATH changes."
    }

    # Ensure the active PowerShell session PATH contains the directory
    $ProcessDirs = $env:PATH -split ';' | ForEach-Object { $_.Trim().TrimEnd('\/') }
    $InProcessPath = $false
    foreach ($d in $ProcessDirs) {
        if ($d -ieq $NormalizedInstallDir) {
            $InProcessPath = $true
            break
        }
    }
    if (-not $InProcessPath) {
        $env:PATH = "$env:PATH;$InstallDir"
    }

    if (-not $NeedsPathUpdate) {
        Write-Host ""
        Write-Info "WPTUI is ready to run! Execute '$AppName' to get started."
    }

} finally {
    Remove-Item -Path $StagingDir -Recurse -Force -ErrorAction SilentlyContinue
}

# download-mesa-opengl.ps1
# Downloads Mesa3D's llvmpipe OpenGL implementation for Windows.
# This provides a software fallback for machines without GPU drivers
# (VMs, RDP sessions, basic display adapters).
#
# Usage: .\scripts\download-mesa-opengl.ps1 -OutDir dist
#
# The script downloads the llvmpipe x64 build from mmozeiko/build-mesa,
# extracts opengl32.dll, and places it in the specified output directory.

param(
    [Parameter(Mandatory=$true)]
    [string]$OutDir
)

$ErrorActionPreference = "Stop"

$MesaVersion = "26.2.0"
$DownloadUrl = "https://github.com/mmozeiko/build-mesa/releases/download/$MesaVersion/mesa-llvmpipe-x64.7z"
$TempDir = Join-Path $env:TEMP "mesa-download"
$ArchivePath = Join-Path $TempDir "mesa-llvmpipe-x64.7z"

Write-Host "Downloading Mesa3D llvmpipe $MesaVersion..."

# Create temp directory
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

# Download the archive
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath

# Extract using 7z (available on GitHub Actions runners)
Write-Host "Extracting opengl32.dll..."
7z e $ArchivePath -o"$TempDir" "opengl32.dll" -y | Out-Null

$DllPath = Join-Path $TempDir "opengl32.dll"
if (-not (Test-Path $DllPath)) {
    Write-Error "Failed to extract opengl32.dll from Mesa archive"
    exit 1
}

# Copy to output directory
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
Copy-Item $DllPath (Join-Path $OutDir "opengl32.dll")

# Clean up
Remove-Item -Recurse -Force $TempDir

Write-Host "Mesa3D opengl32.dll ($MesaVersion) placed in $OutDir"

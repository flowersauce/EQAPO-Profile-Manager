#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z]+)*)?$')]
    [string] $Version,

    # Override only when a prerelease needs a distinct numeric MSI version.
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string] $MsiVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Release packaging requires Windows.' }
if (-not $MsiVersion) {
    if ($Version.Contains('-')) {
        throw 'Prerelease builds require an explicit, unique -MsiVersion.'
    }
    $MsiVersion = $Version
}
$numericVersion = [version] $MsiVersion
if ($numericVersion.Major -gt 255 -or $numericVersion.Minor -gt 255 -or $numericVersion.Build -gt 65535) {
    throw 'MsiVersion must fit Windows Installer limits: 255.255.65535.'
}

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$outputRoot = Join-Path $repoRoot 'dist'
# A new staging directory avoids reusing user config or a previous portable flag.
$stage = Join-Path $outputRoot ('.stage-' + [guid]::NewGuid().ToString('N'))
$binaryDir = Join-Path $stage 'binary'
$portableDir = Join-Path $stage 'portable/EQM'
$null = New-Item -ItemType Directory -Path $binaryDir -Force
$msiPath = Join-Path $outputRoot "EQM-$Version-windows-x64.msi"
$zipPath = Join-Path $outputRoot "EQM-$Version-windows-x64-portable.zip"
$hashPath = Join-Path $outputRoot "EQM-$Version-SHA256SUMS.txt"
$stagedMsi = Join-Path $stage ([IO.Path]::GetFileName($msiPath))
$stagedZip = Join-Path $stage ([IO.Path]::GetFileName($zipPath))
$stagedHash = Join-Path $stage ([IO.Path]::GetFileName($hashPath))
foreach ($artifact in @($msiPath, $zipPath, $hashPath)) {
    if (Test-Path -LiteralPath $artifact) { throw "Artifact already exists: $artifact. Use a new version or move the old artifact first." }
}

Push-Location $repoRoot
try {
    & (Join-Path $repoRoot 'scripts/build-windows.ps1') `
        -OutputPath (Join-Path $binaryDir 'eqm.exe') -Version $Version -Release

    & (Join-Path $repoRoot 'scripts/package-portable.ps1') `
        -BinaryPath (Join-Path $binaryDir 'eqm.exe') -PortableDir $portableDir -ZipPath $stagedZip
    & (Join-Path $repoRoot 'scripts/package-msi.ps1') `
        -SourceDir $binaryDir -MsiVersion $MsiVersion -MsiPath $stagedMsi `
        -IntermediateDir (Join-Path $stage 'wix')

    $hashes = foreach ($artifact in @($stagedZip, $stagedMsi)) {
        '{0}  {1}' -f (Get-FileHash -LiteralPath $artifact -Algorithm SHA256).Hash, [IO.Path]::GetFileName($artifact)
    }
    $hashes | Set-Content -LiteralPath $stagedHash -Encoding utf8
    foreach ($artifact in @($stagedZip, $stagedMsi, $stagedHash)) {
        Move-Item -LiteralPath $artifact -Destination $outputRoot
    }
    Write-Host "Portable ZIP: $zipPath"
    Write-Host "Per-user MSI: $msiPath"
    Write-Host "SHA256:       $hashPath"
    if ($Version.Contains('-')) {
        Write-Host 'WinGet manifest: the separate generator currently accepts numeric release versions only.'
    } else {
        Write-Host "WinGet manifest: after uploading the MSI to the v$Version GitHub Release, run:"
        Write-Host ".\scripts\new-winget-manifest.ps1 -Version $Version"
    }
}
finally {
    Pop-Location
}

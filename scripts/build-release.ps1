#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z]+)*)?$')]
    [string] $Version,

    # Override only when a prerelease needs a distinct numeric MSI version.
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string] $MsiVersion,
    [switch] $IncludeStore,
    [string] $SdkBin,
    [string] $Python,
    [string] $Uv = 'uv'
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
if ($IncludeStore -and ($Version.Contains('-') -or ([version] $Version).Major -lt 1)) {
    throw 'Store packaging requires a stable version with major component greater than zero.'
}

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$outputRoot = Join-Path $repoRoot 'output'
$publicDir = Join-Path $outputRoot 'public'
$storeDir = Join-Path $outputRoot 'store'
# Stage beside output so publishing and rollback use same-volume directory moves.
$stage = Join-Path $repoRoot ('.release-stage-' + [guid]::NewGuid().ToString('N'))
$nextOutput = Join-Path $stage 'output'
$previousOutput = Join-Path $stage 'previous-output'
$binaryDir = Join-Path $stage 'binary'
$portableDir = Join-Path $stage 'portable/EQM'
$msiPath = Join-Path $publicDir "EQM-$Version-windows-x64.msi"
$zipPath = Join-Path $publicDir "EQM-$Version-windows-x64-portable.zip"
$hashPath = Join-Path $publicDir "EQM-$Version-SHA256SUMS.txt"
$msixPath = Join-Path $storeDir "EQM-$Version-windows-x64-store.msix"
$stagedMsix = Join-Path $stage ([IO.Path]::GetFileName($msixPath))
$stagedMsi = Join-Path $stage ([IO.Path]::GetFileName($msiPath))
$stagedZip = Join-Path $stage ([IO.Path]::GetFileName($zipPath))
$stagedHash = Join-Path $stage ([IO.Path]::GetFileName($hashPath))

function Assert-ReleaseDirectory([string] $Path) {
    $fullPath = [IO.Path]::GetFullPath($Path)
    if (Test-Path -LiteralPath $Path) {
        $item = Get-Item -LiteralPath $Path -Force
        if (-not $item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) {
            throw "Expected a regular release directory: $Path"
        }
        $fullPath = (Resolve-Path -LiteralPath $Path).ProviderPath
    }
    if (-not $fullPath.StartsWith($repoRoot + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Release directory is outside the repository: $Path"
    }
}

Assert-ReleaseDirectory $outputRoot
if (Test-Path -LiteralPath $stage) { throw "Staging directory already exists: $stage" }
$keepStage = $false
Push-Location $repoRoot
try {
    $null = New-Item -ItemType Directory -Path $binaryDir -Force
    & (Join-Path $repoRoot 'scripts/build-windows.ps1') `
        -OutputPath (Join-Path $binaryDir 'eqm.exe') -Version $Version -Release

    & (Join-Path $repoRoot 'scripts/package-portable.ps1') `
        -BinaryPath (Join-Path $binaryDir 'eqm.exe') -PortableDir $portableDir -ZipPath $stagedZip
    & (Join-Path $repoRoot 'scripts/package-msi.ps1') `
        -SourceDir $binaryDir -MsiVersion $MsiVersion -MsiPath $stagedMsi `
        -IntermediateDir (Join-Path $stage 'wix')

    if ($IncludeStore) {
        & (Join-Path $repoRoot 'scripts/package-msix.ps1') `
            -BinaryPath (Join-Path $binaryDir 'eqm.exe') -Version $Version -MsixPath $stagedMsix `
            -IntermediateDir (Join-Path $stage 'msix') -SdkBin $SdkBin -Python $Python -Uv $Uv
    }

    $hashes = foreach ($artifact in @($stagedZip, $stagedMsi)) {
        '{0}  {1}' -f (Get-FileHash -LiteralPath $artifact -Algorithm SHA256).Hash, [IO.Path]::GetFileName($artifact)
    }
    $hashes | Set-Content -LiteralPath $stagedHash -Encoding utf8
    $nextPublic = Join-Path $nextOutput 'public'
    $null = New-Item -ItemType Directory -Path $nextPublic -Force
    foreach ($artifact in @($stagedZip, $stagedMsi, $stagedHash)) {
        Move-Item -LiteralPath $artifact -Destination $nextPublic
    }
    if ($IncludeStore) {
        $nextStore = Join-Path $nextOutput 'store'
        $null = New-Item -ItemType Directory -Path $nextStore -Force
        Move-Item -LiteralPath $stagedMsix -Destination $nextStore
    }

    foreach ($directory in @($outputRoot, $nextOutput, $previousOutput)) { Assert-ReleaseDirectory $directory }
    if (Test-Path -LiteralPath $outputRoot) {
        Move-Item -LiteralPath $outputRoot -Destination $previousOutput
    }
    try {
        Move-Item -LiteralPath $nextOutput -Destination $outputRoot
    }
    catch {
        try {
            if (Test-Path -LiteralPath $previousOutput) {
                Assert-ReleaseDirectory $previousOutput
                Assert-ReleaseDirectory $outputRoot
                Move-Item -LiteralPath $previousOutput -Destination $outputRoot
            }
        }
        catch {
            $keepStage = $true
            throw "Could not restore the previous output; retained it at $previousOutput. $($_.Exception.Message)"
        }
        throw
    }
    if ($IncludeStore) {
        Write-Host "Store MSIX:   $msixPath"
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
    if (-not $keepStage -and (Test-Path -LiteralPath $stage)) {
        Assert-ReleaseDirectory $stage
        Remove-Item -LiteralPath $stage -Recurse -Force
    }
}

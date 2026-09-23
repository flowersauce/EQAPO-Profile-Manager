#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+(?:-[0-9A-Za-z]+(?:\.[0-9A-Za-z]+)*)?$')]
    [string] $Version,

    # Windows Installer compares three numeric fields, not prerelease labels.
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string] $MsiVersion
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Release packaging requires Windows.' }
$numericVersion = [version] $MsiVersion
if ($numericVersion.Major -gt 255 -or $numericVersion.Minor -gt 255 -or $numericVersion.Build -gt 65535) {
    throw 'MsiVersion must fit Windows Installer limits: 255.255.65535.'
}

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$outputRoot = Join-Path $repoRoot 'dist'
# A new staging directory avoids reusing user config or a previous portable flag.
$stage = Join-Path $outputRoot ('.stage-' + [guid]::NewGuid().ToString('N'))
$binaryDir = Join-Path $stage 'binary'
$portableRoot = Join-Path $stage 'portable'
$portableDir = Join-Path $portableRoot 'EQM'
$null = New-Item -ItemType Directory -Path $binaryDir, (Join-Path $portableDir 'config') -Force
$msiPath = Join-Path $outputRoot "EQM-$Version-windows-x64.msi"
$zipPath = Join-Path $outputRoot "EQM-$Version-windows-x64-portable.zip"
$hashPath = Join-Path $outputRoot "EQM-$Version-SHA256SUMS.txt"
$stagedMsi = Join-Path $stage ([IO.Path]::GetFileName($msiPath))
$stagedZip = Join-Path $stage ([IO.Path]::GetFileName($zipPath))
$stagedHash = Join-Path $stage ([IO.Path]::GetFileName($hashPath))
foreach ($artifact in @($msiPath, $zipPath, $hashPath)) {
    if (Test-Path -LiteralPath $artifact) { throw "Artifact already exists: $artifact. Use a new version or move the old artifact first." }
}

$oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
Push-Location $repoRoot
try {
    & dotnet tool restore
    if ($LASTEXITCODE -ne 0) { throw 'WiX tool restore failed.' }
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    & go build -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $binaryDir 'eqm.exe') ./cmd/eqm
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }

    Copy-Item -LiteralPath (Join-Path $binaryDir 'eqm.exe') -Destination $portableDir
    [IO.File]::WriteAllText((Join-Path $portableDir 'portable.flag'), '')
    Compress-Archive -LiteralPath $portableDir -DestinationPath $stagedZip

    & dotnet tool run wix -- build (Join-Path $repoRoot 'packaging/windows/Package.wxs') `
        -arch x64 -d "MsiVersion=$MsiVersion" -d "SourceDir=$binaryDir" `
        -intermediateFolder (Join-Path $stage 'wix') -o $stagedMsi
    if ($LASTEXITCODE -ne 0) { throw 'WiX MSI build failed.' }

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
    Write-Host 'winget PackageIdentifier: Flowersauce.EQAPOProfileManager'
}
finally {
    $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
    Pop-Location
}

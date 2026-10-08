param(
    [Parameter(Mandatory)] [string] $SourceDir,
    [Parameter(Mandatory)] [string] $MsiVersion,
    [Parameter(Mandatory)] [string] $MsiPath,
    [Parameter(Mandatory)] [string] $IntermediateDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$wix = Get-Command wix -CommandType Application -ErrorAction SilentlyContinue
if (-not $wix) {
    throw 'WiX is required on PATH. Install it manually before packaging.'
}

Push-Location $repoRoot
try {
    & $wix.Source build (Join-Path $repoRoot 'packaging/windows/Package.wxs') `
        -arch x64 -d "MsiVersion=$MsiVersion" -d "SourceDir=$SourceDir" `
        -ext WixToolset.Util.wixext `
        -intermediateFolder $IntermediateDir -o $MsiPath
    if ($LASTEXITCODE -ne 0) { throw 'WiX MSI build failed.' }
}
finally {
    Pop-Location
}

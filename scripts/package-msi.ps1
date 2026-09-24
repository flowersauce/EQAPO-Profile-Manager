param(
    [Parameter(Mandatory)] [string] $SourceDir,
    [Parameter(Mandatory)] [string] $MsiVersion,
    [Parameter(Mandatory)] [string] $MsiPath,
    [Parameter(Mandatory)] [string] $IntermediateDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
Push-Location $repoRoot
try {
    & dotnet tool run wix -- build (Join-Path $repoRoot 'packaging/windows/Package.wxs') `
        -arch x64 -d "MsiVersion=$MsiVersion" -d "SourceDir=$SourceDir" `
        -intermediateFolder $IntermediateDir -o $MsiPath
    if ($LASTEXITCODE -ne 0) { throw 'WiX MSI build failed.' }
}
finally {
    Pop-Location
}

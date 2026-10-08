#requires -Version 7.0
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $BinaryPath,
    [Parameter(Mandatory)] [ValidatePattern('^\d+\.\d+\.\d+$')] [string] $Version,
    [Parameter(Mandatory)] [string] $MsixPath,
    [Parameter(Mandatory)] [string] $IntermediateDir,
    [string] $SdkBin,
    [string] $Python,
    [string] $Uv = 'uv'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'MSIX packaging requires Windows.' }
$numericVersion = [version] $Version
if ($numericVersion.Major -lt 1 -or $numericVersion.Major -gt 65535 -or $numericVersion.Minor -gt 65535 -or $numericVersion.Build -gt 65535) {
    throw 'Store version must fit 1..65535.0..65535.0..65535; its fourth component is always 0.'
}
$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$BinaryPath = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($BinaryPath)
$MsixPath = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($MsixPath)
$IntermediateDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($IntermediateDir)
if (-not (Test-Path -LiteralPath $BinaryPath -PathType Leaf)) { throw "Missing executable: $BinaryPath" }
foreach ($path in @($MsixPath, $IntermediateDir)) {
    if (Test-Path -LiteralPath $path) { throw "Output already exists: $path. Use a new version or staging directory." }
}
if (-not $SdkBin) {
    $sdkRoot = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits/10/bin'
    $candidates = @(Get-ChildItem -LiteralPath $sdkRoot -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '^\d+\.\d+\.\d+\.\d+$' } |
        Sort-Object { [version] $_.Name } -Descending)
    foreach ($candidate in $candidates) {
        $bin = Join-Path $candidate.FullName 'x64'
        if ((Test-Path -LiteralPath (Join-Path $bin 'makeappx.exe')) -and (Test-Path -LiteralPath (Join-Path $bin 'makepri.exe'))) {
            $SdkBin = $bin
            break
        }
    }
}
if (-not $SdkBin) { throw 'Install a Windows SDK with makeappx.exe and makepri.exe, or specify -SdkBin.' }
$makeappx = Join-Path $SdkBin 'makeappx.exe'
$makepri = Join-Path $SdkBin 'makepri.exe'
foreach ($tool in @($makeappx, $makepri)) {
    if (-not (Test-Path -LiteralPath $tool -PathType Leaf)) { throw "Missing SDK tool: $tool" }
}
if (-not (Get-Command $Uv -CommandType Application -ErrorAction SilentlyContinue)) {
    throw 'Install uv to run the SVG icon generator with the project dependencies in uv.lock.'
}

$payload = Join-Path $IntermediateDir 'package'
$null = New-Item -ItemType Directory -Path $payload
Copy-Item -LiteralPath $BinaryPath -Destination (Join-Path $payload 'eqm.exe')
Copy-Item -LiteralPath (Join-Path $repoRoot 'LICENSE') -Destination $payload
$uvArgs = @('run', '--locked', '--project', $repoRoot)
if ($Python) { $uvArgs += @('--python', $Python) }
& $Uv @uvArgs python (Join-Path $repoRoot 'scripts/generate-app-icon.py') --msix-assets (Join-Path $payload 'Assets')
if ($LASTEXITCODE -ne 0) { throw 'MSIX icon generation failed.' }

$manifestPath = Join-Path $payload 'AppxManifest.xml'
$template = Get-Content -LiteralPath (Join-Path $repoRoot 'packaging/msix/AppxManifest.xml.in') -Raw
$template.Replace('@APP_VERSION@', "$Version.0") | Set-Content -LiteralPath $manifestPath -Encoding utf8NoBOM
$priConfig = Join-Path $IntermediateDir 'priconfig.xml'
& $makepri createconfig /cf $priConfig /dq lang-en-US /pv 10.0.0
if ($LASTEXITCODE -ne 0) { throw 'MSIX PRI configuration failed.' }
& $makepri new /pr $payload /cf $priConfig /mn $manifestPath /of (Join-Path $payload 'resources.pri')
if ($LASTEXITCODE -ne 0) { throw 'MSIX resource indexing failed.' }
$null = New-Item -ItemType Directory -Path ([IO.Path]::GetDirectoryName($MsixPath)) -Force
& $makeappx pack /d $payload /p $MsixPath
if ($LASTEXITCODE -ne 0) { throw 'MSIX packaging or manifest validation failed.' }
Write-Host "Unsigned Store MSIX: $MsixPath"

param(
    [string] $OutputPath = 'eqm.exe',
    [string] $Version,
    [switch] $Release
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$iconPath = Join-Path $repoRoot 'internal/resources/icons/app/eqm.ico'
$resourcePrefix = Join-Path $repoRoot 'cmd/eqm/rsrc'
$resourcePath = "${resourcePrefix}_windows_amd64.syso"
$exePath = if ([IO.Path]::IsPathRooted($OutputPath)) {
    $OutputPath
} else {
    Join-Path $repoRoot $OutputPath
}

if (-not (Test-Path -LiteralPath $iconPath -PathType Leaf)) { throw "Icon not found: $iconPath" }
if (Test-Path -LiteralPath $resourcePath) { throw "Temporary resource already exists: $resourcePath" }
$goWinres = Get-Command go-winres -CommandType Application -ErrorAction SilentlyContinue
if (-not $goWinres) {
    throw 'go-winres is required on PATH. Install it manually before building.'
}

$oldGOOS, $oldGOARCH, $oldCGO = $env:GOOS, $env:GOARCH, $env:CGO_ENABLED
Push-Location $repoRoot

try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'

    & $goWinres.Source simply --manifest none `
        --icon $iconPath --arch amd64 --out $resourcePrefix
    if ($LASTEXITCODE -ne 0) { throw 'Windows icon resource generation failed.' }

    $buildArgs = @('build', '-trimpath')
    $ldflags = if ($Release) { '-s -w' } else { '' }
    if ($Version) { $ldflags += " -X main.version=$Version" }
    if ($ldflags) { $buildArgs += @('-ldflags', $ldflags.Trim()) }
    $buildArgs += @('-o', $exePath, './cmd/eqm')
    & go @buildArgs
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
}
finally {
    if (Test-Path -LiteralPath $resourcePath) { Remove-Item -LiteralPath $resourcePath }
    $env:GOOS, $env:GOARCH, $env:CGO_ENABLED = $oldGOOS, $oldGOARCH, $oldCGO
    Pop-Location
}

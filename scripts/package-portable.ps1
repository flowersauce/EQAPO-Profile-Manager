param(
    [Parameter(Mandatory)] [string] $BinaryPath,
    [Parameter(Mandatory)] [string] $PortableDir,
    [Parameter(Mandatory)] [string] $ZipPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$null = New-Item -ItemType Directory -Path (Join-Path $PortableDir 'config') -Force
Copy-Item -LiteralPath $BinaryPath -Destination $PortableDir
[IO.File]::WriteAllText((Join-Path $PortableDir 'portable.flag'), '')
Compress-Archive -LiteralPath $PortableDir -DestinationPath $ZipPath

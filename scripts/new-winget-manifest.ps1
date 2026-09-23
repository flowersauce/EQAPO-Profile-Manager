#requires -Version 7.0
<#
.SYNOPSIS
Generate WinGet manifests from the final GitHub Release MSI.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [ValidatePattern('^\d+\.\d+\.\d+$')]
    [string] $Version,
    [string] $OutputDirectory = (Join-Path $PSScriptRoot '../dist/winget')
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
if (-not $IsWindows) { throw 'Manifest generation requires Windows.' }

$identifier = 'Flowersauce.EQAPOProfileManager'
$repository = 'https://github.com/flowersauce/EQAPO-Profile-Manager'
$installerUrl = "$repository/releases/download/v$Version/EQM-$Version-windows-x64.msi"
$outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
$manifestDirectory = Join-Path $outputRoot "manifests/f/Flowersauce/EQAPOProfileManager/$Version"
if (Test-Path -LiteralPath $manifestDirectory) {
    throw "Output already exists: $manifestDirectory. Move the previous manifest directory before regenerating."
}
$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) ('eqm-winget-' + [guid]::NewGuid().ToString('N'))
$null = New-Item -ItemType Directory -Path $tempDirectory
$msiPath = Join-Path $tempDirectory 'eqm.msi'
$installer = $null
$database = $null

function Read-MsiProperty([string] $Name) {
    $view = $null
    $record = $null
    try {
        $view = $database.OpenView("SELECT ``Value`` FROM ``Property`` WHERE ``Property`` = '$Name'")
        # COM method results must not become part of the returned property value.
        $null = $view.Execute()
        $record = $view.Fetch()
        if ($null -eq $record) { return '' }
        return [string] $record.StringData(1)
    }
    finally {
        if ($null -ne $record) { $null = [Runtime.InteropServices.Marshal]::FinalReleaseComObject($record) }
        if ($null -ne $view) {
            $null = $view.Close()
            $null = [Runtime.InteropServices.Marshal]::FinalReleaseComObject($view)
        }
    }
}

# Single-quoted YAML scalars safely preserve punctuation in MSI metadata.
function ConvertTo-YamlString([string] $Value) {
    if ($Value -match '[\r\n\x00-\x1f]') { throw 'Unexpected control character in MSI metadata.' }
    return "'" + $Value.Replace("'", "''") + "'"
}

try {
    Write-Host "Downloading $installerUrl"
    Invoke-WebRequest -Uri $installerUrl -OutFile $msiPath
    $sha256 = (Get-FileHash -LiteralPath $msiPath -Algorithm SHA256).Hash
    $installer = New-Object -ComObject WindowsInstaller.Installer
    $database = $installer.OpenDatabase($msiPath, 0)
    $productVersion = Read-MsiProperty 'ProductVersion'
    $productName = Read-MsiProperty 'ProductName'
    $publisher = Read-MsiProperty 'Manufacturer'
    $productCode = Read-MsiProperty 'ProductCode'
    $upgradeCode = Read-MsiProperty 'UpgradeCode'
    if ($productVersion -cne $Version -or $productName -cne 'EQM' -or $publisher -cne 'flowersauce') {
        throw "Unexpected MSI identity: $productName / $publisher / $productVersion"
    }
    foreach ($code in @($productCode, $upgradeCode)) {
        if ($code -notmatch '^\{[0-9A-Fa-f]{8}(-[0-9A-Fa-f]{4}){3}-[0-9A-Fa-f]{12}\}$') {
            throw "Invalid MSI product or upgrade code: $code"
        }
    }
    if ((Read-MsiProperty 'ALLUSERS') -ne '') { throw 'Expected a per-user MSI with no ALLUSERS value.' }
    $summary = $database.SummaryInformation(0)
    try {
        if (([string] $summary.Property(7)).Split(';')[0] -ne 'x64') { throw 'Expected an x64 MSI.' }
    }
    finally { $null = [Runtime.InteropServices.Marshal]::FinalReleaseComObject($summary) }

    $common = "PackageIdentifier: $identifier`nPackageVersion: '$Version'"
    $files = [ordered]@{}
    $files["$identifier.yaml"] = @"
$common
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.12.0
"@
    $files["$identifier.installer.yaml"] = @"
$common
InstallerType: wix
Scope: user
UpgradeBehavior: install
Commands:
- eqm
Installers:
- Architecture: x64
  InstallerUrl: $installerUrl
  InstallerSha256: $sha256
  ProductCode: $(ConvertTo-YamlString $productCode)
  AppsAndFeaturesEntries:
  - DisplayName: $(ConvertTo-YamlString $productName)
    Publisher: $(ConvertTo-YamlString $publisher)
    DisplayVersion: $(ConvertTo-YamlString $productVersion)
    ProductCode: $(ConvertTo-YamlString $productCode)
    UpgradeCode: $(ConvertTo-YamlString $upgradeCode)
ManifestType: installer
ManifestVersion: 1.12.0
"@
    foreach ($locale in @('en-US', 'zh-CN')) {
        $description = if ($locale -eq 'zh-CN') { '轻松导入、切换和管理 Equalizer APO 配置。' } else { 'Import, switch, and manage Equalizer APO profiles.' }
        $manifestType = if ($locale -eq 'en-US') { 'defaultLocale' } else { 'locale' }
        $files["$identifier.locale.$locale.yaml"] = @"
$common
PackageLocale: $locale
Publisher: flowersauce
PublisherUrl: https://github.com/flowersauce
PackageName: EQM
PackageUrl: $repository
License: MIT
LicenseUrl: $repository/blob/v$Version/LICENSE
ShortDescription: $description
Moniker: eqm
Tags:
- audio
- autoeq
- equalizer-apo
ReleaseNotesUrl: $repository/releases/tag/v$Version
ManifestType: $manifestType
ManifestVersion: 1.12.0
"@
        # Moniker belongs only to the defaultLocale schema.
        if ($locale -ne 'en-US') { $files["$identifier.locale.$locale.yaml"] = $files["$identifier.locale.$locale.yaml"] -replace '(?m)^Moniker: eqm\r?\n', '' }
    }
    $null = New-Item -ItemType Directory -Path $manifestDirectory
    foreach ($entry in $files.GetEnumerator()) {
        [IO.File]::WriteAllText((Join-Path $manifestDirectory $entry.Key), $entry.Value.TrimEnd() + "`n", [Text.UTF8Encoding]::new($false))
    }
    Write-Host "Manifests: $manifestDirectory"
    Write-Host "MSI SHA256: $sha256"
    Write-Host 'Next: validate the manifests, test installation as a standard user, then submit the directory to microsoft/winget-pkgs.'
}
finally {
    if ($null -ne $database) { $null = [Runtime.InteropServices.Marshal]::FinalReleaseComObject($database) }
    if ($null -ne $installer) { $null = [Runtime.InteropServices.Marshal]::FinalReleaseComObject($installer) }
    # Only remove the known downloaded file and its now-empty private directory.
    if (Test-Path -LiteralPath $msiPath) { Remove-Item -LiteralPath $msiPath }
    Remove-Item -LiteralPath $tempDirectory
}

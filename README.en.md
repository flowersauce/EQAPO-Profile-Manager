<p align="center"><img src="assets/icons/app/eqm-logo.svg" alt="EQM logo" width="120"></p>

<h1 align="center">EQAPO-Profile-Manager</h1>

<p align="center">Switch EQ profiles easily in any terminal, rather than opening Equalizer APO's Configuration Editor. Import preconfigured profiles, apply them as needed, disable EQ, or remove profiles you no longer use.</p>

<p align="center">
  <img alt="Windows" src="https://img.shields.io/badge/Windows-0078D4?style=flat-square&logo=windows&logoColor=white">
  <img alt="Go" src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="CLI" src="https://img.shields.io/badge/CLI-2D2D2D?style=flat-square&logo=terminal&logoColor=white">
  <img alt="License" src="https://img.shields.io/badge/License-MIT-2DA44E?style=flat-square">
</p>

<p align="center">
  <a href="README.md">中文</a> · English
</p>

## Features

EQM is for people who already have usable EQ profiles and want to switch between them. For everyday use, you can leave Equalizer APO's **Configuration Editor** closed: choose a profile in your terminal without adjusting filters or editing configuration files.

- **Switch quickly**: select and apply an imported profile when changing headphones or presets.
- **Turn EQ off**: choose `None` in the switch menu to disable EQ.
- **Import preconfigured profiles**: add GraphicEQ or common parametric EQ text files with any extension or no extension, preserving names and contents.
- **Organize profiles**: view your collection, rename or remove profiles, print their contents, or choose an application to open a file.
- **Terminal interaction**: arrow-key menus, Tab path completion, and Chinese or English selected from your system language. Installer and portable editions require no additional runtime.

Equalizer APO processes the audio; EQM imports and switches existing profiles. EQM does not generate equalizer settings. Complex configurations using Include, convolution, or other directives are currently not supported.

### Everyday use

```powershell
eqm switch
```

Use ↑/↓ or j/k to select a profile and Enter to apply it. Choose `None` to disable EQ, or press Esc to cancel. You can also use the short command `eqm s`. The selected item is cyan; `✓` marks the applied profile.

Use `eqm import` to add a profile, `eqm remove` to select and confirm deletion of an unwanted profile, and `eqm list` to view your collection. Before your first use, complete the initialization and import described below.

## Getting started

### Download and install

First, install and configure **Equalizer APO**, and prepare a compatible equalizer text profile. Files may have any extension or none.

Need a profile to try? Visit [AutoEQ](https://autoeq.app/), find your headphones, and export an Equalizer APO profile to use with EQM.

**[Download the latest release](https://github.com/flowersauce/EQAPO-Profile-Manager/releases/latest)** and choose:

| Edition                 | Getting started                                                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| Installer `.msi`        | Run the installer, then open a new terminal and run `eqm`. Installation needs no administrator privileges.                        |
| Portable `portable.zip` | Extract the archive, open a terminal in the `EQM` folder, and run `.\eqm.exe`. Keep the entire folder, including `portable.flag`. |

Use EQM from a terminal; installers do not create Start menu shortcuts. MSI and winget use the user PATH; the Store edition uses a system app execution alias. Uninstalling removes the corresponding PATH entry or alias and EQM settings: `%LOCALAPPDATA%\EQM` for MSI and package data for the Store edition. Upgrades keep settings. Uninstalling EQM leaves Equalizer APO's config folder untouched.

### First-time setup

```powershell
eqm init
```

Initialize when you first start using EQM: select the Equalizer APO installation folder and confirm that EQM can manage its configuration. Press Enter to accept the suggested default path.

Initialization preserves the original `config.txt` content as comments, so its previous settings become inactive. Read the confirmation prompt before proceeding. Changes to protected folders request administrator approval while keeping your interaction in the current terminal.

### Add a profile

```powershell
eqm import
```

Enter the path to an existing profile. Tab completes paths; the profile is imported directly into the management folder. Run `eqm switch` to select and apply it. Use the same import command whenever you add another profile.

For the portable edition, replace `eqm` with `.\eqm.exe`. `eqm list` prints the sorted profiles and exits.

All profiles live directly in `<APO>\config\eqm-profiles`, with no category folders. Use names such as `SHP9500 - Harman` to distinguish profiles. Existing folders and their contents are preserved but omitted from the list; import their profiles again to copy them into the root. A missing or legacy category selection displays `None`; the next mutation disables the old Include.

## Command reference

| Command | Alias | Action |
| --- | --- | --- |
| `eqm switch` | `s` | Select and apply a profile, or disable EQ |
| `eqm list` | `l` | List profiles; output filenames when redirected |
| `eqm import` | `im` | Import an existing profile |
| `eqm` | — | Show version, current status, and help |
| `eqm rename` | `rn` | Rename a profile |
| `eqm remove` | `rm` | Delete a profile |
| `eqm print` | `p` | Print profile contents |
| `eqm open` | `o` | Choose an application to open a profile |
| `eqm init` | `in` | Set the Equalizer APO folder for first-time use |

For example, `eqm s` runs `eqm switch`. The old `show` command was removed; use `print` or `p`. Arbitrary prefixes are not accepted.

## Build

Install Go 1.25 or later and PowerShell on Windows, manually install `go-winres` and put it on PATH, then run:

```powershell
git clone https://github.com/flowersauce/EQAPO-Profile-Manager.git
cd EQAPO-Profile-Manager
go install github.com/tc-hib/go-winres@v0.3.3
.\scripts\build-windows.ps1
.\eqm.exe
```

The script calls `go-winres` from PATH and embeds `internal/resources/icons/app/eqm.ico` in the executable. The SVG source is at `assets/icons/app/eqm-logo.svg`. A directly built executable stores settings in `%LOCALAPPDATA%\EQM`. For portable mode, create an empty `portable.flag` file next to the executable.

To produce an MSI installer and portable ZIP, also install PowerShell 7, the WiX CLI on PATH, its matching official Util extension, and the required .NET runtime. Packaging uses existing tools and does not install them:

```powershell
.\scripts\build-release.ps1 -Version 1.1.0
```

The release script calls the EXE build and packaging scripts; you do not need to run `build-windows.ps1` first or edit the default version in `main.go`. It writes ZIP, MSI, and SHA256 checksums to `output/public`. After all packages succeed, it replaces `output` with only the current artifacts. Rebuilding the same version is supported; failed builds keep the previous output, and staging files are cleaned automatically.

Add `-IncludeStore` to produce the Store MSIX in `output/store`, using the reserved `Flowersauce.EQM` identity. Version `1.1.0` becomes `1.1.0.0` in the manifest; the CLI and artifact filenames keep `1.1.0`. This additionally needs the Windows SDK and existing icon conversion tools. See [Windows release notes](docs/windows-release.md) and [MSIX notes](packaging/msix/README.md) for permissions approval, standalone packaging, and local verification.

## License

Created by [flowersauce](https://github.com/flowersauce) and released under the [MIT License](LICENSE). You are welcome to use, modify, and distribute it while retaining the copyright and license notices.

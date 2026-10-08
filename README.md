<p align="center"><img src="assets/icons/app/eqm-logo.svg" alt="EQM logo" width="120"></p>

<h1 align="center">EQAPO-Profile-Manager</h1>

<p align="center">在任意终端中轻松切换 EQ 配置，而不必打开 Equalizer APO 配置编辑器。导入预设配置，按需应用，也可停用 EQ 或移除不再需要的配置。</p>

<p align="center">
  <img alt="Windows" src="https://img.shields.io/badge/Windows-0078D4?style=flat-square&logo=windows&logoColor=white">
  <img alt="Go" src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="CLI" src="https://img.shields.io/badge/CLI-2D2D2D?style=flat-square&logo=terminal&logoColor=white">
  <img alt="License" src="https://img.shields.io/badge/License-MIT-2DA44E?style=flat-square">
</p>

<p align="center">
  中文 · <a href="README.en.md">English</a>
</p>

## 功能介绍

EQM 面向已经有可用均衡配置、希望按需切换的用户。日常使用时无需打开 Equalizer APO 的 **Configuration Editor** 调整滤波器或编辑配置文件：在终端中选择要用的配置即可。

- **快速切换**：从已导入的配置中选择并应用，方便在不同耳机或预设之间切换。
- **随时停用**：在切换菜单中选择 `None`，停用 EQ。
- **导入预设配置**：支持 GraphicEQ 与常见参数均衡文本，包括无后缀文件，原样保留名称和内容。
- **整理配置**：查看已导入的配置，按需重命名或删除；也可打印内容、选择应用打开文件。
- **终端交互**：方向键选择、Tab 路径补全，跟随系统使用中文或英文；提供安装版与便携版，无需额外安装运行环境。

Equalizer APO 负责音频处理，EQM 负责导入和切换已有配置，不生成均衡参数。当前暂不支持包含 Include、卷积等指令的复杂配置。

### 日常使用

```powershell
eqm switch
```

用 ↑/↓ 或 j/k 选择配置，按 Enter 应用；选择 `None` 可停用 EQ，Esc 取消。也可以输入简写 `eqm s`。选中项为青色，`✓` 标记已应用配置。

添加新配置用 `eqm import`；不再需要的配置用 `eqm remove` 选择并确认删除；查看已有配置用 `eqm list`。首次使用前，完成下方的初始化和首次导入即可。

## 开始使用

### 下载与安装

使用前，请先安装并配置好 **Equalizer APO**，准备一个可用的均衡文本配置。文件可以没有后缀，或使用 `.txt` 等其他后缀。

还没有配置？可以前往 [AutoEQ](https://autoeq.app/) 查找自己的耳机，导出适用于 Equalizer APO 的配置来体验 EQM。

前往 **[下载最新版本](https://github.com/flowersauce/EQAPO-Profile-Manager/releases/latest)**，选择：

| 版本                  | 如何使用                                                                          |
| --------------------- | --------------------------------------------------------------------------------- |
| 安装版 `.msi`         | 双击安装，然后打开新终端运行 `eqm`。安装无需管理员权限。                          |
| 便携版 `portable.zip` | 解压后在 `EQM` 目录打开终端，运行 `.\eqm.exe`。请保留整个目录及 `portable.flag`。 |

EQM 从终端使用，安装版不创建开始菜单快捷方式。普通安装版和 winget 使用用户 PATH；Store 使用系统执行别名。卸载清理对应 PATH 项或别名及 EQM 设置：普通安装版为 `%LOCALAPPDATA%\EQM`，Store 为包内设置。升级保留设置，卸载不修改 Equalizer APO 的 config 目录。

### 首次设置

```powershell
eqm init
```

初始化只需在开始使用时完成：选择 Equalizer APO 安装目录，确认由 EQM 管理配置。直接按 Enter 可采用提示中的默认目录。

初始化会将原 `config.txt` 内容保留为注释，原有配置不再生效，请留意确认提示。修改受保护目录时会请求管理员授权，完成后继续留在当前终端。

### 添加配置

```powershell
eqm import
```

输入现成配置文件的路径，支持 Tab 补全；配置直接导入管理目录。导入完成后，用 `eqm switch` 选择并应用。以后添加配置也使用这条导入命令。

便携版请把命令中的 `eqm` 替换为 `.\eqm.exe`。`eqm list` 直接输出按名称排序的配置并退出。

所有配置平面存放于 `<APO>\config\eqm-profiles`，不提供分类目录。可用 `SHP9500 - Harman` 等名称区分用途。已有子目录及文件会保留，但不纳入列表；可重新导入其中的配置，将其复制到根目录。当前配置文件丢失或引用旧版分类时显示 `None`，下次提交修改时禁用旧 Include。

## 命令参考

| 命令 | 缩写 | 功能 |
| --- | --- | --- |
| `eqm switch` | `s` | 选择并应用配置，或停用 EQ |
| `eqm list` | `l` | 列出配置；重定向时输出文件名 |
| `eqm import` | `im` | 导入现成配置 |
| `eqm` | — | 查看版本、当前状态和帮助 |
| `eqm rename` | `rn` | 重命名配置 |
| `eqm remove` | `rm` | 删除配置 |
| `eqm print` | `p` | 打印配置内容 |
| `eqm open` | `o` | 选择应用打开配置 |
| `eqm init` | `in` | 首次设置 Equalizer APO 目录 |

例如 `eqm s` 等同于 `eqm switch`。旧命令 `show` 已移除，使用 `print` 或 `p`；不支持任意前缀缩写。

## 构建

在 Windows 上安装 Go 1.25 或更新版本及 PowerShell，并手动安装 `go-winres`、将其加入 PATH，再运行：

```powershell
git clone https://github.com/flowersauce/EQAPO-Profile-Manager.git
cd EQAPO-Profile-Manager
go install github.com/tc-hib/go-winres@v0.3.3
.\scripts\build-windows.ps1
.\eqm.exe
```

脚本会调用 PATH 中的 `go-winres`，将 `internal/resources/icons/app/eqm.ico` 嵌入 exe；SVG 源文件位于 `assets/icons/app/eqm-logo.svg`。直接构建的程序将设置保存在 `%LOCALAPPDATA%\EQM`。如需便携模式，在 exe 同目录创建空的 `portable.flag` 文件。

生成 MSI 安装包和便携 ZIP 还需要 PowerShell 7，以及已手动安装并加入 PATH 的 WiX 命令行工具、同版本官方 Util 扩展和所需的 .NET 运行时。发布脚本直接调用 PATH 中的 `wix` 与已安装扩展，不自动安装工具：

```powershell
.\scripts\build-release.ps1 -Version 1.1.0
```

发布脚本会调用 EXE 构建和打包脚本，无须先运行 `build-windows.ps1`，也无须修改 `main.go` 的默认版本。ZIP、MSI 与 SHA256 校验文件输出至 `output/public`。全部打包成功后整体替换 `output`，只保留本次产物；支持同版本重新构建，失败保留上一份产物，暂存内容自动清理。

商店 MSIX 使用 `.\scripts\build-release.ps1 -Version 1.1.0 -IncludeStore`，额外需要 Windows SDK 和现有图标转换工具，输出到 `output/store`；商店身份已绑定 `Flowersauce.EQM`。清单包版本为 `1.1.0.0`，程序显示及文件名仍使用 `1.1.0`。权限审批、单独打包与本地验收见 [Windows 发布说明](docs/windows-release.md) 和 [MSIX 说明](packaging/msix/README.md)。

## 许可

由 [flowersauce](https://github.com/flowersauce) 开发，采用 [MIT License](LICENSE) 开源。欢迎使用、修改和分发，请保留原有版权与许可声明。

<p align="center"><img src="assets/icons/app/eqm-logo.svg" alt="EQM logo" width="120"></p>

<h1 align="center">EQAPO-Profile-Manager</h1>

<p align="center">轻量级 Equalizer APO 配置管理器。在终端中导入、切换和管理现成的 Equalizer APO 配置，无需反复手动编辑配置文件。</p>

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

- **导入配置**：管理适用于 Equalizer APO 的 `.txt` 配置，支持 GraphicEQ 与参数均衡配置，包括 AutoEQ 生成的文件。
- **快速切换**：在多个配置之间选择并应用，也可随时停用 EQ。
- **集中管理**：查看内容、选择应用打开、重命名或删除配置。
- **轻松交互**：方向键选择、Tab 路径补全，跟随系统使用中文或英文。
- **灵活使用**：提供安装版与便携版，无需额外安装运行环境。

EQM 负责管理已有配置，不生成均衡参数。当前支持 GraphicEQ 与常见参数均衡配置，暂不支持包含 Include、卷积等指令的复杂配置。

### 常用命令

| 命令         | 功能                     |
| ------------ | ------------------------ |
| `eqm`        | 查看版本、当前状态和帮助 |
| `eqm init`   | 设置 Equalizer APO 目录  |
| `eqm import` | 导入配置                 |
| `eqm switch` | 切换或停用配置           |
| `eqm list`   | 列出所有配置             |
| `eqm show`   | 查看配置内容             |
| `eqm open`   | 选择应用打开配置         |
| `eqm rename` | 重命名配置               |
| `eqm remove` | 删除配置                 |

## 开始使用

### 下载与安装

使用前，请先安装并配置好 **Equalizer APO**，准备一个可用的 `.txt` 均衡配置文件。

还没有配置？可以前往 [AutoEQ](https://autoeq.app/) 查找自己的耳机，导出适用于 Equalizer APO 的配置来体验 EQM。

前往 **[下载最新版本](https://github.com/flowersauce/EQAPO-Profile-Manager/releases/latest)**，选择：

| 版本                  | 如何使用                                                                          |
| --------------------- | --------------------------------------------------------------------------------- |
| 安装版 `.msi`         | 双击安装，然后打开新终端运行 `eqm`。安装无需管理员权限。                          |
| 便携版 `portable.zip` | 解压后在 `EQM` 目录打开终端，运行 `.\eqm.exe`。请保留整个目录及 `portable.flag`。 |

### 应用配置

```powershell
eqm init
eqm import
eqm switch
```

1. **初始化**：选择 Equalizer APO 安装目录，确认由 EQM 管理配置。直接按 Enter 可采用提示中的默认目录。
2. **导入**：输入配置文件路径，支持 Tab 补全。
3. **切换**：选择刚导入的配置并应用；选择 `None` 可停用 EQ。

便携版请把命令中的 `eqm` 替换为 `.\eqm.exe`。列表使用方向键选择，Enter 确认，Esc 取消。

初始化会将原 `config.txt` 内容保留为注释，原有配置不再生效，请留意确认提示。修改受保护目录时会请求管理员授权，完成后继续留在当前终端。

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

生成 MSI 安装包和便携 ZIP 还需要 PowerShell 7，以及可运行 WiX 6 的 .NET SDK / Runtime。先手动恢复仓库固定版本的 WiX 工具：

```powershell
dotnet tool restore
.\scripts\build-release.ps1 -Version 1.0.0
```

发布脚本会调用 EXE 构建和打包脚本，无须先运行 `build-windows.ps1`，也无须修改 `main.go` 的默认版本。产物与 SHA256 校验文件输出至 `dist`，不会覆盖同名文件。

## 许可

由 [flowersauce](https://github.com/flowersauce) 开发，采用 [MIT License](LICENSE) 开源。欢迎使用、修改和分发，请保留原有版权与许可声明。

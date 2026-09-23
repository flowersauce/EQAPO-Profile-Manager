# EQAPO-Profile-Manager

**EQM** 是面向 Windows 的轻量 Equalizer APO 配置管理器。通过命令 `eqm` 导入、切换和管理 AutoEQ 配置，无需反复手动编辑 `config.txt`。

- 支持 AutoEQ GraphicEQ 和 ParametricEQ 文本配置，导入时保留原始内容。
- 使用行内交互向导，支持中文路径、Tab 补全和行内候选预览。
- 根据 Windows 显示语言自动使用简体中文或英文，其他语言回退英文。
- 提供用户级 MSI 和便携 ZIP，运行时无需安装 Go 或 .NET。
- 普通操作保持普通权限；受限提交通过 UAC 在后台完成，交互继续留在原终端。

## 安装

发布包见 [GitHub Releases](https://github.com/flowersauce/EQAPO-Profile-Manager/releases)。尚无发布包时，可按下文从源码构建。使用前需自行安装并配置好 Equalizer APO；EQM 不负责安装音频驱动。

### MSI 安装版

安装 `EQM-1.0.0-windows-x64.msi` 后，打开新的终端即可使用 `eqm`。

- 安装目录：`%LOCALAPPDATA%\Programs\EQM`
- 用户设置：`%LOCALAPPDATA%\EQM\config.json`
- 安装本身无需管理员权限，仅维护当前用户的 PATH。
- 卸载移除程序及对应 PATH 项，保留用户设置和 APO 内的配置文件。

### 便携版

解压 `EQM-1.0.0-windows-x64-portable.zip`，进入 `EQM` 目录运行 `./eqm.exe`。请保留与程序同目录的 `portable.flag`；设置将写入自身的 `config/config.json`。

便携版不修改 PATH，移动时将整个 `EQM` 目录一并移动。两种版本的 EQ 文件都保存在 Equalizer APO 的 `config/eqm-profiles` 中。

winget 尚未收录；计划使用标识 `Flowersauce.EQAPOProfileManager`。

## 快速开始

在安装版的新终端中运行以下命令；便携版将 `eqm` 替换为 `.\eqm.exe`：

```powershell
eqm init
eqm import
eqm switch
```

1. `init`：指定 Equalizer APO 安装根目录，确认接管 `config.txt`。默认目录以占位提示显示，留空按 Enter 采用；已有设置时优先使用之前保存的目录。
2. `import`：输入或拖入一个 AutoEQ `.txt` 文件。导入不会自动应用，完成后可选择是否删除源文件。
3. `switch`：选择配置并应用，或选择 `None` 停用 EQ。

接管时，原 `config.txt` 内容会保留为注释，并由 EQM 的托管区域负责选择配置。修改前请阅读确认提示；具体文件结构见 [设计方案](docs/design.md)。

## 命令

| 命令 | 功能 |
| --- | --- |
| `eqm` | 显示问候、版本、作者、仓库、当前状态及命令帮助 |
| `eqm init` | 设置或更新 Equalizer APO 安装目录 |
| `eqm import` | 导入 AutoEQ 配置，重名时确认覆盖 |
| `eqm switch` | 切换配置，`None` 表示停用 |
| `eqm list` | 列出已管理的配置 |
| `eqm show` | 选择配置，将内容打印到终端 |
| `eqm open` | 选择配置，显示 Windows“打开方式”窗口 |
| `eqm rename` | 修改文件名，保留 `.txt` 后缀并更新相关引用 |
| `eqm remove` | 确认后删除配置，删除当前配置时停用 EQ |

所有子命令均不接受位置参数或 flag。除 `list` 和裸 `eqm` 外，命令需要交互终端。未初始化时，裸 `eqm` 仍会显示概览并提示运行 `eqm init`。

选择列表使用 `↑/↓` 或 `j/k` 移动，Enter 提交，Esc 取消。路径输入支持 Tab 补全；候选预览不会自动变为提交内容。输入步骤以 `>` 开头，选择/确认以 `?` 开头，`^` 为提示，`!` 为警告，`✗` 为错误，`✓` 为已完成或已应用。设置 `NO_COLOR` 可关闭颜色。

## 配置与权限

程序只在用户设置中保存 APO 安装根目录；配置列表直接读取 `config/eqm-profiles`，当前选择来自 APO `config.txt`，不维护额外索引。

源文件和待接管的 `config.txt` 支持 UTF-8（可带 BOM）。写入 APO 的程序注释固定为英文，原文和 Unicode 文件名保持不变。

修改受保护目录时，EQM 会请求 UAC，后台管理员进程仅执行已确认的提交。拒绝提权会在原终端提示；文件发生外部变化或操作部分完成时会明确反馈。`open` 调用系统打开方式窗口，外部应用的编辑和保存由该应用负责。

## 从源码构建

需要 Windows 和 Go 1.25 或更新版本：

```powershell
git clone https://github.com/flowersauce/EQAPO-Profile-Manager.git
cd EQAPO-Profile-Manager
go build -o eqm.exe ./cmd/eqm
.\eqm.exe
```

直接构建不生成 `portable.flag`，默认使用安装版设置目录。源码中的 `_test.go` 是回归测试，不会编入发布程序。开发验证命令：

```powershell
go test ./...
go vet ./...
```

## 构建发布包

另需 PowerShell 7 和能够运行 WiX 6 的 .NET SDK / runtime。仓库固定 WiX CLI 为 6.0.2，发布脚本会恢复本地工具：

```powershell
.\scripts\build-release.ps1 -Version 1.0.0 -MsiVersion 1.0.0
```

产物位于 `dist`：x64 MSI、带便携标记的 ZIP 和 SHA256 校验文件。脚本不安装、不上传或提交 winget，也不覆盖同名产物。版本、升级及手工验收流程见 [Windows 安装与发布](docs/windows-release.md)。

## 文档与许可证

- [设计方案](docs/design.md)：命令、交互、目录和产品边界。
- [实现备注](docs/tech-notes.md)：架构、文件处理、权限与测试。
- [Windows 安装与发布](docs/windows-release.md)：打包、升级、卸载和验收。

作者：[flowersauce](https://github.com/flowersauce)。项目使用 [MIT License](LICENSE)。

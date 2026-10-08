# EQM Microsoft Store / MSIX

商店展示图标和中英文介绍头图见 [素材清单](../../assets/microsoft-store/design.md)，与 MSIX 包内资源分开保存。

## 已预留身份

| 字段 | 值 |
| --- | --- |
| Package/Identity/Name | `Flowersauce.EQM` |
| Package/Identity/Publisher | `CN=4276B3BE-2A10-4D07-BF6B-2419400FDCF6` |
| Package/Properties/PublisherDisplayName | `Flowersauce` |
| Package Family Name | `Flowersauce.EQM_fw24f0by96da4` |
| Store ID | `9PBS8DSTDZGQ` |
| 商店地址 | https://apps.microsoft.com/detail/9PBS8DSTDZGQ |

标识来自 Partner Center；名称、Publisher 和应用 Id 在后续版本保持稳定。链接作为发布入口预留，尚未宣称产品已上架。

## 打包

手动安装 Windows SDK（需 `makeappx.exe`、`makepri.exe`）、uv 和 Python 3.10+。图标依赖 `resvg_py` 与 Pillow 由项目 `pyproject.toml` 和 `uv.lock` 管理，通过 uv 同步到项目 `.venv`，无需全局 pip 安装。`assets/icons/app/eqm-logo.svg` 是唯一图标源，EXE 的 ICO 和 MSIX 的 PNG 均由 `scripts/generate-app-icon.py` 从此 SVG 生成。

```powershell
uv sync --locked
# 修改 SVG 后更新 EXE 的 ICO；MSIX PNG 在打包时自动生成。
uv run --locked python scripts/generate-app-icon.py

# 同时生成公开 ZIP / MSI 和商店 MSIX。
.\scripts\build-release.ps1 -Version 1.1.0 -IncludeStore

# 已有 EXE 时单独生成商店包，不重做 MSI。
.\scripts\package-msix.ps1 -BinaryPath .\eqm.exe -Version 1.1.0 `
  -MsixPath .\output\store\EQM-1.1.0-windows-x64-store.msix `
  -IntermediateDir (Join-Path $env:TEMP ('eqm-msix-' + [guid]::NewGuid().ToString('N')))
```

默认选择已安装的最新 x64 Windows SDK；可传 `-SdkBin 'C:\Program Files (x86)\Windows Kits\10\bin\<版本>\x64'`、`-Uv <uv.exe 路径>` 和 `-Python <python.exe 路径>`。`-Python` 指定 uv 用于项目环境的解释器，不绕过锁文件。打包使用 `uv run --locked`，依赖声明与锁文件不一致时拒绝继续。Store 版本限三段数字、主版本大于零，清单的第四段固定为 `0`。每次提交递增版本；独立打包拒绝覆盖同名输出和已有暂存目录，发布脚本则在打包成功后整体替换 `output` 并自动清理暂存内容。

当前版本 `1.1.0` 对应清单 `Identity/Version="1.1.0.0"`；程序显示及 MSIX 文件名仍使用 `1.1.0`。这是 [Store 包版本规范](https://learn.microsoft.com/en-us/windows/apps/publish/publish-your-app/app-package-requirements?pivots=store-installer-msix) 对清单的要求，不要求文件名采用四段版本。

包仅含 EXE、许可、清单与由 SVG 生成的 PNG / PRI 资源，不含 `portable.flag`、用户设置、APO 或驱动。SDK 负责清单语义验证；此处生成供 Partner Center 上传的未签名 `.msix`。本地安装需要匹配清单 Publisher 的可信测试签名；脚本不创建证书、签名、安装或上传。Windows Desktop 最低版本为 10.0.19041.0。

包声明 console execution alias `eqm.exe`，安装后从新终端使用 `eqm`。Windows 在 `%LOCALAPPDATA%\Microsoft\WindowsApps` 管理别名，该目录通常已在用户 PATH 中；本包不追加 EQM 安装目录。别名可在系统“应用执行别名”设置中开关，正常卸载随包移除。若 MSI 与 MSIX 并存，用 `Get-Command eqm -All` 确认实际使用的版本。

EQM 从终端使用，清单设置 `AppListEntry="none"`，不创建开始菜单应用入口；执行别名继续有效。参见 [AppListEntry](https://learn.microsoft.com/en-us/uwp/schemas/appxpackage/uapmanifestschema/element-uap-visualelements) 和 [别名的 PATH 查找](https://learn.microsoft.com/en-us/windows/terminal/command-line-arguments#command-line-syntax)。

## 权限与商店审核

### 隐藏入口豁免

当前包已同时声明 `AppListEntry="none"` 和执行别名，但 Partner Center 上传验证仍返回 headless app / `HeadlessAppBypass` 错误。该错误要求商店为 EQM 产品授予隐藏入口豁免；执行别名不替代这项授权，`HeadlessAppBypass` 也不是需要添加到清单的 capability。本地 MakeAppx 验证和安装成功不代表商店已授予豁免。

通过 [Windows Developer Support](https://developer.microsoft.com/en-us/windows/support/) 的 Non-Technical Support 创建商店提交支持工单，申请为 Store ID `9PBS8DSTDZGQ` 启用 `HeadlessAppBypass`。附产品标识 `Flowersauce.EQM`、PFN `Flowersauce.EQM_fw24f0by96da4`、验证错误原文及截图。说明 EQM 是由用户在终端通过 `eqm` 调用的 CLI；隐藏开始菜单入口是为了避免直接启动后终端立即退出，本包不注册 Windows 服务或启动任务。微软工作人员对同类问题也建议交由商店团队通过 [支持工单](https://learn.microsoft.com/en-us/answers/questions/2115779/headlessappbypass-waiver) 处理。

### 按需提权审批

使用 `packagedClassicApp`、`mediumIL`，并声明 `runFullTrust` 与 `allowElevation`。EXE 默认普通权限；只有用户已经确认的文件提交遭遇访问拒绝时才启动隐藏 UAC 工作进程。工作进程不修改 ACL、不安装 APO、不加载驱动、不执行任意 shell。

`runFullTrust` 表示普通桌面程序能力；提权依赖 `allowElevation`。微软要求后者在上架前提交详细理由并按严格标准审批，参见 [官方能力声明](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/app-capability-declarations)。请开发者先向 `reportapp@microsoft.com` 申请，批准前不要将打包成功视为商店审核通过。

`HeadlessAppBypass` 与 `allowElevation` 分别解决隐藏应用入口和按需提权，需要分别确认授权。

申请理由可说明：EQM 是开源的 Equalizer APO 配置管理 CLI；默认普通权限，通常官方 APO 安装目录可写。对于保留了受限 ACL 的既有安装，用户明确确认后才请求 UAC，只修改选定 APO 的托管配置和配置文件；命名管道校验双方进程身份，提交前核对文件快照，退出后不保留后台服务。附源码地址、Store ID 和能复现访问拒绝的测试场景。

## 设置和验收

MSIX 设置位于原用户 `%LOCALAPPDATA%\Packages\Flowersauce.EQM_fw24f0by96da4\LocalState\EQM\config.json`；路径通过包身份检测，不依赖工作目录或写入重定向。提权上下文保留原用户目录和包身份，另一管理员账户确认 UAC 时也使用这一位置。不会迁移 MSI 或便携设置。

LocalState 属于包数据，正常卸载时由 Windows 清理；本包不启用 MSIX Persistent Identity，不把商店设置写到包外。升级保留设置，卸载清理 `%LOCALAPPDATA%\Packages\Flowersauce.EQM_fw24f0by96da4\LocalState\EQM`，不修改 APO 的 config 目录。参见 [MSIX LocalState 生命周期](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/msix-windows10-windows11#msix-persistent-identity)。

请在测试账户 / Sandbox 中验证：清单版本 `1.1.0.0`、图标、执行别名可从终端使用、开始菜单应用列表无 EQM 入口，普通权限的全部命令、平面配置列表，受限 APO ACL 下的 UAC，同账户及另一管理员账户，确认后外部变更的冲突反馈，升级后设置保留。卸载前在 LocalState/EQM 中放入测试文件及子目录，正常卸载后确认该目录及包的执行别名均已移除，并比较 APO config 目录内容与哈希，确认不变。现有日常使用的 APO 无需卸载重装。

参考：[执行别名](https://learn.microsoft.com/en-us/uwp/schemas/appxpackage/uapmanifestschema/element-uap5-appexecutionalias)、[包身份 API](https://learn.microsoft.com/en-us/windows/win32/api/appmodel/nf-appmodel-getcurrentpackagefamilyname)、[MakePri](https://learn.microsoft.com/en-us/windows/uwp/app-resources/makepri-exe-command-options)。

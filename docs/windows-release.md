# Windows 安装与发布

## 标识与目录

| 项目 | 值 |
| --- | --- |
| 项目名称 | EQAPO-Profile-Manager |
| 产品简称 / MSI 显示名 | EQM |
| MSI Manufacturer / winget Publisher | `flowersauce` |
| CLI | `eqm` |
| winget PackageIdentifier | `Flowersauce.EQAPOProfileManager` |
| 安装目录 | `%LOCALAPPDATA%\Programs\EQM` |
| 安装版配置 | `%LOCALAPPDATA%\EQM\config.json` |
| 便携版配置 | `<exe 目录>\config\config.json` |
| MSIX Identity | `Flowersauce.EQM` |
| MSIX 配置 | `%LOCALAPPDATA%\Packages\Flowersauce.EQM_fw24f0by96da4\LocalState\EQM\config.json` |
| Store ID | `9PBS8DSTDZGQ` |

三种包使用同一个可执行文件。启动时先检测包身份，MSIX 使用原用户包 LocalState；没有包身份时根据 exe 同目录 `portable.flag` 区分便携和 MSI。路径解析不创建文件，配置由 `init` 提交后保存。

标记仅由发布脚本在便携包暂存目录生成，不纳入源码仓库或 MSI。便携包不包含现有用户配置；其中的空 `config` 目录用于首次初始化。

## 构建发布包

构建环境：Windows x64、PowerShell 7、Go 1.25+、WiX CLI 及与其同版本的官方 Util 扩展，以及 WiX 所需的 .NET 运行时。当前本地验证使用已安装的 WiX 7.0.0 / Util 7.0.0。手动运行 `go install github.com/tc-hib/go-winres@v0.3.3` 并将可执行文件加入 PATH；构建和打包脚本只调用已安装工具，不自动下载工具或扩展。SVG 源文件位于 `assets/icons/app/eqm-logo.svg`，供 EXE 构建使用的 ICO 位于 `internal/resources/icons/app/eqm.ico`。用户运行 EQM 不需要 Go 或 .NET。

Python 图标工具由项目根目录的 `pyproject.toml` 和 `uv.lock` 管理，需要安装 uv 和 Python 3.10+。运行 `uv sync --locked` 在项目 `.venv` 中安装锁定的 `resvg_py` 与 Pillow；修改 SVG 后运行 `uv run --locked python scripts/generate-app-icon.py` 更新 ICO。EXE 构建使用仓库中的 ICO，MSIX 打包通过 `uv run --locked` 从 SVG 生成 PNG。

```powershell
.\scripts\build-release.ps1 -Version 1.1.0
```

正式版只需传入一次 `Version`：发布脚本将它写入 CLI、产物名称和 MSI 版本。`main.go` 的默认版本无需修改，也无需先运行 `build-windows.ps1`。预发布版必须额外传入独立的三段数字 `-MsiVersion`；每次 MSI 发布都必须使用递增的数字版本，不能让预发布版与正式版复用同一个 MSI 版本。UpgradeCode 与现有 EqmExecutable 组件 GUID 均固定，常规版本升级时不能重新生成。

当前发布版本为 `1.1.0`。Store 清单身份版本使用四段 `1.1.0.0`，第四段 `0` 为商店保留；程序显示与产物文件名仍使用三段版本，商店文件名为 `EQM-1.1.0-windows-x64-store.msix`。

输出目录：

```text
output/
├── public/
│   ├── EQM-<Version>-windows-x64-portable.zip
│   ├── EQM-<Version>-windows-x64.msi
│   └── EQM-<Version>-SHA256SUMS.txt
├── store/
│   └── EQM-<Version>-windows-x64-store.msix
└── winget-manifests/manifests/f/Flowersauce/EQAPOProfileManager/<Version>/
```

ZIP 顶层为 `EQM`，含 exe、标记及空配置目录；MSI 仅安装 exe，不含便携标记或用户配置。校验文件只列公开 ZIP / MSI；现有 MSI 文件名和 GitHub 下载地址保持兼容。

每次在仓库根部创建独立暂存目录，不读取本地 `config` 或其他用户数据。全部请求的打包成功后整体替换 `output`，只保留本次产物，旧版本、旧清单及暂存内容自动清理；未指定 `-IncludeStore` 时不保留旧商店包。同版本可以重新构建，构建失败保留上一份完整产物。脚本不安装、不发布、不提交 winget。

MSIX 是可选发布通道，增加 `-IncludeStore`：

```powershell
.\scripts\build-release.ps1 -Version 1.1.0 -IncludeStore
```

需手动安装 Windows SDK（makeappx / makepri）、uv 和 Python；图标依赖由 uv 按锁文件同步到项目 `.venv`，所有图标均以 `assets/icons/app/eqm-logo.svg` 为源。允许传入 `-SdkBin`、`-Uv`（uv 可执行文件路径）与 `-Python`（uv 选用的 Python 解释器）。商店包使用已预留身份、生成未签名 MSIX，保留按需 UAC 并声明 `allowElevation`。当前隐藏开始菜单入口的包在 Partner Center 遇到 `HeadlessAppBypass` 校验，需为该产品申请豁免；此授权与 `allowElevation` 提权审批独立。商店权限审批、单独打包、签名与验收见 [MSIX 说明](../packaging/msix/README.md)，本地打包或安装成功不代表获准上架。

## winget manifest

将最终 MSI 上传到对应 GitHub Release 后，在 Windows PowerShell 7 中运行（无需安装 wingetcreate 或 YAML 模块）：

```powershell
.\scripts\new-winget-manifest.ps1 -Version 1.1.0
```

脚本下载 `v<Version>` Release 中的 x64 MSI，只读提取元数据和 SHA256，核对版本、名称、发布者、架构和 ALLUSERS。输出位于 `output/winget-manifests/manifests/f/Flowersauce/EQAPOProfileManager/<Version>`，包含 version、installer、英文默认描述及中文描述四份 YAML。可传 `-OutputDirectory`，拒绝覆盖已有版本目录。脚本不构建、不安装、不验证或自动提交 PR，下载失败不改用本地包。

生成后手动执行：

```powershell
winget validate --manifest .\output\winget-manifests\manifests\f\Flowersauce\EQAPOProfileManager\1.1.0
winget install --manifest .\output\winget-manifests\manifests\f\Flowersauce\EQAPOProfileManager\1.1.0 --scope user
```

完成下述验收后，将生成的版本目录按相同路径放入 `microsoft/winget-pkgs` 的 fork 并发起 PR。不要把下载的 MSI 或整个 `output` 一并提交。

基于最终发布的 MSI 生成 manifest，不使用占位下载地址或校验值：

- `PackageIdentifier: Flowersauce.EQAPOProfileManager`；前缀大小写无需与 Publisher 相同，后续版本保持标识不变。
- `PackageName: EQM`、`Publisher: flowersauce`，与系统卸载信息一致；若使用完整项目名作为 PackageName，需通过 `AppsAndFeaturesEntries` 对齐安装记录。
- 显式填写 `InstallerType: wix` 和 `Scope: user`。
- 不填写 `InstallerLocale`；生成后检查并移除该字段，包括各 Installers 条目中的值。`PackageLocale` 描述 manifest 文本语言，应正常保留。
- 下载地址、SHA256、ProductCode 和 AppsAndFeaturesEntries 必须来自同一份最终 MSI，不从另一次构建复制。

提交前手动执行 `winget validate --manifest <目录>`，再在普通权限终端执行 `winget install --manifest <目录> --scope user`，确认无安装 UAC、用户目录和 PATH 正确。若未开启本地 manifest 功能，需先在管理员终端执行一次 `winget settings --enable LocalManifestFiles`；这与 MSI 本身的安装权限无关。

首次收录前的发布整理允许同版本重新打包：执行脚本，更新 Release 资源及校验文件后才生成 manifest。已安装旧 EQM 包的测试环境应先卸载旧包，不以同版本覆盖安装验证升级。后续正式更新递增版本，已提交 manifest 引用的资源保持不变。

参考：[winget manifest 编写与测试](https://github.com/microsoft/winget-pkgs/blob/master/doc/Authoring.md)。

## 安装、升级与卸载

WiX 使用 `Scope="perUser"`，程序文件写入当前用户目录，注册表组件键位于 HKCU。没有提升权限的自定义操作，也不修改系统 PATH。Windows Installer 自行维护卸载注册信息。

MSI 使用 `Language="0"` 声明语言中立，不限定为英文安装包；这不改变 EQM 根据系统语言切换中英文的行为。参见 [Windows Installer 语言中立声明](https://learn.microsoft.com/windows/win32/msi/localizing-a-windows-installer-package)。

PATH 使用 MSI Environment 表追加 `%LOCALAPPDATA%\Programs\EQM` 对应的绝对路径，`Part="last"`、`System="no"`、`Permanent="no"`。卸载移除该项，保留其他 PATH 项。安装后打开新的终端使用 `eqm`，已有终端可能仍持有旧环境。

MSI / winget 不创建开始菜单快捷方式。Windows 搜索仍可能把 PATH 中的 `eqm.exe` 显示为“运行命令”；该结果不属于安装器注册的快捷方式。MSIX 使用系统执行别名并设置 `AppListEntry="none"` 隐藏开始菜单应用入口，卸载时由 Windows 移除包别名。应用执行别名需要包身份，普通 MSI 沿用用户 PATH，不额外注册身份包。参见 [Windows 包扩展](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/desktop-to-uwp-extensions)。

普通 MSI 与 winget 共用卸载规则：移除安装文件、空安装目录和对应 PATH 项，并递归删除当前用户 `%LOCALAPPDATA%\EQM` 配置目录及其内容。WiX Util `RemoveFolderEx` 使用固定私有路径，条件为 `REMOVE~="ALL" AND NOT UPGRADINGPRODUCTCODE`；升级和修复保留设置。升级使用 `MajorUpgrade`，旧版本移除位于安装事务内。不要以另一个用户的管理员身份安装用户级 MSI。

Store 的 EQM 设置仅存于包 LocalState；正常卸载由 Windows 清理，不启用 MSIX Persistent Identity，也不将商店设置写到包外。所有安装版卸载均不修改 Equalizer APO 的 config 目录、EQ 文件或托管配置。参见 [WiX 递归目录清理](https://docs.firegiant.com/wix/schema/util/removefolderex/) 与 [MSIX LocalState 生命周期](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/msix-windows10-windows11#msix-persistent-identity)。

## 按需 UAC

程序默认普通权限运行。已确认的提交操作遇到 `ACCESS_DENIED` 或 `PRIVILEGE_NOT_HELD` 时，通过 `ShellExecuteExW` 的 `runas` 与 `SW_HIDE` 启动后台管理员工作进程。原终端继续承载向导与结果，不打开可见的管理员终端，不要求重复输入或确认。

- 使用绝对 exe 路径及 Windows 参数转义，不经过 cmd 或 PowerShell；保留原命令参数和工作目录。
- 固定别名在提权前规范化为完整命令。私有启动上下文保留原用户 LocalAppData、包身份、界面语言、随机管道名称及父 PID；仅管理员进程接受。业务操作通过限制 ACL、拒绝远程连接且双向验证 PID 的命名管道传递。
- 工作进程只执行已经确认的业务提交，重新核对文件身份与字节后调用原有 core 操作；UAC 等待期间文件发生变化时提示冲突。
- 原向导收到成功结果后继续下一步；同一命令复用管理员进程，结束后关闭连接，不留下后台服务。
- 拒绝 UAC 显示黄色取消提示并立即结束当前命令，退出码为 `2`；已提权仍被拒绝访问时停止，不循环提权。
- 部分完成的事务不自动重放；通信断开且结果未知时提示先检查状态。初始化之前的读取、校验错误仍在原终端反馈。
- 非交互环境不弹 UAC 或等待输入；若操作受限，提示在交互终端中重试。

## 本地验收

项目允许自动构建、测试和打包；安装验证需明确指示。常规检查：

```powershell
go test ./...
go vet ./...
```

安装包和 UAC 涉及真实系统状态，应使用测试账户或 Windows Sandbox 验证：

1. 标准用户安装 MSI 无 UAC；文件、卸载项与 PATH 均属于当前用户，系统 PATH 不变。
2. 新终端可找到 `eqm`；初始化后配置写入 `%LOCALAPPDATA%\EQM`，安装目录没有 `config.json`。
3. 便携包有标记且使用自身配置；MSI 和源码仓库没有标记；不触发旧配置自动迁移。
4. 可写 APO 目录不提权，包括官方安装器正常授予 Users 访问权限的默认 Program Files 配置目录；受限 ACL 的测试目录在已确认提交遭遇拒绝后才弹 UAC。确认后原终端完成，无可见新窗口或重复问题；拒绝后立即结束当前命令并返回取消状态。同账户和另一管理员账户均需验证，日常 APO 无需卸载重装。
5. 验证初始化、切换、导入及源清理、改名、删除；UAC 等待时外部改文件应提示冲突。工作进程退出、原终端关闭、已提权仍不可写及部分完成失败均不自动重放或循环提权。
6. 升级和修复后 exe / PATH 正常、EQM 设置保留；分别通过普通 MSI 和 winget 卸载，确认 `%LOCALAPPDATA%\EQM` 整个目录（含测试子目录及文件）被删除，对应 PATH 项移除。卸载前后比较 APO config 目录内容与哈希，确认不变。
7. 验证直接导入、按名称排序的平面列表、文件改名 / 删除；确认后文件身份或内容变化必须阻止提交。用 `s`、`im`、`rn`、`rm` 重复受限 ACL 测试，确认缩写提权成功。子目录不纳入管理，文件移出根目录后只读显示 None，下一次修改禁用旧引用；旧版分类文件可重新导入且默认保留源文件。
8. MSIX 额外按 [MSIX 验收说明](../packaging/msix/README.md) 检查执行别名、LocalState、包升级和卸载。真实 APO 重载行为由本地验证确认。

参考：[命名管道客户端 PID](https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-getnamedpipeclientprocessid)、[WiX 安装范围](https://docs.firegiant.com/wix/schema/wxs/packagescopetype/)、[WiX Environment](https://docs.firegiant.com/wix/schema/wxs/environment/)、[Windows Installer Environment 表](https://learn.microsoft.com/windows/win32/msi/environment-table)、[ShellExecuteEx 参数](https://learn.microsoft.com/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow)。

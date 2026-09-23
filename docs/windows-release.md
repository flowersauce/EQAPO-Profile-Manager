# Windows 安装与发布

## 标识与目录

| 项目 | 值 |
| --- | --- |
| 项目名称 | EQAPO-Profile-Manager |
| 产品简称 / MSI 显示名 | EQM |
| CLI | `eqm` |
| winget PackageIdentifier | `Flowersauce.EQAPOProfileManager` |
| 安装目录 | `%LOCALAPPDATA%\Programs\EQM` |
| 安装版配置 | `%LOCALAPPDATA%\EQM\config.json` |
| 便携版配置 | `<exe 目录>\config\config.json` |

两种包使用同一个可执行文件。启动时检测 exe 同目录的 `portable.flag`，存在时使用便携目录，否则使用安装版配置目录。路径解析不创建文件；配置由 `init` 提交后保存。程序目录与安装版用户配置分离。

标记仅由发布脚本在便携包暂存目录生成，不纳入源码仓库或 MSI。便携包不包含现有用户配置；其中的空 `config` 目录用于首次初始化。

## 构建发布包

构建环境：Windows x64、PowerShell 7、Go 1.25+、能够运行 WiX 6 的 .NET SDK / runtime。WiX CLI 固定为 6.0.2，由仓库的 `.config/dotnet-tools.json` 管理，发布脚本调用 `dotnet tool restore` 获取它。用户运行 EQM 不需要 Go 或 .NET。

```powershell
.\scripts\build-release.ps1 -Version 1.0.0 -MsiVersion 1.0.0
```

`Version` 写入 CLI。`MsiVersion` 是独立的三段数字版本，必须随每次 MSI 发布递增；正式版与 CLI 版本保持一致，首版均为 `1.0.0`。预发布后缀不能直接用于 MSI，也不要给连续发布包复用相同 MSI 版本。UpgradeCode 与现有 EqmExecutable 组件 GUID 均固定，常规版本升级时不能重新生成。

输出至 `dist`：

- `EQM-<Version>-windows-x64-portable.zip`，顶层目录为 `EQM`，含 exe、标记及空配置目录。
- `EQM-<Version>-windows-x64.msi`，仅安装 exe，不含便携标记或用户配置。
- `EQM-<Version>-SHA256SUMS.txt`。

每次使用新的暂存目录，不读取本地 `config` 或其他用户数据，不自动删除暂存内容，也不覆盖已有同名发布文件。编译与 MSI 打包都成功后才将产物移至 `dist` 顶层。脚本只构建，不安装、不发布、不向 winget 提交。

winget 后续提交使用上述 PackageIdentifier、`InstallerType: wix` 和 `Scope: user`；下载地址与校验值必须对应实际发布的 MSI。当前不生成带占位地址或虚假校验值的 manifest。

## 安装、升级与卸载

WiX 使用 `Scope="perUser"`，程序文件写入当前用户目录，注册表组件键位于 HKCU。没有提升权限的自定义操作，也不修改系统 PATH。Windows Installer 自行维护卸载注册信息。

PATH 使用 MSI Environment 表追加 `%LOCALAPPDATA%\Programs\EQM` 对应的绝对路径，`Part="last"`、`System="no"`、`Permanent="no"`。卸载移除该项，保留其他 PATH 项。安装后打开新的终端使用 `eqm`，已有终端可能仍持有旧环境。

卸载移除安装文件及空安装目录；保留 `%LOCALAPPDATA%\EQM` 用户配置和 APO 内的 EQ 文件及托管配置。升级使用 `MajorUpgrade`，旧版本移除位于安装事务内。不要以另一个用户的管理员身份安装用户级 MSI。

## 按需 UAC

程序默认普通权限运行。已确认的提交操作遇到 `ACCESS_DENIED` 或 `PRIVILEGE_NOT_HELD` 时，通过 `ShellExecuteExW` 的 `runas` 与 `SW_HIDE` 启动后台管理员工作进程。原终端继续承载向导与结果，不打开可见的管理员终端，不要求重复输入或确认。

- 使用绝对 exe 路径及 Windows 参数转义，不经过 cmd 或 PowerShell；保留原命令参数和工作目录。
- 私有启动上下文保留原用户 LocalAppData、界面语言、随机管道名称及父 PID；仅管理员进程接受。具体业务操作通过限制 ACL、拒绝远程连接且双向验证 PID 的命名管道传递。
- 工作进程只执行已经确认的业务提交，重新核对文件身份与字节后调用原有 core 操作；UAC 等待期间文件发生变化时提示冲突。
- 原向导收到成功结果后继续下一步；同一命令复用管理员进程，结束后关闭连接，不留下后台服务。
- 拒绝 UAC 在原步骤提示，可重试或取消；已提权仍被拒绝访问时停止，不循环提权。
- 部分完成的事务不自动重放；通信断开且结果未知时提示先检查状态。初始化之前的读取、校验错误仍在原终端反馈。
- 非交互环境不弹 UAC 或等待输入；若操作受限，提示在交互终端中重试。

## 本地验收

按项目工作流手动执行构建、测试和安装验证：

```powershell
go test ./...
go vet ./...
```

安装包和 UAC 涉及真实系统状态，应使用测试账户或 Windows Sandbox 验证：

1. 标准用户安装 MSI 无 UAC；文件、卸载项与 PATH 均属于当前用户，系统 PATH 不变。
2. 新终端可找到 `eqm`；初始化后配置写入 `%LOCALAPPDATA%\EQM`，安装目录没有 `config.json`。
3. 便携包有标记且使用自身配置；MSI 和源码仓库没有标记；不触发旧配置自动迁移。
4. 可写 APO 目录不提权；默认 Program Files 目录在确认接管后弹 UAC，确认后原终端直接完成，不出现可见新窗口或重复问题；拒绝后原步骤可重试或取消。同账户和另一管理员账户均需验证。
5. 验证初始化、切换、导入及源清理、改名、删除；UAC 等待时外部改文件应提示冲突。工作进程退出、原终端关闭、已提权仍不可写及部分完成失败均不自动重放或循环提权。
6. 升级后 exe 更新、PATH 不重复、配置保留；卸载只移除对应 PATH 项，不删除用户配置或 APO 文件。

参考：[命名管道客户端 PID](https://learn.microsoft.com/windows/win32/api/winbase/nf-winbase-getnamedpipeclientprocessid)、[WiX 安装范围](https://docs.firegiant.com/wix/schema/wxs/packagescopetype/)、[WiX Environment](https://docs.firegiant.com/wix/schema/wxs/environment/)、[Windows Installer Environment 表](https://learn.microsoft.com/windows/win32/msi/environment-table)、[ShellExecuteEx 参数](https://learn.microsoft.com/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow)。

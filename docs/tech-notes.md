# 实现备注（不参与方案讨论）

本文件保存「怎么做」层面的细节，供真正动手时参考。
产品层面的定义见 `docs/design.md`，待定问题见 `docs/open-questions.md`。

---

## 1. 技术选型

| 用途 | 选择 | 说明 |
| --- | --- | --- |
| 语言 | Go（当前最新稳定版 1.27.1；`go.mod` 记录实际版本） | 单文件静态 exe，`CGO_ENABLED=0` |
| 行内界面 | `bubbletea` **v2**（模块路径 `charm.land/bubbletea/v2`，要求 Go ≥1.25）+ `lipgloss` | 默认行内渲染，非全屏；按键、raw mode、resize、帧差分都由框架处理 |
| JSON | `encoding/json`（stdlib） | `config.json` / `profile.json` |
| 参数解析 | `flag`（stdlib）+ 手写子命令分发 | 不引入 cobra |
| 数据目录 | `init` 时由用户指定，默认 `os.UserCacheDir()`（Windows 上即 `%LocalAppData%`），可被 `EQM_HOME` 覆盖 | |
| 路径 / Unicode | `os` / `path/filepath`（stdlib） | Windows 上 stdlib 内部走 UTF-16 |
| 注册表探测 | `golang.org/x/sys/windows/registry` | 只读 |
| 控制台代码页 | `golang.org/x/sys/windows` 的 `SetConsoleOutputCP` | 另有 `MoveFileEx` 与 `MOVEFILE_*` |
| 测试 / 格式化 | `testing` / `gofmt` / `go vet` | 零配置 |
| 界面语言 | 系统语言用 `x/sys/windows` 的 `GetUserPreferredUILanguages`；文案用 `//go:embed` 内嵌中英两份 map | 见 §13 |

取舍：`bubbletea` 会带进一串传递依赖（`termenv`、`go-runewidth`、`cancelreader` 等），
这是本项目唯一的依赖树，换来的是 Windows 控制台按键、raw mode、resize、帧差分全部现成。

不引入：原生文件对话框绑定、cobra / urfave-cli、日志框架、数据库、GUI 框架。

---

## 2. 代码结构

```text
AutoEQ-APO-Manager/
├── go.mod / go.sum
├── docs/
│   ├── design.md            # 产品方案
│   ├── tech-notes.md        # 本文件
│   └── open-questions.md
├── cmd/eqm/main.go          # 入口：参数分发
├── internal/
│   ├── core/                # 纯逻辑，不依赖终端，可单测
│   │   ├── appconfig.go     # config.json 读写与默认值
│   │   ├── profile.go       # Profile 元数据模型
│   │   ├── store.go         # 库：列举 / 查找 / 导入 / 删除 / 重名处理
│   │   ├── eqparser.go      # 格式识别与轻量校验（只读不改）
│   │   ├── naming.go        # 文件名 → 显示名 / 目录 slug
│   │   ├── apobridge.go     # 托管 config.txt / 写 eqm\current.txt / 备份 / 覆盖替换
│   │   ├── render.go        # 「Profile 列表 → []string」纯函数，交互与 --plain 共用
│   │   └── paths.go         # 数据目录与 APO 目录解析（平台判断集中处）
│   ├── ui/                  # 行内交互界面
│   │   ├── select.go        # Model：items/cursor/keymap/filter
│   │   ├── confirm.go       # Y/n 单键确认
│   │   ├── input.go         # 单行输入 + Tab 补全 + 粘贴 chip
│   │   ├── paste.go         # 解析拖拽/粘贴文本 → 候选路径（纯函数，可单测）
│   │   ├── styles.go        # lipgloss 样式与按列宽截断
│   │   └── runner.go        # TTY 判断与降级为纯文本输出
│   └── cli/
│       ├── commands.go      # switch / import / remove / init（+ show）
│       └── dispatch.go      # flag.NewFlagSet 装配
└── testdata/fixtures/       # 真实 AutoEQ 样例：GraphicEQ / ParametricEQ / BOM / LF / 乱码
```

约束：

- `internal/core` 不依赖终端库，不读写 `os.Stdin` / `os.Stdout`，函数只接受显式路径参数。
- `internal/ui` 只做「把一组字符串渲染成可选择的列表」，业务规则不在这里。
- 平台判断只允许出现在 `core/paths.go` 与 `ui/runner.go`。
- bubbletea 的 `Model` 只持有字符串 / 整数状态，不持有文件句柄。

---

## 3. 数据文件与目录

```text
<存放目录>\                            # init 时由用户指定；默认 %LOCALAPPDATA%\AutoEQ-APO-Manager；可用 EQM_HOME 覆盖
├── config.json
└── profiles/
    ├── philips-shp9500/
    │   ├── profile.json
    │   └── eq.txt                     # AutoEQ 原始内容，原样保存
    └── moondrop-aria-2/

<APO config 目录>\                     # 从注册表 ConfigPath 读，或用户指定
├── config.txt                         # eqm 托管的入口，固定不变
└── eqm\
    └── current.txt                    # 指向当前生效配置，切换时重写
```

`config.json`：

```json
{
  "version": 1,
  "apo_config_dir": "D:/APP/Tools/EqualizerAPO/config",
  "current": "philips-shp9500"
}
```

（原来的 `runtime_mode` 字段已取消：指针文件必须在 APO 的 config 目录树内才能触发自动重载，
见 4.3。）

`profile.json`：

```json
{
  "name": "Philips SHP9500",
  "format": "GraphicEQ",
  "source": "AutoEQ",
  "file": "eq.txt",
  "imported_at": "2026-09-22T11:25:00Z",
  "source_filename": "Philips SHP9500 GraphicEq.txt"
}
```

约定：JSON 里的路径统一 UTF-8 + 正斜杠；写出的文本文件用 UTF-8 无 BOM + CRLF；
读入时容忍 BOM、LF/CRLF 与行尾空白。

---

## 4. Equalizer APO 接入

### 4.1 托管 `config.txt`

```txt
# 此文件由 AutoEQ-APO-Manager 自动管理，请勿手动修改。
# Managed by AutoEQ-APO-Manager. Do not edit manually.
Include: eqm\current.txt
```

首次接管：

1. 不存在 → 直接创建。
2. 已托管（含托管标记）→ 直接覆盖。
3. 未托管 → 先备份为 `config.txt.eqm.bak`，再询问是否把原内容导入为一个 Profile
   （默认名 `Existing config`），最后写入托管内容。

### 4.2 `eqm\current.txt`（放在 APO 的 config 目录树内）

切换时写入：

```txt
# 此文件由 AutoEQ-APO-Manager 自动管理，请勿手动修改。
Include: D:\EQM\profiles\philips-shp9500\eq.txt
```

原声模式写：

```txt
# Original
```

### 4.3 为什么这个指针文件必须在 APO 的 config 目录里（已核实）

Equalizer APO 的配置重载机制（源码 `FilterEngine::initialize` / `notificationThread`）：

- 它用 `FindFirstChangeNotificationW(configPath, true, FILE_NOTIFY_CHANGE_FILE_NAME |
  FILE_NOTIFY_CHANGE_LAST_WRITE)` 监视 **`ConfigPath` 整棵子树**（第二个参数为 `true`
  表示递归），有约 10 ms 的去抖。
- 一旦有变化就 `loadConfig()` **整份重读**，包括所有 `Include:`。
- **关键限制**：监视范围仅限 `ConfigPath` 子树。`Include:` 指向目录外的文件，
  改动它**不会**触发自动重载。

因此：

- `eqm\current.txt` 必须放在 `ConfigPath` 里，它每次切换都会变，是触发重载的关键；
- 指向库里的 `eq.txt` 用**绝对路径**没问题 —— 只要指针文件本身变了，APO 就会整份重读，
  顺带读到目录外的 `eq.txt`；
- 所以切换流程 = 只重写 `eqm\current.txt` 一个文件。

**代价**：需要写 APO 的 config 目录。默认安装位于 `C:\Program Files\EqualizerAPO\config`，
意味着常用操作需要管理员权限（APO 自带的编辑器也是这个处境）。`init` 会把
「config 目录能否写入」当作必检项，不能写就当场说明。

### 4.4 `Include:` 的其他行为（已核实）

| 事实 | 影响 |
| --- | --- |
| 语法是 `Include: <路径>`，**只去掉行首空白，不解析引号** | 路径含空格可以直接写，不要加引号；写文件时**不要带行尾空格**（会被当作路径的一部分） |
| 相对路径按**包含该行的文件所在目录**解析（不是根 config 目录） | `config.txt` 里的 `Include: eqm\current.txt` 指向 `<ConfigPath>\eqm\current.txt` |
| **支持嵌套 Include**（被包含的文件里还能有 `Include:`），深度上限 `RECURSION_LIMIT = 100`，超限静默跳过；**没有环检测** | 我们只用 2 层（`config.txt` → `eqm\current.txt` → 库里的 `eq.txt`），远低于上限；但仍应避免写出会自我引用的文件 |
| 路径用 `CreateFileW` 打开，未加 `\\?\` 前缀，相对路径在 `MAX_PATH`（260）缓冲里拼 | 路径总长应保持在 260 字符内 |
| 行编码：先按 UTF-8 解，出现替换字符 `U+FFFD` 再按系统 ANSI 解码；**不支持 UTF-16** | 我们写 UTF-8（无 BOM）即可；含非 ASCII 的库路径有较大机会正常工作，但仍列在待实测里 |
| **没有 BOM 剥离逻辑** | 带 BOM 时 BOM 会粘在第一行的命令名上，导致该行被静默忽略。所以**必须写无 BOM 的 UTF-8** |
| 无法识别的行、未知命令、不匹配 `命令: 参数` 的行都被静默忽略 | 出错时不会有任何提示，写文件要自己保证正确 |
| 文件大小与行长无限制 | 不需要担心 GraphicEQ 那种长行 |

### 4.5 写入可靠性

1. 在同一目录写临时文件（如 `current.txt.eqm-tmp`，不用 `*.txt` 后缀，避免被 APO 的文件变化监视当成真配置）。
2. 用 `os.Rename` 覆盖目标（Windows 上底层是 `MoveFileEx(..., MOVEFILE_REPLACE_EXISTING)`）。
   注意 `MoveFileEx` 不承诺原子性，目标只是「避免 APO 读到半截内容」。
3. 若实测仍会被读到中间状态，改用 `windows.MoveFileEx` + `MOVEFILE_WRITE_THROUGH`。

不做：文件锁、修改权限、后台守护进程、实时监控。

### 4.6 探测 APO 位置

`init` 里让用户填的是 APO 的**安装根目录或 config 目录，两者都接受**，程序自己判断：

1. 环境变量 `EQM_APO_CONFIG_DIR`；
2. 注册表 `HKLM\SOFTWARE\EqualizerAPO`（只读，已核实键名与值名）：
   `ConfigPath` 就是 config 目录（APO 读的是 `<ConfigPath>\config.txt`），
   另有 `InstallPath`、`EnableTrace`；读取要用 64 位视图（`KEY_WOW64_64KEY`）；
3. 常见路径如 `C:\Program Files\EqualizerAPO`；
4. 用户手输。

归一化与校验规则（全部在 `init` 输入当下完成）：

- 输入目录下直接有 `config.txt` → 它就是 config 目录；
- 输入目录下有 `config\config.txt` → 取它的 `config` 子目录；
- 两者都不满足 → 报错并让用户重新输入；
- 路径含非 ASCII 字符 → 警告（可能让 APO 读不到配置）并建议换路径；
- `config.txt` 不可写 → 提示需要管理员权限。

另外 `init` 第二项是 **eqm 自己的存放目录**（默认 `%LOCALAPPDATA%\AutoEQ-APO-Manager`），
用户可自定义。它只影响 `profiles/` 与 `config.json` 的位置（`eqm\current.txt` 在 APO 那侧），
因此改存放目录只需要重跑一次 `init`（重新生成 `config.txt` 里的 `Include:` 行）。

### 4.7 自愈规则

以下情况不报错、不问用户，直接修正并打一行提示（对应 `design.md` 第 6 节原则 6）：

| 情况 | 处理 |
| --- | --- |
| `current` 指向的配置目录或 `eq.txt` 不存在 | 将 `current` 置为 `null`，把 `eqm\current.txt` 重写为 `# Original`，提示「配置 X 的文件已丢失，已回到原声」 |
| `config.txt` 被用户改乱（缺少托管标记或 `Include:` 行） | 重新生成托管内容 |
| `config.json` 缺失但存放目录里已有 `profiles/` | 提示未初始化，引导重跑 `init`（不静默重建） |
| 导入时文档夹的源文件已被删 | 不影响（内容已存入库） |

`doctor` / `check` 不再作为独立命令；它的职责已经拆到上面（`init` 当场校验 + 自愈）。
「怀疑出问题」时的做法是重跑 `eqm init`。

---

## 5. 格式识别与校验

| 判定 | 条件 | 结果 |
| --- | --- | --- |
| GraphicEQ | 存在 `GraphicEQ:` 行且其后有 `频率 增益; ...` 数据 | `format = "GraphicEQ"` |
| ParametricEQ | 存在匹配 `Filter\s*\d*\s*:` 的行 | `format = "ParametricEQ"` |
| 两者都有 | 优先 ParametricEQ，并记一条警告 | 取一 |
| 都没有 | 拒绝导入 | 见下 |

容忍：任意数量 `Preamp:` 行、`#` 注释、空行；关键字不区分大小写。

只做「是不是受支持的配置」加结构性检查；不校验增益范围与滤波器合法性，不重算参数，
**不做自动修复**。

拒绝时输出：

```text
无法导入该文件。

未检测到受支持的 Equalizer APO AutoEQ 配置。

支持：
- GraphicEQ
- ParametricEQ
```

---

## 6. 命名与 slug

显示名推断（去掉扩展名后剥掉结尾的格式标记，不区分大小写，允许空格/下划线/连字符）：

```text
Philips SHP9500 GraphicEq.txt     → Philips SHP9500
Moondrop Aria 2 GraphicEq.txt     → Moondrop Aria 2
Philips SHP9500 ParametricEQ.txt  → Philips SHP9500
```

剥完为空或过短时询问用户。

目录 slug：转小写；ASCII 字母数字与 `.`、`_` 保留；其余折叠为单个 `-`；去首尾 `-`；
限长 64；冲突追加 `-2`、`-3`；slug 为空时退化为 `profile-<8位短哈希>`。
slug 只作为唯一标识，不要求与显示名一致。

重名比较的是显示名（大小写不敏感），默认不静默覆盖，给出「覆盖 / 重命名导入 / 取消」。

---

## 7. 行内界面实现要点

### 7.1 形态

- 提示渲染在滚动记录里，**不设 `View.AltScreen = true`**（一旦开启，`tea.Println` 也会失效）。
- 每一行都按终端宽度截断，绝不产生自动换行（否则重绘行数算错，是这类界面最典型的 bug）。
- 不画边框，只有「选中标记 + 文本」。
- 收尾必须显式：退出时返回空 `View()`，或 `tea.Sequence(tea.ClearScreen, tea.Quit)`，
  结果用 `tea.Println(...)` 写出去（依赖框架自动清屏是不可靠的）。

### 7.2 `select` 组件

| 部分 | 内容 |
| --- | --- |
| Model | `items []string`、`cursor int`、`filter string`、`chosen int`（-1 = 取消）、`width int` |
| Update | `up`/`k`、`down`/`j`、`home`/`g`、`end`/`G`、`enter`、`esc`/`ctrl+c`、数字键、可打印字符（过滤）、`backspace` |
| View | 前缀 `❯ ` 标当前项，`*` 标生效项；按 `width` 截断 |
| 结束 | `enter` → `chosen = cursor` + `tea.Quit`；`esc` → `chosen = -1` + `tea.Quit` |

- 列表内容由 `core/render.go` 的纯函数产生，`--plain` 与交互模式共用同一份行。
- `items` / `cursor` / `filter` / `chosen` 可作为可读字段供测试断言。

### 7.3 非交互降级（`ui/runner.go`）

```go
func interactive() bool {
    fi, err := os.Stdout.Stat()
    return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
```

- 为假或 `TERM=dumb`：能纯输出就纯输出（`switch` / `switch --plain` / `show`，退出码 0）；
  确实缺必需参数时（管道里的 `import` / `switch <名字>`）报错说明并退出 1。
- 官方形态是 `term.IsTerminal(os.Stdout.Fd())`，非 TTY 时加 `tea.WithoutRenderer()` 或
  `WithInput(nil)`；我们用更彻底的版本，非 TTY 时根本不进 bubbletea。
- 宽度来自 `tea.WindowSizeMsg`；测试或非交互场景可用 `tea.WithWindowSize(w, h)`。

### 7.4 路径输入与粘贴 chip

- 收到 `tea.PasteMsg` 后：去掉 NUL 字符 → 按终端类型解析候选路径 → `os.Stat` 校验。
- 命中文件 → 存为附件（只允许 1 个），字段显示 `[<basename>]`，内部保存完整路径。
- 未命中 → 当普通文本插入，并给一行提示（不静默失败）。
- chip 用 lipgloss 渲染，超长名按列宽截断补 `…`，完整路径显示在提示行。
- 解析器必须同时接受带引号与裸路径（细节见 §11）。

---

## 8. 命令与退出码

```text
eqm                    概览 + 帮助（当前配置、两个目录、命令列表）
eqm switch [<name>|<index>|original]   缺参数时列出配置并可选；带参数直接切
eqm import [<file>] [--name <name>] [--no-apply] [--move]
eqm remove [<name>] [--yes]
eqm init               设置并校验两个目录（APO 位置 + eqm 存放位置）
eqm show <name>        （可选）看某条配置的内容
eqm --version / --help
```

- `--plain`（或非 TTY）时 `switch` 只输出列表，不读键盘。
- 列表内容由 `core/render.go` 统一产生，交互与 `--plain` 共用同一份行。

| 退出码 | 含义 |
| --- | --- |
| 0 | 成功 |
| 1 | 失败（用法错误、缺参数、文件访问失败） |
| 2 | 用户取消（`Esc` / `Ctrl+C`） |
| 3 | 未初始化 / 配置损坏 |
| 4 | 导入文件格式不受支持 |

- 正常输出写 stdout，错误写 stderr；参数错误时补一段用法提示。
- 颜色只在 stdout 是终端时启用；`NO_COLOR`（任意值）关闭。
- 名字匹配：先精确（大小写不敏感），再唯一前缀；歧义时报错并列出候选。

---

## 9. 测试与常用命令

```bash
go build ./...                       # 编译
go test ./...                        # 单测
go vet ./... && gofmt -l .           # 静态检查与格式
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o eqm.exe ./cmd/eqm
```

| 测试层 | 内容 |
| --- | --- |
| 单元测试 | `naming` 推断、`eqparser` 正负样例、slug 冲突、`store` 增删改查与重名、`apobridge` 备份与替换（`t.TempDir()`）、`render` 纯函数 |
| 界面逻辑 | `ui/select` 的 Model 喂 `tea.KeyMsg` 序列断言光标 / 选中 / 取消；`ui/paste` 表驱动断言各类粘贴文本；中文名按列宽截断 |
| 夹具 | `testdata/fixtures/`：GraphicEQ、ParametricEQ、带 BOM、LF、空文件、随机文本 |
| 手工验收（Windows） | 初始化、导入、切换后主观听感变化、收尾后滚动记录是否干净、重定向不阻塞、删除生效项、只读目录报错、非 ASCII 用户名 |

---

## 10. 里程碑

| 阶段 | 内容 | 验收 |
| --- | --- | --- |
| M0 | `go.mod` + `cmd/eqm` 骨架、`eqm --version` | `go build ./...` 与 `go vet ./...` 通过 |
| M1 | `paths` / `appconfig` / `apobridge` + `init`（含当场校验与首次接管） | 生成的 `config.txt` 能被 APO 加载 |
| M2 | `eqparser` / `naming` / `store` + `import` / `switch --plain` | 导入真实 AutoEQ 样例无误 |
| M3 | 行内列表（`select` / `confirm` / `input` / `style` / `runner` / `paste`）+ `switch` | Windows 上切换即时生效，收尾干净 |
| M4 | `remove` / `show` / 重名覆盖流程 | 边界情况按设计表现 |
| M5 | 测试补齐、发布 zip | 干净机器上直接运行 |

---

## 11. 已核实事实

### 11.1 借鉴与不借鉴 `gh`

| `gh` 的行为 | 取舍 |
| --- | --- |
| 提示渲染在滚动记录里，不进备用屏幕 | 借鉴 |
| 缺参数时才提示，且都能用 flag 绕过 | 借鉴 |
| 列表：方向键 + 直接打字过滤 + 分页（`survey`，`PageSize` 20） | 借鉴（分页改为跟随光标滑动） |
| 无参数打帮助 | 不借鉴（`eqm` 进列表） |
| 退出码 0 / 1 / 2（取消） | 借鉴语义，另加 3 / 4 |
| `NO_COLOR` / `CLICOLOR` / `GH_FORCE_TTY` | 只用 `NO_COLOR` |
| `--json` / `--jq` / `--template` | v1 不做 |
| `GH_PROMPT_DISABLED` | 不做（TTY 判定已覆盖脚本场景） |
| 销毁类操作 `--yes` | 借鉴 |

### 11.2 粘贴与拖拽

| 事实 | 意义 |
| --- | --- |
| bubbletea v2 原生支持 bracketed paste 且默认开启，内容以 `tea.PasteMsg{Content}` 送到 `Update`（另有 `PasteStartMsg` / `PasteEndMsg`）；关闭用 `View.DisableBracketedPasteMode = true` | 提供了「在文本进输入框前截住」的钩子 |
| Windows Terminal 的拖拽不是读剪贴板：WT 自己解析拖入文件，把路径文本按 bracketed paste 送进终端 | 拖拽与粘贴是同一件事，只需一个解析器 |
| WT 多个路径之间默认一个空格分隔，**结尾不加分隔符** | 解析不能依赖结尾空格 |
| WT **只有路径含空格时才加双引号**（路径转换开启时用单引号），无空格路径是裸的 `C:\a\b.txt` | 解析器要同时吃两种形式（crush 的解析器在此有缺陷） |
| 可用 `WT_SESSION` 判断 Windows Terminal；Rio 在 Windows 上会夹 `\x00` | 先清 NUL 再解析 |
| 剪贴板文件列表是 `CF_HDROP`，`golang.design/x/clipboard` 的 `ReadFiles(ctx)` 可读且无需 cgo；`atotto/clipboard` 只支持文本 | 用于「资源管理器 Ctrl+C 后 Ctrl+V」（见 open-questions Q9） |
| 参考实现：crush `internal/fsext/paste.go`、docker/docker-agent `pkg/tui/components/editor/paste.go`；chip 渲染参考 crush `internal/ui/attachments/attachments.go` | 照抄解析规则而不是自己猜 |

### 11.3 chip 输入框现状

| 方案 | 结论 |
| --- | --- |
| `bubbles` 的 `textinput` / `textarea` | 只有普通文本，无 chip 支持 |
| `huh` 的 `Input` / `FilePicker` | `FilePicker` 是目录浏览，不是 chip |
| Codex CLI 的 `TextElement` | 多附件编辑器方案（占位符映射 + 重新编号），单文件场景不需要 |
| 自实现 | 一个字段 + 至多一个附件 + lipgloss 渲染，约几十行 |

---

## 12. 待实测清单

大部分 APO 行为已经在 2026-09-22 通过源码与官方文档核实（结论见 §4.3、§4.4），
这里只留真正还没把握的。

### Equalizer APO 行为（只能在 Windows 上验证）

1. **自动重载的实际表现**：切换后 Equalizer APO 多久生效（预期是亚秒级，因为有 10 ms
   去抖 + 整份重读），以及需不需要重启播放流。
2. **非 ASCII 路径**：库里路径含中文时 APO 能否正常 `CreateFile` 打开
   （源码显示路径会转成宽字符再打开，且行文本先按 UTF-8 解码，所以预期可用，但要实测）。
3. **需要管理员权限的场景**：默认安装下 `Program Files\EqualizerAPO\config` 不可写时，
   以管理员身份运行 `eqm` 的体验如何；是否值得在文档里主推「把 APO 配置目录挪到可写位置」。
4. `Include:` 进来的文件是否需要自带 `Preamp:`（本方案认为不需要）。
5. 长路径边界：接近 260 字符的路径是否真的失败（验证 §4.4 里的 `MAX_PATH` 结论）。

验证方式：手工构造最小 `config.txt` + `Include` 组合，改一改被包含的文件，观察 APO
Configurator 与 Edit 窗口的反馈。结论记录到 `docs/apo-behavior-notes.md`。

### 界面行为（Windows Terminal 与旧版 conhost 各跑一遍）

6. 收尾后滚动记录是否干净（空 `View()` 与 `ClearScreen` 两种写法都要试）。
7. 拖动窗口改变宽度后列表是否正确重排。v2 的 `signals_windows.go` 里
   `listenForResize` 是 no-op（issue #1601 未闭环）；兜底方案是每次按键自行读一次宽度
   （`x/term` 的 `GetSize` 或 `windows.GetConsoleScreenBufferInfo`）。
8. 方向键在 v2 的 VT 输入模式下是否正常（v2 已不再用控制台事件）。
9. 把文件从资源管理器拖进终端，解析器是否吃下实际字节。
10. 中文在两处的显示效果：列表里的列宽与截断；旧版 conhost 的字体/代码页
    （启动切到 UTF-8 后是否足够，还是要在文档里建议用 Windows Terminal）。
11. 系统语言检测在中文系统与英文系统上各跑一次，确认 `LC_ALL` 等环境变量的覆盖生效。

第 6-11 条用同一个 60 行的最小 `select` 原型加一段文案即可覆盖。

---

## 13. 界面语言实现

已核实的可用手段：

| 需求 | 手段 |
| --- | --- |
| 读系统语言 | `windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)` 返回 `[]string{"zh-CN","en-US",…}`，取第一个即可。**注意**：`GetUserDefaultUILanguage` / `GetUserDefaultLocaleName` 并没有被 `x/sys/windows` 导出，需要自己 `NewLazySystemDLL` 包装（不必要，用上面那个就够） |
| 用户覆盖 | 按 gettext 惯例取第一个非空值：`LANGUAGE` → `LC_ALL` → `LC_MESSAGES` → `LANG`；Windows 原生命令行通常不设这些变量，所以只在 Git Bash / WSL / 手动设置时生效 |
| 语言匹配 | 简单的 `zh*` → 中文、`en*` → 英文、其余 → 英文即可；需要更正式可上 `golang.org/x/text/language` 的 `NewMatcher` |
| 文案存放 | v1 用 `//go:embed` 内嵌两份 `map[string]string`（或两个 JSON）最简单，零依赖；`go-i18n` 适合要复数规则/翻译流程时再换 |
| 中文输出 | 启动时 `windows.SetConsoleOutputCP(65001)` 与 `SetConsoleCP(65001)`；Go 源码本身永远是 UTF-8，源文件不要加 BOM。旧版 conhost 是 GDI 渲染且没有字体回退，中文可能显示为方框，需要在文档里建议用 Windows Terminal |
| 管道输出 | 重定向时控制台代码页无关，直接输出 UTF-8 字节即可 |

约定：**文案不硬写在代码里**；`--lang` 参数预留但 v1 可以先不做。

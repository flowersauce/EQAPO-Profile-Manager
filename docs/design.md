# AutoEQ-APO-Manager 设计方案

> 状态：v0.5（按反馈收敛：裸 `eqm` 接管概览，取消独立的自检命令）。
> 本文档只回答「我们要做一个什么东西」，不写怎么做。
> 实现细节见 `docs/tech-notes.md`；待定问题见 `docs/open-questions.md`。

---

## 1. 它是什么

AutoEQ-APO-Manager（命令 `eqm`）是给 **AutoEQ + Equalizer APO** 用户用的轻量配置管理器。

一句话：

> 让用户不用再翻到目录里手动改 `config.txt`，就能导入、管理、切换 AutoEQ 生成的 EQ 配置。

技术前提：**首选 Go 实现**，产出一个单文件可执行程序，用户不需要安装任何运行时。

### 管什么，不管什么

| 管 | 不管 |
| --- | --- |
| 存放、校验、命名、列出、切换 Profile | 生成 EQ 曲线（AutoEQ 负责） |
| 托管 Equalizer APO 的 `config.txt` | 处理音频（Equalizer APO 负责） |
| 识别 GraphicEQ / ParametricEQ 两种格式 | 修改、重算、优化 EQ 参数 |
| 提供少量命令行子命令 | 通用 EQ 编辑器、游戏调音、GUI |

---

## 2. 使用前提

1. 用户已安装 Equalizer APO。
2. 用户已从 AutoEQ 下载适用于 Equalizer APO 的配置（`EqualizerAPO GraphicEq` 或
   `ParametricEq`，推荐前者）。

---

## 3. 命令与体验

命令面刻意保持很小：**4 个动作 + `--version` / `--help`**。

```text
eqm                     概览 + 帮助（当前配置、两个目录、命令列表）
eqm switch [<名字|序号>]  不带参数时列出配置并可用键盘切换；带参数直接切
eqm import [<文件>]      导入 AutoEQ 配置
eqm remove [<名字>]      删除配置，不带参数时进入选择列表
eqm init                 设置并校验两个目录（Equalizer APO 位置、eqm 存放位置）
eqm --version / --help
```

约定：**交互只用来补齐参数**。参数给全（或加 `--yes`）时全程无交互，可直接写进脚本；
在管道或重定向里运行不会卡住等键盘。

### 3.1 敲 `eqm`：概览 + 帮助

不带参数时不做任何改动，只把「现在什么情况」和「能做什么」摆出来：

```text
❯ eqm

AutoEQ-APO-Manager  v0.1.0

  当前配置        Philips SHP9500（GraphicEQ）
  配置总数        3 个
  Equalizer APO   D:\APP\Tools\EqualizerAPO
  eqm 存放目录    D:\EQM

  eqm switch      选择要使用的配置
  eqm import      导入 AutoEQ 配置
  eqm remove      删除配置
  eqm init        设置并校验两个目录

  完整选项见 eqm --help
```

- 这一页相当于「状态 + 关于」：**软件版本、当前配置、两个目录、能做什么**。
- 还没初始化过时，这里直接提示「先运行 `eqm init`」，并把这个页面当成引导。
- 当前配置已经失效（文件被删或挪走）时，这里说明「已自动回到原声」。

### 3.2 选择与切换（`eqm switch`）

敲命令就出现列表，上下键移动，**直接打字即可过滤**（形态参考 `gh auth login`）：

```text
❯ eqm switch
? 选择要使用的配置  [↑/↓ 移动，输入字符过滤]

❯ Original
  Philips SHP9500  *
  Moondrop Aria 2
```

- `Enter` 立即应用，按完即生效（Equalizer APO 会自己重新加载）。
- `Esc` 取消，什么都不改。
- `*` 标出当前生效的配置；`Original`（原声）固定在第一项。
- 列表超出屏幕时跟随光标滚动。
- 直接切换（脚本用）：`eqm switch "Philips SHP9500"` 或 `eqm switch 2`。

### 3.3 导入（`eqm import`）

```text
❯ eqm import

AutoEQ 配置文件路径：

❯ [Philips SHP9500 GraphicEq.txt]

Enter 导入    Esc 取消
```

- 路径可以手输、Tab 补全，也可以**直接把文件拖进终端或粘贴**（显示成上面这种
  `[文件名]` 的样子，实际保存的是完整路径）。
- 也可以一次给全参数：`eqm import "D:\Downloads\Philips SHP9500 GraphicEq.txt"`。
- 导入时识别格式；不是受支持的文件就直接拒绝，不做自动修复。
- 导入完成后，原来那份下载文件**默认保留**（是否顺手清理见 `open-questions.md` Q7）。

### 3.4 删除（`eqm remove`）

不带参数时同样是一个可用键盘选择的列表，删除前要确认：

```text
❯ eqm remove
? 选择要删除的配置  [↑/↓ 移动，输入字符过滤]

❯ Philips SHP9500
  Moondrop Aria 2

删除后无法恢复，确认删除？[y/N]
```

- 删的是**当前正在使用**的配置时，额外提示一句（并建议先切回 Original）。
- 删除当前生效项之后，`Original` 自动生效，不留悬空引用。
- 脚本用法：`eqm remove "Philips SHP9500" --yes`。

### 3.5 初始化（`eqm init`）：设置 + 当场校验

第一次运行（以及以后想换目录、怀疑出问题时）都用这一条。两项都直接输入路径，不弹对话框：

```text
1) Equalizer APO 的位置

   填安装根目录或 config 目录都可以：
   > D:\APP\Tools\EqualizerAPO

2) eqm 存放配置的位置（默认 %LOCALAPPDATA%\AutoEQ-APO-Manager）

   > D:\EQM
```

**输入当下就校验，有问题立刻说，不留到以后**：

- 路径不存在 → 当场报错并重新输入。
- 路径里找不到 `config.txt`（无论是 `目录\config.txt` 还是 `目录\config\config.txt`）→
  当场报错并说明应该指向哪里。
- 路径含非 ASCII 字符 → 当场警告（这类路径可能让 Equalizer APO 读不到配置），
  并建议换一个纯英文路径。
- `config.txt` 不可写 → 当场提示需要管理员权限，并给出重试方式。

校验通过后写入配置，并在 Equalizer APO 那边准备好被托管的 `config.txt`（见 3.6）。
因为 Equalizer APO 那边只有一行 `Include:` 指过来，**换存放目录只要重跑一次 `eqm init`**。

### 3.6 首次接管 `config.txt`（只发生一次）

这是本项目唯一会碰到用户既有文件的地方，处理顺序：

1. `config.txt` 不存在 → 直接创建。
2. 已由 `eqm` 托管（文件里带托管标记）→ 直接重写，无副作用。
3. 存在但未被托管（用户自己写过配置）→ **先备份**为 `config.txt.eqm.bak`，
   再问一句「要不要把里面的内容保存成一条配置？」（默认名 `Existing config`），
   然后才写入托管内容。

托管之后，`config.txt` 的形状是固定的（指向 APO 自己目录下的一个小文件）：

```txt
# 此文件由 AutoEQ-APO-Manager 自动管理，请勿手动修改。
Include: eqm\current.txt
```

而 `eqm\current.txt` 才是「现在用哪条配置」的指针：

```txt
# 此文件由 AutoEQ-APO-Manager 自动管理，请勿手动修改。
Include: D:\EQM\profiles\philips-shp9500\eq.txt
```

- 切换配置时只重写第二个文件（一行 Include），`config.txt` 永远不动。
- **第二个文件必须在 Equalizer APO 的 config 目录里面**。原因已经核实：Equalizer APO 只监视
  自己 config 目录树内的文件变化，指向目录外的文件改动它不会自动重载。
- 代价：需要能写 Equalizer APO 的 config 目录（默认装在 `Program Files` 下属管理员权限）。
  因此 `eqm init` 会把「能不能写」当成必检项，不能写就当场说清楚。

含义是：**这两个文件以后由 `eqm` 负责，用户不要在这里写自己的东西**。
本项目不做「把用户的 filter 和 AutoEQ 的配置合并」这种事；用户原有内容只会被原样备份，
或原样存成一条独立配置。

---

## 4. 数据放在哪（高层描述）

- 所有配置存在 **eqm 存放目录**里（`eqm init` 时指定，默认在 `%LOCALAPPDATA%`），
  不堆在 Equalizer APO 的安装目录下。
- 每条配置一个子目录：一份**原样的** EQ 文件 + 一份很小的元数据（名字、格式、来源）。
- Equalizer APO 那边只有两个由 `eqm` 管理的文件：`config.txt`（入口，固定不变）和
  `eqm\current.txt`（指向当前配置，切换时重写）。放在 `config` 目录内部是因为
  Equalizer APO 只监视自己目录树里的变化，放在外面它不会自动生效。
- 导入的 EQ 内容原样保存，不加工、不重算。
- **原声（Original）** 是内置选项，不对应任何 EQ 文件；选它时 Equalizer APO 仍在音频链里，
  但不做任何调整。

具体目录结构、元数据字段见 `docs/tech-notes.md`。

---

## 5. v1 做什么，不做什么

**必须做**

- 初始化：设置 + 当场校验两个目录（存在性、是否找得到 `config.txt`、非 ASCII 警告、
  能否写入 Equalizer APO 的 config 目录）
- 首次接管 `config.txt`：先备份、可把原内容存成一条配置（见 3.6）
- 导入配置：GraphicEQ / ParametricEQ 识别、重名处理、非法文件拒绝
- 列表选择与切换、原声模式、删除（当前生效项额外提示）
- 概览页（裸 `eqm`）：版本号、当前配置、两个目录、能做什么
- 自愈：任何时刻发现当前指向的配置不存在，就自动退回原声并提示一句
- 中英双语界面（见第 7 节）

**不做**

- 独立的自检命令（原本设想的 `check`/`doctor`）：它能查的东西都已经有归属，
  见 `open-questions.md` Q4
- 在线搜索 / 下载 AutoEQ、EQ 曲线显示、EQ 参数编辑、游戏专用调音、HeSuVi、REW、多层 EQ 组合
- Linux / PipeWire / EasyEffects 等其他后端
- GUI、全屏 TUI、鼠标操作、系统文件对话框
- 常驻后台、实时监控、自动更新
- 合并用户自己的 filter 与 AutoEQ 配置

---

## 6. 必须守住的原则

1. **不擅自破坏**：接管或覆盖用户已有文件之前，先备份或先询问；默认不静默覆盖。
2. **不动用户的东西**：AutoEQ 内容原样保存，导入的源文件默认保留。
3. **不做全屏**：提示出现在滚动记录里，不接管整块屏幕，不隐藏 shell 历史。
4. **不阻塞**：不能提问的环境里不假装能提问。
5. **当场校验**：能在输入时发现的问题就在输入时说，不留到以后。
6. **自愈**：发现当前指向的配置不存在，就自动退回原声并提示，不留悬空引用。
7. **轻**：不常驻、不驻后台、不改系统设置、不装驱动。

---

## 7. 界面语言（中英双语）

- 界面文案一律走双语资源，**不把中文硬写在代码里**。
- 默认跟随系统：读 Windows 的显示语言（已核实：Go 可直接调系统的
  `GetUserPreferredUILanguages`，返回 `zh-CN` / `en-US` 这类名字，不需要额外依赖）。
- 用户可覆盖：环境变量优先（`LC_ALL` / `LANGUAGE` / `LC_MESSAGES` / `LANG`，
  与 gettext 习惯一致），将来也可以加 `--lang` 参数。
- v1 只准备 **简体中文** 与 **英文** 两套；其他语言回退到英文。
- 中文在旧版 conhost（cmd 窗口）里可能因字体与代码页而显示异常，
  启动时会把控制台切到 UTF-8；Windows Terminal 不受影响（细节见 `tech-notes.md`）。

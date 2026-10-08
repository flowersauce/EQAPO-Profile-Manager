# EQM Social Preview 设计

已确认的正式版本用于 GitHub 仓库和链接分享。目录保留最终图片、可编辑源文件、设计说明以及生成所需字体与许可。

| 文件 | 用途 |
| --- | --- |
| [social-preview.png](social-preview.png) | 正式 PNG，1280×640，40,233 字节 |
| [social-preview.svg](social-preview.svg) | 可编辑 SVG，内嵌原始 EQM Logo |
| [fonts/](fonts/README.md) | FloriInputUI Medium 字体、来源与许可 |

## 文案与布局

介绍：

> 在任意终端中轻松切换 EQ 配置
>
> 而不必打开 Equalizer APO 配置编辑器

三行保留终端形式：

```text
> eqm import  导入
> eqm switch  切换    预设 EQ 配置
> eqm remove  移除
```

“预设 EQ 配置”是三项动作的共同对象，与“切换”共用基线、字号，横向间距为 34 px。预设指用户准备好的配置，不暗示软件内置配置或生成均衡参数。

- 背景：Atom Dark，`#21252b`。
- 字体：FloriInputUI Medium，沿用 Flori-Input 的已许可字体。
- 名称／主要介绍／补充介绍：96／32／24 px。
- 命令／动作及共享对象：25／23 px；三行间距 40 px。
- 提示符：导入绿色 `#98c379`、切换蓝色 `#61afef`、移除红色 `#e06c75`。
- 仓库链接：19 px，前置紫色圆点 `#c678dd`。
- 命令区域位于介绍文字与仓库链接之间的中心，中心 y 为 438。

Logo 原图为 [eqm-logo.svg](../icons/app/eqm-logo.svg)，几何与颜色保持不变。实际图形为 400×400，边界 `(760,120) → (1160,520)`，中心 `(960,320)`，位于右半侧正中心。上、下、右边距均为 120 px。

## 生成与维护

修改 SVG 后，在项目根目录运行：

```powershell
.\.venv\Scripts\python.exe .\scripts\generate-social-preview.py
```

默认只更新正式 PNG，不产生临时检查图。需要检查缩略图时，用 `--review-output` 明确指定输出位置，例如系统临时目录：

```powershell
.\.venv\Scripts\python.exe .\scripts\generate-social-preview.py `
  --review-output (Join-Path $env:TEMP 'eqm-social-preview-review.png')
```

脚本复用 Pillow 与 resvg-py，并明确加载本目录字体。更新 Logo 时，将唯一图标源的 `logo-art` 组同步至 SVG。旧版本、检查图和逐版临时记录已清理，不作为生成输入。

[Microsoft Store 素材](../microsoft-store/design.md) 从正式 SVG 独立生成中英文 16:9 头图。商店生成器只读取本目录，不修改正式 Social Preview。

## 验证

正式图片已检查完整尺寸、缩略图、Logo 位置和边距、命令区居中、字体、彩色提示符、仓库圆点及原始 Logo 几何。整理时直接保留已确认图片与 SVG 的内容，SHA256 为：

```text
SVG  F2AACDF365C508C3B01FB1C91CA6C6889439BC7CB8749CE38307B9FAC1F19776
PNG  CC38414598153140A6234EC2994A1228B68AC1BE56B428256E05AF21C36E0BA3
```

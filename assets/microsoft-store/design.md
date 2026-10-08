# EQM Microsoft Store 素材设计

正式素材沿用 [Social Preview](../social-preview/social-preview.svg) 的构图、Atom Dark 背景、FloriInputUI Medium 字体、原始 Logo 和彩色终端提示符。目录只保存正式图片、可编辑源文件及本设计说明，不随 MSIX 打包。

## 文件与用途

| 图片 | 用途 | 尺寸与格式 |
| --- | --- | --- |
| [app-icon.png](app-icon.png) | 中英文共用的 1:1 应用磁贴图标 | 300×300，RGBA PNG，外围透明 |
| [zh-CN/cover.png](zh-CN/cover.png) | 中文介绍头图 | 1600×900，16:9，RGB PNG |
| [en-US/cover.png](en-US/cover.png) | 英文介绍头图 | 1600×900，16:9，RGB PNG |

可编辑源文件：[中文 cover.svg](zh-CN/cover.svg)、[英文 cover.svg](en-US/cover.svg)。图标直接从唯一源文件 [eqm-logo.svg](../icons/app/eqm-logo.svg) 渲染。

两个语言页面的“Microsoft Store 徽标 → 1:1 应用磁贴图标（300×300）”均使用 `app-icon.png`。

## 文案与构图

中文介绍：

> 在任意终端中轻松切换 EQ 配置
>
> 而不必打开 Equalizer APO 配置编辑器

英文介绍：

> Easily switch EQ profiles in any terminal
>
> Without opening Equalizer APO’s Configuration Editor

保留 `eqm import`、`eqm switch`、`eqm remove` 三行命令。动作文字分别为“导入／切换／移除”和“Import／Switch／Remove”；共同对象为“预设 EQ 配置”和“Preset EQ profiles”，放在 switch 行，与动作共用基线。预设指用户准备好的配置，不宣称软件内置配置或生成均衡参数。

- 中文保留 Social Preview 的文字与几何关系，将 1280×640 等比放大至 1600×800，上下各延展 50 px 同色背景。
- 英文沿用同一构图，两行介绍使用 28／22 px 的逻辑字号；“Switch” 后根据字体宽度保留 34 px 的逻辑间距。
- Logo 原始几何与颜色不变，实际边界为 `(950,200) → (1450,700)`，尺寸 500×500，中心 `(1200,450)`。
- 字体来源与许可见 [字体说明](../social-preview/fonts/README.md)。

## 上传说明

头图按 Flori-Input 的介绍首图规格制作，实际终端截图后续补充。[微软截图与图像指南](https://learn.microsoft.com/en-us/windows/apps/publish/publish-your-app/msix/screenshots-and-images) 建议截图展示实际界面，不添加额外 Logo 或营销信息；介绍头图放入截图栏的做法与该指南不完全一致，提交时需确认采用方式。

含文字的 1600×900 头图不用于独立的“16:9 超级主图”栏；该栏要求 1920×1080 或 3840×2160，且不含产品标题或其他文字。本次范围为图标与两张介绍头图，暂不包含其他可选主图、Xbox 图片或预告片。

如采用头图作为介绍首图，可使用以下图片说明：

| 语言 | 图片说明 |
| --- | --- |
| 中文（中国） | EQM：在终端中导入、切换与移除预设 EQ 配置。 |
| English (United States) | EQM: import, switch, and remove preset EQ profiles from your terminal. |

## 生成与验证

在项目根目录运行：

```powershell
.\.venv\Scripts\python.exe .\scripts\generate-store-images.py
```

默认只更新正式 PNG 和 SVG，不生成检查图。需要检查图时，用 `--review-output` 明确指定位置，例如系统临时目录：

```powershell
.\.venv\Scripts\python.exe .\scripts\generate-store-images.py `
  --review-output (Join-Path $env:TEMP 'eqm-store-review.png')
```

脚本复用项目已有的 Pillow 与 resvg-py，只写本目录和明确指定的检查图位置。正式 Social Preview、原始 Logo、字体和 Flori-Input 参考项目保持只读。SVG 从正式 Social Preview 重建；手工修改生成 SVG 前请另存。

图片已检查尺寸、色彩模式、图标透明边缘、完整画面与缩略图、Logo 位置、字体、英文间距及基线。整理后重新生成，正式 PNG／SVG 的 SHA256 保持一致。图标、中文头图、英文头图大小分别为 4,580／53,987／51,019 字节。

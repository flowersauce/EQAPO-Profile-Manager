# AutoEQ-APO-Manager

给 **AutoEQ + Equalizer APO** 用户用的轻量配置文件管理器：导入、管理、切换 AutoEQ 生成的
EQ 配置，不用再翻到目录里手动改 `config.txt`。

命令行工具名 `eqm`，用 Go 实现（单文件可执行，用户无需安装运行时）。

> 当前处于设计阶段，尚无代码。

## 文档

- [设计方案](docs/design.md)：要做一个什么东西（定位、命令与体验、数据放在哪、范围与原则）
- [待定问题](docs/open-questions.md)：还需要拍板的决策
- [实现备注](docs/tech-notes.md)：库的选择、文件格式、Equalizer APO 的行为细节

## 命令一览

```text
eqm                概览 + 帮助（含当前配置与两个目录）
eqm switch         选择要使用的配置（列表 + 上下键 + 打字过滤）
eqm import         导入 AutoEQ 配置
eqm remove         删除配置
eqm init           设置并校验两个目录
```

// Package i18n keeps user-facing messages separate from APO file content.
package i18n

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
)

type Language struct{ Chinese bool }

func ForLocale(locale string) Language {
	language := strings.ToLower(strings.Split(strings.ReplaceAll(locale, "_", "-"), "-")[0])
	return Language{Chinese: language == "zh"}
}

func (l Language) Text(key string, args ...any) string {
	pair, ok := messages[key]
	if !ok {
		return key
	}
	text := pair[0]
	if l.Chinese {
		text = pair[1]
	}
	if len(args) > 0 {
		text = fmt.Sprintf(text, args...)
	}
	return text
}

func (l Language) Error(err error) string {
	var e *fault.Error
	if errors.As(err, &e) {
		s := l.Text(e.Key, e.Args...)
		if e.Cause != nil {
			s += ": " + l.Error(e.Cause)
		}
		return Safe(s)
	}
	return Safe(err.Error())
}

// Safe prevents filenames and OS errors from injecting terminal control sequences.
func Safe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}

var messages = map[string][2]string{
	"settingsLocation":      {"Cannot determine an absolute user settings location.", "无法确定用户配置目录的绝对路径。"},
	"portableFlag":          {"portable.flag must be a regular file beside eqm.exe.", "portable.flag 必须是 eqm.exe 同目录下的普通文件。"},
	"elevationContext":      {"Invalid internal elevation context. Run eqm normally.", "内部提权上下文无效，请正常运行 eqm。"},
	"elevationRequired":     {"Administrator access is needed. Approve UAC to continue this operation here.", "此操作需要管理员权限，请在 UAC 中确认，随后将在当前终端继续。"},
	"elevationCancelled":    {"Administrator access was declined. The pending operation was not continued.", "已拒绝管理员提权，未继续执行待完成的操作。"},
	"elevationDisconnected": {"Administrator connection was lost. The operation may have completed; check the current state before retrying.", "管理员连接已断开，操作可能已经完成；请检查当前状态后再重试。"},
	"workerError":           {"%s", "%s"},
	"elevationFailed":       {"Could not start or wait for the administrator instance", "无法启动或等待管理员实例"},
	"elevationDenied":       {"Access is still denied; check file permissions and read-only attributes", "仍无法访问，请检查文件权限和只读属性"},
	"elevationTerminal":     {"This operation needs administrator access. Run the command in an interactive terminal to approve UAC.", "此操作需要管理员权限，请在交互终端运行该命令以确认 UAC 提权。"},
	"healedFailure":         {"The missing selection was reset to None, but the operation failed. Run the command again", "丢失的选择已恢复为 None，但本次操作失败，请重新运行命令"},
	"usage":                 {"Unknown command or arguments. Run eqm for an overview.", "未知命令或参数。运行 eqm 查看命令概览。"},
	"tty":                   {"This command requires an interactive terminal (stdin, stdout and stderr).", "此命令需要交互终端（stdin、stdout 和 stderr）。"},
	"uninitialized":         {"Not initialized. Run eqm init.", "尚未初始化，请运行 eqm init。"},
	"settings":              {"Invalid settings; run eqm init to repair them", "程序设置无效，请运行 eqm init 修复"},
	"regular":               {"Not a regular file: %s", "不是普通文件：%s"},
	"directory":             {"Not a directory, or a reparse point: %s", "不是目录或属于重解析点：%s"},
	"busy":                  {"Another eqm operation is running. Try again shortly.", "另一个 eqm 操作正在进行，请稍后重试。"},
	"changed":               {"File changed during this operation. Restart the command: %s", "操作期间文件已变化，请重新运行命令：%s"},
	"encoding":              {"Expected UTF-8 text (an optional BOM is supported).", "文件应为 UTF-8 文本（支持 BOM）。"},
	"filename":              {"Use a valid Windows .txt filename; None is reserved.", "请输入有效的 Windows .txt 文件名；None 是保留名。"},
	"managed":               {"The managed section is damaged or ambiguous; config.txt was not overwritten.", "托管区域损坏或含义不明确，未覆盖 config.txt。"},
	"invalidProfile":        {"Invalid profile filename: %s", "配置文件名无效：%s"},
	"collision":             {"That filename already exists. Choose another name.", "文件名已存在，请输入其他名称。"},
	"rootInvalid":           {"Choose the Equalizer APO installation root containing config", "请选择包含 config 子目录的 Equalizer APO 安装根目录"},
	"initPartial":           {"APO configuration was updated, but settings could not be saved. Retry eqm init", "APO 配置已更新，但程序设置未保存，请重试 eqm init"},
	"onePath":               {"Enter exactly one file or directory path.", "请只输入一个文件或目录路径。"},
	"format":                {"Not a supported AutoEQ GraphicEQ or ParametricEQ text file.", "不是受支持的 AutoEQ GraphicEQ 或 ParametricEQ 文本文件。"},
	"sameFile":              {"The source is already the managed file; nothing was imported.", "源文件就是已管理的文件，未执行导入。"},
	"sourceKept":            {"Import completed; the source file was kept", "导入已完成，源文件已保留"},
	"renamePartial":         {"Rename could not be rolled back. Check this file and config.txt: %s", "改名未能回滚，请检查此文件及 config.txt：%s"},
	"removePartial":         {"The file was not deleted; the selection may now be None", "文件未删除，当前选择可能已变为 None"},
	"cancelled":             {"Cancelled.", "已取消。"},
	"yes":                   {"Yes", "是"},
	"no":                    {"No", "否"},
	"chooseHelp":            {"↑/↓ or j/k · Enter select · Esc cancel", "↑/↓ 或 j/k · Enter 选择 · Esc 取消"},
	"inputHelp":             {"Enter submit · Esc cancel", "Enter 提交 · Esc 取消"},
	"pathHelp":              {"Tab accept / cycle · Enter submit · Esc cancel", "Tab 接受/切换补全 · Enter 提交 · Esc 取消"},
	"working":               {"Working…", "正在处理…"},
	"rootQuestion":          {"Enter the Equalizer APO installation directory", "请输入 Equalizer APO 安装目录"},
	"takeoverQuestion":      {"Allow eqm to manage config.txt?", "允许 eqm 接管 config.txt 吗？"},
	"takeoverHelp":          {"%s · Original lines will be kept as comments; EQ starts at None.", "%s · 原内容将保留为注释，初始选择为 None。"},
	"initDone":              {"Initialized.", "初始化完成。"},
	"selectSwitch":          {"Select a profile", "选择要应用的配置"},
	"selectShow":            {"Select a profile to show", "选择要查看的配置"},
	"selectOpen":            {"Select a profile to open", "选择要打开的配置"},
	"selectRename":          {"Select a profile to rename", "选择要重命名的配置"},
	"selectRemove":          {"Select a profile to remove", "选择要删除的配置"},
	"appliedLegend":         {"✓ applied", "✓ 已应用"},
	"switched":              {"Applied: %s", "已应用：%s"},
	"empty":                 {"No managed profiles.", "暂无已管理的配置。"},
	"sourceQuestion":        {"Enter the AutoEQ file path to import", "请输入想要导入的 AutoEQ 文件路径"},
	"overwriteQuestion":     {"Overwrite %s?", "覆盖 %s 吗？"},
	"deleteSourceQuestion":  {"Import completed. Delete the source file?", "导入已完成，是否删除源文件？"},
	"imported":              {"Imported: %s", "已导入：%s"},
	"renameQuestion":        {"Enter a new filename", "输入新文件名称"},
	"renameHint":            {"No file extension needed", "无需输入文件后缀"},
	"openFailed":            {"Could not open the profile", "无法打开配置文件"},
	"openDialogShown":       {"Open With dialog displayed: %s", "已显示打开方式窗口：%s"},
	"initPathHelp":          {"Tab accept / complete · Enter submit (empty uses default) · Esc cancel", "Tab 接受/补全 · Enter 提交（留空使用默认路径）· Esc 取消"},
	"renamed":               {"Renamed: %s", "已改名：%s"},
	"removeQuestion":        {"Delete %s?", "删除 %s 吗？"},
	"removed":               {"Removed: %s", "已删除：%s"},
	"status":                {"Current profile", "当前配置"},
	"count":                 {"Profiles", "配置数量"},
	"apoDir":                {"EqualizerAPO", "EqualizerAPO"},
	"version":               {"Version", "版本"},
	"aboutHeading":          {"About", "关于"},
	"infoHeading":           {"Information", "信息"},
	"helpHeading":           {"Help", "帮助"},
	"author":                {"Author", "作者"},
	"repository":            {"Repository", "仓库"},
	"commandSwitch":         {"Select a profile", "选择并应用配置"},
	"commandList":           {"List profiles", "列出配置"},
	"commandImport":         {"Import an AutoEQ file", "导入 AutoEQ 文件"},
	"commandRemove":         {"Delete a profile", "删除配置"},
	"commandShow":           {"Show profile contents", "显示配置内容"},
	"commandOpen":           {"Open a profile with an app", "选择应用打开配置"},
	"commandRename":         {"Rename a profile file", "修改配置文件名"},
	"commandInit":           {"Set the EqualizerAPO directory", "设置 EqualizerAPO 安装目录"},
}

package host

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestSnapshotBuilderHasNoHardcodedCJK 把"侧栏文案跟随界面语言"钉在源码层。
//
// fillDetails 是 TUI 侧栏的唯一数据来源。它散落在 host.go 中间，而 host.go
// 因为控制流与展示混在一起没法整体纳入 i18n 覆盖闸（几百条 fmt.Errorf 面向
// 维护者，整包纳入只会让白名单变成噪音）。这里只钉这一个函数：
// 曾经就是它把 "第%d卷·第%d弧"、"%s（%s）" 写死，越南语界面下侧栏半中半越。
//
// 只查汉字不够——真正让界面露馅的常是（）、。「」这类全角标点。
func TestSnapshotBuilderHasNoHardcodedCJK(t *testing.T) {
	src, err := os.ReadFile("host.go")
	if err != nil {
		t.Fatal(err)
	}
	body, ok := funcBody(string(src), "func (h *Host) fillDetails(")
	if !ok {
		t.Fatal("找不到 fillDetails，本测试的扫描锚点已失效，请更新")
	}

	cjk := regexp.MustCompile(`[\p{Han}\x{3000}-\x{303F}\x{FF01}-\x{FF60}]`)
	lit := regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	// 结构化日志面向维护者，与 i18n 覆盖闸同一条判定界线。
	logCall := regexp.MustCompile(`(?:slog|log|logger\(\))\.(?:Debug|Info|Warn|Error)\(`)
	for i, line := range strings.Split(body, "\n") {
		if s := strings.TrimSpace(line); s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		if logCall.MatchString(line) {
			continue
		}
		for _, m := range lit.FindAllStringSubmatch(line, -1) {
			if cjk.MatchString(m[1]) {
				t.Errorf("fillDetails+%d 硬编码 CJK 文案 %q：侧栏必须走 i18n.T", i, m[1])
			}
		}
	}
}

// funcBody 返回以 prefix 开头的函数体（按大括号配对截取，字符串里的括号不计）。
func funcBody(src, prefix string) (string, bool) {
	i := strings.Index(src, prefix)
	if i < 0 {
		return "", false
	}
	start := strings.IndexByte(src[i:], '{')
	if start < 0 {
		return "", false
	}
	start += i
	depth, inStr, esc := 0, false, false
	for j := start; j < len(src); j++ {
		c := src[j]
		switch {
		case esc:
			esc = false
		case c == '\\' && inStr:
			esc = true
		case c == '"':
			inStr = !inStr
		case inStr:
		case c == '{':
			depth++
		case c == '}':
			if depth--; depth == 0 {
				return src[start : j+1], true
			}
		}
	}
	return "", false
}

package host

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/dongphuongman/novel-cli/internal/i18n"
)

// 汉字之外还要盯全角标点：真正让界面"看起来还是中文"的往往是 （）、。「」，
// 它们不在 \p{Han} 里。截图里那行 architect_short（…）就是这么漏过去的。
var hanInDisplay = regexp.MustCompile(`[\p{Han}\x{3000}-\x{303F}\x{FF01}-\x{FF60}]`)

// TestDisplayStringsFollowUILanguage 是"界面语言必须真的生效"的行为级验收。
//
// 为什么不能只靠 i18n 的源码扫描闸：那道闸看的是**源码里有没有中文字面量**，
// 看不出常量是否在 init 期定型。曾经的真实故障就是这个形状——事件流、TOOL 行、
// 侧栏在越南语界面下整片仍是中文，而源码扫描完全无感（那些文案根本不在
// uiPackages 覆盖的目录里）。本测试直接切语言、渲染、断言产出里没有汉字。
func TestDisplayStringsFollowUILanguage(t *testing.T) {
	original := i18n.Current()
	t.Cleanup(func() { i18n.Use(original) })

	for _, loc := range []i18n.Locale{i18n.VI, i18n.EN} {
		i18n.Use(loc)

		got := map[string]string{
			"dispatchSummary": dispatchSummary("writer", "Write chapter 13"),
			"dispatchDetail":  dispatchDetail("Write chapter 13", "continue with the next chapter"),
			"retry.n":         retryPrefix(2, 0, 3*time.Second),
			"retry.max":       retryPrefix(2, 5, 0),
			"tool.commit":     displayToolName("commit_chapter", json.RawMessage(`{"chapter":13}`)),
			"tool.context":    displayToolName("novel_context", json.RawMessage(`{"chapter":13}`)),
			"tool.readdraft":  displayToolName("read_chapter", json.RawMessage(`{"chapter":12,"source":"draft"}`)),
			"tool.reviewarc":  displayToolName("save_review", json.RawMessage(`{"chapter":22,"scope":"arc","verdict":"accept"}`)),
			"tool.reviewch":   displayToolName("save_review", json.RawMessage(`{"chapter":7,"scope":"chapter"}`)),
		}
		for _, tool := range []string{"plan_chapter", "read_chapter", "novel_context", "save_review", "commit_chapter"} {
			got["header."+tool] = toolDisplays[tool].header()
		}

		for name, s := range got {
			if hanInDisplay.MatchString(s) {
				t.Errorf("locale=%s %s 渲染出汉字：%q", loc, name, s)
			}
			if strings.TrimSpace(s) == "" {
				t.Errorf("locale=%s %s 渲染为空", loc, name)
			}
		}
	}
}

// TestToolHeaderResolvesPerCall 锁死"文案在使用时解析"这条纪律：
// toolDisplays 是包级 map，若把文案在 init 期塞进字段，/config 热切换界面语言后
// 流式输出会永远停留在启动时的语言——不报错，只是悄悄不跟随。
func TestToolHeaderResolvesPerCall(t *testing.T) {
	original := i18n.Current()
	t.Cleanup(func() { i18n.Use(original) })

	i18n.Use(i18n.EN)
	en := toolDisplays["novel_context"].header()
	i18n.Use(i18n.VI)
	vi := toolDisplays["novel_context"].header()

	if en == vi {
		t.Fatalf("切换界面语言后 header 未变化，说明文案在 init 期定型：%q", en)
	}
	for _, s := range []string{en, vi} {
		if !strings.HasPrefix(s, "✻ ") {
			t.Errorf("header 应保留版式符号，got %q", s)
		}
	}
}

package lang_test

import (
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/lang"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/stylestat"
)

// 本文件验证的是"接线是否真的通了"，而不是 lang 包内部逻辑。
//
// 动机：加一个配置字段很容易，让它真的改变机械行为很容易漏。这些测试从
// 配置出发，穿过各个消费方，断言行为确实随语言改变——漏接一处就红。

// TestConfigResolvesLanguage：配置字段能一路解析到 Profile，且缺省安全。
func TestConfigResolvesLanguage(t *testing.T) {
	var cfg bootstrap.Config
	cfg.FillDefaults()
	if cfg.Language != string(lang.Default) {
		t.Fatalf("FillDefaults 后 Language = %q，期望 %q", cfg.Language, lang.Default)
	}
	if cfg.Lang().Code() != lang.ZH {
		t.Fatalf("默认配置应解析为中文，得到 %q", cfg.Lang().Code())
	}

	cfg.Language = "vi"
	if cfg.Lang().Code() != lang.VI {
		t.Fatalf("Language=vi 未解析为越南语，得到 %q", cfg.Lang().Code())
	}

	cfg.Language = "gibberish"
	if cfg.Lang().Code() != lang.ZH {
		t.Fatalf("非法语言应回落到中文，得到 %q", cfg.Lang().Code())
	}
}

// TestSystemDefaultsFollowLanguage：机械基线黑名单随语言切换。
// 旧实现下越南语书拿到的是中文黑名单——等于没有黑名单。
func TestSystemDefaultsFollowLanguage(t *testing.T) {
	zh := rules.SystemDefaults(lang.Resolve("zh")).Structured
	vi := rules.SystemDefaults(lang.Resolve("vi")).Structured
	en := rules.SystemDefaults(lang.Resolve("en")).Structured

	if !contains(zh.ForbiddenPhrases, "某种程度上") {
		t.Error("中文基线丢了套句")
	}
	if !contains(vi.ForbiddenPhrases, "điều đáng chú ý là") {
		t.Error("越南语基线没有本语言套句")
	}
	if !contains(en.ForbiddenPhrases, "it is worth noting that") {
		t.Error("英语基线没有本语言套句")
	}
	// 三套表必须互不相同，否则说明某个语言没真正接上
	if sameSet(zh.ForbiddenPhrases, vi.ForbiddenPhrases) {
		t.Error("越南语基线与中文完全相同，说明未按语言分流")
	}
}

// TestRuleCheckerCatchesLocalizedSlop：黑名单换语言后，机械检查真的能抓到。
// 这是端到端的一条线：lang → rules.SystemDefaults → rules.Check。
func TestRuleCheckerCatchesLocalizedSlop(t *testing.T) {
	viText := "Điều đáng chú ý là anh ấy đã đi. Theo một cách nào đó, mọi thứ đều ổn."
	viRules := rules.SystemDefaults(lang.Resolve("vi")).Structured
	if got := rules.Check(viText, viRules); len(got) == 0 {
		t.Error("越南语套句未被机械检查抓到")
	}
	// 同一段文本用中文基线检查必须一条不报——证明旧实现确实是静默失效
	zhRules := rules.SystemDefaults(lang.Resolve("zh")).Structured
	if got := rules.Check(viText, zhRules); len(got) != 0 {
		t.Errorf("中文基线不该在越南语正文上报违规，得到 %d 条", len(got))
	}
}

// TestStylestatFollowsLanguage：全书文体统计随语言产出非空结果。
// 旧实现下越南语正文的 Patterns 恒为空，diag 一路绿灯却什么都没查。
func TestStylestatFollowsLanguage(t *testing.T) {
	viChapters := repeat(
		"Anh không phải không muốn nói, mà là không dám nói. "+
			"Khẽ nhíu mày, tim thắt lại. Dường như mọi thứ đã thay đổi.", 5)

	withVI := stylestat.Compute(stylestat.Input{Chapters: viChapters, Lang: lang.Resolve("vi")})
	if withVI == nil || len(withVI.Patterns) == 0 {
		t.Fatal("越南语统计未匹配到任何句式模式——机械防线没接上")
	}

	withZH := stylestat.Compute(stylestat.Input{Chapters: viChapters, Lang: lang.Resolve("zh")})
	if withZH != nil && len(withZH.Patterns) != 0 {
		t.Error("中文模式表不该在越南语正文上匹配")
	}
}

// TestStylestatPhrasesAreWordsNotFragments：越南语短语必须是词，不是字符碎片。
func TestStylestatPhrasesAreWordsNotFragments(t *testing.T) {
	chapters := repeat("Anh ấy gật đầu rồi bước ra ngoài trong im lặng.", 20)
	stats := stylestat.Compute(stylestat.Input{Chapters: chapters, Lang: lang.Resolve("vi")})
	if stats == nil || len(stats.TopPhrases) == 0 {
		t.Skip("样本未达到频次阈值，短语挖掘未产出")
	}
	for _, ph := range stats.TopPhrases {
		if !strings.Contains(ph.Text, " ") {
			t.Errorf("越南语短语 %q 不含空格，说明仍在按字符切 n-gram", ph.Text)
		}
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, s := range a {
		if !contains(b, s) {
			return false
		}
	}
	return true
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

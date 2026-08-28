package lang

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestZeroValueIsChinese 钉死零值安全：Profile 内含 *regexp.Regexp，
// 零值上调用任何方法都不得 panic，且行为必须等价于 Default。
func TestZeroValueIsChinese(t *testing.T) {
	var zero Profile
	def := Resolve("")

	if zero.Code() != def.Code() {
		t.Fatalf("零值 Code = %q，期望 %q", zero.Code(), def.Code())
	}
	if zero.CountWords("一二三") != 3 {
		t.Fatalf("零值 CountWords = %d，期望 3", zero.CountWords("一二三"))
	}
	if zero.BCP47() != "zh-CN" {
		t.Fatalf("零值 BCP47 = %q", zero.BCP47())
	}
	// 不得 panic
	zero.SplitSentences("甲。乙。")
	zero.StylePatterns()
	zero.DefaultRules()
	zero.ChapterHeading(1, "起")
	zero.OutputDirective()
	if zero.TitlePrefixRe() == nil || zero.OpeningTimeRe() == nil {
		t.Fatal("零值的正则不得为 nil")
	}
}

// TestChineseWordCountUnchanged 保证存量中文书的字数口径逐字节不变——
// Progress.TotalWordCount 是持久化事实，换算口径就等于损坏历史数据。
func TestChineseWordCountUnchanged(t *testing.T) {
	zh := Resolve("zh")
	for _, s := range []string{
		"他推开门，风雪灌了进来。",
		"第 1 章 起势\n\n少年抬头。",
		"",
	} {
		if got, want := zh.CountWords(s), utf8.RuneCountInString(s); got != want {
			t.Fatalf("CountWords(%q) = %d，期望与 RuneCount 一致 = %d", s, got, want)
		}
	}
}

// TestWordCountByLanguage 是这次改动的核心动机：越南语/英语按词计，
// 不是按字符计。用字符数当词数会让字数规则整体失真约 5 倍。
func TestWordCountByLanguage(t *testing.T) {
	cases := []struct {
		code string
		text string
		want int
	}{
		{"vi", "Anh đẩy cửa ra, gió tuyết ùa vào.", 8},
		{"en", "He pushed the door open and the snow came in.", 10},
		{"vi", "   nhiều   khoảng   trắng   ", 3},
		{"en", "", 0},
	}
	for _, c := range cases {
		if got := Resolve(c.code).CountWords(c.text); got != c.want {
			t.Errorf("%s CountWords(%q) = %d，期望 %d", c.code, c.text, got, c.want)
		}
	}

	// 同一段越南语文本：字符数远大于词数，正是失真的来源。
	vi := Resolve("vi")
	text := "Anh đẩy cửa ra, gió tuyết ùa vào."
	if vi.CountWords(text) >= utf8.RuneCountInString(text) {
		t.Fatal("越南语词数应显著小于字符数，否则说明仍在按字符计")
	}
}

// TestStylePatternsActuallyMatch 是防"静默失效"的关键测试：
// 每种语言的模式表必须真的能在该语言的文本上匹配到。
// 旧实现下越南语/英语正文恒返回 0 条，diag 一路绿灯却什么都没检查。
func TestStylePatternsActuallyMatch(t *testing.T) {
	samples := map[string]string{
		"zh": "他不是不想说，而是不敢说。眼中闪过一丝犹豫，心头一紧。",
		"vi": "Anh không phải không muốn nói, mà là không dám nói. Khẽ nhíu mày, tim thắt lại.",
		"en": "It was not that he would not speak, but that he could not. His eyes narrowed; his heart clenched.",
	}
	for code, text := range samples {
		p := Resolve(code)
		hits := 0
		for _, pat := range p.StylePatterns() {
			if pat.Re.MatchString(text) {
				hits++
			}
		}
		if hits == 0 {
			t.Errorf("%s：模式表在本语言样本上零匹配，机械防线等于没有", code)
		}
	}
}

// TestPatternsDoNotCrossLanguages 保证语言表之间不串味：
// 中文模式不该在英语正文上乱报，反之亦然。
func TestPatternsDoNotCrossLanguages(t *testing.T) {
	en := Resolve("en")
	zhText := "他不是不想说，而是不敢说。"
	for _, pat := range en.StylePatterns() {
		if pat.Re.MatchString(zhText) {
			t.Errorf("英语模式 %q 匹配了中文正文", pat.Name)
		}
	}
}

// TestSplitSentences 覆盖两套断句体系。
func TestSplitSentences(t *testing.T) {
	if got := Resolve("zh").SplitSentences("甲。乙！丙？"); len(got) != 3 {
		t.Errorf("中文断句 = %v，期望 3 句", got)
	}
	if got := Resolve("en").SplitSentences("One. Two! Three?"); len(got) != 3 {
		t.Errorf("英语断句 = %v，期望 3 句", got)
	}
	if got := Resolve("vi").SplitSentences("Một. Hai! Ba?"); len(got) != 3 {
		t.Errorf("越南语断句 = %v，期望 3 句", got)
	}
}

// TestChapterHeading 钉死导出标题的语言化。
func TestChapterHeading(t *testing.T) {
	cases := []struct{ code, title, want string }{
		{"zh", "起势", "第 7 章 起势"},
		{"zh", "", "第 7 章"},
		{"vi", "Khởi đầu", "Chương 7: Khởi đầu"},
		{"vi", "", "Chương 7"},
		{"en", "Opening", "Chapter 7: Opening"},
		{"en", "", "Chapter 7"},
	}
	for _, c := range cases {
		if got := Resolve(c.code).ChapterHeading(7, c.title); got != c.want {
			t.Errorf("%s ChapterHeading = %q，期望 %q", c.code, got, c.want)
		}
	}
}

// TestTitlePrefixMatchesOwnHeading 保证"标题前缀识别"与"标题生成"同源：
// 导出器生成的标题必须能被同一 Profile 的正则认出来，否则剥重复标题会失效。
func TestTitlePrefixMatchesOwnHeading(t *testing.T) {
	for _, p := range Supported() {
		heading := p.ChapterHeading(12, "X")
		if !p.TitlePrefixRe().MatchString(heading) {
			t.Errorf("%s：自己生成的标题 %q 不被自己的 TitlePrefixRe 识别", p.Code(), heading)
		}
		if !p.TitlePrefixRe().MatchString("## " + heading) {
			t.Errorf("%s：Markdown 标题形式 %q 不被识别", p.Code(), "## "+heading)
		}
	}
}

// TestOutputDirective：**每种**创作语言都必须拿到指令，且点名语言与字数口径。
//
// 中文曾被特判为"不注入"（提示词本来就是中文，省前缀字节）。assets 全量英文化
// 之后这条特判会让中文书拿到英文提示词却没有任何语言指令——模型多半直接写英文，
// 不报错、只是产出物换了语言。这里把"中文也必须注入"钉死，防止特判被复活。
func TestOutputDirective(t *testing.T) {
	for _, code := range []string{"zh", "vi", "en"} {
		p := Resolve(code)
		d := p.OutputDirective()
		if d == "" {
			t.Fatalf("%s 必须注入语言指令", code)
		}
		if !strings.Contains(d, p.DisplayName()) {
			t.Errorf("%s 的指令未点名语言：%q", code, d)
		}
		if !strings.Contains(d, p.CountUnit()) {
			t.Errorf("%s 的指令未说明字数口径", code)
		}
	}
}

// TestResolveFallback：未知值不得返回零值 Profile，必须落到 Default。
func TestResolveFallback(t *testing.T) {
	for _, s := range []string{"", "  ", "klingon", "zz"} {
		if got := Resolve(s).Code(); got != Default {
			t.Errorf("Resolve(%q) = %q，期望回落到 %q", s, got, Default)
		}
	}
	// 别名要认得
	for in, want := range map[string]Code{
		"ZH": ZH, "zh-CN": ZH, "Chinese": ZH,
		"VI": VI, "Vietnamese": VI, "tieng viet": VI,
		"EN": EN, "English": EN,
	} {
		if got := Resolve(in).Code(); got != want {
			t.Errorf("Resolve(%q) = %q，期望 %q", in, got, want)
		}
	}
	if IsSupported("klingon") {
		t.Error("IsSupported 不该认未知语言")
	}
}

// TestDefaultRulesAreIsolated 保证返回的是副本：调用方改动不得污染全局表。
func TestDefaultRulesAreIsolated(t *testing.T) {
	p := Resolve("en")
	a := p.DefaultRules()
	a.FatigueWords["suddenly"] = 999
	a.ForbiddenPhrases[0] = "mutated"

	b := p.DefaultRules()
	if b.FatigueWords["suddenly"] == 999 {
		t.Error("FatigueWords 被调用方改动污染")
	}
	if b.ForbiddenPhrases[0] == "mutated" {
		t.Error("ForbiddenPhrases 被调用方改动污染")
	}
}

// TestEveryLanguageIsComplete 防止新增语言时漏填字段——漏填的代价是静默失效。
func TestEveryLanguageIsComplete(t *testing.T) {
	for _, p := range Supported() {
		name := string(p.Code())
		if p.DisplayName() == "" || p.EnglishName() == "" || p.BCP47() == "" || p.CountUnit() == "" {
			t.Errorf("%s：显示字段有缺", name)
		}
		if len(p.StylePatterns()) == 0 {
			t.Errorf("%s：模式表为空", name)
		}
		r := p.DefaultRules()
		if len(r.ForbiddenPhrases) == 0 || len(r.FatigueWords) == 0 {
			t.Errorf("%s：机械基线黑名单为空", name)
		}
		if p.ShortEndingLimit() <= 0 {
			t.Errorf("%s：ShortEndingLimit 未设置", name)
		}
		lo, hi := p.GramRange()
		if lo <= 0 || hi < lo {
			t.Errorf("%s：GramRange 非法 (%d,%d)", name, lo, hi)
		}
	}
}

// Package lang 是创作语言的唯一事实源：所有依赖语言的机械行为都从这里取值，
// 不在各自的包里各写一份。
//
// 定位（architecture.md §2.5 铁律四）：本包只固化**可证明的语言事实**——
// 一个"字"怎么数、一句话在哪里断、章节标题长什么样、哪些定长串是本语言的 AI 套句。
// 它不做文学判断：写得好不好、这句是不是套话，仍然归 Worker / Editor 裁定。
//
// 为什么必须有这一层：原先 WordCount = RuneCountInString、stylestat 正则、
// rules.SystemDefaults 的黑名单都隐含"正文是简体中文"。换成越南语/英语后这些
// 检查**不报错，只是恒返回零**——机械防线静默失效，比直接报错危险得多。
//
// 扩展新语言：在 profiles 里加一个 Profile 字面量即可，无需改任何调用方。
package lang

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Code 是语言标识，同时是 config.json 里 "language" 字段的取值。
type Code string

const (
	ZH Code = "zh" // 简体中文
	VI Code = "vi" // Tiếng Việt
	EN Code = "en" // English
)

// Default 是未配置时的语言。保持 ZH 以确保存量书目零行为变化。
const Default = ZH

// GramMode 决定高频短语挖掘的切分单位。
type GramMode int

const (
	// GramRune 按字符切 n-gram：适用于无词间空格的语言（中文）。
	GramRune GramMode = iota
	// GramWord 按空格分词切 n-gram：适用于拉丁/越南语书写系统。
	GramWord
)

// Pattern 是一条句式模式的事实定义：名字给人看，正则给代码数。
type Pattern struct {
	Name string
	Re   *regexp.Regexp
}

// DefaultRules 是本语言的机械基线黑名单，喂给 rules.SystemDefaults。
type DefaultRules struct {
	// ForbiddenPhrases 定长固定串的 AI 套句，出现即违规。
	ForbiddenPhrases []string
	// FatigueWords 疲劳词 → 单章允许出现次数上限。
	FatigueWords map[string]int
}

// Profile 是一门语言的全部机械行为。字段私有，只经方法暴露，
// 避免调用方绕过语义直接读正则。
type Profile struct {
	code        Code
	displayName string
	nativeName  string
	bcp47       string
	countUnit   string

	gramMode  GramMode
	gramMin   int
	gramMax   int
	shortEnd  int
	repeatMin int
	sentence  *regexp.Regexp
	titleRe   *regexp.Regexp
	openingRe *regexp.Regexp
	patterns  []Pattern
	rules     DefaultRules

	chapterWord string // "第 %d 章" 的语言化形式由 ChapterHeading 组装
}

// resolved 让 Profile 的零值可用：未初始化的 Profile 等价于 Default。
//
// 这不是可有可无的便利——Profile 内含 *regexp.Regexp，零值上直接调用会 panic。
// 让零值退化为中文（而不是崩溃）符合本项目的失败姿态：机械层出错时回到已知安全
// 的默认口径，并且用户在第一章就能从产出语言看出配置没生效。
func (p Profile) resolved() Profile {
	if p.code == "" {
		return profiles[Default]
	}
	return p
}

// Code 返回语言标识。零值 Profile 报告 Default。
func (p Profile) Code() Code { return p.resolved().code }

// DisplayName 返回给界面用的语言名（本语言书写）。
func (p Profile) DisplayName() string { return p.resolved().nativeName }

// EnglishName 返回英文名，供日志与配置回显使用。
func (p Profile) EnglishName() string { return p.resolved().displayName }

// BCP47 返回 EPUB / XHTML 用的语言标签。
func (p Profile) BCP47() string { return p.resolved().bcp47 }

// CountUnit 是字数的量词（"字" / "từ" / "words"），用于回显与提示词。
func (p Profile) CountUnit() string { return p.resolved().countUnit }

// CountWords 按本语言语义计算篇幅。
//
// 中文：一个字符即一个字，与历史实现逐字节等价（存量 Progress 不失真）。
// 越南语/英语：按空白分词计词——"3000 từ" 约等于 15000 字符，两者差 5 倍，
// 用字符数当词数会让用户的字数规则和 diag 的字数异常判定整体失真。
func (p Profile) CountWords(s string) int {
	p = p.resolved()
	if p.gramMode == GramRune {
		return utf8.RuneCountInString(s)
	}
	return len(strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r)
	}))
}

// SplitSentences 按本语言的句末标点断句，返回去空后的句子列表。
func (p Profile) SplitSentences(s string) []string {
	p = p.resolved()
	var out []string
	for _, seg := range p.sentence.Split(s, -1) {
		if t := strings.TrimSpace(seg); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// StylePatterns 返回本语言的 AI 文风句式模式表。
func (p Profile) StylePatterns() []Pattern { return p.resolved().patterns }

// DefaultRules 返回本语言的机械基线黑名单。
func (p Profile) DefaultRules() DefaultRules {
	p = p.resolved()
	out := DefaultRules{
		ForbiddenPhrases: append([]string(nil), p.rules.ForbiddenPhrases...),
		FatigueWords:     make(map[string]int, len(p.rules.FatigueWords)),
	}
	for k, v := range p.rules.FatigueWords {
		out.FatigueWords[k] = v
	}
	return out
}

// GramMode / GramRange 供短语挖掘选择切分单位与 n-gram 长度。
func (p Profile) GramMode() GramMode    { return p.resolved().gramMode }
func (p Profile) GramRange() (int, int) { r := p.resolved(); return r.gramMin, r.gramMax }

// ShortEndingLimit 是"短结尾"的阈值，单位与 CountWords 一致。
func (p Profile) ShortEndingLimit() int { return p.resolved().shortEnd }

// RepeatSentenceMin 是"值得统计的重复句"的最短长度，单位与 CountWords 一致。
// 太短的句子（"他点头。"/"He nodded."）天然高频，不是复读证据。
func (p Profile) RepeatSentenceMin() int { return p.resolved().repeatMin }

// SplitWords 按空白切词，供词级 n-gram 使用。中文档案返回单字切分，
// 让调用方可以用同一套代码路径处理两种书写系统。
func (p Profile) SplitWords(s string) []string {
	if p.resolved().gramMode == GramRune {
		out := make([]string, 0, len(s))
		for _, r := range s {
			out = append(out, string(r))
		}
		return out
	}
	return strings.FieldsFunc(s, unicode.IsSpace)
}

// TitlePrefixRe 匹配"第N章 / Chương N / Chapter N"这类机制前缀。
//
// 两处共用同一份事实：stylestat 用它统计 writer 是否混用标题形式，
// 导出器用它剥掉正文里重复的标题行。因此它必须能认出**本 Profile 自己
// 生成的标题**（TestTitlePrefixMatchesOwnHeading 钉死这条），也要容忍
// writer 手写的紧凑形式（"第12章"）。
func (p Profile) TitlePrefixRe() *regexp.Regexp { return p.resolved().titleRe }

// OpeningTimeRe 匹配以时间词开篇的迹象（章章"夜色降临"是结构固化信号）。
func (p Profile) OpeningTimeRe() *regexp.Regexp { return p.resolved().openingRe }

// ChapterLabel 是纯语言事实：本语言怎么称呼"第 N 章"。
// 不含标题、不含分隔——分隔宽度是**排版**决定，归各导出格式自己拿主意
// （TXT 用两格做视觉沟槽，EPUB 用一格），语言层不该替它们选。
func (p Profile) ChapterLabel(n int) string {
	p = p.resolved()
	if p.code == ZH {
		return fmt.Sprintf("第 %d 章", n)
	}
	return fmt.Sprintf("%s %d", p.chapterWord, n)
}

// TitleJoiner 是标签与标题之间的标点。中文靠空格分隔不加标点，
// 越南语/英语的书面惯例是冒号。返回值不含空格。
func (p Profile) TitleJoiner() string {
	if p.resolved().code == ZH {
		return ""
	}
	return ":"
}

// ChapterHeading 组装紧凑形式的章节标题（EPUB 用；TXT 自行加宽沟槽）。
// title 为空时只输出标签。
func (p Profile) ChapterHeading(n int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return p.ChapterLabel(n)
	}
	return p.ChapterLabel(n) + p.TitleJoiner() + " " + title
}

// VolumeLabel 是纯语言事实："第 N 卷" / "Quyển N" / "Volume N"，不含标题与分隔。
func (p Profile) VolumeLabel(n int) string {
	switch p.resolved().code {
	case ZH:
		return fmt.Sprintf("第 %d 卷", n)
	case VI:
		return fmt.Sprintf("Quyển %d", n)
	default:
		return fmt.Sprintf("Volume %d", n)
	}
}

// VolumeHeading 组装紧凑形式的卷分隔标题（EPUB 用；TXT 自行加宽沟槽）。
func (p Profile) VolumeHeading(n int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return p.VolumeLabel(n)
	}
	return p.VolumeLabel(n) + p.TitleJoiner() + " " + title
}

// BookTitleLine 组装 book.md 的标题行。中文用书名号《》，拉丁书写系统没有这个
// 约定，硬套只会让越南语/英语的产物看起来像排版事故。
func (p Profile) BookTitleLine(title string) string {
	title = strings.TrimSpace(title)
	if p.resolved().code == ZH {
		return "# 《" + title + "》"
	}
	return "# " + title
}

// SynopsisLabel 是 book.md 里"简介"那一节的标题。它是**产物**的一部分
// （读者会看到），不是界面文案，所以归 lang 而不是 i18n。
func (p Profile) SynopsisLabel() string {
	switch p.resolved().code {
	case ZH:
		return "简介"
	case VI:
		return "Giới thiệu"
	default:
		return "Synopsis"
	}
}

// CoverLabel / TocLabel 是 EPUB 导航里读者看得见的固定标签。
// 它们和正文一样属于产出物，不是界面文案——导出越南语小说却在目录页
// 写"目录"是产出物缺陷，不是 i18n 的范畴。
func (p Profile) CoverLabel() string {
	switch p.resolved().code {
	case ZH:
		return "封面"
	case VI:
		return "Bìa sách"
	default:
		return "Cover"
	}
}

func (p Profile) TocLabel() string {
	switch p.resolved().code {
	case ZH:
		return "目录"
	case VI:
		return "Mục lục"
	default:
		return "Contents"
	}
}

// OutputDirective 是注入所有创作 Worker 系统提示词的语言硬约束。
//
// 它是**约束**不是**风格**：正文语言不能由模型即兴决定，也不随用户
// 需求里偶然出现的外语词漂移。
//
// 为什么中文也必须注入（曾经不注入）：提示词与参考资料本来是中文，中文书
// 不写指令也不会跑偏，所以早先中文返回空串以省下前缀字节。assets 全量英文化
// 之后这个前提没了——中文书拿到的是一份**英文提示词加零条语言指令**，模型
// 极可能直接写出英文正文。这类失效不报错，只是产出物悄悄换了语言，
// 因此宁可多花几十个 token，也要让每种语言都拿到明确指令。
//
// 指令本身用英文书写：它是英文提示词的一部分，混用语言只会让模型在语言上摇摆。
func (p Profile) OutputDirective() string {
	p = p.resolved()
	// 英语的 nativeName 与 displayName 相同，不重复成 "English (English)"。
	name := p.nativeName
	if p.displayName != p.nativeName {
		name = p.nativeName + " (" + p.displayName + ")"
	}
	return fmt.Sprintf(`## Output language (hard constraint, overrides every other requirement)

The **prose, chapter titles, character names, place names, outline text, summaries and review notes** of this book are written in **%s**, without exception.

- This prompt and the reference material are written in English; that is your working language, not the language of the book. **Everything you produce is written in %s.**
- Words from another language appearing in the user's requirement describe what they want; they are not an instruction to write the book in that language.
- Any length target is measured in **%s**, as counted in %s.
- Keep a proper noun in its original script only when the user explicitly asks for that; localize everything else.
- Punctuation, quotation marks and dialogue formatting follow the written conventions of %s.`,
		name, p.nativeName, p.countUnit, p.nativeName, p.nativeName)
}

// WithOutputDirective 把语言硬约束追加到一份系统提示词末尾。
//
// 为什么在**末尾**：提示词缓存的收益来自请求前缀逐字节稳定
// （docs/prompt-cache-design.md §4）。语言指令是本书级常量，挂在尾部既不破坏
// 内置提示词的既有前缀，也让"动态内容进尾部"这条纪律继续成立。
//
// 为什么要有这个方法而不是各调用点自己拼：凡是"让模型产出会落盘的文本"的路径
// 都必须带指令——Worker、导入管线、仿写画像、修订分析都算。少接一处不会报错，
// 只会让那条路径的产出物悄悄换成提示词的语言（英文）。集中成一个方法，
// 新增路径时照抄一行即可，不必重新想一遍规则。
func (p Profile) WithOutputDirective(prompt string) string {
	directive := p.OutputDirective()
	if directive == "" {
		return prompt
	}
	return prompt + "\n\n" + directive
}

// Resolve 把配置值解析为 Profile。空值、未知值一律回落到 Default，
// 绝不因为配置错字让创作中断——错字的代价是语言回到中文，用户立刻能看见。
func Resolve(s string) Profile {
	key := Code(strings.ToLower(strings.TrimSpace(s)))
	switch key {
	case "zh-cn", "zh_cn", "chinese", "中文":
		key = ZH
	case "vietnamese", "tiếng việt", "tieng viet":
		key = VI
	case "english":
		key = EN
	}
	if p, ok := profiles[key]; ok {
		return p
	}
	return profiles[Default]
}

// IsSupported 报告 s 是否解析到一个显式声明的语言（用于配置校验回显）。
func IsSupported(s string) bool {
	key := Code(strings.ToLower(strings.TrimSpace(s)))
	_, ok := profiles[key]
	return ok
}

// Supported 按稳定顺序返回全部语言，供设置向导与 /config 列表使用。
func Supported() []Profile {
	return []Profile{profiles[ZH], profiles[VI], profiles[EN]}
}

var profiles = map[Code]Profile{
	ZH: {
		code:        ZH,
		displayName: "Simplified Chinese",
		nativeName:  "简体中文",
		bcp47:       "zh-CN",
		countUnit:   "字",
		chapterWord: "第",
		gramMode:    GramRune,
		gramMin:     3,
		gramMax:     6,
		shortEnd:    30,
		repeatMin:   12,
		sentence:    regexp.MustCompile(`[。！？\n]+`),
		titleRe:     regexp.MustCompile(`^#{0,2}\s*第\s*[零〇一二三四五六七八九十百千万\d]+\s*章`),
		openingRe:   regexp.MustCompile(`夜|清晨|黎明|天亮|醒来|晨光|一整夜`),
		patterns: []Pattern{
			{"矫正句『不是…(而)是…』", regexp.MustCompile(`不是[^。！？\n]{1,24}?[，、]?(?:而)?是`)},
			{"计时量词『X息/X瞬』", regexp.MustCompile(`[一两二三四五六七八九十几数半][息瞬]`)},
			{"明喻『像一/仿佛/如同/宛如』", regexp.MustCompile(`像一|仿佛|如同|宛如`)},
			{"沉默节拍『沉默了/没有说话/没有回头』", regexp.MustCompile(`沉默了|没有说话|没有回头`)},
			{"神态模板『眼中闪过/嘴角勾起/咬了咬唇』", regexp.MustCompile(`眼[中底]闪过|目光一凝|瞳孔一缩|眼眶微红|嘴角[微轻一]?[勾扬翘]|咬了咬唇|不可置信`)},
			{"躯体反应『心头一紧/身子一颤/倒吸凉气』", regexp.MustCompile(`心头一[紧沉颤]|身子一[颤震僵]|倒吸(?:了)?一口凉气`)},
			{"思维标记『心想/意识到/感到/觉得』", regexp.MustCompile(`心想|意识到|感到|觉得`)},
			{"抽象套话『一种说不出的/的意义在于』", regexp.MustCompile(`一种说不出的|说不清[的道]|的意义在于|真正的[^。！？\n]{1,10}是`)},
		},
		rules: DefaultRules{
			ForbiddenPhrases: []string{"某种程度上", "值得注意的是", "不知为何", "五味杂陈"},
			FatigueWords: map[string]int{
				"不禁": 1, "竟然": 1, "仿佛": 2, "此外": 1, "然而": 2,
				"一丝": 2, "一抹": 2, "一缕": 2, "宛如": 1, "不由得": 1,
				"像一": 3, "沉默了": 2, "没有说话": 2, "几息": 3, "一息": 3, "数息": 2,
			},
		},
	},

	VI: {
		code:        VI,
		displayName: "Vietnamese",
		nativeName:  "Tiếng Việt",
		bcp47:       "vi",
		countUnit:   "từ",
		chapterWord: "Chương",
		gramMode:    GramWord,
		gramMin:     2,
		gramMax:     5,
		shortEnd:    12,
		repeatMin:   6,
		sentence:    regexp.MustCompile(`[.!?…]+[\s\n]+|[.!?…]+$|\n+`),
		titleRe:     regexp.MustCompile(`(?i)^#{0,2}\s*chương\s+[\dIVXLCDM]+`),
		openingRe:   regexp.MustCompile(`(?i)đêm|sáng sớm|rạng đông|bình minh|tỉnh dậy|thức dậy|ánh ban mai|suốt đêm`),
		patterns: []Pattern{
			{"Câu đính chính『không phải… mà là…』", regexp.MustCompile(`(?i)không phải[^.!?\n]{1,40}?mà là`)},
			{"Câu tăng tiến『không chỉ… mà còn…』", regexp.MustCompile(`(?i)không chỉ[^.!?\n]{1,40}?mà còn`)},
			{"So sánh『như thể/tựa như/dường như』", regexp.MustCompile(`(?i)như thể|tựa như|dường như|hệt như`)},
			{"Nhịp im lặng『im lặng/không nói gì/không quay đầu』", regexp.MustCompile(`(?i)im lặng một lúc|không nói gì|không quay đầu lại`)},
			{"Mẫu thần thái『khẽ nhíu mày/khóe môi nhếch/ánh mắt thoáng』", regexp.MustCompile(`(?i)khẽ nhíu mày|nhíu mày|khóe môi[^.!?\n]{0,8}nhếch|ánh mắt[^.!?\n]{0,10}thoáng|đồng tử[^.!?\n]{0,6}co`)},
			{"Phản ứng cơ thể『tim thắt lại/rùng mình/hít sâu』", regexp.MustCompile(`(?i)tim[^.!?\n]{0,6}(thắt lại|đập thình thịch)|rùng mình|hít (một hơi )?thật sâu|lạnh sống lưng`)},
			{"Dấu hiệu tư duy『nghĩ thầm/nhận ra/cảm thấy』", regexp.MustCompile(`(?i)nghĩ thầm|chợt nhận ra|nhận ra rằng|cảm thấy rằng`)},
			{"Sáo ngữ trừu tượng『cảm giác khó tả/ý nghĩa nằm ở』", regexp.MustCompile(`(?i)cảm giác khó tả|khó tả thành lời|ý nghĩa (thật sự |nằm) ở|điều thật sự[^.!?\n]{1,16}là`)},
		},
		rules: DefaultRules{
			ForbiddenPhrases: []string{
				"theo một cách nào đó", "một cách nào đó", "điều đáng chú ý là",
				"không biết vì sao", "trăm mối cảm xúc đan xen", "cảm xúc lẫn lộn khó tả",
			},
			FatigueWords: map[string]int{
				"không khỏi": 1, "quả nhiên": 1, "dường như": 2, "ngoài ra": 1, "tuy nhiên": 2,
				"một thoáng": 2, "một tia": 2, "một luồng": 2, "tựa như": 1, "bất giác": 1,
				"như thể": 3, "im lặng một lúc": 2, "không nói gì": 2, "khẽ": 4, "chợt": 3,
			},
		},
	},

	EN: {
		code:        EN,
		displayName: "English",
		nativeName:  "English",
		bcp47:       "en",
		countUnit:   "words",
		chapterWord: "Chapter",
		gramMode:    GramWord,
		gramMin:     2,
		gramMax:     5,
		shortEnd:    12,
		repeatMin:   6,
		sentence:    regexp.MustCompile(`[.!?…]+[\s\n]+|[.!?…]+$|\n+`),
		titleRe:     regexp.MustCompile(`(?i)^#{0,2}\s*chapter\s+[\dIVXLCDM]+`),
		openingRe:   regexp.MustCompile(`(?i)\b(night|dawn|daybreak|sunrise|woke up|awoke|first light|all night)\b`),
		patterns: []Pattern{
			{"Corrective 'not X, but Y'", regexp.MustCompile(`(?i)\bnot\s+[^.!?\n]{1,40}?,?\s+but\s+(?:rather\s+)?`)},
			{"Escalation 'not only... but also'", regexp.MustCompile(`(?i)\bnot only\b[^.!?\n]{1,40}?\bbut also\b`)},
			{"Simile 'like a/as if/as though'", regexp.MustCompile(`(?i)\b(like a|as if|as though|much like)\b`)},
			{"Silence beat 'said nothing/did not look back'", regexp.MustCompile(`(?i)\b(said nothing|didn't say a word|did not look back|fell silent)\b`)},
			{"Expression template 'eyes narrowed/jaw tightened'", regexp.MustCompile(`(?i)\b(eyes? narrowed|jaw (tightened|clenched)|lips? (quirked|curved)|raised an eyebrow|pupils contracted)\b`)},
			{"Body reaction 'heart clenched/shiver down the spine'", regexp.MustCompile(`(?i)\b(heart (clenched|skipped|hammered)|shivers? (ran|down)|sharp intake of breath|breath he didn't know)\b`)},
			{"Thought marker 'realized/felt/thought to himself'", regexp.MustCompile(`(?i)\b(realized that|thought to (him|her)self|couldn't help but|felt that)\b`)},
			{"Abstract filler 'a testament to/the point is'", regexp.MustCompile(`(?i)\b(a testament to|indescribable feeling|what (really|truly) matter(s|ed) is|the (real|true) meaning of)\b`)},
		},
		rules: DefaultRules{
			ForbiddenPhrases: []string{
				"in some sense", "it is worth noting that", "it's worth noting that",
				"for some reason", "a whirlwind of emotions", "a mix of emotions",
				"a testament to", "in the realm of", "navigate the complexities",
				"little did he know", "little did she know",
			},
			FatigueWords: map[string]int{
				"suddenly": 2, "indeed": 1, "seemingly": 2, "moreover": 1, "however": 2,
				"a hint of": 2, "a trace of": 2, "a wave of": 2, "as though": 1, "somehow": 1,
				"like a": 4, "fell silent": 2, "said nothing": 2, "merely": 2, "utterly": 2,
			},
		},
	},
}

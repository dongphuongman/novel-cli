// Package stylestat 对已写正文做全书级风格统计，产出纯事实。
//
// 动机：弧内评审窗口（~10 章）对全书级模式固化天然失明——句式 tic 章均几十次、
// 章末形态同构、跨章复读，单章看每处都"正常"，只有全书统计能暴露。统计归代码
// （确定性、零幻觉），裁定归 LLM（editor 按数字判维度分，writer 据此自避免）。
// Compute 供离线评测一次性全量计算；运行时使用 Tracker 按章节增量维护。
package stylestat

import (
	"sort"
	"strings"
	"unicode"

	"github.com/dongphuongman/novel-cli/internal/lang"
)

// minChapters 少于此章数不出统计——样本太小，频率没有意义。
const minChapters = 5

// phraseWindow 动态短语挖掘只看最近 N 章：writer 需要避免的是"现在的口头禅"。
const phraseWindow = 20

// Input 统计输入。Chapters 按章号升序；Stopwords 为角色名等专有名词，
// 动态短语挖掘时跳过（出场人名天然高频，不是文风问题）。
type Input struct {
	Chapters  []string
	Titles    []string
	Stopwords []string

	// Lang 决定断句、n-gram 切分单位与模式表。零值 = 中文（lang.Profile 零值安全），
	// 保证旧调用方行为不变。
	Lang lang.Profile
}

// Stats 全书风格统计结果。所有字段都是事实计数，不含任何裁定或指令。
type Stats struct {
	Chapters          int            `json:"chapters"`
	Patterns          []PatternStat  `json:"patterns,omitempty"`
	TopPhrases        []PhraseStat   `json:"top_phrases,omitempty"`
	RepeatedSentences []SentenceStat `json:"repeated_sentences,omitempty"`
	Ending            EndingStat     `json:"ending"`
	OpeningTimeRate   float64        `json:"opening_time_rate"`
	TitleFormats      *TitleStat     `json:"title_formats,omitempty"`
}

// PatternStat 固定句式模式类的全书计数（通用 AI 文风 tic）。
type PatternStat struct {
	Name       string  `json:"name"`
	Total      int     `json:"total"`
	PerChapter float64 `json:"per_chapter"`
}

// PhraseStat 最近 phraseWindow 章内挖掘出的高频短语。
type PhraseStat struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}

// SentenceStat 跨章逐字重复的长句（复读交代的直接证据）。
type SentenceStat struct {
	Text     string `json:"text"`
	Chapters int    `json:"chapters"`
	Count    int    `json:"count"`
}

// EndingStat 章末行形态分布。短结尾本身合法，全书同构才是问题。
type EndingStat struct {
	ShortRatio  float64 `json:"short_ratio"`
	MedianRunes int     `json:"median_runes"`
}

// TitleStat 章节标题「第N章」前缀混用计数（混用=机制痕迹暴露在产物里）。
type TitleStat struct {
	WithPrefix    int `json:"with_prefix"`
	WithoutPrefix int `json:"without_prefix"`
}

// 句式模式、断句正则、标题前缀、短结尾阈值全部迁往 internal/lang——它们是
// **语言事实**，不是本包的策略。中文那一套在越南语/英语正文上恒零匹配，
// 留在这里就等于让机械防线在换语言后静默失效（这正是迁移的动因）。

// Compute 计算全书风格统计；章数不足时返回 nil。
func Compute(in Input) *Stats {
	n := len(in.Chapters)
	if n < minChapters {
		return nil
	}
	all := strings.Join(in.Chapters, "\n")
	lp := in.Lang // 零值安全：等价于中文

	s := &Stats{Chapters: n}
	for _, def := range lp.StylePatterns() {
		total := len(def.Re.FindAllStringIndex(all, -1))
		if total == 0 {
			continue
		}
		s.Patterns = append(s.Patterns, PatternStat{
			Name:       def.Name,
			Total:      total,
			PerChapter: round1(float64(total) / float64(n)),
		})
	}
	s.TopPhrases = minePhrases(recentWindow(in.Chapters), in.Stopwords, lp)
	s.RepeatedSentences = repeatedSentences(in.Chapters, lp)
	s.Ending = endingShape(in.Chapters, lp)
	s.OpeningTimeRate = openingTimeRate(in.Chapters, lp)
	s.TitleFormats = titleFormats(in.Titles, lp)
	return s
}

func recentWindow(chapters []string) []string {
	if len(chapters) <= phraseWindow {
		return chapters
	}
	return chapters[len(chapters)-phraseWindow:]
}

// minePhrases 在窗口内挖掘高频短语。
//
// 切分单位随语言而变（lang.GramMode）：中文按字切 3-6 字 n-gram；越南语/英语
// 按词切 2-5 词 n-gram。用字符 n-gram 去切"anh ấy"只会得到"nh ấ"这类碎片，
// 统计出来的"口头禅"全是噪声——这正是必须分语言的原因。
//
// 过滤：含标点/空白、首尾虚词、命中专有名词；去重：与已选短语互为子串的丢弃。
func minePhrases(chapters []string, stopwords []string, lp lang.Profile) []PhraseStat {
	text := strings.Join(chapters, "\n")
	threshold := max(8, len(chapters)/2)
	lo, hi := lp.GramRange()

	counts := make(map[string]int)
	if lp.GramMode() == lang.GramWord {
		countWordGrams(counts, text, lo, hi)
	} else {
		countRuneGrams(counts, text, lo, hi)
	}

	stopGrams := stopwordBigrams(stopwords)
	type cand struct {
		text  string
		count int
	}
	var cands []cand
	for g, c := range counts {
		if c < threshold || hitStopword(g, stopGrams) {
			continue
		}
		cands = append(cands, cand{g, c})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].count != cands[j].count {
			return cands[i].count > cands[j].count
		}
		// 同频取更长的（信息量更大），再按字典序稳定排序
		if len(cands[i].text) != len(cands[j].text) {
			return len(cands[i].text) > len(cands[j].text)
		}
		return cands[i].text < cands[j].text
	})

	var out []PhraseStat
	for _, c := range cands {
		if len(out) >= 8 {
			break
		}
		dup := false
		for _, picked := range out {
			if strings.Contains(picked.Text, c.text) || strings.Contains(c.text, picked.Text) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, PhraseStat{Text: c.text, Count: c.count})
		}
	}
	return out
}

// countRuneGrams 按字切 n-gram（中文等无词间空格的书写系统）。
func countRuneGrams(counts map[string]int, text string, lo, hi int) {
	runes := []rune(text)
	for size := lo; size <= hi; size++ {
		for i := 0; i+size <= len(runes); i++ {
			gram := runes[i : i+size]
			if !validGram(gram) {
				continue
			}
			counts[string(gram)]++
		}
	}
}

// countWordGrams 按词切 n-gram（拉丁/越南语书写系统）。
// 词先剥掉两端标点并转小写：句首的 "Anh" 与句中的 "anh" 是同一个口头禅。
func countWordGrams(counts map[string]int, text string, lo, hi int) {
	words := make([]string, 0, len(text)/4)
	for _, w := range strings.FieldsFunc(text, unicode.IsSpace) {
		w = strings.ToLower(strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		}))
		if w != "" {
			words = append(words, w)
		}
	}
	for size := lo; size <= hi; size++ {
		for i := 0; i+size <= len(words); i++ {
			gram := words[i : i+size]
			if wordGramEdgeStop[gram[0]] || wordGramEdgeStop[gram[len(gram)-1]] {
				continue
			}
			counts[strings.Join(gram, " ")]++
		}
	}
}

// wordGramEdgeStop 是首尾虚词表：以它们开头或结尾的 n-gram 是语法碎片，
// 不是文风短语。与中文的 gramEdgeStop 同一用途、同一取舍（宁可过滤偏严）。
var wordGramEdgeStop = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`
		the a an of to in on at and or but is was were be been being
		he she it they we you i him her them his hers its their our my
		that this these those with for from as by
		anh cô nó họ chúng tôi bạn tôi mình của và với là đã đang sẽ
		một những các cái này kia đó thì mà nhưng ở trên trong cho từ
	`) {
		m[w] = true
	}
	return m
}()

// gramEdgeStop 首尾为这些虚词/代词的 n-gram 不是文风短语，跳过。
const gramEdgeStop = "的了着是在和与就也都还又把被他她它我你这那"

func validGram(gram []rune) bool {
	for _, r := range gram {
		if r < 0x4E00 || r > 0x9FFF { // 仅纯汉字片段
			return false
		}
	}
	if strings.ContainsRune(gramEdgeStop, gram[0]) || strings.ContainsRune(gramEdgeStop, gram[len(gram)-1]) {
		return false
	}
	return true
}

// stopwordBigrams 把专有名词拆成 2 字片段：人名常以部分形式入文
// （"九渊负手"含"九渊"），按整名匹配会漏网。宁可过滤偏严——短语事实少一条
// 无碍，人名混进口头禅清单才是噪声。
func stopwordBigrams(stopwords []string) []string {
	var grams []string
	for _, w := range stopwords {
		runes := []rune(strings.TrimSpace(w))
		if len(runes) < 2 {
			continue
		}
		for i := 0; i+2 <= len(runes); i++ {
			grams = append(grams, string(runes[i:i+2]))
		}
	}
	return grams
}

func hitStopword(gram string, stopGrams []string) bool {
	for _, g := range stopGrams {
		if strings.Contains(gram, g) {
			return true
		}
	}
	return false
}

// repeatedSentences 找跨 ≥3 章逐字重复的 ≥12 字句子，按次数取 top 5。
func repeatedSentences(chapters []string, lp lang.Profile) []SentenceStat {
	type rec struct {
		count    int
		chapters map[int]struct{}
	}
	seen := make(map[string]*rec)
	for ci, text := range chapters {
		for sent, count := range chapterSentenceCounts(text, lp) {
			r := seen[sent]
			if r == nil {
				r = &rec{chapters: make(map[int]struct{})}
				seen[sent] = r
			}
			r.count += count
			r.chapters[ci] = struct{}{}
		}
	}

	var out []SentenceStat
	for sent, r := range seen {
		if len(r.chapters) < 3 {
			continue
		}
		out = append(out, SentenceStat{Text: truncateRunes(sent, 40), Chapters: len(r.chapters), Count: r.count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Text < out[j].Text
	})
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

// trimWrappedQuotes 剥掉包裹引号：同一句台词带/不带前引号不应算成两条。
func trimWrappedQuotes(sentence string) string {
	return strings.Trim(strings.TrimSpace(sentence), `"“”‘’「」『』`)
}

func endingShape(chapters []string, lp lang.Profile) EndingStat {
	var lengths []int
	short := 0
	for _, text := range chapters {
		line := lastNonEmptyLine(text)
		if line == "" {
			continue
		}
		n := len([]rune(line))
		lengths = append(lengths, n)
		if n <= lp.ShortEndingLimit() {
			short++
		}
	}
	if len(lengths) == 0 {
		return EndingStat{}
	}
	sort.Ints(lengths)
	return EndingStat{
		ShortRatio:  round2(float64(short) / float64(len(lengths))),
		MedianRunes: lengths[len(lengths)/2],
	}
}

func openingTimeRate(chapters []string, lp lang.Profile) float64 {
	hit := 0
	for _, text := range chapters {
		if lp.OpeningTimeRe().MatchString(firstParagraph(text)) {
			hit++
		}
	}
	return round2(float64(hit) / float64(len(chapters)))
}

func titleFormats(titles []string, lp lang.Profile) *TitleStat {
	if len(titles) == 0 {
		return nil
	}
	t := &TitleStat{}
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		if lp.TitlePrefixRe().MatchString(title) {
			t.WithPrefix++
		} else {
			t.WithoutPrefix++
		}
	}
	// 只有混用才值得上报；统一格式不是事实意义上的问题
	if t.WithPrefix == 0 || t.WithoutPrefix == 0 {
		return nil
	}
	return t
}

func lastNonEmptyLine(text string) string {
	lines := strings.Split(text, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// firstParagraph 取第一个非空且非 Markdown 标题的行（章文件首行常是 # 标题）。
func firstParagraph(text string) string {
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }
func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

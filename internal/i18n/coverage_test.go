package i18n_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 本测试扫描界面层源码，禁止新的硬编码中文字符串溜回去。
//
// 为什么需要它：i18n 的失败模式不是报错，是"下一个改 TUI 的人顺手写了个中文
// 字面量"。没有这道闸，覆盖率会随时间悄悄退化，而且只有中文用户之外的人才看得见。
//
// 例外清单是**白名单而非黑名单**：新增例外必须在这里显式写明理由，
// 让"这条为什么不翻译"成为一次有记录的决定，而不是默认行为。

// uiPackages 是必须走 i18n 的界面层。
//
// cmd 不能漏：flag 解析错误、die() 的收尾提示、update 的回显都是用户可见输出。
// 早先这里确实漏了 cmd，结果配置错误正文已经是越南语，紧跟其后的
// "（详细错误已记录到 …）" 仍是中文——半句一种语言，比全中文更难堪。
var uiPackages = []string{
	"../entry/tui",
	"../entry/headless",
	"../entry/startup",
	"../bootstrap",
	"../../cmd/ainovel-cli",
}

// uiFiles 是**逐文件**纳入的界面层来源。
//
// 为什么不整包纳入 internal/host：该包同时承载控制流与展示层，绝大多数中文
// 字面量是 fmt.Errorf 与 slog——面向维护者的深层排障材料，按上方判定界线本就
// 不该翻译。整包纳入会逼出一份几百条的 allowed 清单，白名单一旦变成噪音就
// 不再是闸门。这里只纳入"产出物只喂给 TUI"的文件。
//
// 历史教训：这几个文件曾把工具行、DISPATCH 行、侧栏快照写成中文常量，
// 越南语界面下整块事件流仍是中文——用户看到的是半中半越的界面，
// 而 uiPackages 完全扫不到它们。
var uiFiles = []string{
	"../host/stream_extract.go",  // 流式工具输出的 header
	"../host/observer_tools.go",  // TOOL / DISPATCH 行摘要
	"../host/observer_events.go", // retry / context 事件文案
	"../host/observer_stream.go",
	"../host/events.go",

	// /import 与 /simulate 的进度显示：整条流程的用户可见输出都在这几个文件里。
	"../host/imp/runner.go",
	"../host/imp/state.go",
	"../host/imp/segment.go",
	"../host/imp/analyze.go",
	"../host/imp/synthesize.go",
	"../host/imp/call.go",
	"../host/imp/source.go",
	"../host/imp/publish.go",
	"../host/imp/workspace.go",
	"../host/sim/runner.go",
	"../host/sim/import.go",
}

// allowed 是允许保留中文字面量的位置与理由。
//
// 键是 "文件名:内容前缀"，值是理由。理由本身就是文档。
//
// 判定界线（新增例外前请先对照）：
//
//	slog.* .................. 面向**维护者**。tui.log 与 meta/diag-export.md 是用户贴到
//	                          GitHub issue 供维护者定位的材料（docs/observability.md §1）。
//	                          71 条日志统一中文，翻译其中几条只会割裂语料，不便检索。
//	发给模型的文本 .......... 与 assets/prompts/ 同为中文，是模型契约不是界面。
//	fmt.Errorf → die()/TUI ... 面向**用户**，必须走 i18n。
var allowed = map[string]string{}

// 汉字 + CJK 全角标点。只查汉字会漏掉 （）、。「」——它们同样让非中文界面
// 露馅，而且最爱藏在 fmt.Sprintf 的版式里。
// logCallRe 认出结构化日志调用。日志面向**维护者**，不受本测试约束——
// 把这条判定界线写成规则，而不是在 allowed 里逐条枚举：枚举会让白名单
// 涨到几十条噪音，白名单一旦不再可读，就没人会认真看它了。
var logCallRe = regexp.MustCompile(`(?:slog|log|logger\(\))\.(?:Debug|Info|Warn|Error)\(`)

// lineIndex 把字节偏移映射回所在整行，供上面的日志判定使用。
type lineIndex struct{ src string }

func lineStarts(src string) lineIndex { return lineIndex{src: src} }

func (l lineIndex) of(pos int) string {
	start := strings.LastIndexByte(l.src[:pos], '\n') + 1
	end := strings.IndexByte(l.src[pos:], '\n')
	if end < 0 {
		return l.src[start:]
	}
	return l.src[start : pos+end]
}

var hanRe = regexp.MustCompile(`[\p{Han}\x{3000}-\x{303F}\x{FF01}-\x{FF60}]`)

func TestUILayerHasNoHardcodedChinese(t *testing.T) {
	used := map[string]bool{}

	var paths []string
	for _, pkg := range uiPackages {
		entries, err := os.ReadDir(pkg)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", pkg, err)
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			paths = append(paths, filepath.Join(pkg, name))
		}
	}
	paths = append(paths, uiFiles...)

	for _, path := range paths {
		name := filepath.Base(path)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", path, err)
		}
		src := string(data)

		lines := lineStarts(src)
		for _, lit := range interpretedStrings(src) {
			if !hanRe.MatchString(lit.text) {
				continue
			}
			if logCallRe.MatchString(lines.of(lit.pos)) {
				continue // 结构化日志面向维护者，见上方判定界线
			}
			if key, why := lookupAllowed(name, lit.text); why != "" {
				used[key] = true
				continue
			}
			line := strings.Count(src[:lit.pos], "\n") + 1
			t.Errorf("%s:%d 硬编码中文界面文案，应走 i18n.T：%q\n"+
				"    如果这条确实不该翻译（发给模型 / 深层排障错误），"+
				"请在 coverage_test.go 的 allowed 里加一条并写明理由。",
				path, line, lit.text)
		}
	}

	// 例外清单必须保持精简：条目失效说明代码已改，应当删掉它。
	for key := range allowed {
		if !used[key] {
			t.Errorf("allowed 里的例外已失效，请删除：%s", key)
		}
	}
}

func lookupAllowed(file, lit string) (string, string) {
	for key, why := range allowed {
		f, prefix, ok := strings.Cut(key, ":")
		if !ok || f != file {
			continue
		}
		if strings.HasPrefix(lit, prefix) {
			return key, why
		}
	}
	return "", ""
}

// strLit 是一条解释型字符串字面量：内容与它在源码里的字节偏移。
type strLit struct {
	text string
	pos  int
}

// interpretedStrings 顺序扫描 Go 源码，只返回**解释型字符串**（"..."）。
//
// 为什么必须是逐字节词法扫描，而不是一条 `"..."` 正则：正则不认识 rune 字面量。
// stream_extract.go 这类字节级 tokenizer 里满是 '"' 和 '\\'，正则会把 rune 里的
// 引号当成字符串开头，从此**每一对引号都错位**——真正的字符串结束引号变成了
// 下一个字符串的开始引号，于是整片代码被当作"字符串内容"报成硬编码中文。
// 这不是漏报而是**成片误报**，会让整道闸门失去可信度，只能靠关掉它来止血。
//
// 注释、raw string（`...`）与 rune 字面量都被跳过：它们不受本测试约束。
func interpretedStrings(src string) []strLit {
	var out []strLit
	for i, n := 0, len(src); i < n; {
		switch {
		case src[i] == '/' && i+1 < n && src[i+1] == '/':
			if j := strings.IndexByte(src[i:], '\n'); j < 0 {
				i = n
			} else {
				i += j
			}
		case src[i] == '/' && i+1 < n && src[i+1] == '*':
			if j := strings.Index(src[i+2:], "*/"); j < 0 {
				i = n
			} else {
				i += 2 + j + 2
			}
		case src[i] == '`':
			if j := strings.IndexByte(src[i+1:], '`'); j < 0 {
				i = n
			} else {
				i += 1 + j + 1
			}
		case src[i] == '\'':
			i = skipQuoted(src, i, '\'')
		case src[i] == '"':
			end := skipQuoted(src, i, '"')
			if end > i+1 {
				out = append(out, strLit{text: src[i+1 : end-1], pos: i})
			}
			i = end
		default:
			i++
		}
	}
	return out
}

// skipQuoted 从 start（引号本身）跳到闭合引号之后的下一个字节，
// 按 Go 词法处理反斜杠转义；未闭合时返回源码长度。
func skipQuoted(src string, start int, quote byte) int {
	for j := start + 1; j < len(src); {
		switch src[j] {
		case '\\':
			j += 2
		case quote:
			return j + 1
		case '\n':
			return j // 未闭合：Go 里不合法，止步于行尾避免吞掉整个文件
		default:
			j++
		}
	}
	return len(src)
}

// scanRoots 是所有可能调用 i18n.T 的包。
var scanRoots = []string{"../entry", "../bootstrap", "../../cmd"}

// TestNoInitTimeTranslation 禁止在包级 var/const 里调用 i18n.T。
//
// 这条约束来自一个真实缺陷：statusDisplay 是包级 map，在 init 期求值，
// 而 i18n.Use(cfg.UILang()) 要等配置读完才跑。结果界面整体是越南语，
// 唯独右上角状态徽章永远是中文——不报错，只是悄悄不跟随。
//
// 同样的道理也堵死了"界面语言热切换"：init 期定死的值不会随 /config 变化。
// 正确写法是包成函数，每次渲染时取。
func TestNoInitTimeTranslation(t *testing.T) {
	declRe := regexp.MustCompile(`^(func|var|const|type)\b`)

	for _, root := range scanRoots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			decl, declLine := "", ""
			for i, line := range strings.Split(string(data), "\n") {
				if m := declRe.FindString(line); m != "" {
					decl, declLine = m, strings.TrimSpace(line)
				}
				if !strings.Contains(line, "i18n.T(") {
					continue
				}
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				if decl == "var" || decl == "const" {
					t.Errorf("%s:%d 在包级 %s 里调用 i18n.T——init 期语言尚未确定，"+
						"该值会被永久钉死。请改成函数。\n    声明：%s",
						path, i+1, decl, declLine)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("扫描 %s 失败: %v", root, err)
		}
	}
}

// hostVisibleRe 抓 internal/host 里**会显示到事件流**的字面量。
//
// 为什么不像别的包那样整包扫：host 有 827 条中文字符串，绝大多数是内部错误、
// 注释和发给模型的文本，不该翻译。真正露给用户的只有 Event 的 Summary/Detail
// 和 runObservedDecision 的裁定标签——这条正则精确挑出这批，其余不碰。
//
// 这条规则来自一次真实反馈：界面整体已是越南语，事件流却仍在打
// "启动裁定""开始创作"。Phase 2 当时把 host 整个划到范围外并记为待办，
// 用户的截图证明那个划分是错的。
// 匹配**所有**事件文案点位（已本地化的和还是字面量的都算），再单独看字面量里
// 有没有中文。只匹配字面量的话，全部改完后命中数归零，"正则空转"自检反而误报。
var hostVisibleRe = regexp.MustCompile(
	`(?:Summary|Detail):\s*(?:fmt\.Sprintf\()?(?:(i18n\.T\()|"((?:[^"\\]|\\.)*)")` +
		`|runObservedDecision\([^,]+,\s*(?:(i18n\.T\()|"((?:[^"\\]|\\.)*)")`)

func TestHostEventTextIsLocalized(t *testing.T) {
	entries, err := os.ReadDir("../host")
	if err != nil {
		t.Fatalf("读取 ../host 失败: %v", err)
	}
	checked := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join("../host", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", path, err)
		}
		src := string(data)
		for _, m := range hostVisibleRe.FindAllStringSubmatchIndex(src, -1) {
			checked++
			for _, g := range [][2]int{{m[4], m[5]}, {m[8], m[9]}} {
				if g[0] < 0 {
					continue
				}
				lit := src[g[0]:g[1]]
				if !hanRe.MatchString(lit) {
					continue
				}
				line := strings.Count(src[:m[0]], "\n") + 1
				t.Errorf("%s:%d 事件文案硬编码中文，应走 i18n.T：%q\n"+
					"    Summary/Detail 与裁定标签都会出现在事件流里，用户读得到。",
					path, line, lit)
			}
		}
	}
	if checked == 0 {
		t.Fatal("一条事件文案都没扫到——正则大概过时了，这条测试正在空转")
	}
}

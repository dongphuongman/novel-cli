package tui

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/dongphuongman/novel-cli/internal/i18n"
)

// 侧栏字段标签走 fieldLabelStyle 的固定列宽。标签超宽时 lipgloss 会**折行**
// （不是截断），值就被挤到第二行，整张表错位。
//
// 中文标签 2~4 字一直没触发，换成越南语/英语后 6 条标签当场折行——单测全绿、
// 翻译也没错，只有真跑起来才看得见。这条测试把它变成可自动发现的约束。
//
// 自维护：标签清单从源码里现扫，新增字段自动纳入，不靠人肉维护列表。
var fieldLabelCallRe = regexp.MustCompile(`render(?:Highlight)?Field\(i18n\.T\("([^"]+)"\)`)

func sidebarFieldLabelKeys(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("读取包目录: %v", err)
	}
	seen := map[string]bool{}
	var keys []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatalf("读取 %s: %v", name, err)
		}
		for _, m := range fieldLabelCallRe.FindAllStringSubmatch(string(data), -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				keys = append(keys, m[1])
			}
		}
	}
	if len(keys) == 0 {
		t.Fatal("一个 renderField 标签都没扫到——正则大概过时了，" +
			"这条测试正在空转，比没有更危险")
	}
	return keys
}

func TestSidebarFieldLabelsNeverWrap(t *testing.T) {
	defer i18n.Use(i18n.Default)
	keys := sidebarFieldLabelKeys(t)

	for _, loc := range i18n.Supported() {
		i18n.Use(loc)
		for _, key := range keys {
			label := i18n.T(key)
			rendered := renderField(label, "X")
			// renderField 末尾固定带一个换行；多出来的就是折行。
			if strings.Count(rendered, "\n") > 1 {
				t.Errorf("%s / %s = %q 撑破了标签列，值被挤到下一行：\n%s\n"+
					"    请缩短译文（侧栏是窄列，加宽会挤掉所有行的取值空间）。",
					loc, key, label, rendered)
			}
		}
	}
}

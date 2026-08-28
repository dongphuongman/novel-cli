package i18n

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// verbRe 抓取 fmt 动词。界面文案只该用最基础的几种，复杂格式属于逻辑不属于翻译。
var verbRe = regexp.MustCompile(`%[-+# 0-9.*]*[a-zA-Z%]`)

func verbs(s string) []string {
	var out []string
	for _, v := range verbRe.FindAllString(s, -1) {
		if v == "%%" {
			continue
		}
		out = append(out, v)
	}
	return out
}

// TestCatalogsHaveIdenticalKeys 是本包最重要的一条约束。
//
// 没有它，"漏翻一句"只会在用户点到那个界面时才暴露，而且表现是突然冒出中文，
// 不是报错。有了它，漏翻在 CI 就红。
func TestCatalogsHaveIdenticalKeys(t *testing.T) {
	base := keys(ZH)
	sort.Strings(base)

	for _, loc := range Supported() {
		if loc == ZH {
			continue
		}
		got := keys(loc)
		sort.Strings(got)

		missing := diff(base, got)
		extra := diff(got, base)
		for _, k := range missing {
			t.Errorf("%s 词表缺 key: %s", loc, k)
		}
		for _, k := range extra {
			t.Errorf("%s 词表多出 key（中文没有）: %s", loc, k)
		}
	}
}

// TestFormatVerbsMatchAcrossCatalogs 防止翻译时改动或漏掉 %s/%d。
// 参数个数或类型对不上会让界面输出 %!s(MISSING)，比没翻译还糟。
func TestFormatVerbsMatchAcrossCatalogs(t *testing.T) {
	for _, key := range keys(ZH) {
		want := verbs(catalogs[ZH][key])
		for _, loc := range Supported() {
			if loc == ZH {
				continue
			}
			got := verbs(catalogs[loc][key])
			if len(got) != len(want) {
				t.Errorf("%s / %s：动词数量不符，中文 %v，本语言 %v", loc, key, want, got)
				continue
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("%s / %s：第 %d 个动词是 %s，中文是 %s", loc, key, i+1, got[i], want[i])
				}
			}
		}
	}
}

// TestNoEmptyValues 空串会走回落路径显示中文，等于静默漏翻。
func TestNoEmptyValues(t *testing.T) {
	for _, loc := range Supported() {
		for k, v := range catalogs[loc] {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s / %s：文案为空", loc, k)
			}
		}
	}
}

// TestNonChineseCatalogsAreActuallyTranslated 抓"复制粘贴中文当占位"的情况。
// 少数 key（如刻意三语并列的语言选择标题、纯符号）豁免。
func TestNonChineseCatalogsAreActuallyTranslated(t *testing.T) {
	exempt := map[string]bool{
		"setup.step.uilang": true, // 刻意三语并列：还没选语言时谁都得看得懂
	}
	han := regexp.MustCompile(`\p{Han}`)
	for _, loc := range []Locale{VI, EN} {
		for k, v := range catalogs[loc] {
			if exempt[k] {
				continue
			}
			if han.MatchString(v) {
				t.Errorf("%s / %s：仍含汉字，疑似未翻译：%q", loc, k, v)
			}
		}
	}
}

// TestTFallsBackInsteadOfFailing：缺失 key 返回 key 本身，不 panic 不返回空。
func TestTFallsBackInsteadOfFailing(t *testing.T) {
	defer Use(Default)

	Use(VI)
	if got := T("khong.ton.tai.key"); got != "khong.ton.tai.key" {
		t.Errorf("缺失 key 应原样返回，得到 %q", got)
	}
	if got := T("common.default"); got == "" || got == "common.default" {
		t.Errorf("已有 key 未取到译文，得到 %q", got)
	}
}

// TestTFormats：带参数的文案能正确展开。
func TestTFormats(t *testing.T) {
	defer Use(Default)
	Use(EN)
	got := T("setup.done.model", "gpt-4o")
	if !strings.Contains(got, "gpt-4o") {
		t.Errorf("参数未展开：%q", got)
	}
	if strings.Contains(got, "%!") {
		t.Errorf("格式化出错：%q", got)
	}
}

// TestResolve 覆盖别名、大小写、未知值与 auto。
func TestResolve(t *testing.T) {
	cases := map[string]Locale{
		"":           Default,
		"  ":         Default,
		"zh":         ZH,
		"ZH":         ZH,
		"zh-CN":      ZH,
		"Chinese":    ZH,
		"vi":         VI,
		"Vietnamese": VI,
		"tieng viet": VI,
		"en":         EN,
		"English":    EN,
		"en_US":      EN,
		"klingon":    Default,
	}
	for in, want := range cases {
		if got := Resolve(in); got != want {
			t.Errorf("Resolve(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// TestFromEnv：auto 按 POSIX 环境变量推断，认不出则回落。
func TestFromEnv(t *testing.T) {
	cases := map[string]Locale{
		"vi_VN.UTF-8":    VI,
		"en_US.UTF-8":    EN,
		"zh_CN.UTF-8":    ZH,
		"vi":             VI,
		"fr_FR.UTF-8":    Default,
		"C":              Default,
		"en-GB":          EN,
		"zh_TW.UTF-8@tw": ZH,
	}
	for env, want := range cases {
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_MESSAGES", "")
		t.Setenv("LANG", env)
		if got := Resolve(Auto); got != want {
			t.Errorf("LANG=%q → %q，期望 %q", env, got, want)
		}
	}
}

// TestEnvPrecedence：LC_ALL 优先于 LC_MESSAGES 优先于 LANG。
func TestEnvPrecedence(t *testing.T) {
	t.Setenv("LANG", "zh_CN.UTF-8")
	t.Setenv("LC_MESSAGES", "en_US.UTF-8")
	t.Setenv("LC_ALL", "vi_VN.UTF-8")
	if got := Resolve(Auto); got != VI {
		t.Errorf("LC_ALL 未取得最高优先级，得到 %q", got)
	}
}

// TestUseRejectsUnknown：Use 收到未知值时回落而不是留下坏状态。
func TestUseRejectsUnknown(t *testing.T) {
	defer Use(Default)
	Use(Locale("klingon"))
	if got := Current(); got != Default {
		t.Errorf("Use 未知语言后 Current = %q，期望 %q", got, Default)
	}
}

func diff(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	var out []string
	for _, s := range a {
		if !set[s] {
			out = append(out, s)
		}
	}
	return out
}

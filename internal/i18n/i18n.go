// Package i18n 是界面语言层：TUI 外壳、设置向导、命令帮助与状态栏的文案来源。
//
// 与 internal/lang 的分工是本包存在的前提，两者不可混为一谈：
//
//	internal/lang  →  产出物语言（正文、大纲、摘要、导出），并决定机械口径
//	internal/i18n  →  界面语言（用户在终端里读到的字），不影响任何创作事实
//
// 一个越南用户完全可能用越南语界面写一本中文小说，反之亦然。把两者绑在一起
// 是设计错误。
//
// 定位（architecture.md §2.3 观察层只观察）：本包是纯展示设施，绝不参与控制流。
// 它不返回错误——文案缺失时逐级回落，界面少一句翻译不该让创作中断。
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// Locale 是界面语言标识，同时是 config.json 里 "ui_language" 字段的取值。
type Locale string

const (
	ZH Locale = "zh"
	VI Locale = "vi"
	EN Locale = "en"
)

// Default 是未配置时的界面语言。保持 ZH：存量用户界面逐字不变。
const Default = ZH

// Auto 是显式请求按系统环境推断的取值。它是**选择加入**的：
// 不写 ui_language 的存量用户不会因为 $LANG=en_US 就突然换成英文界面。
const Auto = "auto"

//go:embed catalog/*.json
var catalogFS embed.FS

// catalogs 在 init 期一次性载入并冻结；运行期只读，无需加锁。
var catalogs = map[Locale]map[string]string{}

// current 用 atomic 持有，允许 /config 热切换界面语言而不与 TUI 渲染竞态。
var current atomic.Value // Locale

func init() {
	for _, loc := range []Locale{ZH, VI, EN} {
		data, err := catalogFS.ReadFile("catalog/" + string(loc) + ".json")
		if err != nil {
			// 词表是编译期嵌入资源，读不到说明构建被破坏，不是运行期状况。
			panic("i18n: 词表缺失 " + string(loc) + ": " + err.Error())
		}
		m := map[string]string{}
		if err := json.Unmarshal(data, &m); err != nil {
			panic("i18n: 词表格式错误 " + string(loc) + ": " + err.Error())
		}
		catalogs[loc] = m
	}
	current.Store(Default)
}

// Resolve 把配置值解析为 Locale。空值回落 Default；"auto" 按环境推断；
// 未知值回落 Default——界面语言写错不该让程序起不来。
func Resolve(s string) Locale {
	key := strings.ToLower(strings.TrimSpace(s))
	if key == Auto {
		return fromEnv()
	}
	switch Locale(key) {
	case ZH, VI, EN:
		return Locale(key)
	}
	switch key {
	case "zh-cn", "zh_cn", "chinese", "中文":
		return ZH
	case "vietnamese", "tiếng việt", "tieng viet", "vi-vn", "vi_vn":
		return VI
	case "english", "en-us", "en_us", "en-gb":
		return EN
	}
	return Default
}

// fromEnv 按 POSIX 惯例读 LC_ALL > LC_MESSAGES > LANG，取语言前缀。
// 认不出的环境一律回落 Default。
func fromEnv() Locale {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		// zh_CN.UTF-8 / vi_VN / en_US.UTF-8@euro → 取第一段的语言码
		prefix := strings.ToLower(raw)
		for _, sep := range []string{".", "@", "_", "-"} {
			if i := strings.Index(prefix, sep); i >= 0 {
				prefix = prefix[:i]
			}
		}
		switch Locale(prefix) {
		case ZH, VI, EN:
			return Locale(prefix)
		}
	}
	return Default
}

// Use 设置进程的界面语言。启动时调用一次；/config 热切换时可再次调用。
func Use(l Locale) {
	switch l {
	case ZH, VI, EN:
		current.Store(l)
	default:
		current.Store(Default)
	}
}

// Current 返回当前界面语言。
func Current() Locale {
	if v, ok := current.Load().(Locale); ok {
		return v
	}
	return Default
}

// T 取一条文案。args 非空时按 fmt.Sprintf 展开。
//
// 逐级回落：当前语言 → 中文 → key 本身。永不 panic、永不返回空串——
// 界面缺一句翻译是瑕疵，让创作中断才是事故。key 原样显示既不好看也不误导，
// 而且一眼就能看出是哪条没翻译。
func T(key string, args ...any) string {
	pattern, ok := catalogs[Current()][key]
	if !ok || pattern == "" {
		if pattern, ok = catalogs[Default][key]; !ok || pattern == "" {
			return key
		}
	}
	if len(args) == 0 {
		return pattern
	}
	return fmt.Sprintf(pattern, args...)
}

// Has 报告某个 key 是否存在于当前语言或中文兜底里，供测试与工具使用。
func Has(key string) bool {
	if _, ok := catalogs[Current()][key]; ok {
		return true
	}
	_, ok := catalogs[Default][key]
	return ok
}

// Supported 按稳定顺序返回全部界面语言。
func Supported() []Locale { return []Locale{ZH, VI, EN} }

// DisplayName 返回该语言用自己书写的名字，供 /config 与设置向导列表使用。
func DisplayName(l Locale) string {
	switch l {
	case VI:
		return "Tiếng Việt"
	case EN:
		return "English"
	default:
		return "简体中文"
	}
}

// keys 返回某语言词表的全部 key，供词表一致性测试使用。
func keys(l Locale) []string {
	out := make([]string, 0, len(catalogs[l]))
	for k := range catalogs[l] {
		out = append(out, k)
	}
	return out
}

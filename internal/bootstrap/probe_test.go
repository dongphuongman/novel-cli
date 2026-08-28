package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voocel/ainovel-cli/internal/i18n"
)

// ProbeUILanguage 的全部价值在于"读配置失败时错误信息还能读懂"。
// 这些用例守住的正是那个场景，而不是快乐路径。

func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".ainovel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ainovel", "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// chdir 把工作目录切到 dir，让 projectConfigPath() 指向它。
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestProbeUILanguage_ReadsProjectConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir()) // 隔离全局配置
	writeConfig(t, dir, `{"ui_language": "vi"}`)
	chdir(t, dir)

	if got := ProbeUILanguage(); got != i18n.VI {
		t.Fatalf("ProbeUILanguage() = %q，期望 %q", got, i18n.VI)
	}
}

// 项目级必须压过全局，与 LoadConfig 的优先级一致——否则错误信息的语言
// 会和随后的界面对不上，比全用中文更让人困惑。
func TestProbeUILanguage_ProjectOverridesGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `{"ui_language": "en"}`)

	dir := t.TempDir()
	writeConfig(t, dir, `{"ui_language": "vi"}`)
	chdir(t, dir)

	if got := ProbeUILanguage(); got != i18n.VI {
		t.Fatalf("项目级未压过全局：得到 %q，期望 %q", got, i18n.VI)
	}
}

// 只有全局配置时应当读到它——headless / CI 常见形态。
func TestProbeUILanguage_FallsBackToGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `{"ui_language": "en"}`)
	chdir(t, t.TempDir())

	if got := ProbeUILanguage(); got != i18n.EN {
		t.Fatalf("未读到全局配置：得到 %q，期望 %q", got, i18n.EN)
	}
}

// 核心用例：配置**语法坏掉**时不得 panic、不得阻塞，安静回落默认。
// 这正是 LoadConfig 会失败的那一刻，探测本身绝不能成为第二个故障点。
func TestProbeUILanguage_BrokenJSONFallsBack(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeConfig(t, dir, `{"ui_language": "vi",,, 这不是合法 JSON`)
	chdir(t, dir)

	if got := ProbeUILanguage(); got != i18n.Default {
		t.Fatalf("坏配置应回落默认：得到 %q，期望 %q", got, i18n.Default)
	}
}

// 全局坏、项目好：仍应读到项目的语言，不被坏文件带偏。
func TestProbeUILanguage_BrokenGlobalGoodProject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeConfig(t, home, `{ 坏掉的全局`)

	dir := t.TempDir()
	writeConfig(t, dir, `{"ui_language": "en"}`)
	chdir(t, dir)

	if got := ProbeUILanguage(); got != i18n.EN {
		t.Fatalf("坏全局把好项目带偏了：得到 %q，期望 %q", got, i18n.EN)
	}
}

// 完全没有配置文件（首次运行）时回落默认，不报错。
func TestProbeUILanguage_NoConfigAtAll(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	chdir(t, t.TempDir())

	if got := ProbeUILanguage(); got != i18n.Default {
		t.Fatalf("无配置应回落默认：得到 %q，期望 %q", got, i18n.Default)
	}
}

// 支持 JSONC 行注释——配置文件本来就允许注释，探测不能因此看不见字段。
func TestProbeUILanguage_HandlesJSONComments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	writeConfig(t, dir, "{\n  // 界面语言\n  \"ui_language\": \"vi\"\n}")
	chdir(t, dir)

	if got := ProbeUILanguage(); got != i18n.VI {
		t.Fatalf("带注释的配置未读到：得到 %q，期望 %q", got, i18n.VI)
	}
}

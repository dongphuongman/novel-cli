package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/host"
	"strings"
)

type fakeModelRuntime struct {
	providers   []string
	models      map[string][]host.ConfiguredModel
	curProvider string
	curModel    string
	thinking    map[string]string // role -> 存储的原始意图
	available   []agentcore.ThinkingLevel
	setCalls    []struct{ role, level string }
	switchCalls int
}

func (f *fakeModelRuntime) ConfiguredProviders() []string { return f.providers }
func (f *fakeModelRuntime) ConfiguredModelOptions(provider string) []host.ConfiguredModel {
	return f.models[provider]
}
func (f *fakeModelRuntime) CurrentModelSelection(role string) (string, string, bool) {
	return f.curProvider, f.curModel, true
}
func (f *fakeModelRuntime) AvailableThinking(role string) []agentcore.ThinkingLevel {
	return f.available
}
func (f *fakeModelRuntime) CurrentThinking(role string) string { return f.thinking[role] }
func (f *fakeModelRuntime) SwitchModel(role, provider, model string) error {
	f.switchCalls++
	f.curProvider, f.curModel = provider, model
	return nil
}
func (f *fakeModelRuntime) SetRoleThinking(role, level string) error {
	f.setCalls = append(f.setCalls, struct{ role, level string }{role, level})
	if f.thinking == nil {
		f.thinking = map[string]string{}
	}
	f.thinking[role] = level
	return nil
}

// 存储的强度意图高于当前模型能力、面板无法呈现时，用户不动强度字段直接应用，
// 不应把意图误抹成初始默认值。
func TestModelSwitchKeepsUnrepresentableThinkingIntent(t *testing.T) {
	rt := &fakeModelRuntime{
		providers:   []string{"proxy"},
		models:      map[string][]host.ConfiguredModel{"proxy": {{Name: "chat-only"}}},
		curProvider: "proxy", curModel: "chat-only",
		thinking:  map[string]string{"writer": "high"},
		available: nil, // 当前模型只有“继承”一档
	}
	st := newModelSwitchState(rt, "writer")
	if st.thinkingKey() != "" {
		t.Fatalf("high 无法呈现时面板应落在继承档，得到 %q", st.thinkingKey())
	}
	if err := st.apply(rt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(rt.setCalls) != 0 {
		t.Fatalf("未改动强度不应回写：%+v", rt.setCalls)
	}
	if rt.thinking["writer"] != "high" {
		t.Fatalf("意图被抹成 %q，应保留 high", rt.thinking["writer"])
	}
}

// 用户在面板里显式改动强度，则应回写为新值。
func TestModelSwitchAppliesExplicitThinkingChange(t *testing.T) {
	rt := &fakeModelRuntime{
		providers:   []string{"proxy"},
		models:      map[string][]host.ConfiguredModel{"proxy": {{Name: "m"}}},
		curProvider: "proxy", curModel: "m",
		thinking:  map[string]string{"writer": ""},
		available: []agentcore.ThinkingLevel{"low", "high"},
	}
	st := newModelSwitchState(rt, "writer")
	st.focus = modelFocusThinking
	st.cycle(1, rt) // 移动强度字段
	want := st.thinkingKey()
	if want == "" {
		t.Fatal("测试前置：应已移动到某个非空强度档")
	}
	if err := st.apply(rt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(rt.setCalls) != 1 || rt.setCalls[0].level != want {
		t.Fatalf("显式改动应回写 %q，得到 %+v", want, rt.setCalls)
	}
}

// TestModelFieldLabelWidthGrowsWithLabels：标签列宽必须跟着实际标签走。
//
// 这条来自实机缺陷：列宽写死 12，中文"推理强度:"占 9 格一直没事，换成越南语
// "Cường độ suy luận:"占 18 格后被 lipgloss 折行，直接把面板边框撑破。
// 单测当时全绿——翻译是对的，渲染是坏的。
func TestModelFieldLabelWidthGrowsWithLabels(t *testing.T) {
	cases := []struct {
		name   string
		labels []string
		want   int
	}{
		{"中文短标签保持历史列宽", []string{"角色", "Provider", "模型", "推理强度"}, modelFieldLabelBaseWidth},
		{"越南语长标签撑开列宽", []string{"Vai", "Provider", "Model", "Cường độ suy luận"}, 19},
		{"英语长标签撑开列宽", []string{"Role", "Provider", "Model", "Reasoning effort"}, 18},
		{"空标签集回落历史列宽", nil, modelFieldLabelBaseWidth},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := modelFieldLabelWidth(c.labels...); got != c.want {
				t.Errorf("modelFieldLabelWidth(%v) = %d，期望 %d", c.labels, got, c.want)
			}
		})
	}
}

// TestModelFieldNeverWraps 是上面那条的行为版：渲染出来必须是单行。
// 一旦折行，面板边框就会被撑破——这正是缺陷的可见形态。
func TestModelFieldNeverWraps(t *testing.T) {
	longest := "Cường độ suy luận"
	w := modelFieldLabelWidth("Vai", "Provider", "Model", longest)
	line := renderModelField(w, longest, "Mặc định (kế thừa)", false)
	if strings.Contains(line, "\n") {
		t.Fatalf("字段渲染出现折行，面板边框会被撑破：\n%s", line)
	}
}

// TestModelFieldsAlignAcrossRows：同一面板内所有行的取值必须起始于同一列，
// 否则视觉上就是一张错位的表。
func TestModelFieldsAlignAcrossRows(t *testing.T) {
	labels := []string{"Vai", "Provider", "Model", "Cường độ suy luận"}
	w := modelFieldLabelWidth(labels...)
	offset := -1
	for _, label := range labels {
		plain := ansi.Strip(renderModelField(w, label, "x", false))
		// 必须按显示列算：strings.Index 给的是字节下标，越南语一个字符占 2~3 字节，
		// 用字节下标比对会得出"不对齐"的假阳性（这条测试第一版就踩了）。
		got := lipgloss.Width(plain[:strings.Index(plain, "[")])
		if offset == -1 {
			offset = got
			continue
		}
		if got != offset {
			t.Errorf("标签 %q 的取值起始列 %d，与首行的 %d 不一致", label, got, offset)
		}
	}
}

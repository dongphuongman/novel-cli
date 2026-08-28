package host

import (
	"strings"
	"testing"

	"github.com/dongphuongman/novel-cli/internal/utils"
)

// feedThinkTags 按给定切片顺序喂入，返回过滤器实际放行的全部文本（含末尾 flush）。
func feedThinkTags(f *thinkTagFilter, chunks ...string) string {
	var out strings.Builder
	for _, c := range chunks {
		out.WriteString(f.feed(c))
	}
	out.WriteString(f.flush())
	return out.String()
}

func TestThinkTagFilterDropsEmptyThinkBlock(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   string
	}{
		{"整段到达", []string{"<think></think>正文"}, "正文"},
		{"带空白与换行", []string{"<think>\n\n</think>\n\n正文"}, "正文"},
		{"开标签被切开", []string{"<thi", "nk>", "</think>", "正文"}, "正文"},
		{"闭标签被切开", []string{"<think>\n", "</thi", "nk>", "正文"}, "正文"},
		{"逐字符到达", strings.Split("<think></think>正文", ""), "正文"},
		{"连续多对空标签", []string{"<think></think>\n<think>  </think>\n正文"}, "正文"},
		{"全空无正文", []string{"<think>\n</think>\n"}, ""},
		{"标签前有正文", []string{"前文<think></think>后文"}, "前文后文"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f thinkTagFilter
			if got := feedThinkTags(&f, tt.chunks...); got != tt.want {
				t.Fatalf("feed(%q) = %q, want %q", tt.chunks, got, tt.want)
			}
		})
	}
}

func TestThinkTagFilterKeepsNonEmptyThinkBlock(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		want   string
	}{
		{"整段到达", []string{"<think>我在想</think>正文"}, "<think>我在想</think>正文"},
		{"跨 chunk", []string{"<think>", "我在想", "</think>", "正文"}, "<think>我在想</think>正文"},
		{"空白后有内容", []string{"<think>\n 我在想\n</think>"}, "<think>\n 我在想\n</think>"},
		{"闭标签未到达", []string{"<think>我在想"}, "<think>我在想"},
		{"不是 think 的尖括号", []string{"a<b>c<thin"}, "a<b>c<thin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f thinkTagFilter
			if got := feedThinkTags(&f, tt.chunks...); got != tt.want {
				t.Fatalf("feed(%q) = %q, want %q", tt.chunks, got, tt.want)
			}
		})
	}
}

// 有内容的 <think> 必须边到边显示，不能憋到 </think> 才整段吐出，
// 否则中转（9router）把推理放在正文通道的模型就失去流式观感。
func TestThinkTagFilterStreamsNonEmptyThinkBlockIncrementally(t *testing.T) {
	var f thinkTagFilter
	if got := f.feed("<think>"); got != "" {
		t.Fatalf("开标签单独到达时 = %q, want 空（还不知道是不是空块）", got)
	}
	if got := f.feed("第一段"); got != "<think>第一段" {
		t.Fatalf("首段内容 = %q, want 立刻放行开标签与内容", got)
	}
	if got := f.feed("第二段"); got != "第二段" {
		t.Fatalf("续段 = %q, want 原样透传", got)
	}
	if got := f.feed("</think>"); got != "</think>" {
		t.Fatalf("闭标签 = %q, want 原样透传", got)
	}
}

func TestThinkTagFilterFlushReleasesHeldTail(t *testing.T) {
	var f thinkTagFilter
	if got := f.feed("正文<thi"); got != "正文" {
		t.Fatalf("feed = %q, want 只放行正文、扣住半截标签", got)
	}
	if got := f.flush(); got != "<thi" {
		t.Fatalf("flush = %q, want 半截标签补发", got)
	}
	if got := f.flush(); got != "" {
		t.Fatalf("二次 flush = %q, want 空（已复位）", got)
	}
}

func TestThinkTagFilterGivesUpOnOversizedBuffer(t *testing.T) {
	var f thinkTagFilter
	blank := strings.Repeat(" ", thinkFilterMaxHold)
	got := f.feed("<think>" + blank)
	if !strings.HasPrefix(got, thinkOpenTag) || len(got) != len(thinkOpenTag)+len(blank) {
		t.Fatalf("超限后 = %q(len=%d), want 放弃过滤、原样放行", got, len(got))
	}
}

// 流式面板层：空标签不得留下任何痕迹（含 ThinkingSep），
// 而规范返回 reasoning 字段的模型其思考段必须照旧带 ThinkingSep 显示。
func TestObserverEmitStreamDeltaFiltersEmptyThinkTags(t *testing.T) {
	newStreamObserver := func(out *strings.Builder) *observer {
		return &observer{
			emitEv: func(Event) {},
			emitD:  func(s string) { out.WriteString(s) },
			emitC:  func() {},
		}
	}

	t.Run("空标签整轮无输出", func(t *testing.T) {
		var out strings.Builder
		o := newStreamObserver(&out)
		for _, d := range []string{"<think>", "\n\n", "</think>", "\n"} {
			o.emitStreamDelta(d, false)
		}
		o.streamClear()
		if out.String() != "" {
			t.Fatalf("emitted %q, want 空", out.String())
		}
		if o.streamHasContent {
			t.Fatal("streamHasContent 被空标签置位，会误插段落分隔")
		}
	})

	t.Run("有内容的标签保留", func(t *testing.T) {
		var out strings.Builder
		o := newStreamObserver(&out)
		o.emitStreamDelta("<think>我在想</think>正文", false)
		if want := "<think>我在想</think>正文"; out.String() != want {
			t.Fatalf("emitted %q, want %q", out.String(), want)
		}
	})

	t.Run("规范 reasoning 通道不受影响", func(t *testing.T) {
		var out strings.Builder
		o := newStreamObserver(&out)
		o.emitStreamDelta("思考中", true)
		o.emitStreamDelta("正文", false)
		want := utils.ThinkingSep + "思考中" + utils.ThinkingSep + "正文"
		if out.String() != want {
			t.Fatalf("emitted %q, want %q", out.String(), want)
		}
	})

	t.Run("reasoning 通道插入前先放掉文本尾巴", func(t *testing.T) {
		var out strings.Builder
		o := newStreamObserver(&out)
		o.emitStreamDelta("正文<thi", false)
		o.emitStreamDelta("思考中", true)
		want := "正文<thi" + utils.ThinkingSep + "思考中"
		if out.String() != want {
			t.Fatalf("emitted %q, want %q", out.String(), want)
		}
	})
}

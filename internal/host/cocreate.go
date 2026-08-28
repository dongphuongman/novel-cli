package host

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/voocel/agentcore"
	"github.com/voocel/ainovel-cli/internal/bootstrap"
	"github.com/voocel/ainovel-cli/internal/store"
)

// 冷启动共创：从零澄清需求，产出整本书的创作指令。
const coCreateSystemPrompt = `You are a novel co-creation assistant. Your job is not to start writing the novel, but to help the user clarify what they want across a few short exchanges, and to keep maintaining a single creative brief that can be handed straight to the writing engine.

Every turn must be emitted strictly in the XML format below: four tags, in this order, each with a correct opening and closing tag.

<reply>
A natural reply for the user: respond to what they said, then ask at most 1 to 2 of the most important open questions. Once there is enough to start, tell them they can press Ctrl+S to begin.
Write this reply in the same language the user is writing in.
</reply>

<draft>
The complete current draft of the creative brief, in Markdown: start directly at level-2 headings, e.g. "## Theme", "## Key elements", "## Still to clarify"; list the points as bullets. Every turn **accumulates** on what is already there and absorbs the user's latest intent. Even when a turn adds nothing, write the whole draft out again verbatim — never abbreviate, never write a placeholder such as "(unchanged from last turn)".
</draft>
` + coCreateProtocolTail

// 阶段共创：小说已写了一部分，规划"后续阶段"的走向。调用方需把当前故事状态摘要
// 追加到本 prompt 之后（"## Current story state" 段），让模型在已写内容的基础上规划。
const stageCoCreateSystemPrompt = `You are a novel "stage co-creation" assistant. This novel is already partly written (see "Current story state" below). The user has paused to plan the next stretch with you before writing continues.

Your job is not to write prose, but to help the user think through where the next stretch goes (the next few chapters / the next arc / the next volume) across a few short exchanges, and to keep maintaining a "direction brief" the writing engine can act on.

Iron rule: every suggestion must be consistent with the plot, characters and foreshadowing already established in "Current story state". Never overturn or ignore what is already written. You plan only what comes next; you do not redesign the book.

Every turn must be emitted strictly in the XML format below: four tags, in this order, each with a correct opening and closing tag.

<reply>
A natural reply for the user: respond to what they said, then ask at most 1 to 2 of the most important open questions. Once the direction is clear enough, tell them they can press Ctrl+S to hand it to the writing engine and continue.
Write this reply in the same language the user is writing in.
</reply>

<draft>
The complete current "direction brief", in Markdown: start directly at level-2 headings, e.g. "## Where it goes next", "## Key turns", "## Foreshadowing to pay off", "## Pacing and length"; list the points as bullets. Every turn **accumulates** on what is already there and absorbs the user's latest intent. Even when a turn adds nothing, write the whole brief out again verbatim — never abbreviate, never write a placeholder such as "(unchanged from last turn)".
</draft>
` + coCreateProtocolTail

// coCreateProtocolTail 是两种共创模式共用的输出协议尾部（<ready> / <suggestions> + 输出规范）。
// 两模式只在开场语境与 <draft> 语义上不同，协议完全一致。
const coCreateProtocolTail = `
<ready>false</ready>

<suggestions>
1-3 lines of "what the user might want to say next", one per line, each starting with "- ". These are prompts for when the user is stuck;
pressing a number key drops one into the input box, where they can edit it before sending.

Requirements:
- Write them in the user's voice, as things the user would say to you — not as questions from the assistant.
- Keep each under about 12 words, vary the sentence shapes, avoid sameness.
- Offer a leaning, a choice or an added intent; do not write the user's whole setup for them in one line.
- Write them in the same language the user is writing in.
</suggestions>

Output rules:
- You must use the four XML tags: <reply> / <draft> / <ready> / <suggestions>, each fully opened and closed.
- Tag names are lowercase ASCII only; never rewrite them as <REPLY> / <REWRITE> or any other variant.
- Add no explanation, reasoning or code fences outside the tags.
- <draft> may contain multi-line Markdown, written directly with newlines and no escaping.
- <ready> contains only true or false. Set true once there is enough to proceed.
- When <ready>true</ready>, <suggestions> may be empty (keep the empty tags <suggestions></suggestions>).`

// CoCreateProgressKind 标识流式回调的内容类型。
const (
	CoCreateProgressThinking = "thinking"
	CoCreateProgressReply    = "reply"
)

// 四段式 XML 标签输出。XML 风格比方括号 marker 更鲁棒——Claude/GPT 训练数据里
// 大量 <thinking>...</thinking> 这类格式，模型几乎不会把 <reply> 改写成 <REWRITE>
// 或其他变体；闭合标签也让流式中段截断更精确（不依赖找下一个 marker 来断尾）。
const (
	tagReply       = "reply"
	tagDraft       = "draft"
	tagReady       = "ready"
	tagSuggestions = "suggestions"
)

func coCreateStream(ctx context.Context, models *bootstrap.ModelSet, sessions *store.SessionStore, sysPrompt string, history []CoCreateMessage, onProgress func(kind, text string)) (reply CoCreateReply, err error) {
	if len(history) == 0 {
		return CoCreateReply{}, fmt.Errorf("cocreate history is empty")
	}

	model := models.ForRole("thinking")

	msgs := []agentcore.Message{agentcore.SystemMsg(sysPrompt)}
	for _, item := range history {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(item.Role)) {
		case "assistant":
			msgs = append(msgs, assistantMsg(content))
		default:
			msgs = append(msgs, agentcore.UserMsg(content))
		}
	}

	var raw, thinking strings.Builder

	// 排查 "cocreate empty response" 等偶发问题需要看到模型实际返回什么。
	// 每轮全程落盘到 <output>/meta/sessions/cocreate.jsonl，与正式创作的 session 日志同位。
	start := time.Now()
	defer func() {
		if sessions == nil {
			return
		}
		if logErr := sessions.LogCoCreate(coCreateLogEntry{
			Time:         time.Now(),
			DurationMS:   time.Since(start).Milliseconds(),
			InputHistory: history,
			RawResponse:  raw.String(),
			RawLen:       len([]rune(raw.String())),
			Thinking:     thinking.String(),
			ParsedReply:  reply.Message,
			ParsedDraft:  reply.Prompt,
			ParsedReady:  reply.Ready,
			ParsedSugs:   reply.Suggestions,
			Error:        errString(err),
		}); logErr != nil {
			slog.Warn("共创会话日志落盘失败", "module", "cocreate", "err", logErr)
		}
	}()

	streamCh, err := model.GenerateStream(ctx, msgs, nil, agentcore.WithMaxTokens(2048))
	if err != nil {
		return CoCreateReply{}, fmt.Errorf("cocreate generate: %w", err)
	}

	var streamed bool
	for ev := range streamCh {
		switch ev.Type {
		case agentcore.StreamEventThinkingDelta:
			thinking.WriteString(ev.Delta)
			if onProgress != nil {
				onProgress(CoCreateProgressThinking, thinking.String())
			}
		case agentcore.StreamEventTextDelta:
			streamed = true
			raw.WriteString(ev.Delta)
			if onProgress != nil {
				onProgress(CoCreateProgressReply, extractReplyPreview(raw.String()))
			}
		case agentcore.StreamEventDone:
			if !streamed {
				raw.WriteString(ev.Message.TextContent())
			}
		case agentcore.StreamEventError:
			if ev.Err != nil {
				return CoCreateReply{}, fmt.Errorf("cocreate generate: %w", ev.Err)
			}
			return CoCreateReply{}, fmt.Errorf("cocreate generate failed")
		}
	}

	// Channel fallback：思考型模型（R1/GLM-Z1/QwQ 等）偶发把完整答案写进
	// reasoning_content 后没切回 final answer 通道，导致 raw 为空但 thinking 含
	// 完整四段。实测见 meta/sessions/cocreate.jsonl —— 直接拿 thinking 当 raw 解析，
	// 协议层已有降级处理（无 [REPLY] 标记时整段当 reply），救场后 UI 体验无差别。
	rawText := raw.String()
	if strings.TrimSpace(rawText) == "" {
		if t := strings.TrimSpace(thinking.String()); t != "" {
			rawText = t
		}
	}
	reply, err = parseCoCreateResponse(rawText)
	return reply, err
}

// coCreateLogEntry 是写入 meta/sessions/cocreate.jsonl 的一行结构。
// 字段命名贴近 jsonl 直查习惯（snake_case），方便 jq 过滤。
type coCreateLogEntry struct {
	Time         time.Time         `json:"time"`
	DurationMS   int64             `json:"duration_ms"`
	InputHistory []CoCreateMessage `json:"input_history"`
	RawResponse  string            `json:"raw_response"`
	RawLen       int               `json:"raw_len"`
	Thinking     string            `json:"thinking,omitempty"`
	ParsedReply  string            `json:"parsed_reply"`
	ParsedDraft  string            `json:"parsed_draft"`
	ParsedReady  bool              `json:"parsed_ready"`
	ParsedSugs   []string          `json:"parsed_sugs,omitempty"`
	Error        string            `json:"error,omitempty"`
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func assistantMsg(text string) agentcore.Message {
	return agentcore.Message{
		Role:      agentcore.RoleAssistant,
		Content:   []agentcore.ContentBlock{agentcore.TextBlock(text)},
		Timestamp: time.Now(),
	}
}

// parseCoCreateResponse 解析 XML 标签输出。模型若没遵守协议（直接说自然语言），
// 整段作为 reply 显示，draft 留空让 session 保留上一轮。
func parseCoCreateResponse(raw string) (CoCreateReply, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return CoCreateReply{}, fmt.Errorf("cocreate empty response")
	}

	reply, draft, ready, suggestions := splitCoCreateMarkers(raw)
	if reply == "" {
		// 模型没遵守 XML 协议：整段作为 reply。
		return CoCreateReply{Message: raw, Prompt: "", Ready: false, Raw: raw}, nil
	}
	return CoCreateReply{
		Message:     reply,
		Prompt:      draft,
		Ready:       ready,
		Suggestions: suggestions,
		Raw:         raw,
	}, nil
}

// splitCoCreateMarkers 按四个 XML 标签切分文本。
// 标签可能缺失（流式中段或模型遗漏），缺失部分对应字段为空 / false / nil。
// 缺失闭标签时，extractTagContent 会取到字符串末尾，仍尽力解析。
func splitCoCreateMarkers(s string) (reply, draft string, ready bool, suggestions []string) {
	reply = extractTagContent(s, tagReply)
	draft = extractTagContent(s, tagDraft)
	readyStr := strings.ToLower(extractTagContent(s, tagReady))
	ready = readyStr == "true" || readyStr == "yes"
	suggestions = parseSuggestions(extractTagContent(s, tagSuggestions))
	return
}

// extractTagContent 从 s 中抠出 <tag>...</tag> 之间的文本。
// 三种偶发故障场景兜底，避免直接走降级丢字段：
//  1. 有开无闭（流式中段）→ 切到下一个已知开标签前
//  2. 无开有闭（模型 typo，如 <suggestions> 写成 <uggestions>）→ 从最近一个已知
//     完整闭合标签的结束位置开始，到 </tag> 之前
//  3. reply 完全无开标签（模型直接以自然语言开篇，末尾贴 </reply>）→ 从开头到 </reply>
func extractTagContent(s, tag string) string {
	open := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	oIdx := strings.Index(s, open)
	if oIdx >= 0 {
		rest := s[oIdx+len(open):]
		if cIdx := strings.Index(rest, closeTag); cIdx >= 0 {
			return strings.TrimSpace(rest[:cIdx])
		}
		// 有开无闭 → 切到下一个已知开标签前
		for _, other := range []string{"<reply>", "<draft>", "<ready>", "<suggestions>"} {
			if other == open {
				continue
			}
			if idx := strings.Index(rest, other); idx >= 0 {
				rest = rest[:idx]
			}
		}
		return strings.TrimSpace(rest)
	}

	// 无开有闭 → 从最近一个已知完整闭合标签的结束位置开始，到 </tag>。
	if cIdx := strings.Index(s, closeTag); cIdx >= 0 {
		prefix := s[:cIdx]
		start := 0
		for _, t := range []string{"</reply>", "</draft>", "</ready>", "</suggestions>"} {
			if t == closeTag {
				continue
			}
			if i := strings.LastIndex(prefix, t); i >= 0 {
				if end := i + len(t); end > start {
					start = end
				}
			}
		}
		return strings.TrimSpace(prefix[start:])
	}
	return ""
}

// parseSuggestions 把 <suggestions> 段每行抠出来，去掉 "- " / "* " / "1. " 等列表前缀。
// 最多保留 3 条；空行、过短（<2 字）、整行像 XML 标签的（typo 开标签兜底残留，
// 例如 <uggestions>）忽略。
func parseSuggestions(text string) []string {
	if text == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 整行像 XML 标签 → 跳过（防 typo 开标签污染）
		if strings.HasPrefix(line, "<") && strings.HasSuffix(line, ">") {
			continue
		}
		// 剥列表前缀
		switch {
		case strings.HasPrefix(line, "- "):
			line = strings.TrimSpace(line[2:])
		case strings.HasPrefix(line, "* "):
			line = strings.TrimSpace(line[2:])
		case isOrderedSuggestion(line):
			line = stripOrderedPrefix(line)
		}
		if len([]rune(line)) < 2 {
			continue
		}
		out = append(out, line)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

// isOrderedSuggestion 判断行首是否形如 "1. " / "12. "（数字+点+空格）。
func isOrderedSuggestion(line string) bool {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	return i > 0 && i+1 < len(line) && line[i] == '.' && line[i+1] == ' '
}

func stripOrderedPrefix(line string) string {
	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i+1 >= len(line) {
		return line
	}
	return strings.TrimSpace(line[i+2:])
}

// extractReplyPreview 流式预览：raw 还在生长时给 UI 一段可显示的文本。
// 找到 <reply> 之后的内容，切到 </reply> 或下一个开标签 <draft> 之前。
// 模型半遵守（漏 <reply> 开标签）时，开头到 </reply> 或 <draft> 都算 reply。
func extractReplyPreview(raw string) string {
	trimmed := strings.TrimSpace(raw)
	open := "<" + tagReply + ">"
	closeTag := "</" + tagReply + ">"
	draftOpen := "<" + tagDraft + ">"

	rest := trimmed
	if rIdx := strings.Index(trimmed, open); rIdx >= 0 {
		rest = trimmed[rIdx+len(open):]
	}
	if cIdx := strings.Index(rest, closeTag); cIdx >= 0 {
		return strings.TrimSpace(rest[:cIdx])
	}
	if dIdx := strings.Index(rest, draftOpen); dIdx >= 0 {
		rest = rest[:dIdx]
	}
	return strings.TrimSpace(rest)
}

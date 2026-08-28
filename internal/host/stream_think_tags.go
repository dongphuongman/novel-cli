package host

import "strings"

// 9router 这类中转把推理放在普通文本通道的 <think> 标签里，而不是独立的
// reasoning 字段。模型本轮没有实际推理时，留下的就是一对空标签；Writer 每章
// 约 5 轮，空标签会在流式面板里堆成噪声。
//
// thinkTagFilter 只丢弃"两个标签之间没有任何非空白字符"这一种情况，带内容的
// <think> 原样透传，且保持边生成边显示。delta 会在任意位置被切开，所以过滤器
// 必须跨调用缓冲：
//   - 文本态下扣住结尾那几个可能是 <think> 前缀的字节；
//   - 进入 <think> 后只在内容仍全为空白时缓冲，一见非空白立刻放行并回到文本态，
//     避免把真实推理憋到 </think> 才整段吐出。
type thinkTagFilter struct {
	state thinkFilterState
	buf   string
}

type thinkFilterState int

const (
	thinkFilterText   thinkFilterState = iota // 普通文本
	thinkFilterOpened                         // 已见 <think>，内容至今全为空白
	thinkFilterSkipWS                         // 刚丢掉一对空标签，吞掉紧随其后的空白
)

const (
	thinkOpenTag  = "<think>"
	thinkCloseTag = "</think>"
	// thinkFilterMaxHold 是兜底闸门：缓冲异常增长时放弃过滤，宁可显示也不吞内容。
	thinkFilterMaxHold = 1024
)

// feed 吃进一段 delta，返回本次可以显示的部分（可能为空，表示还在等后续 delta）。
func (f *thinkTagFilter) feed(delta string) string {
	f.buf += delta
	var out strings.Builder
	for {
		switch f.state {
		case thinkFilterText:
			i := strings.Index(f.buf, thinkOpenTag)
			if i < 0 {
				// 结尾可能是被切开的 <think>，扣住它，其余照发。
				hold := tagPrefixSuffixLen(f.buf, thinkOpenTag)
				out.WriteString(f.buf[:len(f.buf)-hold])
				f.buf = f.buf[len(f.buf)-hold:]
				return out.String()
			}
			out.WriteString(f.buf[:i])
			f.buf = f.buf[i:] // 开标签留在缓冲里，是空是满还不知道
			f.state = thinkFilterOpened

		case thinkFilterOpened:
			inner := f.buf[len(thinkOpenTag):]
			j := strings.Index(inner, thinkCloseTag)
			if j < 0 {
				if blankOrPartialClose(inner) && len(f.buf) <= thinkFilterMaxHold {
					return out.String() // 还看不出空不空，继续等
				}
				// 有实质内容：连开标签一起放行，回到文本态逐段透传剩余推理。
				out.WriteString(f.buf)
				f.buf = ""
				f.state = thinkFilterText
				return out.String()
			}
			end := len(thinkOpenTag) + j + len(thinkCloseTag)
			if strings.TrimSpace(inner[:j]) == "" {
				f.buf = f.buf[end:] // 整对空标签丢掉
				f.state = thinkFilterSkipWS
				break
			}
			out.WriteString(f.buf[:end])
			f.buf = f.buf[end:]
			f.state = thinkFilterText

		case thinkFilterSkipWS:
			// 空标签后面通常还跟着换行，一并吞掉，否则面板上留下空行。
			f.buf = strings.TrimLeft(f.buf, " \t\r\n")
			if f.buf == "" {
				return out.String()
			}
			f.state = thinkFilterText
		}
	}
}

// flush 放掉全部缓冲并复位。轮次结束（streamClear）或真 reasoning 通道插入前
// 调用，保证扣住的尾巴既不会丢，也不会跨轮次错位。
func (f *thinkTagFilter) flush() string {
	out := f.buf
	f.buf = ""
	f.state = thinkFilterText
	return out
}

// blankOrPartialClose 判断 <think> 内容至今是否"仍可能是空的"：
// 全为空白，或空白后跟着一个尚不完整的 </think> 前缀（闭标签被切开了）。
func blankOrPartialClose(inner string) bool {
	rest := strings.TrimLeft(inner, " \t\r\n")
	if rest == "" {
		return true
	}
	return len(rest) < len(thinkCloseTag) && strings.HasPrefix(thinkCloseTag, rest)
}

// tagPrefixSuffixLen 返回 s 结尾处能匹配 tag 真前缀的最大长度（无匹配为 0）。
func tagPrefixSuffixLen(s, tag string) int {
	n := len(tag) - 1
	if n > len(s) {
		n = len(s)
	}
	for k := n; k > 0; k-- {
		if strings.HasSuffix(s, tag[:k]) {
			return k
		}
	}
	return 0
}

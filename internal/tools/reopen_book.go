package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/voocel/agentcore/schema"
	"github.com/dongphuongman/novel-cli/internal/domain"
	"github.com/dongphuongman/novel-cli/internal/errs"
	"github.com/dongphuongman/novel-cli/internal/store"
)

// ReopenBookTool 把已完结的书重新打开进入返工态，由 Engine 在干预动作边界调用。
// 完本后 completePhaseGate 硬拦一切 subagent 派发，用户无法返工已写章节。
// 本工具不是 subagent，complete 期可调：它原子地把 phase 切回 writing、目标章入
// PendingRewrites、flow=rewriting，随后 Flow Router 照既有返工队列派 writer 逐章重写，
// 队列跑完 commit_chapter 自动重新收尾完结。Gate / Router / edit / commit 重逻辑均无需改动。
type ReopenBookTool struct {
	store *store.Store
}

func NewReopenBookTool(s *store.Store) *ReopenBookTool {
	return &ReopenBookTool{store: s}
}

func (t *ReopenBookTool) Name() string  { return "reopen_book" }
func (t *ReopenBookTool) Label() string { return "Reopen for rework" }

func (t *ReopenBookTool) Description() string {
	return "Reopen a finished book (phase=complete) into rework state, for when the user asks to rewrite or polish a few chapters after completion. " +
		"chapters lists the completed chapter numbers to rework; they enter the rewrite queue, the host dispatches writer chapter by chapter, and the book re-completes automatically once all are done. " +
		"Use it only when the book is complete and the user explicitly asks to change chapters already written. A request for new plot or more length is not rework; do not use this tool for that."
}

// 写工具，禁止并发。
func (t *ReopenBookTool) ReadOnly(_ json.RawMessage) bool        { return false }
func (t *ReopenBookTool) ConcurrencySafe(_ json.RawMessage) bool { return false }

func (t *ReopenBookTool) ActivityDescription(_ json.RawMessage) string {
	return "Reopen for rework"
}

func (t *ReopenBookTool) Schema() map[string]any {
	return schema.Object(
		schema.Property("chapters", schema.Array("List of completed chapter numbers to rework (at least one)", schema.Int(""))).Required(),
		schema.Property("reason", schema.String("Reason for the rework (optional, e.g. \"strip special characters\")")),
	)
}

func (t *ReopenBookTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Chapters []int  `json:"chapters"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w: %w", errs.ErrToolArgs, err)
	}
	if len(a.Chapters) == 0 {
		return nil, fmt.Errorf("chapters cannot be empty; name the chapters to rework: %w", errs.ErrToolArgs)
	}

	progress, err := t.store.Progress.Load()
	if err != nil {
		return nil, fmt.Errorf("load progress: %w: %w", errs.ErrStoreRead, err)
	}
	if progress == nil {
		return nil, fmt.Errorf("progress is not initialized: %w", errs.ErrToolPrecondition)
	}
	// 只能返工已写章；不在已完成集合的章号属续写/越界，明确拒绝引导用户走篇幅调整。
	var invalid []int
	for _, ch := range a.Chapters {
		if !slices.Contains(progress.CompletedChapters, ch) {
			invalid = append(invalid, ch)
		}
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("chapter %v is not written yet; reopen only reworks completed chapters (use a length adjustment to add or extend plot): %w", invalid, errs.ErrToolPrecondition)
	}

	// phase 前置校验在 store.Reopen 内兜底（仅 complete 可调）。
	if err := t.store.Progress.Reopen(a.Chapters, a.Reason); err != nil {
		return nil, fmt.Errorf("reopen: %w: %w", errs.ErrStoreWrite, err)
	}

	// checkpoint：与 complete_book 对称（GlobalScope + meta/progress.json）。
	if _, err := t.store.Checkpoints.AppendArtifact(domain.GlobalScope(), "reopen", "meta/progress.json"); err != nil {
		return nil, fmt.Errorf("checkpoint reopen: %w: %w", errs.ErrStoreWrite, err)
	}

	return json.Marshal(map[string]any{
		"reopened":         true,
		"phase":            string(domain.PhaseWriting),
		"pending_rewrites": a.Chapters,
		"next_step":        "reopened and the target chapters are queued. Wait for the host to dispatch writer chapter by chapter; the book re-completes automatically once all are done.",
	})
}

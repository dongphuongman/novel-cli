package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/store"

	"github.com/voocel/ainovel-cli/internal/lang"
)

// DraftChapterTool 写入整章草稿，替代旧的 write_scene + polish_chapter 流水线。
// Agent 自主决定一次写完还是分批续写。
type DraftChapterTool struct {
	store *store.Store
	// lp 决定回给 Writer 的 word_count 口径。它必须与 commit_chapter 和用户
	// 字数规则同源——回报字符数而规则说的是"词"，Writer 会照着错的信号收笔。
	lp lang.Profile
}

func NewDraftChapterTool(store *store.Store, lp lang.Profile) *DraftChapterTool {
	return &DraftChapterTool{store: store, lp: lp}
}

func (t *DraftChapterTool) Name() string { return "draft_chapter" }
func (t *DraftChapterTool) Description() string {
	return "Write the chapter's prose. mode=write overwrites the whole chapter; mode=append appends to the existing draft (continuing or revising)"
}
func (t *DraftChapterTool) Label() string { return "Write chapter" }

// 写工具，禁止并发（读-改-写竞态）。
func (t *DraftChapterTool) ReadOnly(_ json.RawMessage) bool        { return false }
func (t *DraftChapterTool) ConcurrencySafe(_ json.RawMessage) bool { return false }

func (t *DraftChapterTool) Schema() map[string]any {
	// mode 标 required 是为了兼容 OpenAI strict tool calling——strict 模式
	// 要求所有 properties 都在 required 列表中。原来的"省略 mode 走 write
	// 默认"行为现在需要模型显式传 mode="write"，Execute 的 default 分支不变。
	return schema.Object(
		schema.Property("chapter", schema.Int("Chapter number")).Required(),
		schema.Property("content", schema.String("The chapter's prose")).Required(),
		schema.Property("mode", schema.Enum("Write mode", "write", "append")).Required(),
	)
}

// StrictSchema 要求 Provider 保证工具参数符合 schema。
func (t *DraftChapterTool) StrictSchema() bool { return true }

func (t *DraftChapterTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Chapter int    `json:"chapter"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, fmt.Errorf("invalid args: %w: %w", errs.ErrToolArgs, err)
	}
	if a.Chapter <= 0 {
		return nil, fmt.Errorf("chapter must be > 0: %w", errs.ErrToolArgs)
	}
	if a.Content == "" {
		return nil, fmt.Errorf("content must not be empty: %w", errs.ErrToolArgs)
	}
	if err := t.store.Progress.ValidateChapterWork(a.Chapter); err != nil {
		return nil, err
	}
	if err := EnsureChapterExpanded(t.store, a.Chapter); err != nil {
		return nil, err
	}
	completed, err := t.store.Progress.IsChapterCompleted(a.Chapter)
	if err != nil {
		return nil, fmt.Errorf("load progress: %w: %w", errs.ErrStoreRead, err)
	}
	if completed {
		// 打磨/重写路径：章节虽已完成，但仍在 pending_rewrites 中，允许覆盖草稿
		progress, err := t.store.Progress.Load()
		if err != nil {
			return nil, fmt.Errorf("load progress: %w: %w", errs.ErrStoreRead, err)
		}
		inRewriteQueue := progress != nil && slices.Contains(progress.PendingRewrites, a.Chapter)
		if !inRewriteQueue {
			return json.Marshal(map[string]any{
				"chapter":   a.Chapter,
				"skipped":   true,
				"completed": true,
				"reason":    fmt.Sprintf("chapter %d is already committed and cannot be overwritten", a.Chapter),
			})
		}
	}
	if err := t.store.Progress.StartChapter(a.Chapter); err != nil {
		return nil, fmt.Errorf("mark chapter in progress: %w", err)
	}

	switch a.Mode {
	case "append":
		if err := t.store.Drafts.AppendDraft(a.Chapter, a.Content); err != nil {
			return nil, fmt.Errorf("append draft: %w", err)
		}
		full, err := t.store.Drafts.LoadDraft(a.Chapter)
		if err != nil {
			return nil, fmt.Errorf("load draft after append: %w", err)
		}
		if _, err := t.store.Checkpoints.AppendArtifact(
			domain.ChapterScope(a.Chapter), "draft",
			fmt.Sprintf("drafts/%02d.draft.md", a.Chapter),
		); err != nil {
			return nil, fmt.Errorf("checkpoint draft: %w", err)
		}
		return json.Marshal(map[string]any{
			"written":    true,
			"chapter":    a.Chapter,
			"mode":       "append",
			"word_count": t.lp.CountWords(full),
			"next_step":  "read the draft back with read_chapter(source=draft), then call check_consistency, then commit_chapter",
		})
	default: // write
		if err := t.store.Drafts.SaveDraft(a.Chapter, a.Content); err != nil {
			return nil, fmt.Errorf("save draft: %w", err)
		}
		if _, err := t.store.Checkpoints.AppendArtifact(
			domain.ChapterScope(a.Chapter), "draft",
			fmt.Sprintf("drafts/%02d.draft.md", a.Chapter),
		); err != nil {
			return nil, fmt.Errorf("checkpoint draft: %w", err)
		}
		return json.Marshal(map[string]any{
			"written":    true,
			"chapter":    a.Chapter,
			"mode":       "write",
			"word_count": t.lp.CountWords(a.Content),
			"next_step":  "read the draft back with read_chapter(source=draft), then call check_consistency, then commit_chapter",
		})
	}
}

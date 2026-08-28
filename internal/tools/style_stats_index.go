package tools

import (
	"fmt"
	"slices"
	"sync"

	"github.com/voocel/ainovel-cli/internal/store"
	"github.com/voocel/ainovel-cli/internal/stylestat"

	"github.com/voocel/ainovel-cli/internal/lang"
)

// StyleStatsIndex 把 Store 中的已完成章节同步到增量统计器。
// 首次 Snapshot 全量恢复一次；之后只加载新增章节，重写由 commit_chapter 主动刷新。
type StyleStatsIndex struct {
	store *store.Store
	lp    lang.Profile

	mu        sync.Mutex
	tracker   *stylestat.Tracker
	completed map[int]struct{}
}

// lp 决定统计口径（模式表/断句/n-gram 单位）。它是本书级事实，
// 构造后不变——中途换语言等于换一本书的统计基线。
func NewStyleStatsIndex(store *store.Store, lp lang.Profile) *StyleStatsIndex {
	return &StyleStatsIndex{store: store, lp: lp}
}

// Lang 暴露本索引的语言档案，供共享同一索引的工具复用同一口径。
func (s *StyleStatsIndex) Lang() lang.Profile { return s.lp }

func (s *StyleStatsIndex) Snapshot(
	completedChapters []int,
	titles, stopwords []string,
) (*stylestat.Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	completed, wanted, err := normalizeCompletedChapters(completedChapters)
	if err != nil {
		return nil, err
	}

	if s.tracker == nil {
		tracker := stylestat.NewTracker(s.lp)
		for _, chapter := range completed {
			text, err := s.loadChapter(chapter)
			if err != nil {
				return nil, err
			}
			tracker.Upsert(chapter, text)
		}
		s.tracker = tracker
		s.completed = wanted
		return tracker.Snapshot(titles, stopwords), nil
	}

	type chapterText struct {
		chapter int
		text    string
	}
	var additions []chapterText
	for _, chapter := range completed {
		if _, ok := s.completed[chapter]; ok {
			continue
		}
		text, err := s.loadChapter(chapter)
		if err != nil {
			return nil, err
		}
		additions = append(additions, chapterText{chapter: chapter, text: text})
	}

	for chapter := range s.completed {
		if _, ok := wanted[chapter]; ok {
			continue
		}
		s.tracker.Remove(chapter)
	}
	for _, addition := range additions {
		s.tracker.Upsert(addition.chapter, addition.text)
	}
	s.completed = wanted
	return s.tracker.Snapshot(titles, stopwords), nil
}

// ChapterCommitted 在提交 Saga 完整成功后刷新一章。索引尚未初始化时，
// 下一次 Snapshot 会从 Progress 事实一次性恢复。
func (s *StyleStatsIndex) ChapterCommitted(chapter int, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tracker == nil {
		return
	}
	s.tracker.Upsert(chapter, text)
	s.completed[chapter] = struct{}{}
}

func (s *StyleStatsIndex) loadChapter(chapter int) (string, error) {
	text, err := s.store.Drafts.LoadChapterText(chapter)
	if err != nil {
		return "", fmt.Errorf("read the final text of chapter %d: %w", chapter, err)
	}
	if text == "" {
		return "", fmt.Errorf("chapter %d is marked complete but has no final text", chapter)
	}
	return text, nil
}

func normalizeCompletedChapters(chapters []int) ([]int, map[int]struct{}, error) {
	normalized := slices.Clone(chapters)
	slices.Sort(normalized)
	set := make(map[int]struct{}, len(normalized))
	for _, chapter := range normalized {
		if chapter <= 0 {
			return nil, nil, fmt.Errorf("a completed chapter number must be greater than 0, got %d", chapter)
		}
		if _, exists := set[chapter]; exists {
			return nil, nil, fmt.Errorf("duplicate completed chapter: %d", chapter)
		}
		set[chapter] = struct{}{}
	}
	return normalized, set, nil
}

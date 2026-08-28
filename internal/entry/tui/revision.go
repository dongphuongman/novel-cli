package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/voocel/ainovel-cli/internal/host"
	"github.com/voocel/ainovel-cli/internal/revision"

	"github.com/voocel/ainovel-cli/internal/i18n"
)

type revisionDoneMsg struct {
	checkOnly bool
	chapters  []int
	result    *revision.Result
	err       error
}

func startRevisionSync(rt *host.Host, args []string) (tea.Cmd, bool, error) {
	checkOnly := false
	for _, arg := range args {
		switch arg {
		case "--check":
			checkOnly = true
		default:
			return nil, false, fmt.Errorf(i18n.T("rev.err.arg"), arg)
		}
	}
	return func() tea.Msg {
		if checkOnly {
			chapters, err := rt.CheckChapterRevisions()
			return revisionDoneMsg{checkOnly: true, chapters: chapters, err: err}
		}
		result, err := rt.SyncChapterRevisions(context.Background())
		return revisionDoneMsg{result: result, err: err}
	}, checkOnly, nil
}

func formatRevisionResult(result *revision.Result) string {
	if result == nil || len(result.Applied) == 0 {
		return i18n.T("ui.sync.none")
	}
	parts := make([]string, 0, len(result.Analyses))
	for i, analysis := range result.Analyses {
		if i >= len(result.Applied) {
			break
		}
		part := fmt.Sprintf(i18n.T("rev.chapter"), result.Applied[i], analysis.ChangeSummary)
		if analysis.StoryChanged {
			part += i18n.T("rev.facts")
		}
		if len(analysis.DownstreamIssues) > 0 {
			part += fmt.Sprintf(i18n.T("rev.conflicts"), len(analysis.DownstreamIssues))
		}
		parts = append(parts, part)
	}
	summary := fmt.Sprintf(i18n.T("rev.accepted"), result.Applied)
	if len(parts) > 0 {
		summary += i18n.T("sep.semi") + strings.Join(parts, i18n.T("sep.semi"))
	}
	return summary
}

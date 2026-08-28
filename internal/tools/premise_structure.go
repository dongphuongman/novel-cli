package tools

import (
	"strings"

	"github.com/voocel/ainovel-cli/internal/domain"
)

// 故事前提的二级标题是**机械解析键**：premiseStructure 按此判断哪一节缺失，
// architect 据此补齐，audit_foundation 据此放行。写错一个字 = 该节对系统不存在。
//
// 规范名为英文（与 assets/prompts 同语言）；中文名保留为别名，存量中文作品
// 逐字不变地继续解析。新增语言只需在 premiseHeadingAliases 里加一行。
const (
	headingGenreTone      = "Genre and Tone"
	headingGenrePosition  = "Genre Positioning"
	headingCoreConflict   = "Central Conflict"
	headingProtagGoal     = "Protagonist Goal"
	headingEndingDir      = "Ending Direction"
	headingNoGoZones      = "Writing No-Go Zones"
	headingSellingPoints  = "Differentiating Selling Points"
	headingDiffHook       = "Differentiating Hook"
	headingCorePromise    = "Core Promise"
	headingStoryEngine    = "Story Engine"
	headingThroughline    = "Relationship and Growth Throughline"
	headingEscalationPath = "Escalation Path"
	headingMidPivot       = "Mid-Game Pivot"
	headingEndgameQ       = "Endgame Question"
	headingShortFormFit   = "Short-Form Fit"
)

// premiseHeadingAliases 把任意书写形式映射到规范名。键在查表前统一小写，
// 因此英文大小写差异（"Genre and tone"）不会让整节静默丢失。
var premiseHeadingAliases = map[string]string{
	// 规范名自身
	"genre and tone":                      headingGenreTone,
	"genre positioning":                   headingGenrePosition,
	"central conflict":                    headingCoreConflict,
	"protagonist goal":                    headingProtagGoal,
	"ending direction":                    headingEndingDir,
	"writing no-go zones":                 headingNoGoZones,
	"differentiating selling points":      headingSellingPoints,
	"differentiating hook":                headingDiffHook,
	"core promise":                        headingCorePromise,
	"story engine":                        headingStoryEngine,
	"relationship and growth throughline": headingThroughline,
	"escalation path":                     headingEscalationPath,
	"mid-game pivot":                      headingMidPivot,
	"endgame question":                    headingEndgameQ,
	"short-form fit":                      headingShortFormFit,

	// 英文常见变体
	"protagonist's goal":                headingProtagGoal,
	"writing no go zones":               headingNoGoZones,
	"mid game pivot":                    headingMidPivot,
	"short form fit":                    headingShortFormFit,
	"relationship / growth throughline": headingThroughline,

	// 中文别名：存量作品与中文创作语言下的产出
	"题材定位":    headingGenrePosition,
	"题材和基调":   headingGenreTone,
	"核心冲突":    headingCoreConflict,
	"主角目标":    headingProtagGoal,
	"结局方向":    headingEndingDir,
	"终局方向":    headingEndingDir,
	"写作禁区":    headingNoGoZones,
	"差异化卖点":   headingSellingPoints,
	"差异化钩子":   headingDiffHook,
	"核心兑现承诺":  headingCorePromise,
	"故事引擎":    headingStoryEngine,
	"关系/成长主线": headingThroughline,
	"升级路径":    headingEscalationPath,
	"中段转折":    headingMidPivot,
	"中期转向":    headingMidPivot,
	"终局命题":    headingEndgameQ,
	"短篇适配性":   headingShortFormFit,
	"本作为什么适合短篇/单卷收束": headingShortFormFit,
}

func parsePremiseSections(premise string) map[string]string {
	lines := strings.Split(premise, "\n")
	sections := make(map[string]string)
	var current string
	var body []string

	flush := func() {
		if current == "" {
			return
		}
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if text != "" {
			sections[current] = text
		}
		body = body[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if heading, ok := canonicalPremiseHeading(trimmed); ok {
			flush()
			current = heading
			continue
		}
		if current != "" {
			body = append(body, line)
		}
	}
	flush()
	return sections
}

func canonicalPremiseHeading(line string) (string, bool) {
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	title := strings.TrimSpace(strings.TrimLeft(line, "#"))
	if title == "" {
		return "", false
	}
	canonical, ok := premiseHeadingAliases[strings.ToLower(title)]
	return canonical, ok
}

func premiseStructure(premise string, tier domain.PlanningTier) map[string]any {
	sections := parsePremiseSections(premise)
	required := requiredPremiseHeadings(tier)
	found := make([]string, 0, len(required))
	var missing []string
	for _, heading := range required {
		if _, ok := sections[heading]; ok {
			found = append(found, heading)
			continue
		}
		missing = append(missing, heading)
	}

	structure := map[string]any{
		"template_ready": len(missing) == 0,
		"found":          found,
		"missing":        missing,
	}
	if len(sections) > 0 {
		structure["section_count"] = len(sections)
	}
	return structure
}

func requiredPremiseHeadings(tier domain.PlanningTier) []string {
	common := []string{
		headingGenreTone,
		headingGenrePosition,
		headingCoreConflict,
		headingProtagGoal,
		headingEndingDir,
		headingNoGoZones,
		headingSellingPoints,
		headingDiffHook,
		headingCorePromise,
	}

	switch tier {
	case domain.PlanningTierLong:
		return append(common,
			headingStoryEngine,
			headingThroughline,
			headingEscalationPath,
			headingMidPivot,
			headingEndgameQ,
		)
	case domain.PlanningTierMid:
		return append(common,
			headingStoryEngine,
			headingMidPivot,
		)
	case domain.PlanningTierShort:
		return append(common,
			headingShortFormFit,
		)
	default:
		return common
	}
}

package tools

import (
	"testing"

	"github.com/dongphuongman/novel-cli/internal/domain"
)

func TestParsePremiseSections(t *testing.T) {
	premise := `# Premise

## 题材和基调
东方玄幻，冷硬成长。

## 题材定位
东方玄幻升级流，面向追求爽点和关系推进的读者。

## 核心冲突
主角必须在宗门规则与个人良知之间做选择。

## 中期转向
旧有修炼路线失效，必须转向禁术体系。
`

	sections := parsePremiseSections(premise)
	// 中文标题必须归一到英文规范名：存量作品的解析行为逐字不变。
	if sections[headingGenreTone] == "" {
		t.Fatalf("expected %s section, got %+v", headingGenreTone, sections)
	}
	if sections[headingGenrePosition] == "" {
		t.Fatalf("expected %s section, got %+v", headingGenrePosition, sections)
	}
	if sections[headingCoreConflict] == "" {
		t.Fatalf("expected %s section, got %+v", headingCoreConflict, sections)
	}
	if sections[headingMidPivot] == "" {
		t.Fatalf("expected 中期转向 alias normalized to %s, got %+v", headingMidPivot, sections)
	}
}

func TestPremiseStructure(t *testing.T) {
	premise := `## 题材和基调
升级流，偏冷硬。

## 题材定位
升级流

## 核心冲突
冲突

## 主角目标
目标

## 终局方向
终局

## 写作禁区
禁区

## 差异化卖点
卖点

## 差异化钩子
钩子

## 核心兑现承诺
兑现

## 故事引擎
引擎

## 中段转折
转折
`

	structure := premiseStructure(premise, domain.PlanningTierMid)
	if ready, _ := structure["template_ready"].(bool); !ready {
		t.Fatalf("expected template_ready, got %+v", structure)
	}
	missing, _ := structure["missing"].([]string)
	if len(missing) != 0 {
		t.Fatalf("expected no missing headings, got %+v", missing)
	}
}

func TestPremiseStructureShortAcceptsLegacyHeadingAlias(t *testing.T) {
	premise := `## 题材和基调
单卷高压营救。

## 题材定位
短篇高密度冒险。

## 核心冲突
主角必须在一夜内救出人质。

## 主角目标
救出人质并活着离开。

## 结局方向
完成任务但付出代价。

## 写作禁区
不扩展成长期连载。

## 差异化卖点
时限压力与连续反转。

## 差异化钩子
每次选择都缩短救援时间。

## 核心兑现承诺
紧迫感、抉择与反转。

## 本作为什么适合短篇/单卷收束
核心矛盾和人物弧线都能在单次任务中完成。
`

	structure := premiseStructure(premise, domain.PlanningTierShort)
	if ready, _ := structure["template_ready"].(bool); !ready {
		t.Fatalf("expected short template_ready, got %+v", structure)
	}
}

// TestParsePremiseSectionsEnglish 是本轮英文化的验收：英文标题必须与中文别名
// 走同一张表，且大小写差异不得让整节静默丢失（丢失不报错，只是该节"不存在"）。
func TestParsePremiseSectionsEnglish(t *testing.T) {
	premise := `# Story Premise

## Genre and Tone
Hard-edged eastern fantasy.

## genre positioning
A progression fantasy for readers who want payoff and relationship movement.

## Central Conflict
The protagonist must choose between the sect's rules and their own conscience.

## Mid-Game Pivot
The old cultivation path stops working; the story turns to forbidden arts.
`

	sections := parsePremiseSections(premise)
	for _, want := range []string{headingGenreTone, headingGenrePosition, headingCoreConflict, headingMidPivot} {
		if sections[want] == "" {
			t.Fatalf("expected %s section, got %+v", want, sections)
		}
	}
}

// TestPremiseStructureEnglishShort 覆盖英文提示词的实际产出形状（短篇十节齐备）。
func TestPremiseStructureEnglishShort(t *testing.T) {
	premise := `## Genre and Tone
A single-volume high-pressure rescue.

## Genre Positioning
Dense short-form adventure.

## Central Conflict
The protagonist has one night to get the hostage out.

## Protagonist Goal
Free the hostage and get out alive.

## Ending Direction
The job succeeds, at a price.

## Writing No-Go Zones
Do not expand into a long serial.

## Differentiating Selling Points
Time pressure and continuous reversals.

## Differentiating Hook
Every choice shortens the time left.

## Core Promise
Urgency, hard choices, reversal.

## Short-Form Fit
The central conflict and the character arc both close inside one mission.
`

	structure := premiseStructure(premise, domain.PlanningTierShort)
	if ready, _ := structure["template_ready"].(bool); !ready {
		t.Fatalf("expected short template_ready, got %+v", structure)
	}
}

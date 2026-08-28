package chapterfacts

import (
	"fmt"
	"strings"

	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
)

// Properties 返回完整章节事实共用的 JSON Schema 字段。
func Properties(includeFeedback bool) []schema.Prop {
	textList := func(description string) map[string]any {
		return schema.Array(description, schema.String(description))
	}
	timeline := schema.Object(
		schema.Property("time", schema.String("In-story time")).Required(),
		schema.Property("event", schema.String("Event")).Required(),
		schema.Property("characters", textList("Characters involved")).Required(),
	)
	foreshadow := schema.Object(
		schema.Property("id", schema.String("Foreshadowing ID")).Required(),
		schema.Property("action", schema.Enum("Action", "plant", "advance", "resolve")).Required(),
		schema.Property("description", llmcontract.Nullable(schema.String("Description when planting; null for other actions"))).Required(),
	)
	relationship := schema.Object(
		schema.Property("character_a", schema.String("Character A")).Required(),
		schema.Property("character_b", schema.String("Character B")).Required(),
		schema.Property("relation", schema.String("The relationship as it stands at the end of this chapter")).Required(),
	)
	stateChange := schema.Object(
		schema.Property("entity", schema.String("Entity")).Required(),
		schema.Property("field", schema.String("Attribute")).Required(),
		schema.Property("old_value", llmcontract.Nullable(schema.String("Value before"))).Required(),
		schema.Property("new_value", schema.String("Value after")).Required(),
		schema.Property("reason", llmcontract.Nullable(schema.String("Reason"))).Required(),
	)
	props := []schema.Prop{
		schema.Property("title", schema.String("Final title")).Required(),
		schema.Property("summary", schema.String("Chapter summary")).Required(),
		schema.Property("characters", textList("Characters appearing")).Required(),
		schema.Property("key_events", textList("Key events")).Required(),
		schema.Property("timeline_events", schema.Array("Timeline events", timeline)).Required(),
		schema.Property("foreshadow_updates", schema.Array("Foreshadowing actions", foreshadow)).Required(),
		schema.Property("relationship_changes", schema.Array("Relationship changes", relationship)).Required(),
		schema.Property("state_changes", schema.Array("State changes", stateChange)).Required(),
		schema.Property("cast_intros", schema.Array("Newly introduced supporting characters", schema.Object(
			schema.Property("name", schema.String("Name")).Required(),
			schema.Property("brief_role", schema.String("Role in brief")).Required(),
		))).Required(),
		schema.Property("hook_type", llmcontract.Nullable(schema.Enum("Chapter-ending hook", domain.HookTypes()...))).Required(),
		schema.Property("dominant_strand", llmcontract.Nullable(schema.Enum("Dominant narrative strand", domain.DominantStrands()...))).Required(),
	}
	if includeFeedback {
		feedback := schema.Object(
			schema.Property("deviation", schema.String("How this chapter deviated from the outline")).Required(),
			schema.Property("suggestion", schema.String("Suggested adjustments to the downstream outline")).Required(),
		)
		feedback["description"] = "The suggestion object for the downstream outline; pass a JSON object directly, never stringified JSON"
		props = append(props, schema.Property("feedback", llmcontract.Nullable(feedback)).Required())
	}
	return props
}

// Validate 校验普通提交与人工修订共用的确定性约束。
func Validate(facts domain.ChapterFacts) error {
	if strings.TrimSpace(facts.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(facts.Summary) == "" {
		return fmt.Errorf("summary is required")
	}
	if len(facts.KeyEvents) == 0 {
		return fmt.Errorf("key_events must contain at least one event")
	}
	if err := validateTextItems("characters", facts.Characters); err != nil {
		return err
	}
	if err := validateTextItems("key_events", facts.KeyEvents); err != nil {
		return err
	}
	for i, event := range facts.TimelineEvents {
		if strings.TrimSpace(event.Time) == "" || strings.TrimSpace(event.Event) == "" {
			return fmt.Errorf("timeline_events[%d] requires time and event", i)
		}
		if err := validateTextItems(fmt.Sprintf("timeline_events[%d].characters", i), event.Characters); err != nil {
			return err
		}
	}
	for i, update := range facts.ForeshadowUpdates {
		if strings.TrimSpace(update.ID) == "" {
			return fmt.Errorf("foreshadow_updates[%d].id is required", i)
		}
		switch update.Action {
		case "plant":
			if strings.TrimSpace(update.Description) == "" {
				return fmt.Errorf("foreshadow_updates[%d] plant requires description", i)
			}
		case "advance", "resolve":
		default:
			return fmt.Errorf("foreshadow_updates[%d].action invalid: %q", i, update.Action)
		}
	}
	for i, change := range facts.RelationshipChanges {
		if strings.TrimSpace(change.CharacterA) == "" || strings.TrimSpace(change.CharacterB) == "" || strings.TrimSpace(change.Relation) == "" {
			return fmt.Errorf("relationship_changes[%d] requires character_a, character_b and relation", i)
		}
		if change.CharacterA == change.CharacterB {
			return fmt.Errorf("relationship_changes[%d] cannot relate a character to itself", i)
		}
	}
	for i, change := range facts.StateChanges {
		if strings.TrimSpace(change.Entity) == "" || strings.TrimSpace(change.Field) == "" || strings.TrimSpace(change.NewValue) == "" {
			return fmt.Errorf("state_changes[%d] requires entity, field and new_value", i)
		}
	}
	for i, intro := range facts.CastIntros {
		if strings.TrimSpace(intro.Name) == "" || strings.TrimSpace(intro.BriefRole) == "" {
			return fmt.Errorf("cast_intros[%d] requires name and brief_role", i)
		}
	}
	if facts.HookType != "" && !domain.ValidHookType(facts.HookType) {
		return fmt.Errorf("invalid hook_type %q", facts.HookType)
	}
	if facts.DominantStrand != "" && !domain.ValidDominantStrand(facts.DominantStrand) {
		return fmt.Errorf("invalid dominant_strand %q", facts.DominantStrand)
	}
	if facts.Feedback != nil && (strings.TrimSpace(facts.Feedback.Deviation) == "" || strings.TrimSpace(facts.Feedback.Suggestion) == "") {
		return fmt.Errorf("feedback requires deviation and suggestion")
	}
	return nil
}

func validateTextItems(name string, items []string) error {
	for i, item := range items {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("%s[%d] cannot be empty", name, i)
		}
	}
	return nil
}

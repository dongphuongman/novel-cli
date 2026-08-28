package imp

import (
	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
)

func nullableString(description string) map[string]any {
	return llmcontract.Nullable(schema.String(description))
}

func stringList(description string) map[string]any {
	return schema.Array(description, schema.String(description))
}

var segmentContract = llmcontract.Contract{
	Name:        "import_segment",
	Description: "Identify chapter, volume/part and ancillary-text boundaries in the imported text",
	Schema: schema.Object(
		schema.Property("boundaries", schema.Array("Boundaries in source order", schema.Object(
			schema.Property("unit_id", schema.String("A unit id inside the owned span")).Required(),
			schema.Property("anchor", nullableString("A source fragment locating the boundary when one unit holds several; otherwise null")).Required(),
			schema.Property("kind", schema.Enum("Boundary kind", kindChapter, kindGroup, kindFrontMatter, kindBackMatter)).Required(),
			schema.Property("title", nullableString("The title verbatim; null when there is none")).Required(),
			schema.Property("uncertain", schema.Bool("Whether the user needs to confirm it")).Required(),
			schema.Property("reason", nullableString("Why it is uncertain; null when no explanation is needed")).Required(),
		))).Required(),
	),
}

var analysisContract = llmcontract.Contract{
	Name:        "import_chapter_analysis",
	Description: "Extract traceable story facts from consecutive chapters",
	Schema: schema.Object(
		schema.Property("chapters", schema.Array("Per-chapter facts, in the same order as the input chapter numbers", chapterFactsSchema())).Required(),
	),
}

func chapterFactsSchema() map[string]any {
	characterEvidence := schema.Object(
		schema.Property("chapter", schema.Int("The chapter the evidence is in")).Required(),
		schema.Property("name", schema.String("Character name")).Required(),
		schema.Property("note", nullableString("A fact about the character; null when there is none")).Required(),
	)
	worldEvidence := schema.Object(
		schema.Property("chapter", schema.Int("The chapter the evidence is in")).Required(),
		schema.Property("category", nullableString("Category of the world fact; null when it cannot be categorized")).Required(),
		schema.Property("fact", schema.String("A world fact the prose states explicitly")).Required(),
	)
	timelineEvent := schema.Object(
		schema.Property("chapter", schema.Int("Chapter number")).Required(),
		schema.Property("time", schema.String("In-story time")).Required(),
		schema.Property("event", schema.String("Event")).Required(),
		schema.Property("characters", stringList("Characters involved")).Required(),
	)
	foreshadow := schema.Object(
		schema.Property("id", schema.String("Reuse the foreshadowing ID from the ledger")).Required(),
		schema.Property("action", schema.Enum("Foreshadowing action", "plant", "advance", "resolve")).Required(),
		schema.Property("description", nullableString("Description of the foreshadowing when planting; may be null otherwise")).Required(),
	)
	relationship := schema.Object(
		schema.Property("character_a", schema.String("Character A")).Required(),
		schema.Property("character_b", schema.String("Character B")).Required(),
		schema.Property("relation", schema.String("Relationship change")).Required(),
		schema.Property("chapter", schema.Int("Chapter number")).Required(),
	)
	stateChange := schema.Object(
		schema.Property("chapter", schema.Int("Chapter number")).Required(),
		schema.Property("entity", schema.String("The character or entity")).Required(),
		schema.Property("field", schema.String("The attribute that changed")).Required(),
		schema.Property("old_value", nullableString("State before the change; null on first appearance")).Required(),
		schema.Property("new_value", schema.String("State after the change")).Required(),
		schema.Property("reason", nullableString("Why it changed; null when the prose does not say")).Required(),
	)
	return schema.Object(
		schema.Property("chapter", schema.Int("Chapter number")).Required(),
		schema.Property("title", schema.String("Chapter title")).Required(),
		schema.Property("summary", schema.String("Summary of this chapter")).Required(),
		schema.Property("key_events", stringList("Key events")).Required(),
		schema.Property("core_event", schema.String("The single most important thing in this chapter")).Required(),
		schema.Property("hook", nullableString("The chapter-ending hook; null when there is none")).Required(),
		schema.Property("scenes", stringList("Sequence of scenes")).Required(),
		schema.Property("characters", stringList("Characters appearing")).Required(),
		schema.Property("character_evidence", schema.Array("Character evidence", characterEvidence)).Required(),
		schema.Property("world_evidence", schema.Array("World-fact evidence", worldEvidence)).Required(),
		schema.Property("timeline_events", schema.Array("Timeline events", timelineEvent)).Required(),
		schema.Property("foreshadow_updates", schema.Array("Foreshadowing updates", foreshadow)).Required(),
		schema.Property("relationship_changes", schema.Array("Relationship change", relationship)).Required(),
		schema.Property("state_changes", schema.Array("State changes", stateChange)).Required(),
		schema.Property("hook_type", schema.Enum("Chapter-ending hook type", domain.HookTypes()...)).Required(),
		schema.Property("dominant_strand", schema.Enum("Dominant narrative strand", domain.DominantStrands()...)).Required(),
	)
}

var rangeContract = llmcontract.Contract{
	Name:        "import_range_digest",
	Description: "Condense the plot and facts of one consecutive chapter span",
	Schema: schema.Object(
		schema.Property("start_chapter", schema.Int("First chapter of the span")).Required(),
		schema.Property("end_chapter", schema.Int("Last chapter of the span")).Required(),
		schema.Property("plot", schema.String("How the main line advances across the chapters")).Required(),
		schema.Property("characters", stringList("Characters with substantive development")).Required(),
		schema.Property("world_facts", stringList("World facts established")).Required(),
		schema.Property("opened_threads", stringList("Long threads opened in this span")).Required(),
		schema.Property("resolved_threads", stringList("Long threads resolved in this span")).Required(),
	),
}

var synthesisContract = llmcontract.Contract{
	Name:        "import_book_synthesis",
	Description: "Synthesize the whole book's facts and give consecutive, complete volume and arc ranges",
	Schema: schema.Object(
		schema.Property("title", nullableString("The formal title as given in the prose; null when it cannot be confirmed")).Required(),
		schema.Property("synopsis", schema.String("The spoiler-free, reader-facing synopsis")).Required(),
		schema.Property("premise", schema.String("The story premise, described in Markdown")).Required(),
		schema.Property("characters", schema.Array("Main characters", schema.Object(
			schema.Property("name", schema.String("Character name")).Required(),
			schema.Property("aliases", stringList("Aliases and epithets")).Required(),
			schema.Property("role", schema.String("Narrative role")).Required(),
			schema.Property("description", schema.String("Character description")).Required(),
			schema.Property("arc", schema.String("Character arc")).Required(),
			schema.Property("traits", stringList("Character traits")).Required(),
			schema.Property("tier", nullableString("Character tier; null when it cannot be judged")).Required(),
		))).Required(),
		schema.Property("world_rules", schema.Array("World rules established by the prose", schema.Object(
			schema.Property("category", schema.String("Rule category")).Required(),
			schema.Property("rule", schema.String("Rule description")).Required(),
			schema.Property("boundary", schema.String("The boundary that cannot be crossed")).Required(),
		))).Required(),
		schema.Property("structure", schema.Array("Consecutive chapter ranges for volumes and arcs", schema.Object(
			schema.Property("title", schema.String("Volume title")).Required(),
			schema.Property("theme", schema.String("The volume's central conflict or theme")).Required(),
			schema.Property("arcs", schema.Array("Story arcs within the volume", schema.Object(
				schema.Property("title", schema.String("Arc title")).Required(),
				schema.Property("goal", schema.String("Arc goal")).Required(),
				schema.Property("start_chapter", schema.Int("Start chapter")).Required(),
				schema.Property("end_chapter", schema.Int("End chapter")).Required(),
			))).Required(),
		))).Required(),
		schema.Property("compass", schema.Object(
			schema.Property("ending_direction", schema.String("Ending direction")).Required(),
			schema.Property("open_threads", stringList("Long threads still unresolved")).Required(),
			schema.Property("estimated_scale", nullableString("An approximate scale; null when it cannot be judged")).Required(),
			schema.Property("last_updated", llmcontract.Nullable(schema.Int("The latest chapter number it is based on; null when not needed"))).Required(),
		)).Required(),
		schema.Property("planning_tier", schema.Enum("Planning tier", "short", "mid", "long")).Required(),
		schema.Property("story_status", schema.Enum("Whether the story is finished", storyOpen, storyClosed, storyUncertain)).Required(),
		schema.Property("status_reason", nullableString("Reason for the status judgment")).Required(),
	),
}

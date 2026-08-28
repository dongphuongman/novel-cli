package sim

import (
	"github.com/voocel/agentcore/schema"
	"github.com/voocel/ainovel-cli/internal/llmcontract"
)

func textList(description string) map[string]any {
	return schema.Array(description, schema.String(description))
}

var sourceReportContract = llmcontract.Contract{
	Name:        "simulation_source_report",
	Description: "Distil reusable writing method from a single corpus text without reproducing it",
	Schema: schema.Object(
		schema.Property("title", llmcontract.Nullable(schema.String("Optional title; null when it cannot be confirmed"))).Required(),
		schema.Property("summary", schema.String("What is worth learning from how the sample is written")).Required(),
		schema.Property("style_observations", textList("Observations on point of view, sentence shapes and descriptive texture")).Required(),
		schema.Property("common_words", textList("Categories of frequent words, imagery and transition words")).Required(),
		schema.Property("plot_patterns", textList("Patterns of plot advance, reversal and escalating conflict")).Required(),
		schema.Property("hook_patterns", textList("Hook patterns at the opening, at the chapter end, and from asymmetric information")).Required(),
		schema.Property("pacing_notes", textList("Scene density and the pacing of information release")).Required(),
		schema.Property("reader_appeal", textList("How it keeps the reader reading")).Required(),
		schema.Property("reusable_techniques", textList("Structural techniques worth borrowing")).Required(),
		schema.Property("warnings", textList("Copying and formula risks that must be avoided")).Required(),
	),
}

var synthesisContract = llmcontract.Contract{
	Name:        "simulation_synthesis",
	Description: "Merge the existing profile and the corpus reports into an actionable style-emulation profile",
	Schema: schema.Object(
		schema.Property("style", schema.Object(
			schema.Property("narrative_voice", textList("Narrative person, distance and control of information")).Required(),
			schema.Property("sentence_rhythm", textList("Sentence rhythm")).Required(),
			schema.Property("prose_texture", textList("Descriptive texture")).Required(),
			schema.Property("perspective", textList("Point-of-view rules")).Required(),
			schema.Property("mood", textList("Emotional register")).Required(),
			schema.Property("do_not_copy", textList("What must not be copied")).Required(),
		)).Required(),
		schema.Property("lexicon", schema.Object(
			schema.Property("common_words", textList("Categories of common words")).Required(),
			schema.Property("emotion_words", textList("Categories of emotion words")).Required(),
			schema.Property("scene_words", textList("Categories of scene words")).Required(),
			schema.Property("transition_words", textList("Categories of transition words")).Required(),
			schema.Property("signature_phrases", textList("Abstracted characteristics of the voice, with no source sentences")).Required(),
		)).Required(),
		schema.Property("plot_design", schema.Object(
			schema.Property("opening_patterns", textList("How it opens")).Required(),
			schema.Property("escalation_patterns", textList("How conflict escalates")).Required(),
			schema.Property("turning_point_patterns", textList("How reversals are designed")).Required(),
			schema.Property("payoff_patterns", textList("How payoff and delivery work")).Required(),
		)).Required(),
		schema.Property("hook_design", schema.Object(
			schema.Property("hook_types", textList("Hook types")).Required(),
			schema.Property("placement", textList("Where hooks are placed")).Required(),
			schema.Property("cliffhanger_patterns", textList("How suspense is held")).Required(),
			schema.Property("payoff_rules", textList("Rules for paying hooks off")).Required(),
		)).Required(),
		schema.Property("pacing_density", schema.Object(
			schema.Property("scene_density", textList("Information density per scene")).Required(),
			schema.Property("information_release", textList("Pacing of information release")).Required(),
			schema.Property("dialogue_action_ratio", textList("The ratio of dialogue, action and interiority")).Required(),
			schema.Property("compression_rules", textList("Rules for expanding and compressing content")).Required(),
		)).Required(),
		schema.Property("reader_engagement", schema.Object(
			schema.Property("methods", textList("How it draws the reader in")).Required(),
			schema.Property("emotional_drivers", textList("Emotional drivers")).Required(),
			schema.Property("progression_rewards", textList("Rewards for staged progress")).Required(),
			schema.Property("anti_patterns", textList("Anti-patterns that weaken the pull")).Required(),
		)).Required(),
		schema.Property("role_guidance", schema.Object(
			schema.Property("architect", textList("Rules for how the architect uses the profile")).Required(),
			schema.Property("writer", textList("Rules for how the writer borrows without copying")).Required(),
			schema.Property("editor", textList("Rules for what the editor checks, including plagiarism risk")).Required(),
		)).Required(),
	),
}

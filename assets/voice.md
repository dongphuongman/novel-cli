## Writing standards

These are quality guidelines, not a checklist to tick off line by line. A chapter has to work on its own terms first; only then does completeness of the checks matter.

- Establish conflict, suspense, desire or a sense of wrongness as early as possible; go light on abstract recap.
- Move the plot with action, dialogue and sensory detail; go light on summary and exposition.
- Give dialogue differences of station, subtext and an aim behind it; never sermonize.
- Render emotion through physical reaction and choice rather than labelling it outright.
- Changes in a relationship need a triggering event; nobody goes from stranger to absolute trust inside a single chapter.
- Release secrets in stages; do not explain a major revelation ahead of what the outline calls for.
- A chapter-ending hook can be a crisis, a choice, emotional aftershock, a shift in a relationship or an unfinished goal — not every chapter needs an outsized cliffhanger.
- **Strip the AI tone**: while writing, avoid every pattern listed in `reference_pack.references.anti_ai_tone` (the five categories: structure, diction, description, dialogue, pacing). The mechanically enumerable fatigue words and stock-phrase thresholds are in `working_memory.user_rules.structured` and are enforced at commit time.
- **Sentence variety**: `episodic_memory.style_stats` (when present) is the code's statistics over prose you have already written — a mirror of your own verbal tics. Actively push down the high-frequency items in it this chapter; the most common sources of ossification are the corrective construction ("not X, but Y"), a single repeated time marker ("a heartbeat later", "a moment later") and same-shaped similes stacked back to back. Rotate the closing move of the chapter (a short sentence that cuts off, a line of dialogue left ringing, an afterimage of the scene, an open question) against recent chapters, and avoid opening every chapter on a time marker like "that night", "at dawn" or "he woke".
- **No recap**: the summaries, foreshadowing and states in `episodic_memory` are a memo of what is already on the page, there for continuity checking — they are not material to be written again. Information already delivered in the previous chapter should be touched in a new chapter only when the plot needs it and only from a new angle; "previously on…" rewrites are forbidden (verbatim repetition across chapters is recorded by style_stats as repeated_sentences).

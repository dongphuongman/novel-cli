# Outline planning template

The point of this template is not to force every work into a fixed length, but to help you judge the work's tier first and then choose the outline granularity.

## Step 1: judge the length tier

### Short form / single-volume story

- Suits: a single conflict, a single goal, a small cast, a concentrated ending
- Reference length: 8–25 chapters
- Recommended format: flat `outline`

### Mid-length / multi-stage story

- Suits: staged escalation, several subplots, relationships that change
- Reference length: 25–60 chapters
- Recommended format: flat `outline`, or lightly layered

### Long-form serial / web-fiction style

- Suits: a genre with natural room for continuous escalation, sustained relational tension, several stage goals, an expandable world, a long-running mystery or a long growth line
- Reference length: 80–200+ chapters
- Recommended format: layered `layered_outline`

## Step 2: decide whether a layered outline is required

If any 2 of the following hold, prefer `layered_outline`:

- The world needs to unfold gradually rather than be explained all at once
- The protagonist's growth is not a single leap but a multi-stage escalation
- Relationships keep changing across several stages
- The middle and late book have different types of central conflict
- The story needs repeated shifts of map, faction, identity or goal
- The genre clearly reads as commercial serial fiction rather than a single-volume story

## Step 3: for long form, do not go straight to a chapter-by-chapter ledger of the whole book

The recommended planning order for long form:

1. What the book sells, and what differentiates it
2. The long-term story engines
3. Volume-level themes and escalation
4. Arc-level goals and stage turns
5. Chapter-level events and hooks

What not to do:

- Write a 20-chapter synopsis first and then stretch it
- Repeat "meet enemy — get stronger — change map" in every volume
- Escalate only the main line and never the relationships
- Spend all the big secrets early and be left repeating formulas in the middle and late book

## Flat outline template (short / mid-length)

```json
[
  {
    "chapter": 1,
    "title": "chapter title",
    "core_event": "the chapter's core event",
    "hook": "the chapter-ending hook",
    "scenes": ["scene 1", "scene 2", "scene 3"]
  }
]
```

## Layered outline template (long form — two-tier rolling expansion of volumes and arcs)

Initial planning uses a two-tier roll: the first 2 volumes have an arc skeleton, the rest are skeleton volumes; the first arc has detailed chapters.

```json
[
  {
    "index": 1,
    "title": "volume one title",
    "theme": "the central conflict / theme this volume adds",
    "arcs": [
      {
        "index": 1,
        "title": "arc one (expanded)",
        "goal": "the local goal, the resistance and the turn",
        "chapters": [
          {"chapter": 1, "title": "chapter title", "core_event": "core event", "hook": "chapter-ending hook", "scenes": ["scene 1", "scene 2"]}
        ]
      },
      {
        "index": 2,
        "title": "arc two (skeleton)",
        "goal": "an outline of this arc's goal",
        "estimated_chapters": 12,
        "chapters": []
      }
    ]
  },
  {
    "index": 2,
    "title": "volume two title",
    "theme": "volume two's theme",
    "arcs": [
      {"index": 1, "title": "arc title", "goal": "arc goal", "estimated_chapters": 15, "chapters": []},
      {"index": 2, "title": "arc title", "goal": "arc goal", "estimated_chapters": 10, "chapters": []}
    ]
  },
  {
    "index": 3,
    "title": "volume three title (skeleton volume)",
    "theme": "volume three's thematic direction",
    "estimated_chapters": 60,
    "arcs": []
  }
]
```

- Arc-level expansion: when the writing reaches a skeleton arc, the architect expands that arc's detailed chapters
- Volume-level expansion: when the writing reaches a skeleton volume, the architect expands that volume's arc structure plus the first arc's chapters

## Long-form volume checklist

Every volume must answer:

- What new information about the world does this volume add?
- Which central conflict does this volume escalate?
- What does the protagonist gain in this volume, and what do they lose?
- How does this volume change the main relationships?
- Once this volume ends, why must the story move into the next one?

## Long-form arc checklist

Every arc must answer:

- What is this arc's explicit goal?
- Where does the resistance come from — who, which rule, what cost?
- What is the turning point?
- Once this arc ends, which states have changed irreversibly?

## Chapter checklist

- Every chapter must serve the goal of the arc it belongs to
- Every chapter must contain one advance that cannot be deleted
- Vary the hooks; do not lean on "a secret is discovered" as the only pattern
- Early chapters cannot merely "introduce the world" — they must advance character and conflict at the same time

# assets content map

Before adding "a paragraph / a document / a rule" to the system, check the table below to decide where it belongs, then look at how it is wired in.

| Directory | What it holds | Who consumes it | How it is wired |
|---|---|---|---|
| `prompts/` | Worker system prompts (writer / editor / architect×2), Arbiter decision prompts and one-shot task prompts (import / simulation / revision) | `agents/build.go`, `internal/arbiter`, the imp / sim / revision runners | The Prompts fields in `load.go`. Note: simulation_guidance is injected by `load.go` at load time, so it does not appear in the md files |
| `references/` | Genre-independent writing knowledge. It does not go into the system prompt; novel_context trims it by role and chapter and injects it into `reference_pack` | writer / editor / architect | **Three wiring points**: add a field to `tools.References` + read it in `load.go` loadReferences + inject it in `novel_context.go` writerReferences / architectReferences. Dropping a file into the directory does not load it automatically |
| `references/genres/<style>/` | Genre-specific knowledge (style-references / arc-templates) | As above, loaded when `style != default` | `load.go` loadReferences |
| `rules/` | The deprecated built-in rules directory; the mechanical baseline has moved into code, and user rules come from natural-language snapshots in `~/.ainovel/rules/*.md` / `./.ainovel/rules/*.md` | `userrules.Service` normalizes them into `meta/user_rules.json`; `novel_context` injects them; `commit_chapter` checks them | The built-in baseline lives in `SystemDefaults()` in `internal/rules/snapshot.go`; the user's `.md` files carry no formatting and no YAML and are normalized from natural language |
| `styles/<style>.md` | Genre writing-style instructions | Assembled into the **writer** system prompt (`agents/build.go`) | The filename is the value of `config.style`. This and `references/genres/<style>/` are two carriers of the same genre concept: the former is style instruction, the latter is knowledge material |

## Deciding where new content belongs (five questions)

1. Must this process be **guaranteed**? → Do not write a prompt; write a code constraint (StopAfterTools / a tool guard / the Flow Router)
2. Is this a decision criterion? → Table-driven flow goes in `internal/flow/router.go`; semantic judgment goes in `prompts/arbiter-*.md`
3. Is this one role's craft or execution standard? → `prompts/<role>.md`
4. Is this a mechanically enumerable default rule (banned words / thresholds)? → `SystemDefaults()` in `internal/rules/snapshot.go`; user-defined rules go in `.ainovel/rules/*.md` and are consumed via the normalized snapshot (length is a soft semantic constraint, so it goes through preferences and is never a mechanical rule)
5. Is this writing knowledge? → `references/` (remember all three wiring points)

## Keeping things consistent

The envelope paths a prompt references (`working_memory.*` and the rest) must stay consistent with `novel_context`. The shape of a tool's arguments is defined only in the tool's Schema; a prompt adds only the business semantics the Schema cannot express, and no longer duplicates the JSON argument list or shape examples.

A prompt may describe how an individual worker executes, but global routing, state transitions and recovery logic are defined by the code alone. Any step that can be determined from Store facts belongs in the Router or a Tool; only judgments that require understanding the novel's content or the user's intent are left to the model.

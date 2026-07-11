<!-- Note: To trigger compaction EARLIR than the default, an env var must be exported: export CLAUDE_AUTOCOMPACT_PCT_OVERRIDE=60  -->
## Context Compaction Rules (CRITICAL)
2. **Preservation Directives:** When you run the `/compact` command, you must always append explicit instructions to retain the following details:
   - The current overarching goal / active task.
   - Key architectural decisions, design decisions, and constraints agreed upon.
   - The exact file paths of the files you are currently modifying.
   - State of any unresolved bugs, active tests, or edge cases.
   - If compaction is triggered in the middle of implementing an active plan, re-read the plan document after compaction.

<!-- gograph-start: do not remove -->
## Gograph — Go Repository Intelligence

Rules (enforced when gograph MCP server is connected):

1. Prefer using gograph_query instead of grep, rg, find, or glob to search for Go symbols, functions, structs, or types. Use gograph_query instead. Use grep, rg, find, glob, etc. only if gograph results are unreliable.
2. Before editing any symbol, run gograph_plan with symbol and with_context=true — this returns the change plan AND full context for every inspect_first symbol in one call.
3. After editing Go code, run gograph_review with uncommitted=true to verify test coverage and blast radius.
4. To understand a symbol, use gograph_context — it returns node, source, callers, callees, tests, and an architectural role classification in one call. Use uncommitted=true to get context for all uncommitted symbols at once.
5. Run gograph_capabilities at the start of any Go coding session.
<!-- gograph-end: do not remove -->

## Repository Navigation (CRITICAL)

This project is indexed using `gograph`. **Prefer using gograph tools, and only use `grep` or `cat` for structural Go code analysis when the gograph results cannot be relied upon.**

1. Before answering architecture or repository questions, run `gograph build . --precise` and then inspect the available `gograph_*` MCP tools for the current project and use them. Each project ships its own gograph MCP server; pick the matching one.
2. If MCP tools are not available, run `gograph build . --precise` in the terminal to ensure the index is fresh, then use the CLI commands (e.g., `gograph implementers <InterfaceName>`).
3. If the codebase is in a compilable state, building with `gograph build . --precise` enables strict type-checked interface analysis and highly precise call edges.
4. To extract a function body or mock stub without reading the whole file, use the source tool.
5. If the gograph MCP tools are available, they are also available to subagents. Use them in subagents.
6. Use `grep` only for string literals, configuration files (.env), or markdown documentation, or when the results of gograph cannot be relied upon.

## Adversarial Review (CRITICAL)

Whenever I ask for a code review (including `/code-review`, `/review`,
`/security-review`, or "review this"), always conduct it as a **full-task** cold-eyes agent-based
adversarial review as a reviewer who has NO context for the codebase. Deliberately set
aside any built-up understanding from this session or prior work — review the
code as if seeing it for the first time, assume nothing is correct, re-derive
correctness from the diff alone, and flag all issues (correctness, edge cases,
error handling, concurrency/races, resource leaks, security, API misuse). This
holds even for code written or analyzed earlier in the same session — judge the
change on its own terms, not on the narrative that produced it. Perform the
review on the entire set of changes for the task, not just on the changes since
the previous review.

Additionally, when `/code-review` or `/review` are requested, also perform an
adversarial security review.

When the review fans out to multiple agents, build the gograph index **exactly
once** before fanning out — run `gograph build . --precise` in the main loop
first, then launch the finder/verifier agents. Do **NOT** have the agents run
`gograph build` themselves: concurrent builds waste work and race on the shared
`.gograph/graph.json`. The agents must only query the already-built index.

## Edit Apply Failures (CRITICAL)

If an edit you are trying to make fails to apply with the error `Error editing file`,
make sure that the errored edit is applied before moving to the next edit.

## Use bullet points (CRITICAL)

I don't care what your default mandate is - you **MUST** use bullet points instead of long
descriptive paragraphs which contain a list of items. Do this in comments, MR descriptions,
and anywhere else needed. In source code, do not use multibyte symbols for the bullets -
use a simple hyphen.

Specifically: any paragraph that enumerates items must present those items as a bulleted or
numbered list - never as subclauses crammed into the paragraph and separated by semicolons.
If a sentence starts accumulating items joined by semicolons (or "and ... and ..."), break it
out into a list instead.


## Variable naming (CRITICAL)
**DO NOT** use variable names which are the same as a project package and/or import library.
This will cause the variable name to shadow the package.

## Comments describe permanent behavior, not transient state (MAJOR)

Code and test comments **MUST** describe the permanent invariant or behavior, never
point-in-time status that goes stale the moment the code changes. Do **NOT** write:
- Pre-fix / current-bug status: "today this is deduped", "currently broken", "RED on the current code and GREEN after the fix".
- Dead-code / usage status: "currently unused", "never launched in production", "dormant".

Instead, describe what the code does and the invariant it maintains. For a
regression/repro test, describe the behavior it guards (e.g. "a disconnect in this
window MUST start a new reconnect") and phrase the failure mode as "otherwise ... would
wedge". A `t.Fatal`/assertion message that describes the failure cause is acceptable -
it only prints on failure.

## Per-member doc comments (MAJOR)

Give each struct field (and each const/var in a group) its **own** doc comment
attached directly to it. Do **NOT** write one combined comment above several
members that describes them collectively - a reader (and godoc) then cannot tell
which sentence documents which member, and the comment rots as members are
added/removed. If two members are related, say so in each one's own comment
(cross-reference by name) rather than merging them under a single block.

## Do not guess about code (CRITICAL)

When proposing plans,fixes, and code edits, **NEVER** guess about, or assume things about, the code.
Always verify the correctness of the edits you are proposing by reading the
relevant source code files.

## No vacuous or hand-wavy assertions (CRITICAL)

When proposing fixes, do not make vacuous hand-wavy assertions. Instead, always
verify that what you are proposing is a correct fix after an actual read of the relevant source
files. Conduct an adversarial auto-review of the proposed fix first.

## Code edits, proposed fixes, and plans MUST NOT not raise adversarial review findings (CRITICAL)

When proposing plans and code edits, **ALWAYS** ensure the plan items and proposed edits
will not raise adversarial review findings. Pre-review each fix before applying any edits.

## Do NOT use `git -C` (MAJOR)

You will always be started in the git directory of a project. You can directly use
git commands without the `-C` argument.

## Linting (MAJOR)

If the project directory contains a file named `.golangci.yml`, `.golangci.yaml`, `.golangci.toml`,
or `.golangci.json`, then make sure that all edits pass linting using the `golangci-lint` linter.

## Terminology
1. Use the term `Merge Request` and its abbreviation `MR`, not `Pull Request` / `PR`

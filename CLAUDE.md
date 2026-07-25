<!-- These rules build upon those in ~/.claude/CLAUDE.md and are an addition to those rules -->
<!-- gograph-start: do not remove -->
## Gograph — Go Repository Intelligence

Rules (enforced when gograph MCP server is connected):

1. Prefer using gograph_query instead of grep, rg, find, or glob to search for Go symbols, functions, structs, or types. Use gograph_query instead. Use grep, rg, find, glob, etc. only if gograph results are unreliable.
2. Before editing any symbol, run gograph_plan with symbol and with_context=true — this returns the change plan AND full context for every inspect_first symbol in one call.
3. After editing Go code, run gograph_review with uncommitted=true to verify test coverage and blast radius.
4. To understand a symbol, use gograph_context — it returns node, source, callers, callees, tests, and an architectural role classification in one call. Use uncommitted=true to get context for all uncommitted symbols at once.
5. Run gograph_capabilities at the start of any Go coding session.
6. If the gograph MCP tools are available, they are also available to subagents. Use them in subagents.
<!-- gograph-end: do not remove -->

## Repository Navigation (CRITICAL)

This project is indexed using `gograph`. **Prefer using gograph tools, and only use `grep` or `cat` for structural Go code analysis when the gograph results cannot be relied upon.**

1. Before answering architecture or repository questions, run `gograph build . --precise` and then inspect the available `gograph_*` MCP tools for the current project and use them. Each project ships its own gograph MCP server; pick the matching one.
2. If MCP tools are not available, run `gograph build . --precise` in the terminal to ensure the index is fresh, then use the CLI commands (e.g., `gograph implementers <InterfaceName>`).
3. If the codebase is in a compilable state, building with `gograph build . --precise` enables strict type-checked interface analysis and highly precise call edges.
4. To extract a function body or mock stub without reading the whole file, use the source tool.
5. If the gograph MCP tools are available, they are also available to subagents. Use them in subagents.
6. Use `grep` only for string literals, configuration files (.env), or markdown documentation, or when the results of gograph cannot be relied upon.

## Adversarial Code Review (CRITICAL)

Whenever I ask for a code review (including `/code-review`, `/review`,
`/security-review`, or "review this"), DO NOT do an ad-hoc inline review
where you just read the diff and reason in the main loop.

Always conduct reviews as a **full-task** adversarial review via the **proper agent-based adversarial review**:
gather the diff, then fan out independent finder agents (via the Agent tool) across the review angles,
verify candidates with verifier agents, sweep for gaps, then synthesize.

Perform the review on the entire set of changes for the task, not just on the changes since
the previous review.

Additionally, when `/code-review` or `/review` are requested, also perform an
adversarial security review.

- Assume nothing is correct, re-derive correctness from the diff alone
- flag all issues (correctness, edge cases, error handling, concurrency/races, resource leaks, security, API misuse).

When the review fans out to multiple agents, build the gograph index **exactly
once** before fanning out — run `gograph build . --precise` in the main loop
first, then launch the finder/verifier agents. Do **NOT** have the agents run
`gograph build` themselves: concurrent builds waste work and race on the shared
`.gograph/graph.json`. The agents must only query the already-built index.




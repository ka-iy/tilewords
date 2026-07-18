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


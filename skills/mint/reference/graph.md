# Code Graph — Reference

Load when `config.graph.enabled` is `true` and the task involves code changes.

---

## When to Use the Graph

| Situation | Graph query | Tool |
|-----------|------------|------|
| Before decomposing a feature | Understand module boundaries, find natural seams | `search_graph`, `get_architecture` |
| Before implementing a spec | Trace blast radius of changes, find callers/callees | `trace_call_path`, `detect_changes` |
| Populating `<can-modify>` | Find all files transitively affected by a change | `trace_call_path` outbound + file mapping |
| Understanding cross-boundary effects | Backend route → OpenAPI spec → frontend client → UI | `trace_call_path` cross_service mode |
| During security review | Trace data flow from user input to output | `trace_call_path` data_flow mode |
| During quality review | Find god functions (high degree), coupling hotspots | `search_graph` with min_degree filter |
| Adversarial testing | Target high-fanout functions (many callers = high risk) | `search_graph` sorted by degree |
| Dead code detection | Find unreachable functions | `query_graph` with Cypher degree filter |
| Understanding architecture | Get high-level overview before planning | `get_architecture` |
| Checking if function exists | Before writing new code, check if it already exists | `search_graph` by name |

## Available MCP Tools

These are exposed by `codebase-memory-mcp` as MCP tools. Use them via tool calls when
the graph is enabled.

### Core Queries

**`search_graph`** — Find nodes by label, name pattern, file pattern, degree.
```json
{
  "project": "mint",
  "label": "Function",
  "name_pattern": ".*Handler.*",
  "min_degree": 5,
  "limit": 20
}
```

**`trace_call_path`** — BFS traversal from a function. Find callers or callees.
```json
{
  "function_name": "buildConfig",
  "project": "mint",
  "direction": "inbound",
  "depth": 3,
  "mode": "calls"
}
```
Modes:
- `calls` — follow CALLS edges (who calls this? what does this call?)
- `data_flow` — follow data through arguments and return types
- `cross_service` — follow HTTP_CALLS and ASYNC_CALLS across service boundaries

**`detect_changes`** — Git diff → impacted symbols. Shows what's affected by recent changes.
```json
{
  "project": "mint",
  "scope": "working_tree",
  "depth": 2
}
```

**`query_graph`** — Execute Cypher queries for complex analysis.
```cypher
MATCH (f:Function)-[:CALLS]->(g:Function)
WHERE f.file_path CONTAINS 'auth'
RETURN f.name, g.name, g.file_path
LIMIT 20
```

### Architecture

**`get_architecture`** — High-level project overview: languages, packages, entry points,
routes, hotspots (high-degree nodes), module clusters.

**`get_graph_schema`** — Node/edge counts and patterns. Useful for understanding graph size.

### Code Reading

**`get_code_snippet`** — Read source code for a symbol by qualified name. Includes
neighbor context (callers, callees) if requested.

**`search_code`** — Regex grep enriched with graph context. Matches are mapped to
containing functions and ranked by relevance (definitions first, popular next, tests last).

## Cross-Boundary Effect Detection

This is mint's key enhancement over raw graph queries. When a change touches one layer,
trace the effects through the full stack:

```
Backend route change
  → detect_changes (direct)
  → trace_call_path outbound (what calls the route handler?)
  → search_graph for Route nodes (does an OpenAPI/Swagger spec reference this?)
  → trace_call_path cross_service (frontend HTTP clients calling this endpoint?)
  → identify UI components affected

Config/schema change
  → detect_changes (direct)
  → search_graph for files importing the config
  → trace_call_path outbound from each importer
  → identify all dependent behavior
```

The orchestrator should chain these queries when decomposing or planning tasks that
touch shared interfaces (API routes, schemas, configs, types).

## Blast Radius Analysis

Before decomposing, estimate blast radius:

1. `detect_changes` for the target files → get directly impacted symbols
2. For each impacted symbol: `trace_call_path` inbound (depth 2) → get callers
3. Collect all affected files across all callers
4. These files go in `<can-modify>` or become separate specs if scope is too large

**Rule of thumb:**
- Blast radius ≤ 3 files → single spec
- Blast radius 4-8 files → consider splitting into 2 specs
- Blast radius > 8 files → definitely split, check for architectural boundaries

## Context Mode Integration

When `config.context.enabled` is `true`, prefer running graph queries through context-mode
tools to keep verbose JSON output out of the main context window:

- **`ctx_execute`** for graph CLI queries — the raw JSON stays sandboxed, only the summary
  enters context. Especially important for `trace_call_path` (can return 50+ nodes) and
  `query_graph` (arbitrary result size).
- **`ctx_index`** the graph results — architecture overview, hotspot lists, and blast radius
  data can be indexed into FTS5 for fast retrieval across multiple agent dispatches.
- **`ctx_search`** for cached graph data — if architecture was queried during decompose,
  the planner can search for it instead of re-querying.

Graph data is highly cacheable — the codebase doesn't change between pipeline steps.
Index once during plan setup, search from subagents.

If context-mode is unavailable, graph queries run via normal Bash with raw output.
This works but uses more tokens.

## Graceful Degradation

When `config.graph.enabled` is `true` but the graph is unavailable (not installed, not
indexed, MCP server not running):

- **Never block.** Fall back to file-based analysis (grep, read).
- **Log a warning:** "Graph unavailable — using file-based analysis. Run `mint doctor` to fix."
- **Don't skip the step.** Do the analysis without the graph — it's less precise but still useful.

## Auto-Indexing

When `config.graph.autoIndex` is `true`, the graph is reindexed:
- On `mint init` (first index)
- On plan mode setup (if stale — checked via `index_status`)
- Background watcher keeps it fresh between sessions

Manual reindex: tell Claude `"reindex the code graph"` or use
`codebase-memory-mcp cli index_repository '{"repo_path": "."}'`

## Troubleshooting

If `codebase-memory-mcp` binary fails to run:
- **GLIBC version error** — the binary requires glibc 2.38+. Upgrade your system or
  build from source: `git clone ... && make -f Makefile.cbm`
- **Not found** — run `mint doctor` to check, or install manually
- **Indexing slow** — first index is full; subsequent are incremental. Large repos (>50K files)
  may take several minutes on first index.

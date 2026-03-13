# context-mode MCP Tool Reference

Full reference for context-mode's 9 MCP tools. context-mode keeps raw tool output out of the
context window via sandboxed execution and provides FTS5 full-text search over indexed content.

## ctx_execute

Run code in a sandboxed subprocess. Only stdout enters context.

```
ctx_execute(language: "shell", code: "npm test", intent: "test failures")
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `language` | enum | Yes | — | `javascript`, `typescript`, `python`, `shell`, `ruby`, `go`, `rust`, `php`, `perl`, `r`, `elixir` |
| `code` | string | Yes | — | Source code to execute |
| `timeout` | number | No | 30000 | Max execution time in ms |
| `background` | boolean | No | false | Keep process running after timeout (for servers/daemons) |
| `intent` | string | No | — | What to look for in output. When provided and output >5KB, auto-indexes into FTS5 and returns only matching sections |

**Returns:** stdout text, or intent-filtered snippets with searchable vocabulary terms when intent is provided and output exceeds 5KB.

**Example — run tests with intent filtering:**
```
ctx_execute(
  language: "shell",
  code: "npm test -- --verbose",
  intent: "errors and failures"
)
```
Returns only test failure sections instead of full test output.

**Example — run a build:**
```
ctx_execute(
  language: "shell",
  code: "npm run build",
  intent: "errors"
)
```

## ctx_execute_file

Process a file in sandbox. File content loaded as `FILE_CONTENT` variable in the chosen language.
Raw file never enters context.

```
ctx_execute_file(path: "src/config.ts", language: "javascript", code: "console.log(FILE_CONTENT.length)")
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `path` | string | Yes | — | File path to process |
| `language` | enum | Yes | — | Same language options as ctx_execute |
| `code` | string | Yes | — | Code that processes FILE_CONTENT |
| `timeout` | number | No | 30000 | Max execution time in ms |
| `intent` | string | No | — | Intent-driven filtering (same as execute) |

**Returns:** stdout from the processing code. FILE_CONTENT is a string variable containing
the file's contents.

**Example — analyze a large diff:**
```
ctx_execute_file(
  path: "/tmp/diff.patch",
  language: "javascript",
  code: "const lines = FILE_CONTENT.split('\\n'); console.log(`${lines.length} lines changed`); console.log(lines.filter(l => l.startsWith('+')).slice(0, 20).join('\\n'))"
)
```

**Example — extract function signatures from a file:**
```
ctx_execute_file(
  path: "src/server.ts",
  language: "javascript",
  code: "const fns = FILE_CONTENT.match(/(?:export )?(?:async )?function \\w+\\([^)]*\\)/g); console.log(fns?.join('\\n') || 'none')"
)
```

## ctx_batch_execute

Run multiple commands + search multiple queries in ONE call. Primary tool for research tasks.

```
ctx_batch_execute(
  commands: [
    { label: "tests", command: "npm test" },
    { label: "lint", command: "npm run lint" }
  ],
  queries: ["error handling pattern"],
  source: "project-analysis"
)
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `commands` | array | No | — | Array of `{ label, command, language?, timeout? }` |
| `queries` | array | No | — | Search queries to run against indexed content |
| `source` | string | No | — | Filter searches to a specific source |

**Returns:** Combined results from all commands and searches. Each command's output is
auto-indexed into FTS5.

**Example — multi-step research:**
```
ctx_batch_execute(
  commands: [
    { label: "deps", command: "cat package.json | jq '.dependencies'" },
    { label: "structure", command: "find src -name '*.ts' | head -30" },
    { label: "tests", command: "find tests -name '*.test.*' | head -20" }
  ]
)
```

## ctx_index

Chunk content into FTS5 with BM25 ranking for later search.

```
ctx_index(path: "docs/api.md", source: "api-docs")
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `content` | string | No | — | Inline content to index (enters context as param -- use sparingly) |
| `path` | string | No | — | File path to index server-side (preferred -- content stays out of context) |
| `source` | string | No | — | Label for the indexed source |

**Returns:** Source ID, chunk count, code chunk count.

**Prefer `path:` over `content:`.** Using `content:` sends the data through context, defeating
the purpose. Use `path:` to index server-side.

**Example — index a documentation file:**
```
ctx_index(path: "docs/architecture.md", source: "architecture")
```

## ctx_search

Query indexed content with multiple queries in one call.

```
ctx_search(queries: ["error handling", "validation pattern"], source: "project-analysis")
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `queries` | array of strings | Yes | — | Search queries |
| `source` | string | No | — | Filter to specific source label |
| `limit` | number | No | — | Max results per query |

**Returns:** Matching content chunks ranked by BM25 relevance.

**Progressive throttling:**
- Calls 1-3: normal (2 results/query)
- Calls 4-8: reduced (1 result/query + warning)
- Calls 9+: blocked (redirects to ctx_batch_execute)

**Search layers (6-layer fallback):**
1. Porter stemming AND (most precise)
2. Porter stemming OR
3. Trigram substring AND
4. Trigram substring OR
5. Fuzzy Levenshtein + Porter (AND then OR)
6. Fuzzy Levenshtein + Trigram (AND then OR)

**Example — search after indexing:**
```
ctx_search(queries: ["authentication middleware", "session handling"], source: "api-docs")
```

## ctx_fetch_and_index

Fetch URL, convert HTML to markdown, chunk and index into FTS5. Raw page content never
enters context.

```
ctx_fetch_and_index(url: "https://docs.example.com/api", source: "external-docs")
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | Yes | — | URL to fetch |
| `source` | string | No | — | Label for indexed content |

**Returns:** Source ID, chunk count. Content is searchable via ctx_search.

**Example — fetch and search documentation:**
```
ctx_fetch_and_index(url: "https://docs.example.com/api/auth", source: "auth-docs")
ctx_search(queries: ["rate limiting", "token refresh"], source: "auth-docs")
```

## ctx_stats

Show context savings, call counts, session statistics.

```
ctx_stats()
```

No parameters. Returns per-tool call counts, bytes saved, and savings ratio.

## ctx_doctor

Diagnose installation: runtimes, hooks, FTS5, versions.

```
ctx_doctor()
```

No parameters. Returns checklist of system health: available runtimes, hook registration
status, FTS5 availability, version info.

## ctx_upgrade

Upgrade to latest version from GitHub, rebuild, reconfigure hooks.

```
ctx_upgrade()
```

No parameters. Pulls latest release, rebuilds, and re-registers hooks.

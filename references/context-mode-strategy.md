# Context Mode Strategy for mint Agents

Decision tree and rules for when to use which context-mode tool. Keeps raw output out of context.

## Decision Tree

```
What do you need?
  |
  +-- Reading a file's content for analysis?
  |     -> ctx_execute_file (process without loading raw content)
  |
  +-- Running a command (test, build, lint)?
  |     -> ctx_execute with language: "shell"
  |        Add intent: "errors and failures" for large output
  |
  +-- Running multiple commands?
  |     -> ctx_batch_execute (one call instead of many)
  |
  +-- Fetching documentation or a URL?
  |     -> ctx_fetch_and_index + ctx_search
  |        (never WebFetch — raw HTML wastes context)
  |
  +-- Querying previously indexed content?
  |     -> ctx_search (max 8 calls before throttle)
  |
  +-- Indexing a local file for later search?
  |     -> ctx_index with path: parameter (never content:)
  |
  +-- Simple one-line command with small output?
        -> Regular Bash (context-mode overhead not worth it)
```

## Cost Comparison

| Operation | Without context-mode | With context-mode | Savings |
|-----------|---------------------|-------------------|---------|
| `npm test` (large suite) | ~15K tokens raw output | ~500 tokens (intent-filtered) | ~97% |
| `npm run build` | ~8K tokens | ~200 tokens (errors only) | ~98% |
| Fetch API docs page | ~20K tokens (raw HTML) | ~300 tokens (searched sections) | ~99% |
| Analyze large file | ~5K tokens (full file in context) | ~500 tokens (processed output) | ~90% |
| 5 research commands | ~25K tokens (5 separate calls) | ~2K tokens (1 batch call) | ~92% |
| grep across codebase | ~10K tokens | ~500 tokens (ctx_execute) | ~95% |

## Rules for Agents

### ALWAYS use context-mode for:

1. **Test runners** — `ctx_execute(language: "shell", code: "npm test", intent: "errors and failures")`
   Test output is verbose. Intent filtering returns only failures.

2. **Build tools** — `ctx_execute(language: "shell", code: "npm run build", intent: "errors")`
   Build logs are massive. Only errors matter.

3. **Lint/type checks** — `ctx_execute(language: "shell", code: "npm run lint", intent: "errors")`
   Same pattern as builds.

4. **Large file analysis** — `ctx_execute_file(path: "...", language: "javascript", code: "...")`
   Process files without loading them into context.

5. **URL fetching** — `ctx_fetch_and_index(url: "...", source: "...")` then `ctx_search(...)`
   Never use WebFetch when context-mode is available.

6. **Multi-command research** — `ctx_batch_execute(commands: [...], queries: [...])`
   Run all research commands in one call instead of sequential Bash calls.

7. **Log analysis, API responses, data inspection** — any command that produces >1KB output.

### NEVER use context-mode for:

1. **Simple one-line commands** with small output — `git status`, `ls`, `pwd`, `echo`.
   The overhead of sandboxing isn't worth it for tiny outputs.

2. **`ctx_index(content: ...)`** with data already in context — this doubles context usage.
   Always use `ctx_index(path: ...)` to index files server-side.

3. **Ignoring progressive throttling warnings** on ctx_search — after 8 calls, switch to
   ctx_batch_execute with queries parameter.

4. **ctx_execute for interactive commands** — anything requiring stdin input won't work
   in the sandbox.

## Anti-Patterns

| Anti-Pattern | Why It's Bad | Do This Instead |
|--------------|-------------|-----------------|
| `ctx_index(content: large_string)` | Content enters context as parameter | `ctx_index(path: "/path/to/file")` |
| Using WebFetch when context-mode is on | Raw HTML floods context | `ctx_fetch_and_index` + `ctx_search` |
| 10+ sequential ctx_search calls | Progressive throttle blocks at 9 | Use `ctx_batch_execute(queries: [...])` |
| ctx_execute for `git status` | Overhead > benefit for tiny output | Regular Bash |
| Ignoring intent parameter | Full output enters context | Add `intent:` for large outputs |
| ctx_execute with `background: true` left running | Zombie processes | Only use for intentional long-running tasks |

## Fallback Protocol

If context-mode tools are unavailable (MCP server not running, not installed):
- Fall back to standard tools transparently
- Bash instead of ctx_execute
- Read instead of ctx_execute_file
- WebFetch instead of ctx_fetch_and_index
- Grep instead of ctx_search

Never block on context-mode unavailability. It's an optimization, not a dependency.

## Session Continuity

When context-mode's session hooks are active:
- File operations, task state, errors, and decisions are tracked automatically
- After context compaction, use `ctx_search(queries: [...], source: "session-events")` to
  recover working state
- No mint-specific session code needed — context-mode handles this natively

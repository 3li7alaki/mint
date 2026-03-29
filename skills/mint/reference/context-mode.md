# Context Mode — Reference

Optional integration with context-mode MCP server.

---

## Detection

Check `config.context.enabled`:
- If false: skip entirely, use standard tools
- If true: verify MCP tools respond (`ctx_doctor` or test `ctx_execute`)
  - If available: agents prefer sandboxed execution
  - If unavailable: log WARNING, fall back to standard tools

## Agent Behavior When Enabled

- `ctx_execute` instead of Bash for data-heavy operations
- `ctx_execute_file` instead of Read for large files
- `ctx_fetch_and_index` instead of WebFetch for URLs
- `ctx_search` for full-text search over indexed content

## Session Continuity

context-mode's session hooks automatically track operations. After compaction, use
`ctx_search(queries: [...], source: "session-events")` to recover state.

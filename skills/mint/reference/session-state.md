# Session State — Reference

Load when you need session lifecycle details beyond what the router covers.

---

## Session ID

Generated once per process: hex timestamp (12 chars) + random suffix (8 chars).
Example: `0195e3a1b2c0-a1b2c3d4`. Cached at module level — stable for session lifetime.
Lexicographically sortable by creation time.

## Schema

```json
{
  "mintInvoked": true,
  "invokedAt": "ISO-8601",
  "task": "short task description",
  "mode": "quick|plan|ship|research|verify",
  "autoCommitOverride": null,
  "designContextLoaded": false,
  "activeSpec": null
}
```

## Field Details

| Field | Type | Purpose |
|-------|------|---------|
| `mintInvoked` | boolean | Hooks check this |
| `invokedAt` | string | ISO timestamp |
| `task` | string | Current task |
| `mode` | string | Routed mode |
| `autoCommitOverride` | bool\|null | Session override for autocommit |
| `designContextLoaded` | boolean | Design context loaded for this task |
| `activeSpec` | string\|null | Currently executing spec path (for scope enforcement) |

## Lifecycle

1. **On invocation:** Write session state. Atomic write (tmp + rename). Verify by reading back.
2. **On autocommit override:** Set `autoCommitOverride`. Persists for entire session.
3. **On completion:** Delete session file. Verify gone.
4. **On abandonment:** Also delete — stale state must not leak.
5. **Hooks:** Pre-edit hook scans session files (most recent first by name).
6. **Stale cleanup:** Sessions > 24h cleaned by `cleanStaleSessions()`. `mint clean` also cleans.

## Autocommit Override Detection

- Flags: `--no-commit`, `--commit`
- Verbal: "no autocommit", "don't commit", "no commits"
- Mid-session: "stop committing" / "start committing again"

When detected: set in session state, announce once, never re-ask.

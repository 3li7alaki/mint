# Browse Mode

Browser automation via PinchTab. Requires `browser.enabled: true` in config.

---

## Process

1. **Pre-flight** — verify PinchTab running at `browser.baseUrl` via `/health`
2. **Load cookies** — if `browser.persistSessions`: read `.mint/.browser-sessions.json`
3. **Route to agent:**
   - Debug tasks → `browser-debugger`
   - Everything else → `browser-runner`
4. **Save cookies** — update `.mint/.browser-sessions.json` with returned cookies
5. **Return result** — page state, data, confirmation, or debug report

## Graceful Degradation

If PinchTab not running:
- Auto-start (`pinchtab &`) if `browser.autoStart` is true
- Poll for health (max 10s)
- If still down: WARNING with install instructions. Never block.

## Commands

- `/browse <url> [task]` — navigate and interact
- `/screenshot [url]` — capture page screenshot
- `/scrape <url> [what]` — extract structured data
- `/browser login <url>` — user logs in, mint saves session
- `/browser sessions` — list saved sessions
- `/browser switch <name>` — switch active session
- `/browser clear` — wipe all sessions

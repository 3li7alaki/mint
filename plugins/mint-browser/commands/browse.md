# /browse Command

Navigate to a URL and perform browser automation tasks.

## Usage

```
/browse <url> [task description]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `url` | Yes | URL to navigate to |
| `task description` | No | What to do on the page (e.g., "fill in the form", "check if the button renders") |

## Examples

```bash
# Navigate and explore
/browse https://myapp.com/login

# Navigate and perform a task
/browse https://myapp.com/login "fill in the form and submit"

# Check local dev server
/browse localhost:3000 "check if the new button renders"

# Scrape information
/browse https://docs.example.com "find the API endpoint for user creation"
```

## Process

1. **Parse arguments:**
   - Extract URL (first argument)
   - Remaining text is the task description
   - If no URL provided: show usage error

2. **Normalize URL:**
   - If URL has no protocol, prepend `http://`
   - If URL is just a port (e.g., `:3000`), prepend `http://localhost`

3. **Load browser config** from `.mint/config.json`:
   - If `browser` key missing: suggest running mint init
   - If `browser.enabled` is `false`: inform user browser features are disabled

4. **Invoke browser-runner agent** with:
   - URL: the normalized URL
   - Task: the task description (or "navigate and snapshot" if none provided)
   - Config: browser config from .mint/config.json

5. **Return result** from browser-runner agent

## Errors

### No URL Provided

```
Error: URL is required.

Usage: /browse <url> [task description]

Examples:
  /browse https://myapp.com/login
  /browse localhost:3000 "check the new button"
```

### Browser Plugin Not Configured

```
Error: Browser plugin not configured.

Add browser config to .mint/config.json:
{
  "browser": {
    "enabled": true,
    "baseUrl": "http://localhost:9867"
  }
}

Or run: mint init (to auto-configure)
```

### PinchTab Not Running

```
Warning: PinchTab not running at http://localhost:9867

Start PinchTab:
  pinchtab &

Or install:
  curl -fsSL https://pinchtab.com/install.sh | sh
```

## Notes

- The browser-runner agent handles all PinchTab interactions
- If no task description is given, the agent navigates and returns a page snapshot
- URLs without a protocol get `http://` prepended automatically
- The command reads `browser.token` from config for authenticated PinchTab instances

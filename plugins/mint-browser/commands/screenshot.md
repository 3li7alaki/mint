# /screenshot Command

Capture a screenshot of a web page.

## Usage

```
/screenshot [url]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `url` | No | URL to screenshot. Defaults to `browser.devServer` from config |

## Examples

```bash
# Screenshot the dev server
/screenshot

# Screenshot a specific URL
/screenshot https://myapp.com/dashboard

# Screenshot a specific local page
/screenshot localhost:3000/login
```

## Process

1. **Determine URL:**
   - If URL provided: use it (normalize protocol)
   - If no URL: use `browser.devServer` from `.mint/config.json`
   - If no URL and no devServer: show error

2. **Load browser config** from `.mint/config.json`

3. **Pre-flight check:**
   - Check PinchTab is running at `browser.baseUrl`
   - If not running: show warning with start instructions

4. **Navigate to URL:**
   ```bash
   curl -s -X POST $BASE_URL/navigate \
     -H "Content-Type: application/json" \
     -d '{"url": "TARGET_URL"}'
   ```

5. **Wait 3 seconds** for page render

6. **Capture screenshot:**
   ```bash
   curl -s "$BASE_URL/screenshot?raw=true" -o /tmp/mint-screenshot-$(date +%s).jpg
   ```

7. **Return** screenshot file path and basic page info

## Output

```
Screenshot saved: /tmp/mint-screenshot-1709123456.jpg
URL: http://localhost:3000/login
Title: Login - My App
```

## Errors

### No URL and No Dev Server

```
Error: No URL provided and no devServer configured.

Usage: /screenshot [url]

Configure a dev server in .mint/config.json:
{
  "browser": {
    "devServer": "http://localhost:3000"
  }
}
```

### PinchTab Not Running

```
Warning: PinchTab not running at http://localhost:9867

Start PinchTab:
  pinchtab &
```

## Notes

- Screenshots are saved to `/tmp/` with a timestamp filename
- The command always waits 3 seconds after navigation for the page to render
- If the page requires authentication, ensure PinchTab has an active session (use `/browse` first to log in)
- Screenshots use JPEG format at default quality. For higher quality: use the API directly with quality parameter

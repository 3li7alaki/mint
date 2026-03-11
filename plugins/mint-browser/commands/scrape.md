# /scrape Command

Extract structured data from a web page.

## Usage

```
/scrape <url> [what to extract]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `url` | Yes | URL to scrape |
| `what to extract` | No | Description of what data to extract (e.g., "list all endpoint names and methods") |

## Examples

```bash
# Extract all text content
/scrape https://docs.example.com/api

# Extract specific information
/scrape https://docs.example.com/api "list all endpoint names and methods"

# Extract structured data
/scrape https://example.com/pricing "pricing tiers with features and prices"

# Extract from local dev server
/scrape localhost:3000/users "list all user names and emails in the table"
```

## Process

1. **Parse arguments:**
   - Extract URL (first argument)
   - Remaining text is the extraction description
   - If no URL: show usage error

2. **Normalize URL:**
   - Prepend `http://` if no protocol
   - Prepend `http://localhost` if just a port

3. **Load browser config** from `.mint/config.json`

4. **Pre-flight check:**
   - Check PinchTab is running
   - If not: show warning and exit

5. **Navigate to URL:**
   ```bash
   curl -s -X POST $BASE_URL/navigate \
     -H "Content-Type: application/json" \
     -d '{"url": "TARGET_URL", "blockImages": true}'
   ```
   Block images by default — scraping is text-focused.

6. **Wait 3 seconds** for page render

7. **Extract data based on task:**

   - **No extraction description:** Return `/text` output (readable page content)
     ```bash
     curl -s "$BASE_URL/text"
     ```

   - **With extraction description:** Use `/text` first, then if more structure needed, use `/snapshot?format=compact` to identify elements and their relationships
     ```bash
     # Start with cheap text extraction
     curl -s "$BASE_URL/text"

     # If needed for structure, add interactive snapshot
     curl -s "$BASE_URL/snapshot?format=compact&selector=main&maxTokens=3000"
     ```

8. **Structure the output** based on what the user asked for. Parse the raw data into the requested format (list, table, key-value pairs, etc.)

9. **Return** structured extraction

## Output

### General Extraction

```
## Scraped: https://docs.example.com/api

**Title:** API Documentation
**URL:** https://docs.example.com/api

### Extracted Data

[Structured data based on user request]

**Source tokens:** ~800 (via /text)
```

### Specific Extraction

```
## Scraped: https://docs.example.com/api

**Extracted:** "list all endpoint names and methods"

| Endpoint | Method | Description |
|----------|--------|-------------|
| /users | GET | List all users |
| /users | POST | Create user |
| /users/:id | GET | Get user by ID |

**Source tokens:** ~800 (via /text)
```

## Errors

### No URL Provided

```
Error: URL is required.

Usage: /scrape <url> [what to extract]

Examples:
  /scrape https://docs.example.com/api
  /scrape https://example.com/pricing "pricing tiers and features"
```

### PinchTab Not Running

```
Warning: PinchTab not running at http://localhost:9867

Start PinchTab:
  pinchtab &
```

## Notes

- Images are blocked by default during scraping to save bandwidth and load time
- Start with `/text` (~800 tokens) before escalating to `/snapshot` (~4,000+ tokens)
- For multi-page scraping, use `/browse` with a task description instead
- The extraction description helps the agent focus on relevant content — be specific
- For authenticated pages, ensure PinchTab has an active session first

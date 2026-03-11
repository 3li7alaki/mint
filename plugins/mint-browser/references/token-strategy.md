# Token Strategy for PinchTab Agents

Concise guide for mint agents on token-efficient browser automation.

## Decision Tree

```
What do you need?
  |
  ├─ Read page content → /text (~800 tokens)
  ├─ Find interactive elements → /snapshot?filter=interactive&format=compact (~3,600 tokens)
  ├─ See what changed → /snapshot?diff=true (varies, often <500 tokens)
  ├─ Full page understanding → /snapshot?format=compact (~4,000-6,000 tokens)
  ├─ Visual verification → /screenshot (~2K vision tokens)
  └─ Full accessibility tree → /snapshot (~10,500 tokens) — last resort
```

## Rules

1. **Start cheap.** Use `/text` when you only need to read. Use `?filter=interactive&format=compact` when you need to act.
2. **Use diff after first snapshot.** On multi-step workflows, add `?diff=true` to subsequent snapshots. You get only what changed.
3. **Scope with selectors.** Add `?selector=main` or `?selector=.content` to ignore nav/footer/sidebar.
4. **Cap output.** Add `?maxTokens=2000` when you need a quick look, not the full tree.
5. **Block images when reading.** Pass `"blockImages": true` on `/navigate` for text-heavy tasks. Saves load time and bandwidth.
6. **Never full snapshot for simple reads.** If you just need to know what text is on the page, `/text` is 13x cheaper than `/snapshot`.

## Combining Parameters

Parameters stack. Use all that apply:

```bash
# Best for interactions: interactive elements, compact format, capped
curl "http://localhost:9867/snapshot?filter=interactive&format=compact&maxTokens=2000"

# Best for reading after action: diff only, compact
curl "http://localhost:9867/snapshot?diff=true&format=compact"

# Best for scoped reading: specific section, compact
curl "http://localhost:9867/snapshot?selector=main&format=compact&maxTokens=3000"
```

## Cost Comparison

| Method | Tokens | Use case |
|--------|--------|----------|
| `/text` | ~800 | Read page content, verify text presence |
| `/snapshot?filter=interactive&format=compact` | ~3,600 | Find buttons/links/inputs to interact with |
| `/snapshot?diff=true&format=compact` | ~200-1,000 | See what changed after an action |
| `/snapshot?format=compact` | ~4,000-6,000 | Understand full page layout |
| `/screenshot` | ~2,000 (vision) | Visual verification, layout checks |
| `/snapshot` (default JSON) | ~10,500 | Full accessibility tree — avoid when possible |

## The 3-Second Rule

Always wait 3 seconds after `/navigate` before taking a snapshot. Chrome needs time to render
the accessibility tree. Without the wait, you get incomplete or empty results.

```bash
curl -X POST http://localhost:9867/navigate \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}' && \
sleep 3 && \
curl "http://localhost:9867/snapshot?filter=interactive&format=compact"
```

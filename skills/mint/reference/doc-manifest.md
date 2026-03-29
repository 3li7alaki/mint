# Doc-Manifest — Reference

Track which documentation depends on which code artifacts.

---

## How It Works

1. Each doc in `.mint/doc-manifest.json` declares sections with `tracks` (glob patterns)
2. When code changes, check if tracked files were modified
3. Documenter reads manifest to know what to update and where

## Staleness Detection

| Strategy | Detects | Best for |
|----------|---------|----------|
| `glob-count` | File count changed | Directory listings, inventories |
| `content-hash` | Contents changed | Config schemas, API refs |
| `git-diff` | Tracked files modified since last doc commit | Narrative docs |

## Location

- Project: `.mint/doc-manifest.json` (committed)
- Template: `templates/doc-manifest.json`

Created during `mint init`. Documenter reads before every update.

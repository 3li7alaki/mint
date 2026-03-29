# Design Intelligence — Reference

Detailed design config and flow. See `modes/design.md` for commands and detection.

---

## Config

```json
{
  "design": {
    "enabled": true,
    "stack": "auto",
    "profile": ".mint/design-profile.json",
    "notes": ".mint/design-notes.md",
    "uiFilePatterns": ["*.tsx", "*.jsx", "*.vue", "*.svelte", "*.css", "*.scss", "*.html"],
    "conventions": [],
    "review": {
      "accessibility": true,
      "consistency": true,
      "performance": true,
      "rtl": false,
      "i18n": false,
      "brand": false
    }
  }
}
```

## Design Context Agent Flow

1. Loads `.mint/design-profile.json` (project's visual DNA)
2. Loads `.mint/design-notes.md` (user's hard rules)
3. Selects relevant reference docs from `standards/design/reference/`
4. Loads anti-patterns and design direction
5. Returns `<design-context>` XML injected into planner

## Installation (during mint init)

1. Optionally install Impeccable skill
2. Auto-detect design assets (components.json, tailwind.config, brand guides)
3. Build initial design profile if UI code exists
4. Configure review checks based on detected features

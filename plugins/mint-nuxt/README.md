# mint-nuxt

Nuxt-specific conventions and review plugin for [mint](https://github.com/3li7alaki/mint).

## Install

```bash
git clone https://github.com/3li7alaki/mint-nuxt .mint/plugins/mint-nuxt
```

Add to `.mint/config.json`:

```json
{
  "plugins": [".mint/plugins/mint-nuxt"]
}
```

## What It Does

Adds a Nuxt reviewer to the stage 2 parallel audit pipeline. Checks:

- **File structure** — pages, components, composables in the right directories
- **Auto-imports** — no manual imports of auto-imported functions
- **Server-only patterns** — server utilities not leaking to client
- **Data fetching** — `useFetch`/`useAsyncData` instead of raw fetch
- **Middleware** — proper patterns and placement
- **Configuration** — runtime vs build-time config, env var prefixes

## Configuration

The plugin adds a `nuxtVersion` config key (default: `4`). Set to `3` for Nuxt 3 projects.

## Hook

- `pre-review` — runs as a stage 2 parallel reviewer alongside quality, security, etc.

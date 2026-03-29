# SSH Mode

Run commands on remote servers. Requires `ssh` config in `.mint/config.json`.

---

## Config

```json
{
  "ssh": {
    "key": "~/.ssh/my-key",
    "environments": {
      "staging": {
        "host": "1.2.3.4",
        "doppler": { "config": "staging", "var": "SERVER_IP" },
        "user": "root",
        "docker": { "container": "my-app-web" }
      }
    }
  }
}
```

## Process

1. **Resolve host** — `.mint/ssh-cache.json` first, then Doppler, then static `host`
2. **Build command** — `ssh -i {key} {user}@{host}` + docker exec if configured
3. **Execute** — run via Bash, return output
4. **On failure** — invalidate cache, re-fetch from Doppler, retry once

## Cache

- File: `.mint/ssh-cache.json` (gitignored)
- Only cache Doppler-fetched hosts
- Invalidate on connection failure

## Examples

| User says | Command |
|-----------|---------|
| "run migrations on staging" | `ssh -i key root@host "docker exec -i container php artisan migrate"` |
| "tail logs on prod" | `ssh -i key root@host "docker exec -i container tail -100 storage/logs/laravel.log"` |

## Notes

- Container names change between deploys — `docker ps --filter` to find current
- Interactive commands (tinker, shell) require a terminal — inform user
- Always expand `~` in key paths

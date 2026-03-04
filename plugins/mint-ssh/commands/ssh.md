# /ssh Command

Connect to a remote server via SSH with Doppler secrets integration.

## Usage

```
/ssh <environment> [--fresh]
```

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `environment` | Yes | Environment name from `ssh.environments` in `.mint/config.json` |
| `--fresh` | No | Force Doppler refresh, ignoring cached host values |

## Examples

```bash
# Connect to staging environment
/ssh staging

# Connect to production with fresh Doppler lookup (ignore cache)
/ssh production --fresh
```

## Process

1. **Load configuration** from `.mint/config.json`
2. **Validate environment** exists in `ssh.environments`
3. **Handle --fresh flag** — if present, invalidate cache before connecting
4. **Invoke ssh-connect agent** to orchestrate the connection

## Configuration

The command reads from `.mint/config.json`:

```json
{
  "ssh": {
    "key": "~/.ssh/my-key",
    "environments": {
      "staging": {
        "host": "192.168.1.100",
        "user": "deploy",
        "docker": {
          "container": "app-web"
        }
      },
      "production": {
        "doppler": {
          "config": "prd",
          "var": "SERVER_IP"
        },
        "user": "root",
        "docker": {
          "container": "prod-app"
        }
      }
    }
  }
}
```

## Errors

### Environment not specified

```
Error: Environment name is required.

Usage: /ssh <environment> [--fresh]

Example: /ssh staging
```

### Environment not found

```
Error: Environment 'invalid-env' not found.

Available environments:
- staging
- production

Usage: /ssh <environment> [--fresh]
```

### No SSH configuration

```
Error: No SSH configuration found in .mint/config.json

Add SSH configuration:
{
  "ssh": {
    "environments": {
      "staging": {
        "host": "192.168.1.100",
        "user": "deploy"
      }
    }
  }
}
```

### No environments configured

```
Error: No environments configured in ssh.environments

Add at least one environment to .mint/config.json:
{
  "ssh": {
    "environments": {
      "staging": {
        "host": "192.168.1.100",
        "user": "deploy"
      }
    }
  }
}
```

## Implementation

When this command is invoked:

1. **Parse arguments:**
   - Extract environment name (first positional argument)
   - Check for `--fresh` flag

2. **Validate environment name:**
   - If missing: show "Environment not specified" error
   - Environment name must not be empty

3. **Load .mint/config.json:**
   - If file doesn't exist: show configuration error
   - If `ssh` key missing: show "No SSH configuration" error
   - If `ssh.environments` empty: show "No environments configured" error

4. **Validate environment exists:**
   - Check `ssh.environments[environment]` exists
   - If not found: list available environments

5. **Handle --fresh flag:**
   - If `--fresh` is present and environment uses Doppler:
     - Invoke `cache-manager.md` with action `invalidate` for this environment
     - This forces a fresh Doppler lookup

6. **Invoke ssh-connect agent:**
   - Pass: environment name, project config
   - Agent handles: host resolution, SSH command building, connection, retries

## Notes

- Environment name is always required — no default environment
- The `--fresh` flag only affects Doppler-fetched hosts (not static hosts)
- For connection details and retry behavior, see `ssh-connect.md` agent

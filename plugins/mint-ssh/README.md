# mint-ssh

SSH access plugin for [mint](https://github.com/3li7alaki/mint). Connect to remote servers with Doppler secrets integration and Docker exec support.

## Install

mint-ssh ships with mint. After installing mint (`npx skills add 3li7alaki/mint`), enable it in `.mint/config.json`:

```json
{
  "plugins": ["plugins/mint-ssh"],
  "ssh": {
    "key": "~/.ssh/my-key",
    "environments": {
      "staging": {
        "host": "1.2.3.4",
        "user": "root",
        "docker": {
          "container": "my-app-web"
        }
      }
    }
  }
}
```

## Requirements

- **SSH**: Standard SSH client (`ssh` command available)
- **Doppler CLI** (optional): Required only if using Doppler for dynamic host resolution

## Configuration

### Top-Level Options

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `ssh.key` | No | System default | Path to SSH private key (e.g., `~/.ssh/id_rsa`) |
| `ssh.environments` | Yes | `{}` | Map of environment names to their configuration |

### Environment Options

Each environment in `ssh.environments` supports:

| Key | Required | Default | Description |
|-----|----------|---------|-------------|
| `host` | No* | — | Static IP or hostname for the server |
| `doppler` | No* | — | Doppler config for dynamic host resolution |
| `doppler.config` | Yes (if doppler) | — | Doppler config name (e.g., `staging`, `production`) |
| `doppler.var` | Yes (if doppler) | — | Doppler secret variable containing the host IP |
| `user` | No | Current user | SSH username |
| `docker` | No | — | Docker configuration for container exec |
| `docker.container` | Yes (if docker) | — | Container name or ID to exec into |

*Either `host` or `doppler` must be provided, but not both.

### Example Configurations

**Simple static host:**

```json
{
  "ssh": {
    "environments": {
      "production": {
        "host": "192.168.1.100",
        "user": "deploy"
      }
    }
  }
}
```

**With Doppler for dynamic host:**

```json
{
  "ssh": {
    "key": "~/.ssh/deploy-key",
    "environments": {
      "staging": {
        "doppler": {
          "config": "staging",
          "var": "SERVER_IP"
        },
        "user": "root",
        "docker": {
          "container": "app-web"
        }
      }
    }
  }
}
```

**Multiple environments:**

```json
{
  "ssh": {
    "key": "~/.ssh/my-key",
    "environments": {
      "staging": {
        "host": "staging.example.com",
        "user": "deploy",
        "docker": {
          "container": "staging-app"
        }
      },
      "production": {
        "doppler": {
          "config": "prd",
          "var": "PROD_SERVER_IP"
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

## Usage

The plugin is command-driven. Use the `/ssh` command to connect:

```
/ssh staging           # Connect to staging environment
/ssh production        # Connect to production environment
```

When Docker is configured, the agent will SSH into the server and exec into the specified container, giving you a shell inside the container.

When Doppler is configured, the agent will fetch the host IP from your Doppler secrets before connecting.

## Testing

These test scenarios verify mint-ssh functionality. Tests require actual SSH targets or a mock SSH server (see [openssh-portable](https://github.com/openssh/openssh-portable) for local testing).

### 1. Static Host Test

**Setup:** Configure with a static host:
```json
{
  "ssh": {
    "environments": {
      "test": {
        "host": "192.168.1.100",
        "user": "testuser"
      }
    }
  }
}
```

**Command:** `/ssh test`

**Expected Output:**
```
Connecting to test environment...
SSH connection established to 192.168.1.100
[Interactive shell session begins]
```

### 2. Doppler Integration Test

**Prerequisites:** Doppler CLI installed and authenticated (`doppler --version`)

**Setup:**
```json
{
  "ssh": {
    "environments": {
      "staging": {
        "doppler": {
          "config": "staging",
          "var": "SERVER_IP"
        },
        "user": "root"
      }
    }
  }
}
```

**Command:** `/ssh staging`

**Expected Output (first call - cache miss):**
```
Fetching host from Doppler (config: staging, var: SERVER_IP)...
Caching Doppler secret...
Connecting to staging environment...
SSH connection established to [resolved-ip]
```

**Command:** `/ssh staging` (second call)

**Expected Output (cache hit):**
```
Using cached Doppler secret for staging...
Connecting to staging environment...
SSH connection established to [resolved-ip]
```

**Skip instructions:** If Doppler CLI is not available, skip this test and use static host configuration instead.

### 3. Cache Invalidation Test

**Prerequisites:** Doppler configured as in test #2

**Command:** `/ssh staging --fresh`

**Expected Output:**
```
Fetching host from Doppler (config: staging, var: SERVER_IP)...
Bypassing cache (--fresh flag)...
Caching Doppler secret...
Connecting to staging environment...
SSH connection established to [resolved-ip]
```

### 4. Docker Exec Test

**Setup:**
```json
{
  "ssh": {
    "environments": {
      "docker-test": {
        "host": "192.168.1.100",
        "user": "root",
        "docker": {
          "container": "my-app-web"
        }
      }
    }
  }
}
```

**Command:** `/ssh docker-test`

**Expected Output:**
```
Connecting to docker-test environment...
SSH connection established to 192.168.1.100
Executing into container: my-app-web
[Container shell session begins]
```

### 5. Error Handling Tests

#### Missing Environment

**Command:** `/ssh nonexistent`

**Expected Output:**
```
Error: Environment 'nonexistent' not found in configuration.
Available environments: staging, production
```

#### Missing Doppler CLI

**Setup:** Doppler configured but CLI not installed

**Command:** `/ssh staging`

**Expected Output:**
```
Error: Doppler CLI not found. Install from https://docs.doppler.com/docs/cli
Alternatively, use static host configuration.
```

#### Invalid Host

**Setup:** Configure with unreachable host

**Command:** `/ssh test`

**Expected Output:**
```
Connecting to test environment...
Error: SSH connection failed - Connection timed out
Host: 10.0.0.1, User: testuser
```

---

**Note:** For automated testing without real SSH targets, consider using:
- Docker container with SSH server for isolated tests
- [mock-ssh-server](https://github.com/carletes/mock-ssh-server) for Python-based mocking
- Local VM or container network for integration tests

## Agents

| Agent | Role |
|-------|------|
| `ssh-connect.md` | Handles SSH connection with optional Doppler lookup and Docker exec |

## Commands

| Command | Description |
|---------|-------------|
| `/ssh <env>` | Connect to the specified environment |

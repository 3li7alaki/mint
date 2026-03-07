# /design:setup Command

Install ui-ux-pro-max and configure project design conventions.

## Usage

```
/design:setup [options]
```

## Options

| Option | Description | Default |
|--------|-------------|---------|
| `--skip-install` | Skip ui-ux-pro-max installation | - |
| `--generate` | Generate design system from scratch | - |
| `--stack <name>` | Specify stack manually | auto-detect |
| `-y, --yes` | Accept defaults, skip prompts | - |

## Examples

```bash
# Interactive setup
/design:setup

# Quick setup with defaults
/design:setup --yes

# Just configure (skill already installed)
/design:setup --skip-install

# Generate new design system
/design:setup --generate

# Specify stack
/design:setup --stack react
```

## Process

1. **Check ui-ux-pro-max installation**
2. **Verify Python 3 availability**
3. **Detect existing design assets**
4. **Configure conventions**
5. **Set up review checks**
6. **Link with mint-shadcn (if present)**
7. **Test and report**

## Output

```
## Setting up mint-ui-ux

### Step 1: ui-ux-pro-max
Checking for skill installation...
✅ Found at .claude/skills/ui-ux-pro-max/

### Step 2: Python
Checking Python 3...
✅ Python 3.11.2

### Step 3: Design Assets
Scanning project...

Found:
  ✅ components.json (shadcn/ui)
  ✅ BRAND_GUIDE.md
  ✅ tailwind.config.js
  ❌ design-system.json

### Step 4: Configuration
Adding to .mint/config.json...
✅ Conventions configured

### Step 5: Review Settings
Enabling checks...
✅ Accessibility, Consistency, Performance, RTL

### Step 6: Integration
Linking with mint-shadcn...
✅ Cross-referenced

### Step 7: Test
Running search test...
✅ Search engine working

─────────────────────────────

Setup complete! Run /design:help for commands.
```

## Implementation

Invokes the `setup-wizard.md` agent with parsed options.

## Notes

- Safe to run multiple times
- Preserves existing conventions
- Auto-detects most settings
- Works with or without shadcn

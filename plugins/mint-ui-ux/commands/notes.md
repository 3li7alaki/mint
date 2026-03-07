# /design:notes Command

Manage design notes — preferences, rules, and decisions.

## Usage

```
/design:notes <action> [args]
```

## Actions

### add

Add a design note.

```bash
/design:notes add "<note>" [--type <type>]
```

**Types:**
- `rule` — Hard constraint that must be followed
- `preference` — Soft preference, can be overridden
- `decision` — Record a design decision with context

**Examples:**
```bash
# Add a hard rule
/design:notes add "never use red for success states" --type rule

# Add a preference
/design:notes add "prefer subtle shadows over borders" --type preference

# Record a decision
/design:notes add "chose Inter over Geist for better Arabic support" --type decision
```

### list

Show all design notes.

```bash
/design:notes list [--type <type>]
```

**Examples:**
```bash
/design:notes list              # All notes
/design:notes list --type rule  # Just rules
```

### remove

Remove a design note.

```bash
/design:notes remove "<note>"
/design:notes remove --number <n>
```

**Examples:**
```bash
/design:notes remove "never use red"
/design:notes remove --number 3
```

### clear

Clear notes of a specific type.

```bash
/design:notes clear --type <type>
```

**Examples:**
```bash
/design:notes clear --type decision  # Clear decision history
```

## File Format

Notes are stored in `.mint/design-notes.md`:

```markdown
# Design Notes

Project design preferences and constraints.

## Hard Rules

Rules that must always be followed. Violations block review.

- Never use red for success states
- Buttons are always rounded-lg, never full-round
- Icons are always 20x20 or 24x24, never other sizes
- No placeholder text as input labels

## Preferences

Design preferences. Can be overridden with good reason.

- Prefer subtle shadows over borders for cards
- Hero sections should have gradient overlays
- Tables always have sticky headers
- Use logical properties (ms-, me-) over directional (ml-, mr-)

## Decisions Made

Design decisions with context for future reference.

- 2024-03-07: Chose Inter over Geist for better Arabic support
- 2024-03-06: Using 4px base grid (not 8px) for tighter control
- 2024-03-05: Cards use shadow-sm not border — cleaner look
```

## How Notes Are Used

### During Planning

The `design-context` agent reads notes and includes them in the context:

```xml
<design-context>
  <notes>
    - Never use red for success states
    - Buttons always rounded-lg
    - Icons 20x20 or 24x24
  </notes>
</design-context>
```

Rules become hard constraints. Preferences guide decisions.

### During Review

The `design-reviewer` agent checks code against rules:

```
### Rule Violations (2 issues)

BLOCKING
  ❌ Alert.tsx:23 — Using red (#ef4444) for success variant
     Rule: Never use red for success states

  ❌ Button.tsx:45 — Using rounded-full
     Rule: Buttons always rounded-lg, never full-round
```

### Natural Language Learning

When you say things like:
- "we always use rounded buttons"
- "never use that color again"
- "our cards should have subtle shadows"

The planner recognizes this as a design preference and adds it to notes.

## Output

### Add Output

```
✅ Added rule: "never use red for success states"

Updated .mint/design-notes.md

Current rules (4):
1. Never use red for success states (new)
2. Buttons are always rounded-lg
3. Icons are always 20x20 or 24x24
4. No placeholder text as input labels
```

### List Output

```
## Design Notes

### Rules (4)
1. Never use red for success states
2. Buttons are always rounded-lg
3. Icons are always 20x20 or 24x24
4. No placeholder text as input labels

### Preferences (3)
1. Prefer subtle shadows over borders
2. Hero sections with gradient overlays
3. Tables with sticky headers

### Decisions (3)
1. 2024-03-07: Inter over Geist for Arabic
2. 2024-03-06: 4px base grid
3. 2024-03-05: Cards use shadow-sm
```

## Implementation

Direct file manipulation on `.mint/design-notes.md`.

Creates file if it doesn't exist.

## Notes

- Rules are blocking — violations fail review
- Preferences are warnings — can be overridden
- Decisions are informational — help understand history
- Notes persist across sessions
- Natural language understood ("we never use X" → rule)

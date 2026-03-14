# /design:notes Command

Manage persistent design rules and preferences.

## Usage

```
/design:notes <action> [options]
```

## Actions

### add

Add a design note (rule, preference, or decision).

```bash
/design:notes add "<note>" [--type rule|preference|decision]
```

**Types:**
- `rule` — Hard constraint, always enforced (BLOCKING in review)
- `preference` — Soft guideline (WARNING in review)
- `decision` — Logged with timestamp for context

**Examples:**
```bash
/design:notes add "Never use red for success states" --type rule
/design:notes add "Prefer subtle shadows over borders for cards" --type preference
/design:notes add "Chose Inter over Geist for better Arabic support" --type decision
```

### list

Show all design notes.

```bash
/design:notes list [--type rule|preference|decision]
```

### remove

Remove a design note.

```bash
/design:notes remove "<note>"
/design:notes remove --number <n>
```

### clear

Clear all notes of a specific type.

```bash
/design:notes clear --type <type>
```

## Implementation

Notes are stored in `.mint/design-notes.md` with three sections: Hard Rules, Preferences, Decisions Made.

The design-context agent loads these during pre-plan. The design-reviewer enforces hard rules as BLOCKING violations and preferences as WARNINGs.

## Notes

- Rules override all other design guidance (profile, references, conventions)
- Preferences influence but don't block
- Decisions include timestamps and context for future reference
- Notes persist across conversations

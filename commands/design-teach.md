# /design:teach Command

One-time setup that gathers design context for your project. Explores the codebase, asks targeted questions, and saves persistent design guidelines.

## Usage

```
/design:teach
```

No arguments — this is an interactive setup flow.

## What It Does

### Step 1: Explore the Codebase

Before asking questions, scans the project to discover:
- README and docs for project purpose and target audience
- Package.json and config files for tech stack and design libraries
- Existing components for current design patterns
- Brand assets (logos, favicons, color values)
- Design tokens and CSS variables
- Style guides or brand documentation

### Step 2: Ask UX-Focused Questions

Only asks about what couldn't be inferred from the codebase:

**Users & Purpose:**
- Who uses this? What's their context?
- What job are they trying to get done?
- What emotions should the interface evoke?

**Brand & Personality:**
- Brand personality in 3 words?
- Reference sites or apps that capture the right feel?
- What should this explicitly NOT look like?

**Aesthetic Preferences:**
- Visual direction? (minimal, bold, elegant, playful, technical, organic)
- Light mode, dark mode, or both?
- Colors that must be used or avoided?

**Accessibility & Inclusion:**
- Specific accessibility requirements?
- Considerations for reduced motion, color blindness?

Skips questions already answered by the codebase scan.

### Step 3: Write Design Context

Synthesizes findings into:
1. Updates `.mint/design-profile.json` with learned identity, mood, constraints
2. Updates `.mint/design-notes.md` with hard rules and preferences from the conversation
3. Optionally updates project CLAUDE.md with a `## Design Context` section

## Implementation

Combines the design-profile agent (codebase analysis) with an interactive questionnaire. Results persist across all future conversations.

## Notes

- Run once per project, re-run to update after major design changes
- Builds on existing profile — doesn't overwrite
- Design context informs all future UI task planning and review
- Adapted from Impeccable's teach-impeccable flow

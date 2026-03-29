# Design Intelligence

Automatic UI/UX awareness. When enabled, UI tasks get design context injected into planning
and design quality checked during review — without the user asking.

---

## Detection

Design context activates when EITHER is true:
1. **Keywords** — task contains: component, page, layout, styling, animation, form, dashboard,
   modal, sidebar, navigation, responsive, loading state, error state, etc.
2. **File patterns** — scope includes files matching `config.design.uiFilePatterns`
   (default: `*.tsx, *.jsx, *.vue, *.svelte, *.css, *.scss, *.html`)

## Context Flow

1. Orchestrator detects UI task (keywords or file patterns)
2. Dispatch `design-context` agent → loads design profile, notes, reference knowledge
3. Returns `<design-context>` XML injected into planner context
4. Planner creates spec with design context baked in
5. During stage 2 review, `design-reviewer` runs alongside other auditors

## Design Review Checks

- AI slop test (always — is this distinguishable from generic AI output?)
- Accessibility (WCAG 2.1 AA — alt text, contrast, focus, semantic HTML)
- Design consistency (tokens, spacing, component reuse)
- Performance (animation, reduced motion, bundle)
- RTL (if enabled — logical properties, directional icons)
- i18n (if enabled — hardcoded strings, inline conditionals)
- Brand (if brand guide configured)

## Reference Knowledge

Vendored in `standards/design/reference/`:
- typography, color-and-contrast, spatial-design, motion-design
- interaction-design, responsive-design, ux-writing

Plus mint's own: rtl.md, i18n.md, anti-patterns.md, design-direction.md

## Commands

- `/design search|system|palette|typography|inspiration`
- `/design:profile build|view|update|diff`
- `/design:notes add|list|remove|clear`
- `/design:review [target] [--check type] [--fix]`
- `/design:tokens export|sync|validate`
- `/design:teach` — one-time setup
- `/design:steer <direction>` — polish, critique, audit, bolder, quieter, etc.

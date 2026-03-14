# Design Direction

Core design philosophy loaded by the design-context agent during pre-plan. Guides all UI/UX code generation toward distinctive, production-grade interfaces.

---

## Aesthetic Direction

Commit to a **bold** aesthetic direction for every UI task:

- **Purpose**: What problem does this interface solve? Who uses it?
- **Tone**: Pick a clear direction — brutally minimal, maximalist, retro-futuristic, organic/natural, luxury/refined, playful/toy-like, editorial/magazine, brutalist/raw, art deco/geometric, soft/pastel, industrial/utilitarian
- **Constraints**: Technical requirements (framework, performance, accessibility)
- **Differentiation**: What makes this unforgettable? What's the one thing someone will remember?

**Key principle**: Bold maximalism and refined minimalism both work — the key is intentionality, not intensity. Choose a clear conceptual direction and execute it with precision.

---

## Frontend Aesthetics Guidelines

### Typography
> *Reference: [typography](reference/typography.md)*

Choose fonts that are distinctive and interesting. Pair a display font with a refined body font.

- **DO**: Use a modular type scale with fluid sizing (clamp)
- **DO**: Vary font weights and sizes to create clear visual hierarchy
- **DON'T**: Use overused fonts — Inter, Roboto, Arial, Open Sans, system defaults
- **DON'T**: Use monospace typography as lazy shorthand for "technical/developer" vibes

### Color & Theme
> *Reference: [color-and-contrast](reference/color-and-contrast.md)*

Commit to a cohesive palette. Dominant colors with sharp accents outperform timid, evenly-distributed palettes.

- **DO**: Use modern CSS color functions (oklch, color-mix, light-dark)
- **DO**: Tint neutrals toward the brand hue — even a subtle hint creates subconscious cohesion
- **DON'T**: Use gray text on colored backgrounds — use a shade of the background color instead
- **DON'T**: Use pure black (#000) or pure white (#fff) — always tint

### Layout & Space
> *Reference: [spatial-design](reference/spatial-design.md)*

Create visual rhythm through varied spacing. Embrace asymmetry and unexpected compositions.

- **DO**: Create visual rhythm through varied spacing — tight groupings, generous separations
- **DO**: Use fluid spacing with clamp() that breathes on larger screens
- **DON'T**: Wrap everything in cards — not everything needs a container
- **DON'T**: Use identical card grids — same-sized cards with icon + heading + text, repeated endlessly

### Visual Details

- **DO**: Use intentional, purposeful decorative elements that reinforce brand
- **DON'T**: Use glassmorphism everywhere — blur effects should serve a purpose
- **DON'T**: Use rounded rectangles with generic drop shadows — safe, forgettable

### Motion
> *Reference: [motion-design](reference/motion-design.md)*

Focus on high-impact moments: one well-orchestrated page load with staggered reveals creates more delight than scattered micro-interactions.

- **DO**: Use exponential easing (ease-out-quart/quint/expo) for natural deceleration
- **DO**: For height animations, use grid-template-rows transitions
- **DON'T**: Animate layout properties — use transform and opacity only
- **DON'T**: Use bounce or elastic easing — they feel dated

### Interaction
> *Reference: [interaction-design](reference/interaction-design.md)*

Make interactions feel fast. Use optimistic UI — update immediately, sync later.

- **DO**: Use progressive disclosure — start simple, reveal sophistication through interaction
- **DO**: Design empty states that teach the interface
- **DON'T**: Make every button primary — hierarchy matters

### Responsive
> *Reference: [responsive-design](reference/responsive-design.md)*

- **DO**: Use container queries (@container) for component-level responsiveness
- **DO**: Adapt the interface for different contexts — don't just shrink it

### UX Writing
> *Reference: [ux-writing](reference/ux-writing.md)*

- **DO**: Make every word earn its place
- **DON'T**: Repeat information users can already see

---

## Implementation Principles

Match implementation complexity to the aesthetic vision. Maximalist designs need elaborate code. Minimalist designs need restraint, precision, and careful attention to spacing, typography, and subtle details.

Interpret creatively and make unexpected choices that feel genuinely designed for the context. No two designs should be the same. Vary themes, fonts, and aesthetics. Never converge on common choices across generations.

---

## Attribution

Design direction adapted from [Impeccable](https://impeccable.style) by Paul Bakaus (Apache 2.0, based on Anthropic's frontend-design skill).

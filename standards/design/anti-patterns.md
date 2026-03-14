# Design Anti-Patterns & AI Slop Detection

Reference for the design reviewer. Violations are categorized by severity.

---

## The AI Slop Test

**Critical quality check**: If you showed this interface to someone and said "AI made this," would they believe you immediately? If yes, that's the problem.

A distinctive interface should make someone ask "how was this made?" not "which AI made this?"

---

## BLOCKING — AI Fingerprints (2024–2025)

These patterns are the telltale signs of AI-generated interfaces. Their presence means the design lacks intentionality.

### Color
- **Purple-to-blue gradients** — the default AI palette
- **Cyan-on-dark** — AI's go-to "tech" look
- **Neon accents on dark backgrounds** — looks "cool" without design decisions
- **Gradient text for "impact"** — decorative, not meaningful
- **Dark mode with glowing accents** as default — avoids real design choices
- **Pure black (#000) or pure white (#fff)** — always tint; pure black/white never appears in nature
- **Gray text on colored backgrounds** — looks washed out; use a shade of the background color or transparency

### Layout
- **Identical card grids** — same-sized cards with icon + heading + text, repeated endlessly
- **Hero metric layout template** — big number, small label, supporting stats, gradient accent
- **Everything centered** — left-aligned text with asymmetric layouts feels more designed
- **Cards inside cards** — visual noise, flatten the hierarchy
- **Everything wrapped in cards** — not everything needs a container

### Typography
- **Overused fonts** — Inter, Roboto, Arial, Open Sans, system defaults without intention
- **Monospace as lazy "technical" shorthand** — doesn't add value
- **Large icons with rounded corners above every heading** — templated, adds no value

### Visual Effects
- **Glassmorphism everywhere** — blur effects, glass cards, glow borders used decoratively
- **Rounded elements with thick colored border on one side** — lazy accent
- **Sparklines as decoration** — tiny charts that look sophisticated but convey nothing
- **Rounded rectangles with generic drop shadows** — safe, forgettable, could be any AI output
- **Bounce or elastic easing** — dated and tacky; real objects decelerate smoothly

### Interaction
- **Every button is primary** — ghost buttons, text links, secondary styles create hierarchy
- **Modals for everything** — modals are lazy; use inline, drawers, or progressive disclosure
- **Placeholder text as labels** — placeholders disappear on input; always use visible labels
- **Redundant copy** — headers that restate the heading, intros that repeat what's visible

---

## WARNING — Common Design Mistakes

These aren't AI-specific but indicate poor design decisions.

### Accessibility
- Missing focus indicators (never `outline: none` without replacement)
- Color contrast below WCAG AA (4.5:1 for text, 3:1 for UI components)
- Touch targets smaller than 44x44px
- Missing alt text on meaningful images
- Form inputs without visible labels
- Non-semantic HTML for interactive elements (div with onClick)
- Using `h-screen` / `min-h-screen` instead of `h-dvh` / `min-h-dvh`

### Color
- Hardcoded hex/rgb values not using design tokens
- Using wrong semantic color (success color for non-success states)
- Relying on color alone to convey information
- Alpha/transparency as substitute for a complete palette

### Typography
- Font family not in design system
- Font size not following type scale
- Body text line length outside 45–75 characters
- Missing font loading strategy (no `font-display: swap`)

### Spacing
- Inconsistent spacing (random px values, not on scale)
- Same padding everywhere (no visual rhythm)
- No fluid spacing with clamp() for larger screens

### Motion
- Animating layout properties (width, height, padding, margin) instead of transform/opacity
- Missing `prefers-reduced-motion` support
- Using `ease` or `linear` instead of exponential easing (ease-out-quart/quint/expo)
- Exit animations same speed as entrance (exits should be ~75% of enter duration)

### Responsive
- Fixed widths that break on mobile
- Hiding critical functionality on mobile
- Not using container queries for component-level responsiveness

### Components
- Reinventing a component that already exists in the design system (shadcn, etc.)
- Inconsistent component variants across the app

---

## INFO — Design Refinement Opportunities

- Dynamic Tailwind classes (breaks purging)
- Large inline SVGs (consider sprites or icon components)
- Missing dark mode variants
- Animation without clear purpose
- Using Inter/Roboto when brand fonts are defined

---

## Attribution

Anti-pattern detection merges knowledge from:
- [Impeccable](https://impeccable.style) by Paul Bakaus (Apache 2.0, based on Anthropic's frontend-design skill)
- mint-ui-ux design intelligence

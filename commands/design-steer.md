# /design:steer Command

Steering commands that adjust design direction for a specific task or component. Each direction applies a focused design lens.

## Usage

```
/design:steer <direction> [target]
```

## Directions

### polish
Final quality pass before shipping. Fixes alignment, spacing, consistency, and detail issues.
- Checks: pixel-perfect alignment, consistent spacing, typography hierarchy, interaction states, transitions, content/copy, icons, forms, edge cases, responsiveness
- **Use when**: Feature is functionally complete and needs a final refinement pass

### critique
UX design review — evaluates visual hierarchy, information architecture, emotional resonance.
- Starts with AI slop detection, then evaluates hierarchy, IA, emotional resonance, discoverability, composition, typography, color purpose, states, microcopy
- **Use when**: You want honest design feedback before shipping

### audit
Technical quality audit — accessibility, performance, theming, responsive design.
- Checks: WCAG compliance, performance issues, theming consistency, responsive behavior, anti-patterns
- Generates a severity-rated report with recommended fixes
- **Use when**: You want a comprehensive quality report

### bolder
Amplify visual impact in designs that are too safe or generic.
- Increases: typography contrast (3-5x scale jumps), color saturation, spatial drama, visual effects
- Creates a single focal point and personality direction
- **Use when**: Design feels generic, safe, or forgettable

### quieter
Tone down overly bold designs — create refined aesthetics.
- Reduces: saturation (70-85%), font weights/sizes, decorative elements, animation intensity
- Maintains functionality and distinctiveness
- **Use when**: Design feels overwhelming, noisy, or visually fatiguing

### distill
Remove unnecessary complexity — reveal essence through simplification.
- Finds the 20% delivering 80% value
- Simplifies: IA, visual clutter, layout, interactions, content, code
- **Use when**: Interface feels bloated, confusing, or over-designed

### colorize
Add strategic color to monochromatic or under-colored designs.
- Introduces 2-4 colors max with 60/30/10 dominance rule
- Uses OKLCH for perceptually uniform scales
- **Use when**: Design is too gray, monochromatic, or lacking visual energy

### animate
Add purposeful motion and micro-interactions.
- Plans: hero moment + feedback layer + transition layer + delight layer
- Uses 100/300/500 timing rule, exponential easing
- **Use when**: Interface feels static, needs feedback or delight moments

### delight
Add unexpected touches that make the experience memorable.
- Targets: success states, empty states, loading, achievements, easter eggs
- Implements via micro-interactions, personality in copy, celebrations
- **Use when**: Interface is functional but forgettable

### clarify
Improve unclear interface text — errors, labels, CTAs, tooltips.
- Fixes: jargon, ambiguity, passive voice, missing context
- Applies to: error messages, form labels, button text, empty states
- **Use when**: Users are confused by interface copy

### harden
Strengthen against edge cases, errors, i18n, and real-world usage.
- Tests: extreme inputs, error scenarios, i18n (text expansion, RTL, charsets), empty/loading states
- Implements: overflow handling, error recovery, validation
- **Use when**: Interface works for happy path but breaks with real data

### adapt
Adapt designs for different screen sizes, devices, or platforms.
- Creates context-specific strategies for mobile, tablet, desktop, print, email
- Uses breakpoints, container queries, touch adaptations
- **Use when**: Design works on one screen size but needs to work everywhere

### normalize
Align features to design system standards and established patterns.
- Replaces one-off components with design system equivalents
- Tokenizes hardcoded values, aligns UX patterns
- **Use when**: Feature diverges from the rest of the app's design language

### extract
Extract reusable components, tokens, and patterns into the design system.
- Identifies 3+ repeated patterns, hardcoded values, inconsistent variations
- Builds components with proper API, variants, accessibility, docs
- **Use when**: Codebase has repeated patterns that should be systematized

### optimize
Improve loading speed, rendering, animations, and bundle size.
- Targets: images, JS bundles, CSS, fonts, rendering, animations, network
- Focuses on Core Web Vitals (LCP < 2.5s, INP < 200ms, CLS < 0.1)
- **Use when**: Interface feels slow or scores poorly on performance metrics

### onboard
Create or improve first-time user experience.
- Designs: welcome screens, feature discovery, guided tours, empty states
- Progressive onboarding — don't overwhelm upfront
- **Use when**: New users struggle to understand or get started

## Examples

```bash
/design:steer polish                           # Polish everything
/design:steer polish src/components/Header.tsx # Polish specific component
/design:steer critique                          # Get design feedback
/design:steer bolder src/pages/landing/        # Make landing page bolder
/design:steer distill src/pages/dashboard/     # Simplify dashboard
/design:steer harden src/components/Form.tsx   # Harden form against edge cases
```

## Implementation

Each direction loads the relevant reference docs from `standards/design/reference/`, applies the project's design profile and notes, then executes the specific design lens.

The AI slop test runs automatically with `critique`, `audit`, and `polish` directions.

## Notes

- Steering commands modify code — they're action commands, not just analysis
- Always loads project design profile for context-aware changes
- Respects design notes hard rules
- Can be combined: steer bold first, then polish
- Adapted from Impeccable's steering command system

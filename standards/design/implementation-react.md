# React / Next.js / Tailwind Implementation Rules

Framework-specific enforcement rules. **Only loaded when the project stack is React or Next.js.** These rules supplement the core design references — they don't replace them.

---

## 1. Server Component Safety

**BLOCKING:**
- Default to React Server Components (RSC). Only add `"use client"` when the component genuinely needs interactivity, browser APIs, or hooks.
- Isolate interactivity in **leaf** client components. Keep parent layouts and data-fetching components as server components.
- Wrap context providers (`ThemeProvider`, `QueryClientProvider`, etc.) in a dedicated `"use client"` wrapper component — don't force the entire layout tree into client mode.
- Global state (Zustand, Jotai, Redux) lives exclusively in client components. Server components read from props, params, or server-side data sources.

**WARNING:**
- If a component only uses `useState` for a toggle, consider replacing with CSS (`:target`, `details/summary`, checkbox hack) to keep it a server component.

---

## 2. Framer Motion Best Practices

**BLOCKING:**
- Use `useMotionValue` + `useTransform` for continuous animations (mouse tracking, scroll progress, drag position). **Never use `useState`** for values that update every frame — it causes a full re-render per frame.
- Use `<AnimatePresence>` for mount/unmount transitions. Without it, exit animations are skipped.
- `staggerChildren` requires both the parent motion container and child motion elements to be in the **same client component tree**. If children are server components or from a different file boundary, stagger won't work.

**WARNING:**
- Memoize perpetual/infinite loop animations (`React.memo`) and isolate them in leaf components. An infinitely-animating parent re-renders its entire subtree.
- Prefer `layout` and `layoutId` props for position/size animations instead of manually animating `x`, `y`, `width`, `height`.
- Use `spring` type for interactive animations (drag, gestures) and `tween` for decorative animations (entrance, exit).

---

## 3. GSAP / Three.js Boundaries

**BLOCKING:**
- Never mix GSAP and Framer Motion in the same component tree. They fight over the same DOM properties and create unpredictable results.
- Use GSAP exclusively for isolated full-page scroll-telling or canvas-based backgrounds — not for standard UI components.
- Wrap GSAP timelines and Three.js scenes in strict `useEffect` cleanup:

```tsx
useEffect(() => {
  const tl = gsap.timeline();
  tl.to(ref.current, { ... });
  return () => tl.kill(); // Always clean up
}, []);
```

**WARNING:**
- Three.js renders belong in a dedicated canvas component, not mixed into the DOM component tree.
- Consider `@react-three/fiber` and `@react-three/drei` over raw Three.js for better React integration and cleanup.

---

## 4. Tailwind Guards

**BLOCKING:**
- Before using any Tailwind syntax, check `package.json` for the installed version. **v3 and v4 have different configurations:**
  - v3: `tailwindcss` plugin in PostCSS, `tailwind.config.js` file, `@tailwind base/components/utilities` directives
  - v4: `@tailwindcss/postcss` plugin, CSS-first config with `@theme`, `@import "tailwindcss"` directive
- Never use v4 syntax (`@theme`, `@import "tailwindcss"`) in a v3 project or vice versa.

**WARNING:**
- Dynamic class names (template literals, string concatenation) break Tailwind's purging. Use complete class strings or `clsx`/`cn` with static alternatives.
- When adding new Tailwind plugins, verify compatibility with the installed major version.

---

## 5. Performance Guardrails

**BLOCKING:**
- Animate only `transform` and `opacity`. Never animate `top`, `left`, `width`, `height`, `padding`, or `margin` — they trigger layout recalculation every frame.
- Apply grain/noise textures to **fixed, `pointer-events-none` pseudo-elements only**. Never apply them to scrolling content or interactive elements.
- z-index is for systemic layers only: navigation, modals, overlays, tooltips. Do not use z-index for visual stacking within a component — use DOM order or `isolation: isolate`.

**WARNING:**
- Never animate `box-shadow` directly. Use a pseudo-element with the target shadow and animate its `opacity` instead.
- Images in scroll-animated sections should use `loading="lazy"` and explicit `width`/`height` to prevent layout shift.
- Avoid `will-change` as a permanent property. Apply it just before animation starts, remove after.

---

## 6. Viewport Safety

**BLOCKING:**
- Use `min-h-[100dvh]` (dynamic viewport height), not `h-screen` or `min-h-screen`. Mobile browsers have variable toolbar heights that `vh` doesn't account for.
- Use CSS Grid over flexbox calc math for page-level layout. `grid-template-rows: auto 1fr auto` is cleaner and more maintainable than `calc(100vh - header - footer)`.

**WARNING:**
- Constrain page width: `max-w-[1400px] mx-auto` or `max-w-7xl` for content containment. Full-bleed sections should still have padded inner containers.
- Test with browser dev tools device emulation — check both the smallest (320px) and largest (2560px) reasonable viewports.

---

## 7. Icon Policy

**WARNING:**
- Use a consistent icon library: `@phosphor-icons/react`, `@radix-ui/react-icons`, or `lucide-react`. Do not mix icon libraries within a project.
- Standardize `strokeWidth` globally (typically 1.5 or 2) — inconsistent stroke weights look unpolished.

**BLOCKING:**
- No emojis in UI code. Use icon components or clean SVGs instead. Emojis render differently across platforms, can't be styled, and look unprofessional in application interfaces.

---

## 8. shadcn/ui

**WARNING:**
- Never use shadcn components in their default state. After installation, customize:
  - Border radius to match project design tokens
  - Colors to match the project palette (not the shadcn defaults)
  - Shadows to match the project's shadow system
  - Font sizes and weights to match the type scale
- The `cn()` utility from shadcn is fine to adopt — it's just `clsx` + `tailwind-merge`.

**INFO:**
- Check `components.json` for the configured style (`default` vs `new-york`) and base color before adding new components.
- Prefer shadcn components over custom implementations for standard patterns (Dialog, Dropdown, Tooltip, etc.).

---

## 9. Dependency Verification

**BLOCKING:**
- Before importing any third-party library, check `package.json` (both `dependencies` and `devDependencies`).
- If the package is not installed, output the install command (`bun add <package>` or `npm install <package>`) and note it as a required step — do not silently import missing packages.

**WARNING:**
- Check for version compatibility. Major version mismatches between related packages (e.g., `framer-motion` v10 vs v11, `@radix-ui` v0 vs v1) cause subtle runtime errors.
- Prefer packages already in the dependency tree over adding new ones for similar functionality.

---

## Attribution

Framework-specific rules informed by [taste-skill](https://github.com/Leonxlnx/taste-skill) by Leon Lin.

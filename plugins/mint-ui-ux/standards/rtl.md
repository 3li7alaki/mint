# RTL (Right-to-Left) Standards

Standards for RTL-compatible UI code. Violations are blocking in design review.

## Why RTL Matters

RTL languages (Arabic, Hebrew, Persian, Urdu) read right-to-left. UI must mirror horizontally — what's on the left in LTR appears on the right in RTL.

Using directional CSS breaks RTL layouts. Logical properties adapt automatically.

## Logical Properties Reference

### Margin and Padding

| ❌ Never Use | ✅ Always Use | Meaning |
|--------------|---------------|---------|
| `ml-*` | `ms-*` | margin-inline-start |
| `mr-*` | `me-*` | margin-inline-end |
| `pl-*` | `ps-*` | padding-inline-start |
| `pr-*` | `pe-*` | padding-inline-end |

### Position

| ❌ Never Use | ✅ Always Use | Meaning |
|--------------|---------------|---------|
| `left-*` | `start-*` | inset-inline-start |
| `right-*` | `end-*` | inset-inline-end |
| `inset-x-0` (with left/right) | `inset-inline-0` | both inline edges |

### Text Alignment

| ❌ Never Use | ✅ Always Use |
|--------------|---------------|
| `text-left` | `text-start` |
| `text-right` | `text-end` |

### Borders

| ❌ Never Use | ✅ Always Use |
|--------------|---------------|
| `border-l-*` | `border-s-*` |
| `border-r-*` | `border-e-*` |
| `rounded-l-*` | `rounded-s-*` |
| `rounded-r-*` | `rounded-e-*` |
| `rounded-tl-*` | `rounded-ts-*` |
| `rounded-tr-*` | `rounded-te-*` |
| `rounded-bl-*` | `rounded-bs-*` |
| `rounded-br-*` | `rounded-be-*` |

### Flexbox and Grid

| ❌ Never Use | ✅ Always Use |
|--------------|---------------|
| `justify-start` with `flex-row` | Works correctly (no change needed) |
| `space-x-*` | `gap-*` (preferred) or `space-x-*` (RTL-safe in Tailwind 3+) |

### Scroll

| ❌ Never Use | ✅ Always Use |
|--------------|---------------|
| `scroll-pl-*` | `scroll-ps-*` |
| `scroll-pr-*` | `scroll-pe-*` |
| `scroll-ml-*` | `scroll-ms-*` |
| `scroll-mr-*` | `scroll-me-*` |

## Icons

Directional icons must flip in RTL:

```tsx
// Icons that need RTL flip
<ChevronRight className="rtl:rotate-180" />
<ArrowLeft className="rtl:rotate-180" />
<ArrowRight className="rtl:rotate-180" />

// Icons that should NOT flip
<Check />        // Universal
<X />            // Universal
<Plus />         // Universal
<ChevronDown />  // Vertical, no flip
<ChevronUp />    // Vertical, no flip
```

## CSS Custom Properties

When writing custom CSS:

```css
/* ❌ BAD */
.sidebar {
  left: 0;
  padding-left: 1rem;
  margin-right: 2rem;
}

/* ✅ GOOD */
.sidebar {
  inset-inline-start: 0;
  padding-inline-start: 1rem;
  margin-inline-end: 2rem;
}
```

## Review Severity

| Violation | Severity |
|-----------|----------|
| Using `ml-*`, `mr-*`, `pl-*`, `pr-*` | **BLOCKING** |
| Using `left-*`, `right-*` for positioning | **BLOCKING** |
| Using `text-left`, `text-right` | **WARNING** (sometimes intentional) |
| Using `border-l-*`, `border-r-*` | **BLOCKING** |
| Using `rounded-l-*`, `rounded-r-*` | **BLOCKING** |
| Directional icon without RTL flip | **WARNING** |
| Custom CSS with `left`/`right`/`padding-left` | **BLOCKING** |

## Testing RTL

1. Add `dir="rtl"` to `<html>` element
2. Verify layout mirrors correctly
3. Check text alignment
4. Verify icons flip appropriately
5. Test navigation flows

## Framework-Specific

### React/Next.js

```tsx
// Use logical properties in inline styles too
<div style={{ paddingInlineStart: '1rem' }} />

// Not
<div style={{ paddingLeft: '1rem' }} />
```

### Vue/Nuxt

```vue
<template>
  <!-- Tailwind logical properties work the same -->
  <div class="ps-4 me-2 text-start" />
</template>
```

## Adding shadcn Components

New shadcn components often use directional properties. Always audit:

```bash
# Search for violations in new component
grep -E "ml-|mr-|pl-|pr-|left-|right-|text-left|text-right|border-l|border-r|rounded-l|rounded-r" ComponentName.tsx
```

Replace all matches with logical equivalents before using the component.

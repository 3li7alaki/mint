# Creative UI Patterns

A catalog of advanced UI patterns for building distinctive interfaces. Each pattern describes the visual effect, when it works well, and how it's typically implemented. Framework-agnostic — specific library choices depend on the project stack.

---

## Navigation & Menus

**Mac OS Dock Magnification** — Icons scale up as the cursor approaches, with neighbors scaling proportionally to create a smooth magnification wave. Good for icon-heavy navigation bars where discoverability matters. Implemented with cursor distance calculations driving scale transforms; libraries like `react-dock` or custom pointer tracking.

**Magnetic Button** — Button content subtly shifts toward the cursor when hovering nearby, creating a gravitational pull effect. Good for CTAs and hero buttons where you want to draw interaction. Implemented with pointer position tracking relative to the button center, applying translate transforms.

**Gooey Menu** — Menu items merge and separate with a viscous, liquid-like blob effect using SVG filters. Good for playful or creative brand contexts where standard menus feel too rigid. Implemented with SVG `feGaussianBlur` + `feColorMatrix` filters applied to a group of elements.

**Dynamic Island** — A fixed notification area that morphs shape and content contextually (expanding for alerts, shrinking for passive status). Good for status-heavy apps that need persistent but non-intrusive feedback. Implemented with layout animations between size states, content swap on state change.

**Contextual Radial Menu** — A circular menu that appears around the cursor on right-click or long-press, placing options equidistant from the trigger point. Good for power-user tools, canvas editors, or spatial interfaces. Implemented with polar coordinate positioning, trigonometric item placement.

**Floating Speed Dial** — A FAB that expands into multiple action buttons in an arc or column on activation. Good for mobile-first interfaces with 3-5 frequent actions. Implemented with staggered scale/translate animations from a single origin point.

**Mega Menu Reveal** — Full-width dropdown panels that reveal rich content (images, nested links, featured items) with a coordinated entrance animation. Good for e-commerce or content-heavy sites with deep navigation hierarchies. Implemented with CSS Grid for layout, staggered opacity/translate for reveal, pointer-safe hover zones.

---

## Layout & Grids

**Bento Grid** — An asymmetric grid with varied cell sizes (1x1, 2x1, 1x2, 2x2) creating a visually dynamic dashboard-like layout. Good for feature showcases, dashboards, and landing page sections. Implemented with CSS Grid using `grid-template-areas` or `span` directives, responsive breakpoint adjustments.

**Masonry Layout** — Items of varying heights packed without gaps, like a brick wall. Good for image galleries, portfolios, and content feeds with heterogeneous card heights. Implemented with CSS `columns`, JavaScript layout libraries (Masonry.js, Isotope), or CSS Grid with `masonry` value (experimental).

**Chroma Grid** — A grid where cells have distinct background colors or gradients, creating a mosaic-like chromatic pattern. Good for portfolio showcases and brand-heavy landing pages. Implemented with CSS Grid and per-cell color assignments, optionally with hover-triggered color shifts.

**Split Screen Scroll** — Two (or more) vertical panels that scroll independently or at different rates. Good for comparison views, storytelling with text+media pairing, or before/after presentations. Implemented with independent scroll containers, or synchronized scroll with offset multipliers.

**Curtain Reveal** — Content panels slide apart (like curtains opening) to reveal content beneath on scroll or interaction. Good for dramatic section transitions and product reveals. Implemented with scroll-driven translate transforms on left/right panels, Intersection Observer for trigger timing.

---

## Cards & Containers

**Parallax Tilt Card** — Card rotates in 3D following cursor position, with layered elements inside moving at different depths. Good for featured product cards, testimonials, or interactive showcases. Implemented with CSS `perspective` and `rotateX`/`rotateY` driven by pointer coordinates; layers use `translateZ` for depth.

**Spotlight Border Card** — Card border or glow effect follows the cursor position, creating a spotlight that traces the card edge. Good for pricing cards, feature highlights, or premium product displays. Implemented with a radial gradient on a pseudo-element positioned at cursor coordinates relative to the card.

**Glassmorphism Panel** — Frosted glass effect with backdrop blur, subtle border, and transparency. Good for overlay content, floating toolbars, and modal alternatives where context visibility matters. Implemented with `backdrop-filter: blur()`, semi-transparent background, subtle border with transparency. Use sparingly (see anti-patterns).

**Holographic Foil Card** — Iridescent color-shifting effect that changes based on cursor position or device orientation, mimicking holographic material. Good for collectible items, NFT displays, premium membership cards. Implemented with multi-layer gradients using `mix-blend-mode`, rotation driven by pointer or device gyroscope.

**Tinder Swipe Stack** — Stacked cards that can be swiped left/right with physics-based gestures, revealing the next card beneath. Good for decision-making UIs (approve/reject), content browsing, and onboarding flows. Implemented with gesture detection (drag + velocity), spring-based throw animations, card stack z-index management.

**Morphing Modal** — An element (card, button, image) smoothly transforms into a modal or expanded view, maintaining visual continuity. Good for detail views, image lightboxes, and any drill-down interaction. Implemented with shared-element transitions (FLIP technique), coordinated scale/position/border-radius animations.

---

## Scroll Animations

**Sticky Scroll Stack** — Sections stick to the top of the viewport and stack on top of each other as the user scrolls, creating a layered card deck effect. Good for feature tours, case study presentations, and storytelling sequences. Implemented with `position: sticky` with incrementing `top` values, optional scale-down on stacked cards.

**Horizontal Scroll Hijack** — Vertical scroll input drives horizontal movement of a content track, creating a side-scrolling experience within a vertical page. Good for timelines, project showcases, and portfolio case studies. Implemented with scroll position mapped to horizontal translate, pinned container with `overflow: hidden`.

**Locomotive Scroll Sequence** — Smooth scroll with inertia and parallax layers moving at different speeds, creating depth and cinematic pacing. Good for editorial sites, brand storytelling, and immersive landing pages. Implemented with smooth scroll libraries (Lenis, Locomotive Scroll), or native CSS `scroll-behavior: smooth` with IntersectionObserver-triggered parallax.

**Zoom Parallax** — Elements scale up or down as the user scrolls, creating a dolly zoom or depth-of-field effect. Good for hero sections, product reveals, and section transitions that need dramatic impact. Implemented with scroll position mapped to scale transforms, optionally combined with opacity for fade-through.

**Scroll Progress Path** — An SVG path or line that draws itself as the user scrolls, visualizing reading or journey progress. Good for long-form content, step-by-step guides, and visual storytelling. Implemented with SVG `stroke-dasharray` and `stroke-dashoffset` driven by scroll percentage.

**Liquid Swipe Transition** — Section transitions that use a fluid, wave-like boundary between old and new content as user scrolls or swipes. Good for full-page section transitions and creative portfolio sites. Implemented with SVG clip-paths or CSS `clip-path` with animated bezier curves, scroll-driven progression.

---

## Galleries & Media

**Dome Gallery** — Images arranged on a curved surface (cylindrical or spherical), rotating on drag or scroll to browse. Good for immersive portfolio displays and product image collections. Implemented with CSS 3D transforms (`rotateY` + `translateZ`) on a perspective container, drag velocity maps to rotation speed.

**Coverflow Carousel** — Center item displayed prominently with adjacent items receding in perspective (scaled down, rotated, dimmed). Good for album art, product catalogs, and featured content sliders. Implemented with 3D transforms (perspective, rotateY, scale) applied based on item distance from center; Swiper.js has a built-in coverflow effect.

**Drag-to-Pan Grid** — A grid larger than the viewport that the user navigates by dragging, like panning a map. Good for mood boards, spatial navigation, and canvas-style interfaces. Implemented with drag gesture tracking (pointer events), translate transforms on a container, optional momentum/inertia.

**Accordion Image Slider** — Vertical or horizontal strips that expand on hover/click to reveal the full image while collapsing neighbors. Good for portfolio showcases and comparison galleries with limited space. Implemented with CSS `flex-grow` transitions or `grid-template-columns`/`rows` animations.

**Hover Image Trail** — Images appear and fade along the cursor path as it moves over a link or area, leaving a trail of visuals. Good for portfolio links, creative agency sites, and editorial navigation. Implemented with a pool of pre-loaded image elements positioned at cursor coordinates with staggered opacity/scale fadeout.

**Glitch Effect Image** — Image distorts with RGB channel splitting, scan lines, or displacement on hover or transition. Good for tech/creative brand contexts and editorial accents. Use sparingly. Implemented with CSS `mix-blend-mode`, duplicated layers with `clip-path` slicing, or canvas pixel manipulation.

---

## Typography & Text

**Kinetic Marquee** — Continuous horizontal scrolling text, often with alternating directions across rows, sometimes reacting to scroll speed. Good for brand statements, client logos, and decorative section dividers. Implemented with CSS `@keyframes` translating a duplicated text strip, or scroll-velocity-linked speed.

**Text Mask Reveal** — Text serves as a mask through which an image, video, or gradient is visible. Good for hero headlines, section titles, and visual impact moments. Implemented with `background-clip: text` / `-webkit-background-clip: text` with transparent text color, or SVG text as clip-path.

**Text Scramble Effect** — Characters cycle through random glyphs before resolving to the final text, like a digital decoder. Good for loading states, hero text reveals, and tech-oriented interfaces. Implemented with interval-based character replacement cycling through a character set until settling on target characters.

**Circular Text Path** — Text flows along a circular or curved SVG path, often rotating continuously. Good for badges, decorative labels, and playful UI accents. Implemented with SVG `<textPath>` on a `<path>` element, optional CSS rotation animation on the container.

**Gradient Stroke Animation** — Text outlines (strokes) with animated gradient fills that shift color over time. Good for hero text, headings, and brand-forward typography moments. Implemented with SVG text `stroke` and animated `stroke-dashoffset`, or CSS `@property` for animatable gradient positions.

**Kinetic Typography Grid** — Words or phrases arranged in a grid that animate independently (scale, rotate, translate) triggered by scroll or time. Good for artistic/editorial landing pages and brand manifesto sections. Implemented with grid layout, per-cell animation with staggered delays, scroll-triggered activation via IntersectionObserver.

---

## Micro-Interactions

**Particle Explosion Button** — On click, the button emits a burst of particles (confetti, sparks, geometric shapes) before completing the action. Good for celebration moments (purchase complete, achievement unlocked, form submitted). Implemented with canvas or DOM particle systems, randomized velocity/angle/color per particle, gravity decay.

**Liquid Pull-to-Refresh** — Pull gesture stretches a liquid-like blob that snaps back when released, triggering refresh. Good for mobile-first list views and feed interfaces. Implemented with SVG path deformation driven by drag distance, spring animation on release.

**Skeleton Shimmer** — Loading placeholder shapes with a diagonal light sweep animation, showing the expected layout structure. Good for any content that loads asynchronously (cards, lists, profiles, feeds). Implemented with a linear gradient background animated via `background-position`, matching the content layout shape.

**Directional Hover Aware Button** — Button detects which edge the cursor enters/exits from and animates a fill or highlight from that direction. Good for navigation items, grid cells, and menu links. Implemented with pointer position relative to element center to determine quadrant, directional translate/clip animation.

**Ripple Click Effect** — A circular wave expands from the click point outward, fading as it grows (Material Design style). Good for buttons, list items, and any tappable surface that needs tactile feedback. Implemented with a pseudo-element or span positioned at click coordinates, scale + opacity animation.

**Animated SVG Line Drawing** — SVG paths draw themselves stroke-by-stroke, revealing illustrations or icons progressively. Good for loading sequences, step indicators, and decorative illustrations on scroll. Implemented with SVG `stroke-dasharray` set to path length and `stroke-dashoffset` animated from full to zero.

**Mesh Gradient Background** — Multi-point gradient blobs that slowly shift position and color, creating an organic, ambient background. Good for hero sections, auth pages, and ambient backgrounds that need to feel alive without being distracting. Implemented with multiple radial gradients with animated positions (CSS `@property` or canvas), or dedicated libraries.

**Lens Blur Depth** — Elements at different scroll positions or z-depths receive varying blur amounts, simulating camera depth-of-field. Good for parallax hero sections, layered compositions, and cinematic scroll experiences. Implemented with `filter: blur()` values driven by scroll position or z-index distance, GPU-composited for performance.

---

## Bento Dashboard Paradigm

Five archetypal tile types for bento-style dashboard layouts. Mix these to create information-rich interfaces with visual variety.

**The Intelligent List** — A compact, scrollable list tile showing ranked or chronological items with status indicators. Good for activity feeds, task lists, top-N rankings, and notification streams. Typically a narrow 1x2 or 1x3 tile with minimal chrome. Implementation: virtualized list for performance, subtle row hover states, status dots or mini-badges.

**The Command Input** — A prominent search or input tile that serves as the primary interaction point. Good for search-driven interfaces, AI chat inputs, and quick-action launchers. Usually a wide 2x1 tile near the top. Implementation: auto-focus on mount, keyboard shortcut indicator, typewriter placeholder animation, instant results preview.

**The Live Status** — A tile displaying real-time or frequently-updated data with ambient animation indicating liveness. Good for system health, market data, weather, and connection status. Usually a small 1x1 tile with a pulsing or breathing indicator. Implementation: WebSocket or polling data source, pulse animation on update, color-coded severity.

**The Wide Data Stream** — A horizontally-oriented tile showing time-series data, charts, or data visualizations. Good for analytics dashboards, monitoring, and trend displays. Usually a wide 2x1 or 3x1 tile. Implementation: lightweight charting (Recharts, Chart.js, or SVG), responsive axis labels, real-time data append with smooth transitions.

**The Contextual UI** — A tile whose content changes based on user state, time, or other context. Good for personalized greetings, contextual tips, weather-aware suggestions, and adaptive shortcuts. Any size, often a 2x1 tile. Implementation: conditional rendering based on user/time/location context, smooth content transitions between states.

---

## Attribution

Pattern library inspired by [taste-skill](https://github.com/Leonxlnx/taste-skill) by Leon Lin.

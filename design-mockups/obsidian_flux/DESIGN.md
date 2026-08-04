---
name: Obsidian Flux
colors:
  surface: '#0f131d'
  surface-dim: '#0f131d'
  surface-bright: '#353944'
  surface-container-lowest: '#0a0e18'
  surface-container-low: '#171b26'
  surface-container: '#1c1f2a'
  surface-container-high: '#262a35'
  surface-container-highest: '#313540'
  on-surface: '#dfe2f1'
  on-surface-variant: '#c7c4d7'
  inverse-surface: '#dfe2f1'
  inverse-on-surface: '#2c303b'
  outline: '#908fa0'
  outline-variant: '#464554'
  surface-tint: '#c0c1ff'
  primary: '#c0c1ff'
  on-primary: '#1000a9'
  primary-container: '#8083ff'
  on-primary-container: '#0d0096'
  inverse-primary: '#494bd6'
  secondary: '#4cd7f6'
  on-secondary: '#003640'
  secondary-container: '#03b5d3'
  on-secondary-container: '#00424e'
  tertiary: '#ddb7ff'
  on-tertiary: '#490080'
  tertiary-container: '#b76dff'
  on-tertiary-container: '#400071'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#e1e0ff'
  primary-fixed-dim: '#c0c1ff'
  on-primary-fixed: '#07006c'
  on-primary-fixed-variant: '#2f2ebe'
  secondary-fixed: '#acedff'
  secondary-fixed-dim: '#4cd7f6'
  on-secondary-fixed: '#001f26'
  on-secondary-fixed-variant: '#004e5c'
  tertiary-fixed: '#f0dbff'
  tertiary-fixed-dim: '#ddb7ff'
  on-tertiary-fixed: '#2c0051'
  on-tertiary-fixed-variant: '#6900b3'
  background: '#0f131d'
  on-background: '#dfe2f1'
  surface-variant: '#313540'
typography:
  display-lg:
    fontFamily: Outfit
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Outfit
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.01em
  headline-lg-mobile:
    fontFamily: Outfit
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  title-md:
    fontFamily: Outfit
    fontSize: 20px
    fontWeight: '500'
    lineHeight: 28px
  body-lg:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-md:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-sm:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.05em
  mono-code:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 4px
  xs: 8px
  sm: 16px
  md: 24px
  lg: 40px
  xl: 64px
  gutter: 20px
  margin-mobile: 16px
  margin-desktop: 32px
---

## Brand & Style

The design system is engineered for deep technical immersion, catering to developers and platform engineers managing complex event architectures. It adopts a **SaaS-Industrial** aesthetic, blending high-utility data density with a sophisticated **Glassmorphic** layer. 

The visual narrative is centered on "The Flow of Data." Surfaces are treated as physical layers of obsidian glass, where light (gradients) and shadow (depth) represent active processes and state changes. The emotional response is one of precision, control, and futuristic reliability. This design system avoids decorative fluff in favor of structural integrity, utilizing a strict grid and sharp, vibrant accents to guide the user through complex logical nodes.

## Colors

The palette is built upon a deep "Obsidian Slate" foundation to maximize visual comfort during long engineering sessions. 

- **Primary Axis:** A gradient transition from Vivid Electric Violet (#6366F1) to Indigo (#4F46E5) is used exclusively for primary actions and active workflow paths.
- **Functional Accents:** Distinct hues are mapped to system states: Cyan for "Test/Simulation" modes, Purple for "Experimentation" buckets, and Amber for "Draft/Pending" states.
- **Surface Strategy:** Backgrounds use the base #0B0F19. Interactive containers use #111827. Borders must remain subtle at #1F2937 to maintain a seamless transition between glass layers.

## Typography

This design system utilizes a dual-typeface strategy to balance character and readability. 

- **Outfit** is used for all display and heading levels. Its geometric clarity provides a modern, "engine-like" feel. 
- **Inter** handles all body copy and UI labels, ensuring maximum legibility at high data densities. 
- **JetBrains Mono** (or a system monospaced font) is reserved for event payloads, code snippets, and node IDs. 

High contrast is maintained by using `Slate-50` for primary text and `Slate-400` for secondary descriptions. Use `uppercase` with increased letter spacing for small labels to denote system metadata.

## Layout & Spacing

The layout operates on a **Fluid-Grid Hybrid** model. Dashboards and data views utilize a 12-column fluid grid. However, the Canvas Editor (The Engine) utilizes a "No Grid" contextual layout with an underlying 8px dot-matrix for node snapping.

- **Data Tables:** High-density spacing (8px cell padding) is preferred for experimentation logs.
- **Workflow Canvas:** Requires expansive margins (40px+) to allow for complex branching visualization.
- **Breakpoints:**
  - Mobile: < 768px (Single column, hidden sidebar)
  - Tablet: 768px - 1280px (Collapsible sidebar, fluid content)
  - Desktop: > 1280px (Fixed navigation, multi-pane panels)

## Elevation & Depth

Hierarchy is established through **Tonal Layering** and **Backdrop Blurs** rather than traditional heavy shadows.

1.  **Level 0 (Floor):** #0B0F19. The base workflow canvas.
2.  **Level 1 (Cards/Nodes):** #111827 with a 1px #1F2937 border. 
3.  **Level 2 (Modals/Toolbars):** Translucent #111827 (80% opacity) with `backdrop-filter: blur(12px)`.
4.  **Active State:** Elements in an active or "running" state gain a soft glow: `0 0 15px -3px rgba(99, 102, 241, 0.3)`.

Shadows, when used, are tight and "ink-like" (#000000 at 40% opacity) to give the impression of floating glass plates without creating visual mud.

## Shapes

The shape language is **Soft-Technical**. We use a primary corner radius of 0.25rem (4px) for most functional components to maintain a crisp, professional edge. Larger containers like Cards or the Workflow Canvas sidebar use 0.5rem (8px). 

**Custom Node Geometry:**
In the workflow engine, we depart from rectangles for specific logic:
- **Diamonds:** Decision points / If-Else logic.
- **Hexagons:** External Webhooks / API Triggers.
- **Split-Boxes:** A/B Test Experimentation buckets.

## Components

- **Buttons:** Primary buttons use the Indigo-Violet gradient with white text. Secondary buttons are "Ghost" style: 1px borders with a subtle hover fill.
- **Workflow Nodes:** Nodes must feature a 2px "status-strip" on the left edge (Emerald for success, Rose for failure). Connectors between nodes are 2px thick, using #334155 (Slate-700) for inactive and #6366F1 for active paths.
- **Glassmorphic Toolbars:** Float top-center in the canvas, using the 12px blur effect and a thin white outline (10% opacity) to catch the light.
- **Data Tables:** Strict 1px horizontal borders. No vertical lines. Row hovering should use a subtle highlight (#1F2937).
- **Chips/Badges:** Use a "Dimmed" style—low opacity backgrounds of the accent color (e.g., 10% Emerald) with high-contrast text for status indicators.
- **Input Fields:** Darker than the surface (#090D14) with a focus ring of #6366F1. Text within fields uses the Mono font for technical accuracy.
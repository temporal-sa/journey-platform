---
name: obsidian-design-system
description: Design guide, color tokens, approved typography, layout structures, Canvas Toolbar, Node Palette, StatusFilterDropdown, Toast standards, and component patterns for building dark theme UI components tailored to user preferences.
---

# Obsidian Design System Guide

This skill defines the visual architecture, color tokens, approved typography hierarchy, reusable layout structures (including `<DirectoryLayout>`), component patterns, and aesthetic guidelines tailored specifically for building dark-mode web components that fit seamlessly into the **Obsidian UI Theme**.

---

## 1. Reusable Layout Components & Structure

### A. `<DirectoryLayout>` Component (`web/src/components/common/DirectoryLayout.tsx`)
All primary directory pages (**Journeys**, **Component Catalog**, **Execution Runs**, **Static Lists**) must use the standardized `<DirectoryLayout>` wrapper to ensure unified padding, top header flex layouts, control filter bars, and background colors:

- **Filter Controls Bar Stacking**: The filter controls slot must be wrapped in `relative z-30` to establish an elevated stacking context, allowing filter dropdown overlays to float cleanly above table content without clipping.

### B. Canvas Toolbar (`web/src/components/editor/CanvasToolbar.tsx`)
- **Full Width Top Pinned Layout**: `absolute top-0 left-0 right-0 w-full z-40`
- **Borderless & Translucent Backdrop**: `border-none bg-[#0F131D]/40 backdrop-blur-md`
- **Left-Aligned Controls**: `justify-start flex items-center gap-1`
- **Zoom Controls**: Includes `Zoom In` and `Zoom Out` (No `Fit View` button).

---

## 2. Approved Fonts & Typography Hierarchy

### Font Families
- **Primary Display & UI Font**: `'Outfit', sans-serif` (Loaded via Google Fonts: weights 400, 500, 600, 700). Used for all headers, titles, buttons, navigation items, and body copy.
- **Technical & Monospace Font**: `font-mono` (`ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas`). Used for IDs, draft versions, code tokens, keyboard shortcuts, timestamps, and numeric counters.

---

## 3. Node Palette Component & Badges (`web/src/components/editor/Palette.tsx`)

### Category Toggle Badges & Node Icon Badges
All category collapse/expand headers (**Triggers**, **Decisions**, **Actions**, **Utilities**) and node icon badges MUST feature rich, consistent 20% accent background fills and matching 40% borders:

- **Triggers**: `bg-[#4cd7f6]/20 border-[#4cd7f6]/40 text-[#4cd7f6]`
- **Decisions**: `bg-[#f59e0b]/20 border-[#f59e0b]/40 text-[#f59e0b]`
- **Actions**: `bg-[#c0c1ff]/20 border-[#c0c1ff]/40 text-[#c0c1ff]`
- **Utilities**: `bg-rose-500/20 border-rose-500/40 text-rose-300`

---

## 4. Table Controls, Status Filter Dropdowns & Clear Filters

### A. `<StatusFilterDropdown>` (`web/src/components/common/StatusFilterDropdown.tsx`)
All status filter controls across directory tables must be rendered as custom dropdowns using `<StatusFilterDropdown>`:

- **Custom Badge Preservation**: Both the closed trigger button and open menu options MUST retain the status's exact custom badge styling (`bg-emerald-500/20 text-emerald-300 border-emerald-500/50`, etc.) with uppercase font-mono typography and checkmark indicators.
- **Solid Overlay Stacking**: The dropdown menu overlay MUST use `z-[100]` with a solid dark surface (`#171b26`) to prevent table text bleed-through or clipping.

### B. Always-Present Clear Filters Button
The **Clear Filters** control across ALL tables must be an **always-present icon-only button** (`icon="filter_alt_off"`) with a native tooltip:

```tsx
<Button
  type="button"
  variant="secondary-dark"
  icon="filter_alt_off"
  aria-label="Clear all active table filters"
  title="Clear all active table filters"
  onClick={() => { /* reset filters */ }}
/>
```

---

## 5. Inline Journey Name Input (`web/src/App.tsx`)

- **Instant Store & UI Sync**: Updates Zustand store (`setDraftName`) and local state (`setEditedName`) immediately on blur/Enter.
- **Clean Input Focus (No Blue Focus Ring)**: Input element MUST use explicit inline styles to suppress native browser accent outlines and blue borders:
  ```tsx
  style={{
    background: '#11141d',
    backgroundColor: '#11141d',
    border: '1px solid #464554',
    outline: 'none',
    boxShadow: 'none',
    WebkitAppearance: 'none',
  }}
  ```
- **No Flashing Text**: No helper copy ("Press Enter to save") next to the input field.
- **Success Toast**: Fires a green Success Toast (`ToastMessageType.SUCCESS`) upon successful rename.

---

## 6. Color Palette, Tokens & Toasts

### Surface & Background Tokens
- **Canvas / App Background**: `#0B0F19` (Deep Slate Black)
- **Modal / Card / Panel Surface**: `#0F131D` (Obsidian Dark Surface)
- **Table Alt Row / Card Header**: `#171b26` (Subtle Dark Slate)
- **Input / Kbd / Inset Surface**: `#1F2433` (Inset Dark Panel)

### Toast Notifications (`web/src/components/common/Toast.tsx`)
- **Position & Overlay**: Top-center floating toast (`fixed top-[1.5rem] left-1/2 -translate-x-1/2 z-50 w-[50vw]`) with backdrop blur and `#0F131D/60` dark glass surface.
- **Status Colored Borders**: Uses status-colored 60% opacity borders (`border-rose-500/60`, `border-emerald-500/60`, `border-amber-500/60`, `border-blue-500/60`).
- **Flat Close Button**: Features a flat close button without beveling (`appearance: none`, flat style).
- **Feedback Toasts**:
  - **Error Toast**: Fires for API failure modes (static list upload failure, draft rename failure, test run failure).
  - **Success Toast**: Fires for successful journey draft rename and save actions.

---

## 7. Aesthetic Rules & Checklist
1. **Always Use `<DirectoryLayout>` for Directory Pages**: Wrap all list/directory pages in `<DirectoryLayout>` with `relative z-30` on controls.
2. **Never Use Native Light Mode Defaults or Native Blue Focus Rings**: Enforce dark slate borders (`#464554`) and `outline: none`.
3. **Enforce 0px Sharp Borders**: Apply `rounded-none` to all interactive components, cards, dropdowns, and modals.
4. **Use Approved Typography**: Use `'Outfit', sans-serif` for headers/labels and `font-mono` for IDs and code tokens.
5. **Always Present Icon-Only Clear Filters**: Render `filter_alt_off` icon button with `title="Clear all active table filters"` on all table filter bars.
---

## 8. Dark Mode Toggle & Preset Button Rules (Mandatory)

### A. Zero Light Mode / White Background Exception
- **NEVER use solid white, light grey, or unstyled light mode backgrounds** (`#ffffff`, `bg-white`, `bg-slate-100`, `bg-gray-100`, `bg-gray-200`) for active toggles, preset signal buttons, category chips, or filter buttons anywhere in the application.
- **Whole Website Dark Theme Enforcement**: The entire application is built exclusively in Obsidian Dark Mode. Light mode buttons or unstyled browser defaults are prohibited.

### B. Standardized Active vs Inactive Toggle Tokens
- **Active State Tokens**: Active toggle & preset buttons MUST use dark theme 20% accent fill tints with matching 50-60% accent borders and bright text:
  - **Cyan Active Preset**: `bg-[#4cd7f6]/20 text-[#4cd7f6] border border-[#4cd7f6]/60 font-bold shadow-sm`
  - **Lavender / Action Active Preset**: `bg-[#c0c1ff]/20 text-[#c0c1ff] border border-[#c0c1ff]/60 font-bold shadow-sm`
  - **Emerald Active Preset**: `bg-emerald-500/20 text-emerald-300 border border-emerald-500/60 font-bold shadow-sm`
  - **Amber Active Preset**: `bg-[#f59e0b]/20 text-[#f59e0b] border border-[#f59e0b]/60 font-bold shadow-sm`
- **Inactive State Tokens**: Inactive toggle & preset buttons MUST use dark slate surface (`bg-[#171b26]`), muted text (`text-[#908fa0]`), and dark slate borders (`border border-[#464554]/60 hover:text-white hover:border-[#dfe2f1]/40`).
- **Button Component**: When rendering interactive toggles in modals or forms, always wrap or use the `<Button>` component (`variant="secondary-dark"` for inactive, dark accent fill tint for active).

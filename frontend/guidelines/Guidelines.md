# HotSpot Interaction & UI Consistency Rules

This project is a restrained, responsive Xuzhou nearby-social product with two semantic themes only: **星际** (`vibrant`) and **清野** (`forest`), each in light and dark mode. Preserve semantic CSS variables and the shared primitives below; do not introduce page-local visual variants for an interaction that already has a shared pattern.

## Non-negotiable interaction primitives

### Compact icon actions
- Any icon-only action in a header, toolbar, floating control, or popover trigger **must use** `src/app/components/shared/ActionIconButton.tsx`.
- Use `variant="ghost"` for contextual actions in flat headers and toolbars. It is always circular and, on hover, receives the theme-aware circular surface from `useHoverConfig().bgHover`, accent icon color, and the shared press / scale motion. Use `surface` only where the visual hierarchy intentionally calls for a persistent elevated circular control; do not create a third local variant.
- Use `active` for an open or selected trigger. Active controls use `--neu-accent-soft` and `--neu-accent-soft-border`.
- Do **not** create square-rounded icon controls, one-off hover fills, or per-page icon hover animation. Use `ActionIconButton` instead.
- The desktop `HeaderBtn` is only a compatibility wrapper around `ActionIconButton`; do not add styling logic to it.

### Selection controls
- Controls that choose one option from a small set use the same segmented-selection pattern as the composer audience chooser: semantic container, compact option rows/cells, `aria-pressed`, accent-soft selected state, and ghost-surface hover for unselected options.
- A selection changes only its selected value. It must **not** dismiss the surrounding menu, sheet, or dialog unless the label explicitly communicates completion or navigation.
- A transient panel closes only by an explicit completion/navigation action, outside press where appropriate, or Escape through `useDismissibleLayer`.

### Buttons, feedback, and focus
- Use `useHoverConfig` through shared components. Do not invent local lift, scale, color, or shadow behavior for controls of an existing type.
- Every interactive control needs a visible `focus-visible` state using semantic accent/ring variables and a real button element when it performs an action.
- Primary actions use `--neu-accent-primary`; supporting actions use the established `neu-surface-flat` or `neu-surface-ghost` hierarchy. Do not hard-code page-level colors.

## Clean-edge shadow policy

“Dirty shadows” — dark or light halos visible outside a rounded, clipped, or tightly adjacent surface — are a visual defect, not a stylistic option.

- Structural shells (desktop workspace, main content field, side navigation, rails, and card canvases) must use **no external shadow**. Separate them with spacing, a semantic border, or a subtle background shift instead.
- Do not combine a rounded parent, `overflow: hidden`, and an outward `box-shadow` on that parent or on a child that touches its edge. Use an inset ring or a border for edge definition.
- External shadows are reserved for transient overlays only: dialogs, sheets, popovers, and intentionally floating map controls. They must be single-direction, low-opacity, and never use paired light/dark neumorphic shadows.
- Persistent cards use `neu-surface-flat`; a focal `neu-surface-hero` may elevate only when it has clear breathing room on every side. Cards in a dense canvas must not carry an always-on outer shadow.
- Icon-only controls, compact navigation, badges, and anchored utility entries must not use external shadows. Use background, tint, border, or focus ring to signal state.
- Before merging a new rounded surface, inspect all four corners in both themes and at the top/bottom of a scroll container. If a halo is visible, remove the outer shadow at the shared primitive rather than masking it locally.

## Layout and theme contract
- Preserve `ContextRail`, `AppearanceMenu`, `ActionIconButton`, `useDismissibleLayer`, `SocialContext`, and centralized `ROUTES` as the shared sources of truth.
- Preserve the token contract in `src/styles/theme.css` and `src/styles/index.css`. Use semantic `--neu-*`, `--semantic-*`, and mapped Tailwind tokens; do not reintroduce retired themes or hard-coded page colors.
- Test interaction changes in both themes and both light/dark modes. On desktop, tablet, and mobile the behavior must be the same even when the layout changes.

## Required verification before handoff
- Run `pnpm verify` after changes to routes, imports, shared components, pages, or features.
- `pnpm verify` parses every app TS/TSX source file, checks JSX Lucide icons and core shared primitives for missing imports, and bundles the actual Foundry entry (`__figma__entrypoint__.ts`) to resolve the dependency graph.
- Do not rely on `pnpm build` as the only guard in this repository: the Foundry project does not use a root Vite `index.html` entry.

## Shared content canvas
- Desktop routed pages render inside the shared `desktop-page-canvas`; tablet routed pages render inside `tablet-page-canvas`. Do not introduce page-local outer `max-width`, centering, or shell gutters for the primary content region.
- The right-hand workspace owns its outer spacing through `desktop-workspace`; its full-height main panel owns the scroll region and canvas alignment, but must not become a rounded or shadowed structural card. Individual pages may compose their own internal columns, but their outer left/right edges must align to this canvas.
- Any wide page with a contextual rail must use `context-rail-grid` / `ContextRail` so the rail width and inter-column gap remain identical to every other routed page.
- Use `content-mode-wide` for feeds, directories, maps, and split workspaces: their internal grids may span the canvas. Use `content-mode-reading` for settings, editing, and narrative detail content: cap only the readable column with `--content-reading-max`, never with a page-local pixel value.

## Entry-point hierarchy
- **Desktop global quick actions** live only in the top-right quick-actions area: Search, Publish, Notifications, Appearance, and Personal profile. Do not duplicate these actions in a desktop page body or a second desktop header.
- **Contextual actions remain local**: Back, Share a specific profile/post/event, map actions, page filters, inline search, and group creation belong to the page whose content they affect. They are not candidates for the global quick-actions area.
- **Cross-platform equivalents are intentional, not duplicates**: when desktop has a global quick action, mobile/tablet may expose the same capability through the bottom composer, a compact page header, or a sheet because the desktop header is absent.
- Before adding an entry, classify it as global, contextual, or a cross-platform equivalent. Reuse the established entry for global actions; do not add a second path without a platform or context reason.

## Affordance integrity
- Do not apply `cursor-pointer`, tap animation, or hover motion to a content item unless it has a real activation result.
- Repeated content units that navigate (avatars, cards, rows, thumbnails) must use a semantic `button` or link, have a concise accessible name, and expose the shared focus treatment.
- “More” affordances must lead to a real expanded list, route, or sheet; they must never be decorative-only.

## Canonical resource routes
- Use API-domain URLs for all new navigation: `/moments/:momentId`, `/groups/:groupId`, `/users/:userId`, `/map`, `/nearby/people`, `/nearby/groups`, and `/me/*`.
- Personal resources are first-class pages: use `/me/moments`, `/me/groups`, `/me/friends`, `/me/following`, and `/me/followers`; do not introduce new `?section=` profile states.
- Legacy URLs may exist only as redirects in `routes.tsx`. New components and links must always use builders from `src/app/lib/routes.ts`.


## Discovery card contract

The three discovery directories — **附近的人**, **附近群组**, and **附近瞬间** — share one card interaction language even though their data layout differs.

- Discovery cards use `neu-card-hover` plus `useHoverConfig().card` and `useHoverConfig().tap`. Do not substitute a local `y`, `scale`, shadow, or color animation.
- Default: the content-specific surface (portrait, group cover, or moment media) remains visually dominant.
- Hover: one shared card lift, soft border emphasis, and theme-aware hover shadow. Do not add a left status line or a full accent fill to discovery cards.
- Press: one shared tactile tap response. Nested actions (follow, join, like, save) stop propagation and use the smaller action feedback rather than re-triggering the card transition.
- Selected is reserved for persistent filters, map/location choices, and current navigation. Directory cards navigate; they should not invent a selected state after opening a detail page.
- Every navigable card must be keyboard reachable (`role="button"`, `tabIndex={0}`, Enter/Space activation) and expose the shared focus ring.
- Keep the content variation purposeful: people foreground identity and availability; groups foreground cover, activity, and membership; moments foreground media and social response. Only the shell and interaction contract are shared.

## Reusable data display primitives

Use `src/app/components/shared/DataDisplay.tsx` before creating page-specific stat, key/value, or status UI:

- `MetricStrip` for horizontally comparable metrics.
- `DataKeyValue` for metadata and settings-style label/value pairs.
- `DataStatus` for compact semantic states.
- List data uses `app-list-row--compact`, `app-list-row--standard`, or `app-list-row--rich` with the shared `neu-row-hover` interaction contract.

## Menu-item interaction

Menus and popovers are navigation/choice surfaces, not floating cards. Menu items may change background, text color, or a check/state indicator on hover; they must **not scale or lift on hover**. Only pressed feedback and control-internal state animation (such as a switch thumb) are appropriate.


### BSP discovery canvas

The three discovery directories — **附近的人**, **附近群组**, and **附近瞬间** — use `BspCardCanvas`, backed by `generateBspCardLayout`, for desktop content cards.

- The canvas observes its real rendered width with `ResizeObserver` and derives a safe dynamic height from width and ratio constraints. Keep that height close to the root's first mathematically splittable boundary (`2 / maxRatio`) rather than making it arbitrarily taller; never pass a hard-coded display width or height. Sidebar and rail changes must automatically regenerate its bounds.
- The solver selects a random value **inside a mathematically valid split interval**, rather than guessing a split point from the whole rectangle. It must preserve the configured minimum size and min/max card ratios.
- Keep the ratio range narrow enough for content readability (normally about `0.68–1.48`). If the target count is difficult, search deterministic seed variants and reduce `minSize`; do not make the canvas arbitrarily taller or loosen the ratio limit.
- The BSP canvas is transparent and has no outer radius, border, clipping, or background plate. The gap belongs to the page background; only individual cards own surfaces and rounded corners.
- Preserve source/keyboard order even when visual positions vary. On mobile and narrow tablet, use the dedicated readable one- or two-column fallback rather than forcing a dense irregular canvas.


## Regression prevention: desktop directory and BSP failures

The following failures have occurred in this project. Treat the rules here as implementation constraints, not optional visual preferences.

### Sticky page headers and scroll boundaries

**What went wrong**
- Applying top padding to the shared desktop scrollport created a physical strip above page-local sticky headers. Scrolling content could occupy that strip and appear above the directory title.
- A translucent sticky header (`color-mix(..., transparent)`) made content scrolling underneath it visibly leak through, which looked identical to content escaping its container.
- Relying on `z-index` alone did not solve the problem when the scrollport geometry itself was wrong.

**Required implementation**
- `desktop-main-scroll` is the physical scrollport and must have `padding-block: 0`.
- Page-specific content spacing belongs inside each page body (`main` / content wrapper), never above a page-local sticky header in the shared scrollport.
- Any directory title bar that overlays scrolling content must be `sticky top-0`, have an opaque `var(--neu-bg)` background, a border-bottom, and a higher local z-index than its content.
- The desktop global header must remain in a separate higher stacking context; the main panel must use `isolation: isolate` so page-local sticky elements cannot escape above it.
- Test by scrolling the first content row directly under every directory header. No avatar, image, card, text, or shadow may be visible through or above the title bar.

### BSP layout feasibility and rendering

**What went wrong**
- A static or arbitrarily tall canvas can make the first BSP split mathematically impossible under bounded card ratios. A near-square root cannot split into two safe leaves, and a root that is too tall can fail for the same reason.
- Randomly sampling the whole split range could miss the narrow legal interval even when a valid split existed.
- Applying BSP only to a “rest” list while keeping a featured fixed grid meant the page as a whole was still not a BSP layout.
- Returning an incomplete list of leaves rendered only a few oversized cards, which looked like the algorithm was not active.

**Required implementation**
- Use `BspCardCanvas` for every desktop discovery directory card collection: **附近的人**, **附近群组**, and **附近瞬间**. Do not wrap part of the same collection in a fixed grid or featured two-column layout.
- Observe the real rendered canvas width with `ResizeObserver`; never use a fixed display canvas size.
- Derive canvas height from the first safe split boundary: approximately `(2 / maxRatio) * width`, with only a small safety offset. Do not increase height just because the item count increases.
- `generateBspCardLayout` must solve the legal split interval from min size and ratio constraints before choosing a random split point. Never randomly guess across the entire parent rectangle.
- Keep the displayed card ratio bounded (normally `0.68–1.48`). When a full target count is difficult, try deterministic seed variants and lower only `minSize`; never loosen the ratio bounds or silently render a partial result.
- Verify a collection with 5, 7, and 12 cards at both expanded and collapsed sidebar widths. The returned layout must contain the full target count and every rendered card must use an absolute rectangle returned by the BSP function.

### Full-height desktop work field

**What went wrong**
- Treating the desktop main area as a giant rounded, shadowed card made page content look clipped or collide visually with the shell corners at the beginning and end of a scroll.

**Required implementation**
- The desktop main field fills the available height and has no structural border, radius, or external shadow.
- Keep any tonal treatment on the main field as a low-contrast background gradient only.
- Content pages own their internal horizontal and vertical padding; the shared shell must not manufacture a large card around them.

## Maintainable component boundaries

Keep ownership clear so a feature can change without creating a second visual or state system:

- `app/routes.tsx` and `app/lib/routes.ts` own URL declarations and route identity only. Routed pages compose features; they do not duplicate global navigation logic.
- `app/components/layout/` owns platform shells, global navigation, shared overlays, and workspace geometry.
- `app/components/shared/` owns cross-domain primitives such as `ActionIconButton`, `BspCardCanvas`, and `DataDisplay`. A shared component must consume semantic tokens and cannot introduce a page-local palette.
- `app/features/<domain>/` owns domain UI and local data contracts. Messaging components live in `features/messaging`; group creation belongs to `features/groups`; profile and location follow the same rule. Do not put domain features back into a generic `components/ui/` bucket.
- Do not keep generated primitive wrappers or package dependencies that have no application import. Add Radix primitives only when their accessibility behavior is needed by a real product surface; do not retain a dormant component library “just in case.”

## Token and selected-state contract

- Use semantic theme variables (`--neu-*`, `--semantic-*`, `--overlay-*`) for application UI. Raw black/white translucency is reserved for media scrims, image overlays, and modal backdrops where it represents content treatment rather than the app palette.
- Persistent selectable rows use `neu-row-hover` plus `data-active="true"` (and `aria-pressed` only when the control is a toggle). The selected state is anchored; active-hover strengthens fill and border but must not lift or scale the row.
- Filter chips, theme choices, and other persistent selections use `aria-pressed="true"` so the same active-hover token contract applies automatically.
- A selected control must keep normal readable text tokens unless its filled background explicitly uses `--neu-on-accent`. Never rely on white text over a theme accent by default.

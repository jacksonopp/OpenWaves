# OpenWaves Admin UI — Design Guidelines

> **Figma source:** [OpenWaves Admin Panel — Streams view](https://www.figma.com/design/lnVoVSqp4NtQcUNg24hkye/OpenWaves?node-id=3-628)

---

## 1. Overview

The OpenWaves admin panel is a management interface for station operators. It exposes station lifecycle controls, live stream monitoring, relay moderation, and federation status in a single-page application.

**Tech stack:**
- **React 19** — component model
- **TanStack Router** — client-side routing (auth guard via `beforeLoad`)
- **TanStack Query** — server-state fetching and polling (`refetchInterval: 3000`)
- **CSS Modules** — scoped per-component styles; no global utility classes

The layout is a fixed top bar + left sidebar + scrollable main content area. All admin routes live under `/admin/ui/` (served by the embedded SPA at `/admin/ui/`).

### Route structure

| Path | Component | Description |
|---|---|---|
| `/admin/ui/login` | `Login` | Admin key entry (public) |
| `/admin/ui/` | `AdminLayout` (auth-guarded) | Shell; redirects to `/streams` |
| `/admin/ui/streams` | `StreamsPage` | Active stations list — main admin view |
| `/admin/ui/overview` | `OverviewPage` | Station stats + live log feed |
| `/admin/ui/moderation` | `ModerationPage` | Moderation tools (placeholder) |
| `/admin/ui/federation` | `FederationPage` | Federation management (placeholder) |

Auth guard: `beforeLoad` checks `localStorage.getItem('adminKey')` and redirects to `/admin/ui/login` if absent.

---

## 2. Color Tokens

The admin panel uses a flat set of design tokens. They are defined as CSS custom properties and should live in `ui/src/index.css` under an `.admin-root` scope (or equivalent) so they are available to all CSS Modules via `var()`.

> **Current state:** These values are hardcoded directly in each CSS Module file (e.g., `Sidebar.module.css`, `AdminLayout.module.css`). Migrating them to custom properties in `index.css` is the recommended next step to enforce consistency.

```css
/* Recommended definition in ui/src/index.css */
:root {
  /* Admin panel tokens */
  --admin-bg:      #fafafa;             /* page / main content background  */
  --admin-surface: #ffffff;             /* card and panel backgrounds       */
  --admin-accent:  #6366f1;             /* primary accent — nav active,     */
                                        /* primary buttons, logo bg         */
  --admin-text:    #0a0a0a;             /* primary text, headings           */
  --admin-muted:   #717182;             /* secondary text, subtitles,       */
                                        /* placeholders, meta               */
  --admin-border:  rgba(0, 0, 0, 0.1); /* card borders, dividers           */
  --admin-live:    #ef4444;             /* LIVE badge background            */
  --admin-danger:  #d4183d;             /* destructive actions, error state */
}
```

### Token quick-reference

| Token | Value | Used for |
|---|---|---|
| `--admin-bg` | `#fafafa` | `<body>`, main content area |
| `--admin-surface` | `#ffffff` | Cards, sidebar, top bar |
| `--admin-accent` | `#6366f1` | Active nav items, primary buttons, logo square, avatar bg |
| `--admin-text` | `#0a0a0a` | All body text, nav labels, headings |
| `--admin-muted` | `#717182` | Subtitles, email addresses, meta text, `"Federated via ActivityPub"` |
| `--admin-border` | `rgba(0,0,0,0.1)` | Card outlines, sidebar/topbar dividers |
| `--admin-live` | `#ef4444` | LIVE badge, numeric notification badges in sidebar nav |
| `--admin-danger` | `#d4183d` | Stop-stream buttons, destructive confirmations |

---

## 3. Typography

**Font family:** `Inter, system-ui, -apple-system, sans-serif`

Inter is the sole typeface. Declare it as a `font-family` on the admin layout root so all descendant components inherit it:

```css
/* AdminLayout.module.css */
.root {
  font-family: Inter, system-ui, -apple-system, sans-serif;
}
```

### Scale

| Role | Size | Weight | Color | Notes |
|---|---|---|---|---|
| Page / section title | 24px | 600 (Semi Bold) | `#0a0a0a` | letter-spacing `0.07px`; used for "Active Streams", "Overview" |
| Page subtitle | 16px | 400 (Regular) | `#717182` | Sits directly below the title, e.g. "Manage live broadcasts" |
| Card title | 18px | 600 | `#0a0a0a` | Inside stream / stat cards |
| Stat number | 32px | 700 (Bold) | `#0a0a0a` | Dashboard stat callouts, `line-height: 1` |
| Nav labels | 16px | 500 (Medium) | `#0a0a0a` or `#ffffff` | Inactive / active respectively |
| Body text | 16px | 400–500 | `#0a0a0a` | General prose inside cards |
| Small / meta | 14px | 400–500 | `#717182` | Subtitles, email addresses, section sub-headings |
| Brand name (top bar) | 16px | 600 | `#0a0a0a` | "OpenWaves Protocol" in the top bar |
| Sidebar title | 24px | 600 | `#0a0a0a` | "OpenWaves" wordmark in sidebar brand area |
| Sidebar subtitle | 12px | 400 | `#717182` | "Admin Panel" under sidebar title |
| Badges / micro | 12px | 500–600 | `#ffffff` or token-specific | Pill labels (LIVE, count badges) |

---

## 4. Layout

The admin shell is a three-region layout built with flexbox.

```
┌─────────────────────────────────────────────┐
│  Top bar  (fixed, full-width, 57px)         │
├──────────┬──────────────────────────────────┤
│          │                                  │
│ Sidebar  │  Main content                    │
│ (256px)  │  (flex: 1, padding: 32px)        │
│          │                                  │
│  User    │                                  │
└──────────┴──────────────────────────────────┘
```

### Top bar

| Property | Value |
|---|---|
| Height | 57px (fixed, `position: fixed; top: 0`) |
| Background | `#ffffff` |
| Border | `border-bottom: 1px solid rgba(0,0,0,0.1)` |
| Padding | `0 24px` |
| z-index | `100` |

Contains: wordmark + vertical divider + Client/Admin toggle tabs on the left; "Federated via ActivityPub" label on the right.

Source: [`ui/src/components/admin/TopBar.module.css`](../ui/src/components/admin/TopBar.module.css)

### Sidebar

| Property | Value |
|---|---|
| Width | 256px (fixed, `min-width: 256px`) |
| Height | 100% of remaining viewport (below top bar) |
| Background | `#ffffff` |
| Border | `border-right: 1px solid rgba(0,0,0,0.1)` |

The sidebar is divided into three vertical zones:
1. **Brand area** — 109px tall, `border-bottom` divider
2. **Navigation** — `flex: 1`, scrollable
3. **User profile footer** — 93px tall, `border-top` divider

Source: [`ui/src/components/admin/Sidebar.module.css`](../ui/src/components/admin/Sidebar.module.css)

### Main content

| Property | Value |
|---|---|
| Width | `flex: 1` (fills remaining width after sidebar) |
| Background | `#fafafa` |
| Padding | `32px` (set per-page, e.g. `OverviewPage.module.css`) |
| Overflow | `overflow-y: auto` |
| Max-width | Unrestricted by the shell; pages may cap their own content width (e.g. `OverviewPage` uses `max-width: 960px`) |

Source: [`ui/src/components/admin/AdminLayout.module.css`](../ui/src/components/admin/AdminLayout.module.css)

---

## 5. Component Patterns

### 5.1 Top Bar — Client / Admin Toggle Tabs

Two pill-shaped tabs sit in the top bar. The inactive tab uses a grey fill; the active tab uses the accent color.

```css
/* inactive */
.tabClient {
  height: 32px;
  padding: 0 14px;
  background: #eceef2;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #0a0a0a;
}

/* active */
.tabAdmin {
  height: 32px;
  padding: 0 14px;
  background: #6366f1;   /* --admin-accent */
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #ffffff;
}
```

### 5.2 Sidebar — Brand Area

```css
.brandIcon {
  width: 40px;
  height: 40px;
  background: #6366f1;      /* --admin-accent */
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.brandHeading {
  font-size: 24px;
  font-weight: 600;
  color: #0a0a0a;           /* --admin-text */
  line-height: 1.1;
}

.brandSub {
  font-size: 12px;
  font-weight: 400;
  color: #717182;           /* --admin-muted */
}
```

### 5.3 Sidebar — Navigation Items

Nav items occupy the full sidebar width at 48px tall. There is a `4px` gap between consecutive items.

```css
/* shared geometry */
.navItem,
.navItemActive {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 48px;
  padding: 0 14px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 500;
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
  box-sizing: border-box;
}

/* inactive */
.navItem {
  background: transparent;
  color: #0a0a0a;
}

.navItem:hover {
  background: rgba(99, 102, 241, 0.06);  /* accent at 6% opacity */
}

/* active */
.navItemActive {
  background: #6366f1;   /* --admin-accent */
  color: #ffffff;
}
```

Icons inside nav items are `20×20px`. When a nav item is active, the icon inherits the white `color` via `currentColor` (SVG stroke/fill should reference `currentColor`).

### 5.4 Sidebar — Notification Count Badge

Displayed inline at the trailing edge of a nav item (e.g., Moderation shows `2`).

```css
.badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  background: #ef4444;    /* --admin-live */
  color: #ffffff;
  font-size: 11px;
  font-weight: 600;
  border-radius: 10px;
}
```

> Note: this badge uses `--admin-live` (`#ef4444`), not `--admin-danger`. Reserve `--admin-danger` (`#d4183d`) for destructive action buttons and error states.

### 5.5 Sidebar — User Profile Footer

```css
.avatar {
  width: 32px;
  height: 32px;
  background: #6366f1;    /* --admin-accent */
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
  color: #ffffff;
}

.userName {
  font-size: 14px;
  font-weight: 500;
  color: #0a0a0a;
}

.userEmail {
  font-size: 12px;
  color: #717182;
}
```

The avatar displays the user's initials (e.g., `AM` for "Admin"). Long names and email addresses are truncated with `text-overflow: ellipsis`.

### 5.6 Buttons

**Primary** — used for the main page action (e.g., "Create New Channel"):

```css
.btnPrimary {
  height: 40px;
  padding: 0 16px;
  background: #6366f1;    /* --admin-accent */
  color: #ffffff;
  border: none;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 500;
  cursor: pointer;
}
```

**Secondary / Outlined** — used for secondary stream actions (e.g., "Monitor"):

```css
.btnSecondary {
  height: 40px;
  padding: 0 16px;
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.2);
  color: #0a0a0a;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}
```

**Icon-only / Stop** — square 40×40px outlined button, centered icon:

```css
.btnIcon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ffffff;
  border: 1px solid rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  cursor: pointer;
  color: #d4183d;         /* --admin-danger for stop/destructive icons */
}
```

### 5.7 Status Badges (LIVE / OFFLINE / etc.)

All status badges share the same pill geometry:

```css
/* base shape */
.statusBadge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 100px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}
```

Variant fills:

| Variant | Background | Text color | Use when |
|---|---|---|---|
| `LIVE` | `#ef4444` | `#ffffff` | Stream is actively broadcasting |
| `OFFLINE` | `#e5e7eb` | `#6b7280` | Station exists but is not live |
| `RELAYING` | `#dbeafe` | `#1d4ed8` | This server is re-hosting another station's stream |
| `INGESTING` | `#dbeafe` | `#1d4ed8` | Stream is being ingested (source side) |

```css
.badgeLive     { background: #ef4444; color: #ffffff; }
.badgeOffline  { background: #e5e7eb; color: #6b7280; }
.badgeRelaying { background: #dbeafe; color: #1d4ed8; }
.badgeIngesting{ background: #dbeafe; color: #1d4ed8; }
```

### 5.8 Cards

Cards are used for stream entries, stat summaries, and other grouped content.

```css
.card {
  background: #ffffff;              /* --admin-surface */
  border: 1px solid rgba(0,0,0,0.1); /* --admin-border */
  border-radius: 12px;
  padding: 24px;
}
```

**Stream card layout** — content (stream name, status badge, author, listener count, uptime, current audio input) is left-aligned; action buttons (Start Stream / Monitor, Stop) are right-aligned via `justify-content: space-between`. Dynamic channels (not defined in `config.yaml`) show a delete button in the settings panel. The settings panel contains three sections: **Audio Input** (silence / test tone / file selector), **Stream** (start/stop stream), and **Relay** (relay controls).

**Stat card** (Overview page) uses a tighter `padding: 20px` and stacks a large stat number (`32px bold`) above a small label (`14px muted`):

```css
.statCard {
  flex: 1;
  background: #ffffff;
  border: 1px solid rgba(0,0,0,0.1);
  border-radius: 12px;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.statNumber {
  font-size: 32px;
  font-weight: 700;
  color: #0a0a0a;
  line-height: 1;
}

.statLabel {
  font-size: 14px;
  color: #717182;
}
```

Source: [`ui/src/pages/admin/OverviewPage.module.css`](../ui/src/pages/admin/OverviewPage.module.css)

---

## 6. CSS Modules Convention

| Rule | Detail |
|---|---|
| **File naming** | `ComponentName.module.css`, co-located with `ComponentName.tsx` |
| **Class naming** | camelCase (e.g., `.navItemActive`, `.statNumber`) |
| **Color values** | Use `var(--admin-*)` tokens once defined in `index.css`; currently hardcoded values per module |
| **No global selectors** | Never write element selectors (`h1 { }`) inside a Module file — use class names only |
| **Transitions** | Interactive elements use `transition: background 0.15s` (no layout-affecting properties) |
| **Box sizing** | `box-sizing: border-box` on any fixed-height element that uses padding |

**Example — a new page:**

```css
/* ui/src/pages/admin/MyPage.module.css */
.page {
  padding: 32px;
}

.heading {
  font-size: 24px;
  font-weight: 600;
  color: #0a0a0a;          /* --admin-text */
  margin: 0 0 6px 0;
}

.subtitle {
  font-size: 14px;
  color: #717182;          /* --admin-muted */
  margin: 0;
}
```

```tsx
// ui/src/pages/admin/MyPage.tsx
import styles from './MyPage.module.css';

export default function MyPage() {
  return (
    <div className={styles.page}>
      <h1 className={styles.heading}>My Page</h1>
      <p className={styles.subtitle}>Description here</p>
    </div>
  );
}
```

---

## 7. Figma Source

The admin panel was designed in Figma. The canonical reference frame is the **Streams** screen:

- **File:** `lnVoVSqp4NtQcUNg24hkye` (OpenWaves)
- **Node:** `3-628` (Streams view — full admin shell)
- **URL:** https://www.figma.com/design/lnVoVSqp4NtQcUNg24hkye/OpenWaves?node-id=3-628

The Figma file is the source of truth for visual decisions (spacing, color, radius, typography). When the Figma and this document conflict, the Figma takes precedence — update this document to match.

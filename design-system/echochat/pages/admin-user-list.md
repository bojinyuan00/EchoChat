# Admin User List Page Overrides

> **PROJECT:** EchoChat
> **Generated:** 2026-03-02 10:58:45
> **Page Type:** Dashboard / Data View

> ⚠️ **IMPORTANT:** Rules in this file **override** the Master file (`design-system/MASTER.md`).
> Only deviations from the Master are documented here. For all other rules, refer to the Master.

---

## Page-Specific Rules

### Layout Overrides

- **Max Width:** 1400px or full-width
- **Grid:** 12-column grid for data flexibility
- **Sections:** 1. Hero (product + aggregate rating), 2. Rating breakdown, 3. Individual reviews, 4. Buy/CTA

### Spacing Overrides

- **Content Density:** High — optimize for information display

### Typography Overrides

- No overrides — use Master typography

### Color Overrides

- **Strategy:** Trust colors. Star ratings gold. Verified badge green. Review sentiment colors.

### Component Overrides

- Avoid: Use for flat single-level sites
- Avoid: Ignore accessibility motion settings
- Avoid: Validate only on submit

---

## Page-Specific Components

- No unique components for this page

---

## Recommendations

- Effects: Hover tooltips, chart zoom on click, row highlighting on hover, smooth filter animations, data loading spinners
- Navigation: Use for sites with 3+ levels of depth
- Animation: Check prefers-reduced-motion media query
- Forms: Validate on blur for most fields
- CTA Placement: After reviews summary + Buy button alongside reviews

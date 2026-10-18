---
title: Themes
description: Seven built-in looks, an accent picker, and a simple format for your own.
tags: [appearance]
---

# Themes

Open **Appearance** from the palette button at the bottom of the sidebar to switch themes and accent colors. Your choice is remembered per browser.

## Built-in themes

| Theme | Style |
| --- | --- |
| **Automatic** | Daylight or Graphite, following your system appearance |
| **Daylight** | Crisp and airy. The default in light mode |
| **Graphite** | Deep, quiet dark mode. The default in dark mode |
| **Glass** | Luminous layered glass over a soft, living color field |
| **Nebula** | Spatial dark glass floating in deep space |
| **Aurora** | Polar light drifting slowly behind smoked glass |
| **Dune** | Warm paper and sunset light, set in a reading serif |
| **Ink** | Editorial monochrome. No glass, no color, just type |

Animated backgrounds respect your system's *Reduce Motion* setting.

## Choosing a theme

- **Appearance** in the sidebar shows a live preview of every theme.
- The ⌘K palette has a *Theme:* command for each one.
- `theme:` in `goki.yaml` or `--theme` sets the default for everyone.
- `?theme=<id>` on any URL previews a theme without saving it.

Switching themes cross-fades smoothly in browsers that support View Transitions. The interface uses SF Pro on Apple devices and the embedded Inter typeface elsewhere, so goki looks the same with or without an internet connection.

## Accent color

Pick one of the accent swatches in **Appearance**, or set any CSS color as the wiki's default:

```yaml
accent: "#ff375f"
```

Selection, link and highlight tints are derived from the accent with `color-mix()`, so a single color restyles the whole interface.

## Your own themes

A theme is a CSS file that sets design tokens. Put it in either place:

- `.goki/themes/<id>.css` inside the wiki — travels with the wiki
- `~/.config/goki/themes/<id>.css` — available to every wiki you serve

The `<id>` must be lowercase letters, digits and dashes. A leading comment sets the display name and whether it is light or dark:

```css
/* @name Solar Flare
   @scheme dark */
[data-theme="solar"] {
  color-scheme: dark;
  --bg: #120a05;
  --backdrop: radial-gradient(900px 600px at 0% 0%, rgba(255, 120, 40, 0.35), transparent 60%);
  --surface: rgba(30, 18, 10, 0.72);
  --glass: rgba(40, 24, 12, 0.5);
  --text: #fff4ea;
  --text-2: #d9b9a0;
  --text-3: #9a7a62;
  --accent: #ff7a2f;
  --accent-contrast: #1a0b02;
}
```

Your theme inherits every token from the shared defaults, so you only need to set what you want to change. Custom themes appear in **Appearance**, in the palette and in `goki themes`. Edits apply on reload. A complete example lives in [`examples/themes/solar.css`](https://github.com/jdtobe/goki/blob/master/examples/themes/solar.css).

## Token reference

| Token | Purpose |
| --- | --- |
| `--bg`, `--backdrop` | Page background color and image (gradients welcome) |
| `--orb-1`…`--orb-3`, `--orb-opacity`, `--orb-animation` | Optional blurred light orbs behind the panels |
| `--surface`, `--surface-2`, `--surface-3` | Main panel and raised surfaces |
| `--glass`, `--glass-border`, `--glass-blur`, `--glass-saturate` | Sidebar, menus and dialogs |
| `--panel-border`, `--panel-blur`, `--panel-radius`, `--frame-gap` | Shape of the floating panels |
| `--text`, `--text-2`, `--text-3` | Primary, secondary and tertiary text |
| `--accent`, `--accent-hover`, `--accent-contrast` | Accent color and text on top of it |
| `--accent-soft`, `--selected`, `--selected-text` | Tints for selection and highlights |
| `--link` | Link color in pages |
| `--border`, `--border-strong`, `--hover`, `--active` | Hairlines and interaction states |
| `--code-bg`, `--code-text`, `--inline-code-bg` | Code blocks |
| `--hl-keyword`, `--hl-string`, `--hl-number`, `--hl-comment`, `--hl-function`, `--hl-type`, `--hl-attr`, `--hl-meta` | Syntax highlighting |
| `--c-note`, `--c-tip`, `--c-important`, `--c-warning`, `--c-caution` | Callout colors |
| `--shadow-1`, `--shadow-2`, `--shadow-3` | Elevation for controls, panels and overlays |
| `--scrim` | Backdrop behind dialogs |
| `--font-ui`, `--font-display`, `--font-text`, `--font-heading`, `--font-mono` | Typefaces |
| `--text-size`, `--text-leading`, `--heading-weight`, `--heading-tracking` | Reading typography |

The built-in themes in [`web/css/themes.css`](https://github.com/jdtobe/goki/blob/master/web/css/themes.css) are the best reference.

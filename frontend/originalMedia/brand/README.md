# Fenster brand

Masters (SVG). The app copies live in `src/assets/`, `public/` and `pkg/notifications/logo.png`.

| File | Use |
|---|---|
| `symbol.svg` | The mark: an "F" built from three glass-pane bars, brand blue. |
| `symbol-small.svg` | Heavier cut with wider gaps for 16-32 px. |
| `symbol-black.svg`, `symbol-white.svg` | One-colour versions. |
| `lockup.svg` | Mark + wordmark, text follows `currentColor` (used in the app header). |
| `lockup-static.svg`, `lockup-white.svg` | Fixed dark text / white text versions (email, dark backgrounds). |
| `lockup-pride.svg` | June variant with striped bars (`allowIconChanges`). |
| `app-icon*.svg` | Rounded tile, square tile (apple-touch, tiles) and maskable (full bleed, safe zone). |

Colour: brand blue `#1973ff`; wordmark dark `#111827`. Wordmark: Quicksand Bold converted to outlines (Quicksand is an OFL font; confirm in `NOTICE` "To verify").
Minimum size: mark 16 px (use `symbol-small.svg` below 32 px), lockup 96 px wide. Clear space around the mark: the width of the stem (about 22 % of the mark height).
Do not: recolour the bars, stretch or rotate the mark, add effects, or use the retired llama mascot (Vikunja's artwork).
Regenerate icons: render the SVGs with `rsvg-convert -w N -h N`; `favicon.ico` = 16/32/48 px of `app-icon-small.svg`.

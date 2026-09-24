# Design

Recorded from `web/v4.html` on 2026-09-24 and approved by the user. Two earlier directions were rejected: the mission-control console (v1/v2) and the neutral modern app (v3). The world below is built from impeccable catalog worlds **operate-c-cracktro-scroller-queue** (page grammar) and **design-canon-creator-hardware-bench** (controls only).

## World

Proof as depth. Work sits on an emissive black void; trust is read as signal strength, never as a badge shelf, card or frame. Nothing on the page is enclosed — no boxes, rules, panels or dividers. Grouping comes from space, brightness and depth. Physical controls (keycaps, a knurled knob, an amber readout) are the only tactile objects, and they sit at the edge of the work, never inside it.

## Palette

Taken from the two boards, unchanged:

| Token | Value | Use |
|---|---|---|
| `--void` | `#000000` | page ground |
| `--ash` | `#1a1a1a` | empty meter track, hairline axis |
| `--dust` | `#4a4a4a` | line markers, comments, inactive labels |
| `--frost` | `#787878` | units, `±`, secondary values, captions |
| `--ivory` | `#e6e2d6` | expressions and results |
| `--signal` | `#ffb800` | proof dots, active marker, graph roots, emphasis |
| `--bone` / `--putty` | `#e6e2d9` / `#c6c2b8` | keycap faces |
| `--gunmetal` / `--charcoal` | `#3d4248` / `#171a1e` | knob body, inactive keycaps |
| `--orange` | `#ff6a00` | the one action key, and refused results |
| `--amber` | `#ffc240` | status readout only |

Lit rows go to `--lit`; receded rows drop to 55% opacity.

**Light variant** — the same world in daylight, not an inversion: paper ground `#e9e5d9`, ink `#16171a`, dust `#7c7b74`, frost `#4f4e49`, rule `#d4d0c3`. Signal darkens to `#8a5a00` for text with `#c98a00` for filled marks, and refusal to `#a63c00`, so both clear 4.5:1 on paper. The speck field lightens to `#a9a496`/`#bdb8a9`, keycaps become white-to-bone with a warm grey shadow, and the detail layer becomes a lighter plane `#f4f1e7` separated by a soft shadow rather than by a border. The amber readout stays a dark screen in both themes, because a lit readout is dark glass in daylight too. Theme follows the OS, the `light` keycap overrides it, and the choice is remembered in `localStorage`.

## Type

- `--mono: "Spline Sans Mono"` — every expression, number, unit, status word, trace and readout. 14px in the sheet, 15px for values, 31px/600 with `-.03em` for the inspected value, 11–12.5px for labels.
- `--bit: "Archivo"` (semi-expanded 112, 600/700) — the wordmark only.
- No bitmap or display face for data. Labels are uppercase with `.14em`–`.18em` tracking; values never are.

## Structure

- `.dust` — three parallax speck layers drifting behind everything (90s / 150s / 220s), off under `prefers-reduced-motion`.
- `.head` — wordmark, sheet names (current one marked `›` in signal), then keycaps (Notepad / Terminal / orange Share), the digits knob, and the amber status readout.
- `.stage > .sheet` — rows on a 3-column grid: `›` marker, expression, then value + status. Reserves right padding while the detail layer is open.
- `.st` — status as signal dots: `••••••` proven, `••••○○` estimate, `○○○○○○` held, word-only for exact/input, orange for refused.
- `.note` — interpretation and warnings under their line, label in signal caps.
- `.plot` — graph drawn straight onto the void, no frame; roots as signal squares, removable gaps as hollow squares.
- `.layer` — the detail plane sliding in from the right (budget / trace / digits / JSON); a gradient-to-black edge and a deep shadow, never a bordered panel. Bottom sheet under 860px.
- `.foot` — engine state, precision, counts.
- `.pad` — the optional keypad, docked at the bottom: a display strip (current entry, `ans`), a group row that swaps the upper key block (sci, calculus, algebra, matrix, stats, trust), and the fixed core block (digits, operators, parentheses, `del`, `C`, `ans`, `unit`, `const`, and the orange `=`). Keys are the hardware bench's keycaps; keys whose engine is not built yet take the inactive keycap treatment and explain themselves in the note line rather than failing. Hidden by default on desktop, open by default on phones, and the choice is remembered.

## Rules

- Colour is meaning: signal marks proof and emphasis, orange marks the one action and refusals, amber belongs to the readout. Everything else is neutral.
- Depth is rank: hovering a value lights it and everything downstream, and pushes the rest back.
- One motion idea: the layer's 280ms ease and the dust drift. No hover animations beyond a colour change.
- Never animate layout properties; opacity and transform only.
- Status words carry a shape (dot pattern) as well as a colour, so they survive greyscale.
- Keyboard: every row value, keycap and tab is reachable; focus ring is signal at 3px offset; Esc closes the layer; ⌘. switches view.

## Open

Remaining screens (first run, ⌘K palette, `why` panel, settings, loading) are not yet designed in this world.

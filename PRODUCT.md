# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Delegated (user asked for a recommendation): Svelte 5 + Vite + CodeMirror 6 for the web UI, static build, maths engine in pure Go compiled to WASM. Astro considered and rejected for the calculator itself (single highly interactive surface); may host future docs/landing pages. CLI (`gocalc`) and terminal REPL share the same Go core.

## Product

go-calc is a calculator and computer algebra system where every answer is either exact or proven, states how uncertain it is and why, and shows where every number came from.

Mechanism: exact rationals by default; midpoint-radius ball arithmetic for irrational results, printing only guaranteed digits; first-order (GUM) measurement uncertainty via forward-mode AD with per-input contribution budgets and correlation; a float64 shadow evaluation that warns, with location, when ordinary floating point would lose digits; provenance for every value (inputs, CODATA 2022 constants with citation, unit definitions); symbolic results (integrals, roots) verified before they are shown; guaranteed graphs that never draw across asymptotes.

## Users

Primary: daily users who come back to calculate — engineering and science students (lab reports, homework), lab and test engineers (measurement budgets), developers (quick maths, unit conversion, exact arithmetic), and everyday money maths (salary, EMI, percentages, lakh/crore formatting).

Secondary: portfolio visitors (hiring managers, engineers) judging engineering depth.

The web surface is designed for daily users first: an editor ready to type into. Demonstrating the product to newcomers must not get in the way of that.

## Principles

1. Say what I read — ambiguous input shows its interpretation; never guess silently.
2. "I don't know" beats a wrong answer — wide intervals, undecided pixels and "no closed form found" are valid outputs.
3. Two kinds of doubt, never mixed — guaranteed numeric bounds and statistical uncertainty (±) are displayed differently.
4. Every warning explains itself — one line by default, `why` for the full story.
5. Exact by default.
6. One core — front ends format, never compute.

## Confirmed product decisions

- Web layout: live notepad (input left, answers aligned right, dependent lines recalculate live) with expandable result cards (uncertainty budget, provenance trace, digit guarantee, JSON, inline graphs).
- Two view modes over the same document and engine, switchable at any time: **Notepad** (the live notepad above) and **Terminal** (CLI-style prompt/response transcript, same commands as the `gocalc` REPL). Switching never loses work; every visual direction must carry both.
- Editing is hybrid: the line being edited recalculates live, while errors and warnings appear only once the caret leaves the line.
- An optional keypad: a full scientific instrument (sci, calculus, algebra, matrix, stats, and go-calc's own trust keys) that types into the sheet rather than replacing it, so a key press and typed text are the same thing underneath. Hidden by default on desktop, shown by default on phones, remembered per person. Keys for engines not yet built are visible but inactive and say so.
- `%` means percent (Soulver-style: `500 + 10%`, `18% of 500`); modulo is `mod`. (Assumed default; user did not object.)
- Names after a number resolve as units (`5 h` is hours); bare names resolve variable → constant → unit.
- Names in unit position resolve as units before constants. A name is in unit position when it is an operand of `*`, `·` or `/` whose other side carries a unit or names one — so `h` in `60 km/h`, `s` in `9.8 m/s^2` and `mol`, `K` in `1 J/(mol·K)` are units, while bare `h` and `1/h` stay Planck's constant. Unit position runs through both sides of `*`, `·`, `/` and into the base of `^`, never through `+`, `-` or a function call. A variable that carries a unit or a declared uncertainty still wins. An interpretation note is printed only when the name is bound in the environment — a variable or a function parameter — because only then could a reasonable reader have meant the other reading; a name whose only other reading is a constant stays silent, since the neighbouring operand already carries a unit.
- An implicit product after `/` binds as a denominator when its name resolves as a unit, and keeps the documented `(1/2)·x` product reading otherwise. So `100 km / 2 h` is a speed and `100 W / 2 m^2` is `50 W/m^2`, while `1/2 x` with `x` a variable is `1.5` when `x` is 3. Because unit-ness depends on what is defined (`h = 3` makes `1/2 h` hours; `h = 12.3 ± 0.2 m` makes it the variable), the decision belongs to the evaluator, not the parser, and both readings print an interpretation note. `^` binds tighter than an implicit product throughout, so `2 m^2` is two square metres.
- A unary sign binds tighter than an implicit product, `mod` and `of`, and looser than `^`. So `-7 mod 3` is `(-7) mod 3` = 2 as it is everywhere else, `-2^2` is `-4`, and `10^-3 m` is a millimetre. Parentheses close a denominator: `1/2 h m` is `1/(2·h·m)` while `(1/2 h) m` is `(1/(2·h))·m`.
- A conversion target may be a unit expression: `to m/s`, `to km/h`, `to J/(mol·K)`, `to m^2`.
- Indian digit grouping accepted in input and available in output.

## Brand

Name: go-calc (command `gocalc`). Visual identity approved 2026-09-24 and recorded in `DESIGN.md`: work on an emissive void with nothing enclosed, trust read as signal strength, and physical controls (keycaps, knurled knob, amber readout, keypad) at the edge of the work. Built from impeccable catalog worlds `operate-c-cracktro-scroller-queue` and `design-canon-creator-hardware-bench`, in dark and light. Rejected along the way: the mission-control console (too retro, too busy) and a neutral Linear/Raycast-style app (too plain).

## Known gaps stated in the product

- A measured constant's uncertainty is reported by `info` and is **not yet propagated into calculations**. `info G` shows the CODATA standard uncertainty; `G` inside an expression behaves as an exact number, so a result depending on it carries no contribution from it. The design spec (4.3) requires that propagation; until it is built, only a declared measurement (`x = 12.3 ± 0.2`) contributes uncertainty to a result. Doing it properly needs the uncertainty table to give one identity per constant rather than a fresh one per mention, so that `G - G` stays exactly zero.

## Honesty constraints

Do not claim firsts for things Qalculate! (exact/approx, CODATA, interval mode) or Graphest (guaranteed graphing) already do. Positioning: "a calculator that shows how far to trust its answers". No invented users, benchmarks or adoption numbers.

## Open decisions

- Hosting target (GitHub Pages assumed, unconfirmed).
- Accessibility bar beyond WCAG 2.2 AA (e.g. screen-reader math output, graph sonification) — undecided.

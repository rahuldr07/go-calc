# go-calc — UI product requirements

For whoever designs the web interface. Everything here is true of the shipped engine: every example was produced by running it, not invented. Where I am unsure, it says so.

Companion documents: `PRODUCT.md` (product truth), `docs/superpowers/specs/2026-09-24-go-calc-design.md` (engineering spec). This document supersedes `DESIGN.md`, which records a visual world that did not survive contact with real content.

---

## 1. What the product is

A calculator where **every answer is either exact or proven, states how uncertain it is and why, and shows where every number came from.**

You type expressions into a document, one per line. Each line gets an answer. Unlike a normal calculator, each answer also carries its *trust*: whether it is exact arithmetic, a proven approximation with a stated number of guaranteed digits, a measurement with an uncertainty budget, or a refusal.

It is not a pocket calculator and should not look like one. The nearest relatives are Soulver, Numi and Calca (notepad calculators) and Qalculate! (exact/interval arithmetic). The differentiator is trust.

## 2. Who uses it

Primary, **daily returning users**:

- **Engineering and science students** — lab reports, homework. They care about units and about the `±`.
- **Lab and test engineers** — measurement budgets. They care about which input dominates the uncertainty.
- **Developers** — quick maths, unit conversion, exact arithmetic. They care about speed and keyboard.
- **Everyday money maths** — salary, EMI, percentages, lakh/crore. They care about none of the above and must not be taxed by it.

Secondary: portfolio visitors judging engineering depth.

**The surface is designed for daily users first: an editor ready to type into.** Demonstrating the product to newcomers must not get in the way of that. That last sentence has been violated by every attempt so far — the trust machinery kept crowding out the typing.

### Three real sessions to design against

**A. Student, lab report.** Types four lines, reads one `±`, screenshots it. Never opens anything. On screen ~90 seconds.

**B. Test engineer, measurement budget.** Types six lines, then spends five minutes on *one* answer — which input dominates, what the sensitivity is, what it looks like at k=2. Needs the budget continuously visible while editing the inputs.

**C. Everyday money.** `45000 * 12`, `18% of 500`, `1 lakh * 12`. Three lines, no units, no uncertainty. Must feel instant and uncluttered. If this person sees a provenance tree they are in the wrong product.

A and C are most of the usage. B is what makes the product worth choosing. The design must serve A and C by default and let B in on request.

## 3. The one hard rule

**The front end formats. It never computes.** Every number on screen is a field of the engine's response. No rounding, no summing a budget, no deciding a digit count in the UI. There is exactly one implementation of the arithmetic and it is in Go.

Practical consequence for design: **you cannot invent a number to make a layout work.** If you want to show "3 of 7 lines are measured", the UI may count lines, but it may not compute or reformat any value.

## 4. The engine contract

One function. The whole document goes in, every line's result comes back, once per keystroke.

**Request**
```json
{ "lines": ["h = 12.3 ± 0.2 m", "h * 2"], "digits": 12, "angle": "rad" }
```

**Response** — real output, `h = 12.3 ± 0.2 m`, `w = 4.5 ± 0.1 m`, `h * w`:
```json
{ "schema": "go-calc.sheet/1",
  "lines": [
    { "n": 2, "text": "55.4 ± 1.5 m·m", "status": "estimate", "digits": 4,
      "uncertainty": {
        "value": "55.4", "reported": "1.5", "standard": 1.5241062955056648,
        "budget": [
          { "name": "w", "share": 0.6512979465323518, "sigma": 0.1, "sensitivity": 12.3 },
          { "name": "h", "share": 0.34870205346764815, "sigma": 0.2, "sensitivity": 4.5 }
        ] },
      "trace": [ { "id": 9, "kind": "operation", "label": "*", "children": [4, 8] },
                 { "id": 4, "kind": "input", "label": "h", "detail": "12.3 ± 0.2 m", "line": 1 } ] } ] }
```

Every field the UI may show, and nothing else exists:

| Field | Meaning |
|---|---|
| `n` | zero-based document line |
| `text` | the answer, already formatted — **render verbatim** |
| `status` | `exact` · `proven` · `estimate` · `held` · `error` |
| `digits` | how many digits are guaranteed |
| `capped` | the precision cap was reached |
| `stopped` | why the retry stopped |
| `qualifiers[]` | e.g. `"zero to 2465 decimal places, the precision available"` |
| `error`, `column` | refusal text and where |
| `uncertainty` | `value`, `reported`, `standard`, `coverage`, `factor`, `budget[]`, `nonlinear` |
| `budget[]` | per input: `name`, `share` (0–1), `sigma`, `sensitivity` |
| `notes[]` | `source` (`parser`\|`evaluator`), `text`, `column` |
| `warning` | the float64 disagreement, one sentence |
| `trace[]` | provenance nodes: `id`, `kind`, `label`, `detail`, `children[]`, `line` |

Responses can arrive out of order; the client drops stale ones. A line can error while its neighbours are fine — **an errored line is the normal state of a live notepad, not an exception.**

## 5. Everything the UI must be able to display

Real engine output. Design against these exact strings — they are the hard cases.

**A plain answer**
```
18% of 500                    90                                  exact
sqrt(2)                       1.41421356237                       proven
```

**A two-part value** — exact form *and* an approximation, both shown:
```
1/3                           1/3  ≈ 0.333333333333               exact
```

**A measurement**
```
h = 12.3 ± 0.2 m              12.30 ± 0.20 m                      estimate
area = h * w                  55.4 ± 1.5 m·m                      estimate
expand(area, k=2)             55.4 ± 3.0 m·m (k = 2)              estimate
```
Note `m·m`, not `m²`. The engine does not collapse unit powers. Do not "fix" this in the UI — see §10.

**A proven zero, with a qualifier**
```
sqrt(2) - sqrt(2)             0
                              qualifier: zero to 2465 decimal places, the precision available
```

**A refusal** — carries line and column:
```
2 +                           8:4: unexpected end of input        error
60 km/h to m/s                cannot convert dimensionless () to length/time (m/s)
```

**A trust warning** — one sentence, names the position:
```
(1e16 + 1) - 1e16             1                                   exact
  warning: float64 would answer 0 here, agreeing with the proven value to no
           significant digits; the worst loss is 53 bits at the - at 6:12
```

**Two notes on one line.** This is real and the layout must survive it. In a document where `h` is already a measurement, `100 km / 2 h` produces:
```
100 km / 2 h                  615000 ± 10000 m·m                  estimate
  note: read as the variable h, not the unit h (time); the variable carries a
        measured uncertainty
  note: read as ((100 km)/2) h
```

**A very long answer.** `2^10000` is **3,011 digits on one line**. This has broken every layout so far. The number must not widen the page; the full value has to be reachable somewhere.

**A budget** — the `area` line above:
```
w   ████████████████░░░░░░░░   65.1 %     u(w) 0.1   ∂/∂w 12.3
h   ████████░░░░░░░░░░░░░░░░   34.9 %     u(h) 0.2   ∂/∂h 4.5
```
Bar length must be the exact share. It is data, not decoration.

**A provenance tree**
```
*
├─ h   12.3 ± 0.2 m     input, line 1
└─ w   4.5 ± 0.1 m      input, line 2
```
Nodes can be `input`, `operation`, `unit`, `constant`, `literal`, `variable`. A constant carries its CODATA citation as `detail`.

## 6. Every state

Design each of these; the ones marked ✱ have been got wrong before.

| State | What it looks like |
|---|---|
| Empty document, first visit ✱ | No demo content. An editor ready to type into. |
| Typing, line incomplete ✱ | Line under the caret recalculates live; **its error stays hidden until the caret leaves.** Incomplete shows nothing, not an error. |
| Line settled | Expression, answer, status. |
| Line refused | Error text and column, neighbours unaffected. |
| Line with 1–3 notes ✱ | Notes stack under the line in a stable order. |
| Line with a warning | Distinct from a note — this is the honesty signal. |
| Long answer ✱ | Must not widen the page; full value reachable. |
| Measured answer | `±` and a budget available. |
| Inspecting a line | Budget, trace, digit guarantee, raw JSON. |
| Engine loading | ~1s on a cold load. |
| Engine failed ✱ | Must say so loudly — not a dead editor. |
| Keypad open / closed | Hidden by default on desktop, open on phone. |
| Terminal view | Transcript over the same document. |
| Theme switch | Dark and light at minimum. |

## 7. Surfaces

1. **Notepad** — the primary surface. A document you type into, each line answered.
2. **Terminal** — a REPL transcript over *the same document*. Switching never loses work. Both views are required; a design that only solves one is incomplete.
3. **Inspector** — budget, trace, digits, JSON for one chosen line.
4. **Keypad** — optional scientific instrument that *types into the sheet*, so a key press and typed text produce the same line. Hidden by default on desktop, open on phone, remembered. Keys whose engine is not built (calculus, algebra, matrix, stats) are visible but inactive and say so — roughly **37 of 80 keys are inactive**, which is a real design problem, not an edge case.
5. **Theme** — see §9.

Not yet designed, and not required for a first pass: first-run guidance, command palette, settings, sharing, graphs.

## 8. Interaction

- Type to compute. No run button.
- Hybrid editing: the caret's line is live; errors and notes appear when the caret leaves.
- Clicking an answer inspects that line.
- `⌘.` switches notepad/terminal. `Esc` closes the inspector.
- Digits are adjustable (1–1000, default 12); changing it re-renders every line.
- Angle mode rad/deg.
- Everything reachable by keyboard. WCAG 2.2 AA.

## 9. Constraints

- **Svelte 5 + Vite**, static build, no router. Engine is Go → WebAssembly in a Web Worker.
- **WASM is 1.33 MiB gzipped** and loads on first paint. Budget: JS ≤ 300 KiB gzipped (currently 136 KiB).
- Recalculation of a 100-line document is ~2 ms. The UI can afford to re-render freely.
- **Dark and light both required.** A theme system exists with 8 worlds and a style axis (font, per-role corner radius, depth) — `web/src/lib/theme/themes.ts`. Reuse or replace it, but corners must stay **per-role**: a ruled column of figures and a moulded keycap want different answers.
- Responsive to phone. The keypad is the default surface on phones.
- `prefers-reduced-motion` respected.

## 10. What has been tried, and why it failed

Four attempts. Read this before designing.

**v1/v2 — mission-control console.** Rejected: too retro, too busy.

**v3 — neutral Linear/Raycast-style app.** Rejected: too plain.

**v4 — "emissive void, nothing enclosed."** Chosen from a catalog and approved *as a static mockup*, then rejected once built. Three reasons, and they are the important lessons:

1. **It was approved as a picture, not a layout.** The mockup was hand-authored HTML with fake content, composed at one width. It had no answer for real content at other sizes.
2. **"Nothing enclosed — no boxes, rules, panels or dividers" is a poster law, not an app law.** With nothing to land in, answers stranded: an expression ending at x=180 with its own answer at x=820, and 770px of dead space beyond. For a dense scannable tool, removing all structure removes the alignment the task depends on.
3. **The page had no composition.** Content occupied about a third; the rest was leftover doing no work. Worse, the product's differentiator — the budget and trace — was hidden behind a click *while the page had a hole in it*.

### Measured evidence from the last attempt

Screenshot: `.playwright-mcp/L-07-final.png`, 1440x900, six lines of real content. These are ink extents, not element boxes.

| Fact | Number |
|---|---|
| `h * w` expression ends | x = 171 |
| its answer begins | x = 664 |
| **void inside one row** | **493 px**, over half the 782 px work column |
| the same gap at 1024 px wide | **653 px** — it grows as the window narrows |
| content ink ends | y = 293 |
| footer begins | y = 875 |
| **empty band below the work** | **1200 x 562 px**, about 62% of the page |

Four more faults visible in that render, all of which a new design should be immune to by construction:

- **The note is orphaned.** `READ AS (100 km)/(2 h)` renders under the expression on the far left while the value it describes is 500 px away on the right. A note belongs to a *value*, not to a line's left edge.
- **The inspector reserves 360 px to report an absence** — "uncertainty · none declared — this answer is exact". Dedicating a third of the width to saying there is nothing to say is worse than leaving it blank.
- **A phantom row.** The document's trailing empty line renders a trust badge, `○○○○○○ HELD`, for a line that does not exist.
- **Status marks collide with their word**: `••••○○ESTIMATE`.

**The structural lesson.** Fixing the page-level hole by adding a second pane immediately exposed that **the row is the broken unit, not the page.** Right-aligning a value to a far edge only reads as one row in a narrow column; at any real width the expression and its answer stop being connected. Any design that keeps "expression hard left, value hard right" must state what holds the two ends together — a leader, a tighter measure, a shared background, proximity — or abandon the arrangement.

**Things that are engine behaviour, not UI bugs — do not design around them:**
- `m·m` instead of `m²`. Units are not collapsed to powers.
- `1/3  ≈ 0.333…` — exact form and approximation both shown.
- A measured constant's uncertainty is reported by `info` but **not propagated** into calculations (a documented gap).
- `2 hbar` means two hectobars, not two reduced Planck constants.

## 11. What I need back

Whatever form suits you — Figma, HTML, screenshots, a reference you want matched:

1. **Notepad at three widths** (phone, ~1100px, ~1440px) with the real content from §5 — including the two-note line, the refusal, and the 3,011-digit answer.
2. **Where the inspector lives** and how it behaves when nothing is selected.
3. **Terminal view.**
4. **Keypad**, including how 37 inactive keys read honestly without looking broken.
5. **How trust is signalled** on a line — the five statuses, distinguishable in greyscale.

If it is easier to point at an existing product and say "like that, but…", that is genuinely more useful to me than a spec.

## 12. Open questions

- First-visit state: empty, demo, or guided? Current assumption: empty.
- Does the inspector earn permanent space, or open on demand?
- Is the keypad worth its cost on desktop given most keys are inactive?
- Should a long answer truncate, wrap, or move?
- How many themes are worth shipping? Eight exist, but measured page backgrounds show only about four are distinct: `void` and `ember` differ by no more than 9 per channel, and `sepia`, `paper` and `slate` are indistinguishable side by side without reading the label. Either pull them apart or ship fewer.
- Does a per-role corner system earn its complexity? It was built (`--radius-sheet` / `--radius-control` / `--radius-surface`) on the argument that a ruled column and a moulded keycap want different answers. That argument still holds, but nothing in the last render made the difference visible.

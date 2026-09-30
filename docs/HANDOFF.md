# go-calc — session handoff

Written 2026-09-30, compressing a session that ran from the first PRD discussion to a finished engine and four rejected UI attempts. Every path, count and quotation here was verified against the repo, not recalled.

---

## Standing constraints — never compress these away

From the user, verbatim, and binding on all future work:

> "remeber no comments no ai tags or coauthors add it somewhere so u will remember and push to main no branches"

1. **No comments in code.** Names and structure carry the meaning. The only `//` in the Go tree are `//go:build` and `//go:embed`.
2. **No AI attribution in commits.** No "Generated with", no `Co-Authored-By`.
3. **Commit and push to `main`. No branches.**

Also recorded in `CLAUDE.md` and in `~/.claude/projects/-home-rahul-projects-go-calc/memory/`.

## Session intent

Build **go-calc**: a calculator and CAS where every answer is either exact or proven, states how uncertain it is and why, and shows where every number came from. Go engine, WASM, web front end.

---

## Where things stand

| | State |
|---|---|
| **Engine (Go)** | Done. 8,702 lines of source, 9,942 of tests, 476 test functions, 13 packages. Tests, `-race`, `vet`, `gofmt` all green. |
| **CLI + REPL** | Done. |
| **WASM bridge** | Done. 1.33 MiB gzipped. |
| **Web front end** | Builds and works, **UI rejected four times**. This is the open problem. |
| **Published** | **No.** The repo has no git remote. Nothing has ever been pushed. |
| **HEAD** | `9cd3b82`, 69 commits. |

All five performance budgets met: typical expression 0.19–0.32 ms (<1 ms), 1000 digits 9.7–17 ms (<1 s), 100-line recalc 1.8–4.4 ms (<10 ms), CLI cold start 1.6–3.4 ms (<50 ms), WASM 1.33 MiB (<4 MiB).

## Files

**Documents (read these first)**
- `docs/UI-PRD.md` — **the live document.** UI requirements written against real engine output, with a measured post-mortem of every failed attempt. Supersedes `DESIGN.md`.
- `PRODUCT.md` — product truth, confirmed decisions, known gaps.
- `docs/superpowers/specs/2026-09-24-go-calc-design.md` — phase-1 engineering spec.
- `DESIGN.md` — **stale.** Records the visual world that did not survive contact with real content. The PRD says so explicitly.
- `docs/superpowers/plans/2026-09-28-web-front-end.md` — the 11-task front-end plan; tasks 1–9 built, 10 (dependency lighting) and 11 (CI/deploy) partly.
- `.superpowers/sdd/2026-09-24-proven-numerics/progress.md` — gitignored ledger holding ~60 `Ruling:` lines from the engine build.

**Go packages** — `cmd/gocalc`, and under `internal/`: `lex parse num elem units consts unc prov value eval doc format`. Dependency rule: lower never imports higher; `eval` must not import `format`.

**Web** — 20 files under `web/src/`, listed in the plan. Engine boundary is `web/src/lib/engine/{schema,worker,client}.ts`.

## Uncommitted work

Eight files, deliberately left alone pending a decision:

```
 M web/src/App.svelte                          two-pane composition
 M web/src/app.css
 M web/src/lib/components/DetailLayer.svelte   docked inspector + empty state
 M web/src/lib/components/Head.svelte
 M web/src/lib/editor/notepad.ts
 M web/src/lib/state/view.svelte.ts
?? web/src/lib/components/ThemeStudio.svelte   theme picker
?? web/src/lib/theme/                          themes.ts + theme.svelte.ts
```

This is the 8-world theme system and the two-pane layout. It may be discarded when a new design lands. **Three known bugs in it, unfixed on purpose:**

1. Corner presets are inert outside the theme dialog — CSS says `.k { border-radius: var(--radius-control) }` but the header buttons use class `key`, not `k`.
2. The dialog's description hardcodes `<b>void</b>`, so it claims the wrong active theme.
3. The detail plane's hero value is clipped; the long-value fix in `3a6e81a` covered the sheet and terminal, not that element.

## Decisions that must not be re-litigated

- **`Number` is a tagged struct, not an interface.** Go was chosen for cold start and a cgo-free WASM build, not bignum throughput; no interface boxing on the exact path.
- **Provenance is always on** — one `int32` per value, an arena per evaluation.
- **Sensitivities are `float64`**, not `big.Float` — uncertainty reports two significant digits.
- **The front end formats, never computes.** Every number on screen is a field of the engine response.
- **The engine is stateless per call.** Whole document in, every line's result out, once per keystroke.
- **`%` is percent, `mod` is modulo.**
- **A unary sign binds tighter than an implicit product, `mod` and `of`, looser than `^`** — so `-7 mod 3` is 2, `10^-3 m` is a millimetre.
- **Parentheses close a denominator whatever they hold.**
- **An interpretation note shows the whole line, once, in the notation typed** — it must read back into the answer it describes. Guarded by a generated round-trip test.
- **Constants may carry a defining formula**; `hbar` is `h / (2 pi)` evaluated to working precision, because an irrational constant is proven, not exact.

## Engine quality history

The engine went through **7 fix waves and 8 adversarial reviews**. Findings worth remembering:

- Two **wrong physical constants** shipped and survived multiple reviews: the electron mass was 10× too small, and `R` held CODATA's rounded display so `N_A · k_B − R` was not zero. The checksum pinned the table against *change*, not against being *wrong*. Every constant is now pinned to its published value independently.
- The last three reviews swept 274k, 33k and 20k generated expressions and found **zero wrong answers**. Every late defect was in how the calculator *words* what it read, not what it computes.
- The invariant that finally caught that class: **every note a line emits is re-parsed and re-evaluated in the line's own environment and must give the same number and dimension.** Generated, not a fixed list.

**Known shipped gap:** a measured constant's uncertainty is reported by `info` but **not propagated** into calculations. Documented in `PRODUCT.md` and `README.md`.

---

## The open problem: the UI

Four attempts, four rejections.

| Attempt | Verdict |
|---|---|
| v1/v2 mission-control console | "too retro, too busy" |
| v3 neutral Linear/Raycast app | "still not good" / too plain |
| v4 emissive void (chosen from the impeccable catalog, worlds #2 + #6) | approved **as a mockup**, then "fully bad and worst verv very bad" once built |
| Two-pane recomposition | "the ui placing itself is not good" |

**Measured evidence from the last render** (`.playwright-mcp/L-07-final.png`, 1440×900):

- `h * w` expression ends at x=171; its answer begins at x=664 — a **493 px void inside one row**, over half the 782 px column. At 1024 px it grows to 653 px.
- Content ink ends at y=293, footer at y=875 — a **1200×562 empty band**, about 62% of the page.
- The note `READ AS (100 km)/(2 h)` renders on the far left while the value it describes is 500 px away.
- The inspector reserved 360 px to say "none declared — this answer is exact".
- A phantom trailing row rendered `○○○○○○ HELD` for a line that does not exist.

**The structural lesson:** fixing the page-level hole by adding a second pane exposed that **the row is the broken unit, not the page.** Right-aligning a value to a far edge only reads as one row in a narrow column.

**My process failure, which matters more than any of the above:** I verified *behaviour* — does it evaluate, does the panel open, do the tests pass — and called that verified, while never looking at the rendered page until the user told me it was bad. Four times. **Look at the screenshot before claiming a UI works.**

## Next steps

1. **The user is bringing a UI design back.** They asked for the PRD to design against; it is committed. Wait for it rather than attempting a fifth redesign unprompted.
2. Decide the fate of the uncommitted theme/layout work: commit as a checkpoint, revert to `3a6e81a`, or leave.
3. **Publishing has never been discussed beyond noting it.** Creating `rahuldr07/go-calc` and pushing is outward-facing and unconfirmed — ask first.
4. Deferred by ruling: propagating measured-constant uncertainty (needs an input-ID cache on `unc.Table`; a naive fix makes `G - G` nonzero).
5. Not built: dependency lighting (needs the sheet schema to carry each line's dependents), sharing, command palette, settings, graphs.

## Cost note

This session ran to roughly **$220**, the bulk of it eight adversarial review subagents on the engine and four browser-QA rounds on the UI. The reviews were worth it — they caught two wrong physical constants and several silent value changes. The UI rounds were not, because they verified the wrong thing.

## Operational gotchas

- **Never run two browser-QA agents at once.** They share a Playwright browser and corrupt each other's captures; one run produced screenshots that did not match the state the agent had just set.
- A killed session corrupted `.git` once — 17 zero-length objects and a bad HEAD. Recovered by pointing `main` at the last complete commit, deleting the empty objects, and rebuilding the index. Back up `.git` before any such surgery.
- `vite preview` must be started detached (`setsid … < /dev/null &`) or the shell call hangs.

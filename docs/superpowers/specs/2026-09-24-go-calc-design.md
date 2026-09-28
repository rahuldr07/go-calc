# go-calc — design spec

Date: 2026-09-24 · Owner: Dileesh Rahul · Status: for review

A calculator and computer algebra system in Go where every answer is either exact or proven, states how uncertain it is and why, and shows where every number came from. Full product context is in [PRODUCT.md](../../../PRODUCT.md); the visual world is in [DESIGN.md](../../../DESIGN.md). This spec covers what gets built, in what order, and how it is verified.

## 1. Scope of this spec

Phase 1 only: the trust core plus both front ends, deployed. Symbolic maths, graphing and Monte Carlo are specified in the PRD and are out of scope here except where the architecture must leave room for them.

Out of scope: the full Risch algorithm, live currency rates, a desktop GUI, loops or general control flow.

## 2. Architecture

Pure Go, standard library first, no cgo (cgo would break the WASM build). Module `github.com/rahuldr07/go-calc`.

```
cmd/gocalc/        CLI: one-shot, run file, REPL
internal/lex/      tokens, positions, Indian digit grouping
internal/parse/    Pratt parser → AST, interpretation notes
internal/num/      exact (big.Rat) + ball arithmetic (mid big.Float, radius mag)
internal/elem/     rigorous elementary functions on balls
internal/units/    dimension vectors, unit table, affine temperature
internal/consts/   embedded CODATA 2022 table + citations (go:embed)
internal/unc/      float64 sensitivities, covariance, contributions
internal/value/    Value = Num + Dim + Kind + Sens + Prov
internal/eval/     evaluator, environment, precision retry, float64 shadow
internal/prov/     dependency graph, trace, JSON export
internal/doc/      documents: line graph, incremental re-evaluation
internal/format/   text + JSON rendering, digit-guarantee logic
wasm/              GOOS=js GOARCH=wasm entry + JS bridge
web/               Svelte 5 + Vite + CodeMirror 6 front end
```

Dependency rule: lower packages never import higher ones. `num` knows nothing of units; `units` knows nothing of uncertainty; `value` composes them. Front ends import only `eval`, `doc` and `format`.

### Core type

```go
type Value struct {
    Num  Number        // Exact(*big.Rat) | Ball{Mid *big.Float; Rad Mag} | Interval
    Dim  units.Dim     // rational exponent vector
    Kind units.Kind    // Absolute | Difference (temperature)
    Sens unc.Sens      // map[InputID]float64, nil when certain
    Prov prov.NodeID
}
```

Sensitivities are `float64`, not `big.Float`: uncertainty is reported to two significant digits, so arbitrary precision buys nothing there. Arbitrary precision is confined to the ball midpoint.

### Architecture decisions (2026-09-24)

1. **`Number` is a tagged struct, not an interface.** Go was chosen for cold start (5 ms measured), a single static binary and a cgo-free WASM build — not for raw big-number throughput, where GMP-backed languages win. Since we gave up peak arithmetic speed by staying pure Go, we keep the shape that costs nothing: no interface boxing, no allocation per operation on the exact path. Correctness is held by the containment property tests either way.
2. **Provenance is always on.** A node id is one `int32` per value and an arena per evaluation, so `trace` is free rather than a mode to enable.
3. **Web front end is plain Vite + Svelte 5**, one static page, no router. Docs and a landing page can be a separate Astro site later.
4. **One repository.** `web/` carries its own `package.json`; CI builds the WASM module and the front end together so they cannot drift.
5. **Performance budget, enforced by benchmarks in CI** and published as measured numbers in the README: typical 12-digit expression under 1 ms; 1,000 digits of `exp(pi·sqrt(163))` under 1 s; recalculating a 100-line document under 10 ms; CLI cold start under 50 ms; WASM bundle under 4 MB compressed.
6. **The browser crosses the boundary once per keystroke**, not once per line: the whole document goes into one `goCalcEval` call and every line's result comes back together, inside a Web Worker so a deep precision retry never freezes the page.

## 3. Decisions made during design

These differ from the PRD and win over it.

1. **`%` is percent, not modulo.** `500 + 10%` is 550, `18% of 500` is 90, `5 %` is 0.05. Modulo is the `mod` keyword.
2. **Unit position beats variable shadowing.** A name straight after a number is a unit (`5 h` is five hours even when `h` is a variable); a bare name resolves variable → constant → unit. Without this rule the PRD's own example breaks `h`, `g` and `t`.
3. **Two view modes over one document:** Notepad and Terminal, switchable at any time, sharing the engine and the document. Switching never loses work.
4. **Sensitivities are float64** (see above).
5. **The rounding-boundary case must terminate.** The digit printer stops at a precision cap and prints one digit fewer rather than looping when a value sits exactly on a rounding boundary.
6. **Web front end:** Svelte 5 + Vite + CodeMirror 6, static build. Astro was considered and rejected for a single interactive surface.
7. **Two themes.** The approved world ships dark (emissive void) and light (the same materials in daylight, with signal and refusal darkened for contrast). Theme follows the OS and is overridable, and the choice persists.

### 3.1 Interaction decisions

8. **Hybrid editing.** The line under the caret recalculates live as it parses; errors, warnings and interpretation notes appear only once the caret leaves the line. Incomplete lines show nothing rather than an error.
9. **The keypad is an optional instrument, not a mode.** It types into the sheet, so a key press and typed text produce the same line, with the same proof, budget and trace. A display strip shows the current entry and `ans`. A group row swaps the upper block — sci, calculus, algebra, matrix, stats, and go-calc's own trust keys (`±`, `corr`, `k=2`, `digits`, `trace`, `why`, `plot`) — over a fixed core block. Hidden by default on desktop, open by default on phones, remembered per person. Keys whose engine is not built yet render inactive and explain themselves rather than failing; they are never hidden, so the instrument is honest about its own range.

### 3.2 Still open

Each needs a decision before the front end is built: first-visit state (empty sheet, demo sheet or guided start); how much lives in the ⌘K palette; how very long values behave in the value column; whether `why` earns a full surface beyond the note plus detail layer; where settings live beyond the knob; sharing (URL-encoded sheet, file, or both); phone specifics; and the accessibility bar beyond WCAG AA.

## 4. What phase 1 ships

### 4.1 Language
Integers, decimals (exact rationals), scientific, `1_000_000`, Indian grouping, hex/bin/oct, fractions. `+ - * / ^ !`, `mod`, percent, unary minus; `^` right-associative, `-2^2 = -4`. A unary sign binds tighter than an implicit product, `mod` and `of` and looser than `^`, so `-7 mod 3` is 2 and `10^-3 m` is a millimetre. Implicit multiplication with an interpretation note whenever it meets `/`. Functions need parentheses. Variables and user functions. `±`, `12.3(2)`, `[1.2 .. 1.4]`. Units after numbers, `to`/`in` conversion. Comments, commands (`why`, `trace`, `info`, `digits`, `deg`, `rad`). Multi-line documents with line and column on every error. The parser never panics.

### 4.2 Numbers
Exact rationals wherever possible, including exact roots of perfect powers. Balls elsewhere: midpoint at p bits, radius always rounded up, every operation guaranteed to contain the true value. Rigorous `sqrt exp log log10 log2 sin cos tan asin acos atan atan2 sinh cosh tanh pow`, `pi`, `e`, by argument reduction plus series with an explicit remainder bound. Precision retry doubles p up to 8192 bits, then prints only what is guaranteed and says so. Comparisons return undecided rather than false when two balls overlap. Domain errors are proven or flagged ("divisor may be zero").

### 4.3 Units and constants
Rational exponent vectors over the seven SI base dimensions plus bit and angle. Exact rational scale factors. Affine temperature with absolute and difference kinds. SI with prefixes, imperial, US customary, time, data, angle, pressure, energy, power, plus lakh and crore as dimensionless multipliers. Dimension errors name both dimensions in words. CODATA 2022 embedded at build time with value, uncertainty, unit, exactness flag and citation; `info G` prints the citation. Constant correlations are out of scope and documented as such.

### 4.4 Uncertainty
`±` declares a standard uncertainty; each uncertain input gets an identity. Propagation is first-order GUM by forward-mode differentiation, so `x - x` is exactly zero. `corr(a, b) = 0.8` adds covariance. Results show the combined uncertainty to two significant digits plus a contribution table. `expand(x, k=2)` and `95%`. A nonlinearity check warns when first-order is unreliable.

### 4.5 Trust reporting
Every expression also runs in a float64 shadow. When the shadow has fewer than six correct digits, a one-line warning names the operation that lost them, estimated per add/sub node as `log2(max(|a|,|b|) / |a ± b|)`; `why` prints the full story. Every value links to a provenance node; `trace x` prints the tree and `--json` exports the graph. Changing a document line re-evaluates exactly its dependents.

### 4.6 Front ends
- **CLI:** `gocalc "expr"`, `gocalc run sheet.calc`, `--json` (stable, versioned `go-calc.result/1`), non-zero exit on error.
- **REPL:** history, completion for names, units and constants, coloured output.
- **Web:** the approved world in `web/v4.html`, rebuilt in Svelte, driving the WASM core. Notepad and Terminal views, the optional keypad, the detail layer (budget, trace, digits, JSON), dependency lighting, keycap controls, digits knob, status readout, dark and light.

## 5. Build order

Each step ends deployable. The web build comes early so "go live" is never blocked.

1. **Skeleton and CI.** Module, `go vet`, staticcheck, tests with `-race`, fuzz smoke, GitHub Actions.
2. **Lexer and parser.** Fuzzed to no panics; parse → print → parse round-trips.
3. **Exact tier.** `big.Rat` arithmetic, formatting, percent, Indian grouping.
4. **Units and constants.** Dimension algebra, conversion, affine temperature, embedded CODATA with a checksum test.
5. **Balls.** Arithmetic with containment property tests against 4× precision, then `sqrt exp log pi`, then trig and the rest.
6. **Precision retry and the digit printer,** including the rounding-boundary stop.
7. **Uncertainty.** Forward-mode sensitivities, covariance, contributions, expansion, nonlinearity check, validated against worked GUM examples.
8. **Provenance and the float64 shadow.** `trace`, `why`, JSON export.
9. **CLI and REPL.**
10. **WASM bridge and the Svelte front end.** Measure the bundle; TinyGo only if the standard build is too large.
11. **Documents.** Incremental re-evaluation, `.calc` files, share links.
12. **Deploy.** Static hosting, README with a GIF and measured numbers.

## 6. Verification

| Layer | Method | Bar |
|---|---|---|
| Lexer, parser | Go fuzzing, round-trip property | No panics; round-trip holds |
| Balls | Every operation checked against 4× precision on random inputs | Zero containment failures in 10⁷ cases per operation |
| Elementary functions | Published reference digits, identities on balls, self-comparison at higher precision | Every guaranteed digit correct |
| Uncertainty | Worked GUM and NIST examples; `x - x` is zero | Matches published budgets |
| Units | Table-driven conversions, affine temperature cases | Whole unit table covered |
| Constants | Checksum against the embedded source file | Pinned per release |
| Precision warnings | Catalogue of cancellation cases plus a benign control set | Warns on all, no false positives |
| Front ends | Golden transcripts for CLI and REPL; the web build screenshotted at desktop and mobile | CI on every push |

The containment tests are the product's whole claim and are written with the code they check, never after.

## 7. Risks

| Risk | Mitigation |
|---|---|
| Hand-written ball arithmetic produces wrong "guaranteed" digits | Property tests first, a small audited core, Arb's published error formulas, no optimisation before tests |
| Interval blow-up over long chains | Exact tier first, sensitivities rather than intervals for uncertainty, honest "undecided" |
| WASM bundle too big or slow | Measured at step 10, TinyGo as fallback, lazy loading |
| Scope creep into symbolic work | Phase 1 is this spec; symbolic maths starts only when every bar above is met |

## 8. Open questions

1. **Name and hosting.** `go-calc` as the repo name is settled; the hosting target (GitHub Pages assumed) is not confirmed.
2. **CODATA source.** Confirm the NIST machine-readable table is stable enough to embed, and record its URL and checksum.
3. **Remaining screens.** First run, the ⌘K palette, the `why` panel, settings and loading are not yet designed in the approved world.
4. **Accessibility bar.** WCAG 2.2 AA is assumed; screen-reader maths output and graph sonification are undecided.

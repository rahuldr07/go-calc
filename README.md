# go-calc

A calculator that shows how far to trust its answers.

Every answer is exact or proven. `0.1 + 0.2` is `0.3`, not `0.30000000000000004`. `(1e16 + 1) - 1e16` is `1`, where float64 gives `0`. What cannot be written exactly is computed as a ball — a midpoint and a rigorous radius — and only the digits the ball guarantees are printed. Measurements carry their uncertainty and say which input dominates it. Units carry dimensions, so `5 m + 3 s` is an error that names both. Every number can say where it came from.

## Status

The exact core, ball arithmetic, measurement uncertainty, provenance, documents, the REPL, the sheet API and the WASM build are done and measured. The numbers below are measured, not claimed.

```
$ gocalc "0.1 + 0.2"
0.3

$ gocalc "sqrt(2)"
1.41421356237

$ gocalc "exp(pi * sqrt(163))"
2.62537412641e+17

$ gocalc "sqrt(2) - sqrt(2)"
0
  zero to 2465 decimal places, the precision available

$ gocalc "(1e16 + 1) - 1e16"
1
  float64 would answer 0 here, agreeing with the proven value to no significant
  digits; the worst loss is 53 bits at the - at 1:12

$ gocalc "3 km to m"
3000 m

$ gocalc "5 m + 3 s"
cannot add length (m) and time (s)

$ gocalc "exp(1000000)"
1:1: too large to work with: writing this number would take more than 10000 digits
```

A document carries measurements and reports what dominates the uncertainty:

```
$ cat sheet.calc
h = 12.3 ± 0.2 m
w = 4.5 ± 0.1 m
h * w

$ gocalc run sheet.calc
h = 12.3 ± 0.2 m
  = 12.30 ± 0.20 m
w = 4.5 ± 0.1 m
  = 4.50 ± 0.10 m
h * w
  = 55.4 ± 1.5 m·m
```

`gocalc --json run sheet.calc` answers with one `go-calc.sheet/1` document for the whole file: a status, the proven digits, the qualifier that makes those digits honest, the contribution budget — `w` at 65% of the variance, `h` at 35% — and the provenance tree for every line.

## Install

```
go install github.com/rahuldr07/go-calc/cmd/gocalc@latest
```

## Use

```
gocalc "expression"
gocalc "-2^2"
gocalc --json "expression"
gocalc --indian "18,50,000 / 12"
gocalc run sheet.calc
gocalc --json run sheet.calc
gocalc trace "expression"
gocalc why "expression"
gocalc repl
```

Options are the two-dashed names only, so a leading `-` begins an expression: `gocalc "-2^2"` answers `-4`. `--` ends the options.

`--json` prints the versioned `go-calc.result/1` schema for one expression and the `go-calc.sheet/1` document for `run`; the sheet API answers under `go-calc.sheet/1` too. `trace`, `why` and `repl` do not produce JSON and say so rather than dropping the flag. Errors exit non-zero.

## What it does today

**Exact where it can be.** Decimals are rationals, so `0.1` is exactly 1/10. Percent (`500 + 10%`), `mod`, factorial, exact roots of perfect powers, `gcd`, `lcm`, `floor`, `ceil`, `round`, `isprime`. Numbers as `1_000`, `18,50,000`, `0xFF`, `0b1010`, `0o17`, `1.5e-3`; lakh and crore.

**Proven where it cannot.** Ball arithmetic with rigorous radii behind `sqrt`, `exp`, `ln`, `log`, `log2`, `log10`, `root`, the trigonometric and hyperbolic functions, `atan2` and `pi`. Precision doubles until the requested digits are guaranteed or a stated stop is reached: the precision cap, an uncertain input, or a deadline. Every printed digit is one the ball guarantees — `digits 30` in the REPL prints thirty proven digits, not thirty digits of hope. Where fewer digits are guaranteed than asked for, the qualifier says so, on the printer as a second line and in the sheet API as the `qualifiers` field beside `text`. A result that would take more than 10,000 digits to write is refused by name rather than ground out.

**Measurement uncertainty.** `h = 12.3 ± 0.2 m` declares a measurement. Uncertainty propagates through first and second derivatives, so nonlinearity is detected rather than assumed away, and the result carries a contribution budget naming each input's share of the variance. `expand(x, k)` widens to a coverage factor. `corr(a, b) = 0.5` declares a correlation, and the budget then carries the covariance term and says so.

**Provenance.** `gocalc trace` prints the tree of where every number in a result came from — literals, constants, named inputs and the line each was defined on. `--json` exports the same tree.

**The float64 shadow.** Every expression is also evaluated in float64. `gocalc why` shows the proven answer beside the float64 one, how many digits they agree to, how many of float64's printed digits are wrong, and which operator lost the bits.

**Units.** SI prefixes, binary prefixes, imperial, time, force, energy, power and pressure, converted through exact rational factors (`inch` is exactly 127/5000 m). Affine temperature: `20 degC to K` is `293.15 K`, `20 degC + 5 K` is `25 degC`, and adding two absolute temperatures is refused. Compound units: `60 km/h`, `9.8 m/s^2`, `1 J/(mol·K)`, `100 W / 2 m^2` (`50 W/m^2`). `^` binds tighter than an implicit product, so `2 m^2` is two square metres, and a numeral-bearing term after `/` stays in the denominator; where that grouping changes the value the calculator prints the reading it took. A name straight after a number is a unit, so `5 h` is five hours even when `h` is a variable — and the calculator says so when a variable of that name could have been meant instead. Long-form names: `hour`, `metre`, `gram`, `litre`, `tonne`.

**Constants.** CODATA 2022 with citations, pinned by checksum so the table cannot drift silently.

A measured constant's uncertainty is reported by `info` and is **not yet propagated into calculations**. `info G` prints `0.0000000000667430 ± 0.0000000000000015 m^3/(kg·s^2)` with its CODATA citation, but `G` used in an expression answers `6.6743e-11 m^3/(kg·s^2)` as if it were exact, so a result that depends on `G` carries no contribution from it and no budget line for it. Spec 4.3 requires that propagation and it is not built; declared measurements (`h = 12.3 ± 0.2 m`) do propagate, and they are the only source of uncertainty in a result today.

**Documents.** `gocalc run sheet.calc` evaluates a file in one environment, in dependency order, detecting cycles. `Doc.Edit` re-evaluates only the lines an edit actually staled.

**The REPL.** `gocalc repl`, with `why`, `trace`, `info`, `digits`, `deg`, `rad`, `unset`, `help` and `exit`, `?prefix` name lookup, and saved history.

**The sheet API.** `doc.EvalRequest` takes a whole sheet as JSON and answers with a status, the proven digits, the uncertainty, the notes and the provenance for every line. The WASM build exposes it to the browser as `goCalcEval`.

## Performance

Measured on a 12th Gen Intel Core i5-12400F, Linux 6.18 (WSL2), `go1.26.5 linux/amd64`, with `-benchtime 2s -count 3`; the figure is the median of three runs. Cold start is measured by forking and waiting on the built binary, not by timing a function call.

| Budget | Measured | Verdict |
| --- | --- | --- |
| a typical 12-digit expression under 1 ms | **0.211 ms** — `2 * sin(pi/4) + sqrt(2) / 3`, parsed, evaluated to 12 proven digits and printed | met |
| 1,000 digits of `exp(pi·sqrt(163))` under 1 s | **9.63 ms** | met |
| recalculating a 100-line document under 10 ms | **1.83 ms** | met |
| CLI cold start under 50 ms | **1.66 ms** — fork to exit of `gocalc "0.1 + 0.2"`; **2.04 ms** for the typical expression above | met |
| the WASM bundle under 4 MB compressed | **1,370,034 bytes (1.31 MiB)** gzipped from 5,058,747 bytes (4.82 MiB) raw | met |

Every budget is met. Nothing here is estimated or extrapolated.

Other paths that earlier work asked to be measured:

| What | Measured |
| --- | --- |
| `sqrt(2) - sqrt(2)` — the slowest provably-zero shape, which runs to the 8192-bit cap because the no-progress stop was removed by ruling | 3.86 ms |
| an expression with three declared uncertainties and a contribution budget (`h * w / sqrt(d)`), including Hessian propagation | 0.066 ms |
| editing one line of the 100-line document when one dependent restales | 0.042 ms |
| editing its root line, when 98 dependents restale | 1.76 ms |
| `doc.EvalRequest` on the same 100-line document — the browser's per-keystroke cost | 3.92 ms |
| the 10,000-digit width guard refusing `exp(1000000)` | 0.085 ms |
| ten refusals in one sheet, through `doc.EvalRequest` | 3.70 ms |
| ball multiplication at 120 bits | 689 ns (add 319 ns, divide 988 ns) |

Two things worth saying plainly. First, `doc.EvalRequest` has no incremental path: it builds a fresh document and re-evaluates every line on every call, so a keystroke in the browser costs 3.92 ms on a 100-line sheet where `Doc.Edit` would cost 0.042 ms for the same edit. That is inside any reasonable frame budget today, but it scales with the document, not with the edit. Second, refusing an unwritable magnitude costs 0.085 ms rather than the minutes it would take to grind one out, which is the point of the guard — but it is not free, and ten of them in one sheet cost 3.70 ms.

Run them yourself:

```
go test ./internal/... -bench . -run XXX
go test ./cmd/gocalc/ -bench . -run XXX
```

## Size

| Artifact | Bytes |
| --- | --- |
| `gocalc` binary, `go build ./cmd/gocalc` | 4,547,026 (4.34 MiB) |
| `gocalc.wasm`, raw | 5,058,747 (4.82 MiB) |
| `gocalc.wasm`, `gzip -9` | 1,370,034 (1.31 MiB) |

`wasm/build.sh` builds the bundle, copies `wasm_exec.js` beside it and prints both sizes against the 4 MiB compressed budget.

## Not built yet

Symbolic maths — nothing differentiates, simplifies or solves in closed form; the derivatives behind uncertainty are numeric. Graphing. Monte Carlo uncertainty, so distributions that the Hessian cannot describe are reported as nonlinear rather than sampled. The web front end: `web/` holds design mockups that are not yet wired to `goCalcEval`, so the WASM bundle currently has no page to live in. The design for the rest is in `docs/superpowers/specs/` and `DESIGN.md`.

## Verification

```
go test ./...
go vet ./...
go test ./internal/parse/ -run XXX -fuzz '^FuzzParse$' -fuzztime 120s
go test ./internal/doc/ -run XXX -fuzz '^FuzzEvalRequest$' -fuzztime 120s
```

455 test functions, 507 cases, green across twelve packages. Four fuzz targets, all clean on the runs recorded here:

| Target | What it checks | Executions |
| --- | --- | --- |
| `parse.FuzzParse` | no panic, and a parse → print → parse round trip | 36,505,884 in 120 s |
| `doc.FuzzEvalRequest` | any bytes in, valid `go-calc.sheet/1` out, every line numbered and statused | 13,820,619 in 120 s |
| `doc.FuzzEvalRequestLine` | the same for a single line, including the width guard's inputs | 2,856,420 in 60 s |
| `lex.FuzzScan` | no panic, and every successful scan ends in EOF | 22,479,150 in 60 s |

The constants table is pinned by checksum so it cannot drift silently.

# go-calc

A calculator that shows how far to trust its answers.

Every result is exact or refused. `0.1 + 0.2` is `0.3`, not `0.30000000000000004`. `(1e16 + 1) - 1e16` is `1`, where float64 gives `0`. Units carry dimensions, so `5 m + 3 s` is an error that names both. Constants carry their CODATA 2022 value and citation.

## Status

Exact core and CLI. Irrational results are refused rather than approximated until ball arithmetic lands, so `sqrt(2)` says so instead of guessing.

```
$ gocalc "0.1 + 0.2"
0.3

$ gocalc "(1e16 + 1) - 1e16"
1

$ gocalc "3 km to m"
3000 m

$ gocalc "20 degC to K"
293.15 K

$ gocalc --indian "18,50,000 / 12"
462500/3  ≈ 1,54,166.666666666667

$ gocalc "500 + 10%"
550

$ gocalc "1/2m"
0.5 m
  read as (1/2)·m

$ gocalc "5 m + 3 s"
cannot add length (m) and time (s)

$ gocalc "sqrt(2)"
1:1: sqrt: this needs ball arithmetic, which is not implemented yet
```

## Install

```
go install github.com/rahuldr07/go-calc/cmd/gocalc@latest
```

## Use

```
gocalc "expression"
gocalc --json "expression"
gocalc --indian "18,50,000 / 12"
gocalc run sheet.calc
```

`--json` prints the versioned `go-calc.result/1` schema. Errors exit non-zero.

## What it does today

- Exact rational arithmetic: decimals are rationals, `0.1` is exactly 1/10.
- Percent (`500 + 10%`), `mod`, factorial, exact roots of perfect powers, `gcd`, `lcm`, `floor`, `ceil`, `round`, `isprime`.
- Numbers as `1_000`, `18,50,000`, `0xFF`, `0b1010`, `0o17`, `1.5e-3`; lakh and crore.
- Units with SI prefixes, binary prefixes, imperial, time, force, energy, power and pressure, converted through exact rational factors (`inch` is exactly 127/5000 m).
- Affine temperature: `20 degC to K` is `293.15 K`, `20 degC + 5 K` is `25 degC`, and adding two absolute temperatures is refused.
- A name straight after a number is a unit, so `5 h` is five hours even when `h` is a variable.
- Compound units: `60 km/h`, `9.8 m/s^2`, `1 J/(mol·K)` — a name multiplied or divided by a unit is read as a unit too, and says so when it could have been a constant. Conversion targets may be unit expressions: `60 km/h to m/s`.
- Long-form unit names: `hour`, `minute`, `second`, `metre`, `meter`, `gram`, `litre`, `liter`, `tonne`.
- Implicit multiplication with an interpretation note whenever it meets division.
- Variables and user functions: `f(x) = x^2 + 1`.
- CODATA 2022 constants with citations, pinned by checksum.
- Documents: `gocalc run sheet.calc` evaluates a file line by line in one environment.

## Not built yet

Ball arithmetic and proven digits, measurement uncertainty with contribution budgets, provenance traces, the float64 shadow warning, the REPL, symbolic maths, graphing, and the web front end. The design for all of it is in `docs/superpowers/specs/` and `DESIGN.md`.

## Verification

```
go test ./...
go test ./internal/parse/ -fuzz FuzzParse -fuzztime 60s
```

The parser is fuzzed for panics and for a parse → print → parse round trip; 13.5M executions have run clean. The constants table is pinned by checksum so it cannot drift silently.

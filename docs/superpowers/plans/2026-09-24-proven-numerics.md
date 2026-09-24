# go-calc Proven Numerics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the exact-only engine into the product go-calc claims to be: proven digits for irrational results, measurement uncertainty with contribution budgets, provenance for every value, a float64 shadow that says where ordinary arithmetic would have failed, documents, a REPL, and a WASM build.

**Architecture:** `num` gains a ball tier (midpoint `big.Float` + upward-rounded radius) behind a tagged `Number` struct — no interface, no boxing on the exact path. `elem` implements elementary functions on balls with proven remainder bounds. `eval` gains a `Ctx` carrying precision, angle mode and digit target, a retry loop that doubles precision until the requested digits are guaranteed, and a float64 shadow pass. `unc` carries `map[InputID]float64` sensitivities through every operation. `prov` records a node per value in a per-evaluation arena. `doc` orders lines by dependency and re-runs only what changed. `wasm` exposes one call that takes a whole document and returns every line's result.

**Tech Stack:** Go 1.26, standard library only (`math/big`, `math`, `go:embed`, native fuzzing), no cgo. Front end later: Svelte 5 + Vite.

**Spec:** `docs/superpowers/specs/2026-09-24-go-calc-design.md`

## Global Constraints

- Module `github.com/rahuldr07/go-calc`; Go 1.26; standard library only; no cgo.
- **No comments in code.** Names and structure carry the meaning. `//go:embed` directives are not comments and are allowed.
- Commit messages carry no AI attribution and no `Co-Authored-By` trailer. Commit to `main`; no branches.
- Package direction is one-way: `lex → parse → num → elem → units → consts → unc → prov → value → eval → doc → format → cmd,wasm`. A lower package never imports a higher one.
- `Number` is a tagged struct, never an interface. The exact path allocates no more than it does today.
- **A printed digit is a promise.** Never print a digit that is not guaranteed by the ball. When precision runs out, print fewer digits and say so.
- Balls use upward rounding for every radius; midpoints use round-to-nearest.
- Statistical uncertainty (`±`, GUM) and guaranteed bounds (balls) are separate mechanisms and are never mixed in one number.
- Every new package ships property tests, not only example tests.
- Existing tests must keep passing: `go test ./...` and `go vet ./...` are green at every commit.

---

### Task 1: Ball arithmetic

**Files:**
- Create: `internal/num/mag.go`, `internal/num/ball.go`, `internal/num/number.go`
- Test: `internal/num/ball_test.go`, `internal/num/number_test.go`

**Interfaces:**
- Consumes: `num.Exact`.
- Produces:
  - `type Mag struct{ ... }` — a non-negative magnitude with upward-rounded arithmetic: `MagFromFloat(f float64) Mag`, `(Mag) Add(Mag) Mag`, `(Mag) Mul(Mag) Mag`, `(Mag) MulFloat(*big.Float) Mag`, `(Mag) Float() *big.Float`, `(Mag) IsZero() bool`, `(Mag) Cmp(Mag) int`, `(Mag) String() string`.
  - `type Ball struct{ Mid *big.Float; Rad Mag }`
  - `func NewBall(mid *big.Float, rad Mag) Ball`
  - `func BallFromExact(e Exact, prec uint) Ball`
  - `func (b Ball) Add(o Ball, prec uint) Ball`, `Sub`, `Mul`, `Div(o Ball, prec uint) (Ball, error)`, `Neg`, `Abs`
  - `func (b Ball) Contains(x *big.Float) bool`
  - `func (b Ball) Lo() *big.Float`, `Hi() *big.Float`
  - `func (b Ball) Sign() (int, bool)` — second result false when the sign is undecided.
  - `type Kind int` with `KindExact`, `KindBall`, `KindInterval`; `type Number struct{ ... }` with constructors `NumExact(Exact) Number`, `NumBall(Ball) Number`, `NumInterval(lo, hi Exact) Number`, accessors `IsExact() bool`, `Exact() Exact`, `Ball(prec uint) Ball`, and arithmetic `Add`, `Sub`, `Mul`, `Div`, `Pow(int)` that stay exact when both sides are exact and fall to balls otherwise.

**Rules the implementation must follow:**
- Radius arithmetic always rounds up: build `Mag` on a `*big.Float` with `SetMode(big.ToPositiveInf)` and a small precision (30 bits is enough).
- Every operation adds the rounding error of the midpoint to the radius: after computing a midpoint at precision `prec`, add `ulp(mid, prec)` to the radius. A helper `roundingError(mid *big.Float, prec uint) Mag` belongs in `ball.go`.
- `Div` returns an error when the divisor ball contains zero, with the message `divisor may be zero`.
- `Sign` reports undecided when the ball straddles zero.

- [ ] **Step 1: Write the property test first**

`internal/num/ball_test.go` must contain, at minimum:

```go
func TestBallContainsTruth(t *testing.T) {
	const prec = 64
	const hi = 400
	rnd := rand.New(rand.NewSource(7))
	for i := 0; i < 200000; i++ {
		a := randExact(rnd)
		b := randExact(rnd)
		ba := BallFromExact(a, prec)
		bb := BallFromExact(b, prec)
		for _, op := range []struct {
			name  string
			ball  func() (Ball, error)
			truth func() (Exact, error)
		}{
			{"add", func() (Ball, error) { return ba.Add(bb, prec), nil }, func() (Exact, error) { return a.Add(b), nil }},
			{"sub", func() (Ball, error) { return ba.Sub(bb, prec), nil }, func() (Exact, error) { return a.Sub(b), nil }},
			{"mul", func() (Ball, error) { return ba.Mul(bb, prec), nil }, func() (Exact, error) { return a.Mul(b), nil }},
			{"div", func() (Ball, error) { return ba.Div(bb, prec) }, func() (Exact, error) { return a.Div(b) }},
		} {
			got, berr := op.ball()
			want, eerr := op.truth()
			if eerr != nil || berr != nil {
				continue
			}
			exactFloat := new(big.Float).SetPrec(hi).SetRat(want.Rat())
			if !got.Contains(exactFloat) {
				t.Fatalf("%s(%s, %s): ball [%s .. %s] does not contain %s", op.name, a, b, got.Lo().Text('g', 20), got.Hi().Text('g', 20), exactFloat.Text('g', 20))
			}
		}
	}
}
```

`randExact` returns rationals spanning magnitudes from `1e-12` to `1e12`, including negatives, zero, and values whose decimal expansion does not terminate.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/num/ -run TestBallContainsTruth`
Expected: FAIL — `undefined: BallFromExact`.

- [ ] **Step 3: Implement `Mag`, `Ball` and `Number`**

Keep `Mag` tiny: a `*big.Float` in `ToPositiveInf` mode at 30 bits. Keep `Ball` operations textbook Arb: `(a±ra) * (b±rb)` gives midpoint `a*b` and radius `|a|·rb + |b|·ra + ra·rb + roundingError`.

- [ ] **Step 4: Run the property test until it is silent**

Run: `go test ./internal/num/ -run TestBallContainsTruth`
Expected: PASS, zero containment failures across 200k cases × 4 operations.

- [ ] **Step 5: Add the edge tests**

Cover: division by a ball straddling zero returns the `divisor may be zero` error; `Sign` is undecided for a straddling ball and decided otherwise; `NumExact(2).Div(NumExact(4))` stays exact (`1/2`); a ball plus an exact stays a ball; `Pow` with a negative exponent.

- [ ] **Step 6: Run the package and commit**

Run: `go test ./internal/num/ && go vet ./internal/num/`

```bash
git add internal/num
git commit -m "feat: ball arithmetic with proven containment"
```

---

### Task 2: Elementary functions on balls

**Files:**
- Create: `internal/elem/elem.go`, `internal/elem/pi.go`, `internal/elem/exp.go`, `internal/elem/log.go`, `internal/elem/trig.go`
- Test: `internal/elem/elem_test.go`

**Interfaces:**
- Consumes: `num.Ball`, `num.Mag`.
- Produces, each taking `(num.Ball, uint) (num.Ball, error)` unless noted:
  `Pi(prec uint) num.Ball`, `E(prec uint) num.Ball`, `Sqrt`, `Exp`, `Log`, `Log10`, `Log2`, `Sin`, `Cos`, `Tan`, `Asin`, `Acos`, `Atan`, `Atan2(y, x num.Ball, prec uint) (num.Ball, error)`, `Sinh`, `Cosh`, `Tanh`, `Pow(base, exp num.Ball, prec uint) (num.Ball, error)`.

**Rules:**
- Every function computes at a working precision above the requested one (`prec + 32` is the floor) and returns a ball whose radius covers both the series truncation and every rounding step.
- `Exp`: reduce by `ln 2` so the argument lands in `[-ln2/2, ln2/2]`, sum the Taylor series until the next term is below the target, bound the tail by the standard geometric bound, then scale back by the power of two.
- `Log`: reduce the mantissa to `[1, 2)`, use `atanh`-style series, add `k·ln2`.
- `Sin`/`Cos`: reduce modulo `π/2` using a `π` computed at working precision plus a guard equal to the exponent of the argument, so that large arguments widen the ball honestly instead of lying.
- `Sqrt`: Newton iteration from a float64 seed, with the final radius accounting for the derivative `1/(2√x)`.
- Domain errors are explicit: `Log` of a ball whose upper bound is not positive returns `log of a value that may not be positive`; `Sqrt` of a possibly-negative ball returns `square root of a value that may be negative`; `Asin`/`Acos` outside `[-1, 1]` say so.
- Radius propagation for any `f`: `rad_out = |f'(mid)|·rad_in + truncation + rounding`, with `|f'|` bounded over the whole input ball, never just at the midpoint.

- [ ] **Step 1: Write the reference-value test first**

`internal/elem/elem_test.go` must check each function against published digits, asserting containment rather than equality:

```go
func TestKnownConstants(t *testing.T) {
	const prec = 200
	cases := []struct {
		name string
		got  num.Ball
		want string
	}{
		{"pi", Pi(prec), "3.14159265358979323846264338327950288419716939937510582097494459"},
		{"e", E(prec), "2.71828182845904523536028747135266249775724709369995957496696763"},
	}
	for _, c := range cases {
		want, _, err := big.ParseFloat(c.want, 10, 256, big.ToNearestEven)
		if err != nil {
			t.Fatal(err)
		}
		if !c.got.Contains(want) {
			t.Errorf("%s: ball does not contain the true value", c.name)
		}
	}
}
```

Add the same shape for `exp(1)`, `log(2)`, `sin(1)`, `cos(1)`, `atan(1)`, `sqrt(2)` using at least 40 published digits each.

- [ ] **Step 2: Write the identity property test**

```go
func TestIdentitiesHoldOnBalls(t *testing.T) {
	const prec = 120
	rnd := rand.New(rand.NewSource(11))
	for i := 0; i < 20000; i++ {
		x := randBall(rnd, prec)
		s, err1 := Sin(x, prec)
		c, err2 := Cos(x, prec)
		if err1 != nil || err2 != nil {
			continue
		}
		sum := s.Mul(s, prec).Add(c.Mul(c, prec), prec)
		if !sum.Contains(big.NewFloat(1)) {
			t.Fatalf("sin²+cos² does not contain 1 for x=%s", x.Mid.Text('g', 20))
		}
		if e, err := Exp(x, prec); err == nil {
			if l, err := Log(e, prec); err == nil && !l.Contains(x.Mid) {
				t.Fatalf("log(exp(x)) does not contain x for x=%s", x.Mid.Text('g', 20))
			}
		}
	}
}
```

- [ ] **Step 3: Run both and watch them fail**

Run: `go test ./internal/elem/`
Expected: FAIL — `undefined: Pi`.

- [ ] **Step 4: Implement, one function per file, in this order**

`Pi` (Chudnovsky or Machin with a proven tail bound), `E`, `Sqrt`, `Exp`, `Log`, then `Sin`/`Cos`/`Tan`, then inverses and hyperbolics, then `Pow` as `exp(y·log x)` with the integer-exponent fast path delegating to `num.Ball.Pow`.

- [ ] **Step 5: Run everything green**

Run: `go test ./internal/elem/ -v`
Expected: every reference value contained, every identity holding, no domain error escaping as a panic.

- [ ] **Step 6: Commit**

```bash
git add internal/elem
git commit -m "feat: proven elementary functions on balls"
```

---

### Task 3: Precision retry and the digit printer

**Files:**
- Create: `internal/eval/ctx.go`
- Modify: `internal/eval/eval.go`, `internal/format/text.go`, `internal/format/json.go`
- Test: `internal/eval/ctx_test.go`, `internal/format/digits_test.go`

**Interfaces:**
- Produces:
  - `type Ctx struct{ Prec uint; MaxPrec uint; Digits int; Degrees bool; Deadline time.Time }` with `func DefaultCtx() *Ctx` (Prec 120, MaxPrec 8192, Digits 12).
  - `func (e *Env) EvalCtx(n parse.Node, ctx *Ctx) (value.Value, error)` — the retry loop; `Eval` calls it with `DefaultCtx()`.
  - `func format.GuaranteedDigits(b num.Ball) int`
  - `func (s Style) NumberBall(b num.Ball) (text string, digits int, capped bool)`

**Rules:**
- The retry loop doubles `Prec` until `GuaranteedDigits` reaches `Digits` or `Prec` exceeds `MaxPrec`. Retrying is pointless when an input is an interval or carries uncertainty — detect that and stop after the first pass.
- The printer emits only digits that every value in the ball agrees on. When the cap is hit it prints what is guaranteed and sets `capped`, which the text renderer shows as `N digits guaranteed (precision cap reached)`.
- **The rounding-boundary rule:** if the ball straddles a rounding boundary at the requested digit count, drop one digit rather than looping. This case must have its own test.

- [ ] **Step 1: Write the tests first**

```go
func TestProvenDigitsForFamousValue(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 30
	n, _, _ := parse.Parse("exp(pi * sqrt(163))")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	text, digits, capped := format.Default.NumberBall(v.Num.Ball(ctx.Prec))
	if capped {
		t.Fatalf("hit the precision cap for 30 digits")
	}
	if digits < 30 {
		t.Fatalf("only %d digits guaranteed, want 30", digits)
	}
	if !strings.HasPrefix(text, "262537412640768743.99999999999925007") {
		t.Fatalf("got %s", text)
	}
}

func TestNeverPrintsUnprovenDigits(t *testing.T) {
	const prec = 64
	b := num.NewBall(big.NewFloat(1.0/3.0), num.MagFromFloat(1e-6))
	text, digits, _ := format.Default.NumberBall(b)
	if digits > 6 {
		t.Fatalf("claimed %d digits from a ball of radius 1e-6: %s", digits, text)
	}
}

func TestRoundingBoundaryTerminates(t *testing.T) {
	done := make(chan struct{})
	go func() {
		b := num.NewBall(big.NewFloat(0.125), num.MagFromFloat(1e-30))
		format.Default.NumberBall(b)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("digit printer did not terminate on a rounding boundary")
	}
}
```

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./internal/eval/ ./internal/format/`
Expected: FAIL — `undefined: EvalCtx`.

- [ ] **Step 3: Implement `Ctx`, the retry loop and the printer**

- [ ] **Step 4: Wire the evaluator's irrational paths to `elem`**

`sqrt`, `exp`, `log`, `log10`, `log2`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `atan2`, `sinh`, `cosh`, `tanh`, `pi`, `e` and fractional `^` now return balls instead of the "not implemented yet" error. The exact tier still wins where it can: `sqrt(16/9)` stays `4/3`.

- [ ] **Step 5: Update the tests that asserted refusal**

`TestIrrationalIsRefusedNotApproximated` in `internal/eval/eval_test.go` is now wrong: replace it with `TestIrrationalIsProven`, asserting that `sqrt(2)` returns a ball containing the true value and that at least 12 digits are guaranteed.

- [ ] **Step 6: Run everything and commit**

Run: `go test ./... && go vet ./...`

```bash
git add internal/eval internal/format
git commit -m "feat: precision retry and the guaranteed-digit printer"
```

---

### Task 4: Measurement uncertainty

**Files:**
- Create: `internal/unc/unc.go`, `internal/unc/budget.go`
- Modify: `internal/value/value.go`, `internal/eval/eval.go`, `internal/parse/parse.go` (accept `12.3(2)`), `internal/format/text.go`
- Test: `internal/unc/unc_test.go`, `internal/eval/unc_test.go`

**Interfaces:**
- Produces:
  - `type InputID int`, `type Sens map[InputID]float64`
  - `type Input struct{ ID InputID; Name string; Value float64; Sigma float64; Line int }`
  - `type Table struct{ ... }` with `func NewTable() *Table`, `func (t *Table) Declare(name string, value, sigma float64, line int) InputID`, `func (t *Table) SetCorr(a, b InputID, rho float64) error`, `func (t *Table) Combined(s Sens) float64`, `func (t *Table) Contributions(s Sens) []Contribution`
  - `type Contribution struct{ Name string; Share float64; Sigma float64; Sensitivity float64 }`
  - Sensitivity combinators: `Lin(a Sens, ca float64, b Sens, cb float64) Sens`, `Scale(a Sens, k float64) Sens`
- `value.Value` gains `Sens unc.Sens`.

**Rules:**
- Propagation is forward-mode: every operation combines its operands' sensitivities with the partial derivatives at the operating point. `x - x` must give exactly zero combined uncertainty, and a test asserts it.
- Combined uncertainty is `sqrt(Σᵢ Σⱼ cᵢ cⱼ ρᵢⱼ σᵢ σⱼ)`; with no declared correlation `ρᵢⱼ` is the identity.
- Contributions are `(cᵢσᵢ)² / u²` for uncorrelated inputs, sorted descending, and always sum to 1 within 1e-9 when no correlation is declared.
- Uncertainty is reported to two significant digits, and the value is rounded to the same decimal place as its uncertainty.
- `±` and balls never mix: a value carries a ball for its numeric error and a `Sens` for its statistical uncertainty, and the formatter shows them differently.

- [ ] **Step 1: Write the GUM worked-example test first**

```go
func TestGumWorkedExample(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	run(t, env, "g = g0")
	v := mustEval(t, env, "t = sqrt(2h / g)")
	u := env.Uncertainty(v)
	if math.Abs(u-0.0127) > 0.0005 {
		t.Errorf("u(t) = %g, want about 0.0127", u)
	}
	cs := env.Contributions(v)
	if len(cs) != 1 || cs[0].Name != "h" || math.Abs(cs[0].Share-1) > 1e-9 {
		t.Errorf("contributions = %+v, want h at 100%%", cs)
	}
}

func TestSameInputCancels(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 10 ± 1")
	v := mustEval(t, env, "x - x")
	if u := env.Uncertainty(v); u != 0 {
		t.Errorf("u(x - x) = %g, want exactly 0", u)
	}
}

func TestDeclaredCorrelation(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 10 ± 1")
	run(t, env, "corr(a, b) = 1")
	v := mustEval(t, env, "a + b")
	if u := env.Uncertainty(v); math.Abs(u-2) > 1e-9 {
		t.Errorf("fully correlated sum u = %g, want 2", u)
	}
}
```

- [ ] **Step 2: Run and watch it fail**

Run: `go test ./internal/eval/ -run TestGum`
Expected: FAIL — the parser does not accept `±` values yet.

- [ ] **Step 3: Implement the table, the combinators and the propagation**

Every arithmetic case in `evalBinary`, every unary, and every `elem` call gains its partial derivative. For `f(x)` the sensitivity is `f'(mid) · s_x`.

- [ ] **Step 4: Add the concise notation and the nonlinearity check**

`12.3(2)` parses as `12.3 ± 0.2`. After propagation, compare the first-order result with a second-order estimate; when they differ by more than 10%, attach the warning `first-order propagation may be unreliable here; Monte Carlo would be better`.

- [ ] **Step 5: Format it**

`1.584 ± 0.013 s` with the budget available as structured data. `expand(x, 2)` and `expand(x, 95%)` produce the expanded uncertainty labelled with its coverage factor.

- [ ] **Step 6: Run everything and commit**

Run: `go test ./... && go vet ./...`

```bash
git add internal/unc internal/value internal/eval internal/parse internal/format
git commit -m "feat: GUM uncertainty with contribution budgets"
```

---

### Task 5: Provenance and the float64 shadow

**Files:**
- Create: `internal/prov/prov.go`, `internal/prov/trace.go`, `internal/eval/shadow.go`
- Modify: `internal/value/value.go`, `internal/eval/eval.go`, `internal/format/json.go`
- Test: `internal/prov/prov_test.go`, `internal/eval/shadow_test.go`

**Interfaces:**
- Produces:
  - `type NodeID int32`, `type NodeKind int` (`Literal`, `Input`, `Constant`, `Unit`, `Operation`, `Function`, `Variable`)
  - `type Node struct{ Kind NodeKind; Label string; Detail string; Line int; Children []NodeID }`
  - `type Arena struct{ ... }` with `func NewArena() *Arena`, `func (a *Arena) Add(n Node) NodeID`, `func (a *Arena) Get(id NodeID) Node`, `func (a *Arena) Tree(id NodeID) string`, `func (a *Arena) JSON(id NodeID) ([]byte, error)`
  - `type Shadow struct{ Value float64; Lost []LossSite }`, `type LossSite struct{ Op string; Pos lex.Pos; BitsLost float64 }`
  - `func (e *Env) Shadow(n parse.Node) Shadow`
  - `func (e *Env) Why(n parse.Node, proven value.Value) string`

**Rules:**
- A constant's node carries its citation; a unit conversion's node carries the factor used; an input's node carries its line.
- Bits lost at an add/sub node is `log2(max(|a|,|b|) / |a ± b|)`, computed in the shadow pass; the worst site is what the one-line warning names.
- The warning fires when the shadow and the proven value agree on fewer than six significant digits.
- `trace` prints the tree; `--json` exports nodes and the root.

- [ ] **Step 1: Write the tests first**

```go
func TestShadowFindsTheCancellation(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("(1e16 + 1) - 1e16")
	sh := env.Shadow(n)
	if sh.Value != 0 {
		t.Fatalf("float64 shadow = %v, want 0", sh.Value)
	}
	if len(sh.Lost) == 0 {
		t.Fatal("no loss site recorded")
	}
	worst := sh.Lost[0]
	if worst.Op != "-" || worst.BitsLost < 50 {
		t.Errorf("worst site = %+v, want the subtraction losing at least 50 bits", worst)
	}
}

func TestNoWarningOnBenignArithmetic(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("2 + 3 * 4")
	if sh := env.Shadow(n); len(sh.Lost) != 0 {
		t.Errorf("benign expression reported loss: %+v", sh.Lost)
	}
}

func TestTraceNamesInputsAndCitations(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	v := mustEval(t, env, "sqrt(2h / g0)")
	tree := env.Arena().Tree(v.Prov)
	for _, want := range []string{"h", "input", "g0", "CODATA 2022"} {
		if !strings.Contains(tree, want) {
			t.Errorf("trace missing %q:\n%s", want, tree)
		}
	}
}
```

- [ ] **Step 2: Run and watch them fail**

Run: `go test ./internal/eval/ -run Shadow`
Expected: FAIL — `undefined: Shadow`.

- [ ] **Step 3: Implement the arena, the shadow pass and `why`**

- [ ] **Step 4: Add the catalogue test**

A table of known cancellation cases (`(1e16+1)-1e16`, `sqrt(x+1)-sqrt(x)` at `x=1e10`, `1-cos(1e-8)`, `log(1+1e-15)`) must each produce a warning, and a benign control set (`2+2`, `355/113`, `exp(1)`, `1e10 + 1e-10`) must produce none.

- [ ] **Step 5: Run everything and commit**

Run: `go test ./... && go vet ./...`

```bash
git add internal/prov internal/eval internal/value internal/format
git commit -m "feat: provenance arena, float64 shadow and why"
```

---

### Task 6: Documents

**Files:**
- Create: `internal/doc/doc.go`, `internal/doc/graph.go`
- Modify: `cmd/gocalc/main.go` (use `doc` for `run`)
- Test: `internal/doc/doc_test.go`

**Interfaces:**
- Produces:
  - `type Line struct{ N int; Src string; Value value.Value; Notes []parse.Note; Err error; Warning string }`
  - `type Doc struct{ ... }` with `func New() *Doc`, `func (d *Doc) SetLines(src []string) []Line`, `func (d *Doc) Edit(n int, src string) []Line` (returns only the lines that changed), `func (d *Doc) Lines() []Line`

**Rules:**
- `Edit` re-evaluates the edited line and its transitive dependents only; a test asserts that an unrelated line is not re-evaluated by counting evaluations.
- A cycle (`a = b + 1`, `b = a + 1`) is reported on the offending line as `this line depends on itself` and does not hang.
- A line that fails leaves later lines evaluable when they do not depend on it.

- [ ] **Step 1: Write the tests first**

```go
func TestEditReevaluatesOnlyDependents(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = 2", "b = a * 3", "c = 99"})
	changed := d.Edit(0, "a = 4")
	var got []int
	for _, l := range changed {
		got = append(got, l.N)
	}
	if !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("changed lines = %v, want [0 1]", got)
	}
}

func TestCycleIsReportedNotHung(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"a = b + 1", "b = a + 1"})
	if lines[0].Err == nil && lines[1].Err == nil {
		t.Fatal("cycle not reported")
	}
}
```

- [ ] **Step 2: Run, implement, run**

Run: `go test ./internal/doc/`

- [ ] **Step 3: Point the CLI's `run` at `doc` and commit**

Run: `go test ./... && go vet ./...`

```bash
git add internal/doc cmd/gocalc
git commit -m "feat: documents with incremental re-evaluation"
```

---

### Task 7: REPL

**Files:**
- Create: `cmd/gocalc/repl.go`
- Modify: `cmd/gocalc/main.go`
- Test: `cmd/gocalc/repl_test.go`

**Interfaces:**
- Produces: `func repl(in io.Reader, out io.Writer, style format.Style) int`

**Rules:**
- Bare keywords work as commands: `why`, `trace <name>`, `info <name>`, `digits <n>`, `deg`, `rad`, `help`, `exit`.
- History persists to `$XDG_STATE_HOME/go-calc/history` (falling back to `$HOME/.local/state/go-calc/history`), capped at 1000 lines.
- Output is plain when stdout is not a terminal, so the tests read it directly.
- No third-party line editor: read lines with `bufio`.

- [ ] **Step 1: Write the test first**

```go
func TestReplEvaluatesAndRemembers(t *testing.T) {
	in := strings.NewReader("x = 2 + 3\nx * 10\nwhy\nexit\n")
	var out bytes.Buffer
	if code := repl(in, &out, format.Style{Digits: 12}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := out.String()
	for _, want := range []string{"5", "50"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}
```

- [ ] **Step 2: Implement, run, commit**

Run: `go test ./cmd/gocalc/`

```bash
git add cmd/gocalc
git commit -m "feat: REPL with commands and history"
```

---

### Task 8: WASM bridge

**Files:**
- Create: `wasm/main.go`, `wasm/build.sh`
- Test: `internal/doc/api_test.go`
- Create: `internal/doc/api.go`

**Interfaces:**
- Produces:
  - `func doc.EvalRequest(req []byte) []byte` — takes `{"lines":["h = 12.3 ± 0.2 m", ...],"digits":12,"angle":"rad"}` and returns `{"schema":"go-calc.sheet/1","lines":[{"n":0,"text":"...","status":"exact|proven|estimate|held|error","digits":12,"uncertainty":{...},"notes":[...],"warning":"...","trace":[...]}]}`.
  - `wasm/main.go` registers `goCalcEval` on `globalThis` and blocks forever.

**Rules:**
- The whole document crosses the boundary in one call; no per-line calls.
- `EvalRequest` never panics: it recovers and returns an error line rather than killing the module. A fuzz test asserts this.
- `wasm/build.sh` runs `GOOS=js GOARCH=wasm go build -o web/public/gocalc.wasm ./wasm` and prints the raw and gzipped sizes.

- [ ] **Step 1: Write the API test first**

```go
func TestEvalRequestShape(t *testing.T) {
	req := []byte(`{"lines":["0.1 + 0.2","5 m + 3 s","sqrt(2)"],"digits":12}`)
	var got map[string]any
	if err := json.Unmarshal(EvalRequest(req), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	lines := got["lines"].([]any)
	if len(lines) != 3 {
		t.Fatalf("got %d lines", len(lines))
	}
	first := lines[0].(map[string]any)
	if first["text"] != "0.3" || first["status"] != "exact" {
		t.Errorf("line 0 = %+v", first)
	}
	if lines[1].(map[string]any)["status"] != "error" {
		t.Errorf("line 1 should be an error")
	}
	if lines[2].(map[string]any)["status"] != "proven" {
		t.Errorf("line 2 should be proven")
	}
}

func FuzzEvalRequest(f *testing.F) {
	f.Add(`{"lines":["1+1"]}`)
	f.Add(`{"lines":[""]}`)
	f.Fuzz(func(t *testing.T, s string) {
		EvalRequest([]byte(s))
	})
}
```

- [ ] **Step 2: Implement, run, build the WASM binary**

Run: `go test ./internal/doc/ && bash wasm/build.sh`
Expected: the script prints both sizes; record them in the commit message.

- [ ] **Step 3: Commit**

```bash
git add internal/doc wasm
git commit -m "feat: sheet API and WASM bridge"
```

---

### Task 9: Benchmarks and the performance budget

**Files:**
- Create: `internal/eval/bench_test.go`, `internal/num/bench_test.go`
- Modify: `README.md`

**Rules:**
- Benchmarks cover: a 12-digit typical expression, 1,000 digits of `exp(pi·sqrt(163))`, a 100-line document recalculation, and ball multiplication at 120 bits.
- The README records the measured numbers against the budget: typical expression under 1 ms, 1,000 digits under 1 s, 100-line recalculation under 10 ms, cold start under 50 ms, WASM under 4 MB compressed. Numbers are measured, never claimed.

- [ ] **Step 1: Write the benchmarks, run them, record the real numbers**

Run: `go test ./internal/... -bench . -benchtime 200x -run XXX`

- [ ] **Step 2: Update the README's status and numbers, then commit**

```bash
git add internal README.md
git commit -m "test: benchmarks and the measured performance budget"
```

# go-calc Exact Core and CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `gocalc` binary that evaluates expressions exactly — rationals, units, constants, percent, Indian grouping — and prints text or JSON, with a parser that never panics.

**Architecture:** A layered Go module. `lex` produces positioned tokens, `parse` builds an AST with interpretation notes, `num` holds exact rational arithmetic, `units` holds dimension algebra and conversion, `value` composes a number with a dimension, `eval` walks the AST against an environment, `format` renders text and JSON, and `cmd/gocalc` is the entry point. Lower packages never import higher ones. Ball arithmetic, uncertainty, provenance and the web front end are later plans; this one stops where irrational results begin, reporting them as unsupported rather than approximating.

**Tech Stack:** Go 1.26, standard library only (`math/big`, `go:embed`, `testing`, native fuzzing). No third-party dependencies, no cgo.

**Spec:** `docs/superpowers/specs/2026-09-24-go-calc-design.md`

## Global Constraints

- Module path `github.com/rahuldr07/go-calc`; Go 1.26; standard library only; no cgo.
- No comments in code. Names and structure carry the meaning.
- Commit messages carry no AI attribution and no `Co-Authored-By` trailer. Commit to `main`; no branches.
- Lower packages never import higher ones: `lex → parse → num/units → value → eval → format → cmd`.
- `%` means percent, never modulo; modulo is the `mod` keyword.
- A name directly after a number resolves as a unit; a bare name resolves variable → constant → unit.
- Decimal literals are exact rationals: `0.1` is exactly 1/10.
- Every error carries a line and column.
- The parser never panics on any input.

---

### Task 1: Module skeleton and CI

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/lex/lex.go`
- Test: `internal/lex/lex_test.go`
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: nothing.
- Produces: package `lex` exists and compiles; CI runs `go vet ./...` and `go test -race ./...`.

- [ ] **Step 1: Write the failing test**

`internal/lex/lex_test.go`:

```go
package lex

import "testing"

func TestPackageBuilds(t *testing.T) {
	if Version != "0.1.0" {
		t.Fatalf("got %q, want %q", Version, "0.1.0")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lex/`
Expected: FAIL — `go.mod` missing, then `undefined: Version`.

- [ ] **Step 3: Write minimal implementation**

```bash
go mod init github.com/rahuldr07/go-calc
```

`internal/lex/lex.go`:

```go
package lex

const Version = "0.1.0"
```

`.gitignore`:

```
/gocalc
/dist/
.impeccable/
.superpowers/
```

`.github/workflows/ci.yml`:

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - run: go vet ./...
      - run: go test -race ./...
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lex/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod .gitignore internal/lex .github/workflows/ci.yml
git commit -m "chore: module skeleton and CI"
```

---

### Task 2: Token scanner

**Files:**
- Modify: `internal/lex/lex.go`
- Create: `internal/lex/token.go`
- Test: `internal/lex/lex_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Kind int` with constants `Num, Ident, Op, LParen, RParen, Comma, Comment, EOF`.
  - `type Pos struct{ Line, Col int }`
  - `type Token struct{ Kind Kind; Text string; Pos Pos }`
  - `func Scan(src string) ([]Token, error)`

- [ ] **Step 1: Write the failing test**

Replace `internal/lex/lex_test.go`:

```go
package lex

import "testing"

func TestScanNumbersAndOperators(t *testing.T) {
	toks, err := Scan("12.3 + 1_000 * 0xFF")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []struct {
		kind Kind
		text string
	}{
		{Num, "12.3"},
		{Op, "+"},
		{Num, "1_000"},
		{Op, "*"},
		{Num, "0xFF"},
		{EOF, ""},
	}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i].Kind != w.kind || toks[i].Text != w.text {
			t.Errorf("token %d = (%v, %q), want (%v, %q)", i, toks[i].Kind, toks[i].Text, w.kind, w.text)
		}
	}
}

func TestScanPositions(t *testing.T) {
	toks, err := Scan("a =\n  42")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if toks[2].Pos.Line != 2 || toks[2].Pos.Col != 3 {
		t.Errorf("42 at %v, want line 2 col 3", toks[2].Pos)
	}
}

func TestScanIndianGroupingAndComment(t *testing.T) {
	toks, err := Scan("18,50,000 # salary")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if toks[0].Kind != Num || toks[0].Text != "18,50,000" {
		t.Errorf("got (%v, %q), want grouped number", toks[0].Kind, toks[0].Text)
	}
	if toks[1].Kind != Comment {
		t.Errorf("got %v, want Comment", toks[1].Kind)
	}
}

func TestScanUnknownRune(t *testing.T) {
	if _, err := Scan("2 $ 3"); err == nil {
		t.Fatal("want error for unknown rune")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/lex/`
Expected: FAIL — `undefined: Scan`.

- [ ] **Step 3: Write minimal implementation**

`internal/lex/token.go`:

```go
package lex

import "fmt"

type Kind int

const (
	Num Kind = iota
	Ident
	Op
	LParen
	RParen
	Comma
	Comment
	EOF
)

func (k Kind) String() string {
	switch k {
	case Num:
		return "number"
	case Ident:
		return "name"
	case Op:
		return "operator"
	case LParen:
		return "("
	case RParen:
		return ")"
	case Comma:
		return ","
	case Comment:
		return "comment"
	default:
		return "end of input"
	}
}

type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

type Token struct {
	Kind Kind
	Text string
	Pos  Pos
}

type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Pos, e.Msg) }
```

`internal/lex/lex.go`:

```go
package lex

import (
	"strings"
	"unicode"
)

const Version = "0.1.0"

const operators = "+-*/^!%×÷·±"

func Scan(src string) ([]Token, error) {
	var toks []Token
	line, col := 1, 1
	rs := []rune(src)
	for i := 0; i < len(rs); {
		r := rs[i]
		start := Pos{Line: line, Col: col}
		switch {
		case r == '\n':
			i++
			line++
			col = 1
		case unicode.IsSpace(r):
			i++
			col++
		case r == '#':
			j := i
			for j < len(rs) && rs[j] != '\n' {
				j++
			}
			toks = append(toks, Token{Comment, string(rs[i:j]), start})
			col += j - i
			i = j
		case unicode.IsDigit(r) || (r == '.' && i+1 < len(rs) && unicode.IsDigit(rs[i+1])):
			j := scanNumber(rs, i)
			toks = append(toks, Token{Num, string(rs[i:j]), start})
			col += j - i
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
				j++
			}
			toks = append(toks, Token{Ident, string(rs[i:j]), start})
			col += j - i
			i = j
		case r == '(':
			toks = append(toks, Token{LParen, "(", start})
			i++
			col++
		case r == ')':
			toks = append(toks, Token{RParen, ")", start})
			i++
			col++
		case r == ',':
			toks = append(toks, Token{Comma, ",", start})
			i++
			col++
		case strings.ContainsRune(operators, r):
			text := string(r)
			if r == '+' && i+1 < len(rs) && rs[i+1] == '-' {
				text = "+-"
				i++
				col++
			}
			toks = append(toks, Token{Op, text, start})
			i++
			col++
		default:
			return nil, &Error{Pos: start, Msg: "unexpected character " + string(r)}
		}
	}
	return append(toks, Token{EOF, "", Pos{Line: line, Col: col}}), nil
}

func scanNumber(rs []rune, i int) int {
	if rs[i] == '0' && i+1 < len(rs) && strings.ContainsRune("xXbBoO", rs[i+1]) {
		j := i + 2
		for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
			j++
		}
		return j
	}
	j := i
	seenDot := false
	for j < len(rs) {
		switch {
		case unicode.IsDigit(rs[j]) || rs[j] == '_':
			j++
		case rs[j] == ',' && j+1 < len(rs) && unicode.IsDigit(rs[j+1]):
			j++
		case rs[j] == '.' && !seenDot && j+1 < len(rs) && unicode.IsDigit(rs[j+1]):
			seenDot = true
			j++
		case (rs[j] == 'e' || rs[j] == 'E') && j+1 < len(rs) && (unicode.IsDigit(rs[j+1]) || ((rs[j+1] == '-' || rs[j+1] == '+') && j+2 < len(rs) && unicode.IsDigit(rs[j+2]))):
			j += 2
		default:
			return j
		}
	}
	return j
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/lex/ -v`
Expected: PASS, all four tests.

- [ ] **Step 5: Add the fuzz target**

Append to `internal/lex/lex_test.go`:

```go
func FuzzScan(f *testing.F) {
	f.Add("12.3 + 1_000")
	f.Add("18,50,000 # c")
	f.Add("0xFF ± 2e-3")
	f.Fuzz(func(t *testing.T, src string) {
		toks, err := Scan(src)
		if err != nil {
			return
		}
		if len(toks) == 0 || toks[len(toks)-1].Kind != EOF {
			t.Fatalf("scan of %q did not end in EOF", src)
		}
	})
}
```

- [ ] **Step 6: Run the fuzzer briefly**

Run: `go test ./internal/lex/ -run FuzzScan -fuzz FuzzScan -fuzztime 30s`
Expected: no crashes. Delete any file written under `testdata/fuzz` only after fixing the panic it records.

- [ ] **Step 7: Commit**

```bash
git add internal/lex
git commit -m "feat: token scanner with positions, grouping and comments"
```

---

### Task 3: Exact rational numbers

**Files:**
- Create: `internal/num/exact.go`
- Test: `internal/num/exact_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Exact struct{ r *big.Rat }`
  - `func ParseExact(lit string) (Exact, error)` — accepts `12.3`, `1_000`, `18,50,000`, `0xFF`, `0b1010`, `0o17`, `1.5e-3`.
  - `func FromInt(i int64) Exact`
  - Methods `Add, Sub, Mul`, `Div(Exact) (Exact, error)`, `Pow(n int) Exact`, `Neg`, `Sign() int`, `IsInt() bool`, `Cmp(Exact) int`, `Rat() *big.Rat`, `String() string`.
  - `func Root(x Exact, n int) (Exact, bool)` — exact only when the result is rational.

- [ ] **Step 1: Write the failing test**

`internal/num/exact_test.go`:

```go
package num

import "testing"

func TestParseExactIsExact(t *testing.T) {
	a, err := ParseExact("0.1")
	if err != nil {
		t.Fatalf("ParseExact: %v", err)
	}
	b, _ := ParseExact("0.2")
	sum := a.Add(b)
	want, _ := ParseExact("0.3")
	if sum.Cmp(want) != 0 {
		t.Errorf("0.1 + 0.2 = %s, want %s", sum, want)
	}
	if got := a.String(); got != "1/10" {
		t.Errorf("0.1 stored as %s, want 1/10", got)
	}
}

func TestParseExactForms(t *testing.T) {
	cases := map[string]string{
		"1_000":     "1000",
		"18,50,000": "1850000",
		"0xFF":      "255",
		"0b1010":    "10",
		"0o17":      "15",
		"1.5e-3":    "3/2000",
	}
	for lit, want := range cases {
		got, err := ParseExact(lit)
		if err != nil {
			t.Fatalf("ParseExact(%q): %v", lit, err)
		}
		if got.String() != want {
			t.Errorf("ParseExact(%q) = %s, want %s", lit, got, want)
		}
	}
}

func TestDivByZero(t *testing.T) {
	a := FromInt(1)
	if _, err := a.Div(FromInt(0)); err == nil {
		t.Fatal("want error dividing by zero")
	}
}

func TestPowAndRoot(t *testing.T) {
	x, _ := ParseExact("16/9")
	r, ok := Root(x, 2)
	if !ok {
		t.Fatal("sqrt(16/9) should be exact")
	}
	if r.String() != "4/3" {
		t.Errorf("sqrt(16/9) = %s, want 4/3", r)
	}
	if _, ok := Root(FromInt(2), 2); ok {
		t.Error("sqrt(2) must not be exact")
	}
	if got := FromInt(2).Pow(-2).String(); got != "1/4" {
		t.Errorf("2^-2 = %s, want 1/4", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/num/`
Expected: FAIL — `undefined: ParseExact`.

- [ ] **Step 3: Write minimal implementation**

`internal/num/exact.go`:

```go
package num

import (
	"errors"
	"math/big"
	"strings"
)

var ErrDivByZero = errors.New("division by zero")

type Exact struct {
	r *big.Rat
}

func FromInt(i int64) Exact { return Exact{new(big.Rat).SetInt64(i)} }

func FromRat(r *big.Rat) Exact { return Exact{new(big.Rat).Set(r)} }

func ParseExact(lit string) (Exact, error) {
	clean := strings.NewReplacer("_", "", ",", "").Replace(lit)
	if len(clean) > 2 && clean[0] == '0' {
		base := 0
		switch clean[1] {
		case 'x', 'X':
			base = 16
		case 'b', 'B':
			base = 2
		case 'o', 'O':
			base = 8
		}
		if base != 0 {
			i, ok := new(big.Int).SetString(clean[2:], base)
			if !ok {
				return Exact{}, errors.New("bad number " + lit)
			}
			return Exact{new(big.Rat).SetInt(i)}, nil
		}
	}
	r, ok := new(big.Rat).SetString(clean)
	if !ok {
		return Exact{}, errors.New("bad number " + lit)
	}
	return Exact{r}, nil
}

func (e Exact) rat() *big.Rat {
	if e.r == nil {
		return new(big.Rat)
	}
	return e.r
}

func (e Exact) Rat() *big.Rat { return new(big.Rat).Set(e.rat()) }

func (e Exact) Add(o Exact) Exact { return Exact{new(big.Rat).Add(e.rat(), o.rat())} }

func (e Exact) Sub(o Exact) Exact { return Exact{new(big.Rat).Sub(e.rat(), o.rat())} }

func (e Exact) Mul(o Exact) Exact { return Exact{new(big.Rat).Mul(e.rat(), o.rat())} }

func (e Exact) Div(o Exact) (Exact, error) {
	if o.Sign() == 0 {
		return Exact{}, ErrDivByZero
	}
	return Exact{new(big.Rat).Quo(e.rat(), o.rat())}, nil
}

func (e Exact) Neg() Exact { return Exact{new(big.Rat).Neg(e.rat())} }

func (e Exact) Sign() int { return e.rat().Sign() }

func (e Exact) IsInt() bool { return e.rat().IsInt() }

func (e Exact) Cmp(o Exact) int { return e.rat().Cmp(o.rat()) }

func (e Exact) String() string { return e.rat().RatString() }

func (e Exact) Pow(n int) Exact {
	if n == 0 {
		return FromInt(1)
	}
	base := e.rat()
	neg := n < 0
	if neg {
		n = -n
	}
	num := new(big.Int).Exp(base.Num(), big.NewInt(int64(n)), nil)
	den := new(big.Int).Exp(base.Denom(), big.NewInt(int64(n)), nil)
	out := new(big.Rat).SetFrac(num, den)
	if neg {
		out = out.Inv(out)
	}
	return Exact{out}
}

func Root(x Exact, n int) (Exact, bool) {
	if n <= 0 || x.Sign() < 0 {
		return Exact{}, false
	}
	num, ok := intRoot(x.rat().Num(), n)
	if !ok {
		return Exact{}, false
	}
	den, ok := intRoot(x.rat().Denom(), n)
	if !ok {
		return Exact{}, false
	}
	return Exact{new(big.Rat).SetFrac(num, den)}, true
}

func intRoot(v *big.Int, n int) (*big.Int, bool) {
	if v.Sign() == 0 {
		return big.NewInt(0), true
	}
	lo, hi := big.NewInt(1), new(big.Int).Add(v, big.NewInt(1))
	exp := big.NewInt(int64(n))
	for lo.Cmp(hi) < 0 {
		mid := new(big.Int).Add(lo, hi)
		mid.Div(mid, big.NewInt(2))
		p := new(big.Int).Exp(mid, exp, nil)
		switch p.Cmp(v) {
		case 0:
			return mid, true
		case -1:
			lo = mid.Add(mid, big.NewInt(1))
		default:
			hi = mid
		}
	}
	return nil, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/num/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/num
git commit -m "feat: exact rational arithmetic"
```

---

### Task 4: Dimensions, units and conversion

**Files:**
- Create: `internal/units/dim.go`
- Create: `internal/units/table.go`
- Test: `internal/units/units_test.go`

**Interfaces:**
- Consumes: `num.Exact`.
- Produces:
  - `type Dim [8]int` — length, mass, time, current, temperature, amount, luminous, information.
  - `func (d Dim) Add(o Dim) Dim`, `func (d Dim) Sub(o Dim) Dim`, `func (d Dim) Scale(n int) Dim`, `func (d Dim) IsNone() bool`, `func (d Dim) Words() string`.
  - `type Kind int` with `Absolute`, `Difference`.
  - `type Unit struct{ Symbol string; Dim Dim; Factor num.Exact; Offset num.Exact; Kind Kind }`
  - `func Lookup(sym string) (Unit, bool)` — handles SI prefixes.
  - `func Convert(v num.Exact, from, to Unit) (num.Exact, error)`

- [ ] **Step 1: Write the failing test**

`internal/units/units_test.go`:

```go
package units

import (
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
)

func TestLookupWithPrefix(t *testing.T) {
	u, ok := Lookup("km")
	if !ok {
		t.Fatal("km not found")
	}
	if u.Factor.String() != "1000" {
		t.Errorf("km factor %s, want 1000", u.Factor)
	}
	if u.Dim != (Dim{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("km dim %v, want length", u.Dim)
	}
}

func TestExactImperialFactor(t *testing.T) {
	u, _ := Lookup("inch")
	if u.Factor.String() != "127/5000" {
		t.Errorf("inch factor %s, want 127/5000 exactly", u.Factor)
	}
}

func TestConvertLength(t *testing.T) {
	km, _ := Lookup("km")
	m, _ := Lookup("m")
	got, err := Convert(num.FromInt(3), km, m)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.String() != "3000" {
		t.Errorf("3 km = %s m, want 3000", got)
	}
}

func TestConvertRejectsMismatch(t *testing.T) {
	m, _ := Lookup("m")
	s, _ := Lookup("s")
	if _, err := Convert(num.FromInt(1), m, s); err == nil {
		t.Fatal("want error converting length to time")
	}
}

func TestDimWords(t *testing.T) {
	if got := (Dim{1, 0, 0, 0, 0, 0, 0, 0}).Words(); got != "length" {
		t.Errorf("Words() = %q, want length", got)
	}
	if got := (Dim{1, 0, -1, 0, 0, 0, 0, 0}).Words(); got != "length/time" {
		t.Errorf("Words() = %q, want length/time", got)
	}
}

func TestCelsiusIsAffine(t *testing.T) {
	c, ok := Lookup("degC")
	if !ok {
		t.Fatal("degC not found")
	}
	if c.Offset.String() != "5463/20" {
		t.Errorf("degC offset %s, want 5463/20", c.Offset)
	}
	k, _ := Lookup("K")
	got, err := Convert(num.FromInt(20), c, k)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.String() != "5863/20" {
		t.Errorf("20 degC = %s K, want 5863/20", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/units/`
Expected: FAIL — `undefined: Lookup`.

- [ ] **Step 3: Write the dimension type**

`internal/units/dim.go`:

```go
package units

import "strings"

type Dim [8]int

var dimNames = [8]string{"length", "mass", "time", "current", "temperature", "amount", "luminous intensity", "information"}

func (d Dim) Add(o Dim) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] + o[i]
	}
	return out
}

func (d Dim) Sub(o Dim) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] - o[i]
	}
	return out
}

func (d Dim) Scale(n int) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] * n
	}
	return out
}

func (d Dim) IsNone() bool { return d == Dim{} }

func (d Dim) Words() string {
	var pos, neg []string
	for i, e := range d {
		if e == 0 {
			continue
		}
		name := dimNames[i]
		mag := e
		if mag < 0 {
			mag = -mag
		}
		if mag > 1 {
			name += "^" + string(rune('0'+mag))
		}
		if e > 0 {
			pos = append(pos, name)
		} else {
			neg = append(neg, name)
		}
	}
	if len(pos) == 0 && len(neg) == 0 {
		return "dimensionless"
	}
	if len(pos) == 0 {
		return "1/" + strings.Join(neg, "·")
	}
	if len(neg) == 0 {
		return strings.Join(pos, "·")
	}
	return strings.Join(pos, "·") + "/" + strings.Join(neg, "·")
}

type Kind int

const (
	Absolute Kind = iota
	Difference
)
```

- [ ] **Step 4: Write the unit table and conversion**

`internal/units/table.go`:

```go
package units

import (
	"errors"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
)

type Unit struct {
	Symbol string
	Dim    Dim
	Factor num.Exact
	Offset num.Exact
	Kind   Kind
}

var (
	length      = Dim{1, 0, 0, 0, 0, 0, 0, 0}
	mass        = Dim{0, 1, 0, 0, 0, 0, 0, 0}
	timeDim     = Dim{0, 0, 1, 0, 0, 0, 0, 0}
	current     = Dim{0, 0, 0, 1, 0, 0, 0, 0}
	temperature = Dim{0, 0, 0, 0, 1, 0, 0, 0}
	amount      = Dim{0, 0, 0, 0, 0, 1, 0, 0}
	luminous    = Dim{0, 0, 0, 0, 0, 0, 1, 0}
	information = Dim{0, 0, 0, 0, 0, 0, 0, 1}
)

func rat(s string) num.Exact {
	v, err := num.ParseExact(s)
	if err != nil {
		panic("bad unit factor " + s)
	}
	return v
}

var base = map[string]Unit{
	"m":    {"m", length, rat("1"), rat("0"), Absolute},
	"g":    {"g", mass, rat("1/1000"), rat("0"), Absolute},
	"s":    {"s", timeDim, rat("1"), rat("0"), Absolute},
	"A":    {"A", current, rat("1"), rat("0"), Absolute},
	"K":    {"K", temperature, rat("1"), rat("0"), Absolute},
	"mol":  {"mol", amount, rat("1"), rat("0"), Absolute},
	"cd":   {"cd", luminous, rat("1"), rat("0"), Absolute},
	"bit":  {"bit", information, rat("1"), rat("0"), Absolute},
	"B":    {"B", information, rat("8"), rat("0"), Absolute},
	"inch": {"inch", length, rat("127/5000"), rat("0"), Absolute},
	"ft":   {"ft", length, rat("381/1250"), rat("0"), Absolute},
	"mile": {"mile", length, rat("201168/125"), rat("0"), Absolute},
	"lb":   {"lb", mass, rat("45359237/100000000"), rat("0"), Absolute},
	"min":  {"min", timeDim, rat("60"), rat("0"), Absolute},
	"h":    {"h", timeDim, rat("3600"), rat("0"), Absolute},
	"day":  {"day", timeDim, rat("86400"), rat("0"), Absolute},
	"degC": {"degC", temperature, rat("1"), rat("5463/20"), Absolute},
	"degF": {"degF", temperature, rat("5/9"), rat("45967/180"), Absolute},
}

var prefixes = map[string]string{
	"Y": "1e24", "Z": "1e21", "E": "1e18", "P": "1e15", "T": "1e12",
	"G": "1e9", "M": "1e6", "k": "1e3", "h": "1e2", "da": "1e1",
	"d": "1e-1", "c": "1e-2", "m": "1e-3", "u": "1e-6", "n": "1e-9",
	"p": "1e-12", "f": "1e-15", "a": "1e-18",
}

func Lookup(sym string) (Unit, bool) {
	if u, ok := base[sym]; ok {
		return u, true
	}
	for p, mul := range prefixes {
		if !strings.HasPrefix(sym, p) || len(sym) == len(p) {
			continue
		}
		rest := sym[len(p):]
		u, ok := base[rest]
		if !ok || u.Offset.Sign() != 0 {
			continue
		}
		u.Symbol = sym
		u.Factor = u.Factor.Mul(rat(mul))
		return u, true
	}
	return Unit{}, false
}

func Convert(v num.Exact, from, to Unit) (num.Exact, error) {
	if from.Dim != to.Dim {
		return num.Exact{}, errors.New("cannot convert " + from.Dim.Words() + " (" + from.Symbol + ") to " + to.Dim.Words() + " (" + to.Symbol + ")")
	}
	si := v.Mul(from.Factor).Add(from.Offset)
	out, err := si.Sub(to.Offset).Div(to.Factor)
	if err != nil {
		return num.Exact{}, err
	}
	return out, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/units/ -v`
Expected: PASS. If `TestCelsiusIsAffine` fails, check that 273.15 is stored as `5463/20` and not as a float.

- [ ] **Step 6: Commit**

```bash
git add internal/units
git commit -m "feat: dimensions, unit table and exact conversion"
```

---

### Task 5: Parser and AST

**Files:**
- Create: `internal/parse/ast.go`
- Create: `internal/parse/parse.go`
- Test: `internal/parse/parse_test.go`

**Interfaces:**
- Consumes: `lex.Scan`, `lex.Token`, `lex.Pos`.
- Produces:
  - `type Node interface{ Pos() lex.Pos; String() string }`
  - `type NumLit struct{ Text string; P lex.Pos }`
  - `type Name struct{ Text string; P lex.Pos }`
  - `type Unary struct{ Op string; X Node; P lex.Pos }`
  - `type Binary struct{ Op string; L, R Node; P lex.Pos; Implicit bool }`
  - `type Call struct{ Fn string; Args []Node; P lex.Pos }`
  - `type Assign struct{ Name string; Params []string; Body Node; P lex.Pos }`
  - `type Note struct{ Pos lex.Pos; Msg string }`
  - `func Parse(src string) (Node, []Note, error)`

- [ ] **Step 1: Write the failing test**

`internal/parse/parse_test.go`:

```go
package parse

import "testing"

func parseOK(t *testing.T, src string) Node {
	t.Helper()
	n, _, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return n
}

func TestPrecedence(t *testing.T) {
	cases := map[string]string{
		"1 + 2 * 3":  "(1 + (2 * 3))",
		"2 ^ 3 ^ 2":  "(2 ^ (3 ^ 2))",
		"-2 ^ 2":     "(-(2 ^ 2))",
		"2x":         "(2 * x)",
		"2(x + 1)":   "(2 * (x + 1))",
		"1/2x":       "((1 / 2) * x)",
		"5 m + 3 s":  "((5 * m) + (3 * s))",
		"17 mod 5":   "(17 mod 5)",
		"500 + 10%":  "(500 + (10 %))",
	}
	for src, want := range cases {
		if got := parseOK(t, src).String(); got != want {
			t.Errorf("Parse(%q) = %s, want %s", src, got, want)
		}
	}
}

func TestImplicitOverDivisionEmitsNote(t *testing.T) {
	_, notes, err := Parse("1/2x")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("got %d notes, want 1", len(notes))
	}
	if notes[0].Msg != "read as (1/2)·x" {
		t.Errorf("note = %q", notes[0].Msg)
	}
}

func TestFunctionCallNeedsParens(t *testing.T) {
	if _, _, err := Parse("sin x"); err == nil {
		t.Fatal("want error for sin x")
	}
	parseOK(t, "sin(x)")
}

func TestAssignmentAndFunctionDefinition(t *testing.T) {
	n := parseOK(t, "r = 5")
	a, ok := n.(*Assign)
	if !ok || a.Name != "r" {
		t.Fatalf("got %T %s, want assignment to r", n, n)
	}
	n = parseOK(t, "f(x) = x ^ 2")
	a, ok = n.(*Assign)
	if !ok || len(a.Params) != 1 || a.Params[0] != "x" {
		t.Fatalf("got %T %s, want f(x) definition", n, n)
	}
}

func TestErrorCarriesPosition(t *testing.T) {
	_, _, err := Parse("1 +")
	if err == nil {
		t.Fatal("want error")
	}
	if got := err.Error(); got[:4] != "1:4:" {
		t.Errorf("error %q, want it to start with 1:4:", got)
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"1+2", "sin(x)", "f(x) = x^2", "1/2x", "5 km to m", ")("} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, src string) {
		n, _, err := Parse(src)
		if err != nil {
			return
		}
		printed := n.String()
		again, _, err := Parse(printed)
		if err != nil {
			t.Fatalf("reparse of %q (from %q) failed: %v", printed, src, err)
		}
		if again.String() != printed {
			t.Fatalf("round trip: %q then %q", printed, again.String())
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/parse/`
Expected: FAIL — `undefined: Parse`.

- [ ] **Step 3: Write the AST**

`internal/parse/ast.go`:

```go
package parse

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/lex"
)

type Node interface {
	Pos() lex.Pos
	String() string
}

type NumLit struct {
	Text string
	P    lex.Pos
}

func (n *NumLit) Pos() lex.Pos  { return n.P }
func (n *NumLit) String() string { return n.Text }

type Name struct {
	Text string
	P    lex.Pos
}

func (n *Name) Pos() lex.Pos  { return n.P }
func (n *Name) String() string { return n.Text }

type Unary struct {
	Op string
	X  Node
	P  lex.Pos
}

func (n *Unary) Pos() lex.Pos { return n.P }
func (n *Unary) String() string {
	if n.Op == "%" || n.Op == "!" {
		return "(" + n.X.String() + " " + n.Op + ")"
	}
	return "(" + n.Op + n.X.String() + ")"
}

type Binary struct {
	Op       string
	L, R     Node
	P        lex.Pos
	Implicit bool
}

func (n *Binary) Pos() lex.Pos  { return n.P }
func (n *Binary) String() string { return "(" + n.L.String() + " " + n.Op + " " + n.R.String() + ")" }

type Call struct {
	Fn   string
	Args []Node
	P    lex.Pos
}

func (n *Call) Pos() lex.Pos { return n.P }
func (n *Call) String() string {
	parts := make([]string, len(n.Args))
	for i, a := range n.Args {
		parts[i] = a.String()
	}
	return n.Fn + "(" + strings.Join(parts, ", ") + ")"
}

type Assign struct {
	Name   string
	Params []string
	Body   Node
	P      lex.Pos
}

func (n *Assign) Pos() lex.Pos { return n.P }
func (n *Assign) String() string {
	if len(n.Params) == 0 {
		return n.Name + " = " + n.Body.String()
	}
	return n.Name + "(" + strings.Join(n.Params, ", ") + ") = " + n.Body.String()
}

type Note struct {
	Pos lex.Pos
	Msg string
}
```

- [ ] **Step 4: Write the Pratt parser**

`internal/parse/parse.go`:

```go
package parse

import (
	"github.com/rahuldr07/go-calc/internal/lex"
)

type parser struct {
	toks  []lex.Token
	i     int
	notes []Note
}

func Parse(src string) (Node, []Note, error) {
	toks, err := lex.Scan(src)
	if err != nil {
		return nil, nil, err
	}
	kept := toks[:0]
	for _, t := range toks {
		if t.Kind != lex.Comment {
			kept = append(kept, t)
		}
	}
	p := &parser{toks: kept}
	n, err := p.parseStatement()
	if err != nil {
		return nil, nil, err
	}
	if p.peek().Kind != lex.EOF {
		return nil, nil, p.errorAt(p.peek(), "unexpected "+p.peek().Kind.String())
	}
	return n, p.notes, nil
}

func (p *parser) peek() lex.Token  { return p.toks[p.i] }
func (p *parser) next() lex.Token  { t := p.toks[p.i]; p.i++; return t }

func (p *parser) errorAt(t lex.Token, msg string) error {
	return &lex.Error{Pos: t.Pos, Msg: msg}
}

func (p *parser) parseStatement() (Node, error) {
	start := p.i
	if p.peek().Kind == lex.Ident {
		name := p.next()
		var params []string
		if p.peek().Kind == lex.LParen {
			save := p.i
			p.next()
			ok := true
			for p.peek().Kind == lex.Ident {
				params = append(params, p.next().Text)
				if p.peek().Kind == lex.Comma {
					p.next()
					continue
				}
				break
			}
			if p.peek().Kind != lex.RParen {
				ok = false
			} else {
				p.next()
			}
			if !ok || p.peek().Text != "=" {
				p.i = save
				params = nil
			}
		}
		if p.peek().Kind == lex.Op && p.peek().Text == "=" {
			p.next()
			body, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			return &Assign{Name: name.Text, Params: params, Body: body, P: name.Pos}, nil
		}
		p.i = start
	}
	return p.parseExpr(0)
}

var binPower = map[string]int{
	"to": 5, "in": 5,
	"+": 10, "-": 10, "±": 10, "+-": 10,
	"*": 20, "/": 20, "×": 20, "÷": 20, "·": 20, "mod": 20,
	"^": 30,
}

func (p *parser) parseExpr(min int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		op := t.Text
		if t.Kind == lex.Ident && (op == "mod" || op == "to" || op == "in") {
		} else if t.Kind != lex.Op || op == "=" {
			if implicitFollows(t) && 20 >= min {
				right, err := p.parseUnaryChain(21)
				if err != nil {
					return nil, err
				}
				if div, ok := left.(*Binary); ok && div.Op == "/" && !div.Implicit {
					p.notes = append(p.notes, Note{Pos: t.Pos, Msg: "read as (" + div.L.String() + "/" + div.R.String() + ")·" + right.String()})
				}
				left = &Binary{Op: "*", L: left, R: right, P: t.Pos, Implicit: true}
				continue
			}
			return left, nil
		}
		power, ok := binPower[op]
		if !ok || power < min {
			return left, nil
		}
		p.next()
		nextMin := power + 1
		if op == "^" {
			nextMin = power
		}
		right, err := p.parseExpr(nextMin)
		if err != nil {
			return nil, err
		}
		left = &Binary{Op: op, L: left, R: right, P: t.Pos}
	}
}

func implicitFollows(t lex.Token) bool {
	switch t.Kind {
	case lex.Num, lex.LParen:
		return true
	case lex.Ident:
		return t.Text != "mod" && t.Text != "to" && t.Text != "in"
	}
	return false
}

func (p *parser) parseUnaryChain(min int) (Node, error) {
	n, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (p *parser) parseUnary() (Node, error) {
	t := p.peek()
	if t.Kind == lex.Op && (t.Text == "-" || t.Text == "+") {
		p.next()
		x, err := p.parseExpr(25)
		if err != nil {
			return nil, err
		}
		if t.Text == "+" {
			return x, nil
		}
		return &Unary{Op: "-", X: x, P: t.Pos}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Node, error) {
	n, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.Kind == lex.Op && (t.Text == "%" || t.Text == "!") {
			p.next()
			n = &Unary{Op: t.Text, X: n, P: t.Pos}
			continue
		}
		return n, nil
	}
}

func (p *parser) parsePrimary() (Node, error) {
	t := p.next()
	switch t.Kind {
	case lex.Num:
		return &NumLit{Text: t.Text, P: t.Pos}, nil
	case lex.Ident:
		if p.peek().Kind == lex.LParen {
			p.next()
			var args []Node
			if p.peek().Kind != lex.RParen {
				for {
					a, err := p.parseExpr(0)
					if err != nil {
						return nil, err
					}
					args = append(args, a)
					if p.peek().Kind == lex.Comma {
						p.next()
						continue
					}
					break
				}
			}
			if p.peek().Kind != lex.RParen {
				return nil, p.errorAt(p.peek(), "expected )")
			}
			p.next()
			return &Call{Fn: t.Text, Args: args, P: t.Pos}, nil
		}
		return &Name{Text: t.Text, P: t.Pos}, nil
	case lex.LParen:
		n, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().Kind != lex.RParen {
			return nil, p.errorAt(p.peek(), "expected )")
		}
		p.next()
		return n, nil
	}
	return nil, p.errorAt(t, "unexpected "+t.Kind.String())
}
```

- [ ] **Step 5: Run tests and fix the two known traps**

Run: `go test ./internal/parse/ -v`
Expected: PASS. Two cases usually need attention:
- `sin x` must fail: a `Name` followed by a bare `Ident` is implicit multiplication, so add a guard in `implicitFollows` — when the left node is a `*Name` whose text is a known function name (`sin cos tan asin acos atan sqrt exp log log10 log2 abs floor ceil round`), return a parse error "function call needs parentheses: did you mean sin(x)?" instead of multiplying.
- `-2 ^ 2` must print `(-(2 ^ 2))`: unary minus binds looser than `^`, which the `parseExpr(25)` call gives you; confirm with the test rather than by reading.

- [ ] **Step 6: Run the fuzzer**

Run: `go test ./internal/parse/ -run FuzzParse -fuzz FuzzParse -fuzztime 60s`
Expected: no panic and no round-trip failure. Fix any crasher before moving on; each one stays in `testdata/fuzz` as a regression test.

- [ ] **Step 7: Commit**

```bash
git add internal/parse
git commit -m "feat: Pratt parser with implicit multiplication notes"
```

---

### Task 6: Value and evaluator

**Files:**
- Create: `internal/value/value.go`
- Create: `internal/eval/env.go`
- Create: `internal/eval/eval.go`
- Test: `internal/eval/eval_test.go`

**Interfaces:**
- Consumes: `parse.Node`, `num.Exact`, `units.Unit`, `units.Dim`.
- Produces:
  - `type value.Value struct{ Num num.Exact; Dim units.Dim; Unit string; Factor num.Exact }`
  - `func eval.NewEnv() *Env`
  - `func (e *Env) Eval(n parse.Node) (value.Value, error)`
  - `func (e *Env) Set(name string, v value.Value)`

- [ ] **Step 1: Write the failing test**

`internal/eval/eval_test.go`:

```go
package eval

import (
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func run(t *testing.T, env *Env, src string) string {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	out := v.Num.String()
	if v.Unit != "" {
		out += " " + v.Unit
	}
	return out
}

func TestArithmeticIsExact(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "0.1 + 0.2"); got != "3/10" {
		t.Errorf("0.1 + 0.2 = %s, want 3/10", got)
	}
	if got := run(t, env, "(1e16 + 1) - 1e16"); got != "1" {
		t.Errorf("cancellation case = %s, want 1", got)
	}
}

func TestPercent(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "500 + 10%"); got != "550" {
		t.Errorf("500 + 10%% = %s, want 550", got)
	}
	if got := run(t, env, "10%"); got != "1/10" {
		t.Errorf("10%% = %s, want 1/10", got)
	}
}

func TestModulo(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "17 mod 5"); got != "2" {
		t.Errorf("17 mod 5 = %s, want 2", got)
	}
}

func TestUnitsAndConversion(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "3 km to m"); got != "3000 m" {
		t.Errorf("3 km to m = %s, want 3000 m", got)
	}
	if got := run(t, env, "5 km + 300 m"); got != "5.3 km" && got != "53/10 km" {
		t.Errorf("5 km + 300 m = %s, want 53/10 km", got)
	}
}

func TestDimensionMismatchNamesBoth(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("5 m + 3 s")
	_, err := env.Eval(n)
	if err == nil {
		t.Fatal("want error")
	}
	if want := "cannot add length (m) and time (s)"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestUnitPositionBeatsVariable(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 3")
	if got := run(t, env, "5 h"); got != "5 h" {
		t.Errorf("5 h = %s, want 5 h (hours), not 15", got)
	}
	if got := run(t, env, "h + 1"); got != "4" {
		t.Errorf("h + 1 = %s, want 4", got)
	}
}

func TestVariablesAndFunctions(t *testing.T) {
	env := NewEnv()
	run(t, env, "f(x) = x ^ 2 + 1")
	if got := run(t, env, "f(3)"); got != "10" {
		t.Errorf("f(3) = %s, want 10", got)
	}
}

func TestIrrationalIsRefusedNotApproximated(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("sqrt(2)")
	if _, err := env.Eval(n); err == nil {
		t.Fatal("sqrt(2) must be refused until ball arithmetic lands")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/eval/`
Expected: FAIL — `undefined: NewEnv`.

- [ ] **Step 3: Write the value type**

`internal/value/value.go`:

```go
package value

import (
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/units"
)

type Value struct {
	Num    num.Exact
	Dim    units.Dim
	Unit   string
	Factor num.Exact
}

func Scalar(n num.Exact) Value {
	return Value{Num: n, Factor: num.FromInt(1)}
}

func WithUnit(n num.Exact, u units.Unit) Value {
	return Value{Num: n, Dim: u.Dim, Unit: u.Symbol, Factor: u.Factor}
}

func (v Value) IsScalar() bool { return v.Dim.IsNone() && v.Unit == "" }

func (v Value) SI() num.Exact {
	if v.Factor.Sign() == 0 {
		return v.Num
	}
	return v.Num.Mul(v.Factor)
}
```

- [ ] **Step 4: Write the environment and evaluator**

`internal/eval/env.go`:

```go
package eval

import (
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

type fn struct {
	params []string
	body   parse.Node
}

type Env struct {
	vars  map[string]value.Value
	funcs map[string]fn
}

func NewEnv() *Env {
	return &Env{vars: map[string]value.Value{}, funcs: map[string]fn{}}
}

func (e *Env) Set(name string, v value.Value) { e.vars[name] = v }

func (e *Env) Get(name string) (value.Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}
```

`internal/eval/eval.go`:

```go
package eval

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func (e *Env) Eval(n parse.Node) (value.Value, error) {
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
		}
		return value.Scalar(x), nil
	case *parse.Name:
		if v, ok := e.vars[t.Text]; ok {
			return v, nil
		}
		if u, ok := units.Lookup(t.Text); ok {
			return value.WithUnit(num.FromInt(1), u), nil
		}
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "unknown name " + t.Text}
	case *parse.Assign:
		if len(t.Params) > 0 {
			e.funcs[t.Name] = fn{params: t.Params, body: t.Body}
			return value.Scalar(num.FromInt(0)), nil
		}
		v, err := e.Eval(t.Body)
		if err != nil {
			return value.Value{}, err
		}
		e.vars[t.Name] = v
		return v, nil
	case *parse.Call:
		f, ok := e.funcs[t.Fn]
		if !ok {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: t.Fn + " needs ball arithmetic, which is not implemented yet"}
		}
		if len(f.params) != len(t.Args) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "wrong number of arguments"}
		}
		inner := NewEnv()
		for k, v := range e.vars {
			inner.vars[k] = v
		}
		for k, v := range e.funcs {
			inner.funcs[k] = v
		}
		for i, p := range f.params {
			a, err := e.Eval(t.Args[i])
			if err != nil {
				return value.Value{}, err
			}
			inner.vars[p] = a
		}
		return inner.Eval(f.body)
	case *parse.Unary:
		return e.evalUnary(t)
	case *parse.Binary:
		return e.evalBinary(t)
	}
	return value.Value{}, errors.New("unsupported expression")
}

func (e *Env) evalUnary(t *parse.Unary) (value.Value, error) {
	x, err := e.Eval(t.X)
	if err != nil {
		return value.Value{}, err
	}
	switch t.Op {
	case "-":
		x.Num = x.Num.Neg()
		return x, nil
	case "%":
		if !x.IsScalar() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "percent needs a plain number"}
		}
		q, err := x.Num.Div(num.FromInt(100))
		if err != nil {
			return value.Value{}, err
		}
		return value.Scalar(q), nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unsupported operator " + t.Op}
}

func (e *Env) evalBinary(t *parse.Binary) (value.Value, error) {
	if t.Implicit {
		if name, ok := t.R.(*parse.Name); ok {
			if _, isVar := e.vars[name.Text]; !isVar {
				if u, ok := units.Lookup(name.Text); ok {
					l, err := e.Eval(t.L)
					if err != nil {
						return value.Value{}, err
					}
					if !l.IsScalar() {
						return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot attach " + name.Text + " to a value that already has a unit"}
					}
					return value.WithUnit(l.Num, u), nil
				}
			}
		}
	}
	if t.Op == "to" || t.Op == "in" {
		return e.convert(t)
	}
	l, err := e.Eval(t.L)
	if err != nil {
		return value.Value{}, err
	}
	r, err := e.Eval(t.R)
	if err != nil {
		return value.Value{}, err
	}
	switch t.Op {
	case "+", "-", "±", "+-":
		if r.IsScalar() && !l.IsScalar() && isPercentNode(t.R) {
			scaled := l
			scaled.Num = l.Num.Mul(num.FromInt(1).Add(r.Num))
			return scaled, nil
		}
		if l.IsScalar() && isPercentNode(t.R) {
			out := l.Num.Mul(num.FromInt(1).Add(r.Num))
			return value.Scalar(out), nil
		}
		if l.Dim != r.Dim {
			verb := "add"
			if t.Op == "-" {
				verb = "subtract"
			}
			return value.Value{}, errors.New("cannot " + verb + " " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")")
		}
		rInL, err := r.SI().Div(nonZero(l.Factor))
		if err != nil {
			return value.Value{}, err
		}
		out := l
		if t.Op == "-" {
			out.Num = l.Num.Sub(rInL)
		} else {
			out.Num = l.Num.Add(rInL)
		}
		return out, nil
	case "*", "×", "·":
		return value.Value{Num: l.SI().Mul(r.SI()), Dim: l.Dim.Add(r.Dim), Unit: mergeUnit(l, r, "·"), Factor: num.FromInt(1)}, nil
	case "/", "÷":
		q, err := l.SI().Div(r.SI())
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "division by zero"}
		}
		return value.Value{Num: q, Dim: l.Dim.Sub(r.Dim), Unit: mergeUnit(l, r, "/"), Factor: num.FromInt(1)}, nil
	case "mod":
		if !l.IsScalar() || !r.IsScalar() || !l.Num.IsInt() || !r.Num.IsInt() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "mod needs whole numbers"}
		}
		q, err := l.Num.Div(r.Num)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "division by zero"}
		}
		whole := num.FromRat(new(bigIntRat(q)).SetInt(q.Rat().Num()))
		_ = whole
		return value.Scalar(modExact(l.Num, r.Num)), nil
	case "^":
		if !r.IsScalar() || !r.Num.IsInt() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "only whole exponents are supported until ball arithmetic lands"}
		}
		exp := int(r.Num.Rat().Num().Int64())
		return value.Value{Num: l.Num.Pow(exp), Dim: l.Dim.Scale(exp), Unit: l.Unit, Factor: l.Factor.Pow(exp)}, nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unsupported operator " + t.Op}
}

func isPercentNode(n parse.Node) bool {
	u, ok := n.(*parse.Unary)
	return ok && u.Op == "%"
}

func unitOrNone(v value.Value) string {
	if v.Unit == "" {
		return "no unit"
	}
	return v.Unit
}

func nonZero(e num.Exact) num.Exact {
	if e.Sign() == 0 {
		return num.FromInt(1)
	}
	return e
}

func mergeUnit(l, r value.Value, sep string) string {
	switch {
	case l.Unit == "" && r.Unit == "":
		return ""
	case l.Unit == "":
		return r.Unit
	case r.Unit == "":
		return l.Unit
	}
	return l.Unit + sep + r.Unit
}

func (e *Env) convert(t *parse.Binary) (value.Value, error) {
	l, err := e.Eval(t.L)
	if err != nil {
		return value.Value{}, err
	}
	name, ok := t.R.(*parse.Name)
	if !ok {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "convert to what unit?"}
	}
	to, ok := units.Lookup(name.Text)
	if !ok {
		return value.Value{}, &lex.Error{Pos: name.P, Msg: "unknown unit " + name.Text}
	}
	from := units.Unit{Symbol: l.Unit, Dim: l.Dim, Factor: nonZero(l.Factor)}
	out, err := units.Convert(l.Num, from, to)
	if err != nil {
		return value.Value{}, err
	}
	return value.WithUnit(out, to), nil
}
```

- [ ] **Step 5: Replace the placeholder mod helper**

The `mod` branch above contains a stub that will not compile (`bigIntRat`). Delete those two lines and add this helper at the bottom of `internal/eval/eval.go`:

```go
func modExact(a, b num.Exact) num.Exact {
	ai := a.Rat().Num()
	bi := b.Rat().Num()
	m := new(big.Int).Mod(ai, bi)
	return num.FromRat(new(big.Rat).SetInt(m))
}
```

Add `"math/big"` to the import block.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/eval/ -v`
Expected: PASS. `TestUnitPositionBeatsVariable` is the one that proves the resolution rule; if it fails, the implicit-multiplication branch in `evalBinary` is not running before variable lookup.

- [ ] **Step 7: Commit**

```bash
git add internal/value internal/eval
git commit -m "feat: exact evaluator with units, percent and unit-position resolution"
```

---

### Task 7: Constants from CODATA

**Files:**
- Create: `internal/consts/consts.go`
- Create: `internal/consts/codata2022.txt`
- Test: `internal/consts/consts_test.go`
- Modify: `internal/eval/eval.go` (name resolution falls through to constants)

**Interfaces:**
- Consumes: `num.Exact`, `units.Unit`.
- Produces:
  - `type Const struct{ Name, Symbol, Unit, Source string; Value num.Exact; Uncertainty num.Exact; Exact bool }`
  - `func Lookup(sym string) (Const, bool)`
  - `func Checksum() string`

- [ ] **Step 1: Write the failing test**

`internal/consts/consts_test.go`:

```go
package consts

import "testing"

func TestLookupExactConstant(t *testing.T) {
	c, ok := Lookup("c")
	if !ok {
		t.Fatal("c not found")
	}
	if !c.Exact {
		t.Error("speed of light must be exact")
	}
	if c.Value.String() != "299792458" {
		t.Errorf("c = %s, want 299792458", c.Value)
	}
	if c.Unit != "m/s" {
		t.Errorf("unit %q, want m/s", c.Unit)
	}
}

func TestLookupMeasuredConstantCarriesUncertainty(t *testing.T) {
	g, ok := Lookup("G")
	if !ok {
		t.Fatal("G not found")
	}
	if g.Exact {
		t.Error("G is measured, not exact")
	}
	if g.Uncertainty.Sign() == 0 {
		t.Error("G must carry an uncertainty")
	}
	if g.Source == "" {
		t.Error("G must carry a citation")
	}
}

func TestChecksumIsStable(t *testing.T) {
	if got := Checksum(); got != "REPLACE_AFTER_FIRST_RUN" {
		t.Errorf("checksum = %q; paste this value into the test once the table is final", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/consts/`
Expected: FAIL — `undefined: Lookup`.

- [ ] **Step 3: Write the data file**

`internal/consts/codata2022.txt` — tab-separated: symbol, name, value, uncertainty, unit, exact flag.

```
c	speed of light in vacuum	299792458	0	m/s	exact
h	Planck constant	662607015e-42	0	J·s	exact
e_charge	elementary charge	1602176634e-28	0	C	exact
k_B	Boltzmann constant	1380649e-29	0	J/K	exact
N_A	Avogadro constant	602214076e15	0	1/mol	exact
g0	standard acceleration of gravity	980665e-5	0	m/s^2	exact
G	Newtonian constant of gravitation	667430e-16	15e-16	m^3/(kg·s^2)	measured
m_e	electron mass	91093837139e-42	28e-42	kg	measured
alpha	fine-structure constant	72973525643e-13	11e-13		measured
```

- [ ] **Step 4: Write the loader**

`internal/consts/consts.go`:

```go
package consts

import (
	_ "embed"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
)

//go:embed codata2022.txt
var raw string

const Source = "CODATA 2022 · physics.nist.gov/cuu/Constants"

type Const struct {
	Name        string
	Symbol      string
	Unit        string
	Source      string
	Value       num.Exact
	Uncertainty num.Exact
	Exact       bool
}

var table = map[string]Const{}

func init() {
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 6 {
			continue
		}
		v, err := num.ParseExact(f[2])
		if err != nil {
			continue
		}
		u, err := num.ParseExact(f[3])
		if err != nil {
			continue
		}
		table[f[0]] = Const{
			Symbol:      f[0],
			Name:        f[1],
			Value:       v,
			Uncertainty: u,
			Unit:        f[4],
			Exact:       f[5] == "exact",
			Source:      Source,
		}
	}
}

func Lookup(sym string) (Const, bool) {
	c, ok := table[sym]
	return c, ok
}

func Checksum() string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}
```

- [ ] **Step 5: Run, then pin the checksum**

Run: `go test ./internal/consts/ -v`
Expected: the first two tests PASS; `TestChecksumIsStable` FAILS and prints the real checksum. Paste that value into the test in place of `REPLACE_AFTER_FIRST_RUN`, then re-run — all three PASS.

- [ ] **Step 6: Wire constants into name resolution**

In `internal/eval/eval.go`, inside `case *parse.Name`, between the variable lookup and the unit lookup, add:

```go
		if c, ok := consts.Lookup(t.Text); ok {
			if c.Unit == "" {
				return value.Scalar(c.Value), nil
			}
			if u, ok := units.Lookup(c.Unit); ok {
				return value.WithUnit(c.Value, u), nil
			}
			return value.Scalar(c.Value), nil
		}
```

Add the `consts` import. Then add this test to `internal/eval/eval_test.go` and run the package:

```go
func TestConstantsResolve(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "c"); got != "299792458 m/s" {
		t.Errorf("c = %s, want 299792458 m/s", got)
	}
}
```

Run: `go test ./internal/eval/ ./internal/consts/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/consts internal/eval
git commit -m "feat: embedded CODATA 2022 constants with citations"
```

---

### Task 8: Formatter and CLI

**Files:**
- Create: `internal/format/text.go`
- Create: `internal/format/json.go`
- Test: `internal/format/format_test.go`
- Create: `cmd/gocalc/main.go`
- Test: `cmd/gocalc/main_test.go`

**Interfaces:**
- Consumes: `value.Value`, `parse.Note`.
- Produces:
  - `func format.Text(v value.Value, notes []parse.Note) string`
  - `func format.JSON(input string, v value.Value, notes []parse.Note) ([]byte, error)`
  - A `gocalc` binary: `gocalc "expr"`, `gocalc --json "expr"`, exit code 1 on error.

- [ ] **Step 1: Write the failing test**

`internal/format/format_test.go`:

```go
package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func TestTextShowsExactAndDecimal(t *testing.T) {
	v := value.Scalar(mustRat(t, "1/3"))
	got := Text(v, nil)
	if !strings.Contains(got, "1/3") || !strings.Contains(got, "0.333333333333") {
		t.Errorf("Text = %q, want the fraction and a decimal", got)
	}
}

func TestTextKeepsWholeNumbersPlain(t *testing.T) {
	if got := Text(value.Scalar(num.FromInt(550)), nil); got != "550" {
		t.Errorf("Text = %q, want 550", got)
	}
}

func TestTextShowsUnit(t *testing.T) {
	u, _ := units.Lookup("m")
	if got := Text(value.WithUnit(num.FromInt(3000), u), nil); got != "3000 m" {
		t.Errorf("Text = %q, want 3000 m", got)
	}
}

func TestJSONSchema(t *testing.T) {
	b, err := JSON("3 km to m", value.Scalar(num.FromInt(3000)), nil)
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["schema"] != "go-calc.result/1" {
		t.Errorf("schema = %v", out["schema"])
	}
	if out["input"] != "3 km to m" {
		t.Errorf("input = %v", out["input"])
	}
}

func mustRat(t *testing.T, s string) num.Exact {
	t.Helper()
	v, err := num.ParseExact(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/format/`
Expected: FAIL — `undefined: Text`.

- [ ] **Step 3: Write the formatter**

`internal/format/text.go`:

```go
package format

import (
	"math/big"
	"strings"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const digits = 12

func Text(v value.Value, notes []parse.Note) string {
	var b strings.Builder
	b.WriteString(number(v))
	if v.Unit != "" {
		b.WriteString(" " + v.Unit)
	}
	for _, n := range notes {
		b.WriteString("\n  read: " + n.Msg)
	}
	return b.String()
}

func number(v value.Value) string {
	r := v.Num.Rat()
	if r.IsInt() {
		return r.Num().String()
	}
	exact := r.RatString()
	dec := r.FloatString(digits)
	dec = strings.TrimRight(dec, "0")
	dec = strings.TrimSuffix(dec, ".")
	if terminates(r) {
		return dec
	}
	return exact + "  ≈ " + dec
}

func terminates(r *big.Rat) bool {
	d := new(big.Int).Set(r.Denom())
	for _, p := range []int64{2, 5} {
		bp := big.NewInt(p)
		for {
			q, m := new(big.Int).QuoRem(d, bp, new(big.Int))
			if m.Sign() != 0 {
				break
			}
			d = q
		}
	}
	return d.Cmp(big.NewInt(1)) == 0
}
```

`internal/format/json.go`:

```go
package format

import (
	"encoding/json"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

type result struct {
	Schema string   `json:"schema"`
	Input  string   `json:"input"`
	Value  valueOut `json:"value"`
	Unit   string   `json:"unit,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

type valueOut struct {
	Exact   string `json:"exact"`
	Decimal string `json:"decimal"`
}

func JSON(input string, v value.Value, notes []parse.Note) ([]byte, error) {
	msgs := make([]string, 0, len(notes))
	for _, n := range notes {
		msgs = append(msgs, n.Msg)
	}
	r := result{
		Schema: "go-calc.result/1",
		Input:  input,
		Value: valueOut{
			Exact:   v.Num.String(),
			Decimal: v.Num.Rat().FloatString(digits),
		},
		Unit:  v.Unit,
		Notes: msgs,
	}
	return json.MarshalIndent(r, "", "  ")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/format/ -v`
Expected: PASS

- [ ] **Step 5: Write the CLI test**

`cmd/gocalc/main_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestRunOneShot(t *testing.T) {
	out, code := run([]string{"500 + 10%"}, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if strings.TrimSpace(out) != "550" {
		t.Errorf("out = %q, want 550", out)
	}
}

func TestRunJSON(t *testing.T) {
	out, code := run([]string{"3 km to m"}, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, `"schema": "go-calc.result/1"`) {
		t.Errorf("out = %q", out)
	}
}

func TestRunErrorExitsNonZero(t *testing.T) {
	out, code := run([]string{"5 m + 3 s"}, false)
	if code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(out, "cannot add length (m) and time (s)") {
		t.Errorf("out = %q", out)
	}
}
```

- [ ] **Step 6: Write the CLI**

`cmd/gocalc/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
)

func main() {
	asJSON := flag.Bool("json", false, "print machine-readable output")
	flag.Parse()
	out, code := run(flag.Args(), *asJSON)
	if code == 0 {
		fmt.Println(out)
	} else {
		fmt.Fprintln(os.Stderr, out)
	}
	os.Exit(code)
}

func run(args []string, asJSON bool) (string, int) {
	if len(args) == 0 {
		return "usage: gocalc [--json] \"expression\"", 2
	}
	src := strings.Join(args, " ")
	node, notes, err := parse.Parse(src)
	if err != nil {
		return err.Error(), 1
	}
	v, err := eval.NewEnv().Eval(node)
	if err != nil {
		return err.Error(), 1
	}
	if asJSON {
		b, err := format.JSON(src, v, notes)
		if err != nil {
			return err.Error(), 1
		}
		return string(b), 0
	}
	return format.Text(v, notes), 0
}
```

- [ ] **Step 7: Run the whole suite**

Run: `go test ./... && go vet ./...`
Expected: PASS everywhere.

- [ ] **Step 8: Try it by hand**

```bash
go run ./cmd/gocalc "0.1 + 0.2"
go run ./cmd/gocalc "3 km to m"
go run ./cmd/gocalc "1/2x"
go run ./cmd/gocalc --json "500 + 10%"
go run ./cmd/gocalc "5 m + 3 s"; echo "exit $?"
```

Expected: `3/10`; `3000 m`; the interpretation note; JSON with the schema field; the dimension error with exit 1.

- [ ] **Step 9: Commit**

```bash
git add internal/format cmd/gocalc
git commit -m "feat: text and JSON formatting with the gocalc CLI"
```

---

## What this plan deliberately leaves out

These are the next plans, in order:

1. **Ball arithmetic and elementary functions** — `internal/num/ball.go`, `internal/elem`, the precision-retry loop and the digit-guarantee printer. Until it lands, `sqrt(2)` is refused rather than approximated, which Task 6's last test pins.
2. **Uncertainty** — `internal/unc`, forward-mode sensitivities, contribution budgets, `expand`.
3. **Provenance and the float64 shadow** — `internal/prov`, `trace`, `why`.
4. **REPL and documents** — history, completion, `.calc` files, incremental re-evaluation.
5. **WASM bridge and the Svelte front end** — built to `DESIGN.md`, from `web/v4.html`.

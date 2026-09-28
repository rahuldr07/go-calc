package eval

import (
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
)

func spelled(t *testing.T, env *Env, src string) string {
	t.Helper()
	return format.Default.Text(mustEval(t, env, src), nil)
}

func TestANegativeExponentKeepsTheFollowingFactor(t *testing.T) {
	cases := map[string]string{
		"10^-3 m":       "0.001 m",
		"9.8 * 10^-3 m": "0.0098 m",
		"2^-2 kg":       "0.25 kg",
		"2^-1 3":        "1.5",
		"2^-1 sqrt(4)":  "1",
		"2^-1 2^2":      "2",
		"-2^2":          "-4",
		"-2 m":          "-2 m",
	}
	for src, want := range cases {
		if got := spelled(t, NewEnv(), src); got != want {
			t.Errorf("%q = %q, want %q", src, got, want)
		}
	}
	if got := spelled(t, NewEnv(), "2^-1 pi"); !strings.HasPrefix(got, "1.5707963267") {
		t.Errorf("2^-1 pi = %q, want half of pi", got)
	}
}

func TestAUnarySignBindsTighterThanModAndOf(t *testing.T) {
	cases := map[string]string{
		"-7 mod 3":       "2",
		"-7 mod -3":      "2",
		"-2 mod 3":       "1",
		"7 mod 3":        "1",
		"(-7) mod 3":     "2",
		"-(7 mod 3)":     "-1",
		"-20% of 500":    "-100",
		"sqrt(-7 mod 3)": "1.41421356237",
	}
	for src, want := range cases {
		if got := spelled(t, NewEnv(), src); got != want {
			t.Errorf("%q = %q, want %q", src, got, want)
		}
	}
	if got := spelled(t, NewEnv(), "2 mod -2 m"); got != "0 m" {
		t.Errorf("2 mod -2 m = %q, want 0 m", got)
	}
}

func TestParenthesesCloseADenominator(t *testing.T) {
	env := NewEnv()
	cases := map[string]string{
		"(1/2 h) m":  "1/7200  ≈ 0.000138888888889 (1/s)·m",
		"(-1/2 h) m": "-1/7200  ≈ -0.000138888888889 (1/s)·m",
		"1/2 h m":    "1/7200  ≈ 0.000138888888889 (1/s)/m",
		"1/2 h":      "1/7200  ≈ 0.000138888888889 1/s",
	}
	for src, want := range cases {
		if got := spelled(t, env, src); got != want {
			t.Errorf("%q = %q, want %q", src, got, want)
		}
	}
}

func TestTheShadowReadsANameTheWayTheEvaluatorDid(t *testing.T) {
	cases := []struct {
		setup []string
		src   string
	}{
		{[]string{"f(h) = 1/2 h"}, "f(5 s)"},
		{[]string{"h = 12.3 ± 0.2 m", "g(h) = 1/2 h"}, "g(3)"},
		{[]string{"k(h) = 1/2 h"}, "k(12.3 ± 0.2)"},
		{[]string{"x = 3 ± 1", "f(h) = 1/2 h"}, "f(x*0 + 3)"},
		{[]string{"x = 3 ± 1", "f(h) = 1/2 h"}, "f(x - x + 3)"},
		{[]string{"x = 3 ± 1", "f(h) = 1/2 h"}, "f(x/x*3)"},
		{[]string{"f(h) = 1/2 h", "g(t) = t + (1 ± 0.5)"}, "f(g(2))"},
		{[]string{"f(h) = 1/2 h^2"}, "f(3)"},
		{[]string{"ke(m, v) = 1/2 m v^2", "outer(m, v) = ke(m, v)"}, "outer(2.0 ± 0.1, 3) / outer(2.0, 3)"},
		{[]string{"f(h) = 1/2 h", "g(x) = f(x)"}, "g(3) * g(12.3 ± 0.2)"},
		{[]string{"f(h) = 1/2 h", "g(x) = f(x)"}, "g(12.3 ± 0.2) * g(3)"},
		{[]string{"h = 3"}, "1/2 h"},
		{nil, "100 km / 2 h"},
	}
	for _, c := range cases {
		env := NewEnv()
		for _, line := range c.setup {
			mustEval(t, env, line)
		}
		n, _, err := parse.Parse(c.src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.src, err)
		}
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("Eval(%q): %v", c.src, err)
		}
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%q after %v warned %q; the shadow read it differently from the evaluator", c.src, c.setup, w)
		}
	}
}

func TestProvenanceNamesTheOperationThatWasTaken(t *testing.T) {
	env := NewEnv()
	tree := env.Arena().Tree(mustEval(t, env, "100 km / 2 h").Prov)
	if head := strings.SplitN(tree, "\n", 2)[0]; !strings.HasPrefix(head, "/") {
		t.Errorf("trace of a speed starts %q, want the division the evaluator took:\n%s", head, tree)
	}
	env = NewEnv()
	tree = env.Arena().Tree(mustEval(t, env, "2 m").Prov)
	if strings.HasPrefix(tree, "/") {
		t.Errorf("trace of a plain product starts with a division:\n%s", tree)
	}
}

func TestOneCallNodeReachedTwiceKeepsEachReading(t *testing.T) {
	env := NewEnv()
	for _, line := range []string{"f(h) = 1/2 h", "g(x) = f(x)"} {
		mustEval(t, env, line)
	}
	n, _, err := parse.Parse("g(3) * g(12.3 ± 0.2)")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	x := env.Why(n, v)
	if x.Warning != "" {
		t.Errorf("warned on a correct answer: %s", x.Warning)
	}
	if x.Digits < 15 {
		t.Errorf("the shadow answered %g against the proven %s, agreeing to %g digits", x.Shadow, format.Default.Text(v, nil), x.Digits)
	}
}

func TestTheConstantsHoldTheirDefiningRelations(t *testing.T) {
	zero := map[string]string{
		"N_A * k_B - R":     "0 (1/mol)·(J/K)",
		"h / (2 pi) - hbar": "0 J·s",
		"hbar * 2 pi - h":   "0 J·s",
	}
	for src, want := range zero {
		got := spelled(t, NewEnv(), src)
		if head := strings.SplitN(got, "\n", 2)[0]; head != want {
			t.Errorf("%q = %q, want %q; the table does not hold the relation it claims", src, head, want)
		}
	}
}

func TestBothParenthesisationsOfADenominatorAgree(t *testing.T) {
	pairs := [][2]string{
		{"1/(2 h) m", "(1/2 h) m"},
		{"1/(h) m", "(1/h) m"},
		{"1/(2 m) s", "(1/2 m) s"},
		{"1/(2 kg) m", "(1/2 kg) m"},
	}
	for _, pair := range pairs {
		a, b := spelled(t, NewEnv(), pair[0]), spelled(t, NewEnv(), pair[1])
		if strings.SplitN(a, "\n", 2)[0] != strings.SplitN(b, "\n", 2)[0] {
			t.Errorf("%q = %q but %q = %q; parentheses must close a denominator either way", pair[0], a, pair[1], b)
		}
	}
	if got := spelled(t, NewEnv(), "1/(2) m"); got != "0.5 m" {
		t.Errorf("1/(2) m = %q, want 0.5 m", got)
	}
}

func TestEveryNoteReadsBackToTheAnswerItDescribes(t *testing.T) {
	lines := []string{
		"1/2 h", "1/2 m s", "-(1/2 m) s", "-(1/2 h) m", "(-1/2 h) m", "(1/2 m) s",
		"100 km / 2 h", "100 W / 2 m^2", "3 m / 2 s / 4 s", "1/2 h m",
		"2 m^2", "2 pi^2", "1/2 m s + 1/2 m s", "sqrt(1/2 h m)", "-(1/2 m s)",
	}
	for _, src := range lines {
		env := NewEnv()
		n, read, err := parse.Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		v, err := env.Eval(n)
		if err != nil {
			continue
		}
		for _, note := range append(read, env.Notes()...) {
			said := strings.TrimPrefix(note.Msg, "read as ")
			if !strings.HasPrefix(note.Msg, "read as ") || strings.Contains(said, " the ") {
				continue
			}
			again, _, err := parse.Parse(said)
			if err != nil {
				t.Errorf("%q said %q, which does not parse: %v", src, note.Msg, err)
				continue
			}
			w, err := NewEnv().Eval(again)
			if err != nil {
				t.Errorf("%q said %q, which does not evaluate: %v", src, note.Msg, err)
				continue
			}
			if w.Dim != v.Dim {
				t.Errorf("%q answered %s but says it read %q, which is %s", src, v.Dim.Words(), said, w.Dim.Words())
				continue
			}
			if format.Default.Number(w) != format.Default.Number(v) {
				t.Errorf("%q answered %q but says it read %q, which is %q", src, format.Default.Text(v, nil), said, format.Default.Text(w, nil))
			}
		}
	}
}

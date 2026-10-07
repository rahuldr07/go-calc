package eval

import (
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func evalSrc(env *Env, src string) (value.Value, error) {
	n, _, err := parse.Parse(src)
	if err != nil {
		return value.Value{}, err
	}
	return env.Eval(n)
}

func solutionOf(t *testing.T, src string) *value.Solution {
	t.Helper()
	v := mustEval(t, NewEnv(), src)
	if v.Solved == nil {
		t.Fatalf("%s did not produce a solution", src)
	}
	return v.Solved
}

func exactRoots(t *testing.T, sol *value.Solution) []string {
	t.Helper()
	out := make([]string, len(sol.Roots))
	for i, r := range sol.Roots {
		if !r.IsExact() {
			t.Fatalf("root %d = %s, want exact", i, r)
		}
		out[i] = r.Exact().String()
	}
	return out
}

func TestComparisonsAreDefinite(t *testing.T) {
	env := NewEnv()
	cases := map[string]bool{
		"2 > 1":              true,
		"2 < 1":              false,
		"2 >= 2":             true,
		"2 <= 1":             false,
		"1/3 == 2/6":         true,
		"1/3 != 1/3":         false,
		"0.1 + 0.2 == 0.3":   true,
		"1 + 1 < 3 * 4":      true,
		"3 km > 2999 m":      true,
		"1 m < 99 cm":        false,
		"pi > 3.14159":       true,
		"pi < 3.14159":       false,
		"sqrt(2) < 1.5":      true,
		"sqrt(2) == 1.5":     false,
		"sqrt(2) != 1.5":     true,
		"(1 < 2) == (2 < 3)": true,
	}
	for src, want := range cases {
		v := mustEval(t, env, src)
		if !v.IsBoolean() || v.Holds() != want {
			t.Errorf("%s = truth %v, want %v", src, v.Truth, want)
		}
	}
}

func TestOverlappingBallsAreUndecidedNotGuessed(t *testing.T) {
	env := NewEnv()
	for _, src := range []string{
		"sqrt(2) * sqrt(2) == 2",
		"sqrt(2) * sqrt(2) >= 2",
		"sqrt(2) * sqrt(2) < 2",
		"sin(pi) == 0",
		"exp(ln(5)) != 5",
	} {
		v, err := evalSrc(env, src)
		if err == nil {
			t.Errorf("%s was decided (truth %v) although the sides cannot be separated", src, v.Truth)
			continue
		}
		if !strings.Contains(err.Error(), "undecided") {
			t.Errorf("%s error = %q, want it to say undecided", src, err)
		}
	}
}

func TestUndecidedComparisonRetriesAtHigherPrecision(t *testing.T) {
	v := mustEval(t, NewEnv(), "sqrt(2) * sqrt(2) > 1.999999999999999999999999999999999999999")
	if !v.IsBoolean() || !v.Holds() {
		t.Errorf("a gap far below the starting precision should still be decided, got truth %v", v.Truth)
	}
}

func TestComparisonRefusesWhatItCannotProve(t *testing.T) {
	env := NewEnv()
	for src, want := range map[string]string{
		"3 m < 2 s":   "cannot compare",
		"(5 ± 1) > 3": "uncertainty",
		"(1 < 2) < 3": "true/false",
		"(1 < 2) + 1": "true/false",
		"-(1 < 2)":    "true/false",
	} {
		_, err := evalSrc(env, src)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s error = %v, want one mentioning %q", src, err, want)
		}
	}
}

func TestSolveLinearIsExact(t *testing.T) {
	for src, want := range map[string]string{
		"solve(2x + 6 = 0, x)":       "-3",
		"solve(3x = 1, x)":           "1/3",
		"solve(x/4 - 1/2 = 0, x)":    "2",
		"solve(2(x - 1) = x + 5, x)": "7",
	} {
		sol := solutionOf(t, src)
		if got := exactRoots(t, sol); len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want [%s]", src, got, want)
		}
	}
}

func TestSolveQuadraticWithPerfectSquareDiscriminantIsExact(t *testing.T) {
	for src, want := range map[string][]string{
		"solve(x^2 - 4 = 0, x)":      {"-2", "2"},
		"solve(x^2 - 5x + 6 = 0, x)": {"2", "3"},
		"solve(2x^2 - x - 1 = 0, x)": {"-1/2", "1"},
		"solve(-x^2 + 1 = 0, x)":     {"-1", "1"},
		"solve(x^2 = 9, x)":          {"-3", "3"},
		"solve(x^2 - 2x + 1 = 0, x)": {"1"},
	} {
		got := exactRoots(t, solutionOf(t, src))
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s = %v, want %v", src, got, want)
		}
	}
}

func TestSolveIrrationalQuadraticRootsAreProvenBalls(t *testing.T) {
	sol := solutionOf(t, "solve(x^2 - 2 = 0, x)")
	if len(sol.Roots) != 2 {
		t.Fatalf("roots = %d, want 2", len(sol.Roots))
	}
	for i, r := range sol.Roots {
		if r.IsExact() {
			t.Fatalf("root %d presented as exact: %s", i, r)
		}
		if d := r.Ball(120).GuaranteedDigits(); d < 25 {
			t.Errorf("root %d guarantees only %d digits", i, d)
		}
	}
	root := sol.Roots[1].Ball(120)
	two := num.BallFromExact(num.FromInt(2), 120)
	square := root.Mul(root, 120)
	diff := square.Sub(two, 120)
	if !diff.ContainsZero() {
		t.Errorf("the proven ball for sqrt(2) does not square to something containing 2: %s", diff)
	}
	if sol.Roots[0].Ball(120).Mid.Sign() >= 0 || sol.Roots[1].Ball(120).Mid.Sign() <= 0 {
		t.Error("roots are not in ascending order around zero")
	}
}

func TestSolveNegativeDiscriminantHasNoRealRoot(t *testing.T) {
	_, err := evalSrc(NewEnv(), "solve(x^2 + 1 = 0, x)")
	if err == nil || !strings.Contains(err.Error(), "no real root") || !strings.Contains(err.Error(), "-4") {
		t.Errorf("error = %v, want a plain no-real-root message naming the discriminant", err)
	}
}

func TestSolveDegenerateEquations(t *testing.T) {
	env := NewEnv()
	for src, want := range map[string]string{
		"solve(x + 1 = x + 2, x)": "no solution",
		"solve(x + 1 = 1 + x, x)": "every value",
	} {
		_, err := evalSrc(env, src)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s error = %v, want one mentioning %q", src, err, want)
		}
	}
}

func TestSolveCubicFindsEveryRootInRange(t *testing.T) {
	sol := solutionOf(t, "solve(x^3 - x = 0, x)")
	if got := exactRoots(t, sol); strings.Join(got, ",") != "-1,0,1" {
		t.Errorf("roots = %v, want [-1 0 1]", got)
	}
	if !strings.Contains(sol.Method, "bisection") {
		t.Errorf("method = %q", sol.Method)
	}
}

func TestSolveIrrationalCubicRootIsBracketedByBisection(t *testing.T) {
	sol := solutionOf(t, "solve(x^3 - 2 = 0, x)")
	if len(sol.Roots) != 1 || sol.Roots[0].IsExact() {
		t.Fatalf("roots = %v, want one proven ball", sol.Roots)
	}
	ball := sol.Roots[0].Ball(120)
	if d := ball.GuaranteedDigits(); d < 15 {
		t.Errorf("guaranteed digits = %d, want at least 15", d)
	}
	cube, _ := ball.Pow(3, 120)
	if !cube.Sub(num.BallFromExact(num.FromInt(2), 120), 120).ContainsZero() {
		t.Errorf("the ball %s does not cube to a range containing 2", ball)
	}
}

func TestSolveTranscendentalRootIsProven(t *testing.T) {
	sol := solutionOf(t, "solve(cos(x) = x, x)")
	if len(sol.Roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(sol.Roots))
	}
	ball := sol.Roots[0].Ball(120)
	if d := ball.GuaranteedDigits(); d < 15 {
		t.Errorf("guaranteed digits = %d, want at least 15", d)
	}
	if !strings.HasPrefix(sol.Roots[0].String(), "0.739085133215") {
		t.Errorf("root = %s, want 0.739085133215...", sol.Roots[0])
	}
}

func TestSolveReportsEveryRootOfAnOscillation(t *testing.T) {
	sol := solutionOf(t, "solve(sin(x) = 0, x, 1, 7)")
	if len(sol.Roots) != 2 {
		t.Fatalf("roots = %d, want pi and 2 pi", len(sol.Roots))
	}
	if sol.Searched != "[1, 7]" {
		t.Errorf("searched = %q", sol.Searched)
	}
}

func TestSolveSaysSoWhenNothingIsFound(t *testing.T) {
	for _, src := range []string{
		"solve(x^4 + 1 = 0, x)",
		"solve(exp(x) = 0, x)",
	} {
		_, err := evalSrc(NewEnv(), src)
		if err == nil || !strings.Contains(err.Error(), "no root found") {
			t.Errorf("%s error = %v, want an honest no-root-found message", src, err)
		}
	}
}

func TestSolveRejectsASignChangeAtAPole(t *testing.T) {
	_, err := evalSrc(NewEnv(), "solve(1/(x - 1/20) = 0, x)")
	if err == nil || !strings.Contains(err.Error(), "discontinuity") {
		t.Errorf("error = %v, want the pole reported as a discontinuity, not a root", err)
	}
}

func TestSolveUsesKnownConstantsAndRefusesUnknownNames(t *testing.T) {
	env := NewEnv()
	mustEval(t, env, "k = 5")
	v := mustEval(t, env, "solve(x^2 = k^2, x)")
	if got := exactRoots(t, v.Solved); strings.Join(got, ",") != "-5,5" {
		t.Errorf("roots = %v", got)
	}
	if _, err := evalSrc(NewEnv(), "solve(x + nothing = 0, x)"); err == nil || !strings.Contains(err.Error(), "unknown name") {
		t.Errorf("error = %v, want unknown name", err)
	}
}

func TestAssignmentAndFunctionDefinitionAreUnchanged(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "x = 3"); got != "3" {
		t.Errorf("x = 3 printed %s", got)
	}
	if v, ok := env.Get("x"); !ok || v.Num.String() != "3" {
		t.Error("x = 3 did not bind x")
	}
	v := mustEval(t, env, "f(x) = x^2")
	if !v.IsDefinition() || v.Def != "f(x)" {
		t.Errorf("f(x) = x^2 gave %+v", v)
	}
	if got := run(t, env, "f(4)"); got != "16" {
		t.Errorf("f(4) = %s", got)
	}
	if got := run(t, env, "solve(f(x) = 9, x)"); got != "-3" {
		t.Errorf("solve over a defined function started at %s, want -3", got)
	}
}

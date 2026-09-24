package eval

import (
	"math"
	"testing"

	"github.com/rahuldr07/go-calc/internal/unc"
)

const step = 1e-4

func slope(f func(float64) float64, x float64) float64 {
	return (f(x+step) - f(x-step)) / (2 * step)
}

func curvature(f func(float64) float64, x float64) float64 {
	return (f(x+step) - 2*f(x) + f(x-step)) / (step * step)
}

func agrees(got, want float64) bool {
	scale := math.Max(1, math.Abs(want))
	return math.Abs(got-want) <= 1e-4*scale
}

func TestEveryElementaryDerivativeMatchesTheFunctionItself(t *testing.T) {
	cases := []struct {
		name string
		at   float64
		f    func(float64) float64
	}{
		{"exp", 0.7, math.Exp},
		{"ln", 2.3, math.Log},
		{"log", 2.3, math.Log},
		{"log10", 2.3, math.Log10},
		{"log2", 2.3, math.Log2},
		{"sinh", 0.6, math.Sinh},
		{"cosh", 0.6, math.Cosh},
		{"tanh", 0.6, math.Tanh},
		{"sin", 0.7, math.Sin},
		{"cos", 0.7, math.Cos},
		{"tan", 0.5, math.Tan},
		{"asin", 0.4, math.Asin},
		{"acos", 0.4, math.Acos},
		{"atan", 0.7, math.Atan},
		{"sqrt", 2.3, math.Sqrt},
	}
	if len(cases) != len(elemPartials) {
		t.Fatalf("%d cases for %d declared derivatives", len(cases), len(elemPartials))
	}
	for _, c := range cases {
		f, ok := elemPartials[c.name]
		if !ok {
			t.Errorf("%s has no declared derivative", c.name)
			continue
		}
		d, err := f(c.at, c.f(c.at))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if want := slope(c.f, c.at); !agrees(d.d1, want) {
			t.Errorf("%s'(%g) = %g, want about %g", c.name, c.at, d.d1, want)
		}
		if want := curvature(c.f, c.at); !agrees(d.d2, want) {
			t.Errorf("%s''(%g) = %g, want about %g", c.name, c.at, d.d2, want)
		}
	}
}

func TestElementaryDerivativesRefuseTheirBoundaries(t *testing.T) {
	for _, c := range []struct{ name, why string }{
		{"ln", "zero"},
		{"log", "zero"},
		{"log10", "zero"},
		{"log2", "zero"},
		{"sqrt", "zero"},
	} {
		at := 0.0
		y := 0.0
		if c.name == "sqrt" {
			at, y = 0, 0
		}
		if _, err := elemPartials[c.name](at, y); err == nil {
			t.Errorf("%s at %s produced a derivative", c.name, c.why)
		}
	}
	for _, name := range []string{"asin", "acos"} {
		if _, err := elemPartials[name](1.5, 0); err == nil {
			t.Errorf("%s outside its domain produced a derivative", name)
		}
	}
}

func unitJets() (unc.Jet, unc.Jet) {
	return unc.Jet{Grad: unc.Sens{1: 1}}, unc.Jet{Grad: unc.Sens{2: 1}}
}

func checkTwoArgument(t *testing.T, label string, j unc.Jet, f func(float64, float64) float64, a, b float64) {
	t.Helper()
	first := func(g func(float64) float64, x float64) float64 { return slope(g, x) }
	second := func(g func(float64) float64, x float64) float64 { return curvature(g, x) }
	inA := func(x float64) float64 { return f(x, b) }
	inB := func(y float64) float64 { return f(a, y) }
	mixed := (f(a+step, b+step) - f(a+step, b-step) - f(a-step, b+step) + f(a-step, b-step)) / (4 * step * step)
	for _, c := range []struct {
		what string
		got  float64
		want float64
	}{
		{"d/da", j.Grad[1], first(inA, a)},
		{"d/db", j.Grad[2], first(inB, b)},
		{"d2/da2", j.Hess[unc.Pair{A: 1, B: 1}], second(inA, a)},
		{"d2/db2", j.Hess[unc.Pair{A: 2, B: 2}], second(inB, b)},
		{"d2/dadb", j.Hess[unc.Pair{A: 1, B: 2}], mixed},
	} {
		if !agrees(c.got, c.want) {
			t.Errorf("%s %s = %g, want about %g", label, c.what, c.got, c.want)
		}
	}
}

func TestPowerDerivativesMatchThePowerItself(t *testing.T) {
	a, b := unitJets()
	j, err := powerJet(a, b, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	checkTwoArgument(t, "pow", j, math.Pow, 2, 3)
}

func TestIntegerPowerDerivativesMatchThePowerItself(t *testing.T) {
	a, _ := unitJets()
	j, err := powerJet(a, unc.Jet{}, 2.5, 4)
	if err != nil {
		t.Fatal(err)
	}
	base := func(x float64) float64 { return math.Pow(x, 4) }
	if want := slope(base, 2.5); !agrees(j.Grad[1], want) {
		t.Errorf("d/dbase = %g, want about %g", j.Grad[1], want)
	}
	if want := curvature(base, 2.5); !agrees(j.Hess[unc.Pair{A: 1, B: 1}], want) {
		t.Errorf("d2/dbase2 = %g, want about %g", j.Hess[unc.Pair{A: 1, B: 1}], want)
	}
}

func TestAtan2DerivativesMatchAtan2Itself(t *testing.T) {
	a, b := unitJets()
	j, err := atan2Jet(a, b, 0.7, 1.3)
	if err != nil {
		t.Fatal(err)
	}
	checkTwoArgument(t, "atan2", j, math.Atan2, 0.7, 1.3)
}

func TestPercentDerivativesUseBothOperands(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "(100 ± 2) + (10 ± 1)%")
	if got := siFloat(v, 120); math.Abs(got-110) > 1e-9 {
		t.Errorf("value = %g, want 110", got)
	}
	if u := env.Uncertainty(v); math.Abs(u-math.Sqrt(5.84)) > 1e-9 {
		t.Errorf("u = %g, want sqrt(5.84)", u)
	}
	w := mustEval(t, env, "(100 ± 2) - (10 ± 1)%")
	if got := siFloat(w, 120); math.Abs(got-90) > 1e-9 {
		t.Errorf("value = %g, want 90", got)
	}
	if u := env.Uncertainty(w); math.Abs(u-math.Sqrt(4.24)) > 1e-9 {
		t.Errorf("u = %g, want sqrt(4.24)", u)
	}
}

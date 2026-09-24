package unc

import (
	"math"
	"testing"
)

func near(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestCombinedIsRootSumOfSquares(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 0, 1, 1)
	b := tab.Declare("b", 0, 2, 1)
	s := Sens{a: 3, b: 2}
	if u := tab.Combined(s); !near(u, 5, 1e-12) {
		t.Errorf("Combined = %g, want 5", u)
	}
}

func TestContributionsSumToOneAndSortDescending(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 0, 1, 1)
	b := tab.Declare("b", 0, 2, 1)
	cs := tab.Contributions(Sens{a: 3, b: 2})
	if len(cs) != 2 {
		t.Fatalf("got %d contributions, want 2", len(cs))
	}
	if cs[0].Name != "b" || !near(cs[0].Share, 16.0/25.0, 1e-12) {
		t.Errorf("first = %+v, want b at 0.64", cs[0])
	}
	if cs[1].Name != "a" || !near(cs[1].Share, 9.0/25.0, 1e-12) {
		t.Errorf("second = %+v, want a at 0.36", cs[1])
	}
	total := 0.0
	for _, c := range cs {
		total += c.Share
	}
	if !near(total, 1, 1e-9) {
		t.Errorf("shares sum to %g, want 1", total)
	}
}

func TestCorrelationChangesTheSum(t *testing.T) {
	cases := []struct {
		rho  float64
		want float64
	}{
		{1, 2},
		{0, math.Sqrt2},
		{-1, 0},
		{0.5, math.Sqrt(3)},
	}
	for _, c := range cases {
		tab := NewTable()
		a := tab.Declare("a", 10, 1, 1)
		b := tab.Declare("b", 10, 1, 2)
		if err := tab.SetCorr(a, b, c.rho); err != nil {
			t.Fatalf("SetCorr(%g): %v", c.rho, err)
		}
		if u := tab.Combined(Sens{a: 1, b: 1}); !near(u, c.want, 1e-12) {
			t.Errorf("rho %g: u = %g, want %g", c.rho, u, c.want)
		}
	}
}

func TestSetCorrRefusesNonsense(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 0, 1, 1)
	b := tab.Declare("b", 0, 1, 1)
	if err := tab.SetCorr(a, b, 1.5); err == nil {
		t.Error("rho above 1 was accepted")
	}
	if err := tab.SetCorr(a, a, 0.5); err == nil {
		t.Error("a correlation with itself was accepted")
	}
	if err := tab.SetCorr(a, InputID(99), 0.5); err == nil {
		t.Error("an unknown input was accepted")
	}
	if err := tab.SetCorr(a, b, math.NaN()); err == nil {
		t.Error("NaN was accepted")
	}
}

func TestCheckRejectsAnImpossibleCorrelationMatrix(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 0, 1, 1)
	b := tab.Declare("b", 0, 1, 1)
	c := tab.Declare("c", 0, 1, 1)
	for _, pair := range [][2]InputID{{a, b}, {a, c}, {b, c}} {
		if err := tab.SetCorr(pair[0], pair[1], -1); err != nil {
			t.Fatal(err)
		}
	}
	s := Sens{a: 1, b: 1, c: 1}
	if !near(tab.Variance(s), -3, 1e-12) {
		t.Errorf("variance = %g, want -3", tab.Variance(s))
	}
	if err := tab.Check(s); err == nil {
		t.Error("an impossible correlation matrix was accepted")
	}
}

func TestCheckAcceptsWhatIsPossible(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 0, 1, 1)
	b := tab.Declare("b", 0, 1, 1)
	if err := tab.SetCorr(a, b, -1); err != nil {
		t.Fatal(err)
	}
	if err := tab.Check(Sens{a: 1, b: 1}); err != nil {
		t.Errorf("Check = %v, want nil", err)
	}
}

func TestLinDropsExactCancellation(t *testing.T) {
	s := Sens{1: 3, 2: 5}
	if got := Lin(s, 1, s, -1); got != nil {
		t.Errorf("Lin(s, 1, s, -1) = %v, want nil", got)
	}
	if got := Scale(s, 2); got[1] != 6 || got[2] != 10 {
		t.Errorf("Scale = %v, want 6 and 10", got)
	}
	if got := Scale(s, 0); got != nil {
		t.Errorf("Scale(s, 0) = %v, want nil", got)
	}
}

func TestComposeCarriesTheProductRule(t *testing.T) {
	x := Jet{Grad: Sens{1: 1}}
	y := Jet{Grad: Sens{2: 1}}
	z := Compose(x, 4, 0, y, 3, 0, 1)
	if z.Grad[1] != 4 || z.Grad[2] != 3 {
		t.Errorf("grad = %v, want 4 and 3", z.Grad)
	}
	if got := z.Hess[Pair{A: 1, B: 2}]; got != 1 {
		t.Errorf("cross term = %g, want 1", got)
	}
	if _, ok := z.Hess[Pair{A: 1, B: 1}]; ok {
		t.Error("a product has no pure second derivative")
	}
}

func TestComposeSquaresTheDiagonal(t *testing.T) {
	x := Jet{Grad: Sens{1: 1}}
	z := Unary(x, 2, 2)
	if got := z.Hess[Pair{A: 1, B: 1}]; got != 2 {
		t.Errorf("second derivative = %g, want 2", got)
	}
}

func TestSecondOrderAndNonlinearity(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 1, 1, 1)
	s := Sens{a: 2}
	h := Curv{Pair{A: a, B: a}: 2}
	if u := tab.Combined(s); !near(u, 2, 1e-12) {
		t.Errorf("first order = %g, want 2", u)
	}
	if u := tab.SecondOrder(s, h); !near(u, math.Sqrt(6), 1e-12) {
		t.Errorf("second order = %g, want sqrt(6)", u)
	}
	if !tab.Nonlinear(s, h) {
		t.Error("a 22 percent gap was not flagged")
	}
	if tab.Nonlinear(s, nil) {
		t.Error("a linear case was flagged")
	}
}

func TestCoverageFactors(t *testing.T) {
	cases := []struct {
		p, k float64
	}{
		{0.95, 1.960}, {0.99, 2.576}, {0.90, 1.645}, {0.6827, 1.000},
	}
	for _, c := range cases {
		k, ok := FactorFor(c.p)
		if !ok {
			t.Fatalf("FactorFor(%g) refused", c.p)
		}
		if !near(k, c.k, 5e-4) {
			t.Errorf("k(%g) = %g, want %g", c.p, k, c.k)
		}
	}
	if _, ok := FactorFor(1); ok {
		t.Error("a probability of 1 was accepted")
	}
	if _, ok := FactorFor(0); ok {
		t.Error("a probability of 0 was accepted")
	}
}

func TestElementaryRecognisesADeclaredInput(t *testing.T) {
	if _, ok := Elementary(Sens{3: 1}); !ok {
		t.Error("a single unit sensitivity was not elementary")
	}
	if _, ok := Elementary(Sens{3: 2}); ok {
		t.Error("a scaled sensitivity was called elementary")
	}
	if _, ok := Elementary(Sens{3: 1, 4: 1}); ok {
		t.Error("a two input sensitivity was called elementary")
	}
}

func TestSecondOrderCountsOffDiagonalTermsTwice(t *testing.T) {
	tab := NewTable()
	x := tab.Declare("x", 1, 1, 1)
	y := tab.Declare("y", 1, 1, 1)
	s := Sens{x: 1, y: 1}
	h := Curv{Pair{A: x, B: y}: 1}
	if u := tab.Combined(s); !near(u, math.Sqrt2, 1e-12) {
		t.Errorf("first order = %g, want sqrt(2)", u)
	}
	if u := tab.SecondOrder(s, h); !near(u, math.Sqrt(3), 1e-12) {
		t.Errorf("second order = %g, want sqrt(3)", u)
	}
}

func TestComposeCarriesTheInnerCurvature(t *testing.T) {
	inner := Unary(Jet{Grad: Sens{1: 1}}, 2, 2)
	outer := Unary(inner, 3, 0)
	if got := outer.Grad[1]; got != 6 {
		t.Errorf("grad = %g, want 6", got)
	}
	if got := outer.Hess[Pair{A: 1, B: 1}]; got != 6 {
		t.Errorf("hess = %g, want 6: a linear outer function still carries the inner curvature", got)
	}
}

func TestNonlinearityUncheckedIsDistinctFromLinear(t *testing.T) {
	tab := NewTable()
	a := tab.Declare("a", 1, 1, 1)
	b := tab.Declare("b", 1, 1, 1)
	if err := tab.SetCorr(a, b, 0.5); err != nil {
		t.Fatal(err)
	}
	s := Sens{a: 1, b: 1}
	h := Curv{Pair{A: a, B: b}: 1}
	if !tab.NonlinearityUnchecked(s, h) {
		t.Error("a curved expression over correlated inputs was not reported as unchecked")
	}
	if tab.Nonlinear(s, h) {
		t.Error("the check ran on correlated inputs")
	}
	if tab.NonlinearityUnchecked(s, nil) {
		t.Error("a linear expression was reported as unchecked")
	}
}

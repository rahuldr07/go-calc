package eval

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func mustEval(t *testing.T, env *Env, src string) value.Value {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	return v
}

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

func TestProductBudgetMatchesHandCalculation(t *testing.T) {
	env := NewEnv()
	run(t, env, "l = 10 ± 0.1")
	run(t, env, "w = 5 ± 0.2")
	v := mustEval(t, env, "l * w")
	if u := env.Uncertainty(v); math.Abs(u-math.Sqrt(4.25)) > 1e-9 {
		t.Errorf("u(l*w) = %g, want sqrt(4.25)", u)
	}
	cs := env.Contributions(v)
	if len(cs) != 2 {
		t.Fatalf("got %d contributions, want 2", len(cs))
	}
	if cs[0].Name != "w" || math.Abs(cs[0].Share-4.0/4.25) > 1e-9 {
		t.Errorf("first = %+v, want w at 0.941", cs[0])
	}
	if cs[1].Name != "l" || math.Abs(cs[1].Share-0.25/4.25) > 1e-9 {
		t.Errorf("second = %+v, want l at 0.059", cs[1])
	}
	if total := cs[0].Share + cs[1].Share; math.Abs(total-1) > 1e-9 {
		t.Errorf("shares sum to %g, want 1", total)
	}
}

func TestPartialCancellationIsNotZero(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 10 ± 1")
	run(t, env, "y = 10 ± 1")
	v := mustEval(t, env, "x - y")
	if u := env.Uncertainty(v); math.Abs(u-math.Sqrt2) > 1e-12 {
		t.Errorf("u(x - y) = %g, want sqrt(2)", u)
	}
	w := mustEval(t, env, "2x - x")
	if u := env.Uncertainty(w); math.Abs(u-1) > 1e-12 {
		t.Errorf("u(2x - x) = %g, want 1", u)
	}
}

func TestUncertaintyFollowsTheUnit(t *testing.T) {
	env := NewEnv()
	run(t, env, "d = 12.3 ± 0.2 km")
	v := mustEval(t, env, "d")
	if u := env.Uncertainty(v); math.Abs(u-200) > 1e-9 {
		t.Errorf("u(d) in SI = %g, want 200", u)
	}
	if got := format.Text(v, nil); got != "12.30 ± 0.20 km" {
		t.Errorf("d = %q, want 12.30 ± 0.20 km", got)
	}
	w := mustEval(t, env, "d to m")
	if got := format.Text(w, nil); got != "12300 ± 200 m" {
		t.Errorf("d to m = %q, want 12300 ± 200 m", got)
	}
	x := mustEval(t, env, "(12.3 ± 0.2) km")
	if u := env.Uncertainty(x); math.Abs(u-200) > 1e-9 {
		t.Errorf("u((12.3 ± 0.2) km) in SI = %g, want 200", u)
	}
	if got := format.Text(x, nil); got != "12.30 ± 0.20 km" {
		t.Errorf("(12.3 ± 0.2) km = %q, want 12.30 ± 0.20 km", got)
	}
}

func TestConciseNotation(t *testing.T) {
	env := NewEnv()
	for _, c := range []struct {
		src   string
		sigma float64
	}{
		{"12.3(2)", 0.2},
		{"1.2345(67)", 0.0067},
		{"1.234(5)e3", 5},
	} {
		v := mustEval(t, env, c.src)
		if u := env.Uncertainty(v); math.Abs(u-c.sigma) > c.sigma*1e-9 {
			t.Errorf("u(%s) = %g, want %g", c.src, u, c.sigma)
		}
	}
	if got := run(t, env, "2(3)"); got != "6" {
		t.Errorf("2(3) = %s, want 6 (implicit multiplication)", got)
	}
}

func TestNonlinearityIsFlagged(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "exp(0 ± 1)")
	if v.Unc.Warning == "" {
		t.Error("a 22 percent second-order gap was not flagged")
	}
	w := mustEval(t, env, "sqrt(2 * (12.3 ± 0.2 m) / g0)")
	if w.Unc.Warning != "" {
		t.Errorf("a well behaved case was flagged: %s", w.Unc.Warning)
	}
}

func TestExpandedUncertainty(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	run(t, env, "t = sqrt(2h / g0)")
	if got := format.Text(mustEval(t, env, "t"), nil); got != "1.584 ± 0.013 s" {
		t.Errorf("t = %q, want 1.584 ± 0.013 s", got)
	}
	if got := format.Text(mustEval(t, env, "expand(t, 2)"), nil); got != "1.584 ± 0.026 s (k = 2)" {
		t.Errorf("expand(t, 2) = %q", got)
	}
	if got := format.Text(mustEval(t, env, "expand(t, 95%)"), nil); got != "1.584 ± 0.025 s (95%, k = 1.96)" {
		t.Errorf("expand(t, 95%%) = %q", got)
	}
}

func TestRoundingFollowsTheUncertainty(t *testing.T) {
	env := NewEnv()
	cases := []struct {
		src  string
		want string
	}{
		{"1234.5678 ± 13", "1235 ± 13"},
		{"1234.5678 ± 130", "1230 ± 130"},
		{"0.00123456 ± 0.0000021", "0.0012346 ± 0.0000021"},
		{"9.96 ± 0.996", "10.0 ± 1.0"},
	}
	for _, c := range cases {
		if got := format.Text(mustEval(t, env, c.src), nil); got != c.want {
			t.Errorf("%s = %q, want %q", c.src, got, c.want)
		}
	}
}

func TestWhatCannotBePropagatedIsRefused(t *testing.T) {
	for _, src := range []string{
		"floor(1.5 ± 0.1)",
		"round(1.5 ± 0.1)",
		"(3 ± 0.1)!",
		"(7 ± 1) mod 3",
		"gcd(8 ± 1, 12)",
		"abs(0 ± 1)",
		"(1 ± 0.1) ± 0.2",
	} {
		env := NewEnv()
		n, _, err := parse.Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if _, err := env.Eval(n); err == nil {
			t.Errorf("%s was answered instead of refused", src)
		}
	}
}

func TestCorrelationNeedsDeclaredMeasurements(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 10 ± 1")
	run(t, env, "c = a + b")
	n, _, err := parse.Parse("corr(c, a) = 0.5")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(n); err == nil {
		t.Error("a derived value was accepted as a declared measurement")
	}
}

func TestInconsistentCorrelationIsRefused(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 1 ± 1")
	run(t, env, "b = 1 ± 1")
	run(t, env, "c = 1 ± 1")
	run(t, env, "corr(a, b) = -1")
	run(t, env, "corr(a, c) = -1")
	run(t, env, "corr(b, c) = -1")
	n, _, err := parse.Parse("a + b + c")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(n); err == nil {
		t.Error("an impossible correlation matrix produced a number")
	}
}

func TestBallsAndUncertaintyStaySeparate(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "12.3 ± 0.2")
	if !v.Num.IsExact() {
		t.Error("a ± value grew a ball radius")
	}
	w := mustEval(t, env, "sqrt(2)")
	if w.IsMeasured() || w.Unc.Sigma != 0 {
		t.Error("a ball radius became a ±")
	}
}

func TestFunctionsCarrySensitivities(t *testing.T) {
	env := NewEnv()
	run(t, env, "f(x) = x ^ 2")
	run(t, env, "a = 3 ± 0.1")
	v := mustEval(t, env, "f(a)")
	if u := env.Uncertainty(v); math.Abs(u-0.6) > 1e-9 {
		t.Errorf("u(f(a)) = %g, want 0.6", u)
	}
}

func TestDegreesScaleTheSensitivity(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Degrees = true
	n, _, err := parse.Parse("sin(30 ± 1)")
	if err != nil {
		t.Fatal(err)
	}
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := math.Cos(math.Pi/6) * math.Pi / 180
	if u := env.Uncertainty(v); math.Abs(u-want) > 1e-12 {
		t.Errorf("u(sin(30 deg ± 1 deg)) = %g, want %g", u, want)
	}
}

func noteMsgs(env *Env) []string {
	out := make([]string, 0, len(env.Notes()))
	for _, n := range env.Notes() {
		out = append(out, n.Msg)
	}
	return out
}

func TestAmbiguousNameSaysWhichReadingItTook(t *testing.T) {
	cases := []struct {
		define string
		use    string
		want   string
	}{
		{"h = 3", "5 h", "read as the unit h (time), not the variable h; the variable is a plain number"},
		{"h = 3 m", "2 h", "read as the variable h, not the unit h (time); the variable already carries a unit"},
		{"h = 12.3 ± 0.2 m", "2 h", "read as the variable h, not the unit h (time); the variable carries a measured uncertainty"},
	}
	for _, c := range cases {
		env := NewEnv()
		run(t, env, c.define)
		mustEval(t, env, c.use)
		got := noteMsgs(env)
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%s then %s: notes = %q, want [%q]", c.define, c.use, got, c.want)
		}
	}
}

func TestUnambiguousNameSaysNothing(t *testing.T) {
	for _, c := range [][2]string{
		{"x = 3", "5 m"},
		{"x = 3", "5 x"},
		{"h = 3", "h + 1"},
	} {
		env := NewEnv()
		run(t, env, c[0])
		mustEval(t, env, c[1])
		if got := noteMsgs(env); len(got) != 0 {
			t.Errorf("%s then %s: notes = %q, want none", c[0], c[1], got)
		}
	}
}

func TestTheNoteReachesTheOutput(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	v := mustEval(t, env, "2 h")
	got := format.Text(v, env.Notes())
	want := "24.60 ± 0.40 m\n  read as the variable h, not the unit h (time); the variable carries a measured uncertainty"
	if got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

func TestNonlinearityIsNotCheckedSilentlyWhenCorrelated(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 1 ± 1")
	run(t, env, "b = 1 ± 1")
	run(t, env, "corr(a, b) = 0.5")
	v := mustEval(t, env, "a * b")
	if v.Unc.Warning != unc.UncheckedWarning {
		t.Errorf("warning = %q, want %q", v.Unc.Warning, unc.UncheckedWarning)
	}
	if !strings.Contains(format.Text(v, nil), unc.UncheckedWarning) {
		t.Errorf("the skipped check did not reach the output: %q", format.Text(v, nil))
	}
	w := mustEval(t, env, "a + b")
	if w.Unc.Warning != "" {
		t.Errorf("a linear expression was annotated: %q", w.Unc.Warning)
	}
}

func TestAStationaryPointIsNotReportedAsCertain(t *testing.T) {
	for _, src := range []string{
		"cos(x)", "x * x", "x ^ 2",
		"cos(x) + 1", "cos(x) * 2", "2 - cos(x)", "(x*x) + (x*x)", "(x*x) / 3",
		"exp(x * x)", "sqrt(4 + x*x)", "atan2(x * x, 1)", "abs(cos(x))",
		"cos(x) ^ 3", "2 ^ (x * x)", "root(4 + x*x, 2)",
		"(x*x) m", "-(x * x)", "(x*x)%",
	} {
		env := NewEnv()
		run(t, env, "x = 0 ± 1")
		v := mustEval(t, env, src)
		if len(v.Curv) == 0 {
			t.Errorf("%s: the curvature was discarded at the stationary point", src)
		}
		if env.Uncertainty(v) != 0 {
			t.Errorf("%s: first-order uncertainty should cancel to zero", src)
		}
		if v.Unc.Warning != unc.NonlinearWarning {
			t.Errorf("%s: warning = %q, want the first-order warning", src, v.Unc.Warning)
		}
		if !strings.Contains(format.Text(v, nil), unc.NonlinearWarning) {
			t.Errorf("%s: the warning did not reach the output: %q", src, format.Text(v, nil))
		}
		if !v.Unc.Degenerate {
			t.Errorf("%s: the first-order estimate was not marked degenerate", src)
		}
	}
	env := NewEnv()
	run(t, env, "x = 10 ± 1")
	v := mustEval(t, env, "x - x")
	if len(v.Curv) != 0 {
		t.Errorf("x - x grew a curvature: %v", v.Curv)
	}
	if v.Unc.Warning != "" {
		t.Errorf("x - x was annotated: %q", v.Unc.Warning)
	}
	if got := format.Text(v, nil); got != "0" {
		t.Errorf("x - x = %q, want 0", got)
	}
	if v.Unc.Degenerate {
		t.Error("x - x was marked degenerate: it is genuinely exact")
	}
}

func TestSelfQuotientIsExactlyOne(t *testing.T) {
	for _, mid := range []string{"0.1", "3.7", "12.3", "7", "1e-3", "6.02e23"} {
		env := NewEnv()
		run(t, env, "x = "+mid+" ± 1")
		v := mustEval(t, env, "x / x")
		if u := env.Uncertainty(v); u != 0 {
			t.Errorf("u(x/x) at %s = %g, want exactly 0", mid, u)
		}
		if len(v.Curv) != 0 {
			t.Errorf("x/x at %s kept a curvature: %v", mid, v.Curv)
		}
		if got := format.Text(v, nil); got != "1" {
			t.Errorf("x/x at %s = %q, want a bare 1", mid, got)
		}
	}
}

func TestAnOverflowingSensitivityIsNotBlamedOnCorrelation(t *testing.T) {
	env := NewEnv()
	n, _, err := parse.Parse("1e400 * (2 ± 1)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Eval(n)
	if err == nil {
		t.Fatal("an overflowing sensitivity produced a number")
	}
	if !errors.Is(err, unc.ErrSensitivityOverflow) {
		t.Errorf("error = %v, want the overflow error", err)
	}
	if strings.Contains(err.Error(), "correlation") {
		t.Errorf("error blames correlation the user never declared: %v", err)
	}
}

func TestACorrelatedBudgetPartitions(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 10 ± 2")
	run(t, env, "corr(a, b) = -1")
	v := mustEval(t, env, "a + b")
	if u := env.Uncertainty(v); math.Abs(u-1) > 1e-12 {
		t.Fatalf("u = %g, want 1", u)
	}
	cs := v.Unc.Budget
	if len(cs) != 3 {
		t.Fatalf("budget = %+v, want two inputs and a covariance row", cs)
	}
	last := cs[2]
	if !last.Covariance || last.Name != unc.CovarianceName {
		t.Errorf("last row = %+v, want the covariance row", last)
	}
	if math.Abs(last.Share+4) > 1e-9 {
		t.Errorf("covariance share = %g, want -4", last.Share)
	}
	total := 0.0
	for _, c := range cs {
		total += c.Share
	}
	if math.Abs(total-1) > 1e-9 {
		t.Errorf("shares sum to %g, want 1", total)
	}
	if !strings.Contains(format.Text(v, nil), "covariance term from declared correlation") {
		t.Errorf("text does not mention the covariance term: %q", format.Text(v, nil))
	}
	out, err := format.JSON("a + b", v, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"covariance": true`) {
		t.Errorf("JSON does not flag the covariance row: %s", out)
	}
}

func TestAnUncorrelatedBudgetHasNoCovarianceRow(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 10 ± 2")
	v := mustEval(t, env, "a + b")
	for _, c := range v.Unc.Budget {
		if c.Covariance {
			t.Errorf("an uncorrelated budget grew a covariance row: %+v", c)
		}
	}
	if strings.Contains(format.Text(v, nil), "covariance") {
		t.Errorf("text mentions covariance with none declared: %q", format.Text(v, nil))
	}
}

func TestAStationaryPointWarnsInJSONToo(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	v := mustEval(t, env, "x * x")
	out, err := format.JSON("x * x", v, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), unc.NonlinearWarning) {
		t.Errorf("JSON dropped the warning: %s", out)
	}
}

func TestUnknownUncertaintyIsNotZeroUncertainty(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	for _, c := range [][2]string{
		{"cos(x)", "1 ± ?"},
		{"x * x", "0 ± ?"},
		{"x ^ 2", "0 ± ?"},
		{"(x * x) m", "0 ± ? m"},
	} {
		v := mustEval(t, env, c[0])
		got := format.Text(v, nil)
		head, _, _ := strings.Cut(got, "\n")
		if head != c[1] {
			t.Errorf("%s = %q, want %q", c[0], head, c[1])
		}
	}
	run(t, env, "y = 3 ± 0")
	certain := mustEval(t, env, "y * y")
	if certain.Unc.Degenerate {
		t.Error("an input with no uncertainty produced an unknown result")
	}
	if got := format.Text(certain, nil); got != "9" {
		t.Errorf("y * y = %q, want a bare 9", got)
	}
}

func TestJSONMarksTheDegenerateEstimateInsteadOfClaimingZero(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	out, err := format.JSON("x * x", mustEval(t, env, "x * x"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"degenerate": true`) {
		t.Errorf("JSON does not mark the estimate degenerate: %s", out)
	}
	if strings.Contains(string(out), `"standard"`) {
		t.Errorf("JSON reports a standard uncertainty for a degenerate estimate: %s", out)
	}
	run(t, env, "a = 10 ± 1")
	plain, err := format.JSON("a * 2", mustEval(t, env, "a * 2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plain), `"standard": 2`) {
		t.Errorf("an ordinary result lost its standard uncertainty: %s", plain)
	}
	if strings.Contains(string(plain), "degenerate") {
		t.Errorf("an ordinary result was marked degenerate: %s", plain)
	}
}

func TestExpandingADegenerateEstimateSaysWhy(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	run(t, env, "z = x * x")
	n, _, err := parse.Parse("expand(z, 2)")
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Eval(n)
	if err == nil {
		t.Fatal("expanding a degenerate estimate produced a number")
	}
	if !strings.Contains(err.Error(), "first-order terms cancel") {
		t.Errorf("error = %v, want it to name the cancellation", err)
	}
}

func TestComposedUnitSymbolsKeepTheirGrouping(t *testing.T) {
	env := NewEnv()
	for _, c := range [][2]string{
		{"(10 m/s) * (2 s)", "(m/s)·s"},
		{"(6 m) / (2 s)", "m/s"},
		{"(1 J) / (1 J/(mol·K))", "J/(J/(mol·K))"},
		{"(3 m) * (2 kg)", "m·kg"},
	} {
		v := mustEval(t, env, c[0])
		if v.Unit != c[1] {
			t.Errorf("%s has unit %q, want %q", c[0], v.Unit, c[1])
		}
	}
}

func TestAStationaryPointSurvivesAStoredVariable(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	run(t, env, "z = x * x")
	for _, src := range []string{"z", "z + 1", "z * 4", "sqrt(z + 4)", "z - z"} {
		v := mustEval(t, env, src)
		if src == "z - z" {
			if v.Unc.Degenerate || v.Unc.Warning != "" {
				t.Errorf("z - z = %+v, want an exact zero", v.Unc)
			}
			continue
		}
		if !v.Unc.Degenerate {
			t.Errorf("%s: a stored stationary point was reported as certain", src)
		}
		if v.Unc.Warning != unc.NonlinearWarning {
			t.Errorf("%s: warning = %q", src, v.Unc.Warning)
		}
	}
}

func TestAStationaryPointSurvivesAUserFunction(t *testing.T) {
	env := NewEnv()
	run(t, env, "sq(v) = v * v")
	run(t, env, "x = 0 ± 1")
	v := mustEval(t, env, "sq(x) + 1")
	if !v.Unc.Degenerate {
		t.Error("a stationary point through a user function was reported as certain")
	}
	if got := format.Text(v, nil); !strings.HasPrefix(got, "1 ± ?") {
		t.Errorf("sq(x) + 1 = %q, want 1 ± ?", got)
	}
}

func TestANonFiniteCurvatureIsRefusedNotDropped(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 1e-300 ± 0.5")
	n, _, err := parse.Parse("x / x")
	if err != nil {
		t.Fatal(err)
	}
	_, err = env.Eval(n)
	if err == nil {
		t.Fatal("a curvature that underflowed to NaN produced a confident answer")
	}
	if !errors.Is(err, unc.ErrSensitivityUndefined) {
		t.Errorf("error = %v, want the undefined-derivative error", err)
	}
}

func TestEveryUnitThisEngineWritesItCanReadBack(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	for _, src := range []string{
		"3 m", "3 km", "3 g", "3 degC", "3 J/(mol·K)", "3 mol",
		"(10 m/s) * (2 s)", "(6 m) / (2 s)", "(1 J) / (1 J/(mol·K))",
		"(3 m) * (2 kg)", "2 / (1 m·K)", "1 / (1 J/(mol·K))", "2 / (1 m)",
		"(3 km) / (2 h)", "(3 km) * (2 km)", "(3 g) * (2 m)",
		"sqrt(2h / g0)", "(1 m) ^ 3", "(1 N) * (1 m)", "R",
		"(1 m/s) / (1 m/s^2)", "((1 m) / (1 s)) / (1 kg)",
		"((6 m) / (2 s)) ^ 2", "((10 m/s) * (2 s)) ^ 2", "(1 m·kg) ^ 2",
		"((1 m) ^ 2) ^ 3", "(((1 m) ^ 2) ^ 3) ^ 2", "(2 / (1 m·K)) ^ 2",
		"(3 degC) ^ 2", "(3 degF) ^ 3", "(3 degC) ^ -1",
		"(3 km) ^ 2", "(3 km) ^ -2", "(2 mm) ^ 3", "(3 g) ^ 2",
		"0.5 * (2 kg) * (((100 m) / (10 s)) ^ 2)",
		"(((6 m) / (2 s)) ^ 2) * (2 s)", "(((6 m) / (2 s)) ^ 2) / (2 s)",
		"((3 degC) ^ 2) ^ 2", "((3 km) ^ 2) ^ 2", "(((1 m) ^ 2) ^ 3) * (1 m)",
		"sqrt((((6 m) / (2 s)) ^ 2))",
	} {
		v := mustEval(t, env, src)
		if v.Unit == "" {
			continue
		}
		u, ok := units.Lookup(v.Unit)
		if !ok {
			t.Errorf("%s printed unit %q, which this engine cannot read back", src, v.Unit)
			continue
		}
		if u.Dim != v.Dim {
			t.Errorf("%s printed unit %q, which reads back as %v, not %v", src, v.Unit, u.Dim, v.Dim)
		}
		if u.Factor.Cmp(v.FactorOrOne()) != 0 || u.Offset.Cmp(v.Offset) != 0 {
			t.Errorf("%s printed unit %q with scale %v offset %v, but the value is scale %v offset %v",
				src, v.Unit, u.Factor, u.Offset, v.FactorOrOne(), v.Offset)
		}
	}
}

func TestComposingScaledUnitsFallsBackToSI(t *testing.T) {
	env := NewEnv()
	for _, c := range [][2]string{
		{"(3 km) / (2 h)", "0.416666666667 m/s"},
		{"(3 km) * (2 km)", "6000000 m^2"},
		{"(3 g) * (2 m)", "0.006 m·kg"},
		{"(3 kg) * (2 m)", "6 kg·m"},
	} {
		got := format.Text(mustEval(t, env, c[0]), nil)
		head, _, _ := strings.Cut(got, "\n")
		if !strings.HasSuffix(head, c[1]) {
			t.Errorf("%s = %q, want it to end in %q", c[0], head, c[1])
		}
	}
}

func TestAPowerSpellsTheDimensionItHolds(t *testing.T) {
	env := NewEnv()
	for _, c := range [][2]string{
		{"((6 m) / (2 s)) ^ 2", "9 (m/s)^2"},
		{"((1 m) ^ 2) ^ 3", "1 (m^2)^3"},
		{"((10 m/s) * (2 s)) ^ 2", "400 ((m/s)·s)^2"},
		{"(1 m·kg) ^ 2", "1 (m·kg)^2"},
		{"(3 km) ^ 2", "9 km^2"},
		{"(3 m) ^ 0", "1"},
		{"(2 m) ^ 1", "2 m"},
		{"0.5 * (2 kg) * (((100 m) / (10 s)) ^ 2)", "100 kg·(m/s)^2"},
	} {
		got := format.Text(mustEval(t, env, c[0]), nil)
		head, _, _ := strings.Cut(got, "\n")
		if head != c[1] {
			t.Errorf("%s = %q, want %q", c[0], head, c[1])
		}
	}
}

func TestAPoweredAffineUnitAgreesWithTheProduct(t *testing.T) {
	env := NewEnv()
	for _, c := range [][2]string{
		{"(3 degC) ^ 2", "(3 degC) * (3 degC)"},
		{"(3 degF) ^ 2", "(3 degF) * (3 degF)"},
	} {
		square := mustEval(t, env, c[0])
		product := mustEval(t, env, c[1])
		if square.Unit != product.Unit || square.Dim != product.Dim {
			t.Errorf("%s has unit %q %v, but %s has unit %q %v",
				c[0], square.Unit, square.Dim, c[1], product.Unit, product.Dim)
		}
		if square.SI(64).Exact().Cmp(product.SI(64).Exact()) != 0 {
			t.Errorf("%s = %v, but %s = %v", c[0], square.Num, c[1], product.Num)
		}
		if square.Offset.Sign() != 0 {
			t.Errorf("%s kept an affine offset of %v", c[0], square.Offset)
		}
	}
}

func TestADerivativeErrorSaysWhichOrderFailed(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 1e-300 ± 0.5")
	for _, c := range [][2]string{
		{"1e400 * x", "first-order"},
		{"x / x", "second-order"},
	} {
		n, _, err := parse.Parse(c[0])
		if err != nil {
			t.Fatal(err)
		}
		_, err = env.Eval(n)
		if err == nil {
			t.Errorf("%s produced a number", c[0])
			continue
		}
		if !strings.Contains(err.Error(), c[1]) {
			t.Errorf("%s failed with %q, which does not say it was the %s term", c[0], err, c[1])
		}
	}
}

func TestAGuaranteedBoundIsNotSpelledLikeAnUncertainty(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 0 ± 1")
	for _, src := range []string{"sin(1e60) + (x*x)", "sin(1e60) + x", "sin(1e60) * (1 ± 0.1)"} {
		v := mustEval(t, env, src)
		head, _, _ := strings.Cut(format.Text(v, nil), "\n")
		if strings.Count(head, "±") > 1 {
			t.Errorf("%s = %q, which shows two ± on one line", src, head)
		}
		if !strings.Contains(head, "[") {
			t.Errorf("%s = %q, want the guaranteed bound as an interval", src, head)
		}
	}
	degenerate := mustEval(t, env, "sin(1e60) + (x*x)")
	head, _, _ := strings.Cut(format.Text(degenerate, nil), "\n")
	if !strings.HasSuffix(head, "± "+format.UnknownUncertainty) {
		t.Errorf("degenerate line = %q, want it to end in the unknown-uncertainty marker", head)
	}
	out, err := format.JSON("sin(1e60) + (x*x)", degenerate, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, `"ball"`) || !strings.Contains(text, `"degenerate": true`) {
		t.Errorf("JSON = %s, want the bound under value.ball and the marker under uncertainty.degenerate", text)
	}
	if strings.Contains(text, `"standard"`) {
		t.Errorf("JSON = %s, want no standard uncertainty beside a degenerate estimate", text)
	}
}

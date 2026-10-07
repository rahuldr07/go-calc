package eval

import (
	"math"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
)

func printed(t *testing.T, env *Env, src string) string {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	return format.Text(v, nil)
}

func TestStatisticsStayExact(t *testing.T) {
	cases := map[string]string{
		"sum(1..10)":                        "55",
		"sum(1..100)":                       "5050",
		"sum(1, 2, 3)":                      "6",
		"sum([1, 2], 3)":                    "6",
		"sum(0.1, 0.2)":                     "3/10",
		"product(1..5)":                     "120",
		"product(1/2, 2/3)":                 "1/3",
		"mean(1, 2)":                        "3/2",
		"mean(1, 2, 3)":                     "2",
		"mean(1..4)":                        "5/2",
		"mean(0.1, 0.2)":                    "3/20",
		"count(1..100)":                     "100",
		"count(1, 2, 3)":                    "3",
		"min(3, 1, 2)":                      "1",
		"max(3, 1, 2)":                      "3",
		"min(1/3, 1/4)":                     "1/4",
		"max(-5, -2)":                       "-2",
		"median(1, 2, 3)":                   "2",
		"median(3, 1, 2)":                   "2",
		"median(1, 2, 3, 4)":                "5/2",
		"median(4, 1, 3, 2)":                "5/2",
		"median(1/3)":                       "1/3",
		"median(1/3, 2/3)":                  "1/2",
		"variance(1..4)":                    "5/3",
		"variancep(1..4)":                   "5/4",
		"variance(2, 4, 4, 4, 5, 5, 7, 9)":  "32/7",
		"variancep(2, 4, 4, 4, 5, 5, 7, 9)": "4",
		"stdevp(2, 4, 4, 4, 5, 5, 7, 9)":    "2",
		"stdev(3, 3, 3)":                    "0",
		"variance(1/2, 3/2)":                "1/2",
	}
	env := NewEnv()
	for src, want := range cases {
		if got := run(t, env, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}

func TestStatisticsAcceptAListOrArguments(t *testing.T) {
	env := NewEnv()
	for _, name := range statisticalFunctions {
		if name == "stdev" || name == "variance" || name == "stdevp" || name == "variancep" {
			continue
		}
		spelled := run(t, env, name+"(4, 1, 3, 2)")
		listed := run(t, env, name+"([4, 1, 3, 2])")
		if spelled != listed {
			t.Errorf("%s: arguments give %s, a list gives %s", name, spelled, listed)
		}
	}
	if got, want := run(t, env, "variance([4, 1, 3, 2])"), run(t, env, "variance(4, 1, 3, 2)"); got != want {
		t.Errorf("variance: a list gives %s, arguments give %s", got, want)
	}
}

func TestSampleStandardDeviationIsAProvenBall(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "stdev(1, 2, 3, 4)")
	if v.Num.IsExact() {
		t.Fatalf("stdev(1,2,3,4) = %s, want a proven ball: the variance 5/3 is not a perfect square", v.Num)
	}
	text := format.Text(v, nil)
	if !strings.HasPrefix(text, "1.29099444874") {
		t.Errorf("stdev(1,2,3,4) = %q, want it to start 1.29099444874", text)
	}
	root := mustEval(t, env, "stdev(1, 3)")
	if root.Num.IsExact() {
		t.Errorf("stdev(1,3) = %s, want sqrt(2) as a proven ball", root.Num)
	}
	if got := run(t, env, "stdev(0, 2, 4)"); got != "2" {
		t.Errorf("stdev(0,2,4) = %s, want exactly 2: its variance 4 is a perfect square", got)
	}
}

func TestSpreadStatesWhichConventionItUses(t *testing.T) {
	cases := map[string]string{
		"stdev(1, 2, 3)":     "sample standard deviation",
		"variance(1, 2, 3)":  "sample variance",
		"stdevp(1, 2, 3)":    "population standard deviation",
		"variancep(1, 2, 3)": "population variance",
	}
	for src, want := range cases {
		env := NewEnv()
		n, _, err := parse.Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("Eval(%q): %v", src, err)
		}
		if got := format.Text(v, env.Notes()); !strings.Contains(got, "read as the "+want) {
			t.Errorf("%s printed %q, want it to say it is the %s", src, got, want)
		}
	}
}

func TestOneValueHasNoSampleSpread(t *testing.T) {
	for _, src := range []string{"stdev(5)", "variance(5)", "stdev([5])"} {
		if got := evalError(t, NewEnv(), src); !strings.Contains(got, "needs at least two values") {
			t.Errorf("%s: got %q", src, got)
		}
	}
	if got := run(t, NewEnv(), "variancep(5)"); got != "0" {
		t.Errorf("variancep(5) = %s, want 0", got)
	}
}

func TestMedianOfAnEvenListIsTheExactMeanOfTheMiddlePair(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "median(1, 2, 4, 8)"); got != "3" {
		t.Errorf("median(1,2,4,8) = %s, want 3", got)
	}
	if got := run(t, env, "median(1, 2, 3, 4)"); got != "5/2" {
		t.Errorf("median(1,2,3,4) = %s, want 5/2", got)
	}
	if got := run(t, env, "median(1/3, 1/2, 1, 2)"); got != "3/4" {
		t.Errorf("median(1/3,1/2,1,2) = %s, want 3/4", got)
	}
	if got := run(t, env, "median(9, 1, 5)"); got != "5" {
		t.Errorf("median(9,1,5) = %s, want 5", got)
	}
}

func TestStatisticsKeepUnits(t *testing.T) {
	env := NewEnv()
	cases := map[string]string{
		"sum(1 m, 2 m)":            "3 m",
		"sum(1 km, 500 m)":         "3/2 km",
		"sum(500 m, 1 km)":         "1500 m",
		"mean(1 km, 500 m)":        "3/4 km",
		"mean(2 m, 3 m)":           "5/2 m",
		"median(1 m, 2 m, 9 m)":    "2 m",
		"median(1 m, 2 m)":         "3/2 m",
		"min(1 km, 500 m)":         "500 m",
		"max(1 km, 500 m)":         "1 km",
		"product(2 m, 3 m)":        "6 m^2",
		"variancep(1 m, 3 m)":      "1 m^2",
		"stdevp(1 m, 3 m)":         "1 m",
		"stdevp(1 km, 3 km)":       "1 km",
		"variancep(1 km, 3 km)":    "1 km^2",
		"count(1 m, 2 m)":          "2",
		"sum(1 m, 2 m, 3 m) / 2 m": "3",
	}
	for src, want := range cases {
		if got := run(t, env, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}

func TestMixedDimensionsAreRefused(t *testing.T) {
	for _, name := range statisticalFunctions {
		src := name + "(1 m, 2 s)"
		got := evalError(t, NewEnv(), src)
		if !strings.Contains(got, "cannot mix length (m) with time (s)") {
			t.Errorf("%s: got %q", src, got)
		}
	}
	for _, src := range []string{"[1 m, 2 s]", "sum([1 m, 2 s])", "sum([1 m], 2 s)", "mean(1, 2 m)"} {
		if got := evalError(t, NewEnv(), src); !strings.Contains(got, "cannot mix") {
			t.Errorf("%s: got %q", src, got)
		}
	}
}

func TestAbsoluteTemperaturesCannotBeSummed(t *testing.T) {
	for _, src := range []string{"sum(20 degC, 30 degC)", "mean(20 degC, 30 degC)", "median(20 degC, 30 degC)", "stdev(20 degC, 30 degC)"} {
		if got := evalError(t, NewEnv(), src); !strings.Contains(got, "absolute temperatures") {
			t.Errorf("%s: got %q", src, got)
		}
	}
	if got := run(t, NewEnv(), "max(20 degC, 30 degC)"); got != "30 degC" {
		t.Errorf("max of temperatures = %s, want 30 degC", got)
	}
	if got := run(t, NewEnv(), "count(20 degC, 30 degC)"); got != "2" {
		t.Errorf("count of temperatures = %s, want 2", got)
	}
}

func TestEmptyListsAreRefused(t *testing.T) {
	for _, src := range []string{"sum()", "mean()", "median()", "min()", "max()", "product()", "stdev()", "variance()", "stdevp()", "variancep()", "sum([])", "mean([])", "max([])"} {
		got := evalError(t, NewEnv(), src)
		if !strings.Contains(got, "needs at least one value") {
			t.Errorf("%s: got %q, want it to refuse the empty list", src, got)
		}
	}
	if got := run(t, NewEnv(), "count([])"); got != "0" {
		t.Errorf("count([]) = %s, want 0", got)
	}
	if got := printed(t, NewEnv(), "[]"); got != "[]" {
		t.Errorf("[] = %q", got)
	}
}

func sigmaOf(t *testing.T, env *Env, src string) float64 {
	t.Helper()
	v := mustEval(t, env, src)
	if !v.IsMeasured() {
		t.Fatalf("%s carries no uncertainty", src)
	}
	return v.Unc.Sigma
}

func TestMeanOfMeasuredValuesHasASmallerUncertainty(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 12 ± 1")
	run(t, env, "c = 11 ± 1")
	if got := sigmaOf(t, env, "mean(a, b)"); math.Abs(got-math.Sqrt(0.5)) > 1e-12 {
		t.Errorf("u(mean(a,b)) = %g, want sqrt(1/2)", got)
	}
	if got := sigmaOf(t, env, "mean(a, b, c)"); math.Abs(got-1/math.Sqrt(3)) > 1e-12 {
		t.Errorf("u(mean(a,b,c)) = %g, want 1/sqrt(3)", got)
	}
	if got := sigmaOf(t, env, "sum(a, b)"); math.Abs(got-math.Sqrt2) > 1e-12 {
		t.Errorf("u(sum(a,b)) = %g, want sqrt(2)", got)
	}
	if got := sigmaOf(t, env, "mean([a, b, c])"); math.Abs(got-1/math.Sqrt(3)) > 1e-12 {
		t.Errorf("u(mean([a,b,c])) = %g, want 1/sqrt(3)", got)
	}
	if got := sigmaOf(t, env, "mean(10 ± 1, 12 ± 1)"); math.Abs(got-math.Sqrt(0.5)) > 1e-12 {
		t.Errorf("u(mean(10±1, 12±1)) = %g, want sqrt(1/2)", got)
	}
}

func TestMeanOfOneMeasurementRepeatedIsNotShrunk(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	if got := sigmaOf(t, env, "mean(a, a, a, a)"); math.Abs(got-1) > 1e-12 {
		t.Errorf("u(mean(a,a,a,a)) = %g, want 1: the same measurement counted four times is still one measurement", got)
	}
	if got := sigmaOf(t, env, "sum(a, a)"); math.Abs(got-2) > 1e-12 {
		t.Errorf("u(sum(a,a)) = %g, want 2", got)
	}
}

func TestMeanHonoursDeclaredCorrelation(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	run(t, env, "b = 12 ± 1")
	run(t, env, "corr(a, b) = 1")
	if got := sigmaOf(t, env, "mean(a, b)"); math.Abs(got-1) > 1e-12 {
		t.Errorf("u(mean(a,b)) with correlation 1 = %g, want 1", got)
	}
}

func TestMeasuredSumsAndProductsCarryUnitsAndUncertainty(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "sum(1 m ± 0.1 m, 2 m ± 0.1 m)")
	if v.Unit != "m" || math.Abs(v.Unc.Sigma-math.Sqrt(0.02)) > 1e-12 {
		t.Errorf("sum = %s with u = %g, want m and sqrt(0.02)", v.Unit, v.Unc.Sigma)
	}
	p := mustEval(t, env, "product(2 ± 0.1, 3 ± 0.1)")
	want := math.Sqrt(0.3*0.3 + 0.2*0.2)
	if math.Abs(p.Unc.Sigma-want) > 1e-9 {
		t.Errorf("u(product) = %g, want %g", p.Unc.Sigma, want)
	}
	km := mustEval(t, env, "mean(1 km ± 0.1 km, 2 km ± 0.1 km)")
	if km.Unit != "km" || math.Abs(km.Unc.Sigma-math.Sqrt(0.01)*1000/math.Sqrt2) > 1e-6 {
		t.Errorf("mean in km = %s with u = %g m", km.Unit, km.Unc.Sigma)
	}
}

func TestStatisticsThatCannotBePropagatedAreRefused(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 10 ± 1")
	for _, src := range []string{"median(a, 2)", "min(a, 2)", "max(a, 2)", "stdev(a, 2)", "variance(a, 2)", "stdevp(a, 2)", "variancep(a, 2)"} {
		if got := evalError(t, env, src); !strings.Contains(got, "cannot carry a measured uncertainty") {
			t.Errorf("%s: got %q", src, got)
		}
	}
	if got := run(t, env, "count(a, 2)"); got != "2" {
		t.Errorf("count(a, 2) = %s, want 2", got)
	}
}

func TestRangesAreInclusive(t *testing.T) {
	env := NewEnv()
	cases := map[string]string{
		"1..5":        "[1, 2, 3, 4, 5]",
		"3..3":        "[3]",
		"-2..2":       "[-2, -1, 0, 1, 2]",
		"1..10..3":    "[1, 4, 7, 10]",
		"1..10..4":    "[1, 5, 9]",
		"5..1..-2":    "[5, 3, 1]",
		"0..1..1":     "[0, 1]",
		"1..2+2":      "[1, 2, 3, 4]",
		"count(1..7)": "7",
		"[1, 2+1]":    "[1, 3]",
	}
	for src, want := range cases {
		if got := printed(t, env, src); got != want {
			t.Errorf("%s = %s, want %s", src, got, want)
		}
	}
}

func TestRangesAreRefusedWhenTheyMakeNoSense(t *testing.T) {
	cases := map[string]string{
		"1..10000000000":      "that range has 10000000000 values: a list holds at most 100000",
		"1..100001":           "a list holds at most 100000",
		"0..1000000000000..1": "a list holds at most 100000",
		"1..2..0":             "a range cannot step by zero",
		"10..1":               "write 10..1..-1",
		"1..10..-1":           "never gets there",
		"10..1..1":            "never gets there",
		"1..2.5":              "a range runs between whole numbers",
		"0.5..3":              "a range runs between whole numbers",
		"1 m..3 m":            "a range runs between whole numbers",
		"1..[2]":              "a range runs between whole numbers",
		"1..(5 ± 1)":          "a range runs between whole numbers",
		"1..(1 < 2)":          "a range runs between whole numbers",
		"(1..3)..5":           "a range runs between whole numbers",
		"[1, 2]..[3]":         "a range runs between whole numbers",
		"1..10..(1/2)":        "a range runs between whole numbers",
	}
	for src, want := range cases {
		if got := evalError(t, NewEnv(), src); !strings.Contains(got, want) {
			t.Errorf("%s: got %q, want it to contain %q", src, got, want)
		}
	}
	if got := printed(t, NewEnv(), "count(1..100000)"); got != "100000" {
		t.Errorf("count(1..100000) = %s, want the largest allowed range to work", got)
	}
}

func TestLongListsRenderTruncatedInTheMiddle(t *testing.T) {
	env := NewEnv()
	got := printed(t, env, "1..100")
	want := "[1, 2, 3, 4, 5, …, 96, 97, 98, 99, 100]\n  100 values"
	if got != want {
		t.Errorf("1..100 = %q, want %q", got, want)
	}
	if got := printed(t, env, "1..12"); got != "[1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]" {
		t.Errorf("1..12 = %q, want every element shown", got)
	}
	if got := printed(t, env, "1..13"); !strings.Contains(got, "…") || !strings.HasSuffix(got, "13 values") {
		t.Errorf("1..13 = %q, want it truncated", got)
	}
	huge := printed(t, env, "1..100000")
	if len(huge) > 200 {
		t.Errorf("a 100000-element list printed %d characters", len(huge))
	}
}

func TestListsDisplayTheirElementsHonestly(t *testing.T) {
	env := NewEnv()
	cases := map[string]string{
		"[1, 2, 3]":            "[1, 2, 3]",
		"[1/3, 0.5]":           "[1/3, 0.5]",
		"[1 m, 2 m]":           "[1 m, 2 m]",
		"[1 km, 500 m]":        "[1 km, 500 m]",
		"[sqrt(2), 1]":         "[1.41421356237, 1]",
		"[10 ± 1, 2]":          "[10.0 ± 1.0, 2]",
		"[1 ± 0.1, 2 ± 0.1] ":  "[1.00 ± 0.10, 2.00 ± 0.10]",
		"[1, 2, 3] # remember": "[1, 2, 3]",
	}
	for src, want := range cases {
		if got := printed(t, env, src); got != want {
			t.Errorf("%s = %q, want %q", src, got, want)
		}
	}
}

func TestListsCanBeNamedAndReused(t *testing.T) {
	env := NewEnv()
	run(t, env, "xs = [4, 8, 15, 16, 23, 42]")
	if got := run(t, env, "sum(xs)"); got != "108" {
		t.Errorf("sum(xs) = %s, want 108", got)
	}
	if got := run(t, env, "mean(xs)"); got != "18" {
		t.Errorf("mean(xs) = %s, want 18", got)
	}
	if got := run(t, env, "median(xs)"); got != "31/2" {
		t.Errorf("median(xs) = %s, want 31/2", got)
	}
	run(t, env, "total(v) = sum(v)")
	if got := run(t, env, "total(1..4)"); got != "10" {
		t.Errorf("total(1..4) = %s, want 10", got)
	}
	if got := run(t, env, "sum(xs, 2)"); got != "110" {
		t.Errorf("sum(xs, 2) = %s, want 110", got)
	}
}

func TestListsRefuseArithmeticInsteadOfReadingZero(t *testing.T) {
	env := NewEnv()
	run(t, env, "xs = [1, 2]")
	for _, src := range []string{"[1, 2] + 1", "1 + [1, 2]", "2 * [1, 2]", "[1, 2] / 2", "xs + xs", "-xs", "xs!", "xs%", "xs ^ 2", "[1, 2] m", "[1, 2] to m", "[1, 2] ± 1", "1 ± [1]", "xs < 3", "xs == xs", "xs mod 2", "2 xs"} {
		if got := evalError(t, env, src); !strings.Contains(got, "cannot do arithmetic on a list") {
			t.Errorf("%s: got %q", src, got)
		}
	}
	for _, src := range []string{"sin(xs)", "sqrt(xs)", "abs(xs)", "gcd(xs, 2)", "floor(xs)", "ln(xs)", "isprime(xs)", "root(xs, 2)", "atan2(xs, 1)"} {
		if got := evalError(t, env, src); !strings.Contains(got, "takes a number, not a list") {
			t.Errorf("%s: got %q", src, got)
		}
	}
}

func TestListsHoldOnlyNumbers(t *testing.T) {
	for src, want := range map[string]string{
		"[[1], 2]":         "a list cannot hold another list",
		"[1 < 2]":          "a list holds numbers, not true/false values",
		"sum(1 < 2)":       "a list holds numbers, not true/false values",
		"sum([1, 2], 1<2)": "a list holds numbers, not true/false values",
	} {
		if got := evalError(t, NewEnv(), src); !strings.Contains(got, want) {
			t.Errorf("%s: got %q, want it to contain %q", src, got, want)
		}
	}
}

func TestOrderingBallsNeedsAProvenOrder(t *testing.T) {
	env := NewEnv()
	if got := printed(t, env, "max(pi, 3, e)"); !strings.HasPrefix(got, "3.14159265359") {
		t.Errorf("max(pi, 3, e) = %q, want pi", got)
	}
	if got := printed(t, env, "min(pi, 3, e)"); !strings.HasPrefix(got, "2.71828182846") {
		t.Errorf("min(pi, 3, e) = %q, want e", got)
	}
	if got := printed(t, env, "median(pi, 3, e)"); got != "3" {
		t.Errorf("median(pi, 3, e) = %q, want 3", got)
	}
}

func TestSumOfBallsIsAProvenBall(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "sum(pi, 1)")
	if v.Num.IsExact() {
		t.Fatalf("sum(pi, 1) is exact")
	}
	if got := printed(t, env, "sum(pi, 1)"); !strings.HasPrefix(got, "4.14159265359") {
		t.Errorf("sum(pi, 1) = %q", got)
	}
	if got := printed(t, env, "mean(pi, e)"); !strings.HasPrefix(got, "2.92993724102") {
		t.Errorf("mean(pi, e) = %q", got)
	}
}

func TestStatisticalNamesAreListed(t *testing.T) {
	for _, name := range statisticalFunctions {
		found := false
		for _, listed := range Functions() {
			found = found || listed == name
		}
		if !found {
			t.Errorf("%s is not listed by Functions()", name)
		}
	}
}

func TestMisspeltStatisticsGetASuggestion(t *testing.T) {
	cases := [][2]string{
		{"mena(1,2)", "unknown function mena: did you mean mean?"},
		{"medain(1,2)", "unknown function medain: did you mean median?"},
		{"varience(1,2)", "unknown function varience: did you mean variance?"},
	}
	for _, c := range cases {
		if got := evalError(t, NewEnv(), c[0]); !strings.HasSuffix(got, c[1]) {
			t.Errorf("%s: got %q, want it to end with %q", c[0], got, c[1])
		}
	}
}

func TestAUserDefinitionOverridesAStatistic(t *testing.T) {
	env := NewEnv()
	run(t, env, "sum(a, b) = a * b")
	if got := run(t, env, "sum(3, 4)"); got != "12" {
		t.Errorf("sum(3, 4) = %s, want the user's definition, 12", got)
	}
}

func TestCommasInsideCallsSeparateArguments(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "sum(10,200)"); got != "210" {
		t.Errorf("sum(10,200) = %s, want 210, not 10200", got)
	}
	if got := run(t, env, "max(1,234,5)"); got != "234" {
		t.Errorf("max(1,234,5) = %s, want 234", got)
	}
	if got := run(t, env, "1,000 + 1"); got != "1001" {
		t.Errorf("1,000 + 1 = %s, want 1001", got)
	}
	if got := run(t, env, "sum([100,200])"); got != "300" {
		t.Errorf("sum([100,200]) = %s, want 300", got)
	}
}

func TestListProvenanceNamesItsParts(t *testing.T) {
	env := NewEnv()
	v := mustEval(t, env, "sum(1..3)")
	trace := env.Arena().Export(v.Prov)
	var labels []string
	for _, n := range trace.Nodes {
		labels = append(labels, n.Label)
	}
	for _, want := range []string{"sum", "range", "1", "3"} {
		found := false
		for _, l := range labels {
			found = found || l == want
		}
		if !found {
			t.Errorf("provenance labels %v lack %q", labels, want)
		}
	}
}

func TestRepeatedEvaluationOfAListIsStable(t *testing.T) {
	env := NewEnv()
	first := printed(t, env, "xs = [pi, 1, 2]")
	again := printed(t, env, "xs")
	if first != again {
		t.Errorf("xs printed %q then %q", first, again)
	}
}

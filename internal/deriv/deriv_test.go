package deriv_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/deriv"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func parseOK(t *testing.T, src string) parse.Node {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return n
}

func derivative(t *testing.T, src string) parse.Node {
	t.Helper()
	d, err := deriv.Differentiate(parseOK(t, src), "x")
	if err != nil {
		t.Fatalf("Differentiate(%q): %v", src, err)
	}
	return d
}

func evalAt(t *testing.T, n parse.Node, x float64) float64 {
	t.Helper()
	env := eval.NewEnv()
	at := new(big.Rat)
	at.SetFloat64(x)
	env.Set("x", value.Scalar(num.FromRat(at)))
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval(%s) at x=%v: %v", n, x, err)
	}
	if v.Num.IsExact() {
		f, _ := v.Num.Exact().Rat().Float64()
		return f
	}
	f, _ := v.Num.Ball(128).Mid.Float64()
	return f
}

func TestDerivativeShapes(t *testing.T) {
	cases := map[string]string{
		"5":                 "0",
		"x":                 "1",
		"y":                 "0",
		"pi":                "0",
		"3x":                "3",
		"x + 7":             "1",
		"x - 7":             "1",
		"7 - x":             "(-1)",
		"x + x":             "2",
		"x * x":             "(x + x)",
		"x^2":               "(2 * x)",
		"x^3":               "(3 * (x ^ 2))",
		"x^1":               "1",
		"sin(x^2)":          "(cos((x ^ 2)) * (2 * x))",
		"x*ln(x)":           "(ln(x) + (x * (1 / x)))",
		"1/x":               "(-(1 / (x ^ 2)))",
		"x/2":               "(1 / 2)",
		"(x + 1)/(x - 1)":   "(((x - 1) - (x + 1)) / ((x - 1) ^ 2))",
		"exp(2*x)":          "(exp((2 * x)) * 2)",
		"x^x":               "((x ^ x) * (ln(x) + (x / x)))",
		"sqrt(x)":           "(1 / (2 * sqrt(x)))",
		"2^x":               "((2 ^ x) * ln(2))",
		"e^x":               "(e ^ x)",
		"cos(x)":            "(-sin(x))",
		"tan(x)":            "(1 / (cos(x) ^ 2))",
		"atan(x)":           "(1 / (1 + (x ^ 2)))",
		"log2(x)":           "(1 / (x * ln(2)))",
		"abs(x)":            "(x / abs(x))",
		"-x":                "(-1)",
		"sin(y)":            "0",
		"sin(cos(x))":       "(-(cos(cos(x)) * sin(x)))",
		"sin(exp(x^2))":     "(cos(exp((x ^ 2))) * (exp((x ^ 2)) * (2 * x)))",
		"ln(sin(x))":        "((1 / sin(x)) * cos(x))",
		"sqrt(1 - x^2)":     "(-((1 / (2 * sqrt((1 - (x ^ 2))))) * (2 * x)))",
		"x^2 + 3x + 1":      "((2 * x) + 3)",
		"a*x^2":             "(a * (2 * x))",
		"x^(1/2)":           "((1 / 2) * (x ^ ((1 / 2) - 1)))",
		"x^-1":              "(-(x ^ (-2)))",
		"sin(x)^2":          "((2 * sin(x)) * cos(x))",
		"x^2 * sin(x)":      "(((2 * x) * sin(x)) + ((x ^ 2) * cos(x)))",
		"x^2 / sin(x)":      "((((2 * x) * sin(x)) - ((x ^ 2) * cos(x))) / (sin(x) ^ 2))",
		"sinh(x) + cosh(x)": "(cosh(x) + sinh(x))",
	}
	for src, want := range cases {
		if got := deriv.Simplify(derivative(t, src)).String(); got != want {
			t.Errorf("d/dx %s = %s, want %s", src, got, want)
		}
	}
}

func TestDerivativeMatchesNumericDifferenceQuotient(t *testing.T) {
	cases := []struct {
		src    string
		points []float64
	}{
		{"x^2", []float64{-2, 0.5, 3}},
		{"x^5 - 3x^2 + 2", []float64{-1.5, 0.7, 2}},
		{"sin(x^2)", []float64{0.3, 1.1, 2}},
		{"x*ln(x)", []float64{0.4, 1.5, 6}},
		{"1/x", []float64{-3, 0.5, 4}},
		{"exp(2*x)", []float64{-1, 0.2, 1.3}},
		{"x^x", []float64{0.5, 1.5, 2.5}},
		{"sqrt(x)", []float64{0.25, 2, 9}},
		{"sin(exp(x^2))", []float64{0.2, 0.6, 1}},
		{"ln(cos(x^3))", []float64{0.2, 0.5, 0.8}},
		{"atan(sinh(x))", []float64{-1, 0.3, 1.7}},
		{"sqrt(1 - x^2)", []float64{-0.6, 0.1, 0.8}},
		{"asin(x/2)", []float64{-1, 0.3, 1.5}},
		{"acos(x^2/3)", []float64{-1, 0.4, 1.2}},
		{"tan(x)/x", []float64{0.3, 0.9, 1.2}},
		{"tanh(x)*cosh(x)", []float64{-1, 0.4, 2}},
		{"log2(x) + log10(x) + log(x)", []float64{0.5, 2, 7}},
		{"2^x", []float64{-1, 0.5, 3}},
		{"e^(x^2)", []float64{-1, 0.5, 1.2}},
		{"(x + 1)/(x^2 + 1)", []float64{-2, 0, 1.5}},
		{"abs(x)", []float64{-2.5, 0.75, 4}},
		{"x^(sin(x))", []float64{0.5, 1.5, 3}},
		{"(2x + 1)^3", []float64{-1, 0.5, 2}},
	}
	const h = 1e-5
	for _, c := range cases {
		original := parseOK(t, c.src)
		symbolic, err := deriv.Differentiate(original, "x")
		if err != nil {
			t.Fatalf("Differentiate(%q): %v", c.src, err)
		}
		simplified := deriv.Simplify(symbolic)
		for _, x := range c.points {
			numeric := (evalAt(t, original, x+h) - evalAt(t, original, x-h)) / (2 * h)
			for label, form := range map[string]parse.Node{"raw": symbolic, "simplified": simplified} {
				got := evalAt(t, form, x)
				if math.Abs(got-numeric) > 1e-6*math.Max(1, math.Abs(numeric)) {
					t.Errorf("d/dx %s at %v (%s) = %v, difference quotient %v", c.src, x, label, got, numeric)
				}
			}
		}
	}
}

func TestChainRuleThroughTwoFunctions(t *testing.T) {
	got := deriv.Simplify(derivative(t, "sin(cos(x))")).String()
	if want := "(-(cos(cos(x)) * sin(x)))"; got != want {
		t.Errorf("d/dx sin(cos(x)) = %s, want %s", got, want)
	}
	got = deriv.Simplify(derivative(t, "exp(sin(x^2))")).String()
	if want := "(exp(sin((x ^ 2))) * (cos((x ^ 2)) * (2 * x)))"; got != want {
		t.Errorf("d/dx exp(sin(x^2)) = %s, want %s", got, want)
	}
}

func TestDifferentiatesWithRespectToTheNamedVariable(t *testing.T) {
	n := parseOK(t, "x*y^2")
	byY, err := deriv.Differentiate(n, "y")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deriv.Simplify(byY).String(), "(x * (2 * y))"; got != want {
		t.Errorf("d/dy x*y^2 = %s, want %s", got, want)
	}
	byX, err := deriv.Differentiate(n, "x")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deriv.Simplify(byX).String(), "(y ^ 2)"; got != want {
		t.Errorf("d/dx x*y^2 = %s, want %s", got, want)
	}
	if _, err := deriv.Differentiate(n, ""); err == nil {
		t.Error("an empty variable name was accepted")
	}
}

func TestTheResultReparsesToItself(t *testing.T) {
	for _, src := range []string{"sin(x^2)", "x*ln(x)", "1/x", "x^x", "sqrt(1 - x^2)", "x^-1", "acos(x)"} {
		d := deriv.Simplify(derivative(t, src))
		again := parseOK(t, d.String())
		if again.String() != d.String() {
			t.Errorf("d/dx %s prints as %s but reparses as %s", src, d, again)
		}
	}
}

func TestTheResultIsANewTree(t *testing.T) {
	original := parseOK(t, "sin(x^2) * x")
	before := original.String()
	first, err := deriv.Differentiate(original, "x")
	if err != nil {
		t.Fatal(err)
	}
	deriv.Simplify(first)
	if original.String() != before {
		t.Errorf("differentiating changed the input: %s became %s", before, original)
	}
	shared := map[parse.Node]bool{}
	var collect func(parse.Node)
	collect = func(n parse.Node) {
		shared[n] = true
		switch t := n.(type) {
		case *parse.Unary:
			collect(t.X)
		case *parse.Binary:
			collect(t.L)
			collect(t.R)
		case *parse.Call:
			for _, a := range t.Args {
				collect(a)
			}
		}
	}
	collect(original)
	var check func(parse.Node)
	check = func(n parse.Node) {
		if shared[n] {
			t.Errorf("the derivative reuses the input's node %s", n)
			return
		}
		switch u := n.(type) {
		case *parse.Unary:
			check(u.X)
		case *parse.Binary:
			check(u.L)
			check(u.R)
		case *parse.Call:
			for _, a := range u.Args {
				check(a)
			}
		}
	}
	check(first)
}

func TestSimplifyTidiesWithoutChangingMeaning(t *testing.T) {
	cases := map[string]string{
		"x + 0":             "x",
		"0 + x":             "x",
		"x - 0":             "x",
		"0 - x":             "(-x)",
		"x * 1":             "x",
		"1 * x":             "x",
		"x * 0":             "0",
		"0 * x":             "0",
		"x ^ 1":             "x",
		"x ^ 0":             "1",
		"1 ^ x":             "1",
		"2 + 3":             "5",
		"2 - 5":             "(-3)",
		"4 * 6":             "24",
		"2 ^ 10":            "1024",
		"6 / 3":             "2",
		"x * -1":            "(-x)",
		"x + -y":            "(x - y)",
		"x - -y":            "(x + y)",
		"-(-x)":             "x",
		"(-x) * (-y)":       "(x * y)",
		"(-x) * y":          "(-(x * y))",
		"(1 * x) + (0 * y)": "x",
		"sin(x * 1)":        "sin(x)",
		"(x + 0) * (y ^ 1)": "(x * y)",
		"2 * (3 + 4)":       "14",
	}
	for src, want := range cases {
		if got := deriv.Simplify(parseOK(t, src)).String(); got != want {
			t.Errorf("Simplify(%s) = %s, want %s", src, got, want)
		}
	}
}

func TestSimplifyKeepsWhatItCannotProve(t *testing.T) {
	cases := map[string]string{
		"x / x":       "(x / x)",
		"0 / x":       "(0 / x)",
		"x - x":       "(x - x)",
		"ln(x) * 0":   "(ln(x) * 0)",
		"0 * (1 / x)": "(0 * (1 / x))",
		"(1 / x) ^ 0": "((1 / x) ^ 0)",
		"0 ^ 0":       "(0 ^ 0)",
		"1 ^ (1 / 0)": "(1 ^ (1 / 0))",
		"7 / 2":       "(7 / 2)",
		"2 ^ -1":      "(2 ^ (-1))",
		"x ^ 2":       "(x ^ 2)",
		"sqrt(x) ^ 2": "(sqrt(x) ^ 2)",
		"1 / 0":       "(1 / 0)",
		"x * y":       "(x * y)",
	}
	for src, want := range cases {
		if got := deriv.Simplify(parseOK(t, src)).String(); got != want {
			t.Errorf("Simplify(%s) = %s, want %s", src, got, want)
		}
	}
}

func TestSimplifyPreservesValue(t *testing.T) {
	for _, src := range []string{
		"x*1 + 0*x + (x^1)*(2 + 3)",
		"(x - 0)^2 - -x",
		"1*(x + 0) / (1*2)",
		"x^0 + 2^3^2 - (-x)*(-3)",
		"sin(x*1)*cos(0 + x) - 0",
	} {
		original := parseOK(t, src)
		simplified := deriv.Simplify(original)
		for _, x := range []float64{-2, 0.5, 3} {
			if a, b := evalAt(t, original, x), evalAt(t, simplified, x); math.Abs(a-b) > 1e-9*math.Max(1, math.Abs(a)) {
				t.Errorf("Simplify(%s) = %s changed the value at x=%v: %v became %v", src, simplified, x, a, b)
			}
		}
	}
}

func TestSimplifyLeavesItsInputAlone(t *testing.T) {
	n := parseOK(t, "(x + 0) * 1 + 0 * ln(x)")
	before := n.String()
	deriv.Simplify(n)
	if n.String() != before {
		t.Errorf("Simplify changed its input: %s became %s", before, n)
	}
}

func TestAnUnknownFunctionIsAnErrorNotAWrongAnswer(t *testing.T) {
	for _, src := range []string{"foo(x)", "floor(x)", "gcd(x, 4)", "sin(x) + round(x)", "root(x, 3)", "atan2(x, 1)", "log(x, 2)"} {
		d, err := deriv.Differentiate(parseOK(t, src), "x")
		if err == nil {
			t.Errorf("Differentiate(%q) = %s, want an error", src, d)
		}
	}
}

func TestAFunctionOfOtherVariablesIsConstant(t *testing.T) {
	d, err := deriv.Differentiate(parseOK(t, "floor(y) * x"), "x")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := deriv.Simplify(d).String(), "floor(y)"; got != want {
		t.Errorf("d/dx floor(y)*x = %s, want %s", got, want)
	}
}

func TestWhatItCannotDifferentiateIsRefusedPlainly(t *testing.T) {
	cases := map[string]string{
		"f(x) = x^2":   "definition",
		"x!":           "factorial",
		"x + 10%":      "percentage",
		"x mod 3":      "mod",
		"(x ± 1) * x":  "±",
		"x to m":       "to",
		"1/2x":         "brackets",
		"sin(x, x)":    "one argument",
		"sin(a = x)":   "one argument",
		"100 km / 2 x": "brackets",
	}
	for src, hint := range cases {
		n, _, err := parse.Parse(src)
		if err != nil {
			continue
		}
		d, err := deriv.Differentiate(n, "x")
		if err == nil {
			t.Errorf("Differentiate(%q) = %s, want an error mentioning %q", src, d, hint)
			continue
		}
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("Differentiate(%q) error = %q, want it to mention %q", src, err, hint)
		}
	}
}

func TestNilInputIsAnError(t *testing.T) {
	if _, err := deriv.Differentiate(nil, "x"); err == nil {
		t.Error("Differentiate(nil) succeeded")
	}
	if deriv.Simplify(nil) != nil {
		t.Error("Simplify(nil) returned a node")
	}
}

func deeplyNested(depth int) parse.Node {
	var n parse.Node = &parse.Name{Text: "x"}
	for i := 0; i < depth; i++ {
		n = &parse.Unary{Op: "-", X: n}
	}
	return n
}

func deeplyChained(depth int) parse.Node {
	var n parse.Node = &parse.Name{Text: "x"}
	for i := 0; i < depth; i++ {
		n = &parse.Call{Fn: "sin", Args: []parse.Node{n}}
	}
	return n
}

func TestDeepNestingIsRefusedWithoutPanicking(t *testing.T) {
	for name, n := range map[string]parse.Node{
		"negations": deeplyNested(200000),
		"sines":     deeplyChained(200000),
	} {
		d, err := deriv.Differentiate(n, "x")
		if err == nil {
			t.Errorf("Differentiate of %d-deep %s succeeded: %.40s", 200000, name, d)
			continue
		}
		if !strings.Contains(err.Error(), "deeply nested") {
			t.Errorf("%s: error %q does not say the expression is too deeply nested", name, err)
		}
		if got := deriv.Simplify(n); got == nil {
			t.Errorf("Simplify of deep %s returned nil", name)
		}
	}
}

func TestAModeratelyDeepExpressionStillWorks(t *testing.T) {
	d, err := deriv.Differentiate(deeplyChained(100), "x")
	if err != nil {
		t.Fatalf("a 100-deep chain of sines was refused: %v", err)
	}
	if got := evalAt(t, deriv.Simplify(d), 0.001); math.IsNaN(got) {
		t.Error("the derivative of a 100-deep chain evaluated to NaN")
	}
}

func TestAnExplodingDerivativeIsRefusedNotBuilt(t *testing.T) {
	var n parse.Node = &parse.Name{Text: "x"}
	for i := 0; i < 400; i++ {
		n = &parse.Binary{Op: "^", L: &parse.Name{Text: "x"}, R: n}
	}
	if d, err := deriv.Differentiate(n, "x"); err == nil {
		t.Errorf("a 400-high power tower was differentiated into %.40s", d)
	}
}

func TestPositionsAreCarriedOver(t *testing.T) {
	d := derivative(t, "  sin(x)")
	if got := d.Pos(); got.Line != 1 || got.Col == 0 {
		t.Errorf("the derivative has position %v, want one from the source", got)
	}
}

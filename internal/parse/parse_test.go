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
		"1 + 2 * 3":       "(1 + (2 * 3))",
		"2 ^ 3 ^ 2":       "(2 ^ (3 ^ 2))",
		"-2 ^ 2":          "(-(2 ^ 2))",
		"2x":              "(2 * x)",
		"2(x + 1)":        "(2 * (x + 1))",
		"1/2x":            "((1 / 2) * x)",
		"2 m^2":           "(2 * (m ^ 2))",
		"2 pi^2":          "(2 * (pi ^ 2))",
		"2 (3)^2":         "(2 * (3 ^ 2))",
		"2^2 m":           "((2 ^ 2) * m)",
		"100 km / 2 h":    "(((100 * km) / 2) * h)",
		"60 km/h":         "((60 * km) / h)",
		"9.8 m/s^2":       "((9.8 * m) / (s ^ 2))",
		"3 m / 2 s / 4 s": "(((((3 * m) / 2) * s) / 4) * s)",
		"18% of 500":      "((18 %) of 500)",
		"5 m + 3 s":       "((5 * m) + (3 * s))",
		"17 mod 5":        "(17 mod 5)",
		"500 + 10%":       "(500 + (10 %))",
		"3 km to m":       "((3 * km) to m)",
	}
	for src, want := range cases {
		if got := parseOK(t, src).String(); got != want {
			t.Errorf("Parse(%q) = %s, want %s", src, got, want)
		}
	}
}

func TestTheParserLeavesADivisionsImplicitTermForTheEvaluator(t *testing.T) {
	for _, src := range []string{"1/2x", "100 km / 2 h", "100 W / 2 m^2", "3 m / 2 s / 4 s"} {
		n, notes, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if len(notes) != 0 {
			t.Errorf("Parse(%q) announced %v; the reading turns on whether the name is a unit, which only the evaluator knows", src, notes)
		}
		b, ok := n.(*Binary)
		if !ok || !b.Implicit || !InDenominator(b.L) {
			t.Errorf("Parse(%q) = %s, want the flat form with the implicit term reachable from a division", src, n)
		}
	}
	for _, src := range []string{"(1/2) x", "3 m/(2 s)", "60 km/h", "2 m^2"} {
		n, _, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if b, ok := n.(*Binary); ok && b.Implicit && InDenominator(b.L) {
			t.Errorf("Parse(%q) = %s, want no division to reach the implicit term", src, n)
		}
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

var implicitOperands = []string{"m", "km", "pi", "x", "(x + 1)", "sqrt(x)", "(2)"}

var implicitNumerals = []string{"2", "0.5", "1e3", "100"}

var implicitExponents = []string{"2", "3", "-1", "0"}

func TestPowerBindsTighterThanImplicitMultiplication(t *testing.T) {
	for _, n := range implicitNumerals {
		for _, operand := range implicitOperands {
			for _, k := range implicitExponents {
				src := n + " " + operand + "^" + k
				want := parseOK(t, n+" * ("+operand+"^"+k+")").String()
				if got := parseOK(t, src).String(); got != want {
					t.Errorf("Parse(%q) = %s, want %s", src, got, want)
				}
			}
		}
	}
}

func TestANumeralTermAfterDivisionKeepsThePowerWithItsBase(t *testing.T) {
	for _, n := range implicitNumerals {
		for _, operand := range implicitOperands {
			for _, tail := range []string{"", "^2", "^-1"} {
				src := "100 W / " + n + " " + operand + tail
				want := parseOK(t, "(100 W / "+n+") ("+operand+tail+")").String()
				if got := parseOK(t, src).String(); got != want {
					t.Errorf("Parse(%q) = %s, want %s", src, got, want)
				}
			}
		}
	}
}

func TestEveryRegroupingIsAnnounced(t *testing.T) {
	quiet := []string{"2 m", "60 km/h", "9.8 m/s^2", "1 J/(mol K)", "2 (m^2)", "3 m/(2 s)", "2^2 m", "1 + 2 * 3",
		"100 km / 2 h", "100 W / 2 m^2", "3 m / 2 s / 4 s", "1/2x"}
	for _, src := range quiet {
		_, notes, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if len(notes) != 0 {
			t.Errorf("Parse(%q) announced %v, want silence", src, notes)
		}
	}
	loud := map[string]string{
		"2 m^2":       "read as 2·(m^2)",
		"2 pi^2":      "read as 2·(pi^2)",
		"(1/2) m^2":   "read as (1/2)·(m^2)",
		"2 sqrt(x)^2": "read as 2·(sqrt(x)^2)",
	}
	for src, want := range loud {
		_, notes, err := Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if len(notes) != 1 || notes[0].Msg != want {
			t.Errorf("Parse(%q) announced %v, want %q", src, notes, want)
		}
	}
}

func TestNegativeExponentDoesNotSwallowTheImplicitProduct(t *testing.T) {
	cases := map[string]string{
		"10^-3 m":       "((10 ^ (-3)) * m)",
		"2^-1 pi":       "((2 ^ (-1)) * pi)",
		"2^-1 3":        "((2 ^ (-1)) * 3)",
		"2^-2 kg":       "((2 ^ (-2)) * kg)",
		"2^-1 sqrt(4)":  "((2 ^ (-1)) * sqrt(4))",
		"2^-1 2^2":      "((2 ^ (-1)) * (2 ^ 2))",
		"9.8 * 10^-3 m": "((9.8 * (10 ^ (-3))) * m)",
		"-2 m":          "((-2) * m)",
		"-2^2":          "(-(2 ^ 2))",
		"-x y":          "((-x) * y)",
	}
	for src, want := range cases {
		if got := parseOK(t, src).String(); got != want {
			t.Errorf("Parse(%q) = %s, want %s", src, got, want)
		}
	}
}

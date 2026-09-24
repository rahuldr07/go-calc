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
		"1 + 2 * 3": "(1 + (2 * 3))",
		"2 ^ 3 ^ 2": "(2 ^ (3 ^ 2))",
		"-2 ^ 2":    "(-(2 ^ 2))",
		"2x":        "(2 * x)",
		"2(x + 1)":  "(2 * (x + 1))",
		"1/2x":      "((1 / 2) * x)",
		"5 m + 3 s": "((5 * m) + (3 * s))",
		"17 mod 5":  "(17 mod 5)",
		"500 + 10%": "(500 + (10 %))",
		"3 km to m": "((3 * km) to m)",
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

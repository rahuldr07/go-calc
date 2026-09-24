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
	if _, err := FromInt(1).Div(FromInt(0)); err == nil {
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
	if got := FromInt(2).Pow(10).String(); got != "1024" {
		t.Errorf("2^10 = %s, want 1024", got)
	}
}

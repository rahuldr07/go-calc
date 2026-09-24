package units

import (
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
)

func TestLookupWithPrefix(t *testing.T) {
	u, ok := Lookup("km")
	if !ok {
		t.Fatal("km not found")
	}
	if u.Factor.String() != "1000" {
		t.Errorf("km factor %s, want 1000", u.Factor)
	}
	if u.Dim != length {
		t.Errorf("km dim %v, want length", u.Dim)
	}
}

func TestExactImperialFactor(t *testing.T) {
	u, _ := Lookup("inch")
	if u.Factor.String() != "127/5000" {
		t.Errorf("inch factor %s, want 127/5000 exactly", u.Factor)
	}
}

func TestBinaryPrefix(t *testing.T) {
	u, ok := Lookup("KiB")
	if !ok {
		t.Fatal("KiB not found")
	}
	if u.Factor.String() != "8192" {
		t.Errorf("KiB factor %s, want 8192 bits", u.Factor)
	}
}

func TestConvertLength(t *testing.T) {
	km, _ := Lookup("km")
	m, _ := Lookup("m")
	got, err := Convert(num.FromInt(3), km, m)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.String() != "3000" {
		t.Errorf("3 km = %s m, want 3000", got)
	}
}

func TestConvertRejectsMismatch(t *testing.T) {
	m, _ := Lookup("m")
	s, _ := Lookup("s")
	if _, err := Convert(num.FromInt(1), m, s); err == nil {
		t.Fatal("want error converting length to time")
	}
}

func TestDimWords(t *testing.T) {
	if got := length.Words(); got != "length" {
		t.Errorf("Words() = %q, want length", got)
	}
	if got := (Dim{1, 0, -1, 0, 0, 0, 0, 0}).Words(); got != "length/time" {
		t.Errorf("Words() = %q, want length/time", got)
	}
	if got := (Dim{2, 0, 0, 0, 0, 0, 0, 0}).Words(); got != "length^2" {
		t.Errorf("Words() = %q, want length^2", got)
	}
}

func TestCelsiusIsAffine(t *testing.T) {
	c, ok := Lookup("degC")
	if !ok {
		t.Fatal("degC not found")
	}
	if c.Offset.String() != "5463/20" {
		t.Errorf("degC offset %s, want 5463/20", c.Offset)
	}
	k, _ := Lookup("K")
	got, err := Convert(num.FromInt(20), c, k)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if got.String() != "5863/20" {
		t.Errorf("20 degC = %s K, want 5863/20", got)
	}
}

func TestSISymbolParenthesisesAMultipleDenominator(t *testing.T) {
	cases := []struct {
		dim  Dim
		want string
	}{
		{Dim{1, 0, -1, 0, -1, 0, 0, 0}, "m/(s·K)"},
		{Dim{1, 0, -1, 0, 0, 0, 0, 0}, "m/s"},
		{Dim{1, 0, -2, 0, 0, 0, 0, 0}, "m/s^2"},
		{Dim{0, 0, -1, 0, -1, 0, 0, 0}, "1/(s·K)"},
		{Dim{0, 0, -1, 0, 0, 0, 0, 0}, "1/s"},
		{Dim{2, 1, -2, 0, 0, 0, 0, 0}, "m^2·kg/s^2"},
		{Dim{}, ""},
	}
	for _, c := range cases {
		if got := SISymbol(c.dim); got != c.want {
			t.Errorf("SISymbol(%v) = %q, want %q", c.dim, got, c.want)
		}
		if c.want == "" {
			continue
		}
		u, ok := Lookup(c.want)
		if !ok {
			t.Errorf("SISymbol(%v) = %q does not parse back", c.dim, c.want)
			continue
		}
		if u.Dim != c.dim {
			t.Errorf("%q parses back as %v, want %v", c.want, u.Dim, c.dim)
		}
	}
}

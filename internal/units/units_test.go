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

package consts

import "testing"

func TestLookupExactConstant(t *testing.T) {
	c, ok := Lookup("c")
	if !ok {
		t.Fatal("c not found")
	}
	if !c.Exact {
		t.Error("speed of light must be exact")
	}
	if c.Value.String() != "299792458" {
		t.Errorf("c = %s, want 299792458", c.Value)
	}
	if c.Unit != "m/s" {
		t.Errorf("unit %q, want m/s", c.Unit)
	}
}

func TestLookupMeasuredConstantCarriesUncertainty(t *testing.T) {
	g, ok := Lookup("G")
	if !ok {
		t.Fatal("G not found")
	}
	if g.Exact {
		t.Error("G is measured, not exact")
	}
	if g.Uncertainty.Sign() == 0 {
		t.Error("G must carry an uncertainty")
	}
	if g.Source == "" {
		t.Error("G must carry a citation")
	}
}

func TestChecksumIsStable(t *testing.T) {
	if got := Checksum(); got != checksumPin {
		t.Errorf("checksum = %q, pinned %q: the embedded table changed", got, checksumPin)
	}
}

package consts

import (
	"math/big"
	"testing"
)

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

func TestEveryConstantMatchesItsPublishedValue(t *testing.T) {
	published := map[string]struct {
		value       string
		uncertainty string
		unit        string
	}{
		"c":        {"2.99792458e8", "0", "m/s"},
		"h":        {"6.62607015e-34", "0", "J·s"},
		"hbar":     {"1.054571817e-34", "0", "J·s"},
		"e_charge": {"1.602176634e-19", "0", "C"},
		"k_B":      {"1.380649e-23", "0", "J/K"},
		"N_A":      {"6.02214076e23", "0", "1/mol"},
		"g0":       {"9.80665", "0", "m/s^2"},
		"G":        {"6.67430e-11", "1.5e-15", "m^3/(kg·s^2)"},
		"m_e":      {"9.1093837139e-31", "2.8e-40", "kg"},
		"m_p":      {"1.67262192595e-27", "5.2e-37", "kg"},
		"alpha":    {"7.2973525643e-3", "1.1e-12", ""},
		"R":        {"8.314462618", "0", "J/(mol·K)"},
	}
	if len(published) != len(All()) {
		t.Fatalf("the table holds %d constants, this test pins %d", len(All()), len(published))
	}
	for name, want := range published {
		c, ok := Lookup(name)
		if !ok {
			t.Errorf("%s is not in the table", name)
			continue
		}
		wantValue, ok := new(big.Rat).SetString(want.value)
		if !ok {
			t.Fatalf("%s: cannot read the published value %q", name, want.value)
		}
		if c.Value.Rat().Cmp(wantValue) != 0 {
			t.Errorf("%s = %s, CODATA 2022 says %s", name, c.Value.Rat().FloatString(40), want.value)
		}
		wantUncertainty, ok := new(big.Rat).SetString(want.uncertainty)
		if !ok {
			t.Fatalf("%s: cannot read the published uncertainty %q", name, want.uncertainty)
		}
		if c.Uncertainty.Rat().Cmp(wantUncertainty) != 0 {
			t.Errorf("%s uncertainty = %s, CODATA 2022 says %s", name, c.Uncertainty.Rat().FloatString(45), want.uncertainty)
		}
		if c.Unit != want.unit {
			t.Errorf("%s unit = %q, want %q", name, c.Unit, want.unit)
		}
		if c.Exact != (want.uncertainty == "0") {
			t.Errorf("%s exact = %v", name, c.Exact)
		}
	}
}

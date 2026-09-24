package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func mustRat(t *testing.T, s string) num.Exact {
	t.Helper()
	v, err := num.ParseExact(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestTextShowsExactAndDecimal(t *testing.T) {
	got := Text(value.Scalar(mustRat(t, "1/3")), nil)
	if !strings.Contains(got, "1/3") || !strings.Contains(got, "0.333333333333") {
		t.Errorf("Text = %q, want the fraction and a decimal", got)
	}
}

func TestTextKeepsTerminatingDecimalsPlain(t *testing.T) {
	if got := Text(value.Scalar(mustRat(t, "3/10")), nil); got != "0.3" {
		t.Errorf("Text = %q, want 0.3", got)
	}
	if got := Text(value.Scalar(num.FromInt(550)), nil); got != "550" {
		t.Errorf("Text = %q, want 550", got)
	}
}

func TestTextShowsUnit(t *testing.T) {
	u, _ := units.Lookup("m")
	if got := Text(value.WithUnit(num.FromInt(3000), u), nil); got != "3000 m" {
		t.Errorf("Text = %q, want 3000 m", got)
	}
}

func TestIndianGrouping(t *testing.T) {
	s := Style{Digits: 12, Grouped: true, Indian: true}
	if got := s.Number(value.Scalar(num.FromInt(1850000))); got != "18,50,000" {
		t.Errorf("Number = %q, want 18,50,000", got)
	}
	w := Style{Digits: 12, Grouped: true}
	if got := w.Number(value.Scalar(num.FromInt(1850000))); got != "1,850,000" {
		t.Errorf("Number = %q, want 1,850,000", got)
	}
}

func TestJSONSchema(t *testing.T) {
	u, _ := units.Lookup("m")
	b, err := JSON("3 km to m", value.WithUnit(num.FromInt(3000), u), nil)
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["schema"] != Schema {
		t.Errorf("schema = %v", out["schema"])
	}
	if out["input"] != "3 km to m" {
		t.Errorf("input = %v", out["input"])
	}
	unit, ok := out["unit"].(map[string]any)
	if !ok || unit["symbol"] != "m" {
		t.Errorf("unit = %v", out["unit"])
	}
}

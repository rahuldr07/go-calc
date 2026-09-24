package format

import (
	"encoding/json"
	"strconv"
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

func TestJSONDecimalIsABareNumber(t *testing.T) {
	cases := []struct {
		name    string
		value   value.Value
		decimal string
		digits  int
	}{
		{"non-terminating", value.Scalar(mustRat(t, "1/3")), "0.333333333333", 12},
		{"terminating", value.Scalar(mustRat(t, "3/10")), "0.3", 1},
		{"integer", value.Scalar(num.FromInt(3000)), "3000", 4},
	}
	for _, c := range cases {
		b, err := JSON("x", c.value, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var out struct {
			Value struct {
				Exact   string `json:"exact"`
				Decimal string `json:"decimal"`
				Digits  int    `json:"digits"`
			} `json:"value"`
		}
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if out.Value.Decimal != c.decimal {
			t.Errorf("%s: decimal = %q, want %q", c.name, out.Value.Decimal, c.decimal)
		}
		if _, err := strconv.ParseFloat(out.Value.Decimal, 64); err != nil {
			t.Errorf("%s: decimal %q does not parse as a number: %v", c.name, out.Value.Decimal, err)
		}
		if out.Value.Digits != c.digits {
			t.Errorf("%s: digits = %d, want %d", c.name, out.Value.Digits, c.digits)
		}
	}
}

func TestJSONBallCarriesTheFractionSeparately(t *testing.T) {
	b, err := JSON("1/3", value.Scalar(mustRat(t, "1/3")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"exact": "1/3"`) {
		t.Errorf("JSON = %s, want the fraction in exact", b)
	}
	if strings.Contains(string(b), "≈") {
		t.Errorf("JSON = %s, the approximation marker belongs in text output only", b)
	}
}

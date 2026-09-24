package format

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/unc"
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

func decodeJSON(t *testing.T, v value.Value) (decimal string, digits int, ball string) {
	t.Helper()
	raw, err := JSON("x", v, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Value struct {
			Exact   string `json:"exact"`
			Ball    string `json:"ball"`
			Decimal string `json:"decimal"`
			Digits  int    `json:"digits"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("invalid JSON %s: %v", raw, err)
	}
	return out.Value.Decimal, out.Value.Digits, out.Value.Ball
}

func bigBall(t *testing.T, lit string, radius float64) num.Ball {
	t.Helper()
	mid, _, err := big.ParseFloat(lit, 10, 200, big.ToNearestEven)
	if err != nil {
		t.Fatal(err)
	}
	return num.NewBall(mid, num.MagFromFloat(radius))
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
		{"negative exact", value.Scalar(mustRat(t, "-1/3")), "-0.333333333333", 12},
		{"narrow ball", value.ScalarNum(num.NumBall(bigBall(t, "1.414213562373095048801688724", 1e-27))), "1.41421356237", 12},
		{"negative ball", value.ScalarNum(num.NumBall(bigBall(t, "-1.414213562373095048801688724", 1e-27))), "-1.41421356237", 12},
		{"scientific ball", value.ScalarNum(num.NumBall(bigBall(t, "1.414213562373095048801688724e30", 1e-6))), "1.41421356237e+30", 12},
		{"zero digits, straddling", value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(0.5), num.MagFromFloat(1)))), "", 0},
		{"zero digits, lopsided", value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(-0.0300926553811), num.MagFromFloat(0.0300927)))), "", 0},
		{"zero digits, clear of zero", value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(3), num.MagFromFloat(2)))), "", 0},
	}
	for _, c := range cases {
		decimal, digits, ball := decodeJSON(t, c.value)
		if decimal != c.decimal {
			t.Errorf("%s: decimal = %q, want %q", c.name, decimal, c.decimal)
		}
		if digits != c.digits {
			t.Errorf("%s: digits = %d, want %d", c.name, digits, c.digits)
		}
		if decimal == "" {
			if ball == "" {
				t.Errorf("%s: no decimal and no ball, the value went missing", c.name)
			}
			continue
		}
		if _, err := strconv.ParseFloat(decimal, 64); err != nil {
			if ne, ok := err.(*strconv.NumError); !ok || ne.Err != strconv.ErrRange {
				t.Errorf("%s: decimal %q is not a number: %v", c.name, decimal, err)
			}
		}
		if strings.ContainsAny(decimal, "\u00b1/\u2248") {
			t.Errorf("%s: decimal %q carries something that is not a number", c.name, decimal)
		}
	}
}

func TestJSONDigitsCountWhatIsProvenNotWhatIsPrinted(t *testing.T) {
	cases := []struct {
		name   string
		value  value.Value
		digits int
	}{
		{"exponent is not a digit", value.ScalarNum(num.NumBall(bigBall(t, "1.414213562373095048801688724e30", 1e-6))), 12},
		{"trimmed zeros are still proven", value.ScalarNum(num.NumBall(bigBall(t, "2.0000000000000000000000000000", 1e-28))), 12},
		{"a radius is not a digit", value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(4.51389830716), num.MagFromFloat(3.00927)))), 0},
		{"a short ball claims only what it proves", value.ScalarNum(num.NumBall(bigBall(t, "2.5", 1e-3))), 3},
	}
	for _, c := range cases {
		decimal, digits, _ := decodeJSON(t, c.value)
		if digits != c.digits {
			t.Errorf("%s: digits = %d for decimal %q, want %d", c.name, digits, decimal, c.digits)
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

func measuredValue(t *testing.T, mid string, sigma float64) value.Value {
	t.Helper()
	v := value.Scalar(mustRat(t, mid))
	v.Sens = unc.Sens{0: 1}
	v.Unc = unc.Report{Sigma: sigma, Budget: []unc.Contribution{{Name: "h", Share: 1, Sigma: sigma, Sensitivity: 1}}}
	return v
}

func TestUncertaintyRoundsToTwoSignificantDigits(t *testing.T) {
	cases := []struct {
		mid   string
		sigma float64
		want  string
	}{
		{"1.58382507996", 0.0012, "1.5838 ± 0.0012"},
		{"1.58382507996", 0.0128765, "1.584 ± 0.013"},
		{"1234.5678", 13, "1235 ± 13"},
		{"1234.5678", 130, "1230 ± 130"},
	}
	for _, c := range cases {
		if got := Text(measuredValue(t, c.mid, c.sigma), nil); got != c.want {
			t.Errorf("%s ± %g = %q, want %q", c.mid, c.sigma, got, c.want)
		}
	}
}

func TestJSONCarriesTheBudget(t *testing.T) {
	v := measuredValue(t, "1.584", 0.0128765)
	b, err := JSON("t", v, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	u, ok := out["uncertainty"].(map[string]any)
	if !ok {
		t.Fatal("no uncertainty block")
	}
	if u["reported"] != "0.013" || u["value"] != "1.584" {
		t.Errorf("uncertainty = %v", u)
	}
	budget, ok := u["budget"].([]any)
	if !ok || len(budget) != 1 {
		t.Fatalf("budget = %v", u["budget"])
	}
	if budget[0].(map[string]any)["name"] != "h" {
		t.Errorf("budget entry = %v", budget[0])
	}
}

func TestJSONOmitsUncertaintyWhenThereIsNone(t *testing.T) {
	b, err := JSON("2", value.Scalar(mustRat(t, "2")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "uncertainty") {
		t.Errorf("uncertainty block appeared for an exact value: %s", b)
	}
}

func wideBall(t *testing.T) num.Ball {
	t.Helper()
	mid := new(big.Float).SetPrec(64).SetFloat64(1.234)
	return num.Ball{Mid: mid, Rad: num.MagFromFloat(0.01)}
}

func TestDegenerateStillReportsWhatIsProven(t *testing.T) {
	v := value.ScalarNum(num.NumBall(wideBall(t)))
	v.Sens = nil
	v.Curv = unc.Curv{unc.Pair{A: 0, B: 0}: 2}
	v.Unc = unc.Report{Degenerate: true, Warning: unc.NonlinearWarning}
	got := Text(v, nil)
	if !strings.HasPrefix(got, "1.2 ± ?") {
		t.Errorf("text = %q, want it to start with 1.2 ± ?", got)
	}
	if !strings.Contains(got, "guaranteed") {
		t.Errorf("text = %q, want the precision note kept in the degenerate branch", got)
	}
	if !strings.Contains(got, unc.NonlinearWarning) {
		t.Errorf("text = %q, want the warning", got)
	}
}

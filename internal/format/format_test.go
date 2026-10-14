package format

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/prov"
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

func TestJSONDetailCarriesProvenanceAndWarning(t *testing.T) {
	a := prov.NewArena()
	one := a.Add(prov.Node{Kind: prov.Literal, Label: "1"})
	root := a.Add(prov.Node{Kind: prov.Operation, Label: "-", Children: []prov.NodeID{one}})
	v := value.Scalar(num.FromInt(1))
	v.Prov = root
	raw, err := JSONDetail("x", v, nil, Detail{Arena: a, Warning: "float64 would answer 0 here"})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Warning    string `json:"warning"`
		Provenance *struct {
			Root  prov.NodeID `json:"root"`
			Nodes []struct {
				ID   prov.NodeID `json:"id"`
				Kind string      `json:"kind"`
			} `json:"nodes"`
		} `json:"provenance"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Warning != "float64 would answer 0 here" {
		t.Errorf("warning = %q", out.Warning)
	}
	if out.Provenance == nil || out.Provenance.Root != root || len(out.Provenance.Nodes) != 2 {
		t.Fatalf("provenance = %+v", out.Provenance)
	}
}

func TestJSONWithoutADetailOmitsBothKeys(t *testing.T) {
	raw, err := JSON("2", value.Scalar(num.FromInt(2)), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"provenance"`, `"warning"`} {
		if strings.Contains(string(raw), key) {
			t.Errorf("plain JSON carries %s:\n%s", key, raw)
		}
	}
}

func TestProvenanceNeverBorrowsThePlusMinusSign(t *testing.T) {
	a := prov.NewArena()
	id := a.Add(prov.Node{Kind: prov.Unit, Label: "km", Detail: "1 km = 1000 m"})
	if strings.Contains(a.Tree(id), "±") {
		t.Errorf("a unit node borrowed ±:\n%s", a.Tree(id))
	}
}

func TestDefinitionPrintsWhatWasDefined(t *testing.T) {
	v := value.Definition("f", []string{"x"})
	if got := Default.Text(v, nil); got != "f(x) defined" {
		t.Errorf("Text = %q, want %q", got, "f(x) defined")
	}
	if got := Default.Text(value.Definition("g", []string{"a", "b"}), nil); got != "g(a, b) defined" {
		t.Errorf("Text = %q, want %q", got, "g(a, b) defined")
	}
}

func TestDefinitionJSONCarriesNoValue(t *testing.T) {
	raw, err := JSON("f(x) = x", value.Definition("f", []string{"x"}), nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("invalid JSON %s: %v", raw, err)
	}
	if out["definition"] != "f(x)" {
		t.Errorf("definition = %v, want f(x)", out["definition"])
	}
	if _, ok := out["value"]; ok {
		t.Errorf("a definition carried a value: %s", raw)
	}
}

func TestOrdinaryResultStillCarriesAValue(t *testing.T) {
	raw, err := JSON("2 + 3", value.Scalar(num.FromInt(5)), nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("invalid JSON %s: %v", raw, err)
	}
	if _, ok := out["value"]; !ok {
		t.Errorf("an ordinary result lost its value: %s", raw)
	}
	if _, ok := out["definition"]; ok {
		t.Errorf("an ordinary result claimed to be a definition: %s", raw)
	}
}

func TestASmallMeasuredValueIsSpeltInScientificNotation(t *testing.T) {
	cases := []struct {
		mid   string
		sigma float64
		want  string
	}{
		{"0.000000000000000000000000000000000662607015", 1e-42, "6.626070150e-34 ± 1.0e-42"},
		{"0.00000000000000000001", 1e-22, "1.000e-20 ± 1.0e-22"},
		{"0.000000000000000000123", 2e-21, "1.230e-19 ± 2.0e-21"},
		{"0.0001", 0.000002, "0.0001000 ± 0.0000020"},
		{"1.584", 0.0128765, "1.584 ± 0.013"},
	}
	for _, c := range cases {
		got := Text(measuredValue(t, c.mid, c.sigma), nil)
		if got != c.want {
			t.Errorf("%s ± %g = %q, want %q", c.mid, c.sigma, got, c.want)
		}
		if len(got) > 40 {
			t.Errorf("%s ± %g spells out %d characters", c.mid, c.sigma, len(got))
		}
	}
}

func TestEveryMeasuredValuePrintsTwoUncertaintyDigitsAndReadsBack(t *testing.T) {
	for midScale := -34; midScale <= 12; midScale++ {
		for sigmaScale := -38; sigmaScale <= 8; sigmaScale++ {
			mid := "1.2345678901e" + strconv.Itoa(midScale)
			sigmaText := "2.3e" + strconv.Itoa(sigmaScale)
			sigma, _ := new(big.Rat).SetString(sigmaText)
			f, _ := sigma.Float64()
			v := measuredValue(t, mid, f)
			text, u, ok := Default.Uncertain(v)
			if !ok {
				continue
			}
			if strings.ContainsRune(u, 'e') {
				if n := significantDigits(mantissaOf(u)); n != uncertaintyDigits {
					t.Errorf("%s ± %s printed ± %q, whose mantissa claims %d significant digits", mid, sigmaText, u, n)
				}
			}
			for _, token := range []string{text, u} {
				if _, ok := new(big.Rat).SetString(token); !ok {
					t.Errorf("%s ± %s: %q does not read back as a number", mid, sigmaText, token)
				}
			}
		}
	}
}

func mantissaOf(text string) string {
	if i := strings.IndexByte(text, 'e'); i >= 0 {
		return text[:i]
	}
	return text
}

func integerList(from, to int64) value.Value {
	var items []value.Value
	for i := from; i <= to; i++ {
		items = append(items, value.Scalar(num.FromInt(i)))
	}
	return value.List(items)
}

func TestTextShowsAShortListInFull(t *testing.T) {
	if got := Text(integerList(1, 3), nil); got != "[1, 2, 3]" {
		t.Errorf("Text = %q, want [1, 2, 3]", got)
	}
	if got := Text(value.List(nil), nil); got != "[]" {
		t.Errorf("Text = %q, want []", got)
	}
}

func TestTextTruncatesALongListInTheMiddle(t *testing.T) {
	got := Text(integerList(1, 10000), nil)
	want := "[1, 2, 3, 4, 5, …, 9996, 9997, 9998, 9999, 10000]\n  10000 values"
	if got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if got := Text(integerList(1, 12), nil); strings.Contains(got, "…") {
		t.Errorf("Text = %q, want twelve elements shown in full", got)
	}
}

func TestTextKeepsExactFractionsAndUnitsInsideAList(t *testing.T) {
	u, _ := units.Lookup("m")
	list := value.List([]value.Value{
		value.Scalar(mustRat(t, "1/3")),
		value.Scalar(mustRat(t, "3/10")),
		value.WithUnit(num.FromInt(2), u),
	})
	if got := Text(list, nil); got != "[1/3, 0.3, 2 m]" {
		t.Errorf("Text = %q, want [1/3, 0.3, 2 m]", got)
	}
}

func TestJSONOfAListCarriesEveryElement(t *testing.T) {
	data, err := JSON("[1, 2]", integerList(1, 2), nil)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Value *ValueOut `json:"value"`
		List  struct {
			Count int `json:"count"`
			Items []struct {
				Value ValueOut `json:"value"`
			} `json:"items"`
		} `json:"list"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Value != nil {
		t.Errorf("a list has no single value, got %+v", out.Value)
	}
	if out.List.Count != 2 || len(out.List.Items) != 2 || out.List.Items[1].Value.Exact != "2" {
		t.Errorf("list = %+v, want two exact elements", out.List)
	}
}

func TestValueDetailOfAListIsEmptyButPresent(t *testing.T) {
	out, unit := Default.ValueDetail(integerList(1, 2))
	if out == nil || out.Exact != "" || unit != nil {
		t.Errorf("ValueDetail = %+v, %+v, want an empty value and no unit", out, unit)
	}
}

func TestJSONStatesTruthInsteadOfANumber(t *testing.T) {
	for _, c := range []struct {
		v     value.Value
		truth string
	}{
		{value.Boolean(true), `"truth": "true"`},
		{value.Boolean(false), `"truth": "false"`},
		{value.Probable("not a proof"), `"truth": "probably true"`},
	} {
		b, err := JSON("x", c.v, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), c.truth) || strings.Contains(string(b), `"value"`) {
			t.Errorf("JSON = %s, want %s and no numeric value", b, c.truth)
		}
	}
	b, _ := JSON("x", value.Probable("not a proof"), nil)
	if !strings.Contains(string(b), `"caveat": "not a proof"`) {
		t.Errorf("a probable answer lost its caveat: %s", b)
	}
}

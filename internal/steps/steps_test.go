package steps

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/prov"
)

func derive(t *testing.T, input string) Derivation {
	t.Helper()
	d, err := Of(input, 12)
	if err != nil {
		t.Fatalf("Of(%q): %v", input, err)
	}
	return d
}

func exprs(d Derivation) []string {
	out := make([]string, len(d.Steps))
	for i, s := range d.Steps {
		out[i] = s.Expr
	}
	return out
}

func TestInnermostFirst(t *testing.T) {
	d := derive(t, "(1+2)*(3+4)")
	want := []string{"1+2", "3+4", "(1+2)*(3+4)"}
	got := exprs(d)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("steps = %v, want %v", got, want)
	}
	results := []string{"3", "7", "21"}
	for i, s := range d.Steps {
		if s.Result != results[i] || s.N != i+1 {
			t.Errorf("step %d = %+v, want result %s", i, s, results[i])
		}
	}
	last := d.Steps[2]
	if len(last.From) != 2 || last.From[0] != "3" || last.From[1] != "7" {
		t.Errorf("last step consumed %v, want [3 7]", last.From)
	}
	if d.Answer != "21" || d.Status != "exact" || d.Note != "" {
		t.Errorf("answer = %q status = %q note = %q", d.Answer, d.Status, d.Note)
	}
}

func TestConstantCitesItsSource(t *testing.T) {
	d := derive(t, "m_e * c^2")
	var cited []Step
	for _, s := range d.Steps {
		if s.Source != "" {
			cited = append(cited, s)
		}
	}
	if len(cited) != 2 || cited[0].Expr != "m_e" || cited[1].Expr != "c" {
		t.Fatalf("cited steps = %+v", cited)
	}
	for _, s := range cited {
		if !strings.Contains(s.Source, "CODATA 2022") {
			t.Errorf("source for %s = %q", s.Expr, s.Source)
		}
	}
	if got := d.Steps[len(d.Steps)-1].Expr; got != "m_e*c^2" {
		t.Errorf("final expression = %q", got)
	}
}

func TestSolveNamesTheQuadraticFormula(t *testing.T) {
	d := derive(t, "solve(x^2 - 4 = 0, x)")
	if len(d.Steps) != 1 {
		t.Fatalf("steps = %+v", d.Steps)
	}
	s := d.Steps[0]
	if s.Expr != "solve(x^2 - 4 = 0, x)" {
		t.Errorf("expr = %q", s.Expr)
	}
	if !strings.HasPrefix(s.Rule, "quadratic formula") {
		t.Errorf("rule = %q", s.Rule)
	}
	if s.Result != "x = -2,  x = 2" {
		t.Errorf("result = %q", s.Result)
	}
}

func TestUnitConversionIsNamed(t *testing.T) {
	d := derive(t, "100 km to m")
	if len(d.Steps) != 2 {
		t.Fatalf("steps = %+v", d.Steps)
	}
	if !strings.Contains(d.Steps[0].Rule, "1 km = 1000 m") {
		t.Errorf("rule = %q", d.Steps[0].Rule)
	}
	if !strings.HasPrefix(d.Steps[1].Rule, "conversion to m") || d.Steps[1].Result != "100000 m" {
		t.Errorf("step = %+v", d.Steps[1])
	}
}

func TestUnprovenRuleIsNotInvented(t *testing.T) {
	d := derive(t, "2^0.5")
	if len(d.Steps) != 1 || d.Steps[0].Rule != "" || d.Steps[0].Source != "" {
		t.Fatalf("steps = %+v", d.Steps)
	}
	if d.Status != "proven" || d.Digits != 12 || d.Bound == "" {
		t.Errorf("status = %q digits = %d bound = %q", d.Status, d.Digits, d.Bound)
	}
}

func TestSingleLiteralHasNoWorking(t *testing.T) {
	d := derive(t, "42")
	if len(d.Steps) != 0 || d.Note == "" || d.Answer != "42" {
		t.Fatalf("derivation = %+v", d)
	}
}

func TestErrorLinePrintsTheError(t *testing.T) {
	d := derive(t, "1 +")
	if d.Error == "" || len(d.Steps) != 0 || d.Answer != "" {
		t.Fatalf("derivation = %+v", d)
	}
}

func TestMinimalParenthesisation(t *testing.T) {
	cases := map[string]string{
		"1+(2*3)":     "1+2*3",
		"(1+2)*3":     "(1+2)*3",
		"1-(2-3)":     "1-(2-3)",
		"(1-2)-3":     "1-2-3",
		"2^3^2":       "2^3^2",
		"(2^3)^2":     "(2^3)^2",
		"-2^2":        "-2^2",
		"(-2)^2":      "(-2)^2",
		"-(1+2)":      "-(1+2)",
		"8/(2*2)":     "8/(2*2)",
		"((1+2))*3+4": "(1+2)*3+4",
	}
	for input, want := range cases {
		d := derive(t, input)
		if len(d.Steps) == 0 {
			t.Errorf("%s: no steps", input)
			continue
		}
		if got := d.Steps[len(d.Steps)-1].Expr; got != want {
			t.Errorf("%s rebuilt as %q, want %q", input, got, want)
		}
	}
}

func TestCyclicChildReferenceTerminates(t *testing.T) {
	line := Line{
		Text:   "1",
		Status: "exact",
		Trace: []prov.NodeOut{
			{ID: 1, Kind: "operation", Label: "+", Value: "1", Children: []prov.NodeID{2, 9}},
			{ID: 2, Kind: "operation", Label: "*", Value: "1", Children: []prov.NodeID{1, 2}},
		},
	}
	d := Derive("x", line)
	if d.Note == "" || len(d.Steps) > maxSteps {
		t.Fatalf("derivation = %+v", d)
	}
	for _, s := range d.Steps {
		if !strings.Contains(s.Expr, elided) {
			t.Errorf("step %d expression %q should show the elided reference", s.N, s.Expr)
		}
	}
}

func TestDeepTraceIsCapped(t *testing.T) {
	var nodes []prov.NodeOut
	const depth = 5000
	for i := 1; i <= depth; i++ {
		n := prov.NodeOut{ID: prov.NodeID(i), Kind: "function", Label: "f", Value: "1"}
		if i < depth {
			n.Children = []prov.NodeID{prov.NodeID(i + 1)}
		}
		nodes = append(nodes, n)
	}
	d := Derive("deep", Line{Text: "1", Status: "exact", Trace: nodes})
	if !d.Truncated || len(d.Steps) > maxSteps {
		t.Fatalf("truncated = %v steps = %d", d.Truncated, len(d.Steps))
	}
}

func TestSharedNodeIsWorkedOnce(t *testing.T) {
	line := Line{
		Text:   "4",
		Status: "exact",
		Trace: []prov.NodeOut{
			{ID: 3, Kind: "operation", Label: "+", Value: "4", Children: []prov.NodeID{2, 2}},
			{ID: 2, Kind: "operation", Label: "+", Value: "2", Children: []prov.NodeID{1, 1}},
			{ID: 1, Kind: "literal", Label: "1", Value: "1"},
		},
	}
	d := Derive("x", line)
	if len(d.Steps) != 2 || d.Note != "" {
		t.Fatalf("derivation = %+v", d)
	}
}

func TestJSONShape(t *testing.T) {
	raw, err := json.Marshal(derive(t, "(1+2)*(3+4)"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["schema"] != Schema || got["input"] != "(1+2)*(3+4)" || got["answer"] != "21" || got["status"] != "exact" {
		t.Errorf("document = %v", got)
	}
	list := got["steps"].([]any)
	if len(list) != 3 {
		t.Fatalf("steps = %v", list)
	}
	last := list[2].(map[string]any)
	if last["n"] != float64(3) || last["expr"] != "(1+2)*(3+4)" || last["result"] != "21" {
		t.Errorf("last step = %v", last)
	}
	if _, has := got["error"]; has {
		t.Errorf("a clean derivation should carry no error")
	}
}

func TestErrorJSONCarriesNoSteps(t *testing.T) {
	raw, err := json.Marshal(derive(t, "1 +"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["error"] == nil || len(got["steps"].([]any)) != 0 {
		t.Errorf("document = %v", got)
	}
}

func TestDecodeRejectsAnEmptySheet(t *testing.T) {
	if _, err := Decode([]byte(`{"lines":[]}`)); err == nil {
		t.Error("an empty sheet should not decode")
	}
	if _, err := Decode([]byte(`not json`)); err == nil {
		t.Error("garbage should not decode")
	}
}

func TestSolveSubstitutesTheQuadraticCoefficients(t *testing.T) {
	rule := derive(t, "solve(x^2 - 4 = 0, x)").Steps[0].Rule
	for _, want := range []string{"a = 1, b = 0, c = -4", "(-0 ± sqrt(0^2 - 4·1·(-4))) / (2·1)", "x = -2,  x = 2", "discriminant 16"} {
		if !strings.Contains(rule, want) {
			t.Errorf("rule %q lacks %q", rule, want)
		}
	}
}

func TestSolveSubstitutesTheLinearCoefficients(t *testing.T) {
	rule := derive(t, "solve(2x + 6 = 0, x)").Steps[0].Rule
	for _, want := range []string{"a = 2, b = 6", "x = -6/2", "x = -3"} {
		if !strings.Contains(rule, want) {
			t.Errorf("rule %q lacks %q", rule, want)
		}
	}
}

func TestSolveBisectionReportsWhatItDid(t *testing.T) {
	rule := derive(t, "solve(cos(x) = x, x)").Steps[0].Rule
	for _, want := range []string{"bisection", "over [-100, 100]", "bisected", "times down to the bracket [0.7390851332151"} {
		if !strings.Contains(rule, want) {
			t.Errorf("rule %q lacks %q", rule, want)
		}
	}
	if strings.Contains(rule, "sqrt") {
		t.Errorf("rule %q invents a closed formula", rule)
	}
}

func TestStepValuesHonourTheRequestedDigits(t *testing.T) {
	d, err := Of("sqrt(2)+1", 30)
	if err != nil {
		t.Fatal(err)
	}
	if got := d.Steps[0].Result; got != "1.41421356237309504880168872421" {
		t.Errorf("sqrt step = %q", got)
	}
}

func TestFromLineWorksFromASheetLine(t *testing.T) {
	sheet := doc.EvalRequest([]byte(`{"lines":["r = 2","pi * r^2"],"digits":12}`))
	var read struct {
		Lines []json.RawMessage `json:"lines"`
	}
	if err := json.Unmarshal(sheet, &read); err != nil {
		t.Fatal(err)
	}
	raw, err := FromLine("pi * r^2", read.Lines[1])
	if err != nil {
		t.Fatal(err)
	}
	var d Derivation
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Steps) == 0 || d.Status != "proven" {
		t.Fatalf("got %+v, want proven working", d)
	}
	if _, err := FromLine("x", []byte("not json")); err == nil {
		t.Error("a malformed line was accepted")
	}
}

func TestEveryStepExplainsItselfInWords(t *testing.T) {
	for input, want := range map[string][]string{
		"(1+2)*(3+4)":   {"Add 1 and 2", "Add 3 and 4", "Multiply 3 by 7"},
		"sqrt(16) + 3!": {"Take the square root of 16: it is 4, because 4 × 4 = 16", "3! = 3 × 2 × 1 = 6", "Add 4 and 6"},
		"18% of 500":    {"18% means 18 ÷ 100", "Take 0.18 of 500, which means 0.18 × 500"},
		"100 km / 2 h":  {"Read 100 km. In SI base units that is 100000 m, because 1 km = 1000 m", "Divide 100 km by 2", "Express 50000 m per h (1 h = 3600 s)"},
		"10 km in mile": {"Read 10 km. In SI base units that is 10000 m, because 1 km = 1000 m", "Convert the result to mile, using 1 mile = 201168/125 m"},
		"mean(1..10)":   {"List the whole numbers from 1 to 10", "Average 1..10: add the values and divide by how many there are"},
		"2^0.5":         {"Raise 2 to the power 1/2, which is its square root"},
		"sin(pi/6)":     {"Use the constant pi = 3.14159265359", "Divide 3.14159265359 by 6", "Take the sine of 0.523598775598"},
	} {
		d := derive(t, input)
		if len(d.Steps) != len(want) {
			t.Errorf("%s: %d steps, want %d: %+v", input, len(d.Steps), len(want), d.Steps)
			continue
		}
		for i, w := range want {
			if d.Steps[i].Explain != w {
				t.Errorf("%s step %d explains %q, want %q", input, i+1, d.Steps[i].Explain, w)
			}
		}
	}
}

func TestTheSummarySaysWhyTheAnswerCanBeTrusted(t *testing.T) {
	for input, want := range map[string]string{
		"1/3 + 1/6":                          "exact arithmetic",
		"2^0.5":                              "every digit shown is certain",
		"(12.4 +- 0.2 cm) * (7.1 +- 0.1 cm)": "standard uncertainty, not a guaranteed bound",
		"10 km in mile":                      "turns the answer back into the unit you asked for",
	} {
		d := derive(t, input)
		if !strings.Contains(strings.Join(d.Summary, " "), want) {
			t.Errorf("%s summary %q does not mention %q", input, d.Summary, want)
		}
	}
}

func TestABareUnitIsNotItsOwnStep(t *testing.T) {
	for _, s := range derive(t, "100 km / 2 h").Steps {
		if strings.Contains(s.Expr, "(…)") {
			t.Errorf("step %d renders a bare unit as a call: %q", s.N, s.Expr)
		}
	}
}

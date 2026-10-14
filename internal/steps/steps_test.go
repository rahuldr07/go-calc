package steps

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
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
		"100 km / 2 h":  {"Read 100 km, where 1 km = 1000 m", "Divide 100 km by 2", "Express 50 km per h (1 h = 3600 s)"},
		"10 km in mile": {"Read 10 km, where 1 km = 1000 m", "Convert the result to mile, using 1 mile = 201168/125 m"},
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
		"10 km in mile":                      "Each step is shown in the units you used",
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

func TestEveryFunctionHasItsOwnExplanation(t *testing.T) {
	for _, input := range []string{
		"abs(-3)", "sqrt(2)", "cbrt(27)", "root(32, 5)", "exp(1)", "ln(2)", "log(100)", "log(8, 2)", "log2(8)", "log10(1000)",
		"sin(1)", "cos(1)", "tan(1)", "asin(1)", "acos(0)", "atan(1)", "atan2(1, 1)", "sinh(1)", "cosh(1)", "tanh(1)",
		"asinh(1)", "acosh(2)", "atanh(0.5)", "sec(1)", "csc(1)", "cot(1)", "deg(1)", "rad(1)",
		"floor(2.7)", "ceil(2.1)", "round(2.5)", "round(2.345, 2)", "trunc(-2.7)", "frac(2.75)", "sign(-4)", "clamp(5, 0, 3)",
		"hypot(3, 4)", "exp2(10)", "gamma(5)", "erf(1)", "erfc(1)", "gcd(12, 18)", "lcm(4, 6)", "isprime(97)", "nCr(5, 2)",
		"nPr(5, 2)", "fib(10)", "mod(17, 5)", "modpow(2, 10, 1000)", "modinv(3, 11)", "factor(360)", "divisors(12)",
		"totient(36)", "nextprime(100)", "prevprime(100)", "primes(20)", "primes(10, 20)", "primes(1..20)", "digitsum(123)",
		"tobase(255, 16)", "hex(255)", "bin(5)", "oct(8)", "sum(1..5)", "product(1..5)", "mean([2, 4, 9])", "median([3, 1, 2])",
		"min(4, 2, 8)", "max(4, 2, 8)", "count([1, 2])", "len([1, 2])", "stdev([1, 2, 3])", "stdevp([1, 2, 3])",
		"variance([1, 2, 3])", "variancep([1, 2, 3])", "sort([3, 1, 2])", "reverse([1, 2])", "unique([1, 1, 2])",
		"mode([1, 2, 2])", "percentile([10, 20, 30], 50)", "cumsum([1, 2])", "diff([1, 4])", "[10, 20][1]", "1..10..2",
		"deriv(x^2, x)", "deriv(x^3, x, 2)", "solve(2x + 4 = 0, x)", "solve(cos(x) = x, x)", "expand(2 +- 0.1, k=2)",
		"7 mod 3", "5!", "50% of 80", "200 + 10%", "100 - 25%", "3 < 5", "2^10", "(1+2i)*(3+4i)", "100 degF in degC", "2 lakh in crore",
	} {
		d := derive(t, input)
		if d.Error != "" || len(d.Steps) == 0 {
			t.Errorf("%s: no working (%s %s)", input, d.Error, d.Note)
			continue
		}
		for _, s := range d.Steps {
			if s.Explain == "" || strings.HasPrefix(s.Explain, "Apply ") || strings.HasPrefix(s.Explain, "Combine ") {
				t.Errorf("%s step %d has no specific explanation: %q", input, s.N, s.Explain)
			}
		}
	}
}

func TestExplanationsThatOnceMisledStayCorrect(t *testing.T) {
	for input, want := range map[string]string{
		"200 + 10%":            "Add 10% of 200 to 200, which means 200 × (1 + 0.1)",
		"100 - 25%":            "Take 25% off 100, which means 100 × (1 - 0.25)",
		"(1+2i)*(3+4i)":        "Multiply 1+2i by 3+4i",
		"sqrt(-4)":             "a negative number has no real square root",
		"log(100)":             "Take the natural logarithm of 100",
		"log(8, 2)":            "Take the base-2 logarithm of 8",
		"round(pi, 3)":         "Round 3.14159265359 to 3 decimal places",
		"round(1234, -2)":      "Round 1234 to the nearest 100",
		"100 degF in degC":     "where K = degF × 5/9 + 45967/180",
		"2 lakh in crore":      "Read 2 lakh: since 1 lakh = 100000, that is 200000",
		"1/3 + 1/6":            "Add 1/3 and 1/6",
		"deriv(x^3, x, 2)":     "Differentiate x^3 with respect to x: the derivative is 3 * x^2, then evaluate it at x = 2",
		"300 m + 1 km":         "Read 300 m, already in SI base units",
		"solve(2x + 4 = 0, x)": "Solve the equation for x by rearranging the linear equation exactly",
	} {
		var said []string
		for _, s := range derive(t, input).Steps {
			said = append(said, s.Explain)
		}
		if !strings.Contains(strings.Join(said, " | "), want) {
			t.Errorf("%s explains itself as %q, want it to contain %q", input, said, want)
		}
	}
	if r := derive(t, "3 < 5").Steps[0].Result; r != "true" {
		t.Errorf("a comparison step shows %q, want true", r)
	}
	if r := derive(t, "(2 +- 0.1) * 3").Steps[0].Result; strings.Contains(r, "± 0") {
		t.Errorf("a measured step claims no uncertainty: %q", r)
	}
	for _, s := range derive(t, "-5 + 2").Steps {
		if strings.HasPrefix(s.Explain, "Negate") {
			t.Errorf("a negative number became its own step: %q", s.Explain)
		}
	}
}

func TestEveryEngineFunctionIsExplained(t *testing.T) {
	for _, name := range eval.Functions() {
		explained := false
		for _, args := range [][]string{{"a"}, {"a", "b"}, {"a", "b", "c"}} {
			text := functionExplanation(prov.NodeOut{Label: name, Value: "v", Detail: "d/dx a = b"}, args)
			if !strings.HasPrefix(text, "Apply ") {
				explained = true
				break
			}
		}
		if !explained {
			t.Errorf("%s has no explanation; add one to explain.go", name)
		}
	}
}

func TestStepsShowTheArithmeticInsideThem(t *testing.T) {
	for input, want := range map[string][]string{
		"mean([2, 4, 9])":                  {"(2 + 4 + 9) ÷ 3 = 15 ÷ 3 = 5"},
		"gcd(48, 18)":                      {"48 = 2 × 18 + 12", "18 = 1 × 12 + 6", "12 = 2 × 6 + 0", "the last non-zero remainder is 6"},
		"hex(255)":                         {"255 ÷ 16 = 15 remainder 15 (f)", "15 ÷ 16 = 0 remainder 15 (f)", "read the remainders from last to first: ff"},
		"nCr(5, 2)":                        {"5! ÷ (2! × 3!) = 120 ÷ (2 × 6) = 10"},
		"10 / 4":                           {"10 ÷ 4 = 5/2 = 2.5"},
		"2^5":                              {"2^5 = 2 × 2 × 2 × 2 × 2 = 32"},
		"-7 mod 3":                         {"-7 = (-3) × 3 + 2, so the remainder is 2"},
		"hypot(3, 4)":                      {"√(3² + 4²) = √(9 + 16) = √25 = 5"},
		"percentile([10, 20, 30, 40], 50)": {"sorted: 10, 20, 30, 40", "position = 50% × (4 - 1) = 1.5, counting from 0", "between 20 and 30: 20 + 0.5 × (30 - 20) = 25"},
		"median(1..6)":                     {"sorted: 1, 2, 3, 4, 5, 6", "an even count has two middle values: (3 + 4) ÷ 2 = 3.5"},
		"totient(36)":                      {"the prime factors of 36 are 2 and 3", "36 × (1 - 1/2) × (1 - 1/3) = 12"},
		"isprime(91)":                      {"91 = 7 × 13, so it is not prime"},
		"stdev([1, 2, 3, 4])":              {"mean = 10 ÷ 4 = 2.5", "squared distances from the mean: (1 - 2.5)² + (2 - 2.5)² + (3 - 2.5)² + (4 - 2.5)² = 5", "divide by n - 1 = 3: 5 ÷ 3 = 5/3", "take the square root: √(5/3) = 1.29099444874"},
		"solve(x^2 - 5x + 6 = 0, x)":       {"write it as a·x² + b·x + c = 0 with a = 1, b = -5, c = 6", "discriminant b² - 4ac = (-5)² - 4·1·6 = 1", "x = (-b ± √discriminant) ÷ 2a = (5 ± √1) ÷ 2, giving x = 2 or x = 3"},
		"200 + 10%":                        {"200 × (1 + 0.1) = 200 × 1.1 = 220"},
	} {
		var got []string
		for _, s := range derive(t, input).Steps {
			got = append(got, s.Work...)
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s works as\n%s\nwant\n%s", input, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
}

func TestApproximateValuesAreNeverWorkedAsFractions(t *testing.T) {
	for _, input := range []string{"pi / 2", "sqrt(2) / 2", "pi * 2", "e^2"} {
		for _, s := range derive(t, input).Steps {
			for _, line := range s.Work {
				if strings.Contains(line, "/") && strings.Contains(line, "0000") {
					t.Errorf("%s works an approximate value as a fraction: %q", input, line)
				}
			}
		}
	}
}

func TestStepsStayInTheUnitsThatWereTyped(t *testing.T) {
	for input, want := range map[string][]string{
		"6.8 L/100km * 412 km in L": {"6.8 L", "0.068 L", "0.068 L/km", "28.016 L/km", "28.016 L", "28.016 L"},
		"60 km/h * 30 min":          {"60 km", "60 km/h", "1800 km/h", "30 km"},
		"100 km / 2 h":              {"100 km", "50 km", "50 km/h"},
		"g0 * 2 s":                  {"9.80665 m/s^2", "19.6133 m/s^2", "19.6133 m/s"},
	} {
		var got []string
		for _, s := range derive(t, input).Steps {
			got = append(got, s.Result)
		}
		if strings.Join(got, " | ") != strings.Join(want, " | ") {
			t.Errorf("%s shows %q, want %q", input, got, want)
		}
	}
}

package doc

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func TestEvalRequestShape(t *testing.T) {
	req := []byte(`{"lines":["0.1 + 0.2","5 m + 3 s","sqrt(2)"],"digits":12}`)
	var got map[string]any
	if err := json.Unmarshal(EvalRequest(req), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	lines := got["lines"].([]any)
	if len(lines) != 3 {
		t.Fatalf("got %d lines", len(lines))
	}
	first := lines[0].(map[string]any)
	if first["text"] != "0.3" || first["status"] != "exact" {
		t.Errorf("line 0 = %+v", first)
	}
	if lines[1].(map[string]any)["status"] != "error" {
		t.Errorf("line 1 should be an error")
	}
	if lines[2].(map[string]any)["status"] != "proven" {
		t.Errorf("line 2 should be proven")
	}
}

func FuzzEvalRequest(f *testing.F) {
	f.Add(`{"lines":["1+1"]}`)
	f.Add(`{"lines":[""]}`)
	f.Add(`{"lines":["h = 12.3 ± 0.2 m","w = 4.5 ± 0.1 m","h * w"],"digits":12,"angle":"deg"}`)
	f.Add(`{"lines":["a = b","b = a"]}`)
	f.Add(`{"lines":["f(x) = 2x","f(3)"],"angle":"rad"}`)
	f.Add(`{"lines":["1/0"],"digits":-4}`)
	f.Fuzz(func(t *testing.T, s string) {
		out := EvalRequest([]byte(s))
		var back wireSheet
		if err := json.Unmarshal(out, &back); err != nil {
			t.Fatalf("EvalRequest(%q) returned invalid JSON %q: %v", s, out, err)
		}
		if back.Schema != SheetSchema {
			t.Fatalf("EvalRequest(%q) answered under schema %q", s, back.Schema)
		}
		for i, l := range back.Lines {
			if l.N != i {
				t.Fatalf("EvalRequest(%q) line %d is numbered %d", s, i, l.N)
			}
			if !known[l.Status] {
				t.Fatalf("EvalRequest(%q) line %d has status %q", s, i, l.Status)
			}
		}
	})
}

func FuzzEvalRequestLine(f *testing.F) {
	for _, seed := range []string{
		"1+1", "", "sqrt(2)", "x = 1 ± 0.1", "f(x) = 2x", "60 km/h to m/s",
		"1/2 x", "corr(a, b) = 0.5", "# note", "2^10000000000", "sin(30)",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, line string) {
		req, err := json.Marshal(sheetRequest{Lines: []string{line}, Digits: 12})
		if err != nil {
			t.Skip()
		}
		out := EvalRequest(req)
		var back wireSheet
		if err := json.Unmarshal(out, &back); err != nil {
			t.Fatalf("line %q returned invalid JSON %q: %v", line, out, err)
		}
		if back.Schema != SheetSchema {
			t.Fatalf("line %q answered under schema %q", line, back.Schema)
		}
		if len(back.Lines) != 1 {
			t.Fatalf("line %q came back as %d lines", line, len(back.Lines))
		}
		l := back.Lines[0]
		if !known[l.Status] {
			t.Fatalf("line %q has status %q", line, l.Status)
		}
		if (l.Status == StatusError) != (l.Error != "") {
			t.Fatalf("line %q is %s and says %q", line, l.Status, l.Error)
		}
		if l.Status != StatusEstimate && strings.Contains(l.Text, "±") {
			t.Fatalf("line %q is %s yet printed a ± in %q", line, l.Status, l.Text)
		}
	})
}

var known = map[string]bool{
	StatusExact:    true,
	StatusProven:   true,
	StatusEstimate: true,
	StatusHeld:     true,
	StatusError:    true,
}

type wireNote struct {
	Source string `json:"source"`
	Text   string `json:"text"`
	Column int    `json:"column"`
}

type wireShare struct {
	Name       string  `json:"name"`
	Share      float64 `json:"share"`
	Covariance bool    `json:"covariance"`
}

type wireUncertainty struct {
	Value      string      `json:"value"`
	Reported   string      `json:"reported"`
	Standard   *float64    `json:"standard"`
	Degenerate bool        `json:"degenerate"`
	Budget     []wireShare `json:"budget"`
	Warning    string      `json:"warning"`
}

type wireNode struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type wireLine struct {
	N           int              `json:"n"`
	Text        string           `json:"text"`
	Status      string           `json:"status"`
	Digits      int              `json:"digits"`
	Capped      bool             `json:"capped"`
	Stopped     string           `json:"stopped"`
	Error       string           `json:"error"`
	Column      int              `json:"column"`
	Uncertainty *wireUncertainty `json:"uncertainty"`
	Notes       []wireNote       `json:"notes"`
	Warning     string           `json:"warning"`
	Trace       []wireNode       `json:"trace"`
}

type wireSheet struct {
	Schema string     `json:"schema"`
	Lines  []wireLine `json:"lines"`
}

func ask(t *testing.T, req string) wireSheet {
	t.Helper()
	out := EvalRequest([]byte(req))
	var got wireSheet
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("EvalRequest(%s) returned invalid JSON %q: %v", req, out, err)
	}
	if got.Schema != SheetSchema {
		t.Fatalf("EvalRequest(%s) answered under schema %q", req, got.Schema)
	}
	return got
}

func document(t *testing.T, lines ...string) wireSheet {
	t.Helper()
	req, err := json.Marshal(sheetRequest{Lines: lines})
	if err != nil {
		t.Fatalf("could not build the request: %v", err)
	}
	return ask(t, string(req))
}

func TestEveryLineOfTheDocumentComesBackInOrder(t *testing.T) {
	for _, src := range [][]string{
		{},
		{""},
		{"1+1"},
		{"# a comment", "", "x = 2", "x^3", "x + "},
		{"a = 1", "b = a + 1", "c = b + 1", "d = c + 1", "e = d + 1"},
	} {
		got := document(t, src...)
		if len(got.Lines) != len(src) {
			t.Fatalf("%v: got %d lines back, wanted %d", src, len(got.Lines), len(src))
		}
		for i, l := range got.Lines {
			if l.N != i {
				t.Errorf("%v: line at index %d is numbered %d", src, i, l.N)
			}
		}
	}
}

func TestStatusNamesWhatTheAnswerIs(t *testing.T) {
	for _, c := range []struct {
		src    []string
		want   string
		atLine int
	}{
		{src: []string{"0.1 + 0.2"}, want: StatusExact},
		{src: []string{"7"}, want: StatusExact},
		{src: []string{"sqrt(2)"}, want: StatusProven},
		{src: []string{"pi"}, want: StatusProven},
		{src: []string{"h = 12.3 ± 0.2 m"}, want: StatusEstimate},
		{src: []string{"h = 12.3 ± 0.2 m", "w = 4.5 ± 0.1 m", "h * w"}, want: StatusEstimate, atLine: 2},
		{src: []string{"f(x) = 2x"}, want: StatusHeld},
		{src: []string{""}, want: StatusHeld},
		{src: []string{"# just a note"}, want: StatusHeld},
		{src: []string{"5 m + 3 s"}, want: StatusError},
		{src: []string{"1 +"}, want: StatusError},
	} {
		got := document(t, c.src...)
		if s := got.Lines[c.atLine].Status; s != c.want {
			t.Errorf("%v line %d: status %q, wanted %q (%+v)", c.src, c.atLine, s, c.want, got.Lines[c.atLine])
		}
	}
}

func TestAnErrorLineCarriesTheMessageAndNoAnswer(t *testing.T) {
	got := document(t, "5 m + 3 s")
	line := got.Lines[0]
	if line.Error == "" {
		t.Fatalf("an error line said nothing: %+v", line)
	}
	if !strings.Contains(line.Error, "length") || !strings.Contains(line.Error, "time") {
		t.Errorf("the message does not name the dimensions: %q", line.Error)
	}
	if line.Text != "" {
		t.Errorf("an error line still offered the answer %q", line.Text)
	}
	if line.Digits != 0 {
		t.Errorf("an error line claimed %d digits", line.Digits)
	}
}

func TestAParseErrorPointsAtTheColumn(t *testing.T) {
	got := document(t, "1 + )")
	line := got.Lines[0]
	if line.Status != StatusError {
		t.Fatalf("wanted an error, got %+v", line)
	}
	if line.Column <= 0 {
		t.Errorf("the error points at no column: %+v", line)
	}
}

func TestADependentOfAFailedLineSaysWhichLineFailed(t *testing.T) {
	got := document(t, "x = 1 m + 1 s", "x + 1")
	if got.Lines[1].Status != StatusError {
		t.Fatalf("the dependent line did not fail: %+v", got.Lines[1])
	}
	if !strings.Contains(got.Lines[1].Error, "line 1") {
		t.Errorf("the dependent line does not name line 1: %q", got.Lines[1].Error)
	}
}

func TestAForwardReferenceNamesTheSymbol(t *testing.T) {
	got := document(t, "y + 1", "y = 2")
	if got.Lines[0].Status != StatusError {
		t.Fatalf("a forward reference was allowed: %+v", got.Lines[0])
	}
	if !strings.Contains(got.Lines[0].Error, "y") {
		t.Errorf("the message does not name y: %q", got.Lines[0].Error)
	}
}

func TestACycleIsReportedOnEveryLineInIt(t *testing.T) {
	got := document(t, "a = b + 1", "b = a + 1")
	for i, l := range got.Lines {
		if l.Status != StatusError {
			t.Errorf("line %d of a cycle answered %+v", i, l)
		}
	}
}

func TestDigitsFollowTheRequest(t *testing.T) {
	for _, digits := range []int{4, 12, 30} {
		got := ask(t, `{"lines":["sqrt(2)"],"digits":`+itoa(digits)+`}`)
		line := got.Lines[0]
		if line.Digits != digits {
			t.Errorf("digits %d: the sheet claims %d", digits, line.Digits)
		}
		if n := significant(line.Text); n != digits {
			t.Errorf("digits %d: %q shows %d significant digits", digits, line.Text, n)
		}
	}
}

func TestAnAbsurdDigitRequestIsClamped(t *testing.T) {
	got := ask(t, `{"lines":["sqrt(2)"],"digits":100000000}`)
	line := got.Lines[0]
	if line.Digits > maxDigits {
		t.Fatalf("the sheet claims %d digits", line.Digits)
	}
	if n := significant(line.Text); n > maxDigits {
		t.Fatalf("the answer shows %d significant digits", n)
	}
}

func TestDigitsBelowOneFallBackToTheDefault(t *testing.T) {
	got := ask(t, `{"lines":["sqrt(2)"],"digits":-3}`)
	if got.Lines[0].Digits != eval.DefaultCtx().Digits {
		t.Errorf("a negative digit count gave %d digits", got.Lines[0].Digits)
	}
}

func TestTheAngleModeChangesTheAnswer(t *testing.T) {
	deg := ask(t, `{"lines":["sin(30)"],"angle":"deg"}`)
	rad := ask(t, `{"lines":["sin(30)"],"angle":"rad"}`)
	if !strings.HasPrefix(deg.Lines[0].Text, "0.5") {
		t.Errorf("sin(30) in degrees answered %q", deg.Lines[0].Text)
	}
	if deg.Lines[0].Text == rad.Lines[0].Text {
		t.Errorf("degrees and radians answered the same: %q", rad.Lines[0].Text)
	}
	if plain := ask(t, `{"lines":["sin(30)"]}`); plain.Lines[0].Text != rad.Lines[0].Text {
		t.Errorf("no angle mode answered %q, radians answered %q", plain.Lines[0].Text, rad.Lines[0].Text)
	}
}

func TestTheWarningIsMeasuredInTheAngleModeItWasAskedFor(t *testing.T) {
	got := ask(t, `{"lines":["sin(30)"],"angle":"deg"}`)
	if got.Lines[0].Warning != "" {
		t.Errorf("degrees mode warned about a radian shadow: %q", got.Lines[0].Warning)
	}
}

func TestAnUnknownAngleModeIsRefused(t *testing.T) {
	got := ask(t, `{"lines":["sin(30)"],"angle":"grad"}`)
	if len(got.Lines) != 1 || got.Lines[0].Status != StatusError {
		t.Fatalf("grad was accepted: %+v", got.Lines)
	}
	if !strings.Contains(got.Lines[0].Error, "grad") {
		t.Errorf("the refusal does not name grad: %q", got.Lines[0].Error)
	}
}

func TestNotesSayWhetherTheParserOrTheEvaluatorSpoke(t *testing.T) {
	got := document(t, "m = 3", "1/2 m")
	notes := got.Lines[1].Notes
	if len(notes) < 2 {
		t.Fatalf("wanted a reading from each of the parser and the evaluator, got %+v", notes)
	}
	sources := map[string]string{}
	for _, n := range notes {
		if n.Text == "" {
			t.Errorf("a note said nothing: %+v", n)
		}
		sources[n.Source] = n.Text
	}
	if _, ok := sources[FromParser]; !ok {
		t.Errorf("no note came from the parser: %+v", notes)
	}
	if _, ok := sources[FromEvaluator]; !ok {
		t.Errorf("no note came from the evaluator: %+v", notes)
	}
}

func TestOnlyTheParserSpeaksWhenNothingWasAmbiguous(t *testing.T) {
	got := document(t, "1/2 x", "x = 1")
	for _, n := range got.Lines[0].Notes {
		if n.Source != FromParser {
			t.Errorf("a plain reading was credited to %q: %+v", n.Source, n)
		}
	}
}

func TestTheTraceCarriesTheProvenanceRootFirst(t *testing.T) {
	got := document(t, "2 * 3")
	trace := got.Lines[0].Trace
	if len(trace) < 3 {
		t.Fatalf("the trace holds %d nodes: %+v", len(trace), trace)
	}
	if trace[0].Kind != "operation" || trace[0].Label != "*" {
		t.Errorf("the root of the trace is %+v", trace[0])
	}
	seen := map[string]bool{}
	for _, n := range trace {
		seen[n.Label] = true
	}
	if !seen["2"] || !seen["3"] {
		t.Errorf("the trace lost an operand: %+v", trace)
	}
}

func TestALineWithNoProvenanceCarriesNoTrace(t *testing.T) {
	got := document(t, "f(x) = 2x")
	if len(got.Lines[0].Trace) != 0 {
		t.Errorf("a definition carried a trace: %+v", got.Lines[0].Trace)
	}
}

func TestAMeasuredLineCarriesItsBudget(t *testing.T) {
	got := document(t, "h = 12.3 ± 0.2 m", "w = 4.5 ± 0.1 m", "h * w")
	u := got.Lines[2].Uncertainty
	if u == nil {
		t.Fatalf("the area reported no uncertainty: %+v", got.Lines[2])
	}
	if u.Standard == nil || *u.Standard <= 0 {
		t.Errorf("the area reported no standard uncertainty: %+v", u)
	}
	names := map[string]bool{}
	total := 0.0
	for _, c := range u.Budget {
		names[c.Name] = true
		total += c.Share
	}
	if !names["h"] || !names["w"] {
		t.Errorf("the budget does not name both inputs: %+v", u.Budget)
	}
	if total < 0.99 || total > 1.01 {
		t.Errorf("the shares add up to %v", total)
	}
}

func TestADeclaredCorrelationShowsUpAsACovarianceTerm(t *testing.T) {
	got := document(t, "a = 10 ± 1", "b = 10 ± 1", "corr(a, b) = 0.9", "a + b")
	u := got.Lines[3].Uncertainty
	if u == nil {
		t.Fatalf("the sum reported no uncertainty: %+v", got.Lines[3])
	}
	found := false
	for _, c := range u.Budget {
		found = found || c.Covariance
	}
	if !found {
		t.Errorf("the budget hides the covariance term: %+v", u.Budget)
	}
}

func TestOnlyAnEstimateEverPrintsAPlusMinus(t *testing.T) {
	for _, src := range [][]string{
		{"sqrt(2)"},
		{"1/3"},
		{"pi * 1e-40"},
		{"0.1 + 0.2"},
		{"f(x) = 2x"},
		{"h = 12.3 ± 0.2 m", "w = 4.5 ± 0.1 m", "h * w"},
		{"x = 1 ± 0.1", "x - x"},
		{"5 m + 3 s"},
		{"exp(sqrt(2)) - exp(sqrt(2))"},
	} {
		for _, l := range document(t, src...).Lines {
			if strings.Contains(l.Text, "±") && l.Status != StatusEstimate {
				t.Errorf("%v: a %s line printed a ± in %q", src, l.Status, l.Text)
			}
		}
	}
}

func TestAnAnswerWithNothingProvenIsAnIntervalAndNotAPlusMinus(t *testing.T) {
	for _, src := range [][]string{
		{"1 + sin(pi) * 1e3000"},
		{"x = 1 ± 0.1", "1 + (exp(x) - exp(x)) * 1e40"},
	} {
		line := last(document(t, src...))
		if line.Digits != 0 {
			t.Fatalf("%v proved %d digits: %+v", src, line.Digits, line)
		}
		if !strings.HasPrefix(line.Text, "[") || !strings.HasSuffix(line.Text, "]") {
			t.Errorf("%v proved nothing yet answered %q instead of an interval", src, line.Text)
		}
		if strings.Contains(line.Text, "±") {
			t.Errorf("%v offered a ± for a guaranteed bound: %q", src, line.Text)
		}
	}
}

func TestAStoppedLineSaysWhyItStopped(t *testing.T) {
	for _, c := range []struct {
		src  []string
		want string
	}{
		{src: []string{"sin(pi)"}, want: "precision cap"},
		{src: []string{"x = 1 ± 0.1", "exp(x) - exp(x)"}, want: "uncertain input"},
	} {
		line := last(document(t, c.src...))
		if !line.Capped {
			t.Errorf("%v stopped short but was not marked capped: %+v", c.src, line)
		}
		if line.Stopped != c.want {
			t.Errorf("%v stopped for %q, wanted %q", c.src, line.Stopped, c.want)
		}
	}
}

func TestALineThatRanToTheEndSaysNothingAboutStopping(t *testing.T) {
	line := last(document(t, "sqrt(2)"))
	if line.Capped || line.Stopped != "" {
		t.Errorf("a full answer claimed it stopped: %+v", line)
	}
}

func last(got wireSheet) wireLine { return got.Lines[len(got.Lines)-1] }

func TestOneLineDoesNotSpendAnotherLinesPrecision(t *testing.T) {
	const tail = "1 + (exp(x) - exp(x)) * 1e40"
	alone := last(document(t, "x = 1 ± 0.1", tail))
	after := last(document(t, "1 + sin(pi) * 1e3000", "x = 1 ± 0.1", tail))
	if alone.Text != after.Text {
		t.Errorf("%s answered %q alone and %q after a costly line", tail, alone.Text, after.Text)
	}
	if alone.Digits != after.Digits {
		t.Errorf("%s proved %d digits alone and %d after a costly line", tail, alone.Digits, after.Digits)
	}
}

func TestAHugeFractionalPowerStillAnswers(t *testing.T) {
	line := last(document(t, "2^1000000.5"))
	if line.Status != StatusProven {
		t.Fatalf("2^1000000.5 answered %+v", line)
	}
	if !strings.Contains(line.Text, "e+301030") {
		t.Errorf("2^1000000.5 answered %q", line.Text)
	}
}

func TestTheSameRequestAnswersTheSameWay(t *testing.T) {
	req := []byte(`{"lines":["h = 12.3 ± 0.2 m","w = 4.5 ± 0.1 m","h * w","sqrt(2)"],"digits":9}`)
	first := EvalRequest(req)
	if second := EvalRequest(req); !bytes.Equal(first, second) {
		t.Errorf("the same request answered\n%s\nand then\n%s", first, second)
	}
}

func TestAnEmptyDocumentIsAnEmptyListOfLines(t *testing.T) {
	out := EvalRequest([]byte(`{"lines":[]}`))
	if !bytes.Contains(out, []byte(`"lines":[]`)) {
		t.Errorf("an empty document answered %s", out)
	}
}

func TestARequestThatIsNotJSONBecomesOneErrorLine(t *testing.T) {
	for _, req := range []string{"", "not json", "[]", `{"lines":3}`, `{"lines":["1+1"`} {
		got := ask(t, req)
		if len(got.Lines) != 1 || got.Lines[0].Status != StatusError {
			t.Errorf("%q answered %+v", req, got.Lines)
		}
		if len(got.Lines) == 1 && got.Lines[0].Error == "" {
			t.Errorf("%q gave an error line with no message", req)
		}
	}
}

func TestAMissingLineListIsAnEmptyDocument(t *testing.T) {
	got := ask(t, `{"digits":12}`)
	if len(got.Lines) != 0 {
		t.Errorf("a request with no lines answered %+v", got.Lines)
	}
}

func TestAPanicBecomesAnErrorLineInsteadOfKillingTheModule(t *testing.T) {
	out := evalSheet([]byte(`{"lines":["1+1"]}`), func(ctx *eval.Ctx) *Doc {
		d := documentUnder(ctx)
		d.evaluator = func(*eval.Env, parse.Node) (value.Value, error) { panic("the arena caught fire") }
		return d
	})
	var got wireSheet
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("a panic returned invalid JSON %q: %v", out, err)
	}
	if got.Schema != SheetSchema {
		t.Fatalf("a panic answered under schema %q", got.Schema)
	}
	if len(got.Lines) != 1 || got.Lines[0].Status != StatusError {
		t.Fatalf("a panic answered %+v", got.Lines)
	}
	if !strings.Contains(got.Lines[0].Error, "the arena caught fire") {
		t.Errorf("the error line hides what happened: %q", got.Lines[0].Error)
	}
}

func TestAPanicWithAnErrorValueIsReportedToo(t *testing.T) {
	out := evalSheet([]byte(`{"lines":["1+1"]}`), func(ctx *eval.Ctx) *Doc {
		d := documentUnder(ctx)
		d.warner = func(*eval.Env, parse.Node, value.Value) string { panic(errCycle) }
		return d
	})
	var got wireSheet
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("a panic returned invalid JSON %q: %v", out, err)
	}
	if len(got.Lines) != 1 || !strings.Contains(got.Lines[0].Error, errCycle.Error()) {
		t.Errorf("a panicking warner answered %+v", got.Lines)
	}
}

func significant(text string) int {
	seen, started := 0, false
	for _, r := range text {
		if r == 'e' || r == 'E' {
			break
		}
		if r < '0' || r > '9' {
			continue
		}
		if r != '0' {
			started = true
		}
		if started {
			seen++
		}
	}
	return seen
}

func itoa(n int) string {
	out, err := json.Marshal(n)
	if err != nil {
		return "0"
	}
	return string(out)
}

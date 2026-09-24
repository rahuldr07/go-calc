package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
)

var plain = format.Style{Digits: format.Digits}

func TestRunOneShot(t *testing.T) {
	out, code := run([]string{"500 + 10%"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if strings.TrimSpace(out) != "550" {
		t.Errorf("out = %q, want 550", out)
	}
}

func TestRunShowsInterpretation(t *testing.T) {
	out, code := run([]string{"1/2m"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "read as 1/(2·m)") {
		t.Errorf("out = %q, want the interpretation note", out)
	}
}

func TestRunJSON(t *testing.T) {
	out, code := run([]string{"3", "km", "to", "m"}, plain, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, `"schema": "go-calc.result/1"`) {
		t.Errorf("out = %q", out)
	}
}

func TestRunErrorExitsNonZero(t *testing.T) {
	out, code := run([]string{"5 m + 3 s"}, plain, false)
	if code == 0 {
		t.Fatal("want non-zero exit")
	}
	if !strings.Contains(out, "cannot add length (m) and time (s)") {
		t.Errorf("out = %q", out)
	}
}

func TestRunFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "# lab\nh = 3 m\nh + 200 cm\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "= 5 m") {
		t.Errorf("out = %q, want 5 m", out)
	}
}

func TestUsageWithoutArguments(t *testing.T) {
	out, code := run(nil, plain, false)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(out, "usage:") {
		t.Errorf("out = %q", out)
	}
}

func TestRunFileRejectsAForwardReference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "total = price * 2\nprice = 5\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 1: uses price before it is defined") {
		t.Errorf("out = %q, want the forward-reference error on line 1", out)
	}
	if strings.Contains(out, "= 10") {
		t.Errorf("out = %q, want no answer for the forward reference", out)
	}
}

func TestRunFileReportsACycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "a = b + 1\nb = a + 1\nz = 7\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 1: this line depends on itself") {
		t.Errorf("out = %q, want the cycle reported on line 1", out)
	}
	if !strings.Contains(out, "= 7") {
		t.Errorf("out = %q, want the independent line still evaluated", out)
	}
}

func TestRunFileExplainsADependentOfAFailedLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	body := "bad = 5 m + 3 s\nuses = bad + 1\nfine = 1 + 1\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain)
	if code == 0 {
		t.Fatalf("want non-zero exit, got %q", out)
	}
	if !strings.Contains(out, "line 2: uses bad, which failed on line 1") {
		t.Errorf("out = %q, want the dependent to name the line it depends on", out)
	}
	if strings.Contains(out, "unknown name bad") {
		t.Errorf("out = %q, leaked a raw evaluator error", out)
	}
	if !strings.Contains(out, "= 2") {
		t.Errorf("out = %q, want the independent line still evaluated", out)
	}
}

func TestSheetShowsWhichReadingItTook(t *testing.T) {
	out, code := runFile(writeSheet(t, "h = 12.3 ± 0.2 m\n2 h\n"), plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := "read as the variable h, not the unit h (time); the variable carries a measured uncertainty"
	if !strings.Contains(out, want) {
		t.Errorf("out = %q, want %q", out, want)
	}
	out, code = runFile(writeSheet(t, "h = 3\n5 h\n"), plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "read as the unit h (time), not the variable h") {
		t.Errorf("out = %q, want the unit reading explained", out)
	}
}

func TestEvalLineCarriesTheEvaluatorNote(t *testing.T) {
	env := eval.NewEnv()
	node, _, err := parse.Parse("h = 3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(node); err != nil {
		t.Fatal(err)
	}
	out, code := evalLine(env, "5 h", plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "read as the unit h (time), not the variable h") {
		t.Errorf("out = %q, want the reading note", out)
	}
}

func writeSheet(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sheet.calc")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTracePrintsTheTree(t *testing.T) {
	out, code := explain("sqrt(2 m / g0)", false, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{"sqrt", "function", "g0", "CODATA 2022", "m", "unit", "└──"} {
		if !strings.Contains(out, want) {
			t.Errorf("trace missing %q:\n%s", want, out)
		}
	}
}

func TestWhyNamesBothAnswers(t *testing.T) {
	out, code := explain("(1e16 + 1) - 1e16", true, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{"proven:", "float64:", "bits at the -"} {
		if !strings.Contains(out, want) {
			t.Errorf("why missing %q:\n%s", want, out)
		}
	}
}

func TestTraceReportsAFailedExpression(t *testing.T) {
	if out, code := explain("5 m + 3 s", false, plain); code == 0 {
		t.Errorf("want non-zero exit, got %q", out)
	}
	if out, code := explain("5 m + 3 s", true, plain); code == 0 {
		t.Errorf("want non-zero exit, got %q", out)
	}
}

func TestOneShotCarriesThePrecisionWarning(t *testing.T) {
	out, code := run([]string{"(1e16 + 1) - 1e16"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "float64 would answer 0") {
		t.Errorf("out = %q, want the float64 warning", out)
	}
	quiet, code := run([]string{"2 + 3 * 4"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, quiet)
	}
	if strings.Contains(quiet, "float64") {
		t.Errorf("out = %q, want no warning on benign arithmetic", quiet)
	}
}

func TestJSONCarriesProvenanceAndWarning(t *testing.T) {
	out, code := run([]string{"(1e16 + 1) - 1e16"}, plain, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{`"provenance"`, `"root"`, `"nodes"`, `"warning"`, `"kind": "literal"`} {
		if !strings.Contains(out, want) {
			t.Errorf("json missing %q:\n%s", want, out)
		}
	}
	quiet, code := run([]string{"2 + 2"}, plain, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, quiet)
	}
	if strings.Contains(quiet, `"warning"`) {
		t.Errorf("json = %s, want no warning key", quiet)
	}
	if !strings.Contains(quiet, `"provenance"`) {
		t.Errorf("json = %s, want provenance on every result", quiet)
	}
}

func TestSheetWarnsOnTheLineThatLosesDigits(t *testing.T) {
	out, code := runFile(writeSheet(t, "fine = 2 + 2\nlost = (1e16 + 1) - 1e16\n"), plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "at 2:") {
		t.Errorf("out = %q, want the warning to name sheet line 2", out)
	}
	if strings.Count(out, "float64") != 1 {
		t.Errorf("out = %q, want exactly one warning", out)
	}
}

func TestSheetTraceReachesAcrossLines(t *testing.T) {
	sheet := doc.New()
	lines := sheet.SetLines([]string{"h = 12.3 ± 0.2 m", "t = sqrt(2h / g0)"})
	tree := sheet.Arena().Tree(lines[1].Value.Prov)
	for _, want := range []string{"h", "input", "line 1", "g0", "CODATA 2022"} {
		if !strings.Contains(tree, want) {
			t.Errorf("trace missing %q:\n%s", want, tree)
		}
	}
}

func TestWhyPrintsTheWrongPrintedDigits(t *testing.T) {
	out, code := explain("12345678901234567890 + 1", true, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	for _, want := range []string{"proven:  12345678901234567891", "printed in full, float64 gives 12345678901234567168", "wrong in its last 3 digits"} {
		if !strings.Contains(out, want) {
			t.Errorf("why missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "float64 gives") != 1 {
		t.Errorf("why repeats itself:\n%s", out)
	}
}

func TestWhyOnAnUnmodelledExpressionSaysSo(t *testing.T) {
	out, code := explain("20 degC to K", true, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "no float64 shadow") {
		t.Errorf("out = %q", out)
	}
}

func TestOneShotWarnsAboutAWrongPrintedInteger(t *testing.T) {
	out, code := run([]string{"123456789 * 123456789"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "15241578750190521") {
		t.Errorf("out = %q, want the proven product", out)
	}
	if !strings.Contains(out, "float64 gives 15241578750190520 here; the last digit is wrong") {
		t.Errorf("out = %q, want the specific integer warning", out)
	}
	quiet, code := run([]string{"12345 * 12345"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, quiet)
	}
	if strings.Contains(quiet, "float64") {
		t.Errorf("out = %q, want a small exact product to stay silent", quiet)
	}
}

func TestRunAnswersASpeedWithASpeed(t *testing.T) {
	for _, c := range [][2]string{
		{"60 km/h", "16.6666666667 m/s"},
		{"60 km/h to m/s", "16.6666666667 m/s"},
		{"60 km/hour", "16.6666666667 m/s"},
		{"9.8 m/s^2", "9.8 m/s^2"},
		{"1 J/(mol·K)", "1 J/(mol·K)"},
		{"100 km/h * 2 h", "200000 m"},
		{"(60 km/h) / (2 h)", "0.00231481481481 m/s^2"},
		{"5 m/s to km/h", "18 km/h"},
	} {
		out, code := run([]string{c[0]}, plain, false)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", c[0], code, out)
		}
		head, _, _ := strings.Cut(out, "\n")
		if !strings.HasSuffix(strings.TrimSpace(head), c[1]) {
			t.Errorf("%s = %q, want it to end in %q", c[0], head, c[1])
		}
	}
}

func TestRunSaysNothingAboutAnUnambiguousUnit(t *testing.T) {
	for _, src := range []string{"60 km/h", "9.8 m/s^2", "1 J/(mol·K)"} {
		out, code := run([]string{src}, plain, false)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", src, code, out)
		}
		if strings.Contains(out, "read as") {
			t.Errorf("%s = %q, want the answer alone", src, out)
		}
		if strings.Count(strings.TrimSpace(out), "\n") != 0 {
			t.Errorf("%s = %q, want one line", src, out)
		}
	}
}

func TestRunStillSaysHowItReadABoundName(t *testing.T) {
	sheet := doc.New()
	lines := sheet.SetLines([]string{"h = 3", "5 h"})
	out, code := render(lines[1], plain, false, sheet.Arena())
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "read as the unit h (time), not the variable h") {
		t.Errorf("out = %q, want the interpretation note", out)
	}
}

func TestRunKeepsTheRuledReadings(t *testing.T) {
	sheet := doc.New()
	lines := sheet.SetLines([]string{"h = 3", "5 h", "h"})
	for i, want := range []string{"3", "5 h", "3"} {
		out, code := render(lines[i], plain, false, sheet.Arena())
		if code != 0 {
			t.Fatalf("line %d: %s", i, out)
		}
		head, _, _ := strings.Cut(out, "\n")
		if strings.TrimSpace(head) != want {
			t.Errorf("line %q = %q, want %q", lines[i].Src, head, want)
		}
	}
}

func TestALeadingMinusBeginsAnExpression(t *testing.T) {
	for _, argv := range [][]string{{"-2^2"}, {"-1"}, {"-5", "+", "3"}, {"-h"}, {"-.5"}} {
		opts, args, err := options(argv)
		if err != nil {
			t.Fatalf("options(%v): %v", argv, err)
		}
		if opts.json || opts.indian || opts.grouped {
			t.Errorf("options(%v) read a flag: %+v", argv, opts)
		}
		if !reflect.DeepEqual(args, argv) {
			t.Errorf("options(%v) kept %v", argv, args)
		}
	}
}

func TestNegativeExpressionsAnswer(t *testing.T) {
	want := map[string]string{"-2^2": "-4", "-1": "-1", "-5 + 3": "-2", "-(2+3)": "-5"}
	for src, answer := range want {
		out, code := run(strings.Fields(src), plain, false)
		if code != 0 {
			t.Fatalf("%s: exit %d: %s", src, code, out)
		}
		if strings.TrimSpace(out) != answer {
			t.Errorf("%s = %q, want %q", src, out, answer)
		}
	}
}

func TestOptionsReadOnlyDoubleDashedNames(t *testing.T) {
	opts, args, err := options([]string{"--json", "--indian", "run", "sheet.calc"})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.json || !opts.indian {
		t.Errorf("options = %+v", opts)
	}
	if !reflect.DeepEqual(args, []string{"run", "sheet.calc"}) {
		t.Errorf("args = %v", args)
	}
	if _, _, err := options([]string{"--bogus", "1+1"}); err == nil {
		t.Error("an unknown option was accepted")
	}
	if _, _, err := options([]string{"--help"}); err == nil {
		t.Error("--help should report the usage")
	}
	if _, args, _ := options([]string{"--", "-2^2"}); !reflect.DeepEqual(args, []string{"-2^2"}) {
		t.Errorf("after --, args = %v", args)
	}
}

func TestJSONRunEmitsTheSheetDocument(t *testing.T) {
	path := writeSheet(t, "1+1\n2+2\n3 +\nq = (1e16+1)-1e16\n")
	out, code := runFileJSON(path, plain)
	if code == 0 {
		t.Errorf("exit 0 with a failing line: %s", out)
	}
	var sheet struct {
		Schema string `json:"schema"`
		Lines  []struct {
			N      int    `json:"n"`
			Text   string `json:"text"`
			Status string `json:"status"`
			Error  string `json:"error"`
		} `json:"lines"`
	}
	if err := json.Unmarshal([]byte(out), &sheet); err != nil {
		t.Fatalf("--json run is not parseable JSON: %v\n%s", err, out)
	}
	if sheet.Schema != doc.SheetSchema {
		t.Errorf("schema = %q, want %q", sheet.Schema, doc.SheetSchema)
	}
	if len(sheet.Lines) != 4 {
		t.Fatalf("got %d lines", len(sheet.Lines))
	}
	if sheet.Lines[0].Text != "2" || sheet.Lines[1].Text != "4" {
		t.Errorf("lines = %+v", sheet.Lines)
	}
	if sheet.Lines[2].Status != doc.StatusError {
		t.Errorf("line 3 = %+v, want an error", sheet.Lines[2])
	}
}

func TestJSONRunOnACleanSheetExitsZero(t *testing.T) {
	out, code := runFileJSON(writeSheet(t, "1+1\n2+2\n"), plain)
	if code != 0 {
		t.Errorf("exit %d on a clean sheet: %s", code, out)
	}
}

func TestJSONIsRefusedWhereItIsNotProduced(t *testing.T) {
	for _, args := range [][]string{{"trace", "1+1"}, {"why", "1+1"}, {"repl"}} {
		out, code := dispatch(settings{json: true}, args, plain)
		if code != 2 {
			t.Errorf("--json %v: exit %d, want 2", args, code)
		}
		if !strings.Contains(out, "--json") || !strings.Contains(out, args[0]) {
			t.Errorf("--json %v said %q, which does not explain itself", args, out)
		}
	}
	for _, args := range [][]string{{"trace", "1+1"}, {"why", "1+1"}} {
		if out, code := dispatch(settings{}, args, plain); code != 0 {
			t.Errorf("%v without --json: exit %d: %s", args, code, out)
		}
	}
	if out, code := dispatch(settings{}, []string{"repl"}, plain); code != startREPL {
		t.Errorf("repl dispatched to %d: %s", code, out)
	}
}

func TestDispatchSendsJSONRunToTheSheetDocument(t *testing.T) {
	path := writeSheet(t, "1+1\n")
	out, code := dispatch(settings{json: true}, []string{"run", path}, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), `{"schema":"`+doc.SheetSchema+`"`) {
		t.Errorf("--json run emitted %q, want the sheet document", out)
	}
	plainOut, code := dispatch(settings{}, []string{"run", path}, plain)
	if code != 0 || !strings.Contains(plainOut, "= 2") {
		t.Errorf("run without --json = %q (exit %d)", plainOut, code)
	}
}

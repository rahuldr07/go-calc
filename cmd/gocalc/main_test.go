package main

import (
	"os"
	"path/filepath"
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
	if !strings.Contains(out, "read as (1/2)·m") {
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
	out, code := runFile(path, plain, false)
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
	out, code := runFile(path, plain, false)
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
	out, code := runFile(path, plain, false)
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
	out, code := runFile(path, plain, false)
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
	out, code := runFile(writeSheet(t, "h = 12.3 ± 0.2 m\n2 h\n"), plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	want := "read as the variable h, not the unit h (time); the variable carries a measured uncertainty"
	if !strings.Contains(out, want) {
		t.Errorf("out = %q, want %q", out, want)
	}
	out, code = runFile(writeSheet(t, "h = 3\n5 h\n"), plain, false)
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
	out, code := runFile(writeSheet(t, "fine = 2 + 2\nlost = (1e16 + 1) - 1e16\n"), plain, false)
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

package doc

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func text(v value.Value) string { return format.Default.Text(v, nil) }

func numbers(lines []Line) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		if l.Err != nil {
			out[i] = "error: " + l.Err.Error()
			continue
		}
		out[i] = text(l.Value)
	}
	return out
}

func indices(lines []Line) []int {
	var out []int
	for _, l := range lines {
		out = append(out, l.N)
	}
	return out
}

func counted(d *Doc) *[]int {
	var seen []int
	inner := d.evaluator
	d.evaluator = func(env *eval.Env, n parse.Node) (value.Value, error) {
		for i, e := range d.lines {
			if e.node == n {
				seen = append(seen, i)
			}
		}
		return inner(env, n)
	}
	return &seen
}

func TestEditReevaluatesOnlyDependents(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = 2", "b = a * 3", "c = 99"})
	changed := d.Edit(0, "a = 4")
	var got []int
	for _, l := range changed {
		got = append(got, l.N)
	}
	if !reflect.DeepEqual(got, []int{0, 1}) {
		t.Errorf("changed lines = %v, want [0 1]", got)
	}
}

func TestCycleIsReportedNotHung(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"a = b + 1", "b = a + 1"})
	if lines[0].Err == nil && lines[1].Err == nil {
		t.Fatal("cycle not reported")
	}
}

func TestEditCountsEvaluations(t *testing.T) {
	d := New()
	seen := counted(d)
	d.SetLines([]string{"a = 2", "b = a * 3", "c = 99"})
	if !reflect.DeepEqual(*seen, []int{0, 1, 2}) {
		t.Fatalf("first pass evaluated %v, want [0 1 2]", *seen)
	}
	*seen = nil
	d.Edit(0, "a = 4")
	if !reflect.DeepEqual(*seen, []int{0, 1}) {
		t.Errorf("edit evaluated %v, want [0 1]", *seen)
	}
}

func TestUnrelatedLineKeepsItsCachedResult(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = 2", "b = a * 3", "c = 99"})
	sentinel := value.Scalar(num.FromInt(-777))
	d.lines[2].value = sentinel
	d.Edit(0, "a = 4")
	if got := text(d.Lines()[2].Value); got != "-777" {
		t.Errorf("line 2 = %q, want the untouched sentinel -777", got)
	}
	if got := numbers(d.Lines()[:2]); !reflect.DeepEqual(got, []string{"4", "12"}) {
		t.Errorf("lines 0 and 1 = %v, want [4 12]", got)
	}
}

func TestDependentsSeeTheNewValue(t *testing.T) {
	d := New()
	got := numbers(d.SetLines([]string{"a = 2", "b = a * 3", "c = b + 1"}))
	if want := []string{"2", "6", "7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first pass = %v, want %v", got, want)
	}
	d.Edit(0, "a = 4")
	if got, want := numbers(d.Lines()), []string{"4", "12", "13"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after edit = %v, want %v", got, want)
	}
}

func TestCycleMessageAndNoHang(t *testing.T) {
	done := make(chan []Line, 1)
	go func() {
		d := New()
		done <- d.SetLines([]string{"a = b + 1", "b = a + 1", "z = 7"})
	}()
	var lines []Line
	select {
	case lines = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SetLines hung on a cycle")
	}
	for _, i := range []int{0, 1} {
		if lines[i].Err == nil || lines[i].Err.Error() != "this line depends on itself" {
			t.Errorf("line %d error = %v, want this line depends on itself", i, lines[i].Err)
		}
	}
	if lines[2].Err != nil || text(lines[2].Value) != "7" {
		t.Errorf("line 2 = %v %q, want 7", lines[2].Err, text(lines[2].Value))
	}
}

func TestSelfReferenceIsACycle(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"a = a + 1"})
	if lines[0].Err == nil || lines[0].Err.Error() != "this line depends on itself" {
		t.Errorf("error = %v, want this line depends on itself", lines[0].Err)
	}
}

func TestRecursiveCallIsACycle(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"f(x) = g(x)", "g(x) = f(x)", "f(1)"})
	if lines[2].Err == nil || lines[2].Err.Error() != "this line depends on itself" {
		t.Errorf("error = %v, want this line depends on itself", lines[2].Err)
	}
}

func TestFailingLineLeavesIndependentLinesAlone(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"bad = 5 m + 3 s", "ok = 2 + 2", "also = ok * 10"})
	if lines[0].Err == nil {
		t.Fatal("line 0 should have failed")
	}
	if got, want := numbers(lines[1:]), []string{"4", "40"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines 1 and 2 = %v, want %v", got, want)
	}
}

func TestDependentOfAFailingLineNamesTheFailure(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"bad = 5 m + 3 s", "uses = bad + 1", "fine = 1 + 1"})
	if lines[1].Err == nil {
		t.Fatal("line 1 should have failed with its input missing")
	}
	if got, want := lines[1].Err.Error(), "uses bad, which failed on line 1"; got != want {
		t.Errorf("line 1 error = %q, want %q", got, want)
	}
	if lines[2].Err != nil || text(lines[2].Value) != "2" {
		t.Errorf("line 2 = %v %q, want 2", lines[2].Err, text(lines[2].Value))
	}
}

func TestDependentNamesEveryFailedInput(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"bad = 5 m + 3 s", "worse = 1 kg + 2 s", "uses = bad + worse"})
	if lines[2].Err == nil {
		t.Fatal("line 2 should have failed")
	}
	want := "uses bad, worse, which failed on lines 1, 2"
	if got := lines[2].Err.Error(); got != want {
		t.Errorf("line 2 error = %q, want %q", got, want)
	}
}

func TestFailureExplanationTravelsDownTheChain(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"bad = 5 m + 3 s", "mid = bad * 2", "far = mid * 2"})
	if got, want := lines[2].Err.Error(), "uses mid, which failed on line 2"; got != want {
		t.Errorf("line 2 error = %q, want %q", got, want)
	}
	changed := d.Edit(0, "bad = 5 m + 3 m")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Fatalf("changed = %v, want [0 1 2]", got)
	}
	if got, want := numbers(d.Lines()), []string{"8 m", "16 m", "32 m"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after the fix = %v, want %v", got, want)
	}
}

func TestEditCanIntroduceACycle(t *testing.T) {
	d := New()
	got := numbers(d.SetLines([]string{"a = 1", "b = a + 1"}))
	if want := []string{"1", "2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first pass = %v, want %v", got, want)
	}
	changed := d.Edit(0, "a = b + 1")
	if idx := indices(changed); !reflect.DeepEqual(idx, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", idx)
	}
	for _, l := range d.Lines() {
		if l.Err == nil || l.Err.Error() != "this line depends on itself" {
			t.Errorf("line %d error = %v, want this line depends on itself", l.N, l.Err)
		}
	}
}

func TestEditCanBreakACycle(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = b + 1", "b = a + 1"})
	changed := d.Edit(0, "a = 1")
	if idx := indices(changed); !reflect.DeepEqual(idx, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", idx)
	}
	if got, want := numbers(d.Lines()), []string{"1", "2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after the edit = %v, want %v", got, want)
	}
}

func TestEditFromDefinitionToPlainExpression(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = 2", "b = a * 3", "c = 99"})
	changed := d.Edit(0, "7 + 1")
	if idx := indices(changed); !reflect.DeepEqual(idx, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", idx)
	}
	lines := d.Lines()
	if lines[0].Err != nil || text(lines[0].Value) != "8" {
		t.Errorf("line 0 = %v %q, want 8", lines[0].Err, text(lines[0].Value))
	}
	if lines[1].Err == nil {
		t.Error("line 1 should fail once nothing defines a")
	}
	if text(lines[2].Value) != "99" {
		t.Errorf("line 2 = %q, want 99", text(lines[2].Value))
	}
}

func TestEditFromPlainExpressionToDefinition(t *testing.T) {
	d := New()
	d.SetLines([]string{"7 + 1", "b = a * 3"})
	if d.Lines()[1].Err == nil {
		t.Fatal("line 1 should fail while nothing defines a")
	}
	changed := d.Edit(0, "a = 2")
	if idx := indices(changed); !reflect.DeepEqual(idx, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", idx)
	}
	if got := text(d.Lines()[1].Value); got != "6" {
		t.Errorf("line 1 = %q, want 6", got)
	}
}

func TestParametersDoNotDependOnOuterLines(t *testing.T) {
	d := New()
	seen := counted(d)
	lines := d.SetLines([]string{"x = 100", "f(x) = x + 1", "f(2)"})
	if got, want := text(lines[2].Value), "3"; got != want {
		t.Fatalf("f(2) = %q, want %q", got, want)
	}
	if deps := d.lines[1].deps; len(deps) != 0 {
		t.Errorf("f's definition depends on %v, want nothing", deps)
	}
	*seen = nil
	changed := d.Edit(0, "x = 500")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0}) {
		t.Errorf("changed = %v, want [0]", got)
	}
	if !reflect.DeepEqual(*seen, []int{0}) {
		t.Errorf("evaluated %v, want [0]", *seen)
	}
	if got := text(d.Lines()[2].Value); got != "3" {
		t.Errorf("f(2) = %q after an unrelated edit, want 3", got)
	}
}

func TestCallerDependsOnFreeNamesOfTheBody(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"a = 10", "g(x) = x + a", "g(1)"})
	if got := text(lines[2].Value); got != "11" {
		t.Fatalf("g(1) = %q, want 11", got)
	}
	changed := d.Edit(0, "a = 20")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0, 2}) {
		t.Errorf("changed = %v, want [0 2]", got)
	}
	if got := text(d.Lines()[2].Value); got != "21" {
		t.Errorf("g(1) = %q, want 21", got)
	}
}

func TestUnitsAndConstantsAreNotDependencies(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"h = 3 m", "h + 200 cm", "3 km to m"})
	if got, want := numbers(lines), []string{"3 m", "5 m", "3000 m"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	if deps := d.lines[2].deps; len(deps) != 0 {
		t.Errorf("a conversion depends on %v, want nothing", deps)
	}
	if deps := d.lines[1].deps; !reflect.DeepEqual(deps, []int{0}) {
		t.Errorf("deps = %v, want [0]", deps)
	}
}

func TestLaterDefinitionShadowsEarlierOne(t *testing.T) {
	d := New()
	got := numbers(d.SetLines([]string{"a = 2", "b = a * 3", "a = 5", "c = a * 3"}))
	if want := []string{"2", "6", "5", "15"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	changed := d.Edit(0, "a = 1")
	if idx := indices(changed); !reflect.DeepEqual(idx, []int{0, 1}) {
		t.Errorf("changed = %v, want [0 1]", idx)
	}
	if got, want := numbers(d.Lines()), []string{"1", "3", "5", "15"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after edit = %v, want %v", got, want)
	}
}

func TestForwardReferenceIsAnError(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"b = a + 1", "a = 41"})
	if lines[0].Err == nil || lines[0].Err.Error() != "uses a before it is defined" {
		t.Errorf("line 0 error = %v, want uses a before it is defined", lines[0].Err)
	}
	if got := text(lines[0].Value); got != "0" {
		t.Errorf("line 0 kept the value %q, want nothing", got)
	}
	if lines[1].Err != nil || text(lines[1].Value) != "41" {
		t.Errorf("line 1 = %v %q, want 41", lines[1].Err, text(lines[1].Value))
	}
}

func TestForwardReferenceNamesEveryName(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"c = a + b", "a = 1", "b = 2"})
	if lines[0].Err == nil || lines[0].Err.Error() != "uses a, b before they are defined" {
		t.Errorf("error = %v, want uses a, b before they are defined", lines[0].Err)
	}
}

func TestForwardReferenceThroughAFunctionBody(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"g(x) = x + a", "g(1)", "a = 10", "g(2)"})
	if lines[1].Err == nil || lines[1].Err.Error() != "uses a before it is defined" {
		t.Errorf("line 1 error = %v, want uses a before it is defined", lines[1].Err)
	}
	if lines[3].Err != nil || text(lines[3].Value) != "12" {
		t.Errorf("line 3 = %v %q, want 12", lines[3].Err, text(lines[3].Value))
	}
}

func TestDefiningAboveClearsTheForwardError(t *testing.T) {
	d := New()
	d.SetLines([]string{"# placeholder", "b = a + 1", "a = 41"})
	if err := d.Lines()[1].Err; err == nil {
		t.Fatal("line 1 should fail while a is defined below it")
	}
	changed := d.Edit(0, "a = 6")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", got)
	}
	if l := d.Lines()[1]; l.Err != nil || text(l.Value) != "7" {
		t.Errorf("line 1 = %v %q, want 7", l.Err, text(l.Value))
	}
}

func TestBlankAndCommentLinesAreInert(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"# lab notes", "", "   ", "h = 3 m", "h + 200 cm"})
	for _, i := range []int{0, 1, 2} {
		if lines[i].Err != nil {
			t.Errorf("line %d error = %v, want none", i, lines[i].Err)
		}
	}
	if got := text(lines[4].Value); got != "5 m" {
		t.Errorf("line 4 = %q, want 5 m", got)
	}
}

func TestEditIntroducingADefinitionRevivesDependents(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"# placeholder", "q = w + 1"})
	if lines[1].Err == nil {
		t.Fatal("line 1 should fail while w is undefined")
	}
	changed := d.Edit(0, "w = 9")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", got)
	}
	if got := text(d.Lines()[1].Value); got != "10" {
		t.Errorf("q = %q, want 10", got)
	}
}

func TestEditRemovingADefinitionBreaksDependents(t *testing.T) {
	d := New()
	d.SetLines([]string{"w = 9", "q = w + 1"})
	changed := d.Edit(0, "# gone")
	if got := indices(changed); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatalf("changed = %v, want [0 1]", got)
	}
	if d.Lines()[1].Err == nil {
		t.Error("q should fail once w is gone")
	}
}

func TestParseErrorIsReportedOnItsLine(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"2 +", "3 + 4"})
	if lines[0].Err == nil {
		t.Error("line 0 should carry the parse error")
	}
	if lines[1].Err != nil || text(lines[1].Value) != "7" {
		t.Errorf("line 1 = %v %q, want 7", lines[1].Err, text(lines[1].Value))
	}
}

func TestNotesTravelWithTheLine(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"1/2m"})
	if len(lines[0].Notes) == 0 {
		t.Fatal("want an interpretation note")
	}
	if !strings.Contains(format.Default.Text(lines[0].Value, lines[0].Notes), "read as 1/(2·m)") {
		t.Errorf("notes = %v", lines[0].Notes)
	}
}

func TestEditOutOfRangeChangesNothing(t *testing.T) {
	d := New()
	d.SetLines([]string{"a = 1"})
	if got := d.Edit(5, "a = 2"); got != nil {
		t.Errorf("Edit out of range = %v, want nil", got)
	}
	if got := text(d.Lines()[0].Value); got != "1" {
		t.Errorf("line 0 = %q, want 1", got)
	}
}

func TestCorrelationDeclaredOnOneLineReachesALaterLine(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{
		"a = 10 ± 1",
		"b = 10 ± 1",
		"corr(a, b) = 1",
		"a + b",
	})
	want := "20.0 ± 2.0\n  the budget includes a covariance term from declared correlation"
	if got := numbers(lines)[3]; got != want {
		t.Errorf("a + b = %q, want %q", got, want)
	}
}

func TestEditingTheCorrelationReevaluatesWhatUsesIt(t *testing.T) {
	d := New()
	d.SetLines([]string{
		"a = 10 ± 1",
		"b = 10 ± 1",
		"corr(a, b) = 1",
		"a + b",
	})
	changed := d.Edit(2, "corr(a, b) = 0")
	if !reflect.DeepEqual(indices(changed), []int{2, 3}) {
		t.Errorf("changed lines = %v, want [2 3]", indices(changed))
	}
	if got := numbers(d.Lines())[3]; got != "20.0 ± 1.4" {
		t.Errorf("a + b = %q, want 20.0 ± 1.4 with no covariance note", got)
	}
}

func TestMeasuredVariableBeatsAUnitName(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{
		"h = 12.3 ± 0.2 m",
		"g = g0",
		"t = sqrt(2h / g)",
	})
	if got := numbers(lines)[2]; got != "1.584 ± 0.013 s" {
		t.Errorf("t = %q, want 1.584 ± 0.013 s", got)
	}
}

func TestPlainVariableStillLosesToAUnitName(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"h = 3", "5 h"})
	if got := numbers(lines)[1]; got != "5 h" {
		t.Errorf("5 h = %q, want 5 h", got)
	}
}

func messages(l Line) []string {
	out := make([]string, 0, len(l.Notes))
	for _, n := range l.Notes {
		out = append(out, n.Msg)
	}
	return out
}

func TestDocumentSaysWhichReadingItTook(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{
		"h = 3",
		"5 h",
		"k = 12.3 ± 0.2 m",
		"2 k",
	})
	plain := "read as the unit h (time), not the variable h; the variable is a plain number"
	if got := messages(lines[1]); !reflect.DeepEqual(got, []string{plain}) {
		t.Errorf("line 2 notes = %q, want [%q]", got, plain)
	}
	if got := messages(lines[3]); len(got) != 0 {
		t.Errorf("line 4 notes = %q, want none: k is not a unit", got)
	}
}

func TestDocumentNotesDoNotPileUpAcrossEdits(t *testing.T) {
	d := New()
	d.SetLines([]string{"h = 3", "5 h"})
	d.Edit(0, "h = 4")
	got := messages(d.Lines()[1])
	if len(got) != 1 {
		t.Errorf("notes after an edit = %q, want exactly one", got)
	}
	if numbers(d.Lines())[1] != "5 h" {
		t.Errorf("5 h = %q, want 5 h", numbers(d.Lines())[1])
	}
}

func TestDocumentReportsTheSkippedNonlinearityCheck(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{
		"a = 1 ± 1",
		"b = 1 ± 1",
		"corr(a, b) = 0.5",
		"a * b",
	})
	if !strings.Contains(text(lines[3].Value), "nonlinearity not checked: inputs are correlated") {
		t.Errorf("a * b = %q, want the skipped check reported", text(lines[3].Value))
	}
}

func TestAFailedLineDropsItsOldEvaluatorNotes(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"h = 3", "5 h"})
	if len(messages(lines[1])) != 1 {
		t.Fatalf("setup notes = %q, want one", messages(lines[1]))
	}
	d.Edit(0, "h = 1/0")
	after := d.Lines()[1]
	if after.Err == nil {
		t.Fatal("line 2 should have failed with its input")
	}
	if got := messages(after); len(got) != 0 {
		t.Errorf("notes on the failed line = %q, want none", got)
	}
}

func TestWarningLandsOnTheLineThatLosesDigits(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"fine = 2 + 2", "lost = (1e16 + 1) - 1e16"})
	if lines[0].Warning != "" {
		t.Errorf("line 1 warned %q", lines[0].Warning)
	}
	if !strings.Contains(lines[1].Warning, "float64") {
		t.Errorf("line 2 warning = %q", lines[1].Warning)
	}
	if !strings.Contains(lines[1].Warning, "at 2:") {
		t.Errorf("line 2 warning = %q, want it to name the sheet line", lines[1].Warning)
	}
}

func TestEditClearsAWarningThatNoLongerApplies(t *testing.T) {
	d := New()
	d.SetLines([]string{"x = (1e16 + 1) - 1e16"})
	changed := d.Edit(0, "x = 2 + 2")
	if len(changed) != 1 {
		t.Fatalf("changed = %d lines", len(changed))
	}
	if changed[0].Warning != "" {
		t.Errorf("warning survived the edit: %q", changed[0].Warning)
	}
}

func TestEveryLineCarriesProvenance(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"h = 12.3 ± 0.2 m", "", "# note", "t = sqrt(2h / g0)"})
	for _, l := range lines {
		if Blank(l.Src) {
			continue
		}
		if l.Err != nil {
			t.Fatalf("line %d: %v", l.N, l.Err)
		}
		if tree := d.Arena().Tree(l.Value.Prov); tree == "" {
			t.Errorf("line %d has no trace", l.N)
		}
	}
	tree := d.Arena().Tree(lines[3].Value.Prov)
	if !strings.Contains(tree, "line 1") {
		t.Errorf("trace does not point back at the input line:\n%s", tree)
	}
}

func TestAUnitIsNotAForwardReference(t *testing.T) {
	for _, c := range []struct {
		src  []string
		want []string
	}{
		{[]string{"v = 60 km/h", "h = 3"}, []string{"50/3  ≈ 16.6666666667 m/s", "3"}},
		{[]string{"x = 5 m/s", "y = x to km/h", "s = 2"}, []string{"5 m/s", "18 km/h", "2"}},
		{[]string{"a = 1 J/(mol·K)", "mol = 7"}, []string{"1 J/(mol·K)", "7"}},
		{[]string{"p = 5 km to km/h·s", "h = 3"}, []string{"18000 (km/h)·s", "3"}},
	} {
		d := New()
		lines := d.SetLines(c.src)
		if got := numbers(lines); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestAUnitNameDefinedAboveStillLinksTheLine(t *testing.T) {
	d := New()
	lines := d.SetLines([]string{"h = 3", "v = 60 km/h"})
	if got := numbers(lines); !reflect.DeepEqual(got, []string{"3", "50/3  ≈ 16.6666666667 m/s"}) {
		t.Errorf("sheet = %v", numbers(lines))
	}
	changed := d.Edit(0, "h = 12.3 ± 0.2 m")
	if len(changed) < 2 {
		t.Errorf("redefining h changed %d lines, want the line that reads h as a unit to follow", len(changed))
	}
	if got := numbers(d.Lines())[1]; !strings.Contains(got, "±") {
		t.Errorf("after h became a measurement, 60 km/h = %q, want the variable reading", got)
	}
}

var implicitShapes = []struct{ src, same string }{
	{"2 m^2", "2 * (m^2)"},
	{"2 km^2", "2 * (km^2)"},
	{"0.5 s^-1", "0.5 * (s^-1)"},
	{"2 pi^2", "2 * (pi^2)"},
	{"3 x^2", "3 * (x^2)"},
	{"2 (x + 1)^2", "2 * ((x + 1)^2)"},
	{"2 sqrt(x)^2", "2 * (sqrt(x)^2)"},
	{"100 km / 2 h", "(100 * km) / (2 * h)"},
	{"3 m / 2 s / 4 s", "((3 * m) / (2 * s)) / (4 * s)"},
	{"100 W / 2 m^2", "(100 * W) / (2 * (m^2))"},
	{"1 / 2 x", "1 / (2 * x)"},
	{"60 km/h", "(60 * km) / h"},
	{"9.8 m/s^2", "(9.8 * m) / (s^2)"},
	{"3 m^2 + 1 m^2", "(3 * (m^2)) + (1 * (m^2))"},
}

func TestImplicitProductsReadAsTheirExplicitGrouping(t *testing.T) {
	for _, c := range implicitShapes {
		got := numbers(New().SetLines([]string{"x = 3", c.src}))[1]
		want := numbers(New().SetLines([]string{"x = 3", c.same}))[1]
		if got != want {
			t.Errorf("%q = %q, want %q (the reading of %q)", c.src, got, want, c.same)
		}
	}
}

func TestImplicitProductsReadTheSameThroughAVariable(t *testing.T) {
	for _, c := range implicitShapes {
		got := numbers(New().SetLines([]string{"x = 3", "q = " + c.src, "q"}))[2]
		want := numbers(New().SetLines([]string{"x = 3", "q = " + c.same, "q"}))[2]
		if got != want {
			t.Errorf("q = %q gives %q, want %q", c.src, got, want)
		}
	}
}

func TestADeclaredMeasurementIsNotSilentlyRaisedToAPower(t *testing.T) {
	lines := New().SetLines([]string{"v = 2 m^3 ± 0.01 m^3", "v"})
	if got := numbers(lines)[1]; got != "2.000 ± 0.010 m^3" {
		t.Errorf("v = %q, want 2.000 ± 0.010 m^3", got)
	}
}

func TestImplicitProductsReadTheSameThroughTheSheetAPI(t *testing.T) {
	for _, c := range implicitShapes {
		got := sheetTexts(t, []string{"x = 3", c.src})[1]
		want := sheetTexts(t, []string{"x = 3", c.same})[1]
		if got != want {
			t.Errorf("sheet %q = %q, want %q", c.src, got, want)
		}
	}
}

func sheetTexts(t *testing.T, lines []string) []string {
	t.Helper()
	req, err := json.Marshal(map[string]any{"lines": lines})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Lines []struct {
			Text  string `json:"text"`
			Error string `json:"error"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(EvalRequest(req), &out); err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(out.Lines))
	for i, l := range out.Lines {
		got[i] = l.Text + l.Error
	}
	return got
}

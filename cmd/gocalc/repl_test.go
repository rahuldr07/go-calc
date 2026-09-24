package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
)

func statedir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	return dir
}

func session(t *testing.T, input string) string {
	t.Helper()
	statedir(t)
	return replOutput(t, input, plain)
}

func replOutput(t *testing.T, input string, style format.Style) string {
	t.Helper()
	var out bytes.Buffer
	if code := repl(strings.NewReader(input), &out, style); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	return out.String()
}

func TestReplEvaluatesAndRemembers(t *testing.T) {
	statedir(t)
	in := strings.NewReader("x = 2 + 3\nx * 10\nwhy\nexit\n")
	var out bytes.Buffer
	if code := repl(in, &out, format.Style{Digits: 12}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := out.String()
	for _, want := range []string{"5", "50"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestReplPrintsOneAnswerPerLine(t *testing.T) {
	got := strings.Split(strings.TrimRight(session(t, "1 + 1\n2 + 2\n3 + 3\n"), "\n"), "\n")
	want := []string{"2", "4", "6"}
	if len(got) != len(want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReplRendersExactlyLikeOneShot(t *testing.T) {
	for _, src := range []string{"2 + 3", "1/3", "1 m ± 2 cm", "500 + 10%", "3 km to m", "1/2m", "sqrt(2)"} {
		want, code := run([]string{src}, plain, false)
		if code != 0 {
			t.Fatalf("%s: one-shot exit %d: %s", src, code, want)
		}
		got := strings.TrimRight(session(t, src+"\n"), "\n")
		if got != want {
			t.Errorf("%s: repl printed\n%s\nbut one-shot printed\n%s", src, got, want)
		}
	}
}

func TestReplKeepsGoingAfterAnError(t *testing.T) {
	got := session(t, "5 m + 3 s\n2 + 2\n")
	if !strings.Contains(got, "cannot add length (m) and time (s)") {
		t.Errorf("missing the error:\n%s", got)
	}
	if !strings.Contains(got, "\n4\n") && !strings.HasSuffix(got, "4\n") {
		t.Errorf("the session stopped at the error:\n%s", got)
	}
}

func TestReplMarksErrorsInPlainText(t *testing.T) {
	got := session(t, "nope + 1\n")
	if !strings.HasPrefix(got, "error: ") {
		t.Errorf("an error must be labelled where a result is not:\n%s", got)
	}
}

func TestReplExitStopsReading(t *testing.T) {
	got := session(t, "1 + 1\nexit\n99 + 1\n")
	if strings.Contains(got, "100") {
		t.Errorf("read past exit:\n%s", got)
	}
}

func TestReplSkipsBlankAndCommentLines(t *testing.T) {
	got := session(t, "\n# a note\n   \n7\n")
	if strings.TrimRight(got, "\n") != "7" {
		t.Errorf("out = %q, want only the answer", got)
	}
}

func TestReplWhyExplainsTheLastResult(t *testing.T) {
	got := session(t, "x = 2 + 3\nx * 10\nwhy\n")
	if !strings.Contains(got, "proven:") || !strings.Contains(got, "float64:") {
		t.Errorf("why did not explain the last result:\n%s", got)
	}
	if !strings.Contains(got, "float64: 50") {
		t.Errorf("why explained something other than x * 10:\n%s", got)
	}
}

func TestReplWhyExplainsAnAssignmentItself(t *testing.T) {
	got := session(t, "x = 2 + 3\nwhy\n")
	if !strings.Contains(got, "float64: 5") {
		t.Errorf("why after an assignment:\n%s", got)
	}
}

func TestReplWhyLooksPastACommand(t *testing.T) {
	got := session(t, "x = 2\nx * 10\ndigits 5\nwhy\n")
	if !strings.Contains(got, "float64: 20") {
		t.Errorf("a command must not become the last result:\n%s", got)
	}
}

func TestReplWhyLooksPastAFailedLine(t *testing.T) {
	got := session(t, "x = 2\nx * 10\n5 m + 3 s\nwhy\n")
	if !strings.Contains(got, "float64: 20") {
		t.Errorf("a failed line must not become the last result:\n%s", got)
	}
}

func TestReplWhyWithAnExpression(t *testing.T) {
	got := session(t, "why 2 + 2\n")
	if !strings.Contains(got, "float64: 4") {
		t.Errorf("why <expression>:\n%s", got)
	}
}

func TestReplWhyBeforeAnyResult(t *testing.T) {
	got := session(t, "why\n")
	if !strings.Contains(got, "error: ") {
		t.Errorf("why with nothing to explain:\n%s", got)
	}
}

func TestReplTraceShowsWhereANumberCameFrom(t *testing.T) {
	got := session(t, "x = 2 m\ntrace x\n")
	if !strings.Contains(got, "[unit]") || !strings.Contains(got, "[literal]") {
		t.Errorf("trace did not show the provenance:\n%s", got)
	}
}

func TestReplTraceWorksTwiceAndAfterAnAssignment(t *testing.T) {
	got := session(t, "x = 2 m\ny = x + 1 m\ntrace x\ntrace y\n")
	first := strings.Index(got, "[unit]")
	last := strings.LastIndex(got, "[unit]")
	if first < 0 || first == last {
		t.Errorf("the second trace lost its provenance:\n%s", got)
	}
}

func TestReplTraceOfTheLastResult(t *testing.T) {
	got := session(t, "2 m + 3 m\ntrace\n")
	if !strings.Contains(got, "[operation]") {
		t.Errorf("bare trace did not trace the last result:\n%s", got)
	}
}

func TestReplTraceOfAnUnknownName(t *testing.T) {
	got := session(t, "trace nope\n")
	if !strings.Contains(got, "error: ") {
		t.Errorf("trace of an unknown name:\n%s", got)
	}
}

func TestReplInfoOnAConstant(t *testing.T) {
	got := session(t, "info c\n")
	for _, want := range []string{"speed of light", "CODATA 2022", "299792458", "m/s"} {
		if !strings.Contains(got, want) {
			t.Errorf("info c missing %q:\n%s", want, got)
		}
	}
}

func TestReplInfoReportsAStatedUncertainty(t *testing.T) {
	line := lastLine(session(t, "info G\n"))
	if !strings.Contains(line, "±") {
		t.Errorf("info on a measured constant must state its uncertainty: %q", line)
	}
	if strings.ContainsAny(line, "[]") {
		t.Errorf("a standard uncertainty is a ±, never an interval: %q", line)
	}
	if exact := lastLine(session(t, "info c\n")); strings.Contains(exact, "±") {
		t.Errorf("an exact constant must not wear a ±: %q", exact)
	}
}

func TestReplInfoOnAUnit(t *testing.T) {
	got := session(t, "info km\n")
	for _, want := range []string{"[unit]", "1 km = 1000 m", "length"} {
		if !strings.Contains(got, want) {
			t.Errorf("info km missing %q:\n%s", want, got)
		}
	}
}

func TestReplInfoOnASessionName(t *testing.T) {
	got := session(t, "speed = 3 m/s\ninfo speed\n")
	if !strings.Contains(got, "this session") || !strings.Contains(got, "3 m/s") {
		t.Errorf("info on a name you defined:\n%s", got)
	}
}

func TestReplInfoOnANameThatShadowsAUnit(t *testing.T) {
	got := session(t, "m = 7\ninfo m\n")
	if !strings.Contains(got, "this session") {
		t.Errorf("your own name must win over the unit table:\n%s", got)
	}
}

func TestReplInfoOnNothingKnown(t *testing.T) {
	got := session(t, "info zzz\n")
	if !strings.Contains(got, "error: ") {
		t.Errorf("info on an unknown name:\n%s", got)
	}
}

func TestReplDigitsControlsThePrintedDigits(t *testing.T) {
	for _, n := range []int{3, 8, 20, 30} {
		got := session(t, "digits "+strconv.Itoa(n)+"\nsqrt(2)\n")
		line := lastLine(got)
		if count := countDigits(line); count != n {
			t.Errorf("digits %d printed %d significant digits (%q)", n, count, line)
		}
	}
}

func TestReplDigitsSurvivesAnAssignment(t *testing.T) {
	got := session(t, "digits 5\nx = sqrt(2)\nx * 1\n")
	line := lastLine(got)
	if countDigits(line) != 5 {
		t.Errorf("digits was forgotten after an assignment: %q", line)
	}
}

func TestReplDigitsReportsAndRefuses(t *testing.T) {
	if got := session(t, "digits\n"); !strings.Contains(got, "12") {
		t.Errorf("bare digits must report the setting:\n%s", got)
	}
	for _, bad := range []string{"digits 0", "digits -3", "digits two", "digits 99999"} {
		got := session(t, bad+"\n")
		if !strings.Contains(got, "error: ") {
			t.Errorf("%q was accepted:\n%s", bad, got)
		}
	}
}

func TestReplDegAndRad(t *testing.T) {
	got := session(t, "deg\nsin(30)\nrad\nsin(30)\n")
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 4 {
		t.Fatalf("out:\n%s", got)
	}
	if !strings.HasPrefix(lines[1], "0.5") {
		t.Errorf("sin(30) in degrees = %q, want 0.5", lines[1])
	}
	if strings.HasPrefix(lines[3], "0.5") {
		t.Errorf("rad did not switch back: %q", lines[3])
	}
}

func TestReplDegreesSurviveALaterAssignment(t *testing.T) {
	got := session(t, "deg\nangle = 30\nsin(angle)\n")
	if !strings.Contains(lastLine(got), "0.5") {
		t.Errorf("degrees were forgotten:\n%s", got)
	}
}

func TestReplHelpNamesEveryCommand(t *testing.T) {
	got := session(t, "help\n")
	for _, c := range commands {
		if !strings.Contains(got, c.name) {
			t.Errorf("help does not mention %q:\n%s", c.name, got)
		}
	}
	if !strings.Contains(got, "?") {
		t.Errorf("help does not mention the lookup command:\n%s", got)
	}
}

func TestEveryAdvertisedFunctionExists(t *testing.T) {
	for _, name := range functions {
		if known(name) {
			continue
		}
		t.Errorf("help advertises %q but the evaluator does not know it", name)
	}
}

func known(name string) bool {
	for _, call := range []string{name + "(1)", name + "(1, 1)"} {
		out, code := run([]string{call}, plain, false)
		if code == 0 || !strings.Contains(out, "unknown function") {
			return true
		}
	}
	return false
}

func TestReplLookupListsMatchingNames(t *testing.T) {
	got := session(t, "speed = 3\n?sp\n")
	if !strings.Contains(got, "speed") {
		t.Errorf("lookup missed a name defined in the session:\n%s", got)
	}
}

func TestReplLookupListsConstantsUnitsAndFunctions(t *testing.T) {
	if got := session(t, "?m_\n"); !strings.Contains(got, "m_e") {
		t.Errorf("lookup missed a constant:\n%s", got)
	}
	if got := session(t, "?mo\n"); !strings.Contains(got, "mol") {
		t.Errorf("lookup missed a unit:\n%s", got)
	}
	if got := session(t, "?sq\n"); !strings.Contains(got, "sqrt") {
		t.Errorf("lookup missed a function:\n%s", got)
	}
	if got := session(t, "?km\n"); !strings.Contains(got, "units: km") {
		t.Errorf("lookup missed a prefixed unit:\n%s", got)
	}
	if got := session(t, "?mo\n"); !strings.Contains(got, "units: mol") {
		t.Errorf("lookup did not group units under their own heading:\n%s", got)
	}
}

func TestReplLookupWithNoMatch(t *testing.T) {
	got := session(t, "?zzzz\n")
	if !strings.Contains(got, "error: ") {
		t.Errorf("lookup with no match:\n%s", got)
	}
}

func TestReplCommandNamesStayUsableAsVariables(t *testing.T) {
	got := session(t, "digits = 4\ndigits + 1\n")
	if !strings.Contains(got, "5") {
		t.Errorf("an assignment to a command name must be an assignment:\n%s", got)
	}
}

func TestReplIsPlainWhenNotATerminal(t *testing.T) {
	got := session(t, "1 + 1\nhelp\ninfo c\n")
	if strings.Contains(got, "\x1b") {
		t.Errorf("escape sequences reached a pipe:\n%q", got)
	}
	if strings.HasPrefix(got, prompt) || strings.Contains(got, "\n"+prompt) {
		t.Errorf("a prompt reached a pipe:\n%q", got)
	}
	if strings.Contains(got, banner) {
		t.Errorf("a banner reached a pipe:\n%q", got)
	}
}

func TestReplHandlesALastLineWithoutANewline(t *testing.T) {
	got := session(t, "2 + 2")
	if strings.TrimRight(got, "\n") != "4" {
		t.Errorf("out = %q", got)
	}
}

func TestReplReportsAReadFailure(t *testing.T) {
	statedir(t)
	var out bytes.Buffer
	if code := repl(brokenReader{}, &out, plain); code == 0 {
		t.Errorf("a read failure must not exit 0:\n%s", out.String())
	}
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestReplWritesHistory(t *testing.T) {
	dir := statedir(t)
	replOutput(t, "1 + 1\ndigits 5\nexit\n", plain)
	lines := historyLines(t, filepath.Join(dir, "go-calc", "history"))
	want := []string{"1 + 1", "digits 5", "exit"}
	if len(lines) != len(want) {
		t.Fatalf("history = %q, want %q", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("history[%d] = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestReplAppendsToExistingHistory(t *testing.T) {
	dir := statedir(t)
	replOutput(t, "1 + 1\n", plain)
	replOutput(t, "2 + 2\n", plain)
	lines := historyLines(t, filepath.Join(dir, "go-calc", "history"))
	if len(lines) != 2 || lines[0] != "1 + 1" || lines[1] != "2 + 2" {
		t.Errorf("history = %q", lines)
	}
}

func TestReplCapsHistoryAtAThousandLines(t *testing.T) {
	dir := statedir(t)
	var b strings.Builder
	for i := 0; i < 1005; i++ {
		b.WriteString("# line " + strconv.Itoa(i) + "\n")
	}
	replOutput(t, b.String(), plain)
	lines := historyLines(t, filepath.Join(dir, "go-calc", "history"))
	if len(lines) != 1000 {
		t.Fatalf("history kept %d lines, want 1000", len(lines))
	}
	if lines[0] != "# line 5" {
		t.Errorf("the oldest kept line is %q, want %q", lines[0], "# line 5")
	}
	if lines[len(lines)-1] != "# line 1004" {
		t.Errorf("the newest kept line is %q", lines[len(lines)-1])
	}
}

func TestReplHistoryFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)
	replOutput(t, "6 * 7\n", plain)
	lines := historyLines(t, filepath.Join(home, ".local", "state", "go-calc", "history"))
	if len(lines) != 1 || lines[0] != "6 * 7" {
		t.Errorf("history = %q", lines)
	}
}

func TestReplSaysWhenHistoryCannotBeSaved(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	got := replOutput(t, "1 + 1\n", plain)
	if !strings.Contains(got, "history not saved") {
		t.Errorf("a lost history must be reported:\n%s", got)
	}
}

func TestReplHistoryIsPrivate(t *testing.T) {
	dir := statedir(t)
	replOutput(t, "1 + 1\n", plain)
	info, err := os.Stat(filepath.Join(dir, "go-calc", "history"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("history mode = %v, want 0600", info.Mode().Perm())
	}
}

func historyLines(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(body), "\n"), "\n")
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return lines[len(lines)-1]
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func TestReplSaysWhenANameTakesOverACommand(t *testing.T) {
	got := session(t, "digits = 4\n")
	if !strings.Contains(got, "out of reach") {
		t.Errorf("shadowing a command must be announced:\n%s", got)
	}
	if again := session(t, "x = 4\n"); strings.Contains(again, "out of reach") {
		t.Errorf("an ordinary name must not be announced:\n%s", again)
	}
}

func TestReplDegreesReachTheFloat64Shadow(t *testing.T) {
	got := session(t, "deg\nasin(0.5)\nwhy\n")
	if !strings.Contains(got, "float64: 30") {
		t.Errorf("the shadow must answer in degrees too:\n%s", got)
	}
	if strings.Contains(got, "float64 would answer") {
		t.Errorf("degree mode must not raise a false warning:\n%s", got)
	}
}

func TestReplDegreesReachTheShadowOfTwoArguments(t *testing.T) {
	got := session(t, "deg\natan2(1, 1)\n")
	if !strings.HasPrefix(lastLine(got), "45") {
		t.Errorf("atan2 in degrees:\n%s", got)
	}
	if strings.Contains(got, "float64 would answer") {
		t.Errorf("degree mode must not raise a false warning:\n%s", got)
	}
}

func TestReplRemembersTheResultItPrinted(t *testing.T) {
	got := session(t, "2 + 2\n3 + 3\nwhy\n")
	if !strings.Contains(got, "float64: 6") {
		t.Errorf("why explained an older result:\n%s", got)
	}
}

func TestReplFallsBackToTheDefaultDigits(t *testing.T) {
	statedir(t)
	got := replOutput(t, "sqrt(2)\n", format.Style{})
	if n := countDigits(lastLine(got)); n != format.Digits {
		t.Errorf("a style with no digits printed %d, want %d: %q", n, format.Digits, lastLine(got))
	}
}

func TestReplInfoOnAFunctionYouDefined(t *testing.T) {
	got := session(t, "f(x) = x^2 + 1\ninfo f\n")
	if !strings.Contains(got, "[function]") || !strings.Contains(got, "this session") {
		t.Errorf("info on a function you defined:\n%s", got)
	}
}

func TestReplLookupFindsAFunctionYouDefined(t *testing.T) {
	got := session(t, "wobble(x) = x\n?wob\n")
	if !strings.Contains(got, "names: wobble") {
		t.Errorf("lookup missed a function you defined:\n%s", got)
	}
}

func TestReplDefinitionReportsItself(t *testing.T) {
	got := session(t, "f(x) = x^2 + 1\ng(a, b) = a * b\nf(3)\n")
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	want := []string{"f(x) defined", "g(a, b) defined", "10"}
	if len(lines) != len(want) {
		t.Fatalf("lines = %q, want %q", lines, want)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestDefinitionReportsItselfInOneShot(t *testing.T) {
	out, code := run([]string{"f(x) = x^2 + 1"}, plain, false)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if out != "f(x) defined" {
		t.Errorf("one-shot printed %q, want %q", out, "f(x) defined")
	}
	if strings.TrimSpace(out) == "0" {
		t.Errorf("a definition must not answer 0")
	}
}

func TestDefinitionReportsItselfInJSON(t *testing.T) {
	out, code := run([]string{"f(x) = x^2 + 1"}, plain, true)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, `"definition": "f(x)"`) {
		t.Errorf("JSON is missing the definition:\n%s", out)
	}
	if strings.Contains(out, `"value"`) {
		t.Errorf("a definition must not carry a numeric value:\n%s", out)
	}
}

func TestDefinitionReportsItselfInASheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sheet.calc")
	if err := os.WriteFile(path, []byte("h(x) = x^2\nh(3)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runFile(path, plain)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	if !strings.Contains(out, "= h(x) defined") {
		t.Errorf("sheet is missing the definition:\n%s", out)
	}
	if !strings.Contains(out, "= 9") {
		t.Errorf("the sheet stopped working:\n%s", out)
	}
}

func TestReplUnsetRestoresACommand(t *testing.T) {
	got := session(t, "digits = 4\ndigits 3\nunset digits\ndigits 3\n")
	if !strings.Contains(got, "out of reach until unset digits") {
		t.Errorf("the way out was not named:\n%s", got)
	}
	if !strings.Contains(got, "\n12\n") {
		t.Errorf("a bound name must win while it is bound:\n%s", got)
	}
	if !strings.Contains(got, "the digits command works again") {
		t.Errorf("unset did not say what it gave back:\n%s", got)
	}
	if !strings.Contains(lastLine(got), "digits is 3") {
		t.Errorf("the command did not come back:\n%s", got)
	}
}

func TestReplUnsetRestoresAUnit(t *testing.T) {
	got := session(t, "m = 7\nm * 2\nunset m\nm * 2\n")
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if lines[1] != "14" {
		t.Errorf("a bound name must win while it is bound: %q\n%s", lines[1], got)
	}
	if !strings.Contains(got, "a unit of length") {
		t.Errorf("unset did not say what it gave back:\n%s", got)
	}
	if lastLine(got) != "2 m" {
		t.Errorf("the unit did not come back: %q\n%s", lastLine(got), got)
	}
}

func TestReplUnsetRestoresAConstant(t *testing.T) {
	got := session(t, "h = 3\nh\nunset h\nh\n")
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if lines[1] != "3" {
		t.Errorf("a bound name must win while it is bound: %q\n%s", lines[1], got)
	}
	if !strings.Contains(got, "Planck constant") {
		t.Errorf("unset did not say what it gave back:\n%s", got)
	}
	if !strings.Contains(lastLine(got), "J·s") {
		t.Errorf("the constant did not come back: %q\n%s", lastLine(got), got)
	}
}

func TestReplUnsetForgetsAFunctionToo(t *testing.T) {
	got := session(t, "f(x) = x\nunset f\nf(1)\n")
	if !strings.Contains(got, "error: ") {
		t.Errorf("unset left the function behind:\n%s", got)
	}
}

func TestReplUnsetStaysReachableWhenShadowed(t *testing.T) {
	got := session(t, "unset = 5\nunset unset\nunset + 1\n")
	if !strings.Contains(got, "no longer a name of yours") {
		t.Errorf("the escape hatch was trapped:\n%s", got)
	}
}

func TestReplUnsetRefusesWhatItDoesNotHold(t *testing.T) {
	cases := map[string]string{
		"unset":     "error: unset needs a name",
		"unset zzz": "error: nothing of yours is called zzz",
		"unset km":  "error: nothing of yours is called km",
	}
	for line, want := range cases {
		if got := strings.TrimRight(session(t, line+"\n"), "\n"); got != want {
			t.Errorf("%q said %q, want %q", line, got, want)
		}
	}
}

func TestInfoDescribesEveryAdvertisedFunction(t *testing.T) {
	for _, name := range functions {
		out := session(t, "info "+name)
		if strings.Contains(out, "nothing is known about") {
			t.Errorf("info %s: %q, but help advertises it", name, strings.TrimSpace(out))
		}
		if !strings.Contains(out, name) {
			t.Errorf("info %s said %q", name, strings.TrimSpace(out))
		}
	}
	if out := session(t, "info wibble"); !strings.Contains(out, "nothing is known about wibble") {
		t.Errorf("info wibble = %q", out)
	}
}

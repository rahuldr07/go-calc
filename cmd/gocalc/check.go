package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	gocalc "github.com/rahuldr07/go-calc"
	"github.com/rahuldr07/go-calc/internal/doc"
)

const checkUsage = "usage: gocalc check [--require=exact|proven|estimate] [--update] [--github] file.calc..."

const goldenSuffix = ".golden"

var trustRank = map[gocalc.Status]int{gocalc.Exact: 3, gocalc.Proven: 2, gocalc.Estimate: 1}

type checkSettings struct {
	require gocalc.Status
	update  bool
	github  bool
	digits  int
}

type finding struct {
	file   string
	line   int
	column int
	msg    string
}

func (f finding) text() string {
	return f.file + ":" + strconv.Itoa(f.line) + ": " + f.msg
}

func (f finding) annotation() string {
	at := "file=" + f.file + ",line=" + strconv.Itoa(f.line)
	if f.column > 0 {
		at += ",col=" + strconv.Itoa(f.column)
	}
	return "::error " + at + "::" + strings.ReplaceAll(f.msg, "\n", "%0A")
}

func check(args []string, style settings) (string, int) {
	set, files, err := checkOptions(args, style)
	if err != nil {
		return err.Error(), 2
	}
	var report strings.Builder
	failed := false
	for _, file := range files {
		summary, found, err := checkFile(file, set)
		if err != nil {
			report.WriteString(file + ": " + err.Error() + "\n")
			failed = true
			continue
		}
		for _, f := range found {
			if set.github {
				report.WriteString(f.annotation() + "\n")
			}
			report.WriteString(f.text() + "\n")
		}
		failed = failed || len(found) > 0
		report.WriteString(summary + "\n")
	}
	if failed {
		return strings.TrimRight(report.String(), "\n"), 1
	}
	return strings.TrimRight(report.String(), "\n"), 0
}

func checkOptions(args []string, style settings) (checkSettings, []string, error) {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	require := fs.String("require", string(gocalc.Estimate), "")
	update := fs.Bool("update", false, "")
	github := fs.Bool("github", os.Getenv("GITHUB_ACTIONS") == "true", "")
	if err := fs.Parse(args); err != nil {
		return checkSettings{}, nil, errors.New(err.Error() + "\n" + checkUsage)
	}
	level := gocalc.Status(*require)
	if _, ok := trustRank[level]; !ok {
		return checkSettings{}, nil, errors.New("--require takes exact, proven or estimate\n" + checkUsage)
	}
	if fs.NArg() == 0 {
		return checkSettings{}, nil, errors.New(checkUsage)
	}
	digits := gocalc.DefaultDigits
	if style.digits > 0 {
		digits = style.digits
	}
	return checkSettings{require: level, update: *update, github: *github, digits: digits}, fs.Args(), nil
}

func checkFile(file string, set checkSettings) (string, []finding, error) {
	lines, err := sheetLines(file)
	if err != nil {
		return "", nil, err
	}
	sheet, err := gocalc.EvalSheet(context.Background(), lines, gocalc.Digits(set.digits))
	if err != nil {
		return "", nil, err
	}
	var found []finding
	counts := map[gocalc.Status]int{}
	for _, r := range sheet.Results {
		if doc.Blank(r.Source) {
			continue
		}
		counts[r.Status]++
		switch {
		case r.Refusal != nil:
			found = append(found, finding{file: file, line: r.Line, column: r.Refusal.Column, msg: "refused: " + r.Refusal.Reason})
		case r.Status == gocalc.Held:
		case trustRank[r.Status] < trustRank[set.require]:
			found = append(found, finding{file: file, line: r.Line, msg: r.Source + " is " + string(r.Status) + ", but " + string(set.require) + " is required"})
		}
	}
	golden := file + goldenSuffix
	recorded := goldenText(sheet)
	switch {
	case set.update:
		if err := os.WriteFile(golden, []byte(recorded), 0o644); err != nil {
			return "", nil, err
		}
	default:
		want, err := os.ReadFile(golden)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return "", nil, err
		default:
			found = append(found, goldenDrift(file, string(want), recorded)...)
		}
	}
	return summaryOf(file, counts, len(found), set.update), found, nil
}

func goldenText(sheet gocalc.Sheet) string {
	var b strings.Builder
	for _, r := range sheet.Results {
		if doc.Blank(r.Source) {
			continue
		}
		answer := r.Text
		if r.Refusal != nil {
			answer = r.Refusal.Reason
		}
		fmt.Fprintf(&b, "%d\t%s\t%s\n", r.Line, r.Status, answer)
	}
	return b.String()
}

func goldenDrift(file, want, got string) []finding {
	expected := goldenLines(want)
	actual := goldenLines(got)
	var found []finding
	for line, was := range expected {
		now, ok := actual[line]
		switch {
		case !ok:
			found = append(found, finding{file: file, line: line, msg: "the golden file expects " + was + " here, but the line no longer gives an answer"})
		case now != was:
			found = append(found, finding{file: file, line: line, msg: "changed from " + was + " to " + now})
		}
	}
	for line, now := range actual {
		if _, ok := expected[line]; !ok {
			found = append(found, finding{file: file, line: line, msg: "new answer " + now + " is not in the golden file; run gocalc check --update to record it"})
		}
	}
	sortFindings(found)
	return found
}

func goldenLines(text string) map[int]string {
	out := map[int]string{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		n, rest, ok := strings.Cut(sc.Text(), "\t")
		if !ok {
			continue
		}
		line, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		status, answer, _ := strings.Cut(rest, "\t")
		out[line] = answer + " (" + status + ")"
	}
	return out
}

func sortFindings(found []finding) {
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j].line < found[j-1].line; j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
}

func summaryOf(file string, counts map[gocalc.Status]int, problems int, updated bool) string {
	var parts []string
	total := 0
	for _, s := range []gocalc.Status{gocalc.Exact, gocalc.Proven, gocalc.Estimate, gocalc.Held, gocalc.Refused} {
		if counts[s] > 0 {
			word := string(s)
			if s == gocalc.Refused {
				word = "refused"
			}
			parts = append(parts, strconv.Itoa(counts[s])+" "+word)
			total += counts[s]
		}
	}
	verdict := "ok"
	switch {
	case updated && problems == 0:
		verdict = "ok, golden file written"
	case updated:
		verdict = "golden file written, " + counted(problems, "problem")
	case problems > 0:
		verdict = counted(problems, "problem")
	}
	return file + ": " + counted(total, "line") + " (" + strings.Join(parts, ", ") + ") — " + verdict
}

func counted(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

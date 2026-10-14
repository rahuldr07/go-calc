package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/steps"
)

const stepIndent = "     "

func showSteps(args []string, style format.Style, asJSON bool) (string, int, bool) {
	d, err := steps.Of(strings.Join(args, " "), style.Digits)
	if err != nil {
		return err.Error(), 1, false
	}
	if asJSON {
		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err.Error(), 1, false
		}
		code := 0
		if d.Error != "" {
			code = 1
		}
		return string(b), code, true
	}
	if d.Error != "" {
		return d.Error, 1, false
	}
	return stepsText(d), 0, false
}

func stepsText(d steps.Derivation) string {
	var b strings.Builder
	b.WriteString(d.Input + "\n")
	if len(d.Steps) > 0 {
		b.WriteString("\n")
	}
	for _, s := range d.Steps {
		if s.Explain != "" {
			b.WriteString(strconv.Itoa(s.N) + ". " + s.Explain + "\n")
			b.WriteString(stepIndent + s.Expr + " = " + s.Result + "\n")
			for _, line := range s.Work {
				b.WriteString(stepIndent + "  " + line + "\n")
			}
		} else {
			b.WriteString(strconv.Itoa(s.N) + ". " + s.Expr + "\n")
			if len(s.From) > 0 {
				b.WriteString(stepIndent + "uses " + joinAnd(s.From) + "\n")
			}
			b.WriteString(stepIndent + "= " + s.Result + "\n")
		}
		if s.Rule != "" && len(s.Work) == 0 && !(s.Explain != "" && (strings.HasPrefix(s.Rule, "unit:") || strings.HasPrefix(s.Rule, "conversion"))) {
			b.WriteString(stepIndent + "rule: " + s.Rule + "\n")
		}
		if s.Source != "" {
			b.WriteString(stepIndent + "source: " + s.Source + "\n")
		}
	}
	if d.Note != "" {
		b.WriteString("\n" + d.Note + "\n")
	}
	for _, line := range d.Summary {
		b.WriteString("\n" + line + "\n")
	}
	b.WriteString("\nanswer: " + strings.ReplaceAll(d.Answer, "\n", "\n"+stepIndent) + "\n")
	b.WriteString(trustLine(d))
	return b.String()
}

func trustLine(d steps.Derivation) string {
	line := "trust:  " + d.Status
	switch {
	case d.Status == doc.StatusExact && d.Digits > 0:
		line += ", shown to " + digitCount(d.Digits)
	case d.Digits > 0:
		line += ", " + digitCount(d.Digits) + " guaranteed"
	}
	if d.Bound != "" {
		line += "\nbound:  " + d.Bound
	}
	return line
}

func joinAnd(parts []string) string {
	if len(parts) < 3 {
		return strings.Join(parts, " and ")
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}

func digitCount(n int) string {
	if n == 1 {
		return "1 digit"
	}
	return strconv.Itoa(n) + " digits"
}

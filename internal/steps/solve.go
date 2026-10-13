package steps

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/prov"
)

const (
	linearRule       = "exact linear solution"
	bisectionRule    = "bisection on proven sign-change brackets"
	quadraticFormula = "x = (-b ± sqrt(b^2 - 4ac)) / 2a"
	linearFormula    = "x = -b/a"
	arrow            = "  →  "

	termVariable     = "variable"
	termDiscriminant = "discriminant"
	termSearched     = "searched"
	termExact        = "exact root"
	termBracket      = "bracket"
	termBisections   = "bisections"
)

func solveRule(n prov.NodeOut, result string) string {
	terms := termsByName(n.Terms)
	switch n.Detail {
	case quadraticRule:
		return quadraticRuleText(terms, result)
	case linearRule:
		return linearRuleText(terms, result)
	case bisectionRule:
		return bisectionRuleText(n, result)
	}
	return n.Detail
}

func termsByName(terms []prov.Term) map[string]string {
	out := make(map[string]string, len(terms))
	for _, t := range terms {
		out[t.Name] = t.Value
	}
	return out
}

func quadraticRuleText(t map[string]string, result string) string {
	a, b, c, disc := t["a"], t["b"], t["c"], t[termDiscriminant]
	if a == "" || b == "" || c == "" {
		return quadraticRule + ": " + quadraticFormula
	}
	v := variableOf(t)
	given := "a = " + a + ", b = " + b + ", c = " + c
	substituted := v + " = (-" + factor(b) + " ± sqrt(" + factor(b) + "^2 - 4·" + factor(a) + "·" + factor(c) + ")) / (2·" + factor(a) + ")"
	return quadraticRule + ": " + quadraticFormula + ", with " + given + ": " + substituted + arrow + result + " (discriminant " + disc + ")"
}

func linearRuleText(t map[string]string, result string) string {
	a, b := t["a"], t["b"]
	if a == "" || b == "" {
		return linearRule + ": " + linearFormula
	}
	v := variableOf(t)
	given := "a = " + a + ", b = " + b
	substituted := v + " = -" + factor(b) + "/" + factor(a)
	return linearRule + ": a·" + v + " + b = 0, so " + linearFormula + ", with " + given + ": " + substituted + arrow + result
}

func bisectionRuleText(n prov.NodeOut, result string) string {
	t := termsByName(n.Terms)
	out := bisectionRule
	if searched := t[termSearched]; searched != "" {
		out += " over " + searched
	}
	out += ": no closed formula applies"
	var found []string
	var pending string
	for _, term := range n.Terms {
		switch term.Name {
		case termExact:
			found = append(found, "the grid hit a root exactly at "+term.Value)
		case termBracket:
			pending = term.Value
		case termBisections:
			found = append(found, "bisected "+term.Value+" times down to the bracket "+pending)
		}
	}
	if len(found) > 0 {
		out += "; " + strings.Join(found, "; ")
	}
	return out + arrow + result
}

func variableOf(t map[string]string) string {
	if v := t[termVariable]; v != "" {
		return v
	}
	return "x"
}

func factor(s string) string {
	if strings.HasPrefix(s, "-") || strings.Contains(s, "/") {
		return "(" + s + ")"
	}
	return s
}

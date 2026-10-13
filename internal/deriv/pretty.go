package deriv

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/parse"
)

const (
	levelSum = iota + 1
	levelProduct
	levelSign
	levelPower
	levelPostfix
	levelAtom
)

func Pretty(n parse.Node) string {
	text, _ := pretty(n)
	return text
}

func level(op string) int {
	switch op {
	case "+", "-":
		return levelSum
	case "*", "·", "×", "/", "÷", "mod":
		return levelProduct
	case "^", "**":
		return levelPower
	}
	return levelAtom
}

func wrap(text string, have, need int) string {
	if have < need {
		return "(" + text + ")"
	}
	return text
}

func pretty(n parse.Node) (string, int) {
	switch t := n.(type) {
	case *parse.Unary:
		x, at := pretty(t.X)
		if t.Op == "%" || t.Op == "!" {
			return wrap(x, at, levelAtom) + t.Op, levelPostfix
		}
		return t.Op + wrap(x, at, levelProduct), levelSign
	case *parse.Binary:
		return binaryText(t)
	case *parse.Call:
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			parts[i] = t.Key(i) + Pretty(a)
		}
		switch t.Fn {
		case parse.ListFn:
			return "[" + strings.Join(parts, ", ") + "]", levelAtom
		case parse.IndexFn:
			return wrapIndexed(t.Args[0]) + "[" + parts[1] + "]", levelAtom
		case parse.RangeFn:
			return strings.Join(parts, parse.RangeFn), levelSum
		}
		return t.Fn + "(" + strings.Join(parts, ", ") + ")", levelAtom
	}
	return n.String(), levelAtom
}

func wrapIndexed(n parse.Node) string {
	text, at := pretty(n)
	return wrap(text, at, levelAtom)
}

func binaryText(t *parse.Binary) (string, int) {
	at := level(t.Op)
	if at == levelAtom {
		return t.String(), levelAtom
	}
	l, lat := pretty(t.L)
	r, rat := pretty(t.R)
	if at == levelPower {
		return wrap(l, lat, levelPostfix) + "^" + wrap(r, rat, levelPower), levelPower
	}
	rightNeed := at
	if t.Op != "+" && t.Op != "*" && t.Op != "·" && t.Op != "×" {
		rightNeed = at + 1
	}
	op := " " + t.Op + " "
	if t.Op == "·" || t.Op == "×" {
		op = " * "
	}
	return wrap(l, lat, at) + op + wrap(r, rat, rightNeed), at
}

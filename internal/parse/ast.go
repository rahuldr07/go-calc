package parse

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/lex"
)

type Node interface {
	Pos() lex.Pos
	String() string
}

type NumLit struct {
	Text    string
	P       lex.Pos
	Grouped bool
}

func (n *NumLit) Pos() lex.Pos   { return n.P }
func (n *NumLit) String() string { return n.Text }

type Name struct {
	Text    string
	P       lex.Pos
	Grouped bool
}

func (n *Name) Pos() lex.Pos   { return n.P }
func (n *Name) String() string { return n.Text }

type Unary struct {
	Op      string
	X       Node
	P       lex.Pos
	Grouped bool
}

func (n *Unary) Pos() lex.Pos { return n.P }
func (n *Unary) String() string {
	if n.Op == "%" || n.Op == "!" {
		return "(" + n.X.String() + " " + n.Op + ")"
	}
	return "(" + n.Op + n.X.String() + ")"
}

type Binary struct {
	Op       string
	L, R     Node
	P        lex.Pos
	Implicit bool
	Grouped  bool
}

func (n *Binary) Pos() lex.Pos   { return n.P }
func (n *Binary) String() string { return "(" + n.L.String() + " " + n.Op + " " + n.R.String() + ")" }

type Call struct {
	Fn      string
	Args    []Node
	Keys    []string
	P       lex.Pos
	Grouped bool
}

func (n *Call) Pos() lex.Pos { return n.P }
func (n *Call) String() string {
	parts := make([]string, len(n.Args))
	for i, a := range n.Args {
		parts[i] = n.Key(i) + a.String()
	}
	return n.Fn + "(" + strings.Join(parts, ", ") + ")"
}

func (n *Call) Key(i int) string {
	if i >= len(n.Keys) || n.Keys[i] == "" {
		return ""
	}
	return n.Keys[i] + " = "
}

type Assign struct {
	Name   string
	Params []string
	Body   Node
	P      lex.Pos
}

func (n *Assign) Pos() lex.Pos { return n.P }
func (n *Assign) String() string {
	if len(n.Params) == 0 {
		return n.Name + " = " + n.Body.String()
	}
	return n.Name + "(" + strings.Join(n.Params, ", ") + ") = " + n.Body.String()
}

type Note struct {
	Pos lex.Pos
	Msg string
}

func regroupings(n Node, notes []Note) []Note {
	if decidedByEvaluator(n) {
		return notes
	}
	if at, ok := regroupedSite(n); ok {
		return append(notes, Note{Pos: at, Msg: "read as " + Reading(n)})
	}
	return notes
}

func decidedByEvaluator(n Node) bool {
	switch t := n.(type) {
	case *Unary:
		return decidedByEvaluator(t.X)
	case *Binary:
		if t.Implicit && InDenominator(t.L) {
			return true
		}
		return decidedByEvaluator(t.L) || decidedByEvaluator(t.R)
	case *Call:
		for _, a := range t.Args {
			if decidedByEvaluator(a) {
				return true
			}
		}
	case *Assign:
		return decidedByEvaluator(t.Body)
	}
	return false
}

func regroupedSite(n Node) (lex.Pos, bool) {
	switch t := n.(type) {
	case *Unary:
		return regroupedSite(t.X)
	case *Binary:
		if t.Implicit && InDenominator(t.L) {
			return lex.Pos{}, false
		}
		if regrouped(t) {
			return t.P, true
		}
		if at, ok := regroupedSite(t.L); ok {
			return at, true
		}
		return regroupedSite(t.R)
	case *Call:
		for _, a := range t.Args {
			if at, ok := regroupedSite(a); ok {
				return at, true
			}
		}
	case *Assign:
		return regroupedSite(t.Body)
	}
	return lex.Pos{}, false
}

func InDenominator(n Node) bool {
	b, ok := n.(*Binary)
	if !ok {
		return false
	}
	if b.Op == "/" || b.Op == "÷" {
		return !b.Grouped && !isGrouped(b.R)
	}
	return b.Implicit && !b.Grouped && InDenominator(b.L)
}

func isGrouped(n Node) bool {
	switch t := n.(type) {
	case *Binary:
		return t.Grouped
	case *Unary:
		return t.Grouped
	case *Name:
		return t.Grouped
	case *NumLit:
		return t.Grouped
	case *Call:
		return t.Grouped
	}
	return false
}

func MarkGrouped(n Node) {
	switch t := n.(type) {
	case *Binary:
		t.Grouped = true
	case *Unary:
		t.Grouped = true
	case *Name:
		t.Grouped = true
	case *NumLit:
		t.Grouped = true
	case *Call:
		t.Grouped = true
	}
}

func regrouped(t *Binary) bool {
	r, ok := t.R.(*Binary)
	if !ok || r.Grouped {
		return false
	}
	return t.Implicit && r.Op == "^"
}

func Reading(n Node) string { return reading(n) }

func reading(n Node) string {
	switch t := n.(type) {
	case *Unary:
		if t.Op == "%" || t.Op == "!" {
			return bracket(t.X) + " " + t.Op
		}
		return t.Op + bracket(t.X)
	case *Binary:
		if t.Implicit {
			return juxtaposed(bracket(t.L), bracket(t.R))
		}
		return bracket(t.L) + operatorSymbol(t.Op) + bracket(t.R)
	case *Call:
		parts := make([]string, len(t.Args))
		for i, a := range t.Args {
			parts[i] = t.Key(i) + reading(a)
		}
		return t.Fn + "(" + strings.Join(parts, ", ") + ")"
	case *Assign:
		if len(t.Params) == 0 {
			return t.Name + " = " + reading(t.Body)
		}
		return t.Name + "(" + strings.Join(t.Params, ", ") + ") = " + reading(t.Body)
	}
	return n.String()
}

func juxtaposed(left, right string) string {
	if endsInName(left) && strings.HasPrefix(right, "(") {
		return "(" + left + ") " + right
	}
	return left + " " + right
}

func endsInName(text string) bool {
	toks, err := lex.Scan(text)
	if err != nil {
		return true
	}
	for len(toks) > 0 && toks[len(toks)-1].Kind == lex.EOF {
		toks = toks[:len(toks)-1]
	}
	if len(toks) == 0 {
		return true
	}
	return toks[len(toks)-1].Kind == lex.Ident
}

func bracket(n Node) string {
	switch n.(type) {
	case *Binary, *Unary:
		return "(" + reading(n) + ")"
	}
	return reading(n)
}

func operatorSymbol(op string) string {
	switch op {
	case "*", "×", "·":
		return "·"
	case "÷":
		return "/"
	case "/", "^":
		return op
	case "+-":
		return " ± "
	}
	return " " + op + " "
}

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
	Text string
	P    lex.Pos
}

func (n *NumLit) Pos() lex.Pos   { return n.P }
func (n *NumLit) String() string { return n.Text }

type Name struct {
	Text string
	P    lex.Pos
}

func (n *Name) Pos() lex.Pos   { return n.P }
func (n *Name) String() string { return n.Text }

type Unary struct {
	Op string
	X  Node
	P  lex.Pos
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
	Fn   string
	Args []Node
	Keys []string
	P    lex.Pos
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
	return append(notes, collectRegroupings(n, nil)...)
}

func collectRegroupings(n Node, out []Note) []Note {
	switch t := n.(type) {
	case *Unary:
		return collectRegroupings(t.X, out)
	case *Call:
		for _, a := range t.Args {
			out = collectRegroupings(a, out)
		}
		return out
	case *Assign:
		return collectRegroupings(t.Body, out)
	case *Binary:
		if t.Implicit && InDenominator(t.L) {
			return out
		}
		if regrouped(t) {
			return append(out, Note{Pos: t.P, Msg: "read as " + Reading(t)})
		}
		out = collectRegroupings(t.L, out)
		return collectRegroupings(t.R, out)
	}
	return out
}

func InDenominator(n Node) bool {
	b, ok := n.(*Binary)
	if !ok {
		return false
	}
	if b.Op == "/" || b.Op == "÷" {
		return !b.Grouped
	}
	return b.Implicit && InDenominator(b.L)
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
	t, ok := n.(*Binary)
	if !ok {
		return n.String()
	}
	return bracket(t.L) + operatorSymbol(t.Op) + bracket(t.R)
}

func bracket(n Node) string {
	if _, ok := n.(*Binary); ok {
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

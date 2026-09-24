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
}

func (n *Binary) Pos() lex.Pos   { return n.P }
func (n *Binary) String() string { return "(" + n.L.String() + " " + n.Op + " " + n.R.String() + ")" }

type Call struct {
	Fn   string
	Args []Node
	P    lex.Pos
}

func (n *Call) Pos() lex.Pos { return n.P }
func (n *Call) String() string {
	parts := make([]string, len(n.Args))
	for i, a := range n.Args {
		parts[i] = a.String()
	}
	return n.Fn + "(" + strings.Join(parts, ", ") + ")"
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

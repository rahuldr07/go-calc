package lex

import "fmt"

type Kind int

const (
	Num Kind = iota
	Ident
	Op
	LParen
	RParen
	Comma
	Comment
	EOF
)

func (k Kind) String() string {
	switch k {
	case Num:
		return "number"
	case Ident:
		return "name"
	case Op:
		return "operator"
	case LParen:
		return "("
	case RParen:
		return ")"
	case Comma:
		return ","
	case Comment:
		return "comment"
	default:
		return "end of input"
	}
}

type Pos struct {
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

type Token struct {
	Kind Kind
	Text string
	Pos  Pos
}

type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Pos, e.Msg) }

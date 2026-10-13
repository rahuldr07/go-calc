package eval

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/deriv"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/value"
)

func (e *Env) derivCall(t *parse.Call, p *pass) (value.Value, error) {
	if len(t.Args) != 2 && len(t.Args) != 3 {
		return value.Value{}, &lex.Error{Pos: t.Pos(), Msg: "deriv takes an expression, a variable, and an optional point"}
	}
	variable, ok := t.Args[1].(*parse.Name)
	if !ok {
		return value.Value{}, &lex.Error{Pos: t.Args[1].Pos(), Msg: "deriv needs a plain variable name as its second argument"}
	}
	derived, err := deriv.Differentiate(t.Args[0], variable.Text)
	if err != nil {
		return value.Value{}, &lex.Error{Pos: t.Pos(), Msg: err.Error()}
	}
	simplified := deriv.Simplify(derived)
	if len(t.Args) == 2 {
		return e.symbolicDeriv(t, p, simplified, variable.Text)
	}
	at, err := e.eval(t.Args[2], p)
	if err != nil {
		return value.Value{}, err
	}
	if !at.IsScalar() {
		return value.Value{}, &lex.Error{Pos: t.Args[2].Pos(), Msg: "deriv evaluates at a plain number"}
	}
	inner := e.child()
	inner.Set(variable.Text, at)
	out, err := inner.eval(simplified, p)
	if err != nil {
		return value.Value{}, err
	}
	e.record(t, p, prov.Node{Kind: prov.Function, Label: "deriv", Detail: "d/d" + variable.Text + " " + deriv.Pretty(simplified)})
	out.Prov = p.idOf(t)
	return out, nil
}

func (e *Env) symbolicDeriv(t *parse.Call, p *pass, derived parse.Node, variable string) (value.Value, error) {
	if derived == nil {
		return value.Value{}, errors.New("the derivative could not be formed")
	}
	e.record(t, p, prov.Node{Kind: prov.Function, Label: "deriv", Detail: "d/d" + variable})
	v := value.Symbolic(deriv.Pretty(derived))
	v.Prov = p.idOf(t)
	return v, nil
}

package eval

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

type undecidedError struct{ failure *lex.Error }

func (u undecidedError) Error() string { return u.failure.Error() }

var comparisons = map[string]bool{"<": true, ">": true, "<=": true, ">=": true, "==": true, "!=": true}

func (e *Env) compare(t *parse.Binary, l, r value.Value, p *pass) (value.Value, error) {
	if l.IsMeasured() || r.IsMeasured() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot compare values that carry an uncertainty: their spread is statistical, so no ordering is proven"}
	}
	if l.IsBoolean() || r.IsBoolean() {
		return compareLogical(t, l, r)
	}
	if l.Dim != r.Dim {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot compare " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")"}
	}
	order, err := e.order(t, l, r, p)
	if err != nil {
		return value.Value{}, err
	}
	return value.Boolean(holdsFor(t.Op, order)), nil
}

func compareLogical(t *parse.Binary, l, r value.Value) (value.Value, error) {
	if !l.IsBoolean() || !r.IsBoolean() || (t.Op != "==" && t.Op != "!=") {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a true/false value can only be compared with == or != against another true/false value"}
	}
	same := l.Truth == r.Truth
	return value.Boolean(same == (t.Op == "==")), nil
}

func (e *Env) order(t *parse.Binary, l, r value.Value, p *pass) (int, error) {
	prec := p.prec()
	left, right := l.SI(prec), r.SI(prec)
	if left.IsExact() && right.IsExact() {
		return left.Exact().Cmp(right.Exact()), nil
	}
	sign, decided := left.Sub(right, prec).Ball(prec).Sign()
	if !decided {
		return 0, undecidedError{&lex.Error{Pos: t.P, Msg: "undecided at this precision: the two sides overlap, so " + t.Op + " is neither proven nor refuted"}}
	}
	return sign, nil
}

func holdsFor(op string, order int) bool {
	switch op {
	case "<":
		return order < 0
	case ">":
		return order > 0
	case "<=":
		return order <= 0
	case ">=":
		return order >= 0
	case "==":
		return order == 0
	}
	return order != 0
}

func isUndecided(err error) bool {
	var u undecidedError
	return errors.As(err, &u)
}

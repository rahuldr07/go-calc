package eval

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/cplx"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const imaginaryUnit = "i"

var errNotComplex = errors.New("that value cannot take part in complex arithmetic")

func imaginary() value.Value {
	return value.Complex(cplx.Rect(num.NumExact(num.FromInt(0)), num.NumExact(num.FromInt(1))))
}

func complexOf(v value.Value) (cplx.Complex, bool) {
	if v.Cx != nil {
		return *v.Cx, true
	}
	if v.IsScalar() && !v.IsMeasured() {
		return cplx.Real(v.Num), true
	}
	return cplx.Complex{}, false
}

func eitherComplex(l, r value.Value) bool { return l.Cx != nil || r.Cx != nil }

func settle(c cplx.Complex) value.Value {
	if c.IsReal() {
		return value.ScalarNum(c.Re())
	}
	return value.Complex(c)
}

func complexBinary(t *parse.Binary, l, r value.Value, prec uint) (value.Value, error) {
	refuse := func(why string) (value.Value, error) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: why}
	}
	if l.Unit != "" || r.Unit != "" {
		return refuse("a complex number cannot carry a unit")
	}
	if l.IsMeasured() || r.IsMeasured() {
		return refuse("a complex number cannot carry a measured uncertainty")
	}
	a, ok := complexOf(l)
	if !ok {
		return refuse("that value cannot take part in complex arithmetic")
	}
	b, ok := complexOf(r)
	if !ok {
		return refuse("that value cannot take part in complex arithmetic")
	}
	switch t.Op {
	case "+":
		return settle(a.Add(b, prec)), nil
	case "-":
		return settle(a.Sub(b, prec)), nil
	case "*", "·", "×":
		return settle(a.Mul(b, prec)), nil
	case "/", "÷":
		out, err := a.Div(b, prec)
		if err != nil {
			return refuse(err.Error())
		}
		return settle(out), nil
	case "^", "**":
		k, ok := complexWholePower(b)
		if !ok {
			return refuse("a complex number can only be raised to a whole power")
		}
		out, err := a.Pow(k, prec)
		if err != nil {
			return refuse(err.Error())
		}
		return settle(out), nil
	}
	return refuse("that operator does not apply to complex numbers")
}

func complexWholePower(c cplx.Complex) (int, bool) {
	if !c.IsReal() || !c.Re().IsExact() {
		return 0, false
	}
	r := c.Re().Exact().Rat()
	if !r.IsInt() {
		return 0, false
	}
	if !r.Num().IsInt64() {
		return 0, false
	}
	k := r.Num().Int64()
	if k > 1<<20 || k < -(1<<20) {
		return 0, false
	}
	return int(k), true
}

func complexSqrt(v value.Value, prec uint) (value.Value, error) {
	c, ok := complexOf(v)
	if !ok {
		return value.Value{}, errNotComplex
	}
	out, err := cplx.Sqrt(c, prec)
	if err != nil {
		return value.Value{}, err
	}
	return settle(out), nil
}

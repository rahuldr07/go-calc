package eval

import (
	"errors"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

var ErrNeedsBalls = errors.New("this needs ball arithmetic, which is not implemented yet")

func (e *Env) Eval(n parse.Node) (value.Value, error) {
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
		}
		return value.Scalar(x), nil
	case *parse.Name:
		return e.evalName(t)
	case *parse.Assign:
		return e.evalAssign(t)
	case *parse.Call:
		return e.evalCall(t)
	case *parse.Unary:
		return e.evalUnary(t)
	case *parse.Binary:
		return e.evalBinary(t)
	}
	return value.Value{}, errors.New("unsupported expression")
}

func (e *Env) evalName(t *parse.Name) (value.Value, error) {
	if v, ok := e.vars[t.Text]; ok {
		return v, nil
	}
	if c, ok := consts.Lookup(t.Text); ok {
		if u, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
			return value.WithUnit(c.Value, u), nil
		}
		v := value.Scalar(c.Value)
		v.Unit = c.Unit
		return v, nil
	}
	if u, ok := units.Lookup(t.Text); ok {
		return value.WithUnit(num.FromInt(1), u), nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unknown name " + t.Text}
}

func (e *Env) evalAssign(t *parse.Assign) (value.Value, error) {
	if len(t.Params) > 0 {
		e.funcs[t.Name] = fn{params: t.Params, body: t.Body}
		return value.Scalar(num.FromInt(0)), nil
	}
	v, err := e.Eval(t.Body)
	if err != nil {
		return value.Value{}, err
	}
	e.vars[t.Name] = v
	return v, nil
}

func (e *Env) evalCall(t *parse.Call) (value.Value, error) {
	if f, ok := e.funcs[t.Fn]; ok {
		if len(f.params) != len(t.Args) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "wrong number of arguments for " + t.Fn}
		}
		inner := e.child()
		for i, p := range f.params {
			a, err := e.Eval(t.Args[i])
			if err != nil {
				return value.Value{}, err
			}
			inner.vars[p] = a
		}
		return inner.Eval(f.body)
	}
	if v, err, handled := e.builtin(t); handled {
		return v, err
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: t.Fn + ": " + ErrNeedsBalls.Error()}
}

func (e *Env) builtin(t *parse.Call) (value.Value, error, bool) {
	args := make([]value.Value, len(t.Args))
	for i, a := range t.Args {
		v, err := e.Eval(a)
		if err != nil {
			return value.Value{}, err, true
		}
		args[i] = v
	}
	fail := func(msg string) (value.Value, error, bool) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: msg}, true
	}
	switch t.Fn {
	case "abs":
		if len(args) != 1 {
			return fail("abs takes one argument")
		}
		v := args[0]
		if v.Num.Sign() < 0 {
			v.Num = v.Num.Neg()
		}
		return v, nil, true
	case "sqrt":
		if len(args) != 1 {
			return fail("sqrt takes one argument")
		}
		if !args[0].IsScalar() {
			return fail("sqrt of a value with units needs ball arithmetic")
		}
		if r, ok := num.Root(args[0].Num, 2); ok {
			return value.Scalar(r), nil, true
		}
		return fail("sqrt: " + ErrNeedsBalls.Error())
	case "root":
		if len(args) != 2 || !args[1].Num.IsInt() {
			return fail("root takes a value and a whole degree")
		}
		if r, ok := num.Root(args[0].Num, int(args[1].Num.Rat().Num().Int64())); ok {
			return value.Scalar(r), nil, true
		}
		return fail("root: " + ErrNeedsBalls.Error())
	case "floor", "ceil", "round":
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		return value.Scalar(roundTo(args[0].Num, t.Fn)), nil, true
	case "gcd", "lcm":
		if len(args) != 2 || !args[0].Num.IsInt() || !args[1].Num.IsInt() {
			return fail(t.Fn + " takes two whole numbers")
		}
		a := args[0].Num.Rat().Num()
		b := args[1].Num.Rat().Num()
		g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(a), new(big.Int).Abs(b))
		if t.Fn == "gcd" {
			return value.Scalar(num.FromRat(new(big.Rat).SetInt(g))), nil, true
		}
		if g.Sign() == 0 {
			return value.Scalar(num.FromInt(0)), nil, true
		}
		l := new(big.Int).Div(new(big.Int).Abs(new(big.Int).Mul(a, b)), g)
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(l))), nil, true
	case "isprime":
		if len(args) != 1 || !args[0].Num.IsInt() {
			return fail("isprime takes a whole number")
		}
		if args[0].Num.Rat().Num().ProbablyPrime(40) {
			return value.Scalar(num.FromInt(1)), nil, true
		}
		return value.Scalar(num.FromInt(0)), nil, true
	}
	return value.Value{}, nil, false
}

func roundTo(x num.Exact, mode string) num.Exact {
	r := x.Rat()
	q, rem := new(big.Int).QuoRem(r.Num(), r.Denom(), new(big.Int))
	if rem.Sign() == 0 {
		return num.FromRat(new(big.Rat).SetInt(q))
	}
	switch mode {
	case "floor":
		if r.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		}
	case "ceil":
		if r.Sign() > 0 {
			q.Add(q, big.NewInt(1))
		}
	default:
		half := new(big.Rat).SetFrac64(1, 2)
		shifted := new(big.Rat).Add(r, half)
		f := new(big.Int).Quo(shifted.Num(), shifted.Denom())
		if shifted.Sign() < 0 && new(big.Int).Rem(shifted.Num(), shifted.Denom()).Sign() != 0 {
			f.Sub(f, big.NewInt(1))
		}
		q = f
	}
	return num.FromRat(new(big.Rat).SetInt(q))
}

func (e *Env) evalUnary(t *parse.Unary) (value.Value, error) {
	x, err := e.Eval(t.X)
	if err != nil {
		return value.Value{}, err
	}
	switch t.Op {
	case "-":
		x.Num = x.Num.Neg()
		return x, nil
	case "%":
		if !x.IsScalar() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "percent needs a plain number"}
		}
		q, err := x.Num.Div(num.FromInt(100))
		if err != nil {
			return value.Value{}, err
		}
		return value.Scalar(q), nil
	case "!":
		if !x.IsScalar() || !x.Num.IsInt() || x.Num.Sign() < 0 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "factorial needs a whole number that is not negative"}
		}
		n := x.Num.Rat().Num()
		if n.Cmp(big.NewInt(20000)) > 0 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "factorial argument too large"}
		}
		f := new(big.Int).MulRange(1, n.Int64())
		if n.Sign() == 0 {
			f = big.NewInt(1)
		}
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(f))), nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unsupported operator " + t.Op}
}

func (e *Env) evalBinary(t *parse.Binary) (value.Value, error) {
	if t.Implicit {
		if v, err, handled := e.implicitUnit(t); handled {
			return v, err
		}
	}
	if t.Op == "to" || t.Op == "in" {
		return e.convert(t)
	}
	l, err := e.Eval(t.L)
	if err != nil {
		return value.Value{}, err
	}
	r, err := e.Eval(t.R)
	if err != nil {
		return value.Value{}, err
	}
	switch t.Op {
	case "+", "-":
		if l.IsAffine() || r.IsAffine() {
			return affineAddSub(t, l, r)
		}
		return addSub(t, l, r)
	case "*", "×", "·":
		return combine(l, r, l.SI().Mul(r.SI()), l.Dim.Add(r.Dim), "·"), nil
	case "/", "÷":
		q, err := l.SI().Div(r.SI())
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "division by zero"}
		}
		return combine(l, r, q, l.Dim.Sub(r.Dim), "/"), nil
	case "mod":
		if !l.IsScalar() || !r.IsScalar() || !l.Num.IsInt() || !r.Num.IsInt() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "mod needs whole numbers"}
		}
		if r.Num.Sign() == 0 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "division by zero"}
		}
		m := new(big.Int).Mod(l.Num.Rat().Num(), r.Num.Rat().Num())
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(m))), nil
	case "^":
		if !r.IsScalar() || !r.Num.IsInt() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "only whole exponents are supported until ball arithmetic lands"}
		}
		exp64 := r.Num.Rat().Num().Int64()
		if exp64 > 100000 || exp64 < -100000 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "exponent too large"}
		}
		exp := int(exp64)
		out := l
		out.Num = l.Num.Pow(exp)
		out.Dim = l.Dim.Scale(exp)
		out.Factor = l.FactorOrOne().Pow(exp)
		if exp != 1 && l.Unit != "" {
			out.Unit = l.Unit + "^" + r.Num.String()
		}
		return out, nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unsupported operator " + t.Op}
}

func (e *Env) implicitUnit(t *parse.Binary) (value.Value, error, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return value.Value{}, nil, false
	}
	u, ok := units.Lookup(name.Text)
	if !ok {
		return value.Value{}, nil, false
	}
	l, err := e.Eval(t.L)
	if err != nil {
		return value.Value{}, err, true
	}
	if !l.IsScalar() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot attach " + name.Text + " to a value that already has a unit"}, true
	}
	if u.Dim.IsNone() {
		return value.Scalar(l.Num.Mul(u.Factor)), nil, true
	}
	return value.WithUnit(l.Num, u), nil, true
}

func addSub(t *parse.Binary, l, r value.Value) (value.Value, error) {
	if r.IsScalar() && isPercentNode(t.R) {
		one := num.FromInt(1)
		scale := one.Add(r.Num)
		if t.Op == "-" {
			scale = one.Sub(r.Num)
		}
		out := l
		out.Num = l.Num.Mul(scale)
		return out, nil
	}
	if l.Dim != r.Dim {
		return value.Value{}, errors.New("cannot " + verbFor(t.Op) + " " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")")
	}
	rInL, err := r.SI().Div(l.FactorOrOne())
	if err != nil {
		return value.Value{}, err
	}
	out := l
	if t.Op == "-" {
		out.Num = l.Num.Sub(rInL)
	} else {
		out.Num = l.Num.Add(rInL)
	}
	return out, nil
}

func affineAddSub(t *parse.Binary, l, r value.Value) (value.Value, error) {
	if l.Dim != r.Dim {
		return value.Value{}, errors.New("cannot " + verbFor(t.Op) + " " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")")
	}
	if l.IsAffine() && r.IsAffine() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot " + verbFor(t.Op) + " two absolute temperatures: use a difference such as 5 K"}
	}
	out := l
	shift, err := r.SI().Div(l.FactorOrOne())
	if err != nil {
		return value.Value{}, err
	}
	if t.Op == "-" {
		out.Num = l.Num.Sub(shift)
	} else {
		out.Num = l.Num.Add(shift)
	}
	return out, nil
}

func verbFor(op string) string {
	if op == "-" {
		return "subtract"
	}
	return "add"
}

func combine(l, r value.Value, n num.Exact, d units.Dim, sep string) value.Value {
	out := value.Value{Num: n, Dim: d, Factor: num.FromInt(1)}
	switch {
	case l.Unit == "" && r.Unit == "":
		out.Unit = ""
	case l.Unit == "":
		out.Unit = r.Unit
		if sep == "/" {
			out.Unit = "1/" + r.Unit
		}
	case r.Unit == "":
		out.Unit = l.Unit
	default:
		out.Unit = l.Unit + sep + r.Unit
	}
	if d.IsNone() {
		out.Unit = ""
	}
	return out
}

func isPercentNode(n parse.Node) bool {
	u, ok := n.(*parse.Unary)
	return ok && u.Op == "%"
}

func unitOrNone(v value.Value) string {
	if v.Unit == "" {
		return "no unit"
	}
	return v.Unit
}

func (e *Env) convert(t *parse.Binary) (value.Value, error) {
	l, err := e.Eval(t.L)
	if err != nil {
		return value.Value{}, err
	}
	name, ok := t.R.(*parse.Name)
	if !ok {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "convert to what unit?"}
	}
	to, ok := units.Lookup(name.Text)
	if !ok {
		return value.Value{}, &lex.Error{Pos: name.P, Msg: "unknown unit " + name.Text}
	}
	out, err := units.Convert(l.Num, l.Unwrap(), to)
	if err != nil {
		return value.Value{}, err
	}
	return value.WithUnit(out, to), nil
}

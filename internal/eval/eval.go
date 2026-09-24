package eval

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func (e *Env) eval(n parse.Node, p *pass) (value.Value, error) {
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
		}
		return value.Scalar(x), nil
	case *parse.Name:
		return e.evalName(t, p)
	case *parse.Assign:
		return e.evalAssign(t, p)
	case *parse.Call:
		return e.evalCall(t, p)
	case *parse.Unary:
		return e.evalUnary(t, p)
	case *parse.Binary:
		return e.evalBinary(t, p)
	}
	return value.Value{}, errors.New("unsupported expression")
}

func (e *Env) evalName(t *parse.Name, p *pass) (value.Value, error) {
	if v, ok := e.vars[t.Text]; ok {
		return p.note(v), nil
	}
	if c, ok := consts.Lookup(t.Text); ok {
		if u, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
			return value.WithUnit(c.Value, u), nil
		}
		v := value.Scalar(c.Value)
		v.Unit = c.Unit
		return v, nil
	}
	switch t.Text {
	case "pi", "π":
		return value.ScalarNum(num.NumBall(elem.Pi(p.prec()))), nil
	case "e":
		return value.ScalarNum(num.NumBall(elem.E(p.prec()))), nil
	}
	if u, ok := units.Lookup(t.Text); ok {
		return value.WithUnit(num.FromInt(1), u), nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unknown name " + t.Text}
}

func (e *Env) evalAssign(t *parse.Assign, p *pass) (value.Value, error) {
	if len(t.Params) > 0 {
		e.funcs[t.Name] = fn{params: t.Params, body: t.Body}
		return value.Scalar(num.FromInt(0)), nil
	}
	v, err := e.eval(t.Body, p)
	if err != nil {
		return value.Value{}, err
	}
	e.vars[t.Name] = v
	return v, nil
}

func (e *Env) evalCall(t *parse.Call, p *pass) (value.Value, error) {
	if f, ok := e.funcs[t.Fn]; ok {
		if len(f.params) != len(t.Args) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "wrong number of arguments for " + t.Fn}
		}
		inner := e.child()
		for i, param := range f.params {
			a, err := e.eval(t.Args[i], p)
			if err != nil {
				return value.Value{}, err
			}
			inner.vars[param] = a
		}
		return inner.eval(f.body, p)
	}
	if v, err, handled := e.builtin(t, p); handled {
		return v, err
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unknown function " + t.Fn}
}

const maxDegree = 100000

var radianIn = map[string]func(num.Ball, uint) (num.Ball, error){
	"sin": elem.Sin,
	"cos": elem.Cos,
	"tan": elem.Tan,
}

var radianOut = map[string]func(num.Ball, uint) (num.Ball, error){
	"asin": elem.Asin,
	"acos": elem.Acos,
	"atan": elem.Atan,
}

var plainElem = map[string]func(num.Ball, uint) (num.Ball, error){
	"exp":   elem.Exp,
	"ln":    elem.Log,
	"log":   elem.Log,
	"log10": elem.Log10,
	"log2":  elem.Log2,
	"sinh":  elem.Sinh,
	"cosh":  elem.Cosh,
	"tanh":  elem.Tanh,
}

func (e *Env) builtin(t *parse.Call, p *pass) (value.Value, error, bool) {
	args := make([]value.Value, len(t.Args))
	for i, a := range t.Args {
		v, err := e.eval(a, p)
		if err != nil {
			return value.Value{}, err, true
		}
		args[i] = v
	}
	prec := p.prec()
	fail := func(msg string) (value.Value, error, bool) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: msg}, true
	}
	ball := func() (num.Ball, error) {
		if !args[0].IsScalar() {
			return num.Ball{}, errors.New(t.Fn + " needs a plain number")
		}
		return args[0].Num.Ball(prec), nil
	}
	switch t.Fn {
	case "abs":
		if len(args) != 1 {
			return fail("abs takes one argument")
		}
		v := args[0]
		v.Num = v.Num.Abs()
		return v, nil, true
	case "sqrt":
		if len(args) != 1 {
			return fail("sqrt takes one argument")
		}
		if !args[0].IsScalar() {
			return fail("sqrt needs a plain number")
		}
		if args[0].Num.IsExact() {
			if r, ok := num.Root(args[0].Num.Exact(), 2); ok {
				return value.Scalar(r), nil, true
			}
		}
		b, err := elem.Sqrt(args[0].Num.Ball(prec), prec)
		if err != nil {
			return fail("sqrt: " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)), nil, true
	case "root":
		if len(args) != 2 || !isWhole(args[1]) {
			return fail("root takes a value and a whole degree")
		}
		whole := args[1].Num.Exact().Rat().Num()
		if !whole.IsInt64() || whole.Int64() <= 0 || whole.Int64() > maxDegree {
			return fail("root needs a degree between 1 and " + strconv.Itoa(maxDegree))
		}
		degree := int(whole.Int64())
		if args[0].Num.IsExact() {
			if r, ok := num.Root(args[0].Num.Exact(), degree); ok {
				return value.Scalar(r), nil, true
			}
		}
		if !args[0].IsScalar() {
			return fail("root needs a plain number")
		}
		b, err := elem.Pow(args[0].Num.Ball(prec), reciprocalBall(degree, prec), prec)
		if err != nil {
			return fail("root: " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)), nil, true
	case "atan2":
		if len(args) != 2 || !args[0].IsScalar() || !args[1].IsScalar() {
			return fail("atan2 takes two plain numbers")
		}
		b, err := elem.Atan2(args[0].Num.Ball(prec), args[1].Num.Ball(prec), prec)
		if err != nil {
			return fail("atan2: " + err.Error())
		}
		return angleOut(b, p, fail)
	case "floor", "ceil", "round":
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		if !args[0].Num.IsExact() {
			return fail(t.Fn + " needs an exact value")
		}
		return value.Scalar(roundTo(args[0].Num.Exact(), t.Fn)), nil, true
	case "gcd", "lcm":
		if len(args) != 2 || !isWhole(args[0]) || !isWhole(args[1]) {
			return fail(t.Fn + " takes two whole numbers")
		}
		a := args[0].Num.Exact().Rat().Num()
		b := args[1].Num.Exact().Rat().Num()
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
		if len(args) != 1 || !isWhole(args[0]) {
			return fail("isprime takes a whole number")
		}
		if args[0].Num.Exact().Rat().Num().ProbablyPrime(40) {
			return value.Scalar(num.FromInt(1)), nil, true
		}
		return value.Scalar(num.FromInt(0)), nil, true
	}
	if f, ok := plainElem[t.Fn]; ok {
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		x, err := ball()
		if err != nil {
			return fail(err.Error())
		}
		b, err := f(x, prec)
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)), nil, true
	}
	if f, ok := radianIn[t.Fn]; ok {
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		x, err := ball()
		if err != nil {
			return fail(err.Error())
		}
		if p.ctx.Degrees {
			x, err = degreesToRadians(x, prec)
			if err != nil {
				return fail(t.Fn + ": " + err.Error())
			}
		}
		b, err := f(x, prec)
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)), nil, true
	}
	if f, ok := radianOut[t.Fn]; ok {
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		x, err := ball()
		if err != nil {
			return fail(err.Error())
		}
		b, err := f(x, prec)
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return angleOut(b, p, fail)
	}
	return value.Value{}, nil, false
}

func angleOut(b num.Ball, p *pass, fail func(string) (value.Value, error, bool)) (value.Value, error, bool) {
	if p.ctx.Degrees {
		converted, err := radiansToDegrees(b, p.prec())
		if err != nil {
			return fail(err.Error())
		}
		b = converted
	}
	return value.ScalarNum(num.NumBall(b)), nil, true
}

func isWhole(v value.Value) bool {
	return v.Num.IsExact() && v.Num.Exact().IsInt()
}

func reciprocalBall(degree int, prec uint) num.Ball {
	r, _ := num.FromInt(1).Div(num.FromInt(int64(degree)))
	return num.BallFromExact(r, prec)
}

func straightAngle(prec uint) num.Ball {
	return num.BallFromExact(num.FromInt(180), prec)
}

func degreesToRadians(b num.Ball, prec uint) (num.Ball, error) {
	return b.Mul(elem.Pi(prec), prec).Div(straightAngle(prec), prec)
}

func radiansToDegrees(b num.Ball, prec uint) (num.Ball, error) {
	return b.Mul(straightAngle(prec), prec).Div(elem.Pi(prec), prec)
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

func (e *Env) evalUnary(t *parse.Unary, p *pass) (value.Value, error) {
	x, err := e.eval(t.X, p)
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
		q, err := x.Num.Div(num.NumExact(num.FromInt(100)), p.prec())
		if err != nil {
			return value.Value{}, err
		}
		return value.ScalarNum(q), nil
	case "!":
		if !x.IsScalar() || !isWhole(x) || x.Num.Exact().Sign() < 0 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "factorial needs a whole number that is not negative"}
		}
		n := x.Num.Exact().Rat().Num()
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

func (e *Env) evalBinary(t *parse.Binary, p *pass) (value.Value, error) {
	if t.Implicit {
		if v, err, handled := e.implicitUnit(t, p); handled {
			return v, err
		}
	}
	if t.Op == "to" || t.Op == "in" {
		return e.convert(t, p)
	}
	l, err := e.eval(t.L, p)
	if err != nil {
		return value.Value{}, err
	}
	r, err := e.eval(t.R, p)
	if err != nil {
		return value.Value{}, err
	}
	prec := p.prec()
	switch t.Op {
	case "+", "-":
		if l.IsAffine() || r.IsAffine() {
			return affineAddSub(t, l, r, prec)
		}
		return addSub(t, l, r, prec)
	case "*", "×", "·":
		return combine(l, r, l.SI(prec).Mul(r.SI(prec), prec), l.Dim.Add(r.Dim), "·"), nil
	case "/", "÷":
		q, err := l.SI(prec).Div(r.SI(prec), prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: divisionMessage(err)}
		}
		return combine(l, r, q, l.Dim.Sub(r.Dim), "/"), nil
	case "mod":
		if !l.IsScalar() || !r.IsScalar() || !isWhole(l) || !isWhole(r) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "mod needs whole numbers"}
		}
		if r.Num.Exact().Sign() == 0 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "division by zero"}
		}
		m := new(big.Int).Mod(l.Num.Exact().Rat().Num(), r.Num.Exact().Rat().Num())
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(m))), nil
	case "^":
		return power(t, l, r, prec)
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unsupported operator " + t.Op}
}

func divisionMessage(err error) string {
	if errors.Is(err, num.ErrDivisorMayBeZero) {
		return err.Error()
	}
	return "division by zero"
}

func power(t *parse.Binary, l, r value.Value, prec uint) (value.Value, error) {
	if r.IsScalar() && isWhole(r) {
		exp64 := r.Num.Exact().Rat().Num().Int64()
		if exp64 > 100000 || exp64 < -100000 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "exponent too large"}
		}
		exp := int(exp64)
		raised, err := l.Num.Pow(exp, prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
		}
		out := l
		out.Num = raised
		out.Dim = l.Dim.Scale(exp)
		out.Factor = l.FactorOrOne().Pow(exp)
		if exp != 1 && l.Unit != "" {
			out.Unit = l.Unit + "^" + r.Num.String()
		}
		return out, nil
	}
	if !l.IsScalar() || !r.IsScalar() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a value with units needs a whole exponent"}
	}
	b, err := elem.Pow(l.Num.Ball(prec), r.Num.Ball(prec), prec)
	if err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "power: " + err.Error()}
	}
	return value.ScalarNum(num.NumBall(b)), nil
}

func (e *Env) implicitUnit(t *parse.Binary, p *pass) (value.Value, error, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return value.Value{}, nil, false
	}
	u, ok := units.Lookup(name.Text)
	if !ok {
		return value.Value{}, nil, false
	}
	l, err := e.eval(t.L, p)
	if err != nil {
		return value.Value{}, err, true
	}
	if !l.IsScalar() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot attach " + name.Text + " to a value that already has a unit"}, true
	}
	if u.Dim.IsNone() {
		return value.ScalarNum(l.Num.Mul(num.NumExact(u.Factor), p.prec())), nil, true
	}
	return value.WithUnitNum(l.Num, u), nil, true
}

func addSub(t *parse.Binary, l, r value.Value, prec uint) (value.Value, error) {
	if r.IsScalar() && isPercentNode(t.R) {
		one := num.NumExact(num.FromInt(1))
		scale := one.Add(r.Num, prec)
		if t.Op == "-" {
			scale = one.Sub(r.Num, prec)
		}
		out := l
		out.Num = l.Num.Mul(scale, prec)
		return out, nil
	}
	if l.Dim != r.Dim {
		return value.Value{}, errors.New("cannot " + verbFor(t.Op) + " " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")")
	}
	rInL, err := r.SI(prec).Div(num.NumExact(l.FactorOrOne()), prec)
	if err != nil {
		return value.Value{}, err
	}
	out := l
	if t.Op == "-" {
		out.Num = l.Num.Sub(rInL, prec)
	} else {
		out.Num = l.Num.Add(rInL, prec)
	}
	return out, nil
}

func affineAddSub(t *parse.Binary, l, r value.Value, prec uint) (value.Value, error) {
	if l.Dim != r.Dim {
		return value.Value{}, errors.New("cannot " + verbFor(t.Op) + " " + l.Dim.Words() + " (" + unitOrNone(l) + ") and " + r.Dim.Words() + " (" + unitOrNone(r) + ")")
	}
	if l.IsAffine() && r.IsAffine() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "cannot " + verbFor(t.Op) + " two absolute temperatures: use a difference such as 5 K"}
	}
	out := l
	shift, err := r.SI(prec).Div(num.NumExact(l.FactorOrOne()), prec)
	if err != nil {
		return value.Value{}, err
	}
	if t.Op == "-" {
		out.Num = l.Num.Sub(shift, prec)
	} else {
		out.Num = l.Num.Add(shift, prec)
	}
	return out, nil
}

func verbFor(op string) string {
	if op == "-" {
		return "subtract"
	}
	return "add"
}

func combine(l, r value.Value, n num.Number, d units.Dim, sep string) value.Value {
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

func (e *Env) convert(t *parse.Binary, p *pass) (value.Value, error) {
	l, err := e.eval(t.L, p)
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
	out, err := units.ConvertNumber(l.Num, l.Unwrap(), to, p.prec())
	if err != nil {
		return value.Value{}, err
	}
	return value.WithUnitNum(out, to), nil
}

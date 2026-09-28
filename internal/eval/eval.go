package eval

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func (e *Env) evalNode(n parse.Node, p *pass) (value.Value, error) {
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

type unitScope struct{ at map[parse.Node]bool }

func (s *unitScope) holds(n parse.Node) bool { return s.at[n] }

func (s *unitScope) enter(n parse.Node) {
	if s.at == nil {
		s.at = map[parse.Node]bool{}
	}
	s.at[n] = true
}

func (s *unitScope) leave(n parse.Node) { delete(s.at, n) }

func (e *Env) evalName(t *parse.Name, p *pass) (value.Value, error) {
	if p.units.holds(t) {
		if v, ok := e.unitInPosition(t, p); ok {
			return v, nil
		}
	}
	if v, ok := e.vars[t.Text]; ok {
		e.namedProv(t, p, v)
		return p.note(v), nil
	}
	if c, ok := consts.Lookup(t.Text); ok {
		e.record(t, p, prov.Node{Kind: prov.Constant, Label: t.Text, Detail: constantDetail(c)})
		if c.Formula != "" {
			return e.derived(c, p)
		}
		if u, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
			return value.WithUnit(c.Value, u), nil
		}
		v := value.Scalar(c.Value)
		v.Unit = c.Unit
		return v, nil
	}
	switch t.Text {
	case "pi", "π":
		e.record(t, p, prov.Node{Kind: prov.Constant, Label: t.Text, Detail: workingPrecision})
		return value.ScalarNum(num.NumBall(elem.Pi(p.prec()))), nil
	case "e":
		e.record(t, p, prov.Node{Kind: prov.Constant, Label: t.Text, Detail: workingPrecision})
		return value.ScalarNum(num.NumBall(elem.E(p.prec()))), nil
	}
	if u, ok := units.Lookup(t.Text); ok {
		e.record(t, p, prov.Node{Kind: prov.Unit, Label: t.Text, Detail: unitDetail(u)})
		return value.WithUnit(num.FromInt(1), u), nil
	}
	return value.Value{}, &lex.Error{Pos: t.P, Msg: "unknown name " + t.Text}
}

func (e *Env) derived(c consts.Const, p *pass) (value.Value, error) {
	n, ok := formulaTree(c)
	if !ok {
		return value.Value{}, &lex.Error{Msg: "the definition of " + c.Symbol + " cannot be read"}
	}
	v, err := NewEnvWith(e.table).evalNode(n, &pass{ctx: p.ctx, prov: map[parse.Node]prov.NodeID{}, divisor: map[parse.Node]bool{}})
	if err != nil {
		return value.Value{}, err
	}
	v.Prov = prov.None
	return v, nil
}

var formulas sync.Map

func formulaTree(c consts.Const) (parse.Node, bool) {
	if n, ok := formulas.Load(c.Symbol); ok {
		return n.(parse.Node), true
	}
	n, _, err := parse.Parse(c.Formula)
	if err != nil {
		return nil, false
	}
	formulas.Store(c.Symbol, n)
	return n, true
}

func (e *Env) unitInPosition(t *parse.Name, p *pass) (value.Value, bool) {
	u, bound, shadowed, asUnit := e.implicitReading(t)
	if shadowed {
		p.notes = append(p.notes, readingNote(t, u, bound))
	}
	if !asUnit {
		return value.Value{}, false
	}
	e.record(t, p, prov.Node{Kind: prov.Unit, Label: t.Text, Detail: unitDetail(u)})
	return value.WithUnit(num.FromInt(1), u), true
}

const workingPrecision = "computed to the working precision"

func (e *Env) evalAssign(t *parse.Assign, p *pass) (value.Value, error) {
	if t.Name == "corr" && len(t.Params) == 2 {
		return e.declareCorr(t, p)
	}
	if len(t.Params) > 0 {
		e.funcs[t.Name] = fn{params: t.Params, body: t.Body}
		return value.Definition(t.Name, t.Params), nil
	}
	outer := p.naming
	if countMeasurements(t.Body) == 1 {
		p.naming = t.Name
	}
	v, err := e.eval(t.Body, p)
	p.naming = outer
	if err != nil {
		return value.Value{}, err
	}
	e.vars[t.Name] = v
	p.assigned = true
	return v, nil
}

var namedArguments = map[string]map[string]int{"expand": {"k": 1}}

func namedArgumentError(t *parse.Call) error {
	allowed := namedArguments[t.Fn]
	for i, key := range t.Keys {
		if key == "" {
			continue
		}
		at, ok := allowed[key]
		if !ok {
			return &lex.Error{Pos: t.P, Msg: t.Fn + " takes no argument called " + key}
		}
		if at != i {
			return &lex.Error{Pos: t.P, Msg: key + " is argument " + strconv.Itoa(at+1) + " of " + t.Fn}
		}
	}
	return nil
}

func (e *Env) evalCall(t *parse.Call, p *pass) (value.Value, error) {
	if err := namedArgumentError(t); err != nil {
		return value.Value{}, err
	}
	if f, ok := e.funcs[t.Fn]; ok {
		if len(f.params) != len(t.Args) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "wrong number of arguments for " + t.Fn}
		}
		inner := e.child()
		measured := make([]bool, len(f.params))
		for i, param := range f.params {
			a, err := e.eval(t.Args[i], p)
			if err != nil {
				return value.Value{}, err
			}
			inner.vars[param] = a
			measured[i] = a.IsMeasured()
		}
		e.bound.record(t, measured)
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
		if !v.IsMeasured() {
			return v, nil, true
		}
		x := siFloat(args[0], prec)
		if x == 0 {
			return fail("abs has no derivative at zero, so no uncertainty can be propagated")
		}
		return v.WithJet(unc.Unary(args[0].Jet(), math.Copysign(1, x), 0)), nil, true
	case "sqrt":
		if len(args) != 1 {
			return fail("sqrt takes one argument")
		}
		arg := args[0]
		inner := arg.Num
		var half units.Dim
		if !arg.IsScalar() {
			h, even := arg.Dim.Half()
			if !even || arg.IsAffine() {
				return fail("sqrt needs a plain number or a value whose dimensions are all even")
			}
			half, inner = h, arg.SI(prec)
		}
		wrap := func(n num.Number, j unc.Jet) value.Value {
			if half.IsNone() {
				return value.ScalarNum(n).WithJet(j)
			}
			return value.WithUnitNum(n, units.SIUnit(half)).WithJet(j)
		}
		if inner.IsExact() && inner.Exact().Sign() < 0 {
			return fail("sqrt: square root of a negative number")
		}
		if inner.IsExact() {
			if r, ok := num.Root(inner.Exact(), 2); ok {
				j, err := elemJet("sqrt", arg.Jet(), siFloat(arg, prec), exactFloat(r))
				if err != nil {
					return fail("sqrt: " + err.Error())
				}
				return wrap(num.NumExact(r), j), nil, true
			}
		}
		b, err := elem.Sqrt(inner.Ball(prec), prec)
		if err != nil {
			return fail("sqrt: " + err.Error())
		}
		j, err := elemJet("sqrt", arg.Jet(), siFloat(arg, prec), numFloat(num.NumBall(b), prec))
		if err != nil {
			return fail("sqrt: " + err.Error())
		}
		return wrap(num.NumBall(b), j), nil, true
	case "root":
		if len(args) != 2 || !isWhole(args[1]) {
			return fail("root takes a value and a whole degree")
		}
		if args[1].IsMeasured() {
			return fail("the degree of a root must be an exact whole number")
		}
		whole := args[1].Num.Exact().Rat().Num()
		if !whole.IsInt64() || whole.Int64() <= 0 || whole.Int64() > maxDegree {
			return fail("root needs a degree between 1 and " + strconv.Itoa(maxDegree))
		}
		degree := int(whole.Int64())
		rootJet := func() (unc.Jet, error) {
			return powerJet(args[0].Jet(), unc.Jet{}, siFloat(args[0], prec), 1/float64(degree))
		}
		if args[0].Num.IsExact() {
			if r, ok := num.Root(args[0].Num.Exact(), degree); ok {
				j, err := rootJet()
				if err != nil {
					return fail("root: " + err.Error())
				}
				return value.Scalar(r).WithJet(j), nil, true
			}
		}
		if !args[0].IsScalar() {
			return fail("root needs a plain number")
		}
		b, err := elem.Pow(args[0].Num.Ball(prec), reciprocalBall(degree, prec), prec)
		if err != nil {
			return fail("root: " + err.Error())
		}
		j, err := rootJet()
		if err != nil {
			return fail("root: " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)).WithJet(j), nil, true
	case "atan2":
		if len(args) != 2 || !args[0].IsScalar() || !args[1].IsScalar() {
			return fail("atan2 takes two plain numbers")
		}
		b, err := elem.Atan2(args[0].Num.Ball(prec), args[1].Num.Ball(prec), prec)
		if err != nil {
			return fail("atan2: " + err.Error())
		}
		j, err := atan2Jet(args[0].Jet(), args[1].Jet(), siFloat(args[0], prec), siFloat(args[1], prec))
		if err != nil {
			return fail("atan2: " + err.Error())
		}
		return angleOut(b, j, p, fail)
	case "expand":
		v, err := e.expand(t, args, p)
		return v, err, true
	case "floor", "ceil", "round":
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		if args[0].IsMeasured() {
			return value.Value{}, cannotPropagate(t.P, t.Fn), true
		}
		if !args[0].Num.IsExact() {
			return fail(t.Fn + " needs an exact value")
		}
		out := args[0]
		out.Num = num.NumExact(roundTo(args[0].Num.Exact(), t.Fn))
		return out.WithJet(unc.Jet{}), nil, true
	case "gcd", "lcm":
		if measured(args...) {
			return value.Value{}, cannotPropagate(t.P, t.Fn), true
		}
		if len(args) != 2 {
			return fail(t.Fn + " takes two whole numbers")
		}
		if !plainNumber(args[0]) || !plainNumber(args[1]) {
			return fail(t.Fn + " needs plain numbers")
		}
		if !isWhole(args[0]) || !isWhole(args[1]) {
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
		if measured(args...) {
			return value.Value{}, cannotPropagate(t.P, t.Fn), true
		}
		if len(args) != 1 {
			return fail("isprime takes a whole number")
		}
		if !plainNumber(args[0]) {
			return fail("isprime needs a plain number")
		}
		if !isWhole(args[0]) {
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
		if msg := certainDomainFailure(t.Fn, args[0]); msg != "" {
			return fail(t.Fn + ": " + msg)
		}
		b, err := f(x, prec)
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		j, err := elemJet(t.Fn, args[0].Jet(), siFloat(args[0], prec), numFloat(num.NumBall(b), prec))
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)).WithJet(j), nil, true
	}
	if f, ok := radianIn[t.Fn]; ok {
		if len(args) != 1 {
			return fail(t.Fn + " takes one argument")
		}
		x, err := ball()
		if err != nil {
			return fail(err.Error())
		}
		jet := args[0].Jet()
		angle := siFloat(args[0], prec)
		if p.ctx.Degrees {
			x, err = degreesToRadians(x, prec)
			if err != nil {
				return fail(t.Fn + ": " + err.Error())
			}
			jet = unc.Unary(jet, math.Pi/180, 0)
			angle = angle * math.Pi / 180
		}
		b, err := f(x, prec)
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		j, err := elemJet(t.Fn, jet, angle, numFloat(num.NumBall(b), prec))
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return value.ScalarNum(num.NumBall(b)).WithJet(j), nil, true
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
		j, err := elemJet(t.Fn, args[0].Jet(), siFloat(args[0], prec), numFloat(num.NumBall(b), prec))
		if err != nil {
			return fail(t.Fn + ": " + err.Error())
		}
		return angleOut(b, j, p, fail)
	}
	return value.Value{}, nil, false
}

var logarithms = map[string]bool{"ln": true, "log": true, "log10": true, "log2": true}

func certainDomainFailure(name string, arg value.Value) string {
	if !logarithms[name] || !arg.Num.IsExact() {
		return ""
	}
	switch arg.Num.Exact().Sign() {
	case 0:
		return "log of zero"
	case -1:
		return "log of a negative number"
	}
	return ""
}

func angleOut(b num.Ball, j unc.Jet, p *pass, fail func(string) (value.Value, error, bool)) (value.Value, error, bool) {
	if p.ctx.Degrees {
		converted, err := radiansToDegrees(b, p.prec())
		if err != nil {
			return fail(err.Error())
		}
		b = converted
		j = unc.Unary(j, 180/math.Pi, 0)
	}
	return value.ScalarNum(num.NumBall(b)).WithJet(j), nil, true
}

func isWhole(v value.Value) bool {
	return v.Num.IsExact() && v.Num.Exact().IsInt()
}

func plainNumber(v value.Value) bool {
	return v.IsScalar() && v.FactorOrOne().Cmp(num.FromInt(1)) == 0 && v.Offset.Sign() == 0
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
		return x.WithJet(unc.Unary(x.Jet(), -1, 0)), nil
	case "%":
		if !x.IsScalar() {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "percent needs a plain number"}
		}
		q, err := x.Num.Div(num.NumExact(num.FromInt(100)), p.prec())
		if err != nil {
			return value.Value{}, err
		}
		return value.ScalarNum(q).WithJet(unc.Unary(x.Jet(), 0.01, 0)), nil
	case "!":
		if x.IsMeasured() {
			return value.Value{}, cannotPropagate(t.P, "factorial")
		}
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
	if t.Op == "to" || t.Op == "in" {
		return e.convert(t, p)
	}
	if t.Op == "±" || t.Op == "+-" {
		return e.measure(t, p)
	}
	inherited := p.units.holds(t)
	l, err := e.evalPositioned(t.L, p, leftUnitPosition(t, inherited))
	if err != nil {
		return value.Value{}, err
	}
	if t.Implicit && parse.InDenominator(t.L) {
		return e.implicitAfterDivision(t, l, p)
	}
	if t.Implicit {
		if v, err, handled := e.attachUnit(t, l, p); handled {
			return v, err
		}
	}
	r, err := e.evalPositioned(t.R, p, rightUnitPosition(t, inherited, !l.IsScalar()))
	if err != nil {
		return value.Value{}, err
	}
	if t.Op == "/" || t.Op == "÷" {
		p.divisor[t] = !t.Grouped
	}
	prec := p.prec()
	switch t.Op {
	case "+", "-":
		if l.IsAffine() || r.IsAffine() {
			return affineAddSub(t, l, r, prec)
		}
		return addSub(t, l, r, prec)
	case "*", "×", "·", "of":
		if v, ok := wearUnit(l, r); ok {
			return v, nil
		}
		out := combine(l, r, l.SI(prec).Mul(r.SI(prec), prec), l.Dim.Add(r.Dim), "·")
		return out.WithJet(productJet(l, r, prec)), nil
	case "/", "÷":
		q, err := l.SI(prec).Div(r.SI(prec), prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: divisionMessage(err)}
		}
		j, err := quotientJet(l, r, prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "the divisor may be zero, so no uncertainty can be propagated"}
		}
		return combine(l, r, q, l.Dim.Sub(r.Dim), "/").WithJet(j), nil
	case "mod":
		if measured(l, r) {
			return value.Value{}, cannotPropagate(t.P, "mod")
		}
		if !plainNumber(l) || !plainNumber(r) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "mod needs plain numbers"}
		}
		if !isWhole(l) || !isWhole(r) {
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

const maxWritableDigits = 10000

const TooLargeMessage = "too large to work with: writing this number would take more than 10000 digits"

func beyondWritableWidth(digits float64) bool { return digits > maxWritableDigits }

func power(t *parse.Binary, l, r value.Value, prec uint) (value.Value, error) {
	if r.IsScalar() && isWhole(r) {
		exp64 := r.Num.Exact().Rat().Num().Int64()
		if exp64 > 100000 || exp64 < -100000 {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "exponent too large"}
		}
		exp := int(exp64)
		j, err := powerJet(l.Jet(), r.Jet(), siFloat(l, prec), float64(exp))
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "power: " + err.Error()}
		}
		if exp == 1 {
			return l.WithJet(j), nil
		}
		base := absolute(l, prec)
		if beyondWritableWidth(base.Num.RoughWidth() * math.Abs(float64(exp))) {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: TooLargeMessage}
		}
		raised, err := base.Num.Pow(exp, prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
		}
		out := value.Value{
			Num:     raised,
			Dim:     base.Dim.Scale(exp),
			Factor:  base.FactorOrOne().Pow(exp),
			Stopped: l.Stopped,
		}
		if out.Dim.IsNone() {
			out.Num = out.Num.Mul(num.NumExact(out.Factor), prec)
			out.Factor = num.FromInt(1)
		} else if base.Unit != "" {
			out.Unit = grouped(base.Unit, "/·^") + "^" + strconv.Itoa(exp)
		}
		return out.WithJet(j), nil
	}
	if !l.IsScalar() || !r.IsScalar() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a value with units needs a whole exponent"}
	}
	b, err := elem.Pow(l.Num.Ball(prec), r.Num.Ball(prec), prec)
	if err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "power: " + err.Error()}
	}
	j, err := powerJet(l.Jet(), r.Jet(), siFloat(l, prec), siFloat(r, prec))
	if err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "power: " + err.Error()}
	}
	return value.ScalarNum(num.NumBall(b)).WithJet(j), nil
}

func (e *Env) implicitReading(name *parse.Name) (units.Unit, value.Value, bool, bool) {
	u, known := units.Lookup(name.Text)
	bound, defined := e.vars[name.Text]
	if defined && (!bound.IsScalar() || bound.IsMeasured()) {
		return u, bound, defined && known, false
	}
	return u, bound, defined && known, known
}

func isProduct(op string) bool {
	switch op {
	case "*", "×", "·", "/", "÷":
		return true
	}
	return false
}

func namesAUnit(n parse.Node) bool {
	switch t := n.(type) {
	case *parse.Name:
		_, known := units.Lookup(t.Text)
		return known
	case *parse.Binary:
		if isProduct(t.Op) {
			return namesAUnit(t.L) || namesAUnit(t.R)
		}
		if t.Op == "^" {
			return namesAUnit(t.L)
		}
	}
	return false
}

func leftUnitPosition(t *parse.Binary, inherited bool) bool {
	if !isProduct(t.Op) {
		return inherited && t.Op == "^"
	}
	return inherited || namesAUnit(t.R)
}

func rightUnitPosition(t *parse.Binary, inherited, leftWearsAUnit bool) bool {
	if t.Implicit {
		return true
	}
	return isProduct(t.Op) && (inherited || leftWearsAUnit || namesAUnit(t.L))
}

func UnitPositions(n parse.Node) map[parse.Node]bool {
	marked := map[parse.Node]bool{}
	markUnitPositions(n, false, marked)
	return marked
}

func markUnitPositions(n parse.Node, inherited bool, marked map[parse.Node]bool) {
	switch t := n.(type) {
	case *parse.Name:
		if inherited {
			marked[t] = true
		}
	case *parse.Unary:
		markUnitPositions(t.X, false, marked)
	case *parse.Binary:
		if t.Op == "to" || t.Op == "in" {
			markUnitPositions(t.L, false, marked)
			return
		}
		markUnitPositions(t.L, leftUnitPosition(t, inherited), marked)
		markUnitPositions(t.R, rightUnitPosition(t, inherited, false), marked)
	case *parse.Call:
		for _, a := range t.Args {
			markUnitPositions(a, false, marked)
		}
	case *parse.Assign:
		markUnitPositions(t.Body, false, marked)
	}
}

func (e *Env) evalPositioned(n parse.Node, p *pass, unitPosition bool) (value.Value, error) {
	if !unitPosition {
		return e.eval(n, p)
	}
	p.units.enter(n)
	defer p.units.leave(n)
	return e.eval(n, p)
}

func (e *Env) attachUnit(t *parse.Binary, l value.Value, p *pass) (value.Value, error, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return value.Value{}, nil, false
	}
	u, bound, ambiguous, asUnit := e.implicitReading(name)
	if ambiguous {
		p.notes = append(p.notes, readingNote(name, u, bound))
	}
	if !asUnit || !l.IsScalar() {
		return value.Value{}, nil, false
	}
	e.record(t, p, prov.Node{Kind: prov.Unit, Label: name.Text, Detail: unitDetail(u), Children: []prov.NodeID{l.Prov}})
	scaled := unc.Unary(l.Jet(), exactFloat(u.Factor), 0)
	if u.Dim.IsNone() {
		return value.ScalarNum(l.Num.Mul(num.NumExact(u.Factor), p.prec())).WithJet(scaled), nil, true
	}
	return value.WithUnitNum(l.Num, u).WithJet(scaled), nil, true
}

func (e *Env) implicitAfterDivision(t *parse.Binary, l value.Value, p *pass) (value.Value, error) {
	r, err := e.evalPositioned(t.R, p, true)
	if err != nil {
		return value.Value{}, err
	}
	prec := p.prec()
	if p.divides(t.L) && e.readsAsUnit(t.R) {
		p.divisor[t] = true
		if r.Dim.IsNone() && r.Unit != "" {
			r = value.ScalarNum(r.SI(prec))
		}
		q, err := l.SI(prec).Div(r.SI(prec), prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: divisionMessage(err)}
		}
		j, err := quotientJet(l, r, prec)
		if err != nil {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "the divisor may be zero, so no uncertainty can be propagated"}
		}
		return combine(l, r, q, l.Dim.Sub(r.Dim), "/").WithJet(j), nil
	}
	if v, ok := wearUnit(l, r); ok {
		return v, nil
	}
	out := combine(l, r, l.SI(prec).Mul(r.SI(prec), prec), l.Dim.Add(r.Dim), "·")
	return out.WithJet(productJet(l, r, prec)), nil
}

func (p *pass) divides(n parse.Node) bool {
	b, ok := n.(*parse.Binary)
	if !ok {
		return false
	}
	if b.Op == "/" || b.Op == "÷" {
		return !b.Grouped
	}
	return b.Implicit && p.divisor[b]
}

func (e *Env) readsAsUnit(n parse.Node) bool {
	switch t := n.(type) {
	case *parse.Name:
		_, _, _, asUnit := e.implicitReading(t)
		return asUnit
	case *parse.Binary:
		if isProduct(t.Op) {
			return e.readsAsUnit(t.L) && e.readsAsUnit(t.R)
		}
		if t.Op == "^" {
			return e.readsAsUnit(t.L)
		}
	}
	return false
}

func isBareUnit(v value.Value) bool {
	if v.Unit == "" || v.IsAffine() || v.IsMeasured() || !v.Num.IsExact() {
		return false
	}
	return v.Num.Exact().Cmp(num.FromInt(1)) == 0
}

func (p *pass) announceRegroupings(n parse.Node) {
	if at, ok := regroupingSite(n); ok {
		p.notes = append(p.notes, parse.Note{Pos: at, Msg: "read as " + parse.Reading(p.effective(n))})
	}
	p.announceInside(n)
}

func (p *pass) announceInside(n parse.Node) {
	switch t := n.(type) {
	case *parse.Unary:
		p.announceInside(t.X)
	case *parse.Binary:
		p.announceInside(t.L)
		p.announceInside(t.R)
	case *parse.Call:
		for _, a := range t.Args {
			p.announceRegroupings(a)
		}
	case *parse.Assign:
		if len(t.Params) == 0 {
			p.announceRegroupings(t.Body)
		}
	}
}

func regroupingSite(n parse.Node) (lex.Pos, bool) {
	switch t := n.(type) {
	case *parse.Unary:
		return regroupingSite(t.X)
	case *parse.Binary:
		if t.Implicit && parse.InDenominator(t.L) {
			return t.P, true
		}
		if at, ok := regroupingSite(t.L); ok {
			return at, true
		}
		return regroupingSite(t.R)
	}
	return lex.Pos{}, false
}

func (p *pass) effective(n parse.Node) parse.Node {
	b, ok := n.(*parse.Binary)
	if !ok {
		return n
	}
	l, r := p.effective(b.L), p.effective(b.R)
	if p.divisor[b] && b.Implicit {
		if d, ok := l.(*parse.Binary); ok && (d.Op == "/" || d.Op == "÷") {
			return &parse.Binary{Op: d.Op, L: d.L, R: &parse.Binary{Op: "*", L: d.R, R: r, Implicit: true}}
		}
	}
	if l == b.L && r == b.R {
		return b
	}
	return &parse.Binary{Op: b.Op, L: l, R: r, P: b.P, Implicit: b.Implicit}
}

func wearUnit(l, r value.Value) (value.Value, bool) {
	if !l.IsScalar() || l.IsMeasured() || r.IsMeasured() {
		return value.Value{}, false
	}
	if !isBareUnit(r) {
		return value.Value{}, false
	}
	out := r
	out.Num = l.Num
	return out.WithJet(unc.Unary(l.Jet(), exactFloat(r.FactorOrOne()), 0)), true
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
		return out.WithJet(percentJet(l, r, t.Op, prec)), nil
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
	return out.WithJet(sumJet(l, r, t.Op)), nil
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
	return out.WithJet(sumJet(l, r, t.Op)), nil
}

func verbFor(op string) string {
	if op == "-" {
		return "subtract"
	}
	return "add"
}

func combine(l, r value.Value, n num.Number, d units.Dim, sep string) value.Value {
	out := value.Value{Num: n, Dim: d, Factor: num.FromInt(1)}
	if !spellsItsOwnSize(l) || !spellsItsOwnSize(r) {
		out.Unit = units.SISymbol(d)
		return out
	}
	switch {
	case l.Unit == "" && r.Unit == "":
		out.Unit = ""
	case l.Unit == "":
		out.Unit = r.Unit
		if sep == "/" {
			out.Unit = "1/" + grouped(r.Unit, "/·")
		}
	case r.Unit == "":
		out.Unit = l.Unit
	default:
		denominator := "/"
		if sep == "/" {
			denominator = "/·"
		}
		out.Unit = grouped(l.Unit, "/") + sep + grouped(r.Unit, denominator)
	}
	if d.IsNone() {
		out.Unit = ""
	}
	return out
}

func spellsItsOwnSize(v value.Value) bool {
	if v.Unit == "" {
		return true
	}
	return v.FactorOrOne().Cmp(num.FromInt(1)) == 0 && v.Offset.Sign() == 0
}

func absolute(v value.Value, prec uint) value.Value {
	if !v.IsAffine() {
		return v
	}
	return value.Value{Num: v.SI(prec), Dim: v.Dim, Unit: units.SISymbol(v.Dim), Factor: num.FromInt(1)}
}

func grouped(unit, ambiguous string) string {
	depth := 0
	for _, r := range unit {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case depth == 0 && strings.ContainsRune(ambiguous, r):
			return "(" + unit + ")"
		}
	}
	return unit
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
	target, ok := unitTarget(t.R)
	if !ok {
		return value.Value{}, &lex.Error{Pos: t.R.Pos(), Msg: "convert to what unit? " + t.R.String() + " is not a unit"}
	}
	to, ok := units.Lookup(target)
	if !ok {
		return value.Value{}, &lex.Error{Pos: t.R.Pos(), Msg: "unknown unit " + target}
	}
	out, err := units.ConvertNumber(l.Num, l.Unwrap(), to, p.prec())
	if err != nil {
		return value.Value{}, err
	}
	e.record(t, p, prov.Node{Kind: prov.Unit, Label: t.Op + " " + target, Detail: unitDetail(to), Children: []prov.NodeID{l.Prov}})
	return value.WithUnitNum(out, to).WithJet(l.Jet()), nil
}

func unitTarget(n parse.Node) (string, bool) {
	switch t := n.(type) {
	case *parse.Name:
		return t.Text, true
	case *parse.NumLit:
		if t.Text == "1" {
			return "1", true
		}
	case *parse.Binary:
		if isProduct(t.Op) {
			return unitPair(t, separatorFor(t.Op))
		}
		if t.Op == "^" {
			base, ok := unitTarget(t.L)
			if !ok {
				return "", false
			}
			exponent, ok := wholeExponent(t.R)
			if !ok {
				return "", false
			}
			return bracketed(t.L, base) + "^" + exponent, true
		}
	}
	return "", false
}

func separatorFor(op string) string {
	if op == "/" || op == "÷" {
		return "/"
	}
	return "·"
}

func unitPair(t *parse.Binary, sep string) (string, bool) {
	left, ok := unitTarget(t.L)
	if !ok {
		return "", false
	}
	right, ok := unitTarget(t.R)
	if !ok {
		return "", false
	}
	return bracketed(t.L, left) + sep + bracketed(t.R, right), true
}

func bracketed(n parse.Node, sym string) string {
	if t, ok := n.(*parse.Binary); ok && isProduct(t.Op) {
		return "(" + sym + ")"
	}
	return sym
}

func wholeExponent(n parse.Node) (string, bool) {
	if u, ok := n.(*parse.Unary); ok && u.Op == "-" {
		digits, ok := wholeExponent(u.X)
		if !ok {
			return "", false
		}
		return "-" + digits, true
	}
	lit, ok := n.(*parse.NumLit)
	if !ok {
		return "", false
	}
	for _, r := range lit.Text {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return lit.Text, len(lit.Text) > 0
}

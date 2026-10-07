package eval

import (
	"fmt"
	"sort"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/solve"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	searchSteps       = 2000
	defaultSearchSpan = 100
	methodLinear      = "exact linear solution"
	methodQuadratic   = "quadratic formula"
	methodBisection   = "bisection on proven sign-change brackets"
)

func (e *Env) solveCall(t *parse.Call, p *pass) (value.Value, error) {
	fail := func(msg string) (value.Value, error) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: msg}
	}
	if len(t.Args) != 2 && len(t.Args) != 4 {
		return fail("solve takes an equation, a variable, and optionally a lower and upper bound to search")
	}
	variable, ok := t.Args[1].(*parse.Name)
	if !ok {
		return fail("solve needs the name of the variable to solve for as its second argument")
	}
	residual := residualOf(t.Args[0])
	span := searchSpan{lo: num.FromInt(-defaultSearchSpan), hi: num.FromInt(defaultSearchSpan)}
	if len(t.Args) == 4 {
		var err error
		if span, err = e.explicitSpan(t, p); err != nil {
			return value.Value{}, err
		}
	} else if sol, handled, err := e.closedForm(t, residual, variable.Text, p); handled {
		return e.solved(t, p, sol, err)
	}
	sol, err := e.bisect(t, residual, variable.Text, span, p)
	return e.solved(t, p, sol, err)
}

func (e *Env) solved(t *parse.Call, p *pass, sol value.Solution, err error) (value.Value, error) {
	if err != nil {
		return value.Value{}, err
	}
	e.record(t, p, prov.Node{Kind: prov.Function, Label: "solve", Detail: sol.Method})
	return value.Solved(sol), nil
}

func residualOf(n parse.Node) parse.Node {
	if b, ok := n.(*parse.Binary); ok && (b.Op == "=" || b.Op == "==") {
		return &parse.Binary{Op: "-", L: b.L, R: b.R, P: b.P}
	}
	return n
}

type searchSpan struct{ lo, hi num.Exact }

func (e *Env) explicitSpan(t *parse.Call, p *pass) (searchSpan, error) {
	var bounds [2]num.Exact
	for i, arg := range t.Args[2:] {
		v, err := e.eval(arg, p)
		if err != nil {
			return searchSpan{}, err
		}
		if !v.IsScalar() || !v.Num.IsExact() {
			return searchSpan{}, &lex.Error{Pos: t.P, Msg: "the search bounds of solve must be exact plain numbers"}
		}
		bounds[i] = v.Num.Exact()
	}
	if bounds[0].Cmp(bounds[1]) >= 0 {
		return searchSpan{}, &lex.Error{Pos: t.P, Msg: "the lower search bound of solve must be below the upper bound"}
	}
	return searchSpan{lo: bounds[0], hi: bounds[1]}, nil
}

func (e *Env) closedForm(t *parse.Call, residual parse.Node, variable string, p *pass) (value.Solution, bool, error) {
	poly, ok := solve.Extract(residual, variable, e.exactConstant)
	if !ok {
		return value.Solution{}, false, nil
	}
	equation := parse.Reading(t.Args[0])
	switch poly.Degree() {
	case 1:
		return e.closedFormResult(t, solve.Linear(poly), variable, methodLinear, equation)
	case 2:
		r, err := solve.Quadratic(poly, p.prec())
		if err != nil {
			return value.Solution{}, true, &lex.Error{Pos: t.P, Msg: "solve: " + err.Error()}
		}
		return e.closedFormResult(t, r, variable, methodQuadratic, equation)
	case 0, -1:
		return e.closedFormResult(t, solve.Linear(poly), variable, methodLinear, equation)
	}
	return value.Solution{}, false, nil
}

func (e *Env) exactConstant(name string) (num.Exact, bool) {
	v, ok := e.vars[name]
	if !ok || !v.IsScalar() || v.IsMeasured() || v.IsBoolean() || !v.Num.IsExact() {
		return num.Exact{}, false
	}
	return v.Num.Exact(), true
}

func (e *Env) closedFormResult(t *parse.Call, r solve.Result, variable, method, equation string) (value.Solution, bool, error) {
	switch r.Outcome {
	case solve.Identity:
		return value.Solution{}, true, &lex.Error{Pos: t.P, Msg: equation + " holds for every value of " + variable + ": there is nothing to solve for"}
	case solve.Contradiction:
		return value.Solution{}, true, &lex.Error{Pos: t.P, Msg: equation + " has no solution: " + variable + " drops out and what is left is false"}
	case solve.NoRealRoot:
		return value.Solution{}, true, &lex.Error{Pos: t.P, Msg: equation + " has no real root: the discriminant is " + r.Discriminant.String() + ", which is below zero"}
	}
	return value.Solution{Var: variable, Roots: r.Roots, Method: method}, true, nil
}

type residualProbe struct {
	env      *Env
	node     parse.Node
	variable string
	pass     *pass
	firstErr error
}

func (e *Env) probeFor(residual parse.Node, variable string, p *pass) *residualProbe {
	scratch := e.child()
	scratch.bound = &boundArgs{measured: map[*parse.Call][][]bool{}}
	return &residualProbe{env: scratch, node: residual, variable: variable, pass: p}
}

func (r *residualProbe) at(x value.Value) (value.Value, error) {
	r.env.vars[r.variable] = x
	r.env.arena = prov.NewArena()
	fresh := &pass{ctx: r.pass.ctx, prov: map[parse.Node]prov.NodeID{}, divisor: map[parse.Node]bool{}}
	v, err := r.env.eval(r.node, fresh)
	if err == nil && (!v.IsScalar() || v.IsMeasured() || v.IsBoolean()) {
		err = fmt.Errorf("solve needs an equation between plain numbers")
	}
	return v, err
}

func (r *residualProbe) sign(x num.Exact) (int, bool) {
	v, err := r.at(value.Scalar(x))
	if err != nil {
		if r.firstErr == nil {
			r.firstErr = err
		}
		return 0, false
	}
	if v.Num.IsExact() {
		return v.Num.Exact().Sign(), true
	}
	return v.Num.Ball(r.pass.prec()).Sign()
}

func (r *residualProbe) continuousOn(lo, hi num.Exact) bool {
	_, err := r.at(value.ScalarNum(num.NumInterval(lo, hi)))
	return err == nil
}

func (e *Env) bisect(t *parse.Call, residual parse.Node, variable string, span searchSpan, p *pass) (value.Solution, error) {
	probe := e.probeFor(residual, variable, p)
	scan := solve.Grid(probe.sign, span.lo, span.hi, searchSteps)
	searched := "[" + span.lo.String() + ", " + span.hi.String() + "]"
	if scan.Decided == 0 {
		if probe.firstErr != nil {
			return value.Solution{}, probe.firstErr
		}
		return value.Solution{}, &lex.Error{Pos: t.P, Msg: "solve could not evaluate the equation anywhere in " + searched}
	}
	roots := make([]num.Number, 0, len(scan.Exact)+len(scan.Brackets))
	for _, x := range scan.Exact {
		roots = append(roots, num.NumExact(x))
	}
	rejected := 0
	for _, b := range scan.Brackets {
		found, ok := solve.Refine(probe.sign, probe.continuousOn, b, p.prec()/2)
		if !ok {
			rejected++
			continue
		}
		roots = append(roots, rootNumber(found, p.prec()))
	}
	if len(roots) == 0 {
		return value.Solution{}, &lex.Error{Pos: t.P, Msg: noRootMessage(variable, searched, rejected)}
	}
	prec := p.prec()
	sort.Slice(roots, func(i, j int) bool { return roots[i].Ball(prec).Mid.Cmp(roots[j].Ball(prec).Mid) < 0 })
	return value.Solution{Var: variable, Roots: roots, Method: methodBisection, Searched: searched}, nil
}

func rootNumber(f solve.Found, prec uint) num.Number {
	if f.Exact() {
		return num.NumExact(f.Lo)
	}
	return num.NumBall(num.NumInterval(f.Lo, f.Hi).Ball(prec))
}

func noRootMessage(variable, searched string, rejected int) string {
	msg := "no root found for " + variable + " in " + searched + ": the equation never changed sign on the search grid, which does not prove there is no root (a root where the curve only touches zero would be missed)"
	if rejected > 0 {
		msg += "; " + fmt.Sprint(rejected) + " sign change(s) came from a discontinuity, not a root"
	}
	return msg
}

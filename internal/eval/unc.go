package eval

import (
	"errors"
	"math"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/suggest"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func numFloat(n num.Number, prec uint) float64 {
	if n.IsExact() {
		f, _ := n.Exact().Rat().Float64()
		return f
	}
	f, _ := n.Ball(prec).Mid.Float64()
	return f
}

func siFloat(v value.Value, prec uint) float64 { return numFloat(v.SI(prec), prec) }

func measured(vs ...value.Value) bool {
	for _, v := range vs {
		if v.IsMeasured() {
			return true
		}
	}
	return false
}

func cannotPropagate(pos lex.Pos, what string) error {
	return &lex.Error{Pos: pos, Msg: what + " cannot carry a measured uncertainty through first-order propagation"}
}

func isMeasurement(n parse.Node) bool {
	b, ok := n.(*parse.Binary)
	return ok && (b.Op == "±" || b.Op == "+-")
}

func countMeasurements(n parse.Node) int {
	switch t := n.(type) {
	case *parse.Unary:
		return countMeasurements(t.X)
	case *parse.Binary:
		k := countMeasurements(t.L) + countMeasurements(t.R)
		if isMeasurement(t) {
			k++
		}
		return k
	case *parse.Call:
		k := 0
		for _, a := range t.Args {
			k += countMeasurements(a)
		}
		return k
	case *parse.Assign:
		return countMeasurements(t.Body)
	}
	return 0
}

func (e *Env) measure(t *parse.Binary, p *pass) (value.Value, error) {
	l, err := e.operand(t.L, p)
	if err != nil {
		return value.Value{}, err
	}
	r, err := e.operand(t.R, p)
	if err != nil {
		return value.Value{}, err
	}
	if measured(l, r) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a measurement carries one ± only"}
	}
	if r.IsAffine() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "the ± must be a temperature difference such as 0.5 K, not the absolute temperature " + unitOrNone(r)}
	}
	out := l
	switch {
	case l.IsScalar() && !r.IsScalar():
		out.Dim, out.Unit, out.Factor, out.Offset = r.Dim, r.Unit, r.FactorOrOne(), r.Offset
	case l.Dim != r.Dim:
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "the uncertainty of " + unitOrNone(l) + " cannot be given in " + unitOrNone(r)}
	}
	prec := p.prec()
	name := t.L.String()
	if p.naming != "" {
		name, p.naming = p.naming, ""
	}
	line := t.P.Line
	if e.line > 0 {
		line = e.line
	}
	sigma := math.Abs(siFloat(r, prec))
	id := e.table.Declare(name, siFloat(out, prec), sigma, line)
	e.record(t, p, prov.Node{Kind: prov.Input, Label: name, Detail: inputDetail(out, sigma, prec), Line: line})
	p.uncertain = true
	return out.WithJet(unc.Jet{Grad: unc.Sens{id: 1}}), nil
}

func (e *Env) declareCorr(t *parse.Assign, p *pass) (value.Value, error) {
	rho, err := e.eval(t.Body, p)
	if err != nil {
		return value.Value{}, err
	}
	if !rho.IsScalar() || rho.IsMeasured() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a correlation coefficient is a plain number"}
	}
	ids := make([]unc.InputID, 2)
	for i, name := range t.Params {
		v, ok := e.vars[name]
		if !ok {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: suggest.Did("unknown name "+name, name, e.nameCandidates())}
		}
		id, ok := unc.Elementary(v.Sens)
		if !ok {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "corr needs two declared measurements: " + name + " is not one"}
		}
		ids[i] = id
	}
	if err := e.table.SetCorr(ids[0], ids[1], numFloat(rho.SI(p.prec()), p.prec())); err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: err.Error()}
	}
	return rho, nil
}

func (e *Env) expand(t *parse.Call, args []value.Value, p *pass) (value.Value, error) {
	if len(args) != 2 {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "expand takes a value and a coverage factor"}
	}
	if e.table.Degenerate(args[0].Sens, args[0].Curv) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "the first-order terms cancel here, so there is no standard uncertainty to expand"}
	}
	if len(args[0].Sens) == 0 {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "expand needs a value with a declared uncertainty"}
	}
	if !args[1].IsScalar() || args[1].IsMeasured() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "the coverage is a plain number such as 2 or 95%"}
	}
	x := siFloat(args[1], p.prec())
	out := args[0]
	if isPercentNode(t.Args[1]) || (x > 0 && x < 1) {
		k, ok := unc.FactorFor(x)
		if !ok {
			return value.Value{}, &lex.Error{Pos: t.P, Msg: "a coverage probability lies strictly between 0 and 1"}
		}
		out.Unc.Cover = unc.Coverage{K: k, Label: unc.ProbabilityLabel(x, k)}
		return out, nil
	}
	if !(x > 0) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a coverage factor is greater than zero"}
	}
	out.Unc.Cover = unc.Coverage{K: x, Label: unc.FactorLabel(x)}
	return out, nil
}

type partials struct {
	d1, d2 float64
}

var errOutsideDomain = errors.New("the derivative does not exist here")

func logPartials(x float64) (partials, error) {
	if x == 0 {
		return partials{}, errOutsideDomain
	}
	return partials{1 / x, -1 / (x * x)}, nil
}

func scaledLog(x, ln float64) (partials, error) {
	p, err := logPartials(x)
	if err != nil {
		return p, err
	}
	return partials{p.d1 / ln, p.d2 / ln}, nil
}

func arcPartials(x, sign float64) (partials, error) {
	d := 1 - x*x
	if d <= 0 {
		return partials{}, errOutsideDomain
	}
	root := math.Sqrt(d)
	return partials{sign / root, sign * x / (d * root)}, nil
}

var elemPartials = map[string]func(x, y float64) (partials, error){
	"exp": func(x, y float64) (partials, error) { return partials{y, y}, nil },
	"ln":  func(x, y float64) (partials, error) { return logPartials(x) },
	"log": func(x, y float64) (partials, error) { return logPartials(x) },
	"log10": func(x, y float64) (partials, error) {
		return scaledLog(x, math.Ln10)
	},
	"log2": func(x, y float64) (partials, error) {
		return scaledLog(x, math.Ln2)
	},
	"sinh": func(x, y float64) (partials, error) { return partials{math.Cosh(x), y}, nil },
	"cosh": func(x, y float64) (partials, error) { return partials{math.Sinh(x), y}, nil },
	"tanh": func(x, y float64) (partials, error) {
		s := 1 - y*y
		return partials{s, -2 * y * s}, nil
	},
	"sin": func(x, y float64) (partials, error) { return partials{math.Cos(x), -y}, nil },
	"cos": func(x, y float64) (partials, error) { return partials{-math.Sin(x), -y}, nil },
	"tan": func(x, y float64) (partials, error) {
		s := 1 + y*y
		return partials{s, 2 * y * s}, nil
	},
	"asin": func(x, y float64) (partials, error) { return arcPartials(x, 1) },
	"acos": func(x, y float64) (partials, error) { return arcPartials(x, -1) },
	"atan": func(x, y float64) (partials, error) {
		d := 1 + x*x
		return partials{1 / d, -2 * x / (d * d)}, nil
	},
	"sqrt": func(x, y float64) (partials, error) {
		if y == 0 {
			return partials{}, errOutsideDomain
		}
		return partials{1 / (2 * y), -1 / (4 * y * y * y)}, nil
	},
}

func elemJet(name string, in unc.Jet, x, y float64) (unc.Jet, error) {
	if in.IsZero() {
		return unc.Jet{}, nil
	}
	f, ok := elemPartials[name]
	if !ok {
		return unc.Jet{}, errors.New(name + " has no declared derivative")
	}
	d, err := f(x, y)
	if err != nil {
		return unc.Jet{}, err
	}
	return unc.Unary(in, d.d1, d.d2), nil
}

func powerJet(a, b unc.Jet, base, exponent float64) (unc.Jet, error) {
	if a.IsZero() && b.IsZero() {
		return unc.Jet{}, nil
	}
	if b.IsZero() {
		if base == 0 && exponent < 2 {
			return unc.Jet{}, errOutsideDomain
		}
		d1 := exponent * math.Pow(base, exponent-1)
		d2 := exponent * (exponent - 1) * math.Pow(base, exponent-2)
		return unc.Unary(a, d1, d2), nil
	}
	if base <= 0 {
		return unc.Jet{}, errors.New("a measured exponent needs a base greater than zero")
	}
	ln := math.Log(base)
	y := math.Pow(base, exponent)
	fa := exponent * math.Pow(base, exponent-1)
	fb := y * ln
	faa := exponent * (exponent - 1) * math.Pow(base, exponent-2)
	fbb := y * ln * ln
	fab := math.Pow(base, exponent-1) * (1 + exponent*ln)
	return unc.Compose(a, fa, faa, b, fb, fbb, fab), nil
}

func atan2Jet(y, x unc.Jet, yv, xv float64) (unc.Jet, error) {
	if y.IsZero() && x.IsZero() {
		return unc.Jet{}, nil
	}
	r2 := xv*xv + yv*yv
	if r2 == 0 {
		return unc.Jet{}, errOutsideDomain
	}
	fy := xv / r2
	fx := -yv / r2
	fyy := -2 * xv * yv / (r2 * r2)
	fxy := (yv*yv - xv*xv) / (r2 * r2)
	fxx := 2 * xv * yv / (r2 * r2)
	return unc.Compose(y, fy, fyy, x, fx, fxx, fxy), nil
}

func productJet(l, r value.Value, prec uint) unc.Jet {
	if !measured(l, r) {
		return unc.Jet{}
	}
	return unc.Compose(l.Jet(), siFloat(r, prec), 0, r.Jet(), siFloat(l, prec), 0, 1)
}

func quotientJet(l, r value.Value, prec uint) (unc.Jet, error) {
	if !measured(l, r) {
		return unc.Jet{}, nil
	}
	a, b := siFloat(l, prec), siFloat(r, prec)
	if b == 0 {
		return unc.Jet{}, errOutsideDomain
	}
	q := a / b
	return unc.Compose(l.Jet(), 1/b, 0, r.Jet(), -q/b, 2*q/(b*b), -1/(b*b)), nil
}

func sumJet(l, r value.Value, op string) unc.Jet {
	if !measured(l, r) {
		return unc.Jet{}
	}
	return unc.Compose(l.Jet(), 1, 0, r.Jet(), signOf(op), 0, 0)
}

func percentJet(l, r value.Value, op string, prec uint) unc.Jet {
	if !measured(l, r) {
		return unc.Jet{}
	}
	sign := signOf(op)
	a, b := siFloat(l, prec), siFloat(r, prec)
	return unc.Compose(l.Jet(), 1+sign*b, 0, r.Jet(), sign*a, 0, sign)
}

func signOf(op string) float64 {
	if op == "-" {
		return -1
	}
	return 1
}

func exactFloat(x num.Exact) float64 {
	f, _ := x.Rat().Float64()
	return f
}

func readingNote(name *parse.Name, u units.Unit, bound value.Value) parse.Note {
	unit := "the unit " + name.Text + " (" + u.Dim.Words() + ")"
	variable := "the variable " + name.Text
	switch {
	case bound.IsMeasured():
		return parse.Note{Pos: name.P, Msg: "read as " + variable + ", not " + unit + "; the variable carries a measured uncertainty"}
	case !bound.IsScalar():
		return parse.Note{Pos: name.P, Msg: "read as " + variable + ", not " + unit + "; the variable already carries a unit"}
	}
	return parse.Note{Pos: name.P, Msg: "read as " + unit + ", not " + variable + "; the variable is a plain number"}
}

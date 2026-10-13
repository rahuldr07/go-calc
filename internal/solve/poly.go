package solve

import (
	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
)

const maxExponent = 64

type Poly []num.Exact

func constant(c num.Exact) Poly { return Poly{c} }

func (p Poly) trimmed() Poly {
	n := len(p)
	for n > 0 && p[n-1].Sign() == 0 {
		n--
	}
	return p[:n]
}

func (p Poly) Degree() int { return len(p.trimmed()) - 1 }

func (p Poly) Coefficient(i int) num.Exact { return p.coefficient(i) }

func (p Poly) coefficient(i int) num.Exact {
	if i < len(p) {
		return p[i]
	}
	return num.FromInt(0)
}

func (p Poly) add(q Poly) Poly {
	n := max(len(p), len(q))
	out := make(Poly, n)
	for i := range out {
		out[i] = p.coefficient(i).Add(q.coefficient(i))
	}
	return out
}

func (p Poly) neg() Poly {
	out := make(Poly, len(p))
	for i, c := range p {
		out[i] = c.Neg()
	}
	return out
}

func (p Poly) mul(q Poly) Poly {
	if len(p) == 0 || len(q) == 0 {
		return Poly{}
	}
	out := make(Poly, len(p)+len(q)-1)
	for i := range out {
		out[i] = num.FromInt(0)
	}
	for i, a := range p {
		for j, b := range q {
			out[i+j] = out[i+j].Add(a.Mul(b))
		}
	}
	return out
}

func (p Poly) isConstant() bool { return p.Degree() <= 0 }

func Extract(n parse.Node, variable string, known func(name string) (num.Exact, bool)) (Poly, bool) {
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return nil, false
		}
		return constant(x), true
	case *parse.Name:
		if t.Text == variable {
			return Poly{num.FromInt(0), num.FromInt(1)}, true
		}
		if c, ok := known(t.Text); ok {
			return constant(c), true
		}
		return nil, false
	case *parse.Unary:
		if t.Op != "-" {
			return nil, false
		}
		x, ok := Extract(t.X, variable, known)
		if !ok {
			return nil, false
		}
		return x.neg(), true
	case *parse.Binary:
		return extractBinary(t, variable, known)
	}
	return nil, false
}

func extractBinary(t *parse.Binary, variable string, known func(name string) (num.Exact, bool)) (Poly, bool) {
	l, ok := Extract(t.L, variable, known)
	if !ok {
		return nil, false
	}
	r, ok := Extract(t.R, variable, known)
	if !ok {
		return nil, false
	}
	switch t.Op {
	case "+":
		return l.add(r), true
	case "-":
		return l.add(r.neg()), true
	case "*", "×", "·":
		return l.mul(r), true
	case "/", "÷":
		return divideByConstant(l, r)
	case "^":
		return raise(l, r)
	}
	return nil, false
}

func divideByConstant(l, r Poly) (Poly, bool) {
	if !r.isConstant() || r.coefficient(0).Sign() == 0 {
		return nil, false
	}
	out := make(Poly, len(l))
	for i, c := range l {
		q, err := c.Div(r.coefficient(0))
		if err != nil {
			return nil, false
		}
		out[i] = q
	}
	return out, true
}

func raise(base, exponent Poly) (Poly, bool) {
	if !exponent.isConstant() {
		return nil, false
	}
	k := exponent.coefficient(0)
	if !k.IsInt() || k.Sign() < 0 {
		return nil, false
	}
	whole := k.Rat().Num()
	if !whole.IsInt64() || whole.Int64() > maxExponent {
		return nil, false
	}
	out := constant(num.FromInt(1))
	for i := int64(0); i < whole.Int64(); i++ {
		out = out.mul(base)
	}
	return out, true
}

type Outcome int

const (
	Roots Outcome = iota
	Identity
	Contradiction
	NoRealRoot
)

type Result struct {
	Outcome      Outcome
	Roots        []num.Number
	Discriminant num.Exact
}

func Linear(p Poly) Result {
	p = p.trimmed()
	if len(p) == 0 {
		return Result{Outcome: Identity}
	}
	if len(p) == 1 {
		return Result{Outcome: Contradiction}
	}
	root, _ := p[0].Neg().Div(p[1])
	return Result{Outcome: Roots, Roots: []num.Number{num.NumExact(root)}}
}

func Quadratic(p Poly, prec uint) (Result, error) {
	p = p.trimmed()
	a, b, c := p.coefficient(2), p.coefficient(1), p.coefficient(0)
	disc := b.Mul(b).Sub(num.FromInt(4).Mul(a).Mul(c))
	switch disc.Sign() {
	case -1:
		return Result{Outcome: NoRealRoot, Discriminant: disc}, nil
	case 0:
		root, err := b.Neg().Div(num.FromInt(2).Mul(a))
		if err != nil {
			return Result{}, err
		}
		return Result{Outcome: Roots, Roots: []num.Number{num.NumExact(root)}, Discriminant: disc}, nil
	}
	twoA := num.FromInt(2).Mul(a)
	if s, ok := num.Root(disc, 2); ok {
		first, err := b.Neg().Sub(s).Div(twoA)
		if err != nil {
			return Result{}, err
		}
		second, err := b.Neg().Add(s).Div(twoA)
		if err != nil {
			return Result{}, err
		}
		return ordered(Result{Outcome: Roots, Roots: []num.Number{num.NumExact(first), num.NumExact(second)}, Discriminant: disc}), nil
	}
	root, err := elem.Sqrt(num.BallFromExact(disc, prec), prec)
	if err != nil {
		return Result{}, err
	}
	s := num.NumBall(root)
	front := num.NumExact(b.Neg())
	denominator := num.NumExact(twoA)
	first, err := front.Sub(s, prec).Div(denominator, prec)
	if err != nil {
		return Result{}, err
	}
	second, err := front.Add(s, prec).Div(denominator, prec)
	if err != nil {
		return Result{}, err
	}
	return ordered(Result{Outcome: Roots, Roots: []num.Number{first, second}, Discriminant: disc}), nil
}

func ordered(r Result) Result {
	if len(r.Roots) == 2 && r.Roots[0].Ball(120).Mid.Cmp(r.Roots[1].Ball(120).Mid) > 0 {
		r.Roots[0], r.Roots[1] = r.Roots[1], r.Roots[0]
	}
	return r
}

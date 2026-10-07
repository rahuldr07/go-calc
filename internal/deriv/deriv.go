package deriv

import (
	"math/big"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
)

const (
	maxDepth         = 500
	maxCost          = 200000
	nodeCost         = 16
	maxSimplifySteps = 1000000
	maxFoldedBits    = 4096
	maxFoldedPower   = 1000
)

type differ struct {
	variable string
	cost     int
	failure  string
}

func Differentiate(n parse.Node, variable string) (result parse.Node, err error) {
	if n == nil {
		return nil, &lex.Error{Msg: "nothing to differentiate"}
	}
	if variable == "" {
		return nil, &lex.Error{Pos: n.Pos(), Msg: "differentiate with respect to what? no variable was named"}
	}
	defer func() {
		if recover() != nil {
			result, err = nil, &lex.Error{Pos: n.Pos(), Msg: "cannot differentiate this expression"}
		}
	}()
	d := &differ{variable: variable}
	out, err := d.diff(n, 0)
	if err != nil {
		return nil, err
	}
	if d.failure != "" {
		return nil, &lex.Error{Pos: n.Pos(), Msg: d.failure}
	}
	return out, nil
}

func (d *differ) fail(p lex.Pos, msg string) error {
	return &lex.Error{Pos: p, Msg: msg}
}

func (d *differ) depends(n parse.Node, depth int) bool {
	if depth > maxDepth {
		return true
	}
	switch t := n.(type) {
	case *parse.Name:
		return t.Text == d.variable
	case *parse.Unary:
		return d.depends(t.X, depth+1)
	case *parse.Binary:
		return d.depends(t.L, depth+1) || d.depends(t.R, depth+1)
	case *parse.Call:
		for _, a := range t.Args {
			if d.depends(a, depth+1) {
				return true
			}
		}
	case *parse.Assign:
		return true
	}
	return false
}

func (d *differ) clone(n parse.Node, depth int) parse.Node {
	if depth > maxDepth {
		d.failure = "too deeply nested to differentiate"
		return n
	}
	d.cost++
	if d.cost > maxCost {
		d.failure = "the derivative is too large to write out"
		return n
	}
	switch t := n.(type) {
	case *parse.NumLit:
		c := *t
		return &c
	case *parse.Name:
		c := *t
		return &c
	case *parse.Unary:
		c := *t
		c.X = d.clone(t.X, depth+1)
		return &c
	case *parse.Binary:
		c := *t
		c.L = d.clone(t.L, depth+1)
		c.R = d.clone(t.R, depth+1)
		return &c
	case *parse.Call:
		c := *t
		c.Args = make([]parse.Node, len(t.Args))
		for i, a := range t.Args {
			c.Args[i] = d.clone(a, depth+1)
		}
		c.Keys = append([]string(nil), t.Keys...)
		return &c
	}
	return n
}

func (d *differ) diff(n parse.Node, depth int) (parse.Node, error) {
	if n == nil {
		return nil, &lex.Error{Msg: "nothing to differentiate"}
	}
	p := n.Pos()
	if depth > maxDepth {
		return nil, d.fail(p, "too deeply nested to differentiate")
	}
	d.cost += nodeCost
	if d.cost > maxCost {
		return nil, d.fail(p, "the derivative is too large to write out")
	}
	if _, defined := n.(*parse.Assign); defined {
		return nil, d.fail(p, "cannot differentiate a definition; differentiate the expression on its right")
	}
	if !d.depends(n, depth) {
		return integer(0, p), nil
	}
	switch t := n.(type) {
	case *parse.Name:
		return integer(1, p), nil
	case *parse.Unary:
		return d.unary(t, depth)
	case *parse.Binary:
		if t.Implicit && parse.InDenominator(t.L) {
			return nil, d.fail(p, "this implicit product after a division can be read two ways; add brackets before differentiating")
		}
		return d.binary(t, depth)
	case *parse.Call:
		return d.call(t, depth)
	}
	return nil, d.fail(p, "cannot differentiate this expression")
}

func (d *differ) unary(t *parse.Unary, depth int) (parse.Node, error) {
	switch t.Op {
	case "-":
		dx, err := d.diff(t.X, depth+1)
		if err != nil {
			return nil, err
		}
		return negate(dx, t.P), nil
	case "%":
		return nil, d.fail(t.P, "cannot differentiate a percentage: what it means depends on the number beside it")
	case "!":
		return nil, d.fail(t.P, "cannot differentiate a factorial")
	}
	return nil, d.fail(t.P, "cannot differentiate the operator "+t.Op)
}

func (d *differ) binary(t *parse.Binary, depth int) (parse.Node, error) {
	p := t.P
	if (t.Op == "+" || t.Op == "-") && (isPercentage(t.L) || isPercentage(t.R)) {
		return nil, d.fail(p, "cannot differentiate a percentage added to or taken from a number: it is a share of the other term")
	}
	lVaries, rVaries := d.depends(t.L, depth+1), d.depends(t.R, depth+1)
	var dl, dr parse.Node
	var err error
	if lVaries {
		if dl, err = d.diff(t.L, depth+1); err != nil {
			return nil, err
		}
	}
	if rVaries {
		if dr, err = d.diff(t.R, depth+1); err != nil {
			return nil, err
		}
	}
	switch t.Op {
	case "+":
		return sum(dl, dr, p), nil
	case "-":
		if !lVaries {
			return negate(dr, p), nil
		}
		if !rVaries {
			return dl, nil
		}
		return binary("-", dl, dr, p), nil
	case "*", "×", "·", "of":
		return d.product(t, dl, dr), nil
	case "/", "÷":
		return d.quotient(t, dl, dr), nil
	case "^":
		return d.power(t, dl, dr), nil
	}
	return nil, d.fail(p, "cannot differentiate the operator "+t.Op)
}

func isPercentage(n parse.Node) bool {
	u, ok := n.(*parse.Unary)
	return ok && u.Op == "%"
}

func sum(dl, dr parse.Node, p lex.Pos) parse.Node {
	if dl == nil {
		return dr
	}
	if dr == nil {
		return dl
	}
	return binary("+", dl, dr, p)
}

func (d *differ) product(t *parse.Binary, dl, dr parse.Node) parse.Node {
	p := t.P
	switch {
	case dl == nil:
		return binary("*", d.clone(t.L, 0), dr, p)
	case dr == nil:
		return binary("*", dl, d.clone(t.R, 0), p)
	}
	left := binary("*", dl, d.clone(t.R, 0), p)
	right := binary("*", d.clone(t.L, 0), dr, p)
	return binary("+", left, right, p)
}

func (d *differ) quotient(t *parse.Binary, dl, dr parse.Node) parse.Node {
	p := t.P
	squared := func() parse.Node { return binary("^", d.clone(t.R, 0), integer(2, p), p) }
	switch {
	case dr == nil:
		return binary("/", dl, d.clone(t.R, 0), p)
	case dl == nil:
		top := negate(binary("*", d.clone(t.L, 0), dr, p), p)
		return binary("/", top, squared(), p)
	}
	top := binary("-", binary("*", dl, d.clone(t.R, 0), p), binary("*", d.clone(t.L, 0), dr, p), p)
	return binary("/", top, squared(), p)
}

func (d *differ) power(t *parse.Binary, df, dg parse.Node) parse.Node {
	p := t.P
	if dg == nil {
		return d.constantExponent(t, df)
	}
	if df == nil {
		return d.constantBase(t, dg)
	}
	f, g := d.clone(t.L, 0), d.clone(t.R, 0)
	whole := binary("^", f, g, p)
	rate := binary("+",
		binary("*", dg, call("ln", p, d.clone(t.L, 0)), p),
		binary("/", binary("*", d.clone(t.R, 0), df, p), d.clone(t.L, 0), p),
		p)
	return binary("*", whole, rate, p)
}

func (d *differ) constantExponent(t *parse.Binary, df parse.Node) parse.Node {
	p := t.P
	if v, ok := integerValue(t.R); ok && v.IsInt64() && v.Int64() == 1 {
		return df
	}
	lowered := binary("^", d.clone(t.L, 0), binary("-", d.clone(t.R, 0), integer(1, p), p), p)
	return binary("*", binary("*", d.clone(t.R, 0), lowered, p), df, p)
}

func (d *differ) constantBase(t *parse.Binary, dg parse.Node) parse.Node {
	p := t.P
	whole := binary("^", d.clone(t.L, 0), d.clone(t.R, 0), p)
	if name, ok := t.L.(*parse.Name); ok && name.Text == "e" {
		return binary("*", whole, dg, p)
	}
	return binary("*", binary("*", whole, call("ln", p, d.clone(t.L, 0)), p), dg, p)
}

func (d *differ) call(t *parse.Call, depth int) (parse.Node, error) {
	p := t.P
	rule, known := outerDerivatives[t.Fn]
	if !known {
		return nil, d.fail(p, "cannot differentiate "+t.Fn+": no derivative rule for it")
	}
	if len(t.Args) != 1 || len(t.Keys) > 0 {
		return nil, d.fail(p, t.Fn+" takes one argument to differentiate")
	}
	inner, err := d.diff(t.Args[0], depth+1)
	if err != nil {
		return nil, err
	}
	outer := rule(d.clone(t.Args[0], 0), d.clone(t.Args[0], 0), p)
	return binary("*", outer, inner, p), nil
}

var outerDerivatives = map[string]func(u, again parse.Node, p lex.Pos) parse.Node{
	"sin": func(u, _ parse.Node, p lex.Pos) parse.Node { return call("cos", p, u) },
	"cos": func(u, _ parse.Node, p lex.Pos) parse.Node { return negate(call("sin", p, u), p) },
	"tan": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("^", call("cos", p, u), integer(2, p), p), p)
	},
	"asin": func(u, _ parse.Node, p lex.Pos) parse.Node { return reciprocalRootOneMinusSquare(u, p) },
	"acos": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return negate(reciprocalRootOneMinusSquare(u, p), p)
	},
	"atan": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("+", integer(1, p), binary("^", u, integer(2, p), p), p), p)
	},
	"sinh": func(u, _ parse.Node, p lex.Pos) parse.Node { return call("cosh", p, u) },
	"cosh": func(u, _ parse.Node, p lex.Pos) parse.Node { return call("sinh", p, u) },
	"tanh": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("^", call("cosh", p, u), integer(2, p), p), p)
	},
	"exp": func(u, _ parse.Node, p lex.Pos) parse.Node { return call("exp", p, u) },
	"ln":  func(u, _ parse.Node, p lex.Pos) parse.Node { return binary("/", integer(1, p), u, p) },
	"log": func(u, _ parse.Node, p lex.Pos) parse.Node { return binary("/", integer(1, p), u, p) },
	"log2": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("*", u, call("ln", p, integer(2, p)), p), p)
	},
	"log10": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("*", u, call("ln", p, integer(10, p)), p), p)
	},
	"sqrt": func(u, _ parse.Node, p lex.Pos) parse.Node {
		return binary("/", integer(1, p), binary("*", integer(2, p), call("sqrt", p, u), p), p)
	},
	"abs": func(u, again parse.Node, p lex.Pos) parse.Node {
		return binary("/", u, call("abs", p, again), p)
	},
}

func reciprocalRootOneMinusSquare(u parse.Node, p lex.Pos) parse.Node {
	inside := binary("-", integer(1, p), binary("^", u, integer(2, p), p), p)
	return binary("/", integer(1, p), call("sqrt", p, inside), p)
}

func integer(v int64, p lex.Pos) parse.Node { return integerNode(big.NewInt(v), p) }

func integerNode(v *big.Int, p lex.Pos) parse.Node {
	if v.Sign() < 0 {
		magnitude := &parse.NumLit{Text: new(big.Int).Neg(v).String(), P: p}
		return &parse.Unary{Op: "-", X: magnitude, P: p}
	}
	return &parse.NumLit{Text: v.String(), P: p}
}

func binary(op string, l, r parse.Node, p lex.Pos) parse.Node {
	return &parse.Binary{Op: op, L: l, R: r, P: p}
}

func negate(x parse.Node, p lex.Pos) parse.Node { return &parse.Unary{Op: "-", X: x, P: p} }

func call(fn string, p lex.Pos, args ...parse.Node) parse.Node {
	return &parse.Call{Fn: fn, Args: args, P: p}
}

func integerValue(n parse.Node) (*big.Int, bool) {
	switch t := n.(type) {
	case *parse.NumLit:
		for _, r := range t.Text {
			if r < '0' || r > '9' {
				return nil, false
			}
		}
		v, ok := new(big.Int).SetString(t.Text, 10)
		return v, ok
	case *parse.Unary:
		if t.Op != "-" {
			return nil, false
		}
		v, ok := integerValue(t.X)
		if !ok {
			return nil, false
		}
		return v.Neg(v), true
	}
	return nil, false
}

func isInteger(n parse.Node, want int64) bool {
	v, ok := integerValue(n)
	return ok && v.IsInt64() && v.Int64() == want
}

func isNegation(n parse.Node) (parse.Node, bool) {
	u, ok := n.(*parse.Unary)
	if !ok || u.Op != "-" {
		return nil, false
	}
	return u.X, true
}

type simplifier struct {
	steps int
}

func Simplify(n parse.Node) parse.Node {
	if n == nil {
		return nil
	}
	s := &simplifier{}
	return s.simplify(n, 0)
}

func (s *simplifier) simplify(n parse.Node, depth int) parse.Node {
	s.steps++
	if depth > maxDepth || s.steps > maxSimplifySteps {
		return n
	}
	switch t := n.(type) {
	case *parse.Unary:
		if t.Op != "-" {
			return n
		}
		return s.negation(t, s.simplify(t.X, depth+1))
	case *parse.Binary:
		l, r := s.simplify(t.L, depth+1), s.simplify(t.R, depth+1)
		return s.binary(t, l, r)
	case *parse.Call:
		c := *t
		c.Args = make([]parse.Node, len(t.Args))
		for i, a := range t.Args {
			c.Args[i] = s.simplify(a, depth+1)
		}
		return &c
	}
	return n
}

func (s *simplifier) negation(t *parse.Unary, x parse.Node) parse.Node {
	if inner, ok := isNegation(x); ok {
		return inner
	}
	if v, ok := integerValue(x); ok {
		return integerNode(v.Neg(v), t.P)
	}
	c := *t
	c.X = x
	return &c
}

func regroup(orig *parse.Binary, result parse.Node) parse.Node {
	if !orig.Grouped {
		return result
	}
	switch t := result.(type) {
	case *parse.Binary:
		c := *t
		c.Grouped = true
		return &c
	case *parse.Unary:
		c := *t
		c.Grouped = true
		return &c
	case *parse.Call:
		c := *t
		c.Grouped = true
		return &c
	}
	return result
}

func (s *simplifier) binary(t *parse.Binary, l, r parse.Node) parse.Node {
	rebuilt := func(l, r parse.Node) parse.Node {
		c := *t
		c.L, c.R = l, r
		return &c
	}
	p := t.P
	lv, lInt := integerValue(l)
	rv, rInt := integerValue(r)
	switch t.Op {
	case "+":
		switch {
		case lInt && lv.Sign() == 0:
			return regroup(t, r)
		case rInt && rv.Sign() == 0:
			return regroup(t, l)
		case lInt && rInt:
			return integerNode(new(big.Int).Add(lv, rv), p)
		}
		if x, ok := isNegation(r); ok && !t.Implicit {
			return regroup(t, &parse.Binary{Op: "-", L: l, R: x, P: p})
		}
		if x, ok := isNegation(l); ok && !t.Implicit {
			if _, alsoNegative := isNegation(r); !alsoNegative {
				return regroup(t, &parse.Binary{Op: "-", L: r, R: x, P: p})
			}
		}
	case "-":
		switch {
		case rInt && rv.Sign() == 0:
			return regroup(t, l)
		case lInt && lv.Sign() == 0:
			return regroup(t, s.negation(&parse.Unary{Op: "-", P: p}, r))
		case lInt && rInt:
			return integerNode(new(big.Int).Sub(lv, rv), p)
		}
		if x, ok := isNegation(r); ok {
			return regroup(t, &parse.Binary{Op: "+", L: l, R: x, P: p})
		}
	case "*", "×", "·":
		switch {
		case lInt && lv.Sign() == 0 && total(r, 0):
			return integer(0, p)
		case rInt && rv.Sign() == 0 && total(l, 0):
			return integer(0, p)
		case lInt && lv.IsInt64() && lv.Int64() == 1:
			return regroup(t, r)
		case rInt && rv.IsInt64() && rv.Int64() == 1:
			return regroup(t, l)
		case lInt && rInt:
			return integerNode(new(big.Int).Mul(lv, rv), p)
		case lInt && lv.IsInt64() && lv.Int64() == -1:
			return regroup(t, s.negation(&parse.Unary{Op: "-", P: p}, r))
		case rInt && rv.IsInt64() && rv.Int64() == -1:
			return regroup(t, s.negation(&parse.Unary{Op: "-", P: p}, l))
		}
		if !t.Implicit {
			return regroup(t, s.pulledNegation(t, l, r))
		}
	case "/", "÷":
		switch {
		case rInt && rv.IsInt64() && rv.Int64() == 1:
			return regroup(t, l)
		case lInt && rInt && rv.Sign() != 0:
			quotient, remainder := new(big.Int).QuoRem(lv, rv, new(big.Int))
			if remainder.Sign() == 0 {
				return integerNode(quotient, p)
			}
		case lInt && lv.Sign() == 0 && rInt && rv.Sign() != 0:
			return integer(0, p)
		}
		if x, ok := isNegation(l); ok {
			if _, divisorNegative := isNegation(r); !divisorNegative {
				return regroup(t, negate(&parse.Binary{Op: t.Op, L: x, R: r, P: p}, p))
			}
		}
	case "^":
		switch {
		case rInt && rv.Sign() == 0 && !(lInt && lv.Sign() == 0) && total(l, 0):
			return integer(1, p)
		case rInt && rv.IsInt64() && rv.Int64() == 1:
			return regroup(t, l)
		case lInt && lv.IsInt64() && lv.Int64() == 1 && total(r, 0):
			return integer(1, p)
		case lInt && rInt:
			if folded, ok := foldPower(lv, rv, p); ok {
				return folded
			}
		}
	}
	return rebuilt(l, r)
}

func (s *simplifier) pulledNegation(t *parse.Binary, l, r parse.Node) parse.Node {
	p := t.P
	lx, lNeg := isNegation(l)
	rx, rNeg := isNegation(r)
	switch {
	case lNeg && rNeg:
		return &parse.Binary{Op: t.Op, L: lx, R: rx, P: p}
	case lNeg:
		return negate(&parse.Binary{Op: t.Op, L: lx, R: r, P: p}, p)
	case rNeg:
		return negate(&parse.Binary{Op: t.Op, L: l, R: rx, P: p}, p)
	}
	c := *t
	c.L, c.R = l, r
	return &c
}

func foldPower(base, exponent *big.Int, p lex.Pos) (parse.Node, bool) {
	if exponent.Sign() < 0 || !exponent.IsInt64() || exponent.Int64() > maxFoldedPower {
		return nil, false
	}
	if base.Sign() == 0 && exponent.Sign() == 0 {
		return nil, false
	}
	if int64(base.BitLen())*exponent.Int64() > maxFoldedBits {
		return nil, false
	}
	return integerNode(new(big.Int).Exp(base, exponent, nil), p), true
}

func total(n parse.Node, depth int) bool {
	if depth > maxDepth {
		return false
	}
	switch t := n.(type) {
	case *parse.NumLit, *parse.Name:
		return true
	case *parse.Unary:
		return t.Op == "-" && total(t.X, depth+1)
	case *parse.Binary:
		switch t.Op {
		case "+", "-", "*", "×", "·":
			return total(t.L, depth+1) && total(t.R, depth+1)
		case "^":
			v, ok := integerValue(t.R)
			return ok && v.Sign() >= 0 && total(t.L, depth+1)
		}
	}
	return false
}

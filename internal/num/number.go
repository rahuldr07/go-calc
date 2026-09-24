package num

import "math/big"

type Kind int

const (
	KindExact Kind = iota
	KindBall
	KindInterval
)

type Number struct {
	kind Kind
	ex   Exact
	ball Ball
	lo   Exact
	hi   Exact
}

func NumExact(e Exact) Number { return Number{ex: e} }

func NumBall(b Ball) Number { return Number{kind: KindBall, ball: b} }

func NumInterval(lo, hi Exact) Number {
	if lo.Cmp(hi) > 0 {
		lo, hi = hi, lo
	}
	return Number{kind: KindInterval, lo: lo, hi: hi}
}

func (n Number) Kind() Kind { return n.kind }

func (n Number) IsExact() bool { return n.kind == KindExact }

func (n Number) Exact() Exact { return n.ex }

func magFromExact(e Exact) Mag {
	return Mag{newMag().SetRat(new(big.Rat).Abs(e.Rat()))}
}

var half = big.NewRat(1, 2)

func (n Number) Ball(prec uint) Ball {
	switch n.kind {
	case KindBall:
		return n.ball
	case KindInterval:
		center := FromRat(new(big.Rat).Mul(new(big.Rat).Add(n.lo.Rat(), n.hi.Rat()), half))
		spread := FromRat(new(big.Rat).Mul(new(big.Rat).Sub(n.hi.Rat(), n.lo.Rat()), half))
		mid := new(big.Float).SetPrec(prec).SetRat(center.Rat())
		return Ball{Mid: mid, Rad: magFromExact(spread).Add(RoundingError(mid, prec))}
	}
	return BallFromExact(n.ex, prec)
}

func (n Number) Add(o Number, prec uint) Number {
	if n.IsExact() && o.IsExact() {
		return NumExact(n.ex.Add(o.ex))
	}
	return NumBall(n.Ball(prec).Add(o.Ball(prec), prec))
}

func (n Number) Sub(o Number, prec uint) Number {
	if n.IsExact() && o.IsExact() {
		return NumExact(n.ex.Sub(o.ex))
	}
	return NumBall(n.Ball(prec).Sub(o.Ball(prec), prec))
}

func (n Number) Mul(o Number, prec uint) Number {
	if n.IsExact() && o.IsExact() {
		return NumExact(n.ex.Mul(o.ex))
	}
	return NumBall(n.Ball(prec).Mul(o.Ball(prec), prec))
}

func (n Number) Div(o Number, prec uint) (Number, error) {
	if n.IsExact() && o.IsExact() {
		q, err := n.ex.Div(o.ex)
		if err != nil {
			return Number{}, err
		}
		return NumExact(q), nil
	}
	b, err := n.Ball(prec).Div(o.Ball(prec), prec)
	if err != nil {
		return Number{}, err
	}
	return NumBall(b), nil
}

func (n Number) Pow(k int, prec uint) (Number, error) {
	if n.IsExact() {
		if k < 0 && n.ex.Sign() == 0 {
			return Number{}, ErrDivByZero
		}
		return NumExact(n.ex.Pow(k)), nil
	}
	b, err := n.Ball(prec).Pow(k, prec)
	if err != nil {
		return Number{}, err
	}
	return NumBall(b), nil
}

func (n Number) String() string {
	switch n.kind {
	case KindBall:
		return n.ball.String()
	case KindInterval:
		return "[" + n.lo.String() + " .. " + n.hi.String() + "]"
	}
	return n.ex.String()
}

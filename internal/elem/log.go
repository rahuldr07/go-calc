package elem

import (
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"
)

var reductionFloor = big.NewFloat(0.70710678118654752)

func atanhSeries(t num.Ball, prec uint) num.Ball {
	square := t.Mul(t, prec)
	spread := magOf(absUpper(square))
	if spread.Cmp(magOf(floatHalf)) > 0 {
		return widen(t, infiniteMag())
	}
	term := t
	sum := t
	bound := magOf(absUpper(t))
	target := magPow2(-int(prec))
	for n := int64(1); n < maxTerms; n++ {
		term = term.Mul(square, prec)
		sum = sum.Add(divInt(term, 2*n+1, prec), prec)
		bound = bound.Mul(spread)
		if magQuoInt(bound, 2*n+1).Cmp(target) <= 0 {
			return widen(sum, magScale(magQuoInt(bound.Mul(spread), 2*n+3), 1))
		}
	}
	return widen(sum, infiniteMag())
}

func logPoint(x *big.Float, prec uint) num.Ball {
	mant := new(big.Float)
	e := x.MantExp(mant)
	if mant.Cmp(reductionFloor) < 0 {
		mant.SetMantExp(mant, 1)
		e--
	}
	wp := prec + 16 + uint(bitsOf(e))
	m := exactBall(mant)
	one := intBall(1)
	t, err := m.Sub(one, wp).Div(m.Add(one, wp), wp)
	if err != nil {
		return widen(m, infiniteMag())
	}
	res := scaleBall(atanhSeries(t, wp), 1)
	if e != 0 {
		res = res.Add(ln2Ball(wp).Mul(intBall(int64(e)), wp), wp)
	}
	return res
}

func bitsOf(e int) int {
	if e < 0 {
		e = -e
	}
	n := 0
	for e > 0 {
		e >>= 1
		n++
	}
	return n
}

func logBall(x num.Ball, prec uint) (num.Ball, error) {
	lo := x.Lo()
	if lo.Sign() <= 0 {
		return num.Ball{}, ErrNonPositiveLog
	}
	core := logPoint(x.Mid, prec)
	if x.Rad.IsZero() {
		return core, nil
	}
	return widen(core, magQuoFloat(x.Rad, lo)), nil
}

func Log(x num.Ball, prec uint) (num.Ball, error) {
	b, err := logBall(normalize(x), workPrec(prec))
	if err != nil {
		return num.Ball{}, err
	}
	return finish(b, prec), nil
}

func Log2(x num.Ball, prec uint) (num.Ball, error) {
	wp := workPrec(prec)
	b, err := logBall(normalize(x), wp)
	if err != nil {
		return num.Ball{}, err
	}
	q, err := b.Div(ln2Ball(wp), wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(q, prec), nil
}

func Log10(x num.Ball, prec uint) (num.Ball, error) {
	wp := workPrec(prec)
	b, err := logBall(normalize(x), wp)
	if err != nil {
		return num.Ball{}, err
	}
	q, err := b.Div(logPoint(new(big.Float).SetPrec(wp).SetInt64(10), wp), wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(q, prec), nil
}

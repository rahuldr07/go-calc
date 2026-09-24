package elem

import (
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"
)

var (
	halvingLimit  = new(big.Float).SetMantExp(floatOne, -5)
	threeQuarters = big.NewFloat(0.75)
)

func sinSeries(r num.Ball, radius num.Mag, prec uint) num.Ball {
	square := r.Mul(r, prec)
	spread := radius.Mul(radius)
	term := r
	sum := r
	bound := radius
	target := magPow2(-int(prec))
	for n := int64(1); n < maxTerms; n++ {
		d := 2 * n * (2*n + 1)
		term = divInt(term.Mul(square, prec), d, prec).Neg()
		sum = sum.Add(term, prec)
		bound = magQuoInt(bound.Mul(spread), d)
		if bound.Cmp(target) <= 0 {
			return widen(sum, bound)
		}
	}
	return widen(sum, infiniteMag())
}

func cosSeries(r num.Ball, radius num.Mag, prec uint) num.Ball {
	square := r.Mul(r, prec)
	spread := radius.Mul(radius)
	term := intBall(1)
	sum := intBall(1)
	bound := magOf(floatOne)
	target := magPow2(-int(prec))
	for n := int64(1); n < maxTerms; n++ {
		d := (2*n - 1) * (2 * n)
		term = divInt(term.Mul(square, prec), d, prec).Neg()
		sum = sum.Add(term, prec)
		bound = magQuoInt(bound.Mul(spread), d)
		if bound.Cmp(target) <= 0 {
			return widen(sum, bound)
		}
	}
	return widen(sum, infiniteMag())
}

func sinCosPoint(x *big.Float, prec uint) (num.Ball, num.Ball, error) {
	if x.Sign() == 0 {
		zero := exactBall(new(big.Float).SetPrec(prec))
		return zero, exactBall(new(big.Float).SetPrec(prec).SetInt64(1)), nil
	}
	e := expOf(x)
	if e > maxTrigArgExp {
		return num.Ball{}, num.Ball{}, ErrArgumentTooLarge
	}
	guard := 0
	if e > 0 {
		guard = e
	}
	wp := prec + 32 + uint(guard)
	quarter := halfPiBall(wp)
	quotient := new(big.Float).SetPrec(uint(guard)+64).Quo(x, quarter.Mid)
	n := nearestInt(quotient)
	r := exactBall(x)
	if n.Sign() != 0 {
		r = r.Sub(quarter.Mul(bigIntBall(n), wp), wp)
	}
	radius := magOf(absUpper(r))
	s := sinSeries(r, radius, wp)
	c := cosSeries(r, radius, wp)
	switch new(big.Int).Mod(n, bigFour).Int64() {
	case 1:
		return c, s.Neg(), nil
	case 2:
		return s.Neg(), c.Neg(), nil
	case 3:
		return c.Neg(), s, nil
	}
	return s, c, nil
}

func clampUnit(b num.Ball) num.Ball {
	lo, hi := b.Lo(), b.Hi()
	if lo.Cmp(floatNegOne) >= 0 && hi.Cmp(floatOne) <= 0 {
		return b
	}
	if lo.Cmp(floatNegOne) < 0 {
		lo = floatNegOne
	}
	if hi.Cmp(floatOne) > 0 {
		hi = floatOne
	}
	prec := b.Mid.Prec() + 8
	if prec < boundPrec {
		prec = boundPrec
	}
	return spanBall(lo, hi, prec)
}

func Sin(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	s, _, err := sinCosPoint(x.Mid, workPrec(prec))
	if err != nil {
		return num.Ball{}, err
	}
	return finish(clampUnit(widen(s, x.Rad)), prec), nil
}

func Cos(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	_, c, err := sinCosPoint(x.Mid, workPrec(prec))
	if err != nil {
		return num.Ball{}, err
	}
	return finish(clampUnit(widen(c, x.Rad)), prec), nil
}

func Tan(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	wp := workPrec(prec)
	s, c, err := sinCosPoint(x.Mid, wp)
	if err != nil {
		return num.Ball{}, err
	}
	q, err := clampUnit(widen(s, x.Rad)).Div(clampUnit(widen(c, x.Rad)), wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(q, prec), nil
}

func atanSeries(t num.Ball, prec uint) num.Ball {
	radius := magOf(absUpper(t))
	if radius.Cmp(magOf(floatOne)) >= 0 {
		return widen(t, infiniteMag())
	}
	square := t.Mul(t, prec)
	spread := radius.Mul(radius)
	term := t
	sum := t
	bound := radius
	target := magPow2(-int(prec))
	for n := int64(1); n < maxTerms; n++ {
		term = term.Mul(square, prec).Neg()
		sum = sum.Add(divInt(term, 2*n+1, prec), prec)
		bound = bound.Mul(spread)
		if magQuoInt(bound, 2*n+1).Cmp(target) <= 0 {
			return widen(sum, magQuoInt(bound.Mul(spread), 2*n+3))
		}
	}
	return widen(sum, infiniteMag())
}

func atanReduce(t num.Ball, prec uint) num.Ball {
	halvings := 0
	for halvings < 200 && absUpper(t).Cmp(halvingLimit) > 0 {
		root := sqrtPositiveBall(intBall(1).Add(t.Mul(t, prec), prec), prec)
		next, err := t.Div(intBall(1).Add(root, prec), prec)
		if err != nil {
			break
		}
		t = next
		halvings++
	}
	return scaleBall(atanSeries(t, prec), halvings)
}

func atanPoint(x *big.Float, prec uint) num.Ball {
	if x.Sign() == 0 {
		return exactBall(new(big.Float).SetPrec(prec))
	}
	wp := prec + 16
	magnitude := new(big.Float).Abs(x)
	inverted := magnitude.Cmp(floatOne) > 0
	t := exactBall(magnitude)
	if inverted {
		q, err := intBall(1).Div(t, wp)
		if err != nil {
			return widen(t, infiniteMag())
		}
		t = q
	}
	res := atanReduce(t, wp)
	if inverted {
		res = halfPiBall(wp).Sub(res, wp)
	}
	if x.Sign() < 0 {
		res = res.Neg()
	}
	return res
}

func atanBall(x num.Ball, prec uint) num.Ball {
	return widen(atanPoint(x.Mid, prec), x.Rad)
}

func Atan(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	return finish(atanBall(x, workPrec(prec)), prec), nil
}

func Atan2(y, x num.Ball, prec uint) (num.Ball, error) {
	y, x = normalize(y), normalize(x)
	wp := workPrec(prec)
	if s, ok := x.Sign(); ok && s > 0 {
		q, err := y.Div(x, wp)
		if err != nil {
			return num.Ball{}, err
		}
		return finish(atanBall(q, wp), prec), nil
	}
	if s, ok := y.Sign(); ok && s != 0 {
		q, err := x.Div(y, wp)
		if err != nil {
			return num.Ball{}, err
		}
		base := halfPiBall(wp)
		if s < 0 {
			base = base.Neg()
		}
		return finish(base.Sub(atanBall(q, wp), wp), prec), nil
	}
	return num.Ball{}, ErrOriginAtan2
}

func oneMinusSquare(a *big.Float, prec uint) num.Ball {
	r := ratOf(a)
	u := new(big.Rat).Sub(ratOne, new(big.Rat).Mul(r, r))
	f := new(big.Float).SetPrec(prec).SetRat(u)
	return num.NewBall(f, num.RoundingError(f, prec))
}

func asinBelowThreeQuarters(b num.Ball, prec uint) num.Ball {
	root := sqrtPositiveBall(intBall(1).Sub(b.Mul(b, prec), prec), prec)
	q, err := b.Div(root, prec)
	if err != nil {
		return widen(b, infiniteMag())
	}
	return atanReduce(q, prec)
}

func asinPoint(v *big.Float, prec uint) num.Ball {
	if v.Sign() == 0 {
		return exactBall(new(big.Float).SetPrec(prec))
	}
	wp := prec + 16
	magnitude := new(big.Float).Abs(v)
	var res num.Ball
	switch {
	case magnitude.Cmp(floatOne) >= 0:
		res = halfPiBall(wp)
	case magnitude.Cmp(threeQuarters) <= 0:
		res = asinBelowThreeQuarters(exactBall(magnitude), wp)
	default:
		root := sqrtPositiveBall(oneMinusSquare(magnitude, wp), wp)
		res = halfPiBall(wp).Sub(asinBelowThreeQuarters(root, wp), wp)
	}
	if v.Sign() < 0 {
		res = res.Neg()
	}
	return res
}

func acosPoint(v *big.Float, prec uint) num.Ball {
	wp := prec + 16
	magnitude := new(big.Float).Abs(v)
	if magnitude.Cmp(floatOne) >= 0 {
		if v.Sign() >= 0 {
			return exactBall(new(big.Float).SetPrec(prec))
		}
		return piBall(wp)
	}
	if magnitude.Cmp(threeQuarters) > 0 {
		root := sqrtPositiveBall(oneMinusSquare(magnitude, wp), wp)
		small := asinBelowThreeQuarters(root, wp)
		if v.Sign() > 0 {
			return small
		}
		return piBall(wp).Sub(small, wp)
	}
	return halfPiBall(wp).Sub(asinPoint(v, wp), wp)
}

func Asin(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	lo, hi := x.Lo(), x.Hi()
	if lo.Cmp(floatNegOne) < 0 || hi.Cmp(floatOne) > 0 {
		return num.Ball{}, ErrAsinDomain
	}
	wp := workPrec(prec)
	return finish(spanBall(asinPoint(lo, wp).Lo(), asinPoint(hi, wp).Hi(), wp), prec), nil
}

func Acos(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	lo, hi := x.Lo(), x.Hi()
	if lo.Cmp(floatNegOne) < 0 || hi.Cmp(floatOne) > 0 {
		return num.Ball{}, ErrAcosDomain
	}
	wp := workPrec(prec)
	return finish(spanBall(acosPoint(hi, wp).Lo(), acosPoint(lo, wp).Hi(), wp), prec), nil
}

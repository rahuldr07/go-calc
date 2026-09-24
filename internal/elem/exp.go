package elem

import (
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"
)

func expReduced(r num.Ball, prec uint) num.Ball {
	radius := magOf(absUpper(r))
	sum := intBall(1)
	term := intBall(1)
	bound := magOf(floatOne)
	target := magPow2(-int(prec))
	for n := int64(1); n < maxTerms; n++ {
		term = divInt(term.Mul(r, prec), n, prec)
		sum = sum.Add(term, prec)
		bound = magQuoInt(bound.Mul(radius), n)
		if n >= 3 && bound.Cmp(target) <= 0 {
			return widen(sum, magScale(bound, 1))
		}
	}
	return widen(sum, infiniteMag())
}

func expPoint(x *big.Float, prec uint) (num.Ball, error) {
	if x.Sign() == 0 {
		return exactBall(new(big.Float).SetPrec(prec).SetInt64(1)), nil
	}
	e := expOf(x)
	if e > maxExpArgExp {
		return num.Ball{}, ErrArgumentTooLarge
	}
	guard := 0
	if e > 0 {
		guard = e
	}
	wp := prec + 32 + uint(guard)
	ln2 := ln2Ball(wp)
	quotient := new(big.Float).SetPrec(uint(guard)+64).Quo(x, ln2.Mid)
	k := nearestInt(quotient)
	r := exactBall(x)
	if k.Sign() != 0 {
		r = r.Sub(ln2.Mul(bigIntBall(k), wp), wp)
	}
	res := expReduced(r, wp)
	if k.Sign() != 0 {
		res = scaleBall(res, int(k.Int64()))
	}
	return res, nil
}

func expBall(x num.Ball, prec uint) (num.Ball, error) {
	core, err := expPoint(x.Mid, prec)
	if err != nil {
		return num.Ball{}, err
	}
	if x.Rad.IsZero() {
		return core, nil
	}
	top, err := expPoint(x.Hi(), prec)
	if err != nil {
		return num.Ball{}, err
	}
	return widen(core, magOf(top.Hi()).Mul(x.Rad)), nil
}

func Exp(x num.Ball, prec uint) (num.Ball, error) {
	b, err := expBall(normalize(x), workPrec(prec))
	if err != nil {
		return num.Ball{}, err
	}
	return finish(b, prec), nil
}

func sinhCoshBalls(x num.Ball, prec uint) (num.Ball, num.Ball, error) {
	rising, err := expBall(x, prec)
	if err != nil {
		return num.Ball{}, num.Ball{}, err
	}
	falling, err := expBall(x.Neg(), prec)
	if err != nil {
		return num.Ball{}, num.Ball{}, err
	}
	return scaleBall(rising.Sub(falling, prec), -1), scaleBall(rising.Add(falling, prec), -1), nil
}

func Sinh(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	wp := workPrec(prec) + cancelGuard(x.Mid, 4096)
	s, _, err := sinhCoshBalls(x, wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(s, prec), nil
}

func Cosh(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	_, c, err := sinhCoshBalls(x, workPrec(prec))
	if err != nil {
		return num.Ball{}, err
	}
	return finish(c, prec), nil
}

func Tanh(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	wp := workPrec(prec) + cancelGuard(x.Mid, 4096)
	s, c, err := sinhCoshBalls(x, wp)
	if err != nil {
		return num.Ball{}, err
	}
	q, err := s.Div(c, wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(q, prec), nil
}

func integerExponent(b num.Ball) (int, bool) {
	if !b.Rad.IsZero() || !b.Mid.IsInt() {
		return 0, false
	}
	i, acc := b.Mid.Int(nil)
	if acc != big.Exact || !i.IsInt64() {
		return 0, false
	}
	v := i.Int64()
	if v < -(1<<20) || v > 1<<20 {
		return 0, false
	}
	return int(v), true
}

func Pow(base, exponent num.Ball, prec uint) (num.Ball, error) {
	base, exponent = normalize(base), normalize(exponent)
	if n, ok := integerExponent(exponent); ok {
		b, err := base.Pow(n, workPrec(prec))
		if err != nil {
			return num.Ball{}, err
		}
		return finish(b, prec), nil
	}
	if base.Lo().Sign() <= 0 {
		return num.Ball{}, ErrNonPositivePow
	}
	wp := workPrec(prec)
	logBase, err := logBall(base, wp)
	if err != nil {
		return num.Ball{}, err
	}
	product := exponent.Mul(logBase, wp)
	res, err := expBall(product, wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(res, prec), nil
}

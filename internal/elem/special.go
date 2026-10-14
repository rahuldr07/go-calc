package elem

import (
	"errors"
	"math"
	"math/big"
	"sync"

	"github.com/rahuldr07/go-calc/internal/num"
)

var (
	ErrPole            = errors.New("gamma has a pole at zero and at every negative whole number")
	ErrSeriesTooLong   = errors.New("the series did not settle within its term limit")
	gammaMinimumAt     = [2]float64{1.4616, 1.4617}
	gammaMinimumBelow  = big.NewFloat(0.8856)
	stirlingBaseOffset = 12.0
	log2E              = 1.4426950408889634
)

func Gamma(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	wp := workPrec(prec) + 16
	b, err := gammaBall(x, wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(b, prec), nil
}

func gammaBall(x num.Ball, wp uint) (num.Ball, error) {
	lo, hi := x.Lo(), x.Hi()
	if lo.Sign() > 0 {
		return gammaPositive(lo, hi, wp)
	}
	if hi.Sign() >= 0 {
		return num.Ball{}, ErrPole
	}
	nearest, _ := new(big.Float).Neg(lo).Int(nil)
	if new(big.Float).SetInt(nearest.Neg(nearest)).Cmp(hi) <= 0 {
		return num.Ball{}, ErrPole
	}
	reflected := intBall(1).Sub(x, wp)
	g, err := gammaPositive(reflected.Lo(), reflected.Hi(), wp)
	if err != nil {
		return num.Ball{}, err
	}
	pi := Pi(wp)
	s, err := Sin(pi.Mul(x, wp), wp)
	if err != nil {
		return num.Ball{}, err
	}
	out, err := pi.Div(s.Mul(g, wp), wp)
	if err != nil {
		return num.Ball{}, ErrPole
	}
	return out, nil
}

func gammaPositive(lo, hi *big.Float, wp uint) (num.Ball, error) {
	a, err := gammaPoint(lo, wp)
	if err != nil || lo.Cmp(hi) == 0 {
		return a, err
	}
	b, err := gammaPoint(hi, wp)
	if err != nil {
		return num.Ball{}, err
	}
	low := minFloat(a.Lo(), b.Lo())
	high := maxFloat(a.Hi(), b.Hi())
	lf, _ := lo.Float64()
	hf, _ := hi.Float64()
	if lf < gammaMinimumAt[1] && hf > gammaMinimumAt[0] {
		low = minFloat(low, gammaMinimumBelow)
	}
	return spanBall(low, high, wp), nil
}

func minFloat(a, b *big.Float) *big.Float {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}

func maxFloat(a, b *big.Float) *big.Float {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func gammaPoint(x *big.Float, wp uint) (num.Ball, error) {
	base := float64(wp)/4 + stirlingBaseOffset
	xf, _ := x.Float64()
	shift := 0
	if xf < base {
		shift = int(math.Ceil(base - xf))
	}
	point := exactBall(x)
	shifted := point.Add(intBall(int64(shift)), wp)
	lnGamma, err := stirling(shifted, wp)
	if err != nil {
		return num.Ball{}, err
	}
	g, err := Exp(lnGamma, wp)
	if err != nil {
		return num.Ball{}, err
	}
	if shift == 0 {
		return g, nil
	}
	rising := point
	for i := 1; i < shift; i++ {
		rising = rising.Mul(point.Add(intBall(int64(i)), wp), wp)
	}
	return g.Div(rising, wp)
}

func stirling(x num.Ball, wp uint) (num.Ball, error) {
	lnX, err := Log(x, wp)
	if err != nil {
		return num.Ball{}, err
	}
	half := new(big.Float).SetFloat64(0.5)
	sum := x.Sub(exactBall(half), wp).Mul(lnX, wp).Sub(x, wp)
	twoPi := scaleBall(Pi(wp), 1)
	lnTwoPi, err := Log(twoPi, wp)
	if err != nil {
		return num.Ball{}, err
	}
	sum = sum.Add(scaleBall(lnTwoPi, -1), wp)
	target := magPow2(-int(wp) - 8)
	xLow := x.Lo()
	square := x.Mul(x, wp)
	power := x
	lowPower := down().Set(xLow)
	lowSquare := down().Mul(xLow, xLow)
	limit := int(4*float64(wp)) + 64
	for k := 1; k <= limit; k++ {
		coefficient := new(big.Rat).Quo(bernoulli(2*k), big.NewRat(int64(2*k)*int64(2*k-1), 1))
		term, err := num.BallFromExact(num.FromRat(coefficient), wp).Div(power, wp)
		if err != nil {
			return num.Ball{}, err
		}
		sum = sum.Add(term, wp)
		power = power.Mul(square, wp)
		lowPower = down().Mul(lowPower, lowSquare)
		next := new(big.Rat).Quo(bernoulli(2*k+2), big.NewRat(int64(2*k+2)*int64(2*k+1), 1))
		bound := magQuoFloat(magFromRat(next), lowPower)
		if bound.Cmp(target) <= 0 {
			return widen(sum, bound), nil
		}
	}
	return num.Ball{}, ErrSeriesTooLong
}

var bernoulliCache = struct {
	sync.Mutex
	numbers []*big.Rat
}{numbers: []*big.Rat{big.NewRat(1, 1)}}

func bernoulli(n int) *big.Rat {
	bernoulliCache.Lock()
	defer bernoulliCache.Unlock()
	for m := len(bernoulliCache.numbers); m <= n; m++ {
		sum := new(big.Rat)
		choose := big.NewInt(1)
		for k := 0; k < m; k++ {
			if k > 0 {
				choose.Mul(choose, big.NewInt(int64(m+2-k)))
				choose.Quo(choose, big.NewInt(int64(k)))
			}
			sum.Add(sum, new(big.Rat).Mul(new(big.Rat).SetInt(choose), bernoulliCache.numbers[k]))
		}
		bernoulliCache.numbers = append(bernoulliCache.numbers, sum.Quo(sum.Neg(sum), big.NewRat(int64(m+1), 1)))
	}
	return bernoulliCache.numbers[n]
}

func Erf(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	wp := workPrec(prec) + 16
	if x.Rad.IsZero() {
		b, err := erfPoint(x.Mid, wp)
		if err != nil {
			return num.Ball{}, err
		}
		return finish(b, prec), nil
	}
	low, err := erfPoint(x.Lo(), wp)
	if err != nil {
		return num.Ball{}, err
	}
	high, err := erfPoint(x.Hi(), wp)
	if err != nil {
		return num.Ball{}, err
	}
	return finish(spanBall(low.Lo(), high.Hi(), wp), prec), nil
}

func erfPoint(x *big.Float, wp uint) (num.Ball, error) {
	if x.Sign() == 0 {
		return intBall(0), nil
	}
	if x.Sign() < 0 {
		b, err := erfPoint(new(big.Float).Neg(x), wp)
		return b.Neg(), err
	}
	xf, _ := x.Float64()
	squared := xf * xf
	if xf >= 1 && squared*log2E > float64(wp)+8 {
		return widen(intBall(1), magPow2(-int(math.Floor(squared*log2E))+1)), nil
	}
	p := wp + uint(math.Ceil(squared*log2E)) + 16
	point := exactBall(x)
	x2 := point.Mul(point, p)
	power := point
	sum := point
	target := magPow2(-int(p))
	for n := int64(1); n < maxTerms; n++ {
		power = divInt(power.Mul(x2, p).Neg(), n, p)
		term := divInt(power, 2*n+1, p)
		sum = sum.Add(term, p)
		if float64(n) > squared+1 && magOf(absUpper(term)).Cmp(target) <= 0 {
			sum = widen(sum, magOf(absUpper(term)))
			root, err := Sqrt(Pi(p), p)
			if err != nil {
				return num.Ball{}, err
			}
			scale, err := intBall(2).Div(root, p)
			if err != nil {
				return num.Ball{}, err
			}
			return sum.Mul(scale, p), nil
		}
	}
	return num.Ball{}, ErrSeriesTooLong
}

package elem

import (
	"errors"
	"math"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"
)

var (
	ErrNegativeSqrt     = errors.New("square root of a value that may be negative")
	ErrNonPositiveLog   = errors.New("log of a value that may not be positive")
	ErrAsinDomain       = errors.New("asin of a value that may be outside [-1, 1]")
	ErrAcosDomain       = errors.New("acos of a value that may be outside [-1, 1]")
	ErrArgumentTooLarge = errors.New("argument magnitude too large to reduce")
	ErrOriginAtan2      = errors.New("atan2 of a point that may be the origin")
	ErrAtan2BranchCut   = errors.New("atan2 of a point that may straddle the negative real axis")
	ErrNonPositivePow   = errors.New("power of a value that may not be positive")
)

const (
	guardBits     = 64
	boundPrec     = 64
	maxExpArgExp  = 28
	maxTrigArgExp = 1 << 12
	maxTerms      = 1 << 22
)

var (
	floatOne    = big.NewFloat(1)
	floatHalf   = big.NewFloat(0.5)
	floatNegOne = big.NewFloat(-1)
	ratOne      = new(big.Rat).SetInt64(1)
	bigFour     = big.NewInt(4)
)

func workPrec(prec uint) uint { return prec + guardBits }

func up() *big.Float {
	return new(big.Float).SetPrec(boundPrec).SetMode(big.ToPositiveInf)
}

func down() *big.Float {
	return new(big.Float).SetPrec(boundPrec).SetMode(big.ToNegativeInf)
}

func magOf(x *big.Float) num.Mag { return num.MagFromBigFloat(x) }

func infiniteMag() num.Mag { return magOf(up().SetInf(false)) }

func unitBall(prec uint) num.Ball {
	return num.NewBall(new(big.Float).SetPrec(prec), magOf(floatOne))
}

func normalize(b num.Ball) num.Ball {
	if b.Mid == nil {
		return num.Ball{Mid: new(big.Float), Rad: b.Rad}
	}
	return b
}

func magFromRat(r *big.Rat) num.Mag {
	return magOf(up().SetRat(new(big.Rat).Abs(r)))
}

func magPow2(exp int) num.Mag {
	return magOf(new(big.Float).SetMantExp(floatOne, exp))
}

func magScale(m num.Mag, exp int) num.Mag {
	return magOf(new(big.Float).SetMantExp(m.Float(), exp))
}

func magQuoInt(m num.Mag, n int64) num.Mag {
	return magOf(up().Quo(m.Float(), new(big.Float).SetInt64(n)))
}

func magQuoFloat(m num.Mag, denLo *big.Float) num.Mag {
	return magOf(up().Quo(m.Float(), denLo))
}

func exactBall(x *big.Float) num.Ball { return num.NewBall(x, num.Mag{}) }

func intBall(n int64) num.Ball { return exactBall(new(big.Float).SetInt64(n)) }

func bigIntBall(n *big.Int) num.Ball { return exactBall(new(big.Float).SetInt(n)) }

func widen(b num.Ball, m num.Mag) num.Ball { return num.NewBall(b.Mid, b.Rad.Add(m)) }

func divInt(b num.Ball, n int64, prec uint) num.Ball {
	q, err := b.Div(intBall(n), prec)
	if err != nil {
		return widen(b, infiniteMag())
	}
	return q
}

func scaleBall(b num.Ball, exp int) num.Ball {
	return num.Ball{
		Mid: new(big.Float).SetMantExp(b.Mid, exp),
		Rad: magScale(b.Rad, exp),
	}
}

func absUpper(b num.Ball) *big.Float {
	return up().Add(new(big.Float).Abs(b.Mid), b.Rad.Float())
}

func finish(b num.Ball, prec uint) num.Ball {
	mid := new(big.Float).SetPrec(prec).Set(b.Mid)
	return num.NewBall(mid, b.Rad.Add(num.RoundingError(mid, prec)))
}

func expOf(x *big.Float) int {
	if x.Sign() == 0 {
		return 0
	}
	return x.MantExp(nil)
}

func cancelGuard(x *big.Float, limit uint) uint {
	e := expOf(x)
	if e >= 0 {
		return 0
	}
	if uint(-e) > limit {
		return limit
	}
	return uint(-e)
}

func nearestInt(q *big.Float) *big.Int {
	adj := new(big.Float).SetPrec(q.Prec() + 8)
	if q.Sign() >= 0 {
		adj.Add(q, floatHalf)
	} else {
		adj.Sub(q, floatHalf)
	}
	i, _ := adj.Int(nil)
	return i
}

func ratOf(x *big.Float) *big.Rat {
	r, _ := x.Rat(nil)
	if r == nil {
		return new(big.Rat)
	}
	return r
}

func spanBall(lo, hi *big.Float, prec uint) num.Ball {
	sum := new(big.Float).SetPrec(prec).Add(lo, hi)
	half := num.RoundingError(sum, prec)
	mid := new(big.Float).SetMantExp(sum, -1)
	width := up().Sub(hi, lo)
	width.SetMantExp(width, -1)
	return num.NewBall(mid, magOf(width).Add(magScale(half, -1)))
}

func sqrtPoint(x *big.Float, prec uint) num.Ball {
	if x.Sign() <= 0 {
		return exactBall(new(big.Float).SetPrec(prec))
	}
	mant := new(big.Float)
	e := x.MantExp(mant)
	if e&1 != 0 {
		mant.SetMantExp(mant, 1)
		e--
	}
	wp := prec + 16
	seed, _ := mant.Float64()
	y := new(big.Float).SetPrec(wp).SetFloat64(math.Sqrt(seed))
	for p := uint(50); p < wp; p *= 2 {
		t := new(big.Float).SetPrec(wp).Quo(mant, y)
		y = new(big.Float).SetPrec(wp).Add(y, t)
		y.SetMantExp(y, -1)
	}
	yr := ratOf(y)
	d := new(big.Rat).Sub(new(big.Rat).Mul(yr, yr), ratOf(mant))
	rad := magQuoFloat(magFromRat(d), down().Set(y))
	return scaleBall(num.NewBall(y, rad), e/2)
}

func sqrtPositiveBall(b num.Ball, prec uint) num.Ball {
	lo := b.Lo()
	if lo.Sign() < 0 {
		lo = new(big.Float)
	}
	return spanBall(sqrtPoint(lo, prec).Lo(), sqrtPoint(b.Hi(), prec).Hi(), prec)
}

func Sqrt(x num.Ball, prec uint) (num.Ball, error) {
	x = normalize(x)
	if x.Lo().Sign() < 0 {
		return num.Ball{}, ErrNegativeSqrt
	}
	return finish(sqrtPositiveBall(x, workPrec(prec)), prec), nil
}

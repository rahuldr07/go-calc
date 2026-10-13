package eval

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	maxRoundPlaces    = 1000
	maxCombinatoricsN = 100000
)

var (
	errBoundary     = errors.New("the value is too close to a rounding boundary to decide at this precision")
	errNotWholePair = errors.New("takes two whole numbers, n and k, with 0 ≤ k ≤ n")
)

func wholeInt(v value.Value) (*big.Int, bool) {
	if !plainNumber(v) || !isWhole(v) {
		return nil, false
	}
	return v.Num.Exact().Rat().Num(), true
}

func smallWhole(v value.Value, lo, hi int64) (int64, bool) {
	n, ok := wholeInt(v)
	if !ok || !n.IsInt64() || n.Int64() < lo || n.Int64() > hi {
		return 0, false
	}
	return n.Int64(), true
}

func scaleOf(places int64) *big.Rat {
	ten := new(big.Int).Exp(big.NewInt(10), big.NewInt(absInt(places)), nil)
	if places >= 0 {
		return new(big.Rat).SetInt(ten)
	}
	return new(big.Rat).SetFrac(big.NewInt(1), ten)
}

func absInt(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func roundExactTo(x num.Exact, mode string, places int64) num.Exact {
	if places == 0 {
		return roundTo(x, mode)
	}
	scale := scaleOf(places)
	shifted := num.FromRat(new(big.Rat).Mul(x.Rat(), scale))
	whole := roundTo(shifted, mode).Rat()
	return num.FromRat(new(big.Rat).Quo(whole, scale))
}

func roundNumber(n num.Number, mode string, places int64, prec uint) (num.Exact, error) {
	if n.IsExact() {
		return roundExactTo(n.Exact(), mode, places), nil
	}
	b := n.Ball(prec)
	lo, _ := b.Lo().Rat(nil)
	hi, _ := b.Hi().Rat(nil)
	low := roundExactTo(num.FromRat(lo), mode, places)
	high := roundExactTo(num.FromRat(hi), mode, places)
	if low.Cmp(high) != 0 {
		return num.Exact{}, errBoundary
	}
	return low, nil
}

func logBase(x, base value.Value, prec uint) (value.Value, error) {
	if x.IsMeasured() || base.IsMeasured() {
		return value.Value{}, errors.New("log with a base cannot propagate a measured uncertainty")
	}
	if !x.IsScalar() || !base.IsScalar() {
		return value.Value{}, errors.New("log needs plain numbers")
	}
	if base.Num.IsExact() {
		switch {
		case base.Num.Exact().Sign() <= 0:
			return value.Value{}, errors.New("the base of a log must be positive")
		case base.Num.Exact().Cmp(num.FromInt(1)) == 0:
			return value.Value{}, errors.New("the base of a log cannot be 1")
		}
	}
	if x.Num.IsExact() && x.Num.Exact().Sign() <= 0 {
		return value.Value{}, errors.New("log of a number that is not positive")
	}
	if k, ok := exactLog(x.Num, base.Num); ok {
		return value.Scalar(num.FromRat(new(big.Rat).SetInt64(k))), nil
	}
	top, err := elem.Log(x.Num.Ball(prec), prec)
	if err != nil {
		return value.Value{}, err
	}
	bottom, err := elem.Log(base.Num.Ball(prec), prec)
	if err != nil {
		return value.Value{}, err
	}
	q, err := top.Div(bottom, prec)
	if err != nil {
		return value.Value{}, errors.New("the base of a log may be 1 at this precision")
	}
	return value.ScalarNum(num.NumBall(q)), nil
}

func exactLog(x, base num.Number) (int64, bool) {
	if !x.IsExact() || !base.IsExact() {
		return 0, false
	}
	target, b := x.Exact().Rat(), base.Exact().Rat()
	if target.Cmp(big.NewRat(1, 1)) == 0 {
		return 0, true
	}
	bits := int64(target.Num().BitLen() + target.Denom().BitLen())
	for _, step := range []int64{1, -1} {
		power := big.NewRat(1, 1)
		for k := step; absInt(k) <= bits; k += step {
			if step > 0 {
				power.Mul(power, b)
			} else {
				power.Quo(power, b)
			}
			if power.Cmp(target) == 0 {
				return k, true
			}
			if power.Num().BitLen()+power.Denom().BitLen() > int(2*bits)+8 {
				break
			}
		}
	}
	return 0, false
}

func combinatoric(fn string, nv, kv value.Value) (value.Value, error) {
	if nv.IsMeasured() || kv.IsMeasured() {
		return value.Value{}, errors.New(fn + " cannot propagate a measured uncertainty")
	}
	n, okN := smallWhole(nv, 0, maxCombinatoricsN)
	k, okK := smallWhole(kv, 0, maxCombinatoricsN)
	if !okN || !okK || k > n {
		return value.Value{}, errors.New(fn + " " + errNotWholePair.Error() + " ≤ " + strconv.Itoa(maxCombinatoricsN))
	}
	if fn == "nCr" {
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(new(big.Int).Binomial(n, k)))), nil
	}
	out := new(big.Int).MulRange(n-k+1, n)
	if k == 0 {
		out.SetInt64(1)
	}
	return value.Scalar(num.FromRat(new(big.Rat).SetInt(out))), nil
}

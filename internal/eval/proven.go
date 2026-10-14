package eval

import (
	"errors"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	maxExactExponent  = 100000
	maxExactFactorial = 100000
)

func init() {
	one := func(name string, run func(c call) (value.Value, error)) {
		register(name, builtinFunc{min: 1, max: 1, usage: name + " takes one value", run: run})
	}
	one("cbrt", cbrtCall)
	one("exp2", exp2Call)
	one("sec", reciprocalTrig(elem.Cos, "cos"))
	one("csc", reciprocalTrig(elem.Sin, "sin"))
	one("cot", reciprocalTrig(elem.Tan, "tan"))
	one("asinh", asinhCall)
	one("acosh", acoshCall)
	one("atanh", atanhCall)
	one("deg", angleScale(true))
	one("rad", angleScale(false))
	one("gamma", gammaCall)
	one("erf", erfCall(false))
	one("erfc", erfCall(true))
	register("hypot", builtinFunc{min: 2, max: 2, usage: "hypot takes two values", run: hypotCall})
}

func proven(b num.Ball) value.Value { return value.ScalarNum(num.NumBall(b)) }

func plainBall(c call, i int) (num.Number, num.Ball, error) {
	n, err := c.plain(i)
	if err != nil {
		return num.Number{}, num.Ball{}, err
	}
	return n, n.Ball(c.prec), nil
}

func isZero(n num.Number) bool { return n.IsExact() && n.Exact().Sign() == 0 }

func cbrtCall(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if n.IsExact() {
		x := n.Exact()
		negative := x.Sign() < 0
		if negative {
			x = x.Neg()
		}
		if r, ok := num.Root(x, 3); ok {
			if negative {
				r = r.Neg()
			}
			return value.Scalar(r), nil
		}
	}
	sign, decided := b.Sign()
	if !decided {
		return value.Value{}, errors.New("cbrt: the value may be zero or change sign at this precision")
	}
	if sign == 0 {
		return value.Scalar(num.FromInt(0)), nil
	}
	out, err := elem.Pow(b.Abs(), reciprocalBall(3, c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("cbrt: " + err.Error())
	}
	if sign < 0 {
		out = out.Neg()
	}
	return proven(out), nil
}

func hypotCall(c call) (value.Value, error) {
	a, b := c.args[0], c.args[1]
	if a.Dim != b.Dim {
		return value.Value{}, errors.New("hypot needs two values of the same kind of quantity")
	}
	if a.IsAffine() || b.IsAffine() {
		return value.Value{}, errors.New("hypot is not defined for absolute temperatures")
	}
	shell := value.Value{Dim: a.Dim, Factor: num.FromInt(1)}
	x, y := a.SI(c.prec), b.SI(c.prec)
	if a.Unit == b.Unit && a.FactorOrOne().Cmp(b.FactorOrOne()) == 0 {
		shell, x, y = a, a.Num, b.Num
	} else if !a.Dim.IsNone() {
		shell.Unit = units.SISymbol(a.Dim)
	}
	square := x.Mul(x, c.prec).Add(y.Mul(y, c.prec), c.prec)
	out := shell
	if square.IsExact() {
		if r, ok := num.Root(square.Exact(), 2); ok {
			out.Num = num.NumExact(r)
			return out, nil
		}
	}
	root, err := elem.Sqrt(square.Ball(c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("hypot: " + err.Error())
	}
	out.Num = num.NumBall(root)
	return out, nil
}

func exp2Call(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if k, ok := smallWhole(c.args[0], -maxExactExponent, maxExactExponent); ok && n.IsExact() {
		power := new(big.Int).Lsh(big.NewInt(1), uint(absInt(k)))
		r := new(big.Rat).SetInt(power)
		if k < 0 {
			r.Inv(r)
		}
		return value.Scalar(num.FromRat(r)), nil
	}
	out, err := elem.Pow(num.BallFromExact(num.FromInt(2), c.prec), b, c.prec)
	if err != nil {
		return value.Value{}, errors.New("exp2: " + err.Error())
	}
	return proven(out), nil
}

func angleIn(c call, b num.Ball) (num.Ball, error) {
	if !c.degrees {
		return b, nil
	}
	return degreesToRadians(b, c.prec)
}

func reciprocalTrig(f func(num.Ball, uint) (num.Ball, error), inner string) func(c call) (value.Value, error) {
	return func(c call) (value.Value, error) {
		_, b, err := plainBall(c, 0)
		if err != nil {
			return value.Value{}, err
		}
		angle, err := angleIn(c, b)
		if err != nil {
			return value.Value{}, err
		}
		v, err := f(angle, c.prec)
		if err != nil {
			return value.Value{}, errors.New(c.name + ": " + err.Error())
		}
		out, err := num.BallFromExact(num.FromInt(1), c.prec).Div(v, c.prec)
		if err != nil {
			return value.Value{}, errors.New(c.name + " is undefined where " + inner + " is zero, and this angle is at or too close to one")
		}
		return proven(out), nil
	}
}

func asinhCall(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if isZero(n) {
		return value.Scalar(num.FromInt(0)), nil
	}
	sign, decided := b.Sign()
	if !decided {
		return value.Value{}, errors.New("asinh: the value may be zero at this precision")
	}
	x := b.Abs()
	root, err := elem.Sqrt(x.Mul(x, c.prec).Add(unitBall(c.prec), c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("asinh: " + err.Error())
	}
	out, err := elem.Log(x.Add(root, c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("asinh: " + err.Error())
	}
	if sign < 0 {
		out = out.Neg()
	}
	return proven(out), nil
}

func unitBall(prec uint) num.Ball { return num.BallFromExact(num.FromInt(1), prec) }

func acoshCall(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if n.IsExact() && n.Exact().Cmp(num.FromInt(1)) == 0 {
		return value.Scalar(num.FromInt(0)), nil
	}
	if s, decided := b.Sub(unitBall(c.prec), c.prec).Sign(); !decided || s < 0 {
		return value.Value{}, errors.New("acosh is defined for values of 1 and above")
	}
	root, err := elem.Sqrt(b.Mul(b, c.prec).Sub(unitBall(c.prec), c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("acosh: " + err.Error())
	}
	out, err := elem.Log(b.Add(root, c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("acosh: " + err.Error())
	}
	return proven(out), nil
}

func atanhCall(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if isZero(n) {
		return value.Scalar(num.FromInt(0)), nil
	}
	ratio, err := unitBall(c.prec).Add(b, c.prec).Div(unitBall(c.prec).Sub(b, c.prec), c.prec)
	if err != nil {
		return value.Value{}, errors.New("atanh is defined strictly between -1 and 1")
	}
	if s, decided := ratio.Sign(); !decided || s <= 0 {
		return value.Value{}, errors.New("atanh is defined strictly between -1 and 1")
	}
	out, err := elem.Log(ratio, c.prec)
	if err != nil {
		return value.Value{}, errors.New("atanh: " + err.Error())
	}
	return proven(out.Mul(num.BallFromExact(mustExact("1/2"), c.prec), c.prec)), nil
}

func mustExact(s string) num.Exact {
	e, err := num.ParseExact(s)
	if err != nil {
		panic(err)
	}
	return e
}

func angleScale(toDegrees bool) func(c call) (value.Value, error) {
	return func(c call) (value.Value, error) {
		n, b, err := plainBall(c, 0)
		if err != nil {
			return value.Value{}, err
		}
		if isZero(n) {
			return value.Scalar(num.FromInt(0)), nil
		}
		convert := degreesToRadians
		if toDegrees {
			convert = radiansToDegrees
		}
		out, err := convert(b, c.prec)
		if err != nil {
			return value.Value{}, errors.New(c.name + ": " + err.Error())
		}
		return proven(out), nil
	}
}

func gammaCall(c call) (value.Value, error) {
	n, b, err := plainBall(c, 0)
	if err != nil {
		return value.Value{}, err
	}
	if n.IsExact() {
		if v, ok, err := exactGamma(n.Exact(), c.prec); ok || err != nil {
			return v, err
		}
	}
	out, err := elem.Gamma(b, c.prec)
	if err != nil {
		return value.Value{}, errors.New("gamma: " + err.Error())
	}
	return proven(out), nil
}

func exactGamma(x num.Exact, prec uint) (value.Value, bool, error) {
	r := x.Rat()
	if r.IsInt() {
		k := r.Num()
		if k.Sign() <= 0 {
			return value.Value{}, true, errors.New("gamma: " + elem.ErrPole.Error())
		}
		if !k.IsInt64() || k.Int64() > maxExactFactorial {
			return value.Value{}, false, nil
		}
		return value.Scalar(num.FromRat(new(big.Rat).SetInt(new(big.Int).MulRange(1, k.Int64()-1)))), true, nil
	}
	twice := new(big.Rat).Mul(r, big.NewRat(2, 1))
	if !twice.IsInt() || !twice.Num().IsInt64() {
		return value.Value{}, false, nil
	}
	m := twice.Num().Int64()
	if m > 2*maxExactFactorial || m < -2*maxExactFactorial {
		return value.Value{}, false, nil
	}
	coefficient := halfIntegerGamma(m)
	root, err := elem.Sqrt(elem.Pi(prec), prec)
	if err != nil {
		return value.Value{}, true, err
	}
	return proven(num.BallFromExact(num.FromRat(coefficient), prec).Mul(root, prec)), true, nil
}

func halfIntegerGamma(twice int64) *big.Rat {
	if twice > 0 {
		n := (twice - 1) / 2
		top := factorial(2 * n)
		bottom := new(big.Int).Mul(new(big.Int).Lsh(big.NewInt(1), uint(2*n)), factorial(n))
		return new(big.Rat).SetFrac(top, bottom)
	}
	n := (1 - twice) / 2
	top := new(big.Int).Mul(new(big.Int).Lsh(big.NewInt(1), uint(2*n)), factorial(n))
	if n%2 == 1 {
		top.Neg(top)
	}
	return new(big.Rat).SetFrac(top, factorial(2*n))
}

func factorial(n int64) *big.Int {
	if n < 2 {
		return big.NewInt(1)
	}
	return new(big.Int).MulRange(2, n)
}

func erfCall(complement bool) func(c call) (value.Value, error) {
	return func(c call) (value.Value, error) {
		n, b, err := plainBall(c, 0)
		if err != nil {
			return value.Value{}, err
		}
		if isZero(n) {
			return value.Scalar(num.FromInt(int64(boolInt(complement)))), nil
		}
		out, err := elem.Erf(b, c.prec)
		if err != nil {
			return value.Value{}, errors.New(c.name + ": " + err.Error())
		}
		if complement {
			out = unitBall(c.prec).Sub(out, c.prec)
		}
		return proven(out), nil
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

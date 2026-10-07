package cplx

import (
	"errors"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/num"
)

var (
	ErrLogOfZero        = errors.New("log of a value that may be zero")
	ErrArgOfZero        = errors.New("argument of a value that may be zero")
	ErrBranchCut        = errors.New("value may straddle the branch cut")
	ErrExponentTooLarge = errors.New("integer exponent too large")
	ErrNotNonNegative   = errors.New("square root of a bound that is negative")
	ErrUnbounded        = errors.New("value has no finite bound")
)

const (
	guardBits        = 64
	maxPowerExponent = 1 << 20
)

var (
	zero = num.NumExact(num.FromInt(0))
	one  = num.NumExact(num.FromInt(1))
	half = num.NumExact(num.FromRat(big.NewRat(1, 2)))
)

type Complex struct {
	re num.Number
	im num.Number
}

func Real(n num.Number) Complex { return Complex{re: n, im: zero} }

func Rect(re, im num.Number) Complex { return Complex{re: re, im: im} }

func (c Complex) Re() num.Number { return c.re }

func (c Complex) Im() num.Number { return c.im }

func (c Complex) IsReal() bool { return isExactZero(c.im) }

func (c Complex) IsExact() bool { return c.re.IsExact() && c.im.IsExact() }

func isExactZero(n num.Number) bool { return n.IsExact() && n.Exact().Sign() == 0 }

func isExactOne(n num.Number) bool { return n.IsExact() && n.Exact().Cmp(num.FromInt(1)) == 0 }

func mayBeZero(n num.Number, prec uint) bool {
	if n.IsExact() {
		return n.Exact().Sign() == 0
	}
	return n.Ball(prec).ContainsZero()
}

func entire(prec uint) num.Number {
	return num.NumBall(num.Ball{
		Mid: new(big.Float).SetPrec(prec),
		Rad: num.MagFromBigFloat(new(big.Float).SetInf(false)),
	})
}

func isUnbounded(n num.Number, prec uint) bool {
	if n.IsExact() {
		return false
	}
	b := n.Ball(prec)
	return b.Rad.Float().IsInf() || (b.Mid != nil && b.Mid.IsInf())
}

func anyUnbounded(prec uint, parts ...num.Number) bool {
	for _, p := range parts {
		if isUnbounded(p, prec) {
			return true
		}
	}
	return false
}

func (c Complex) isUnbounded(prec uint) bool { return anyUnbounded(prec, c.re, c.im) }

func addNum(a, b num.Number, prec uint) num.Number {
	if anyUnbounded(prec, a, b) {
		return entire(prec)
	}
	switch {
	case isExactZero(a):
		return b
	case isExactZero(b):
		return a
	}
	return a.Add(b, prec)
}

func subNum(a, b num.Number, prec uint) num.Number {
	if anyUnbounded(prec, a, b) {
		return entire(prec)
	}
	switch {
	case isExactZero(b):
		return a
	case isExactZero(a):
		return b.Neg()
	}
	return a.Sub(b, prec)
}

func mulNum(a, b num.Number, prec uint) num.Number {
	if anyUnbounded(prec, a, b) {
		return entire(prec)
	}
	switch {
	case isExactZero(a) || isExactZero(b):
		return zero
	case isExactOne(a):
		return b
	case isExactOne(b):
		return a
	}
	return a.Mul(b, prec)
}

func (c Complex) Add(o Complex, prec uint) Complex {
	return Complex{re: addNum(c.re, o.re, prec), im: addNum(c.im, o.im, prec)}
}

func (c Complex) Sub(o Complex, prec uint) Complex {
	return Complex{re: subNum(c.re, o.re, prec), im: subNum(c.im, o.im, prec)}
}

func (c Complex) Mul(o Complex, prec uint) Complex {
	return Complex{
		re: subNum(mulNum(c.re, o.re, prec), mulNum(c.im, o.im, prec), prec),
		im: addNum(mulNum(c.re, o.im, prec), mulNum(c.im, o.re, prec), prec),
	}
}

func (c Complex) Neg() Complex { return Complex{re: c.re.Neg(), im: c.im.Neg()} }

func (c Complex) Conj() Complex { return Complex{re: c.re, im: c.im.Neg()} }

func (c Complex) Div(o Complex, prec uint) (Complex, error) {
	if c.isUnbounded(prec) || o.isUnbounded(prec) {
		return Complex{}, ErrUnbounded
	}
	if mayBeZero(o.re, prec) && mayBeZero(o.im, prec) {
		if isExactZero(o.re) && isExactZero(o.im) {
			return Complex{}, num.ErrDivByZero
		}
		return Complex{}, num.ErrDivisorMayBeZero
	}
	den := squareSum(o, prec)
	numer := c.Mul(o.Conj(), prec)
	re, err := divNum(numer.re, den, prec)
	if err != nil {
		return Complex{}, err
	}
	im, err := divNum(numer.im, den, prec)
	if err != nil {
		return Complex{}, err
	}
	return Complex{re: re, im: im}, nil
}

func divNum(a, b num.Number, prec uint) (num.Number, error) {
	if isExactZero(a) && !mayBeZero(b, prec) {
		return zero, nil
	}
	return a.Div(b, prec)
}

func workPrec(prec uint) uint { return prec + guardBits }

func roundedMul(x, y *big.Float, prec uint, mode big.RoundingMode) *big.Float {
	return new(big.Float).SetPrec(prec).SetMode(mode).Mul(x, y)
}

func roundedAdd(x, y *big.Float, prec uint, mode big.RoundingMode) *big.Float {
	return new(big.Float).SetPrec(prec).SetMode(mode).Add(x, y)
}

func squareBounds(n num.Number, prec uint) (*big.Float, *big.Float) {
	b := n.Ball(prec)
	lo, hi := b.Lo(), b.Hi()
	wp := workPrec(prec)
	loSquareDown := roundedMul(lo, lo, wp, big.ToNegativeInf)
	loSquareUp := roundedMul(lo, lo, wp, big.ToPositiveInf)
	hiSquareDown := roundedMul(hi, hi, wp, big.ToNegativeInf)
	hiSquareUp := roundedMul(hi, hi, wp, big.ToPositiveInf)
	switch {
	case lo.Sign() >= 0:
		return loSquareDown, hiSquareUp
	case hi.Sign() <= 0:
		return hiSquareDown, loSquareUp
	}
	if loSquareUp.Cmp(hiSquareUp) > 0 {
		return new(big.Float), loSquareUp
	}
	return new(big.Float), hiSquareUp
}

func squareSumBounds(c Complex, prec uint) (*big.Float, *big.Float) {
	reLo, reHi := squareBounds(c.re, prec)
	imLo, imHi := squareBounds(c.im, prec)
	wp := workPrec(prec)
	return roundedAdd(reLo, imLo, wp, big.ToNegativeInf), roundedAdd(reHi, imHi, wp, big.ToPositiveInf)
}

func spanBall(lo, hi *big.Float, prec uint) num.Ball {
	if lo.IsInf() || hi.IsInf() {
		return num.Ball{Mid: new(big.Float).SetPrec(prec), Rad: num.MagFromBigFloat(new(big.Float).SetInf(false))}
	}
	wp := workPrec(prec)
	mid := new(big.Float).SetPrec(prec).Add(lo, hi)
	mid.SetMantExp(mid, -1)
	above := new(big.Float).SetPrec(wp).SetMode(big.ToPositiveInf).Sub(hi, mid)
	below := new(big.Float).SetPrec(wp).SetMode(big.ToPositiveInf).Sub(mid, lo)
	if below.Cmp(above) > 0 {
		above = below
	}
	return num.NewBall(mid, num.MagFromBigFloat(above))
}

func squareSum(c Complex, prec uint) num.Number {
	if c.IsExact() {
		return num.NumExact(c.re.Exact().Mul(c.re.Exact()).Add(c.im.Exact().Mul(c.im.Exact())))
	}
	lo, hi := squareSumBounds(c, prec)
	return num.NumBall(spanBall(lo, hi, prec))
}

func sqrtBounds(lo, hi *big.Float, prec uint) (num.Number, error) {
	if hi.Sign() < 0 {
		return num.Number{}, ErrNotNonNegative
	}
	if lo.Sign() < 0 {
		lo = new(big.Float)
	}
	if hi.IsInf() {
		return num.NumBall(spanBall(new(big.Float), hi, prec)), nil
	}
	low, err := elem.Sqrt(num.NewBall(lo, num.Mag{}), prec)
	if err != nil {
		return num.Number{}, err
	}
	high, err := elem.Sqrt(num.NewBall(hi, num.Mag{}), prec)
	if err != nil {
		return num.Number{}, err
	}
	return num.NumBall(spanBall(low.Lo(), high.Hi(), prec)), nil
}

func sqrtNonNegative(n num.Number, prec uint) (num.Number, error) {
	if n.IsExact() {
		if root, ok := num.Root(n.Exact(), 2); ok {
			return num.NumExact(root), nil
		}
		b, err := elem.Sqrt(num.BallFromExact(n.Exact(), workPrec(prec)), prec)
		if err != nil {
			return num.Number{}, err
		}
		return num.NumBall(b), nil
	}
	b := n.Ball(prec)
	return sqrtBounds(b.Lo(), b.Hi(), prec)
}

func (c Complex) Abs(prec uint) (num.Number, error) {
	if c.isUnbounded(prec) {
		return num.Number{}, ErrUnbounded
	}
	switch {
	case c.IsReal():
		return c.re.Abs(), nil
	case isExactZero(c.re):
		return c.im.Abs(), nil
	case c.IsExact():
		return sqrtNonNegative(squareSum(c, prec), prec)
	}
	lo, hi := squareSumBounds(c, prec)
	return sqrtBounds(lo, hi, prec)
}

func (c Complex) Arg(prec uint) (num.Number, error) {
	if c.isUnbounded(prec) {
		return num.Number{}, ErrUnbounded
	}
	if c.IsReal() {
		return realArg(c.re, prec)
	}
	b, err := elem.Atan2(c.im.Ball(prec), c.re.Ball(prec), prec)
	if err != nil {
		return num.Number{}, err
	}
	return num.NumBall(b), nil
}

func realArg(re num.Number, prec uint) (num.Number, error) {
	sign, known := signOf(re, prec)
	switch {
	case !known || sign == 0:
		return num.Number{}, ErrArgOfZero
	case sign > 0:
		return zero, nil
	}
	return num.NumBall(elem.Pi(prec)), nil
}

func signOf(n num.Number, prec uint) (int, bool) {
	if n.IsExact() {
		return n.Exact().Sign(), true
	}
	return n.Ball(prec).Sign()
}

func (c Complex) Pow(k int, prec uint) (Complex, error) {
	if k > maxPowerExponent || k < -maxPowerExponent {
		return Complex{}, ErrExponentTooLarge
	}
	negative := k < 0
	if negative {
		k = -k
	}
	result := Real(one)
	base := c
	for k > 0 {
		if k&1 == 1 {
			result = result.Mul(base, prec)
		}
		k >>= 1
		if k > 0 {
			base = base.Mul(base, prec)
		}
	}
	if negative {
		return Real(one).Div(result, prec)
	}
	return result, nil
}

func Sqrt(c Complex, prec uint) (Complex, error) {
	if c.isUnbounded(prec) {
		return Complex{}, ErrUnbounded
	}
	modulus, err := c.Abs(prec)
	if err != nil {
		return Complex{}, err
	}
	re, err := sqrtNonNegative(mulNum(addNum(modulus, c.re, prec), half, prec), prec)
	if err != nil {
		return Complex{}, err
	}
	imMagnitude, err := sqrtNonNegative(mulNum(subNum(modulus, c.re, prec), half, prec), prec)
	if err != nil {
		return Complex{}, err
	}
	im, err := signedImaginary(c, imMagnitude, prec)
	if err != nil {
		return Complex{}, err
	}
	return Complex{re: re, im: im}, nil
}

func signedImaginary(c Complex, magnitude num.Number, prec uint) (num.Number, error) {
	if isExactZero(magnitude) {
		return zero, nil
	}
	sign, known := signOf(c.im, prec)
	switch {
	case known && sign >= 0:
		return magnitude, nil
	case known:
		return magnitude.Neg(), nil
	}
	if reSign, reKnown := signOf(c.re, prec); reKnown && reSign > 0 {
		bound := magnitude.Ball(prec).Hi()
		return num.NumBall(num.NewBall(new(big.Float).SetPrec(prec), num.MagFromBigFloat(bound))), nil
	}
	return num.Number{}, ErrBranchCut
}

func expNum(n num.Number, prec uint) (num.Number, error) {
	if isExactZero(n) {
		return one, nil
	}
	b, err := elem.Exp(n.Ball(prec), prec)
	if err != nil {
		return num.Number{}, err
	}
	return num.NumBall(b), nil
}

func Exp(c Complex, prec uint) (Complex, error) {
	if c.isUnbounded(prec) {
		return Complex{}, ErrUnbounded
	}
	scale, err := expNum(c.re, prec)
	if err != nil {
		return Complex{}, err
	}
	if c.IsReal() {
		return Real(scale), nil
	}
	imBall := c.im.Ball(prec)
	cosine, err := elem.Cos(imBall, prec)
	if err != nil {
		return Complex{}, err
	}
	sine, err := elem.Sin(imBall, prec)
	if err != nil {
		return Complex{}, err
	}
	return Complex{
		re: mulNum(scale, num.NumBall(cosine), prec),
		im: mulNum(scale, num.NumBall(sine), prec),
	}, nil
}

func logModulus(c Complex, prec uint) (num.Number, error) {
	if c.IsReal() {
		return logAbs(c.re.Abs(), prec)
	}
	logSum, err := logAbs(squareSum(c, prec), prec)
	if err != nil {
		return num.Number{}, err
	}
	return mulNum(logSum, half, prec), nil
}

func logAbs(n num.Number, prec uint) (num.Number, error) {
	if n.IsExact() {
		if n.Exact().Sign() <= 0 {
			return num.Number{}, ErrLogOfZero
		}
		if isExactOne(n) {
			return zero, nil
		}
	}
	b, err := elem.Log(n.Ball(prec), prec)
	if err != nil {
		return num.Number{}, ErrLogOfZero
	}
	return num.NumBall(b), nil
}

func Log(c Complex, prec uint) (Complex, error) {
	if c.isUnbounded(prec) {
		return Complex{}, ErrUnbounded
	}
	re, err := logModulus(c, prec)
	if err != nil {
		return Complex{}, err
	}
	im, err := c.Arg(prec)
	if err != nil {
		return Complex{}, err
	}
	return Complex{re: re, im: im}, nil
}

func isNegative(n num.Number) bool {
	if n.IsExact() {
		return n.Exact().Sign() < 0
	}
	b := n.Ball(guardBits)
	return b.Mid != nil && b.Mid.Sign() < 0
}

func realText(n num.Number, parenthesize bool) string {
	if n.IsExact() || !parenthesize {
		return n.String()
	}
	return "(" + n.String() + ")"
}

func imaginaryText(magnitude num.Number) string {
	switch {
	case isExactOne(magnitude):
		return "i"
	case magnitude.IsExact() && magnitude.Exact().IsInt():
		return magnitude.String() + "i"
	}
	return "(" + magnitude.String() + ")i"
}

func (c Complex) String() string {
	switch {
	case c.IsReal():
		return c.re.String()
	case isExactZero(c.re):
		if isNegative(c.im) {
			return "-" + imaginaryText(c.im.Neg())
		}
		return imaginaryText(c.im)
	}
	sign, magnitude := "+", c.im
	if isNegative(c.im) {
		sign, magnitude = "-", c.im.Neg()
	}
	return realText(c.re, true) + sign + imaginaryText(magnitude)
}

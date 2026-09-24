package num

import (
	"errors"
	"math/big"
)

var ErrDivisorMayBeZero = errors.New("divisor may be zero")

type Ball struct {
	Mid *big.Float
	Rad Mag
}

func NewBall(mid *big.Float, rad Mag) Ball {
	return Ball{Mid: new(big.Float).Set(mid), Rad: rad}
}

func BallFromExact(e Exact, prec uint) Ball {
	mid := new(big.Float).SetPrec(prec).SetRat(e.Rat())
	return Ball{Mid: mid, Rad: RoundingError(mid, prec)}
}

func oneBall(prec uint) Ball {
	return Ball{Mid: new(big.Float).SetPrec(prec).SetInt64(1)}
}

func RoundingError(mid *big.Float, prec uint) Mag {
	if mid.Acc() == big.Exact || mid.Sign() == 0 {
		return Mag{}
	}
	if mid.IsInf() {
		return Mag{newMag().SetInf(false)}
	}
	return Mag{newMag().SetMantExp(magOne, mid.MantExp(nil)-int(prec))}
}

var magOne = big.NewFloat(1)

func (b Ball) mid() *big.Float {
	if b.Mid == nil {
		return new(big.Float)
	}
	return b.Mid
}

func (b Ball) Add(o Ball, prec uint) Ball {
	mid := new(big.Float).SetPrec(prec).Add(b.mid(), o.mid())
	return Ball{Mid: mid, Rad: b.Rad.Add(o.Rad).Add(RoundingError(mid, prec))}
}

func (b Ball) Sub(o Ball, prec uint) Ball {
	mid := new(big.Float).SetPrec(prec).Sub(b.mid(), o.mid())
	return Ball{Mid: mid, Rad: b.Rad.Add(o.Rad).Add(RoundingError(mid, prec))}
}

func (b Ball) Mul(o Ball, prec uint) Ball {
	mid := new(big.Float).SetPrec(prec).Mul(b.mid(), o.mid())
	rad := b.Rad.MulFloat(o.mid()).
		Add(o.Rad.MulFloat(b.mid())).
		Add(b.Rad.Mul(o.Rad)).
		Add(RoundingError(mid, prec))
	return Ball{Mid: mid, Rad: rad}
}

func (b Ball) Div(o Ball, prec uint) (Ball, error) {
	absDen := new(big.Float).Abs(o.mid())
	if absDen.Cmp(o.Rad.float()) <= 0 {
		return Ball{}, ErrDivisorMayBeZero
	}
	mid := new(big.Float).SetPrec(prec).Quo(b.mid(), o.mid())
	num := b.Rad.MulFloat(absDen).Add(o.Rad.MulFloat(b.mid()))
	gap := newMagDown().Sub(absDen, o.Rad.float())
	den := newMagDown().Mul(absDen, gap)
	rad := Mag{newMag().Quo(num.float(), den)}
	return Ball{Mid: mid, Rad: rad.Add(RoundingError(mid, prec))}, nil
}

func (b Ball) Pow(n int, prec uint) (Ball, error) {
	if n == 0 {
		return oneBall(prec), nil
	}
	negative := n < 0
	if negative {
		n = -n
	}
	result := oneBall(prec)
	base := b
	for n > 0 {
		if n&1 == 1 {
			result = result.Mul(base, prec)
		}
		n >>= 1
		if n > 0 {
			base = base.Mul(base, prec)
		}
	}
	if negative {
		return oneBall(prec).Div(result, prec)
	}
	return result, nil
}

func (b Ball) Neg() Ball {
	return Ball{Mid: new(big.Float).Neg(b.mid()), Rad: b.Rad}
}

func (b Ball) Abs() Ball {
	return Ball{Mid: new(big.Float).Abs(b.mid()), Rad: b.Rad}
}

func (b Ball) boundPrec() uint {
	p := b.mid().Prec()
	if p < 64 {
		p = 64
	}
	return p + magPrec
}

func (b Ball) Lo() *big.Float {
	return new(big.Float).SetPrec(b.boundPrec()).SetMode(big.ToNegativeInf).Sub(b.mid(), b.Rad.float())
}

func (b Ball) Hi() *big.Float {
	return new(big.Float).SetPrec(b.boundPrec()).SetMode(big.ToPositiveInf).Add(b.mid(), b.Rad.float())
}

func (b Ball) Contains(x *big.Float) bool {
	return b.Lo().Cmp(x) <= 0 && b.Hi().Cmp(x) >= 0
}

func (b Ball) Sign() (int, bool) {
	if b.Rad.IsZero() {
		return b.mid().Sign(), true
	}
	abs := new(big.Float).Abs(b.mid())
	if abs.Cmp(b.Rad.float()) > 0 {
		return b.mid().Sign(), true
	}
	return 0, false
}

func (b Ball) String() string {
	return b.mid().Text('g', 12) + " ± " + b.Rad.String()
}

const maxDigits = 5000

var bigTen = big.NewInt(10)

func pow10(k int) *big.Float {
	if k < 0 {
		k = 0
	}
	return new(big.Float).SetInt(new(big.Int).Exp(bigTen, big.NewInt(int64(k)), nil))
}

func decimalExponent(x *big.Float) int {
	abs := new(big.Float).Abs(x)
	if abs.Sign() == 0 || abs.IsInf() {
		return 0
	}
	est := int(float64(abs.MantExp(nil)) * log2Of10Inverse)
	for compareToPow10(abs, est) < 0 {
		est--
	}
	for compareToPow10(abs, est+1) >= 0 {
		est++
	}
	return est
}

const log2Of10Inverse = 0.30102999566398119521

func compareToPow10(abs *big.Float, k int) int {
	if k >= 0 {
		return abs.Cmp(pow10(k))
	}
	scaled := new(big.Float).SetPrec(abs.Prec()+64).Mul(abs, pow10(-k))
	return scaled.Cmp(floatOne)
}

var floatOne = big.NewFloat(1)

func (b Ball) GuaranteedDigits() int {
	mid := b.mid()
	if b.Rad.IsZero() {
		if mid.Sign() == 0 {
			return maxDigits
		}
		return clampDigits(decimalDigits(mid.Prec()) - decimalExponent(mid))
	}
	rad := b.Rad.float()
	if rad.IsInf() {
		return 0
	}
	spread := newMag().Add(rad, rad)
	return clampDigits(-decimalExponent(spread) - 1)
}

func clampDigits(d int) int {
	if d < 0 {
		return 0
	}
	if d > maxDigits {
		return maxDigits
	}
	return d
}

func decimalDigits(prec uint) int {
	return clampDigits(int(float64(prec) * log2Of10Inverse))
}

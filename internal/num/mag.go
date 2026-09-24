package num

import (
	"math"
	"math/big"
)

const magPrec = 30

type Mag struct {
	f *big.Float
}

func newMag() *big.Float {
	return new(big.Float).SetPrec(magPrec).SetMode(big.ToPositiveInf)
}

func newMagDown() *big.Float {
	return new(big.Float).SetPrec(magPrec).SetMode(big.ToNegativeInf)
}

func MagFromFloat(f float64) Mag {
	if math.IsNaN(f) {
		return Mag{newMag().SetInf(false)}
	}
	return Mag{newMag().SetFloat64(math.Abs(f))}
}

func MagFromBigFloat(x *big.Float) Mag {
	m := newMag()
	m.Abs(x)
	return Mag{m}
}

func (m Mag) float() *big.Float {
	if m.f == nil {
		return new(big.Float).SetPrec(magPrec).SetMode(big.ToPositiveInf)
	}
	return m.f
}

func (m Mag) Float() *big.Float { return new(big.Float).Set(m.float()) }

func (m Mag) Add(o Mag) Mag { return Mag{newMag().Add(m.float(), o.float())} }

func (m Mag) Mul(o Mag) Mag { return Mag{newMag().Mul(m.float(), o.float())} }

func (m Mag) MulFloat(x *big.Float) Mag {
	abs := newMag()
	abs.Abs(x)
	return Mag{newMag().Mul(m.float(), abs)}
}

func (m Mag) IsZero() bool { return m.float().Sign() == 0 }

func (m Mag) Cmp(o Mag) int { return m.float().Cmp(o.float()) }

func (m Mag) String() string { return m.float().Text('g', 6) }

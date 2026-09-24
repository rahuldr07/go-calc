package num

import (
	"errors"
	"math"
	"math/big"
	"strings"
)

var ErrDivByZero = errors.New("division by zero")

type Exact struct {
	r *big.Rat
}

func FromInt(i int64) Exact { return Exact{new(big.Rat).SetInt64(i)} }

func FromRat(r *big.Rat) Exact { return Exact{new(big.Rat).Set(r)} }

func ParseExact(lit string) (Exact, error) {
	clean := strings.NewReplacer("_", "", ",", "").Replace(lit)
	if len(clean) > 2 && clean[0] == '0' {
		base := 0
		switch clean[1] {
		case 'x', 'X':
			base = 16
		case 'b', 'B':
			base = 2
		case 'o', 'O':
			base = 8
		}
		if base != 0 {
			i, ok := new(big.Int).SetString(clean[2:], base)
			if !ok {
				return Exact{}, errors.New("bad number " + lit)
			}
			return Exact{new(big.Rat).SetInt(i)}, nil
		}
	}
	r, ok := new(big.Rat).SetString(clean)
	if !ok {
		return Exact{}, errors.New("bad number " + lit)
	}
	return Exact{r}, nil
}

func (e Exact) rat() *big.Rat {
	if e.r == nil {
		return new(big.Rat)
	}
	return e.r
}

func (e Exact) Rat() *big.Rat { return new(big.Rat).Set(e.rat()) }

func (e Exact) Add(o Exact) Exact { return Exact{new(big.Rat).Add(e.rat(), o.rat())} }

func (e Exact) Sub(o Exact) Exact { return Exact{new(big.Rat).Sub(e.rat(), o.rat())} }

func (e Exact) Mul(o Exact) Exact { return Exact{new(big.Rat).Mul(e.rat(), o.rat())} }

func (e Exact) Div(o Exact) (Exact, error) {
	if o.Sign() == 0 {
		return Exact{}, ErrDivByZero
	}
	return Exact{new(big.Rat).Quo(e.rat(), o.rat())}, nil
}

func (e Exact) Neg() Exact { return Exact{new(big.Rat).Neg(e.rat())} }

func (e Exact) Sign() int { return e.rat().Sign() }

func (e Exact) IsInt() bool { return e.rat().IsInt() }

func (e Exact) Cmp(o Exact) int { return e.rat().Cmp(o.rat()) }

func (e Exact) String() string { return e.rat().RatString() }

func (e Exact) Pow(n int) Exact {
	if n == 0 {
		return FromInt(1)
	}
	base := e.rat()
	neg := n < 0
	if neg {
		n = -n
	}
	exp := big.NewInt(int64(n))
	num := new(big.Int).Exp(base.Num(), exp, nil)
	den := new(big.Int).Exp(base.Denom(), exp, nil)
	out := new(big.Rat).SetFrac(num, den)
	if neg {
		out = out.Inv(out)
	}
	return Exact{out}
}

func (e Exact) RoughWidth() float64 {
	r := e.rat()
	if r.Sign() == 0 {
		return 0
	}
	whole := intRoughLog10(r.Num())
	if part := intRoughLog10(r.Denom()); part > whole {
		return part
	}
	return whole
}

func intRoughLog10(x *big.Int) float64 {
	if x.Sign() == 0 {
		return 0
	}
	mant := new(big.Float).SetPrec(64)
	exp := new(big.Float).SetPrec(64).SetInt(x).MantExp(mant)
	m, _ := mant.Float64()
	return (float64(exp) + math.Log2(math.Abs(m))) * log2Of10Inverse
}

const maxRootBits = 8192

func Root(x Exact, n int) (Exact, bool) {
	if n <= 0 || x.Sign() < 0 {
		return Exact{}, false
	}
	if x.rat().Num().BitLen() > maxRootBits || x.rat().Denom().BitLen() > maxRootBits {
		return Exact{}, false
	}
	num, ok := intRoot(x.rat().Num(), n)
	if !ok {
		return Exact{}, false
	}
	den, ok := intRoot(x.rat().Denom(), n)
	if !ok {
		return Exact{}, false
	}
	return Exact{new(big.Rat).SetFrac(num, den)}, true
}

func intRoot(v *big.Int, n int) (*big.Int, bool) {
	if v.Sign() == 0 {
		return big.NewInt(0), true
	}
	lo, hi := big.NewInt(1), new(big.Int).Add(v, big.NewInt(1))
	exp := big.NewInt(int64(n))
	one := big.NewInt(1)
	two := big.NewInt(2)
	for lo.Cmp(hi) < 0 {
		mid := new(big.Int).Add(lo, hi)
		mid.Div(mid, two)
		p := new(big.Int).Exp(mid, exp, nil)
		switch p.Cmp(v) {
		case 0:
			return mid, true
		case -1:
			lo = new(big.Int).Add(mid, one)
		default:
			hi = mid
		}
	}
	return nil, false
}

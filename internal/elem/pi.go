package elem

import (
	"math/big"
	"sync"

	"github.com/rahuldr07/go-calc/internal/num"
)

func fixedBall(v *big.Int, shift uint, errUlps int64, prec uint) num.Ball {
	mid := new(big.Float).SetPrec(prec).SetInt(v)
	rad := num.RoundingError(mid, prec).Add(magOf(new(big.Float).SetInt64(errUlps)))
	mid.SetMantExp(mid, -int(shift))
	return num.NewBall(mid, magScale(rad, -int(shift)))
}

func atanInverseFixed(n int64, shift uint) (*big.Int, int64) {
	scale := new(big.Int).Lsh(big.NewInt(1), shift)
	base := big.NewInt(n)
	square := new(big.Int).Mul(base, base)
	power := new(big.Int).Quo(scale, base)
	sum := new(big.Int)
	term := new(big.Int)
	divisor := new(big.Int)
	var k int64
	for power.Sign() != 0 && k < maxTerms {
		term.Quo(power, divisor.SetInt64(2*k+1))
		if k&1 == 0 {
			sum.Add(sum, term)
		} else {
			sum.Sub(sum, term)
		}
		power.Quo(power, square)
		k++
	}
	return sum, 3*k + 8
}

func piFixed(shift uint) (*big.Int, int64) {
	a5, e5 := atanInverseFixed(5, shift)
	a239, e239 := atanInverseFixed(239, shift)
	v := new(big.Int).Lsh(a5, 4)
	v.Sub(v, new(big.Int).Lsh(a239, 2))
	return v, 16*e5 + 4*e239
}

func eFixed(shift uint) (*big.Int, int64) {
	term := new(big.Int).Lsh(big.NewInt(1), shift)
	sum := new(big.Int).Set(term)
	divisor := new(big.Int)
	var k int64 = 1
	for term.Sign() != 0 && k < maxTerms {
		term.Quo(term, divisor.SetInt64(k))
		sum.Add(sum, term)
		k++
	}
	return sum, 2*k + 8
}

func ln2Fixed(shift uint) (*big.Int, int64) {
	scale := new(big.Int).Lsh(big.NewInt(1), shift)
	power := new(big.Int).Quo(scale, big.NewInt(3))
	nine := big.NewInt(9)
	sum := new(big.Int)
	term := new(big.Int)
	divisor := new(big.Int)
	var k int64
	for power.Sign() != 0 && k < maxTerms {
		term.Quo(power, divisor.SetInt64(2*k+1))
		sum.Add(sum, term)
		power.Quo(power, nine)
		k++
	}
	sum.Lsh(sum, 1)
	return sum, 6*k + 8
}

type constCache struct {
	mu    sync.Mutex
	prec  uint
	value num.Ball
}

func (c *constCache) get(prec uint, build func(uint) num.Ball) num.Ball {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.value.Mid == nil || c.prec < prec {
		p := prec
		if p < 256 {
			p = 256
		}
		c.value = build(p)
		c.prec = p
	}
	return num.NewBall(c.value.Mid, c.value.Rad)
}

var (
	piCache  constCache
	eCache   constCache
	ln2Cache constCache
)

func piBall(prec uint) num.Ball {
	return piCache.get(prec, func(p uint) num.Ball {
		shift := p + 32
		v, e := piFixed(shift)
		return fixedBall(v, shift, e, p)
	})
}

func eBall(prec uint) num.Ball {
	return eCache.get(prec, func(p uint) num.Ball {
		shift := p + 32
		v, e := eFixed(shift)
		return fixedBall(v, shift, e, p)
	})
}

func ln2Ball(prec uint) num.Ball {
	return ln2Cache.get(prec, func(p uint) num.Ball {
		shift := p + 32
		v, e := ln2Fixed(shift)
		return fixedBall(v, shift, e, p)
	})
}

func halfPiBall(prec uint) num.Ball { return scaleBall(piBall(prec), -1) }

func Pi(prec uint) num.Ball { return finish(piBall(workPrec(prec)), prec) }

func E(prec uint) num.Ball { return finish(eBall(workPrec(prec)), prec) }

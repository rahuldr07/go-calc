package format

import (
	"math"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/value"
)

const uncertaintyDigits = 2

func (s Style) Uncertain(v value.Value) (string, string, bool) {
	u, ok := reported(v)
	if !ok {
		return "", "", false
	}
	mid, ok := midpoint(v)
	if !ok {
		return "", "", false
	}
	decimals := placesFor(u)
	if !provenTo(v, decimals) {
		return "", "", false
	}
	return s.groupDecimal(at(mid, decimals)), at(new(big.Rat).SetFloat64(u), decimals), true
}

func reported(v value.Value) (float64, bool) {
	u := v.Unc.Sigma
	if k := v.Unc.Cover.K; k > 0 {
		u *= k
	}
	if f, _ := v.FactorOrOne().Rat().Float64(); f != 0 {
		u /= math.Abs(f)
	}
	if u <= 0 || math.IsInf(u, 0) || math.IsNaN(u) {
		return 0, false
	}
	return u, true
}

func midpoint(v value.Value) (*big.Rat, bool) {
	if v.Num.IsExact() {
		return v.Num.Exact().Rat(), true
	}
	r, _ := v.Num.Ball(ballPrec).Mid.Rat(nil)
	if r == nil {
		return nil, false
	}
	return r, true
}

func provenTo(v value.Value, decimals int) bool {
	if v.Num.IsExact() {
		return true
	}
	if decimals < 0 {
		decimals = 0
	}
	b := v.Num.Ball(ballPrec)
	return b.Lo().Text('f', decimals) == b.Hi().Text('f', decimals)
}

func placesFor(u float64) int {
	decimals := uncertaintyDigits - 1 - int(math.Floor(math.Log10(u)))
	rounded, ok := new(big.Rat).SetString(at(new(big.Rat).SetFloat64(u), decimals))
	if !ok || rounded.Sign() == 0 {
		return decimals
	}
	f, _ := rounded.Float64()
	if again := uncertaintyDigits - 1 - int(math.Floor(math.Log10(f))); again != decimals {
		return again
	}
	return decimals
}

func at(r *big.Rat, decimals int) string {
	if decimals >= 0 {
		return r.FloatString(decimals)
	}
	step := ratPow10(-decimals)
	whole, ok := new(big.Int).SetString(new(big.Rat).Quo(r, step).FloatString(0), 10)
	if !ok {
		return r.FloatString(0)
	}
	return new(big.Int).Mul(whole, step.Num()).String()
}

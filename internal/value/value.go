package value

import (
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/units"
)

type Value struct {
	Num    num.Exact
	Dim    units.Dim
	Unit   string
	Factor num.Exact
	Offset num.Exact
}

func Scalar(n num.Exact) Value {
	return Value{Num: n, Factor: num.FromInt(1)}
}

func WithUnit(n num.Exact, u units.Unit) Value {
	return Value{Num: n, Dim: u.Dim, Unit: u.Symbol, Factor: u.Factor, Offset: u.Offset}
}

func (v Value) Unwrap() units.Unit {
	return units.Unit{Symbol: v.Unit, Dim: v.Dim, Factor: v.FactorOrOne(), Offset: v.Offset}
}

func (v Value) IsAffine() bool { return v.Offset.Sign() != 0 }

func (v Value) IsScalar() bool { return v.Dim.IsNone() && v.Unit == "" }

func (v Value) FactorOrOne() num.Exact {
	if v.Factor.Sign() == 0 {
		return num.FromInt(1)
	}
	return v.Factor
}

func (v Value) SI() num.Exact { return v.Num.Mul(v.FactorOrOne()).Add(v.Offset) }

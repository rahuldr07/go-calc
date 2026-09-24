package value

import (
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/units"
)

type Value struct {
	Num     num.Number
	Dim     units.Dim
	Unit    string
	Factor  num.Exact
	Offset  num.Exact
	Stopped Stop
	Sens    unc.Sens
	Curv    unc.Curv
	Unc     unc.Report
}

func (v Value) Jet() unc.Jet { return unc.Jet{Grad: v.Sens, Hess: v.Curv} }

func (v Value) WithJet(j unc.Jet) Value {
	v.Sens, v.Curv, v.Unc = j.Grad, j.Hess, unc.Report{}
	return v
}

func (v Value) IsMeasured() bool { return len(v.Sens) > 0 || len(v.Curv) > 0 }

func Scalar(n num.Exact) Value { return ScalarNum(num.NumExact(n)) }

func ScalarNum(n num.Number) Value {
	return Value{Num: n, Factor: num.FromInt(1)}
}

func WithUnit(n num.Exact, u units.Unit) Value { return WithUnitNum(num.NumExact(n), u) }

func WithUnitNum(n num.Number, u units.Unit) Value {
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

func (v Value) SI(prec uint) num.Number {
	return v.Num.Mul(num.NumExact(v.FactorOrOne()), prec).Add(num.NumExact(v.Offset), prec)
}

type Stop int

const (
	StopNone Stop = iota
	StopPrecisionCap
	StopUncertainInput
	StopDeadline
)

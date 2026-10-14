package eval

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

var errUndecidedClamp = errors.New("undecided at this precision: clamp cannot prove which side of a limit the value is on")

func init() {
	register("sign", builtinFunc{min: 1, max: 1, usage: "sign takes one value", run: signCall})
	register("frac", builtinFunc{min: 1, max: 1, usage: "frac takes one value", run: fracCall})
	register("clamp", builtinFunc{min: 3, max: 3, usage: "clamp takes a value, a lower limit and an upper limit", run: clampCall})
}

func signCall(c call) (value.Value, error) {
	x := c.args[0]
	if x.IsAffine() {
		return value.Value{}, errors.New("sign is not defined for absolute temperatures: use differences such as 5 K")
	}
	if x.Num.IsExact() {
		return value.Scalar(num.FromInt(int64(x.Num.Exact().Sign()))), nil
	}
	s, decided := x.Num.Ball(c.prec).Sign()
	if !decided {
		return value.Value{}, errors.New("undecided at this precision: the value may be zero, so its sign is neither proven positive nor negative")
	}
	return value.Scalar(num.FromInt(int64(s))), nil
}

func fracCall(c call) (value.Value, error) {
	x := c.args[0]
	if x.IsAffine() {
		return value.Value{}, errors.New("frac is not defined for absolute temperatures")
	}
	whole, err := roundNumber(x.Num, "trunc", 0, c.prec)
	if err != nil {
		return value.Value{}, errors.New("frac: " + err.Error())
	}
	out := x
	out.Num = x.Num.Sub(num.NumExact(whole), c.prec)
	return out, nil
}

func clampCall(c call) (value.Value, error) {
	x, lo, hi := c.args[0], c.args[1], c.args[2]
	if x.Dim != lo.Dim || x.Dim != hi.Dim {
		return value.Value{}, errors.New("clamp needs the value and both limits in the same kind of quantity")
	}
	order := func(a, b value.Value) (int, error) {
		o, err := orderOfNumbers(a.SI(c.prec), b.SI(c.prec), c.prec, lex.Pos{})
		if err != nil {
			return 0, errUndecidedClamp
		}
		return o, nil
	}
	if bad, err := order(lo, hi); err != nil || bad > 0 {
		if err != nil {
			return value.Value{}, err
		}
		return value.Value{}, errors.New("clamp needs the lower limit below the upper limit")
	}
	if below, err := order(x, lo); err != nil || below < 0 {
		if err != nil {
			return value.Value{}, err
		}
		return lo, nil
	}
	if above, err := order(x, hi); err != nil || above > 0 {
		if err != nil {
			return value.Value{}, err
		}
		return hi, nil
	}
	return x, nil
}

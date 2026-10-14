package eval

import (
	"errors"
	"math/big"
	"strconv"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

type call struct {
	name    string
	args    []value.Value
	prec    uint
	degrees bool
}

type builtinFunc struct {
	min, max int
	usage    string
	lists    bool
	run      func(c call) (value.Value, error)
}

const anyCount = -1

var registered = map[string]builtinFunc{}

func register(name string, f builtinFunc) { registered[name] = f }

func (f builtinFunc) invoke(name string, args []value.Value, prec uint, degrees bool) (value.Value, error) {
	if len(args) < f.min || (f.max != anyCount && len(args) > f.max) {
		return value.Value{}, errors.New(f.usage)
	}
	for _, a := range args {
		if a.IsList() && !f.lists {
			return value.Value{}, errors.New(name + " takes a number, not a list")
		}
		if a.IsMeasured() {
			return value.Value{}, errors.New(name + " cannot propagate a measured uncertainty, so it needs exact or proven inputs")
		}
		if a.IsBoolean() {
			return value.Value{}, errors.New(name + " takes numbers, not true/false values")
		}
		if a.IsComplex() {
			return value.Value{}, errors.New(name + " takes real numbers")
		}
	}
	return f.run(call{name: name, args: args, prec: prec, degrees: degrees})
}

func (c call) whole(i int) (*big.Int, error) {
	n, ok := wholeInt(c.args[i])
	if !ok {
		return nil, errors.New(c.name + " takes whole numbers")
	}
	return n, nil
}

func (c call) bounded(i int, lo, hi int64) (int64, error) {
	n, ok := smallWhole(c.args[i], lo, hi)
	if !ok {
		return 0, errors.New(c.name + " takes a whole number from " + strconv.FormatInt(lo, 10) + " to " + strconv.FormatInt(hi, 10))
	}
	return n, nil
}

func (c call) plain(i int) (num.Number, error) {
	if !plainNumber(c.args[i]) {
		return num.Number{}, errors.New(c.name + " needs a plain number")
	}
	return c.args[i].Num, nil
}

func integer(n *big.Int) value.Value {
	return value.Scalar(num.FromRat(new(big.Rat).SetInt(n)))
}

func integers(ns []*big.Int) value.Value {
	items := make([]value.Value, len(ns))
	for i, n := range ns {
		items[i] = integer(n)
	}
	return value.List(items)
}

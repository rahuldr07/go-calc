package eval

import (
	"errors"
	"sort"

	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

var errListOrder = errors.New("undecided at this precision: two values overlap, so their order is neither proven nor refuted")

func init() {
	many := func(name string, run func(c call, items []value.Value) (value.Value, error)) {
		register(name, builtinFunc{min: 1, max: anyCount, lists: true, usage: name + " takes a list or some values", run: func(c call) (value.Value, error) {
			items := flatten(c.args)
			if err := plainItems(name, items); err != nil {
				return value.Value{}, err
			}
			if err := sameKind(items); len(items) > 0 && err != nil {
				return value.Value{}, errors.New(name + ": " + err.Error())
			}
			return run(c, items)
		}})
	}
	many("sort", sortCall)
	many("reverse", reverseCall)
	many("unique", uniqueCall)
	many("len", lenCall)
	many("mode", modeCall)
	many("cumsum", cumsumCall)
	many("diff", diffCall)
	register("percentile", builtinFunc{min: 2, max: 2, lists: true, usage: "percentile takes a list and a percentage from 0 to 100", run: percentileCall})
}

func sortedItems(c call, items []value.Value) ([]value.Value, error) {
	out := append([]value.Value(nil), items...)
	var failure error
	sort.SliceStable(out, func(i, j int) bool {
		o, err := orderOfNumbers(out[i].SI(c.prec), out[j].SI(c.prec), c.prec, lex.Pos{})
		if err != nil {
			failure = errListOrder
		}
		return o < 0
	})
	return out, failure
}

func sortCall(c call, items []value.Value) (value.Value, error) {
	out, err := sortedItems(c, items)
	if err != nil {
		return value.Value{}, errors.New("sort: " + err.Error())
	}
	return value.List(out), nil
}

func reverseCall(_ call, items []value.Value) (value.Value, error) {
	out := make([]value.Value, len(items))
	for i, v := range items {
		out[len(items)-1-i] = v
	}
	return value.List(out), nil
}

func lenCall(_ call, items []value.Value) (value.Value, error) {
	return value.Scalar(num.FromInt(int64(len(items)))), nil
}

func exactKey(c call, v value.Value) (string, error) {
	si := v.SI(c.prec)
	if !si.IsExact() {
		return "", errors.New(c.name + " compares values for equality, so it needs exact values")
	}
	return si.Exact().String(), nil
}

func uniqueCall(c call, items []value.Value) (value.Value, error) {
	seen := map[string]bool{}
	var out []value.Value
	for _, v := range items {
		key, err := exactKey(c, v)
		if err != nil {
			return value.Value{}, err
		}
		if !seen[key] {
			seen[key] = true
			out = append(out, v)
		}
	}
	return value.List(out), nil
}

func modeCall(c call, items []value.Value) (value.Value, error) {
	if len(items) == 0 {
		return value.Value{}, errors.New("mode needs at least one value")
	}
	counts := map[string]int{}
	first := map[string]value.Value{}
	best := 0
	for _, v := range items {
		key, err := exactKey(c, v)
		if err != nil {
			return value.Value{}, err
		}
		counts[key]++
		if _, ok := first[key]; !ok {
			first[key] = v
		}
		best = max(best, counts[key])
	}
	var modes []value.Value
	for _, v := range items {
		key, _ := exactKey(c, v)
		if counts[key] == best {
			modes = append(modes, v)
			counts[key] = -1
		}
	}
	if len(modes) == 1 {
		return modes[0], nil
	}
	sorted, err := sortedItems(c, modes)
	if err != nil {
		return value.Value{}, err
	}
	return value.List(sorted), nil
}

func percentileCall(c call) (value.Value, error) {
	if !c.args[0].IsList() || c.args[1].IsList() {
		return value.Value{}, errors.New("percentile takes a list and a percentage from 0 to 100")
	}
	items := c.args[0].List.Items
	if len(items) == 0 {
		return value.Value{}, errors.New("percentile needs at least one value")
	}
	if err := plainItems("percentile", items); err != nil {
		return value.Value{}, err
	}
	if err := sameKind(items); err != nil {
		return value.Value{}, errors.New("percentile: " + err.Error())
	}
	p := c.args[1]
	if !plainNumber(p) || !p.Num.IsExact() || p.Num.Exact().Sign() < 0 || p.Num.Exact().Cmp(num.FromInt(100)) > 0 {
		return value.Value{}, errors.New("percentile takes an exact percentage from 0 to 100")
	}
	sorted, err := sortedItems(c, items)
	if err != nil {
		return value.Value{}, errors.New("percentile: " + err.Error())
	}
	rank := p.Num.Exact().Mul(num.FromInt(int64(len(sorted) - 1)))
	rank, _ = rank.Div(num.FromInt(100))
	below := roundTo(rank, "floor")
	at := int(below.Rat().Num().Int64())
	if at >= len(sorted)-1 {
		return sorted[len(sorted)-1], nil
	}
	weight := num.NumExact(rank.Sub(below))
	lo, hi := sorted[at].SI(c.prec), sorted[at+1].SI(c.prec)
	mixed := lo.Add(hi.Sub(lo, c.prec).Mul(weight, c.prec), c.prec)
	return inUnitOf(sorted[0], mixed, c.prec)
}

func cumsumCall(c call, items []value.Value) (value.Value, error) {
	if err := noAbsoluteTemperature(c, items); err != nil {
		return value.Value{}, err
	}
	out := make([]value.Value, len(items))
	running := num.NumExact(num.FromInt(0))
	for i, v := range items {
		running = running.Add(v.SI(c.prec), c.prec)
		next, err := inUnitOf(items[0], running, c.prec)
		if err != nil {
			return value.Value{}, err
		}
		out[i] = next
	}
	return value.List(out), nil
}

func diffCall(c call, items []value.Value) (value.Value, error) {
	if err := noAbsoluteTemperature(c, items); err != nil {
		return value.Value{}, err
	}
	if len(items) < 2 {
		return value.List(nil), nil
	}
	out := make([]value.Value, len(items)-1)
	for i := 1; i < len(items); i++ {
		step, err := inUnitOf(items[0], items[i].SI(c.prec).Sub(items[i-1].SI(c.prec), c.prec), c.prec)
		if err != nil {
			return value.Value{}, err
		}
		out[i-1] = step
	}
	return value.List(out), nil
}

func plainItems(name string, items []value.Value) error {
	for _, v := range items {
		if msg := unlistable(v); msg != "" {
			return errors.New(name + ": " + msg)
		}
		if v.IsMeasured() {
			return errors.New(name + " cannot carry a measured uncertainty through, so it needs exact or proven values")
		}
	}
	return nil
}

func noAbsoluteTemperature(c call, items []value.Value) error {
	for _, v := range items {
		if v.IsAffine() {
			return errors.New(c.name + " is not defined for absolute temperatures: use differences such as 5 K")
		}
	}
	return nil
}

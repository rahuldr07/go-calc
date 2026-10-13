package eval

import (
	"math/big"
	"slices"
	"strconv"

	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

const maxListLength = 100000

var statisticalFunctions = []string{"sum", "product", "mean", "median", "min", "max", "count", "stdev", "variance", "stdevp", "variancep"}

var pickingFunctions = map[string]bool{"min": true, "max": true, "count": true}

var orderingFunctions = map[string]bool{"min": true, "max": true, "median": true}

var spreadFunctions = map[string]bool{"stdev": true, "variance": true, "stdevp": true, "variancep": true}

var spreadDescriptions = map[string]string{
	"stdev":     "the sample standard deviation: it divides by n - 1, and stdevp divides by n",
	"variance":  "the sample variance: it divides by n - 1, and variancep divides by n",
	"stdevp":    "the population standard deviation: it divides by n, and stdev divides by n - 1",
	"variancep": "the population variance: it divides by n, and variance divides by n - 1",
}

func acceptsLists(name string) bool { return slices.Contains(statisticalFunctions, name) }

func (e *Env) listLiteral(t *parse.Call, p *pass) (value.Value, error) {
	items := make([]value.Value, len(t.Args))
	for i, a := range t.Args {
		v, err := e.eval(a, p)
		if err != nil {
			return value.Value{}, err
		}
		if msg := unlistable(v); msg != "" {
			return value.Value{}, &lex.Error{Pos: a.Pos(), Msg: msg}
		}
		items[i] = v
	}
	if err := sameKind(items); err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a list holds one kind of quantity: " + err.Error()}
	}
	return value.List(items), nil
}

func (e *Env) indexCall(t *parse.Call, p *pass) (value.Value, error) {
	refuse := func(at lex.Pos, msg string) (value.Value, error) {
		return value.Value{}, &lex.Error{Pos: at, Msg: msg}
	}
	list, err := e.eval(t.Args[0], p)
	if err != nil {
		return value.Value{}, err
	}
	if !list.IsList() {
		return refuse(t.P, "only a list can be indexed: [1, 2, 3][0] is 1")
	}
	at, err := e.eval(t.Args[1], p)
	if err != nil {
		return value.Value{}, err
	}
	n, ok := smallWhole(at, -maxListLength, maxListLength)
	if !ok || at.IsMeasured() {
		return refuse(t.Args[1].Pos(), "an index is a whole number: 0 is the first item and -1 the last")
	}
	items := list.List.Items
	size := int64(len(items))
	if size == 0 {
		return refuse(t.Args[1].Pos(), "the list is empty, so it has no items to pick")
	}
	pick := n
	if pick < 0 {
		pick += size
	}
	if pick < 0 || pick >= size {
		return refuse(t.Args[1].Pos(), "index "+strconv.FormatInt(n, 10)+" is outside a list of "+strconv.FormatInt(size, 10)+": use 0 to "+strconv.FormatInt(size-1, 10)+", or -1 for the last")
	}
	return items[pick], nil
}

func unlistable(v value.Value) string {
	switch {
	case v.IsList():
		return "a list cannot hold another list"
	case v.IsBoolean():
		return "a list holds numbers, not true/false values"
	case v.IsSymbolic() || v.IsDefinition() || v.Solved != nil:
		return "a list holds numbers"
	}
	return ""
}

func sameKind(items []value.Value) error {
	for _, v := range items[min(1, len(items)):] {
		if v.Dim != items[0].Dim {
			return &mixedKindError{first: items[0], other: v}
		}
	}
	return nil
}

type mixedKindError struct{ first, other value.Value }

func (m *mixedKindError) Error() string {
	return "cannot mix " + m.first.Dim.Words() + " (" + unitOrNone(m.first) + ") with " + m.other.Dim.Words() + " (" + unitOrNone(m.other) + ")"
}

func (e *Env) rangeCall(t *parse.Call, p *pass) (value.Value, error) {
	ends := make([]*big.Int, len(t.Args))
	for i, a := range t.Args {
		v, err := e.eval(a, p)
		if err != nil {
			return value.Value{}, err
		}
		if v.IsList() || v.IsBoolean() || v.IsMeasured() || !plainNumber(v) || !isWhole(v) {
			return value.Value{}, &lex.Error{Pos: a.Pos(), Msg: "a range runs between whole numbers"}
		}
		ends[i] = v.Num.Exact().Rat().Num()
	}
	start, stop, step := ends[0], ends[1], big.NewInt(1)
	switch {
	case len(ends) == 3:
		step = ends[2]
	case stop.Cmp(start) < 0:
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a range counts upwards: for " + start.String() + " down to " + stop.String() + " write " + start.String() + ".." + stop.String() + "..-1"}
	}
	if step.Sign() == 0 {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a range cannot step by zero"}
	}
	if span := new(big.Int).Sub(stop, start); span.Sign() != 0 && span.Sign() != step.Sign() {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "a range from " + start.String() + " to " + stop.String() + " never gets there with a step of " + step.String()}
	}
	count := new(big.Int).Quo(new(big.Int).Sub(stop, start), step)
	count.Add(count, big.NewInt(1))
	if count.Cmp(big.NewInt(maxListLength)) > 0 {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: "that range has " + count.String() + " values: a list holds at most " + strconv.Itoa(maxListLength)}
	}
	items := make([]value.Value, count.Int64())
	at := new(big.Int).Set(start)
	for i := range items {
		items[i] = value.Scalar(num.FromRat(new(big.Rat).SetInt(at)))
		at.Add(at, step)
	}
	return value.List(items), nil
}

func flatten(args []value.Value) []value.Value {
	var items []value.Value
	for _, a := range args {
		if a.IsList() {
			items = append(items, a.List.Items...)
			continue
		}
		items = append(items, a)
	}
	return items
}

func (e *Env) statistic(t *parse.Call, args []value.Value, p *pass) (value.Value, error) {
	fail := func(msg string) (value.Value, error) {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: msg}
	}
	items := flatten(args)
	for _, v := range items {
		if msg := unlistable(v); msg != "" {
			return fail(t.Fn + ": " + msg)
		}
	}
	if len(items) == 0 {
		if t.Fn == "count" {
			return value.Scalar(num.FromInt(0)), nil
		}
		return fail(t.Fn + " needs at least one value, and this list is empty")
	}
	if err := sameKind(items); err != nil {
		return fail(t.Fn + ": " + err.Error())
	}
	if t.Fn == "count" {
		return value.Scalar(num.FromInt(int64(len(items)))), nil
	}
	if !pickingFunctions[t.Fn] {
		for _, v := range items {
			if v.IsAffine() {
				return fail(t.Fn + " is not defined for absolute temperatures: use differences such as 5 K")
			}
		}
	}
	if (orderingFunctions[t.Fn] || spreadFunctions[t.Fn]) && measured(items...) {
		return value.Value{}, cannotPropagate(t.P, t.Fn)
	}
	if spreadFunctions[t.Fn] {
		p.notes = append(p.notes, parse.Note{Pos: t.P, Msg: "read as " + spreadDescriptions[t.Fn]})
	}
	prec := p.prec()
	switch t.Fn {
	case "sum":
		return sumOf(items, prec)
	case "mean":
		return meanOf(items, prec)
	case "product":
		return productOf(items, prec)
	case "min", "max":
		return extremeOf(t, items, prec)
	case "median":
		return medianOf(t, items, prec)
	}
	return spreadOf(t, items, prec)
}

func inUnitOf(first value.Value, n num.Number, prec uint) (value.Value, error) {
	scaled, err := n.Div(num.NumExact(first.FactorOrOne()), prec)
	if err != nil {
		return value.Value{}, err
	}
	out := first
	out.Num = scaled
	return out.WithJet(unc.Jet{}), nil
}

func siNumbers(items []value.Value, prec uint) []num.Number {
	out := make([]num.Number, len(items))
	for i, v := range items {
		out[i] = v.SI(prec)
	}
	return out
}

func total(xs []num.Number, prec uint) num.Number {
	sum := num.NumExact(num.FromInt(0))
	for _, x := range xs {
		sum = sum.Add(x, prec)
	}
	return sum
}

func summedJet(items []value.Value) unc.Jet {
	var j unc.Jet
	for _, v := range items {
		j = unc.Compose(j, 1, 0, v.Jet(), 1, 0, 0)
	}
	return j
}

func sumOf(items []value.Value, prec uint) (value.Value, error) {
	out, err := inUnitOf(items[0], total(siNumbers(items, prec), prec), prec)
	if err != nil {
		return value.Value{}, err
	}
	if measured(items...) {
		out = out.WithJet(summedJet(items))
	}
	return out, nil
}

func meanOf(items []value.Value, prec uint) (value.Value, error) {
	n := num.NumExact(num.FromInt(int64(len(items))))
	mean, err := total(siNumbers(items, prec), prec).Div(n, prec)
	if err != nil {
		return value.Value{}, err
	}
	out, err := inUnitOf(items[0], mean, prec)
	if err != nil {
		return value.Value{}, err
	}
	if measured(items...) {
		out = out.WithJet(unc.Unary(summedJet(items), 1/float64(len(items)), 0))
	}
	return out, nil
}

func productOf(items []value.Value, prec uint) (value.Value, error) {
	product := num.NumExact(num.FromInt(1))
	var dim units.Dim
	for _, v := range items {
		product = product.Mul(v.SI(prec), prec)
		dim = dim.Add(v.Dim)
	}
	out := value.Value{Num: product, Dim: dim, Factor: num.FromInt(1)}
	if !dim.IsNone() {
		out.Unit = units.SISymbol(dim)
	}
	if !measured(items...) {
		return out, nil
	}
	running, jet := siFloat(items[0], prec), items[0].Jet()
	for _, v := range items[1:] {
		x := siFloat(v, prec)
		jet = unc.Compose(jet, x, 0, v.Jet(), running, 0, 1)
		running *= x
	}
	return out.WithJet(jet), nil
}

func orderOfNumbers(a, b num.Number, prec uint, at lex.Pos) (int, error) {
	if a.IsExact() && b.IsExact() {
		return a.Exact().Cmp(b.Exact()), nil
	}
	sign, decided := a.Sub(b, prec).Ball(prec).Sign()
	if !decided {
		return 0, undecidedError{&lex.Error{Pos: at, Msg: "undecided at this precision: two values overlap, so their order is neither proven nor refuted"}}
	}
	return sign, nil
}

func extremeOf(t *parse.Call, items []value.Value, prec uint) (value.Value, error) {
	xs := siNumbers(items, prec)
	want := 1
	if t.Fn == "min" {
		want = -1
	}
	best := 0
	for i := 1; i < len(xs); i++ {
		order, err := orderOfNumbers(xs[i], xs[best], prec, t.P)
		if err != nil {
			return value.Value{}, err
		}
		if order == want {
			best = i
		}
	}
	return items[best].WithJet(unc.Jet{}), nil
}

func medianOf(t *parse.Call, items []value.Value, prec uint) (value.Value, error) {
	xs := siNumbers(items, prec)
	var failure error
	slices.SortFunc(xs, func(a, b num.Number) int {
		order, err := orderOfNumbers(a, b, prec, t.P)
		if err != nil && failure == nil {
			failure = err
		}
		return order
	})
	if failure != nil {
		return value.Value{}, failure
	}
	mid := len(xs) / 2
	if len(xs)%2 == 1 {
		return inUnitOf(items[0], xs[mid], prec)
	}
	between, err := xs[mid-1].Add(xs[mid], prec).Div(num.NumExact(num.FromInt(2)), prec)
	if err != nil {
		return value.Value{}, err
	}
	return inUnitOf(items[0], between, prec)
}

func spreadOf(t *parse.Call, items []value.Value, prec uint) (value.Value, error) {
	population := t.Fn == "stdevp" || t.Fn == "variancep"
	freedom := len(items) - 1
	if population {
		freedom = len(items)
	}
	if freedom == 0 {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: t.Fn + " needs at least two values: one value has no spread to estimate from a sample"}
	}
	xs := siNumbers(items, prec)
	mean, err := total(xs, prec).Div(num.NumExact(num.FromInt(int64(len(xs)))), prec)
	if err != nil {
		return value.Value{}, err
	}
	squares := num.NumExact(num.FromInt(0))
	for _, x := range xs {
		d := x.Sub(mean, prec)
		squares = squares.Add(d.Mul(d, prec), prec)
	}
	variance, err := squares.Div(num.NumExact(num.FromInt(int64(freedom))), prec)
	if err != nil {
		return value.Value{}, err
	}
	if t.Fn == "variance" || t.Fn == "variancep" {
		return varianceValue(items[0], variance, prec)
	}
	deviation, err := squareRoot(variance, prec)
	if err != nil {
		return value.Value{}, &lex.Error{Pos: t.P, Msg: t.Fn + ": " + err.Error()}
	}
	return inUnitOf(items[0], deviation, prec)
}

func squareRoot(n num.Number, prec uint) (num.Number, error) {
	if n.IsExact() {
		if r, ok := num.Root(n.Exact(), 2); ok {
			return num.NumExact(r), nil
		}
	}
	b, err := elem.Sqrt(n.Ball(prec), prec)
	if err != nil {
		return num.Number{}, err
	}
	return num.NumBall(b), nil
}

func varianceValue(first value.Value, variance num.Number, prec uint) (value.Value, error) {
	dim := first.Dim.Scale(2)
	factor := first.FactorOrOne().Pow(2)
	scaled, err := variance.Div(num.NumExact(factor), prec)
	if err != nil {
		return value.Value{}, err
	}
	if dim.IsNone() {
		return value.ScalarNum(variance), nil
	}
	out := value.Value{Num: scaled, Dim: dim, Factor: factor}
	out.Unit = units.SISymbol(dim)
	if first.Unit != "" && !spellsItsOwnSize(first) {
		out.Unit = grouped(first.Unit, "/·^") + "^2"
	}
	return out, nil
}

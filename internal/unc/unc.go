package unc

import (
	"errors"
	"maps"
	"math"
	"slices"
)

type InputID int

type Sens map[InputID]float64

type Pair struct{ A, B InputID }

type Curv map[Pair]float64

type Jet struct {
	Grad Sens
	Hess Curv
}

func (j Jet) IsZero() bool { return len(j.Grad) == 0 && len(j.Hess) == 0 }

type Input struct {
	ID    InputID
	Name  string
	Value float64
	Sigma float64
	Line  int
}

type Table struct {
	inputs []Input
	corr   map[Pair]float64
}

var ErrInconsistentCorrelation = errors.New("the declared correlations are not consistent: no real combined uncertainty exists")

const noUncertainty = "no uncertainty can be reported here: "

const (
	tooLarge = " is larger than a 64-bit float can hold"
	notANum  = " is not a number"
)

const (
	firstOrder    = "a first-order sensitivity"
	secondOrder   = "a second-order curvature"
	declaredSigma = "a declared uncertainty"
	combinedVar   = "the combined variance"
)

var ErrSensitivityOverflow = errors.New(noUncertainty + "a derivative or an uncertainty" + tooLarge)

var ErrSensitivityUndefined = errors.New(noUncertainty + "a derivative" + notANum)

type termError struct {
	kind error
	text string
}

func (e *termError) Error() string { return noUncertainty + e.text }

func (e *termError) Unwrap() error { return e.kind }

func NewTable() *Table { return &Table{corr: map[Pair]float64{}} }

func (t *Table) Declare(name string, value, sigma float64, line int) InputID {
	id := InputID(len(t.inputs))
	t.inputs = append(t.inputs, Input{ID: id, Name: name, Value: value, Sigma: math.Abs(sigma), Line: line})
	return id
}

func (t *Table) SetCorr(a, b InputID, rho float64) error {
	if !t.valid(a) || !t.valid(b) {
		return errors.New("corr needs two declared measurements")
	}
	if a == b {
		return errors.New("a measurement is always fully correlated with itself")
	}
	if math.IsNaN(rho) || rho < -1 || rho > 1 {
		return errors.New("a correlation coefficient lies between -1 and 1")
	}
	if t.corr == nil {
		t.corr = map[Pair]float64{}
	}
	t.corr[key(a, b)] = rho
	return nil
}

func (t *Table) Correlated(s Sens) bool { return t.correlatedAmong(ids(s)) }

func (t *Table) correlatedAmong(order []InputID) bool {
	for i, a := range order {
		for _, b := range order[i+1:] {
			if t.corr[key(a, b)] != 0 {
				return true
			}
		}
	}
	return false
}

func involved(s Sens, h Curv) []InputID {
	set := make(map[InputID]bool, len(s)+2*len(h))
	for id := range s {
		set[id] = true
	}
	for p := range h {
		set[p.A] = true
		set[p.B] = true
	}
	return slices.Sorted(maps.Keys(set))
}

func (t *Table) Variance(s Sens) float64 {
	order := ids(s)
	total := 0.0
	for _, i := range order {
		for _, j := range order {
			total += s[i] * s[j] * t.rho(i, j) * t.sigma(i) * t.sigma(j)
		}
	}
	return total
}

func (t *Table) Combined(s Sens) float64 {
	if len(s) == 0 {
		return 0
	}
	if err := t.Check(s); err != nil {
		return math.NaN()
	}
	v := t.Variance(s)
	if v <= 0 {
		return 0
	}
	return math.Sqrt(v)
}

func (t *Table) Check(s Sens) error {
	if len(s) == 0 {
		return nil
	}
	for _, id := range ids(s) {
		if err := finite(s[id], firstOrder); err != nil {
			return err
		}
		if err := finite(t.sigma(id), declaredSigma); err != nil {
			return err
		}
	}
	v := t.Variance(s)
	if err := finite(v, combinedVar); err != nil {
		return err
	}
	if v >= 0 {
		return nil
	}
	scale := 0.0
	for id, c := range s {
		scale += (c * t.sigma(id)) * (c * t.sigma(id))
	}
	if v > -1e-12*scale {
		return nil
	}
	return ErrInconsistentCorrelation
}

func (t *Table) CheckCurv(h Curv) error {
	for _, p := range slices.SortedFunc(maps.Keys(h), func(a, b Pair) int {
		if a.A != b.A {
			return int(a.A - b.A)
		}
		return int(a.B - b.B)
	}) {
		if err := finite(h[p], secondOrder); err != nil {
			return err
		}
	}
	return nil
}

func finite(x float64, term string) error {
	switch {
	case math.IsNaN(x):
		return &termError{ErrSensitivityUndefined, term + notANum}
	case math.IsInf(x, 0):
		return &termError{ErrSensitivityOverflow, term + tooLarge}
	}
	return nil
}

func (t *Table) valid(id InputID) bool { return id >= 0 && int(id) < len(t.inputs) }

func (t *Table) sigma(id InputID) float64 {
	if !t.valid(id) {
		return 0
	}
	return t.inputs[id].Sigma
}

func (t *Table) name(id InputID) string {
	if !t.valid(id) || t.inputs[id].Name == "" {
		return "unnamed input"
	}
	return t.inputs[id].Name
}

func (t *Table) rho(a, b InputID) float64 {
	if a == b {
		return 1
	}
	return t.corr[key(a, b)]
}

func key(a, b InputID) Pair {
	if a > b {
		a, b = b, a
	}
	return Pair{A: a, B: b}
}

func ids(s Sens) []InputID { return slices.Sorted(maps.Keys(s)) }

func Lin(a Sens, ca float64, b Sens, cb float64) Sens {
	out := Sens{}
	for i, v := range a {
		out[i] += ca * v
	}
	for i, v := range b {
		out[i] += cb * v
	}
	for i, v := range out {
		if v == 0 {
			delete(out, i)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func Scale(a Sens, k float64) Sens { return Lin(a, k, nil, 0) }

func Elementary(s Sens) (InputID, bool) {
	if len(s) != 1 {
		return 0, false
	}
	for id, c := range s {
		if c == 1 {
			return id, true
		}
	}
	return 0, false
}

func Compose(a Jet, fa, faa float64, b Jet, fb, fbb, fab float64) Jet {
	grad := Lin(a.Grad, fa, b.Grad, fb)
	hess := Curv{}
	addCurv(hess, a.Hess, fa)
	addCurv(hess, b.Hess, fb)
	addHess(hess, a.Grad, b.Grad, faa, fab, fbb)
	for p, v := range hess {
		if v == 0 {
			delete(hess, p)
		}
	}
	if len(hess) == 0 {
		hess = nil
	}
	if len(grad) == 0 && hess == nil {
		return Jet{}
	}
	return Jet{Grad: grad, Hess: hess}
}

func Unary(a Jet, d1, d2 float64) Jet { return Compose(a, d1, d2, Jet{}, 0, 0, 0) }

func addCurv(dst, src Curv, k float64) {
	if k == 0 {
		return
	}
	for p, v := range src {
		dst[p] += k * v
	}
}

func addHess(dst Curv, ga, gb Sens, faa, fab, fbb float64) {
	if faa == 0 && fab == 0 && fbb == 0 {
		return
	}
	union := make(map[InputID]bool, len(ga)+len(gb))
	for id := range ga {
		union[id] = true
	}
	for id := range gb {
		union[id] = true
	}
	order := slices.Sorted(maps.Keys(union))
	for x, i := range order {
		for _, j := range order[x:] {
			term := faa*ga[i]*ga[j] + fab*(ga[i]*gb[j]+ga[j]*gb[i]) + fbb*gb[i]*gb[j]
			if term != 0 {
				dst[Pair{A: i, B: j}] += term
			}
		}
	}
}

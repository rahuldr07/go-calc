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

func NewTable() *Table { return &Table{corr: map[Pair]float64{}} }

func (t *Table) Declare(name string, value, sigma float64, line int) InputID {
	id := InputID(len(t.inputs))
	t.inputs = append(t.inputs, Input{ID: id, Name: name, Value: value, Sigma: math.Abs(sigma), Line: line})
	return id
}

func (t *Table) Rename(id InputID, name string) {
	if t.valid(id) {
		t.inputs[id].Name = name
	}
}

func (t *Table) Lookup(id InputID) (Input, bool) {
	if !t.valid(id) {
		return Input{}, false
	}
	return t.inputs[id], true
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

func (t *Table) Correlated(s Sens) bool {
	ids := ids(s)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			if t.corr[key(a, b)] != 0 {
				return true
			}
		}
	}
	return false
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
	v := t.Variance(s)
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
	if len(grad) == 0 {
		return Jet{}
	}
	if len(hess) == 0 {
		hess = nil
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

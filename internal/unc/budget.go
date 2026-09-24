package unc

import (
	"math"
	"slices"
	"strconv"
	"strings"
)

type Contribution struct {
	Name        string
	Share       float64
	Sigma       float64
	Sensitivity float64
	Covariance  bool
}

const CovarianceName = "declared correlation"

type Coverage struct {
	K     float64
	Label string
}

type Report struct {
	Sigma      float64
	Budget     []Contribution
	Cover      Coverage
	Warning    string
	Degenerate bool
}

const NonlinearWarning = "first-order propagation may be unreliable here; Monte Carlo would be better"

const UncheckedWarning = "nonlinearity not checked: inputs are correlated"

func (t *Table) Contributions(s Sens) []Contribution {
	u := t.Combined(s)
	if u == 0 || math.IsNaN(u) {
		return nil
	}
	out := make([]Contribution, 0, len(s)+1)
	alone := 0.0
	for _, id := range ids(s) {
		c := s[id]
		sigma := t.sigma(id)
		term := (c * sigma) * (c * sigma)
		alone += term
		out = append(out, Contribution{
			Name:        t.name(id),
			Share:       term / (u * u),
			Sigma:       sigma,
			Sensitivity: c,
		})
	}
	slices.SortStableFunc(out, func(a, b Contribution) int {
		switch {
		case a.Share > b.Share:
			return -1
		case a.Share < b.Share:
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	if cov := t.Variance(s) - alone; math.Abs(cov) > 1e-12*u*u {
		out = append(out, Contribution{Name: CovarianceName, Share: cov / (u * u), Covariance: true})
	}
	return out
}

func (t *Table) SecondOrder(s Sens, h Curv) float64 {
	if t.Check(s) != nil {
		return math.NaN()
	}
	extra := 0.0
	for p, v := range h {
		si, sj := t.sigma(p.A), t.sigma(p.B)
		term := 0.5 * v * v * si * si * sj * sj
		if p.A != p.B {
			term *= 2
		}
		extra += term
	}
	v := t.Variance(s) + extra
	if v <= 0 {
		return 0
	}
	return math.Sqrt(v)
}

func (t *Table) Degenerate(s Sens, h Curv) bool {
	if len(h) == 0 {
		return false
	}
	if u := t.Combined(s); math.IsNaN(u) || u != 0 {
		return false
	}
	return t.SecondOrder(s, h) > 0
}

func (t *Table) NonlinearityUnchecked(s Sens, h Curv) bool {
	return len(h) > 0 && t.correlatedAmong(involved(s, h))
}

func (t *Table) Nonlinear(s Sens, h Curv) bool {
	if t.NonlinearityUnchecked(s, h) || len(h) == 0 {
		return false
	}
	u := t.Combined(s)
	second := t.SecondOrder(s, h)
	if math.IsNaN(u) || math.IsNaN(second) {
		return false
	}
	if u == 0 {
		return second > 0
	}
	return math.Abs(second-u)/u > 0.1
}

func FactorFor(p float64) (float64, bool) {
	if !(p > 0 && p < 1) {
		return 0, false
	}
	return math.Sqrt2 * math.Erfinv(p), true
}

func FactorLabel(k float64) string {
	return "k = " + trimFloat(k)
}

func ProbabilityLabel(p, k float64) string {
	return trimFloat(p*100) + "%, k = " + trimFloat(k)
}

func trimFloat(x float64) string {
	text := strconv.FormatFloat(x, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	return strings.TrimSuffix(text, ".")
}

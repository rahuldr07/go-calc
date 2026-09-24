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
}

type Coverage struct {
	K     float64
	Label string
}

type Report struct {
	Sigma   float64
	Budget  []Contribution
	Cover   Coverage
	Warning string
}

const NonlinearWarning = "first-order propagation may be unreliable here; Monte Carlo would be better"

func (t *Table) Contributions(s Sens) []Contribution {
	u := t.Combined(s)
	if u == 0 {
		return nil
	}
	out := make([]Contribution, 0, len(s))
	for _, id := range ids(s) {
		c := s[id]
		sigma := t.sigma(id)
		out = append(out, Contribution{
			Name:        t.name(id),
			Share:       (c * sigma) * (c * sigma) / (u * u),
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
	return out
}

func (t *Table) SecondOrder(s Sens, h Curv) float64 {
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

func (t *Table) Nonlinear(s Sens, h Curv) bool {
	if len(h) == 0 || t.Correlated(s) {
		return false
	}
	u := t.Combined(s)
	if u == 0 {
		return false
	}
	return math.Abs(t.SecondOrder(s, h)-u)/u > 0.1
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

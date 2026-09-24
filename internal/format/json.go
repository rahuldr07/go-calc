package format

import (
	"encoding/json"
	"math"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const Schema = "go-calc.result/1"

type unitOut struct {
	Symbol string         `json:"symbol"`
	Dims   map[string]int `json:"dims,omitempty"`
}

type valueOut struct {
	Exact   string `json:"exact,omitempty"`
	Ball    string `json:"ball,omitempty"`
	Decimal string `json:"decimal,omitempty"`
	Digits  int    `json:"digits"`
	Capped  bool   `json:"capped,omitempty"`
}

type contributionOut struct {
	Name        string  `json:"name"`
	Share       float64 `json:"share"`
	Sigma       float64 `json:"sigma"`
	Sensitivity float64 `json:"sensitivity"`
	Covariance  bool    `json:"covariance,omitempty"`
}

type uncertaintyOut struct {
	Value      string            `json:"value,omitempty"`
	Reported   string            `json:"reported,omitempty"`
	Standard   *float64          `json:"standard,omitempty"`
	Degenerate bool              `json:"degenerate,omitempty"`
	Coverage   string            `json:"coverage,omitempty"`
	Factor     float64           `json:"factor,omitempty"`
	Budget     []contributionOut `json:"budget,omitempty"`
	Warning    string            `json:"warning,omitempty"`
}

type result struct {
	Schema      string          `json:"schema"`
	Input       string          `json:"input"`
	Value       valueOut        `json:"value"`
	Unit        *unitOut        `json:"unit,omitempty"`
	Uncertainty *uncertaintyOut `json:"uncertainty,omitempty"`
	Notes       []string        `json:"notes,omitempty"`
}

var dimKeys = [8]string{"L", "M", "T", "I", "Θ", "N", "J", "B"}

func JSON(input string, v value.Value, notes []parse.Note) ([]byte, error) {
	msgs := make([]string, 0, len(notes))
	for _, n := range notes {
		msgs = append(msgs, n.Msg)
	}
	out := valueOut{}
	if v.Num.IsExact() {
		out.Exact = v.Num.Exact().String()
		out.Decimal, out.Digits = Default.provenExact(v.Num.Exact().Rat())
	} else {
		b := v.Num.Ball(ballPrec)
		_, _, out.Capped = Default.NumberBall(b)
		out.Ball = tightBounds(b)
		out.Decimal, out.Digits = Default.provenBall(b)
	}
	r := result{Schema: Schema, Input: input, Value: out, Notes: msgs, Uncertainty: uncertainty(v)}
	if v.Unit != "" {
		dims := map[string]int{}
		for i, e := range v.Dim {
			if e != 0 {
				dims[dimKeys[i]] = e
			}
		}
		r.Unit = &unitOut{Symbol: v.Unit, Dims: dims}
	}
	return json.MarshalIndent(r, "", "  ")
}

func (s Style) provenBall(b num.Ball) (string, int) {
	digits := b.GuaranteedDigits()
	if digits > s.digitBudget() {
		digits = s.digitBudget()
	}
	if digits == 0 {
		return "", 0
	}
	text, shown := provenDecimal(b, digits)
	if shown == 0 {
		return "", 0
	}
	return text, shown
}

func (s Style) provenExact(r *big.Rat) (string, int) {
	text, complete := s.Decimal(value.Scalar(num.FromRat(r)))
	if complete {
		return text, exactSignificantDigits(r)
	}
	return text, s.digitBudget()
}

func uncertainty(v value.Value) *uncertaintyOut {
	mid, reported, ok := Default.Uncertain(v)
	if !ok && v.Unc.Warning == "" {
		return nil
	}
	scale := 1.0
	if f, _ := v.FactorOrOne().Rat().Float64(); f != 0 {
		scale = math.Abs(f)
	}
	out := &uncertaintyOut{
		Value:      mid,
		Reported:   reported,
		Degenerate: v.Unc.Degenerate,
		Coverage:   v.Unc.Cover.Label,
		Factor:     v.Unc.Cover.K,
		Warning:    v.Unc.Warning,
	}
	if !v.Unc.Degenerate {
		standard := v.Unc.Sigma / scale
		out.Standard = &standard
	}
	for _, c := range v.Unc.Budget {
		out.Budget = append(out.Budget, contributionOut{
			Name:        c.Name,
			Share:       c.Share,
			Sigma:       c.Sigma,
			Sensitivity: c.Sensitivity,
			Covariance:  c.Covariance,
		})
	}
	return out
}

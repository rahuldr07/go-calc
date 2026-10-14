package format

import (
	"encoding/json"
	"math"
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/value"
)

const Schema = "go-calc.result/1"

type UnitOut struct {
	Symbol string         `json:"symbol"`
	Dims   map[string]int `json:"dims,omitempty"`
}

type PartOut struct {
	Exact   string `json:"exact,omitempty"`
	Ball    string `json:"ball,omitempty"`
	Decimal string `json:"decimal,omitempty"`
}

type ComplexOut struct {
	Real      PartOut `json:"real"`
	Imaginary PartOut `json:"imaginary"`
	Exact     bool    `json:"exact"`
	Digits    int     `json:"digits"`
	Text      string  `json:"text"`
}

type ValueOut struct {
	Exact   string `json:"exact,omitempty"`
	Ball    string `json:"ball,omitempty"`
	Decimal string `json:"decimal,omitempty"`
	Digits  int    `json:"digits"`
	Capped  bool   `json:"capped,omitempty"`
}

type Contribution struct {
	Name        string  `json:"name"`
	Share       float64 `json:"share"`
	Sigma       float64 `json:"sigma"`
	Sensitivity float64 `json:"sensitivity"`
	Covariance  bool    `json:"covariance,omitempty"`
}

type Uncertainty struct {
	Value      string         `json:"value,omitempty"`
	Reported   string         `json:"reported,omitempty"`
	Standard   *float64       `json:"standard,omitempty"`
	Degenerate bool           `json:"degenerate,omitempty"`
	Coverage   string         `json:"coverage,omitempty"`
	Factor     float64        `json:"factor,omitempty"`
	Budget     []Contribution `json:"budget,omitempty"`
	Warning    string         `json:"warning,omitempty"`
}

type ItemOut struct {
	Value       *ValueOut    `json:"value"`
	Unit        *UnitOut     `json:"unit,omitempty"`
	Uncertainty *Uncertainty `json:"uncertainty,omitempty"`
}

type ListOut struct {
	Count int       `json:"count"`
	Items []ItemOut `json:"items,omitempty"`
}

type result struct {
	Schema      string       `json:"schema"`
	Input       string       `json:"input"`
	Definition  string       `json:"definition,omitempty"`
	Truth       string       `json:"truth,omitempty"`
	Caveat      string       `json:"caveat,omitempty"`
	Value       *ValueOut    `json:"value,omitempty"`
	Unit        *UnitOut     `json:"unit,omitempty"`
	List        *ListOut     `json:"list,omitempty"`
	Uncertainty *Uncertainty `json:"uncertainty,omitempty"`
	Provenance  *prov.Export `json:"provenance,omitempty"`
	Warning     string       `json:"warning,omitempty"`
	Notes       []string     `json:"notes,omitempty"`
}

type Detail struct {
	Arena   *prov.Arena
	Warning string
}

var dimKeys = [8]string{"L", "M", "T", "I", "Θ", "N", "J", "B"}

func JSON(input string, v value.Value, notes []parse.Note) ([]byte, error) {
	return JSONDetail(input, v, notes, Detail{})
}

func JSONDetail(input string, v value.Value, notes []parse.Note, d Detail) ([]byte, error) {
	msgs := make([]string, 0, len(notes))
	for _, n := range notes {
		msgs = append(msgs, n.Msg)
	}
	if v.IsDefinition() {
		return json.MarshalIndent(result{Schema: Schema, Input: input, Definition: v.Def, Notes: msgs}, "", "  ")
	}
	if v.IsBoolean() {
		r := result{Schema: Schema, Input: input, Truth: TruthWord(v), Caveat: v.Caveat, Notes: msgs}
		if d.Arena != nil && d.Arena.Has(v.Prov) {
			trace := d.Arena.Export(v.Prov)
			r.Provenance = &trace
		}
		return json.MarshalIndent(r, "", "  ")
	}
	if v.IsList() {
		r := result{Schema: Schema, Input: input, List: Default.listOut(v.List.Items), Notes: msgs, Warning: d.Warning}
		if d.Arena != nil && d.Arena.Has(v.Prov) {
			trace := d.Arena.Export(v.Prov)
			r.Provenance = &trace
		}
		return json.MarshalIndent(r, "", "  ")
	}
	out, unit := Default.ValueDetail(v)
	r := result{Schema: Schema, Input: input, Value: out, Unit: unit, Notes: msgs, Uncertainty: UncertaintyOf(v), Warning: d.Warning}
	if d.Arena != nil && d.Arena.Has(v.Prov) {
		trace := d.Arena.Export(v.Prov)
		r.Provenance = &trace
	}
	return json.MarshalIndent(r, "", "  ")
}

func TruthWord(v value.Value) string {
	switch {
	case v.Truth == value.Likely:
		return "probably true"
	case v.Holds():
		return "true"
	}
	return "false"
}

func (s Style) listOut(items []value.Value) *ListOut {
	out := &ListOut{Count: len(items)}
	for _, item := range items {
		amount, unit := s.ValueDetail(item)
		out.Items = append(out.Items, ItemOut{Value: amount, Unit: unit, Uncertainty: UncertaintyOf(item)})
	}
	return out
}

func (s Style) ValueDetail(v value.Value) (*ValueOut, *UnitOut) {
	out := &ValueOut{}
	if v.IsList() {
		return out, nil
	}
	if v.Num.IsExact() {
		out.Exact = v.Num.Exact().String()
	} else {
		out.Ball = tightBounds(v.Num.Ball(ballPrec))
	}
	out.Decimal, out.Digits, out.Capped = s.Proven(v)
	if v.Unit == "" {
		return out, nil
	}
	dims := map[string]int{}
	for i, e := range v.Dim {
		if e != 0 {
			dims[dimKeys[i]] = e
		}
	}
	return out, &UnitOut{Symbol: v.Unit, Dims: dims}
}

func (s Style) Proven(v value.Value) (string, int, bool) {
	if v.Num.IsExact() {
		text, digits := s.provenExact(v.Num.Exact().Rat())
		return text, digits, false
	}
	b := v.Num.Ball(ballPrec)
	_, _, capped := s.NumberBall(b)
	text, digits := s.provenBall(b)
	return text, digits, capped
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

func UncertaintyOf(v value.Value) *Uncertainty {
	mid, reported, ok := Default.Uncertain(v)
	if !ok && v.Unc.Warning == "" {
		return nil
	}
	scale := 1.0
	if f, _ := v.FactorOrOne().Rat().Float64(); f != 0 {
		scale = math.Abs(f)
	}
	out := &Uncertainty{
		Value:      mid,
		Reported:   reported,
		Degenerate: v.Unc.Degenerate,
		Coverage:   v.Unc.Cover.Label,
		Factor:     v.Unc.Cover.K,
		Warning:    v.Unc.Warning,
	}
	if !v.Unc.Degenerate && (ok || v.IsMeasured()) {
		standard := v.Unc.Sigma / scale
		out.Standard = &standard
	}
	for _, c := range v.Unc.Budget {
		out.Budget = append(out.Budget, Contribution{
			Name:        c.Name,
			Share:       c.Share,
			Sigma:       c.Sigma,
			Sensitivity: c.Sensitivity,
			Covariance:  c.Covariance,
		})
	}
	return out
}

func (s Style) partOf(n num.Number) PartOut {
	out := PartOut{}
	if n.IsExact() {
		out.Exact = n.Exact().String()
	} else {
		out.Ball = tightBounds(n.Ball(ballPrec))
	}
	out.Decimal, _ = s.Decimal(value.ScalarNum(n))
	return out
}

func (s Style) ComplexDetail(v value.Value) *ComplexOut {
	if v.Cx == nil {
		return nil
	}
	digits := weakestDigits(s, v.Cx.Re(), v.Cx.Im())
	return &ComplexOut{
		Real:      s.partOf(v.Cx.Re()),
		Imaginary: s.partOf(v.Cx.Im()),
		Exact:     v.Cx.IsExact(),
		Digits:    digits,
		Text:      v.Cx.String(),
	}
}

func weakestDigits(s Style, parts ...num.Number) int {
	weakest := 0
	for _, part := range parts {
		_, digits, _ := s.Proven(value.ScalarNum(part))
		if part.IsExact() && part.Exact().Sign() == 0 {
			continue
		}
		if weakest == 0 || digits < weakest {
			weakest = digits
		}
	}
	return weakest
}

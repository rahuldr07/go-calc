package format

import (
	"encoding/json"
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

type result struct {
	Schema string   `json:"schema"`
	Input  string   `json:"input"`
	Value  valueOut `json:"value"`
	Unit   *unitOut `json:"unit,omitempty"`
	Notes  []string `json:"notes,omitempty"`
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
		out.Ball = v.Num.String()
		out.Decimal, out.Digits = Default.provenBall(b)
	}
	r := result{Schema: Schema, Input: input, Value: out, Notes: msgs}
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

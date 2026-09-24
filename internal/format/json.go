package format

import (
	"encoding/json"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const Schema = "go-calc.result/1"

type unitOut struct {
	Symbol string         `json:"symbol"`
	Dims   map[string]int `json:"dims,omitempty"`
}

type valueOut struct {
	Exact   string `json:"exact"`
	Decimal string `json:"decimal"`
	Digits  int    `json:"digits"`
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
	r := result{
		Schema: Schema,
		Input:  input,
		Value: valueOut{
			Exact:   v.Num.String(),
			Decimal: trimDecimal(v.Num.Rat().FloatString(Digits)),
			Digits:  Digits,
		},
		Notes: msgs,
	}
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

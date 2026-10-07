package doc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/value"
)

const SheetSchema = "go-calc.sheet/1"

const (
	StatusExact    = "exact"
	StatusProven   = "proven"
	StatusEstimate = "estimate"
	StatusHeld     = "held"
	StatusError    = "error"
)

const (
	FromParser    = "parser"
	FromEvaluator = "evaluator"
)

const maxDigits = 1000

type sheetRequest struct {
	Lines  []string `json:"lines"`
	Digits int      `json:"digits"`
	Angle  string   `json:"angle"`
}

type noteOut struct {
	Source string `json:"source"`
	Text   string `json:"text"`
	Column int    `json:"column,omitempty"`
}

type lineOut struct {
	N           int                 `json:"n"`
	Text        string              `json:"text"`
	Qualifiers  []string            `json:"qualifiers,omitempty"`
	Status      string              `json:"status"`
	Digits      int                 `json:"digits"`
	Capped      bool                `json:"capped,omitempty"`
	Value       *format.ValueOut    `json:"value,omitempty"`
	Unit        *format.UnitOut     `json:"unit,omitempty"`
	Stopped     string              `json:"stopped,omitempty"`
	Error       string              `json:"error,omitempty"`
	Column      int                 `json:"column,omitempty"`
	Uncertainty *format.Uncertainty `json:"uncertainty,omitempty"`
	Notes       []noteOut           `json:"notes,omitempty"`
	Warning     string              `json:"warning,omitempty"`
	Trace       []prov.NodeOut      `json:"trace,omitempty"`
}

type sheetOut struct {
	Schema string    `json:"schema"`
	Lines  []lineOut `json:"lines"`
}

func EvalRequest(req []byte) []byte { return evalSheet(req, documentUnder) }

func evalSheet(req []byte, open func(*eval.Ctx) *Doc) (out []byte) {
	defer func() {
		if r := recover(); r != nil {
			out = errorSheet(failure(r))
		}
	}()
	answer, err := sheetOf(req, open)
	if err != nil {
		return errorSheet(err)
	}
	return answer
}

func sheetOf(req []byte, open func(*eval.Ctx) *Doc) ([]byte, error) {
	var r sheetRequest
	if err := json.Unmarshal(req, &r); err != nil {
		return nil, err
	}
	ctx, err := settings(r)
	if err != nil {
		return nil, err
	}
	d := open(ctx)
	d.SetLines(r.Lines)
	return json.Marshal(sheetOut{Schema: SheetSchema, Lines: d.sheet(format.Style{Digits: ctx.Digits})})
}

func documentUnder(ctx *eval.Ctx) *Doc {
	d := New()
	d.evaluator = func(env *eval.Env, n parse.Node) (value.Value, error) {
		each := *ctx
		return env.EvalCtx(n, &each)
	}
	d.warner = func(env *eval.Env, n parse.Node, v value.Value) string {
		each := *ctx
		return env.WhyCtx(n, v, &each).Warning
	}
	return d
}

func settings(r sheetRequest) (*eval.Ctx, error) {
	ctx := eval.DefaultCtx()
	if r.Digits > 0 {
		ctx.Digits = r.Digits
	}
	if ctx.Digits > maxDigits {
		ctx.Digits = maxDigits
	}
	switch r.Angle {
	case "", "rad", "radians":
	case "deg", "degrees":
		ctx.Degrees = true
	default:
		return nil, errors.New("angles are read as rad or deg, not " + strconv.Quote(r.Angle))
	}
	return ctx, nil
}

func (d *Doc) sheet(style format.Style) []lineOut {
	out := make([]lineOut, len(d.lines))
	for i, e := range d.lines {
		out[i] = d.sheetLine(i, e, style)
	}
	return out
}

func (d *Doc) sheetLine(i int, e *entry, style format.Style) lineOut {
	out := lineOut{N: i, Notes: readings(e), Warning: e.warning}
	switch {
	case e.err != nil:
		out.Status = StatusError
		out.Error = e.err.Error()
		out.Column = columnOf(e.err)
	case Blank(e.src):
		out.Status = StatusHeld
	case e.value.IsDefinition() || e.value.IsSymbolic():
		out.Status = StatusHeld
		out.Text, out.Qualifiers = answer(style, e.value)
	default:
		out.Status = statusOf(e.value)
		out.Text, out.Qualifiers = answer(style, e.value)
		out.Value, out.Unit = style.ValueDetail(e.value)
		out.Digits, out.Capped = out.Value.Digits, out.Value.Capped
		out.Stopped = whyStopped(e.value.Stopped)
		out.Uncertainty = format.UncertaintyOf(e.value)
		out.Trace = d.arena.Export(e.value.Prov).Nodes
	}
	return out
}

func statusOf(v value.Value) string {
	switch {
	case v.IsMeasured():
		return StatusEstimate
	case v.Num.IsExact():
		return StatusExact
	}
	return StatusProven
}

func answer(style format.Style, v value.Value) (string, []string) {
	head, rest, _ := strings.Cut(style.Text(v, nil), "\n")
	if rest == "" {
		return head, nil
	}
	lines := strings.Split(rest, "\n")
	qualifiers := make([]string, 0, len(lines))
	for _, l := range lines {
		if trimmed := strings.TrimSpace(l); trimmed != "" {
			qualifiers = append(qualifiers, trimmed)
		}
	}
	if len(qualifiers) == 0 {
		return head, nil
	}
	return head, qualifiers
}

func whyStopped(s value.Stop) string {
	switch s {
	case value.StopPrecisionCap:
		return "precision cap"
	case value.StopUncertainInput:
		return "uncertain input"
	case value.StopDeadline:
		return "time limit"
	}
	return ""
}

func readings(e *entry) []noteOut {
	if len(e.notes) == 0 {
		return nil
	}
	out := make([]noteOut, 0, len(e.notes))
	for i, n := range e.notes {
		source := FromEvaluator
		if i < len(e.read) {
			source = FromParser
		}
		out = append(out, noteOut{Source: source, Text: n.Msg, Column: n.Pos.Col})
	}
	return out
}

func columnOf(err error) int {
	var pointed *lex.Error
	if errors.As(err, &pointed) {
		return pointed.Pos.Col
	}
	return 0
}

func failure(r any) error {
	if err, ok := r.(error); ok {
		return errors.New("the calculator stopped on this document: " + err.Error())
	}
	return fmt.Errorf("the calculator stopped on this document: %v", r)
}

const lastResort = `{"schema":"` + SheetSchema + `","lines":[{"n":0,"text":"","status":"` + StatusError + `","digits":0,"error":"the request could not be read"}]}`

func errorSheet(err error) []byte {
	out, marshalled := json.Marshal(sheetOut{
		Schema: SheetSchema,
		Lines:  []lineOut{{Status: StatusError, Error: err.Error()}},
	})
	if marshalled != nil {
		return []byte(lastResort)
	}
	return out
}

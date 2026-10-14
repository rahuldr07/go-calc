package steps

import (
	"encoding/json"
	"errors"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/prov"
)

const Schema = "go-calc.steps/1"

const (
	maxDepth  = 64
	maxSteps  = 200
	maxRender = 2000
)

const (
	noWorkingLiteral = "a single literal is already a value, so there is no working to show"
	noWorkingHeld    = "this line holds a definition or a symbolic form, so there is no working to show"
	noWorkingValue   = "the answer is already a value, so there is no working to show"
	cutOff           = "the working was cut off at the depth or step limit"
	incomplete       = "the provenance trace has a cycle or a missing node, so the working is incomplete"
)

type Line struct {
	Text       string         `json:"text"`
	Status     string         `json:"status"`
	Digits     int            `json:"digits"`
	Error      string         `json:"error"`
	Qualifiers []string       `json:"qualifiers"`
	Value      *Bound         `json:"value"`
	Trace      []prov.NodeOut `json:"trace"`
}

type Bound struct {
	Ball string `json:"ball"`
}

type Step struct {
	N       int      `json:"n"`
	Explain string   `json:"explain,omitempty"`
	Expr    string   `json:"expr"`
	From    []string `json:"from,omitempty"`
	Result  string   `json:"result"`
	Rule    string   `json:"rule,omitempty"`
	Source  string   `json:"source,omitempty"`
}

type Derivation struct {
	Schema     string   `json:"schema"`
	Input      string   `json:"input"`
	Steps      []Step   `json:"steps"`
	Answer     string   `json:"answer,omitempty"`
	Status     string   `json:"status,omitempty"`
	Digits     int      `json:"digits,omitempty"`
	Bound      string   `json:"bound,omitempty"`
	Qualifiers []string `json:"qualifiers,omitempty"`
	Note       string   `json:"note,omitempty"`
	Summary    []string `json:"summary,omitempty"`
	Truncated  bool     `json:"truncated,omitempty"`
	Error      string   `json:"error,omitempty"`
}

func Decode(raw []byte) (Line, error) {
	var sheet struct {
		Lines []Line `json:"lines"`
	}
	if err := json.Unmarshal(raw, &sheet); err != nil {
		return Line{}, err
	}
	if len(sheet.Lines) == 0 {
		return Line{}, errors.New("the sheet has no lines")
	}
	return sheet.Lines[0], nil
}

func Of(input string, digits int) (Derivation, error) {
	req, err := json.Marshal(struct {
		Lines  []string `json:"lines"`
		Digits int      `json:"digits"`
	}{Lines: []string{input}, Digits: digits})
	if err != nil {
		return Derivation{}, err
	}
	line, err := Decode(doc.EvalRequest(req))
	if err != nil {
		return Derivation{}, err
	}
	return Derive(input, line), nil
}

func Derive(input string, line Line) Derivation {
	out := Derivation{Schema: Schema, Input: input, Steps: []Step{}}
	if line.Status == doc.StatusError {
		out.Error = line.Error
		return out
	}
	out.Answer, out.Status, out.Digits, out.Qualifiers = line.Text, line.Status, line.Digits, line.Qualifiers
	if line.Value != nil {
		out.Bound = line.Value.Ball
	}
	if len(line.Trace) == 0 {
		out.Note = noWorkingHeld
		return out
	}
	w := newWalker(input, line)
	w.visit(line.Trace[0].ID, 0)
	out.Steps, out.Truncated = w.steps, w.truncated
	out.Summary = summaryOf(line, w)
	switch {
	case w.truncated:
		out.Note = cutOff
	case w.broken:
		out.Note = incomplete
	case len(w.steps) == 0:
		out.Note = idleNote(line.Trace[0])
	}
	return out
}

func idleNote(root prov.NodeOut) string {
	if root.Kind == kindLiteral {
		return noWorkingLiteral
	}
	return noWorkingValue
}

func FromLine(input string, raw []byte) ([]byte, error) {
	var line Line
	if err := json.Unmarshal(raw, &line); err != nil {
		return nil, err
	}
	return json.Marshal(Derive(input, line))
}

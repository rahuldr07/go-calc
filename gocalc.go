package gocalc

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/rahuldr07/go-calc/internal/doc"
)

type Status string

const (
	Exact    Status = doc.StatusExact
	Proven   Status = doc.StatusProven
	Estimate Status = doc.StatusEstimate
	Held     Status = doc.StatusHeld
	Refused  Status = doc.StatusError
)

const (
	DefaultDigits  = 12
	MaxDigits      = 1000
	DefaultTimeout = 10 * time.Second
)

type Result struct {
	Line        int
	Source      string
	Text        string
	Status      Status
	Digits      int
	Exact       string
	Bounds      string
	Decimal     string
	Unit        string
	Uncertainty string
	Float64     string
	Qualifiers  []string
	Notes       []string
	Warning     string
	Refusal     *Refusal
}

func (r Result) Trusted() bool { return r.Status == Exact || r.Status == Proven }

type Refusal struct {
	Line   int
	Column int
	Reason string
}

func (r *Refusal) Error() string {
	where := "line " + strconv.Itoa(r.Line)
	if r.Column > 0 {
		where += ", column " + strconv.Itoa(r.Column)
	}
	return where + ": " + r.Reason
}

type Sheet struct {
	Results []Result
}

func (s Sheet) Refusals() []*Refusal {
	var out []*Refusal
	for _, r := range s.Results {
		if r.Refusal != nil {
			out = append(out, r.Refusal)
		}
	}
	return out
}

type Option func(*settings)

type settings struct {
	digits  int
	degrees bool
	timeout time.Duration
}

func Digits(n int) Option { return func(s *settings) { s.digits = n } }

func Degrees() Option { return func(s *settings) { s.degrees = true } }

func Timeout(d time.Duration) Option { return func(s *settings) { s.timeout = d } }

var (
	ErrNoLines   = errors.New("gocalc: there is nothing to evaluate")
	ErrBadDigits = errors.New("gocalc: digits must be between 1 and " + strconv.Itoa(MaxDigits))
)

func Eval(ctx context.Context, expression string, opts ...Option) (Result, error) {
	sheet, err := EvalSheet(ctx, []string{expression}, opts...)
	if err != nil {
		return Result{}, err
	}
	if len(sheet.Results) == 0 {
		return Result{}, ErrNoLines
	}
	r := sheet.Results[0]
	if r.Refusal != nil {
		return r, r.Refusal
	}
	return r, nil
}

func EvalSheet(ctx context.Context, lines []string, opts ...Option) (Sheet, error) {
	if len(lines) == 0 {
		return Sheet{}, ErrNoLines
	}
	set := settings{digits: DefaultDigits, timeout: DefaultTimeout}
	for _, opt := range opts {
		opt(&set)
	}
	if set.digits < 1 || set.digits > MaxDigits {
		return Sheet{}, ErrBadDigits
	}
	if err := ctx.Err(); err != nil {
		return Sheet{}, err
	}
	req, err := json.Marshal(request{Lines: lines, Digits: set.digits, Angle: angleOf(set.degrees)})
	if err != nil {
		return Sheet{}, err
	}
	deadline := time.Now().Add(set.timeout)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	done := make(chan []byte, 1)
	go func() { done <- doc.EvalRequestBefore(req, deadline) }()
	select {
	case <-ctx.Done():
		return Sheet{}, ctx.Err()
	case raw := <-done:
		return decode(raw, lines)
	}
}

type request struct {
	Lines  []string `json:"lines"`
	Digits int      `json:"digits"`
	Angle  string   `json:"angle"`
}

func angleOf(degrees bool) string {
	if degrees {
		return "deg"
	}
	return "rad"
}

type wireSheet struct {
	Lines []wireLine `json:"lines"`
}

type wireLine struct {
	N          int      `json:"n"`
	Text       string   `json:"text"`
	Qualifiers []string `json:"qualifiers"`
	Status     string   `json:"status"`
	Digits     int      `json:"digits"`
	Value      *struct {
		Exact   string `json:"exact"`
		Ball    string `json:"ball"`
		Decimal string `json:"decimal"`
	} `json:"value"`
	Unit *struct {
		Symbol string `json:"symbol"`
	} `json:"unit"`
	Uncertainty *struct {
		Reported string `json:"reported"`
	} `json:"uncertainty"`
	Float64 *struct {
		Value string `json:"value"`
	} `json:"float64"`
	Notes []struct {
		Text string `json:"text"`
	} `json:"notes"`
	Warning string `json:"warning"`
	Error   string `json:"error"`
	Column  int    `json:"column"`
}

var located = regexp.MustCompile(`^\d+:\d+:\s*`)

func decode(raw []byte, lines []string) (Sheet, error) {
	var w wireSheet
	if err := json.Unmarshal(raw, &w); err != nil {
		return Sheet{}, err
	}
	out := Sheet{Results: make([]Result, 0, len(w.Lines))}
	for _, l := range w.Lines {
		out.Results = append(out.Results, result(l, lines))
	}
	return out, nil
}

func result(l wireLine, lines []string) Result {
	r := Result{
		Line:       l.N + 1,
		Text:       l.Text,
		Status:     Status(l.Status),
		Digits:     l.Digits,
		Qualifiers: l.Qualifiers,
		Warning:    l.Warning,
	}
	if l.N >= 0 && l.N < len(lines) {
		r.Source = lines[l.N]
	}
	if l.Value != nil {
		r.Exact, r.Bounds, r.Decimal = l.Value.Exact, l.Value.Ball, l.Value.Decimal
	}
	if l.Unit != nil {
		r.Unit = l.Unit.Symbol
	}
	if l.Uncertainty != nil {
		r.Uncertainty = l.Uncertainty.Reported
	}
	if l.Float64 != nil {
		r.Float64 = l.Float64.Value
	}
	for _, n := range l.Notes {
		r.Notes = append(r.Notes, n.Text)
	}
	if r.Status == Refused {
		r.Refusal = &Refusal{Line: r.Line, Column: l.Column, Reason: located.ReplaceAllString(l.Error, "")}
	}
	return r
}

package eval

import (
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/value"
)

type fn struct {
	params []string
	body   parse.Node
}

type Env struct {
	vars  map[string]value.Value
	funcs map[string]fn
	table *unc.Table
	line  int
	notes []parse.Note
}

func NewEnv() *Env { return NewEnvWith(unc.NewTable()) }

func NewEnvWith(t *unc.Table) *Env {
	if t == nil {
		t = unc.NewTable()
	}
	return &Env{vars: map[string]value.Value{}, funcs: map[string]fn{}, table: t}
}

func (e *Env) Table() *unc.Table { return e.table }

func (e *Env) SetLine(n int) { e.line = n }

func (e *Env) Notes() []parse.Note { return e.notes }

func distinct(notes []parse.Note) []parse.Note {
	if len(notes) < 2 {
		return notes
	}
	seen := make(map[parse.Note]bool, len(notes))
	out := notes[:0]
	for _, n := range notes {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func (e *Env) Uncertainty(v value.Value) float64 { return e.table.Combined(v.Sens) }

func (e *Env) Contributions(v value.Value) []unc.Contribution {
	return e.table.Contributions(v.Sens)
}

func (e *Env) Set(name string, v value.Value) { e.vars[name] = v }

func (e *Env) Get(name string) (value.Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Env) child() *Env {
	c := NewEnvWith(e.table)
	c.line = e.line
	for k, v := range e.vars {
		c.vars[k] = v
	}
	for k, v := range e.funcs {
		c.funcs[k] = v
	}
	return c
}

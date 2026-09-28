package eval

import (
	"sort"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
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
	arena *prov.Arena
	bound *boundArgs
	line  int
	notes []parse.Note
}

type boundArgs struct {
	measured map[*parse.Call][][]bool
}

func (b *boundArgs) clear() { b.measured = map[*parse.Call][][]bool{} }

func (b *boundArgs) record(t *parse.Call, measured []bool) {
	b.measured[t] = append(b.measured[t], measured)
}

func (b *boundArgs) at(t *parse.Call, call, i int) (bool, bool) {
	calls, ok := b.measured[t]
	if !ok || len(calls) == 0 {
		return false, false
	}
	if call >= len(calls) {
		return false, false
	}
	m := calls[call]
	if i >= len(m) {
		return false, false
	}
	return m[i], true
}

func NewEnv() *Env { return NewEnvWith(unc.NewTable()) }

func NewEnvWith(t *unc.Table) *Env {
	if t == nil {
		t = unc.NewTable()
	}
	return &Env{vars: map[string]value.Value{}, funcs: map[string]fn{}, table: t, arena: prov.NewArena(), bound: &boundArgs{measured: map[*parse.Call][][]bool{}}}
}

func (e *Env) Arena() *prov.Arena { return e.arena }

func (e *Env) SetArena(a *prov.Arena) {
	if a != nil {
		e.arena = a
	}
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

func (e *Env) Unset(name string) bool {
	_, wasVar := e.vars[name]
	_, wasFunc := e.funcs[name]
	delete(e.vars, name)
	delete(e.funcs, name)
	return wasVar || wasFunc
}

func (e *Env) Get(name string) (value.Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Env) child() *Env {
	c := NewEnvWith(e.table)
	c.arena = e.arena
	c.bound = e.bound
	c.line = e.line
	for k, v := range e.vars {
		c.vars[k] = v
	}
	for k, v := range e.funcs {
		c.funcs[k] = v
	}
	return c
}

func (e *Env) Names() []string {
	out := make([]string, 0, len(e.vars)+len(e.funcs))
	for name := range e.vars {
		out = append(out, name)
	}
	for name := range e.funcs {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

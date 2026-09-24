package eval

import (
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

type fn struct {
	params []string
	body   parse.Node
}

type Env struct {
	vars  map[string]value.Value
	funcs map[string]fn
}

func NewEnv() *Env {
	return &Env{vars: map[string]value.Value{}, funcs: map[string]fn{}}
}

func (e *Env) Set(name string, v value.Value) { e.vars[name] = v }

func (e *Env) Get(name string) (value.Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Env) child() *Env {
	c := NewEnv()
	for k, v := range e.vars {
		c.vars[k] = v
	}
	for k, v := range e.funcs {
		c.funcs[k] = v
	}
	return c
}

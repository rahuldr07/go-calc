package eval

import (
	"maps"
	"slices"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/suggest"
	"github.com/rahuldr07/go-calc/internal/units"
)

var switchedFunctions = []string{"abs", "sqrt", "root", "atan2", "expand", "floor", "ceil", "round", "gcd", "lcm", "isprime"}

var dispatchedFunctions = []string{"solve", "deriv"}

var builtinConstants = []string{"pi", "π", "e"}

func Functions() []string {
	return suggest.Sorted(
		switchedFunctions,
		dispatchedFunctions,
		slices.Collect(maps.Keys(radianIn)),
		slices.Collect(maps.Keys(radianOut)),
		slices.Collect(maps.Keys(plainElem)),
	)
}

func Constants() []string {
	return suggest.Sorted(consts.Symbols(), builtinConstants)
}

func Units() []string {
	return units.Symbols()
}

func (e *Env) knowsFunction(name string) bool {
	if _, ok := e.funcs[name]; ok {
		return true
	}
	return slices.Contains(Functions(), name)
}

func (e *Env) unknownFunction(t *parse.Call) error {
	candidates := suggest.Sorted(Functions(), slices.Collect(maps.Keys(e.funcs)))
	return &lex.Error{Pos: t.P, Msg: suggest.Did("unknown function "+t.Fn, t.Fn, candidates)}
}

func (e *Env) nameCandidates() []string {
	return suggest.Sorted(slices.Collect(maps.Keys(e.vars)), Constants(), Units())
}

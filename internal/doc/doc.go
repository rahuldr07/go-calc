package doc

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/unc"
	"github.com/rahuldr07/go-calc/internal/value"
)

var errCycle = errors.New("this line depends on itself")

type Line struct {
	N       int
	Src     string
	Value   value.Value
	Notes   []parse.Note
	Err     error
	Warning string
}

type entry struct {
	src      string
	node     parse.Node
	notes    []parse.Note
	parseErr error
	def      string
	params   []string
	corr     []string
	refs     []string
	units    []string
	read     []parse.Note
	deps     []int
	fwd      []string
	cyclic   bool
	value    value.Value
	err      error
	warning  string
}

type Doc struct {
	lines     []*entry
	defs      map[string][]int
	table     *unc.Table
	arena     *prov.Arena
	evaluator func(*eval.Env, parse.Node) (value.Value, error)
	warner    func(*eval.Env, parse.Node, value.Value) string
}

func New() *Doc {
	return &Doc{defs: map[string][]int{}, table: unc.NewTable(), arena: prov.NewArena(), evaluator: evalNode, warner: warnNode}
}

func (d *Doc) Arena() *prov.Arena { return d.arena }

func evalNode(env *eval.Env, n parse.Node) (value.Value, error) { return env.Eval(n) }

func warnNode(env *eval.Env, n parse.Node, v value.Value) string { return env.Warning(n, v) }

func (d *Doc) SetLines(src []string) []Line {
	d.lines = make([]*entry, len(src))
	for i, s := range src {
		d.lines[i] = read(s)
	}
	d.link()
	for _, i := range d.schedule() {
		d.evaluate(i)
	}
	return d.Lines()
}

func (d *Doc) Edit(n int, src string) []Line {
	if n < 0 || n >= len(d.lines) {
		return nil
	}
	was := make([][]int, len(d.lines))
	wasCyclic := make([]bool, len(d.lines))
	for i, e := range d.lines {
		was[i], wasCyclic[i] = e.deps, e.cyclic
	}
	d.lines[n] = read(src)
	d.link()
	order := d.schedule()
	stale := map[int]bool{n: true}
	for i, e := range d.lines {
		if !sameInts(was[i], e.deps) || wasCyclic[i] != e.cyclic {
			stale[i] = true
		}
	}
	for _, i := range order {
		if stale[i] {
			continue
		}
		for _, j := range d.lines[i].deps {
			if stale[j] {
				stale[i] = true
				break
			}
		}
	}
	for _, i := range order {
		if stale[i] {
			d.evaluate(i)
		}
	}
	changed := make([]Line, 0, len(stale))
	for i := range d.lines {
		if stale[i] {
			changed = append(changed, d.line(i))
		}
	}
	return changed
}

func (d *Doc) Lines() []Line {
	out := make([]Line, len(d.lines))
	for i := range d.lines {
		out[i] = d.line(i)
	}
	return out
}

func (d *Doc) line(i int) Line {
	e := d.lines[i]
	return Line{N: i, Src: e.src, Value: e.value, Notes: e.notes, Err: e.err, Warning: e.warning}
}

func read(src string) *entry {
	e := &entry{src: src}
	if Blank(src) {
		return e
	}
	node, notes, err := parse.Parse(src)
	if err != nil {
		e.parseErr = err
		return e
	}
	e.node, e.notes, e.read = node, notes, notes
	if a, ok := node.(*parse.Assign); ok {
		if a.Name == "corr" && len(a.Params) == 2 {
			e.corr = a.Params
		} else {
			e.def, e.params = a.Name, a.Params
		}
	}
	worn := eval.UnitPositions(node)
	e.refs = append(freeNames(node, nil, worn, nil), e.corr...)
	e.units = unitNames(node, nil, worn, nil)
	return e
}

func (d *Doc) evaluate(i int) {
	e := d.lines[i]
	e.value, e.err, e.warning, e.notes = value.Value{}, nil, "", e.read
	broken := d.brokenInputs(i)
	switch {
	case e.node == nil:
		e.err = e.parseErr
	case e.cyclic:
		e.err = errCycle
	case len(e.fwd) > 0:
		e.err = usedBeforeDefined(e.fwd)
	case broken != nil:
		e.err = broken
	default:
		env := d.env(i)
		e.value, e.err = d.evaluator(env, e.node)
		e.notes = append(append([]parse.Note{}, e.read...), env.Notes()...)
		if e.err == nil {
			e.warning = d.warner(env, e.node, e.value)
		}
	}
}

func (d *Doc) brokenInputs(i int) error {
	var names []string
	var at []int
	for _, j := range d.lines[i].deps {
		if j == i || d.lines[j].err == nil {
			continue
		}
		names = append(names, d.lines[j].def)
		at = append(at, j+1)
	}
	return dependsOnFailure(names, at)
}

func (d *Doc) env(i int) *eval.Env {
	env := eval.NewEnvWith(d.table)
	env.SetArena(d.arena)
	env.SetLine(i + 1)
	for _, j := range d.lines[i].deps {
		bind(env, d.lines[j])
	}
	return env
}

func bind(env *eval.Env, e *entry) {
	if e.node == nil || e.err != nil {
		return
	}
	if len(e.params) > 0 || len(e.corr) == 2 {
		_, _ = env.Eval(e.node)
		return
	}
	if e.def != "" {
		env.Set(e.def, e.value)
	}
}

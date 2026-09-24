package doc

import (
	"errors"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/parse"
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
	refs     []string
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
	evaluator func(*eval.Env, parse.Node) (value.Value, error)
}

func New() *Doc {
	return &Doc{defs: map[string][]int{}, evaluator: evalNode}
}

func evalNode(env *eval.Env, n parse.Node) (value.Value, error) { return env.Eval(n) }

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
	e.node, e.notes = node, notes
	if a, ok := node.(*parse.Assign); ok {
		e.def, e.params = a.Name, a.Params
	}
	e.refs = freeNames(node, nil, nil)
	return e
}

func (d *Doc) evaluate(i int) {
	e := d.lines[i]
	e.value, e.err, e.warning = value.Value{}, nil, ""
	switch {
	case e.node == nil:
		e.err = e.parseErr
	case e.cyclic:
		e.err = errCycle
	case len(e.fwd) > 0:
		e.err = usedBeforeDefined(e.fwd)
	default:
		e.value, e.err = d.evaluator(d.env(i), e.node)
	}
}

func (d *Doc) env(i int) *eval.Env {
	env := eval.NewEnv()
	for _, j := range d.lines[i].deps {
		bind(env, d.lines[j])
	}
	return env
}

func bind(env *eval.Env, e *entry) {
	if e.node == nil || e.err != nil {
		return
	}
	if len(e.params) > 0 {
		_, _ = env.Eval(e.node)
		return
	}
	if e.def != "" {
		env.Set(e.def, e.value)
	}
}

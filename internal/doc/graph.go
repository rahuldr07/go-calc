package doc

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
)

func Blank(src string) bool {
	trimmed := strings.TrimSpace(src)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

func freeNames(n parse.Node, bound map[string]bool, worn map[parse.Node]bool, out []string) []string {
	switch t := n.(type) {
	case *parse.Name:
		if !bound[t.Text] && !wornAsUnit(t, worn) {
			out = append(out, t.Text)
		}
	case *parse.Unary:
		out = freeNames(t.X, bound, worn, out)
	case *parse.Binary:
		out = freeNames(t.L, bound, worn, out)
		if !unitOperand(t) {
			out = freeNames(t.R, bound, worn, out)
		}
	case *parse.Call:
		if !bound[t.Fn] {
			out = append(out, t.Fn)
		}
		for _, a := range t.Args {
			out = freeNames(a, bound, worn, out)
		}
	case *parse.Assign:
		out = freeNames(t.Body, withParams(bound, t.Params), worn, out)
	}
	return out
}

func wornAsUnit(t *parse.Name, worn map[parse.Node]bool) bool {
	if !worn[t] {
		return false
	}
	_, known := units.Lookup(t.Text)
	return known
}

func withParams(bound map[string]bool, params []string) map[string]bool {
	if len(params) == 0 {
		return bound
	}
	inner := make(map[string]bool, len(bound)+len(params))
	for name := range bound {
		inner[name] = true
	}
	for _, p := range params {
		inner[p] = true
	}
	return inner
}

func unitOperand(t *parse.Binary) bool {
	if t.Op == "to" || t.Op == "in" {
		return true
	}
	name, ok := t.R.(*parse.Name)
	if !ok || !t.Implicit {
		return false
	}
	_, ok = units.Lookup(name.Text)
	return ok
}

func (d *Doc) link() {
	d.defs = map[string][]int{}
	for i, e := range d.lines {
		if e.def != "" {
			d.defs[e.def] = append(d.defs[e.def], i)
		}
	}
	for i, e := range d.lines {
		e.deps, e.fwd = d.depsOf(i)
	}
}

func (d *Doc) resolve(name string, from int) (int, bool) {
	where, ok := d.defs[name]
	if !ok {
		return 0, false
	}
	best, found := 0, false
	for _, i := range where {
		if i < from {
			best, found = i, true
		}
	}
	if found {
		return best, true
	}
	return where[0], true
}

func (d *Doc) depsOf(from int) ([]int, []string) {
	if len(d.lines[from].params) > 0 {
		return nil, nil
	}
	reached := map[int]bool{}
	settled := map[string]bool{}
	open := map[int]bool{}
	var fwd []string
	var expand func([]string)
	expand = func(names []string) {
		for _, name := range names {
			at, ok := d.resolve(name, from)
			if !ok {
				settled[name] = true
				continue
			}
			if len(d.lines[at].params) > 0 && open[at] {
				reached[from] = true
				continue
			}
			if settled[name] {
				continue
			}
			settled[name] = true
			reached[at] = true
			if at > from {
				fwd = append(fwd, name)
			}
			if len(d.lines[at].params) > 0 {
				open[at] = true
				expand(d.lines[at].refs)
				delete(open, at)
			}
		}
	}
	expand(d.lines[from].refs)
	if at, ok := d.previousAnswer(from); ok {
		reached[at] = true
	}
	for _, name := range d.lines[from].units {
		if at, ok := d.resolve(name, from); ok && at < from {
			expand([]string{name})
		}
	}
	if len(d.lines[from].corr) == 0 {
		for j := 0; j < from; j++ {
			c := d.lines[j].corr
			if len(c) == 2 && (settled[c[0]] || settled[c[1]]) {
				reached[j] = true
			}
		}
	}
	deps := make([]int, 0, len(reached))
	for i := range reached {
		deps = append(deps, i)
	}
	sortInts(deps)
	return deps, fwd
}

const AnswerName = "ans"

func (d *Doc) previousAnswer(from int) (int, bool) {
	if _, defined := d.defs[AnswerName]; defined || !refersTo(d.lines[from].refs, AnswerName) {
		return 0, false
	}
	for j := from - 1; j >= 0; j-- {
		e := d.lines[j]
		if Blank(e.src) || len(e.params) > 0 || len(e.corr) == 2 {
			continue
		}
		return j, true
	}
	return 0, false
}

func refersTo(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func (d *Doc) schedule() []int {
	const (
		fresh = iota
		open
		settled
	)
	state := make([]int, len(d.lines))
	var stack []int
	order := make([]int, 0, len(d.lines))
	var visit func(int)
	visit = func(i int) {
		switch state[i] {
		case settled:
			return
		case open:
			for k := len(stack) - 1; k >= 0; k-- {
				d.lines[stack[k]].cyclic = true
				if stack[k] == i {
					return
				}
			}
			return
		}
		state[i] = open
		stack = append(stack, i)
		for _, j := range d.lines[i].deps {
			visit(j)
		}
		stack = stack[:len(stack)-1]
		state[i] = settled
		order = append(order, i)
	}
	for i := range d.lines {
		d.lines[i].cyclic = false
	}
	for i := range d.lines {
		visit(i)
	}
	return order
}

func sortInts(xs []int) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *entry) label(n int) string {
	switch {
	case e.def != "":
		return e.def
	case len(e.corr) == 2:
		return "corr(" + strings.Join(e.corr, ", ") + ")"
	}
	return AnswerName
}

func usedBeforeDefined(names []string) error {
	switch len(names) {
	case 0:
		return nil
	case 1:
		return errors.New("uses " + names[0] + " before it is defined")
	}
	return errors.New("uses " + strings.Join(names, ", ") + " before they are defined")
}

func dependsOnFailure(names []string, at []int) error {
	switch len(names) {
	case 0:
		return nil
	case 1:
		return errors.New("uses " + names[0] + ", which failed on line " + strconv.Itoa(at[0]))
	}
	return errors.New("uses " + strings.Join(names, ", ") + ", which failed on lines " + joinInts(at))
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return strings.Join(parts, ", ")
}

func unitNames(n parse.Node, bound map[string]bool, worn map[parse.Node]bool, out []string) []string {
	switch t := n.(type) {
	case *parse.Name:
		if !bound[t.Text] && wornAsUnit(t, worn) {
			out = append(out, t.Text)
		}
	case *parse.Unary:
		out = unitNames(t.X, bound, worn, out)
	case *parse.Binary:
		out = unitNames(t.L, bound, worn, out)
		if !unitOperand(t) {
			out = unitNames(t.R, bound, worn, out)
			break
		}
		if name, ok := t.R.(*parse.Name); ok && t.Implicit && !bound[name.Text] {
			out = append(out, name.Text)
		}
	case *parse.Call:
		for _, a := range t.Args {
			out = unitNames(a, bound, worn, out)
		}
	case *parse.Assign:
		out = unitNames(t.Body, withParams(bound, t.Params), worn, out)
	}
	return out
}

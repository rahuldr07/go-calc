package steps

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/prov"
)

const (
	kindLiteral  = "literal"
	kindConstant = "constant"
	kindUnit     = "unit"
	kindFunction = "function"

	solveLabel       = "solve"
	conversionPrefix = "to "
	quadraticRule    = "quadratic formula"
)

type walker struct {
	line      Line
	input     string
	nodes     map[prov.NodeID]prov.NodeOut
	done      map[prov.NodeID]bool
	active    map[prov.NodeID]bool
	steps     []Step
	truncated bool
	broken    bool
	rootID    prov.NodeID
}

func newWalker(input string, line Line) *walker {
	nodes := make(map[prov.NodeID]prov.NodeOut, len(line.Trace))
	for _, n := range line.Trace {
		if _, seen := nodes[n.ID]; !seen {
			nodes[n.ID] = n
		}
	}
	return &walker{
		line:   line,
		input:  input,
		nodes:  nodes,
		done:   map[prov.NodeID]bool{},
		active: map[prov.NodeID]bool{},
		steps:  []Step{},
		rootID: line.Trace[0].ID,
	}
}

func (w *walker) visit(id prov.NodeID, depth int) {
	n, ok := w.nodes[id]
	if !ok {
		w.broken = true
		return
	}
	if w.active[id] {
		w.broken = true
		return
	}
	if w.done[id] {
		return
	}
	if depth > maxDepth {
		w.truncated = true
		return
	}
	w.active[id] = true
	for _, child := range n.Children {
		w.visit(child, depth+1)
	}
	delete(w.active, id)
	w.done[id] = true
	step, worked := w.stepOf(n)
	if !worked {
		return
	}
	if len(w.steps) >= maxSteps {
		w.truncated = true
		return
	}
	step.N = len(w.steps) + 1
	w.steps = append(w.steps, step)
}

func (w *walker) stepOf(n prov.NodeOut) (Step, bool) {
	switch n.Kind {
	case kindConstant:
		return Step{Explain: w.explain(n), Expr: n.Label, Result: n.Value, Source: n.Detail}, true
	case kindVariable:
		return Step{Explain: w.explain(n), Expr: n.Label, Result: n.Value}, true
	case kindUnit:
		if len(n.Children) == 0 {
			return Step{}, false
		}
		return Step{Explain: w.explain(n), Expr: w.expression(n), From: w.operands(n), Result: w.result(n), Rule: w.rule(n)}, true
	case kindOperation:
		if w.signOfLiteral(n) {
			return Step{}, false
		}
		return Step{Explain: w.explain(n), Expr: w.expression(n), From: w.operands(n), Result: w.result(n), Rule: w.rule(n)}, true
	case kindFunction:
		if n.Label == "list" {
			return Step{}, false
		}
		return Step{Explain: w.explain(n), Expr: w.expression(n), From: w.operands(n), Result: w.result(n), Rule: w.rule(n)}, true
	}
	return Step{}, false
}

func (w *walker) signOfLiteral(n prov.NodeOut) bool {
	if n.Label != "-" || len(n.Children) != 1 {
		return false
	}
	child, ok := w.nodes[n.Children[0]]
	return ok && child.Kind == kindLiteral
}

func (w *walker) expression(n prov.NodeOut) string {
	if n.Kind == kindFunction && len(n.Children) == 0 && n.ID == w.rootID {
		return w.input
	}
	r := &renderer{nodes: w.nodes, budget: maxRender, path: map[prov.NodeID]bool{}}
	text, _ := r.render(n.ID, 0)
	return text
}

func (w *walker) result(n prov.NodeOut) string {
	if n.Kind == kindFunction && len(n.Children) == 0 && n.ID == w.rootID && w.line.Text != "" {
		return w.line.Text
	}
	return n.Value
}

func (w *walker) operands(n prov.NodeOut) []string {
	var out []string
	for _, id := range n.Children {
		child, ok := w.nodes[id]
		if !ok || child.Value == "" {
			continue
		}
		if child.Kind == kindLiteral && child.Label == child.Value {
			continue
		}
		out = append(out, child.Value)
	}
	return out
}

func (w *walker) rule(n prov.NodeOut) string {
	if n.Label == solveLabel {
		return solveRule(n, w.result(n))
	}
	return ruleOf(n)
}

func ruleOf(n prov.NodeOut) string {
	if n.Detail == "" {
		return ""
	}
	if n.Kind != kindUnit {
		return n.Detail
	}
	if strings.HasPrefix(n.Label, conversionPrefix) {
		return "conversion " + n.Label + ", " + n.Detail
	}
	return "unit: " + n.Detail
}

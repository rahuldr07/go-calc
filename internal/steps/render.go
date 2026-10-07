package steps

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/prov"
)

const (
	precLoose   = 0
	precDefault = 1
	precUnary   = 3
	precPower   = 4
	precAtom    = 9

	elided = "…"
)

var binaryPrecedence = map[string]int{
	"=": -1, "==": -1, "<": -1, ">": -1, "<=": -1, ">=": -1, "!=": -1,
	"+": 1, "-": 1,
	"*": 2, "/": 2, "%": 2,
	"^": precPower,
}

var postfixOperators = map[string]bool{"!": true, "%": true}

type renderer struct {
	nodes  map[prov.NodeID]prov.NodeOut
	budget int
	path   map[prov.NodeID]bool
}

func (r *renderer) render(id prov.NodeID, depth int) (string, int) {
	n, ok := r.nodes[id]
	r.budget--
	if !ok || r.path[id] || depth > maxDepth || r.budget < 0 {
		return elided, precAtom
	}
	r.path[id] = true
	defer delete(r.path, id)
	kids := r.children(n, depth)
	switch n.Kind {
	case kindUnit:
		return r.unit(n, kids)
	case "operation":
		return r.operation(n, kids)
	case kindFunction:
		return call(n.Label, kids), precAtom
	}
	return n.Label, precAtom
}

type child struct {
	text string
	prec int
}

func (r *renderer) children(n prov.NodeOut, depth int) []child {
	out := make([]child, len(n.Children))
	for i, id := range n.Children {
		out[i].text, out[i].prec = r.render(id, depth+1)
	}
	return out
}

func (r *renderer) unit(n prov.NodeOut, kids []child) (string, int) {
	if len(kids) != 1 {
		return call(n.Label, kids), precAtom
	}
	return wrap(kids[0], precAtom) + " " + n.Label, precLoose
}

func (r *renderer) operation(n prov.NodeOut, kids []child) (string, int) {
	switch len(kids) {
	case 1:
		if postfixOperators[n.Label] {
			return wrap(kids[0], precAtom) + n.Label, precAtom
		}
		if n.Label == "-" || n.Label == "+" {
			return n.Label + wrap(kids[0], precUnary), precUnary
		}
	case 2:
		return binary(n.Label, kids[0], kids[1])
	}
	return call(n.Label, kids), precAtom
}

func binary(op string, left, right child) (string, int) {
	p, known := binaryPrecedence[op]
	if !known {
		p = precDefault
	}
	leftNeeds, rightNeeds := p, p+1
	if op == "^" {
		leftNeeds, rightNeeds = p+1, p
	}
	if !known {
		leftNeeds, rightNeeds = p+1, p+1
	}
	return wrap(left, leftNeeds) + op + wrap(right, rightNeeds), p
}

func wrap(c child, needs int) string {
	if c.prec < needs {
		return "(" + c.text + ")"
	}
	return c.text
}

func call(label string, kids []child) string {
	switch label {
	case "range":
		if len(kids) == 2 || len(kids) == 3 {
			parts := make([]string, len(kids))
			for i, k := range kids {
				parts[i] = wrap(k, precAtom)
			}
			return strings.Join(parts, "..")
		}
	case "list":
		parts := make([]string, len(kids))
		for i, k := range kids {
			parts[i] = k.text
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	if len(kids) == 0 {
		return label + "(" + elided + ")"
	}
	parts := make([]string, len(kids))
	for i, k := range kids {
		parts[i] = k.text
	}
	return label + "(" + strings.Join(parts, ", ") + ")"
}

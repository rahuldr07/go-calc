package prov

import (
	"encoding/json"
	"strconv"
	"strings"
)

const maxDepth = 512

type NodeOut struct {
	ID       NodeID   `json:"id"`
	Kind     string   `json:"kind"`
	Label    string   `json:"label"`
	Detail   string   `json:"detail,omitempty"`
	Line     int      `json:"line,omitempty"`
	Children []NodeID `json:"children,omitempty"`
}

type Export struct {
	Root  NodeID    `json:"root"`
	Nodes []NodeOut `json:"nodes"`
}

func (a *Arena) Tree(id NodeID) string {
	if !a.Has(id) {
		return ""
	}
	var b strings.Builder
	a.draw(&b, id, "", "", 0)
	return b.String()
}

func (a *Arena) draw(b *strings.Builder, id NodeID, prefix, branch string, depth int) {
	n := a.Get(id)
	b.WriteString(prefix + branch + Describe(n) + "\n")
	if depth >= maxDepth {
		return
	}
	inner := prefix
	switch branch {
	case "":
	case lastBranch:
		inner += "    "
	default:
		inner += "│   "
	}
	for i, c := range n.Children {
		mark := midBranch
		if i == len(n.Children)-1 {
			mark = lastBranch
		}
		a.draw(b, c, inner, mark, depth+1)
	}
}

const (
	midBranch  = "├── "
	lastBranch = "└── "
)

func Describe(n Node) string {
	out := n.Label + "  [" + n.Kind.String() + "]"
	if n.Detail != "" {
		out += "  " + n.Detail
	}
	if n.Line > 0 {
		out += "  line " + strconv.Itoa(n.Line)
	}
	return out
}

func (a *Arena) Export(id NodeID) Export {
	out := Export{Root: id, Nodes: []NodeOut{}}
	if !a.Has(id) {
		out.Root = None
		return out
	}
	seen := map[NodeID]bool{}
	var walk func(NodeID, int)
	walk = func(cur NodeID, depth int) {
		if seen[cur] || !a.Has(cur) || depth > maxDepth {
			return
		}
		seen[cur] = true
		n := a.Get(cur)
		out.Nodes = append(out.Nodes, NodeOut{
			ID:       cur,
			Kind:     n.Kind.String(),
			Label:    n.Label,
			Detail:   n.Detail,
			Line:     n.Line,
			Children: n.Children,
		})
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(id, 0)
	return out
}

func (a *Arena) JSON(id NodeID) ([]byte, error) { return json.Marshal(a.Export(id)) }

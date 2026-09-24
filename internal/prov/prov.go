package prov

type NodeID int32

const None NodeID = 0

type NodeKind int

const (
	Literal NodeKind = iota
	Input
	Constant
	Unit
	Operation
	Function
	Variable
)

func (k NodeKind) String() string {
	switch k {
	case Literal:
		return "literal"
	case Input:
		return "input"
	case Constant:
		return "constant"
	case Unit:
		return "unit"
	case Operation:
		return "operation"
	case Function:
		return "function"
	case Variable:
		return "variable"
	}
	return "unknown"
}

type Node struct {
	Kind     NodeKind
	Label    string
	Detail   string
	Line     int
	Children []NodeID
}

type Arena struct {
	nodes []Node
}

func NewArena() *Arena { return &Arena{nodes: make([]Node, 1)} }

func (a *Arena) Add(n Node) NodeID {
	kept := make([]NodeID, 0, len(n.Children))
	for _, c := range n.Children {
		if a.Has(c) {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		kept = nil
	}
	n.Children = kept
	a.nodes = append(a.nodes, n)
	return NodeID(len(a.nodes) - 1)
}

func (a *Arena) Has(id NodeID) bool { return id > None && int(id) < len(a.nodes) }

func (a *Arena) Get(id NodeID) Node {
	if !a.Has(id) {
		return Node{}
	}
	n := a.nodes[id]
	n.Children = append([]NodeID(nil), n.Children...)
	return n
}

func (a *Arena) Len() int { return len(a.nodes) }

func (a *Arena) Truncate(n int) {
	if n >= 1 && n <= len(a.nodes) {
		a.nodes = a.nodes[:n]
	}
}

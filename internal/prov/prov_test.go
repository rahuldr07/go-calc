package prov

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestArenaRoundTripsANode(t *testing.T) {
	a := NewArena()
	id := a.Add(Node{Kind: Input, Label: "h", Detail: "12.3 ± 0.2 m", Line: 4})
	got := a.Get(id)
	if got.Kind != Input || got.Label != "h" || got.Detail != "12.3 ± 0.2 m" || got.Line != 4 {
		t.Errorf("Get(%d) = %+v", id, got)
	}
}

func TestNoneIsNeverAStoredNode(t *testing.T) {
	a := NewArena()
	if a.Has(None) {
		t.Error("the arena claims to hold the empty id")
	}
	if id := a.Add(Node{Label: "1"}); id == None {
		t.Error("Add handed back the empty id")
	}
	if got := a.Get(None); got.Label != "" {
		t.Errorf("Get(None) = %+v, want the zero node", got)
	}
	if tree := a.Tree(None); tree != "" {
		t.Errorf("Tree(None) = %q, want empty", tree)
	}
}

func TestAddDropsUnknownChildren(t *testing.T) {
	a := NewArena()
	one := a.Add(Node{Kind: Literal, Label: "1"})
	id := a.Add(Node{Kind: Operation, Label: "+", Children: []NodeID{one, None, NodeID(9999)}})
	if kids := a.Get(id).Children; len(kids) != 1 || kids[0] != one {
		t.Errorf("children = %v, want just %d", kids, one)
	}
}

func TestTreeDrawsEveryLeaf(t *testing.T) {
	a := NewArena()
	two := a.Add(Node{Kind: Literal, Label: "2"})
	h := a.Add(Node{Kind: Input, Label: "h", Detail: "12.3 ± 0.2 m", Line: 1})
	prod := a.Add(Node{Kind: Operation, Label: "·", Children: []NodeID{two, h}})
	g := a.Add(Node{Kind: Constant, Label: "g0", Detail: "CODATA 2022"})
	quot := a.Add(Node{Kind: Operation, Label: "/", Children: []NodeID{prod, g}})
	root := a.Add(Node{Kind: Function, Label: "sqrt", Children: []NodeID{quot}})
	tree := a.Tree(root)
	for _, want := range []string{"sqrt", "function", "literal", "input", "12.3 ± 0.2 m", "line 1", "CODATA 2022", "constant", "operation"} {
		if !strings.Contains(tree, want) {
			t.Errorf("tree missing %q:\n%s", want, tree)
		}
	}
	if lines := strings.Count(strings.TrimSpace(tree), "\n") + 1; lines != 6 {
		t.Errorf("tree has %d lines, want 6:\n%s", lines, tree)
	}
	if strings.Count(tree, "└") != 3 || strings.Count(tree, "├") != 2 {
		t.Errorf("tree branches look wrong:\n%s", tree)
	}
}

func TestTreeIndentsNestedChildren(t *testing.T) {
	a := NewArena()
	leaf := a.Add(Node{Kind: Literal, Label: "1"})
	mid := a.Add(Node{Kind: Operation, Label: "-", Children: []NodeID{leaf}})
	root := a.Add(Node{Kind: Function, Label: "abs", Children: []NodeID{mid}})
	lines := strings.Split(strings.TrimRight(a.Tree(root), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %v", lines)
	}
	if indent(lines[0]) != 0 || indent(lines[1]) != 0 || indent(lines[2]) != 4 {
		t.Errorf("indents = %d %d %d, want 0 0 4\n%s", indent(lines[0]), indent(lines[1]), indent(lines[2]), strings.Join(lines, "\n"))
	}
}

func indent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " │"))
}

func TestJSONCarriesTheRootAndEveryReachableNode(t *testing.T) {
	a := NewArena()
	unreached := a.Add(Node{Kind: Literal, Label: "99"})
	one := a.Add(Node{Kind: Literal, Label: "1"})
	two := a.Add(Node{Kind: Literal, Label: "2"})
	root := a.Add(Node{Kind: Operation, Label: "+", Children: []NodeID{one, two}})
	raw, err := a.JSON(root)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Root  NodeID `json:"root"`
		Nodes []struct {
			ID       NodeID   `json:"id"`
			Kind     string   `json:"kind"`
			Label    string   `json:"label"`
			Children []NodeID `json:"children"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Root != root {
		t.Errorf("root = %d, want %d", out.Root, root)
	}
	if len(out.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 (the unreached %d must be absent)", len(out.Nodes), unreached)
	}
	kinds := map[NodeID]string{}
	for _, n := range out.Nodes {
		kinds[n.ID] = n.Kind
	}
	if kinds[root] != "operation" || kinds[one] != "literal" {
		t.Errorf("kinds = %v", kinds)
	}
}

func TestJSONOfNoneIsEmpty(t *testing.T) {
	a := NewArena()
	raw, err := a.JSON(None)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"nodes":[{`) {
		t.Errorf("JSON(None) = %s, want no nodes", raw)
	}
}

func TestTruncateGivesBackTheIdsItReclaimed(t *testing.T) {
	a := NewArena()
	keep := a.Add(Node{Kind: Literal, Label: "keep"})
	mark := a.Len()
	a.Add(Node{Kind: Literal, Label: "throw"})
	a.Truncate(mark)
	if a.Len() != mark {
		t.Errorf("Len = %d, want %d", a.Len(), mark)
	}
	if got := a.Add(Node{Kind: Literal, Label: "next"}); int(got) != mark {
		t.Errorf("next id = %d, want %d", got, mark)
	}
	if a.Get(keep).Label != "keep" {
		t.Error("truncation lost a node below the mark")
	}
}

func TestEveryKindSpellsItself(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range []NodeKind{Literal, Input, Constant, Unit, Operation, Function, Variable} {
		name := k.String()
		if name == "" || seen[name] {
			t.Errorf("kind %d spells %q", k, name)
		}
		seen[name] = true
	}
}

func TestWhatTheArenaHandsOutCannotBeUsedToRewriteIt(t *testing.T) {
	a := NewArena()
	one := a.Add(Node{Kind: Literal, Label: "1"})
	two := a.Add(Node{Kind: Literal, Label: "2"})
	root := a.Add(Node{Kind: Operation, Label: "+", Children: []NodeID{one, two}})

	got := a.Get(root)
	got.Children[0] = NodeID(999)
	if a.Get(root).Children[0] != one {
		t.Errorf("Get let a caller rewrite the arena: %v", a.Get(root).Children)
	}

	export := a.Export(root)
	for i := range export.Nodes {
		if export.Nodes[i].ID == root {
			export.Nodes[i].Children[1] = NodeID(999)
		}
	}
	if a.Get(root).Children[1] != two {
		t.Errorf("Export let a caller rewrite the arena: %v", a.Get(root).Children)
	}
	if again := a.Export(root); again.Nodes[0].Children[1] != two {
		t.Errorf("a second export saw the first export's edit: %v", again.Nodes[0].Children)
	}
}

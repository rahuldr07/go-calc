package eval

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func evalError(t *testing.T, env *Env, src string) string {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	_, err = env.Eval(n)
	if err == nil {
		t.Fatalf("Eval(%q) succeeded, want an error", src)
	}
	return err.Error()
}

func TestMisspeltNamesGetASuggestion(t *testing.T) {
	cases := [][2]string{
		{"sqre(4)", "unknown function sqre: did you mean sqrt?"},
		{"sqtr(4)", "unknown function sqtr: did you mean sqrt?"},
		{"isprim(7)", "unknown function isprim: did you mean isprime?"},
		{"slove(x=1,x)", "unknown function slove: did you mean solve?"},
		{"dreiv(x^2,x)", "unknown function dreiv: did you mean deriv?"},
		{"sin(pii)", "unknown name pii: did you mean pi?"},
		{"Pi", "unknown name Pi: did you mean pi?"},
		{"10 km in mils", "unknown unit mils: did you mean mile?"},
	}
	for _, c := range cases {
		if got := evalError(t, NewEnv(), c[0]); !strings.HasSuffix(got, c[1]) {
			t.Errorf("%s: got %q, want it to end with %q", c[0], got, c[1])
		}
	}
}

func TestNothingCloseGetsNoSuggestion(t *testing.T) {
	for _, src := range []string{"xyzzy(1)", "qwertyuiop", "10 km in zzzzzz"} {
		if got := evalError(t, NewEnv(), src); strings.Contains(got, "did you mean") {
			t.Errorf("%s: got %q, want no suggestion", src, got)
		}
	}
}

func TestKnownFunctionKeepsItsNamedArgumentError(t *testing.T) {
	got := evalError(t, NewEnv(), "expand((x+1)^2, j=1)")
	if !strings.Contains(got, "takes no argument called j") {
		t.Errorf("got %q", got)
	}
}

func TestUserDefinitionsAreSuggested(t *testing.T) {
	env := NewEnv()
	run(t, env, "velocity = 5")
	run(t, env, "double(x) = 2 * x")
	if got := evalError(t, env, "veloctiy + 1"); !strings.HasSuffix(got, "unknown name veloctiy: did you mean velocity?") {
		t.Errorf("got %q", got)
	}
	if got := evalError(t, env, "dobule(3)"); !strings.HasSuffix(got, "unknown function dobule: did you mean double?") {
		t.Errorf("got %q", got)
	}
}

func TestLiveListsHoldKnownMembers(t *testing.T) {
	for name, c := range map[string]struct {
		list []string
		want []string
	}{
		"Functions": {Functions(), []string{"sqrt", "solve", "deriv", "sin", "asin", "ln", "isprime"}},
		"Constants": {Constants(), []string{"c", "G", "pi", "e"}},
		"Units":     {Units(), []string{"m", "kg", "mile"}},
	} {
		if len(c.list) == 0 {
			t.Errorf("%s is empty", name)
		}
		for _, w := range c.want {
			if name == "Units" && w == "kg" {
				continue
			}
			if !slices.Contains(c.list, w) {
				t.Errorf("%s lacks %q", name, w)
			}
		}
	}
}

func TestEveryListedFunctionIsDispatched(t *testing.T) {
	for _, name := range Functions() {
		env := NewEnv()
		n, _, err := parse.Parse(name + "()")
		if err != nil {
			continue
		}
		_, err = env.Eval(n)
		if err != nil && strings.Contains(err.Error(), "unknown function") {
			t.Errorf("%s is listed but not dispatched: %v", name, err)
		}
	}
}

func TestSwitchedFunctionsMatchTheBuiltinSwitch(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "eval.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.FuncDecl)
		if !ok || decl.Name.Name != "builtin" {
			return true
		}
		for _, stmt := range decl.Body.List {
			sw, ok := stmt.(*ast.SwitchStmt)
			if !ok {
				continue
			}
			for _, clause := range sw.Body.List {
				for _, expr := range clause.(*ast.CaseClause).List {
					if lit, ok := expr.(*ast.BasicLit); ok {
						s, _ := strconv.Unquote(lit.Value)
						found = append(found, s)
					}
				}
			}
		}
		return false
	})
	slices.Sort(found)
	want := slices.Clone(switchedFunctions)
	slices.Sort(want)
	if !slices.Equal(found, want) {
		t.Errorf("builtin switch handles %v, switchedFunctions lists %v", found, want)
	}
}

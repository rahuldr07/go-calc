package eval

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/prov"
)

var benignOperands = []string{"2", "7.5", "0.125", "3/4", "1e6", "1e-6", "123456789", "0.1", "12.25", "1000"}

var benignShapes = []string{
	"%[1]s + %[2]s",
	"%[1]s - %[2]s",
	"%[1]s * %[2]s",
	"%[1]s / %[2]s",
	"(%[1]s + %[2]s) * %[1]s",
	"(%[1]s * %[2]s) / %[2]s",
	"%[1]s + %[2]s * 2",
	"sqrt(%[1]s * %[1]s + %[2]s * %[2]s)",
	"exp(%[2]s / 1e9) * %[1]s",
	"abs(%[1]s) + abs(%[2]s)",
	"%[1]s m + %[2]s m",
	"%[1]s km / %[2]s h",
	"%[1]s + 10%%",
	"%[1]s - 5%%",
	"%[1]s m to cm",
	"%[1]s * g0",
	"%[1]s / pi",
	"atan(%[1]s / %[2]s)",
	"%[1]s ^ 2 + %[2]s",
	"log(%[1]s + 1e3)",
	"floor(%[1]s) + ceil(%[2]s)",
	"%[1]s ± %[2]s",
}

func TestShadowDoesNotWarnAboutOrdinaryArithmetic(t *testing.T) {
	warned := 0
	checked := 0
	for _, shape := range benignShapes {
		for _, a := range benignOperands {
			for _, b := range benignOperands {
				src := fmt.Sprintf(shape, a, b)
				env := NewEnv()
				n := mustParse(t, src)
				v, err := env.Eval(n)
				if err != nil {
					continue
				}
				checked++
				if w := env.Warning(n, v); w != "" {
					warned++
					if warned <= 10 {
						t.Errorf("%s: warned %q", src, w)
					}
				}
			}
		}
	}
	if checked < 1000 {
		t.Fatalf("only %d expressions were checked", checked)
	}
	if warned > 0 {
		t.Errorf("%d of %d ordinary expressions drew a warning", warned, checked)
	}
}

func TestShadowWarnsAboutEveryKnownCancellation(t *testing.T) {
	var srcs []string
	for _, a := range []string{"1e16", "1e17", "1e18", "1e20"} {
		srcs = append(srcs, "("+a+" + 1) - "+a)
	}
	for _, a := range []string{"1e10", "1e12", "1e14", "1e16"} {
		srcs = append(srcs, "sqrt("+a+" + 1) - sqrt("+a+")")
	}
	for _, x := range []string{"1e-8", "1e-9", "1e-10", "1e-12"} {
		srcs = append(srcs, "1 - cos("+x+")")
	}
	for _, x := range []string{"1e-15", "1e-16", "1e-17"} {
		srcs = append(srcs, "log(1 + "+x+")")
	}
	for _, src := range srcs {
		env := NewEnv()
		n := mustParse(t, src)
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("Eval(%q): %v", src, err)
		}
		if w := env.Warning(n, v); w == "" {
			t.Errorf("%s: no warning; shadow = %+v, proven = %s", src, env.Shadow(n), v.Num.String())
		}
	}
}

func TestProvenanceReachesEveryLeafOfEveryExpression(t *testing.T) {
	srcs := []string{
		"2 + 3 * 4",
		"sqrt(2) / pi",
		"3 km to m",
		"5 ft + 2 inch",
		"g0 * 2",
		"20 degC to degF",
		"(1 + 2) ^ 3",
		"-7 mod 3",
		"5!",
		"atan2(1, 2)",
		"gcd(12, 18)",
		"1e3 + 25%",
		"round(2.5)",
		"abs(-4) ^ 2",
		"root(27, 3)",
	}
	for _, src := range srcs {
		env := NewEnv()
		n := mustParse(t, src)
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("Eval(%q): %v", src, err)
		}
		if v.Prov == prov.None {
			t.Errorf("%s: no provenance node", src)
			continue
		}
		export := env.Arena().Export(v.Prov)
		if export.Root != v.Prov || len(export.Nodes) == 0 {
			t.Errorf("%s: export = %+v", src, export)
			continue
		}
		known := map[prov.NodeID]bool{}
		for _, node := range export.Nodes {
			known[node.ID] = true
		}
		for _, node := range export.Nodes {
			for _, c := range node.Children {
				if !known[c] {
					t.Errorf("%s: node %d names an unexported child %d", src, node.ID, c)
				}
			}
		}
		if tree := env.Arena().Tree(v.Prov); strings.TrimSpace(tree) == "" {
			t.Errorf("%s: empty tree", src)
		}
	}
}

func TestProvenanceNamesEveryConstantAndUnitInTheSource(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{"2 s * g0", []string{"g0", "CODATA 2022", "s", "constant", "unit"}},
		{"3 km to m", []string{"km", "1 km = 1000 m", "to m"}},
		{"sqrt(2) / pi", []string{"sqrt", "pi", "function", "constant"}},
		{"5 ft + 2 inch", []string{"ft", "inch", "381/1250", "127/5000"}},
		{"c / 2", []string{"c", "speed of light", "CODATA 2022"}},
	}
	for _, c := range cases {
		env := NewEnv()
		v := mustEval(t, env, c.src)
		tree := env.Arena().Tree(v.Prov)
		for _, want := range c.want {
			if !strings.Contains(tree, want) {
				t.Errorf("%s: trace missing %q:\n%s", c.src, want, tree)
			}
		}
	}
}

func TestProvenanceSurvivesThePrecisionRetry(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Prec = 16
	ctx.Digits = 30
	n := mustParse(t, "sin(1) * exp(2) / pi")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatal(err)
	}
	tree := env.Arena().Tree(v.Prov)
	for _, want := range []string{"sin", "exp", "pi"} {
		if !strings.Contains(tree, want) {
			t.Errorf("trace missing %q after a retry:\n%s", want, tree)
		}
	}
	if got := strings.Count(tree, "sin"); got != 1 {
		t.Errorf("the retry left %d copies of sin:\n%s", got, tree)
	}
	if got := env.Arena().Len(); got != 8 {
		t.Errorf("arena holds %d nodes after %d retries, want the 7 of one pass", got, retries)
	}
}

const retries = 3

func TestVariablesKeepTheirProvenanceAcrossLines(t *testing.T) {
	env := NewEnv()
	run(t, env, "a = 2 + 3")
	run(t, env, "b = a * 4")
	v := mustEval(t, env, "b + 1")
	tree := env.Arena().Tree(v.Prov)
	for _, want := range []string{"b", "a", "variable", "2", "3", "4"} {
		if !strings.Contains(tree, want) {
			t.Errorf("trace missing %q:\n%s", want, tree)
		}
	}
}

func TestAnAssignedVariableKeepsAValidTraceThroughARetry(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Prec = 16
	ctx.Digits = 30
	if _, err := env.EvalCtx(mustParse(t, "a = pi / 3"), ctx); err != nil {
		t.Fatal(err)
	}
	held, ok := env.Get("a")
	if !ok {
		t.Fatal("a was not stored")
	}
	tree := env.Arena().Tree(held.Prov)
	for _, want := range []string{"pi", "3", "/"} {
		if !strings.Contains(tree, want) {
			t.Errorf("stored trace missing %q:\n%s", want, tree)
		}
	}
	v := mustEval(t, env, "a * 2")
	if later := env.Arena().Tree(v.Prov); !strings.Contains(later, "pi") {
		t.Errorf("a later line lost the stored trace:\n%s", later)
	}
}

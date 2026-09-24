package eval

import (
	"math"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func mustParse(t *testing.T, src string) parse.Node {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return n
}

func TestShadowFindsTheCancellation(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("(1e16 + 1) - 1e16")
	sh := env.Shadow(n)
	if sh.Value != 0 {
		t.Fatalf("float64 shadow = %v, want 0", sh.Value)
	}
	if len(sh.Lost) == 0 {
		t.Fatal("no loss site recorded")
	}
	worst := sh.Lost[0]
	if worst.Op != "-" || worst.BitsLost < 50 {
		t.Errorf("worst site = %+v, want the subtraction losing at least 50 bits", worst)
	}
}

func TestNoWarningOnBenignArithmetic(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("2 + 3 * 4")
	if sh := env.Shadow(n); len(sh.Lost) != 0 {
		t.Errorf("benign expression reported loss: %+v", sh.Lost)
	}
}

func TestTraceNamesInputsAndCitations(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 12.3 ± 0.2 m")
	v := mustEval(t, env, "sqrt(2h / g0)")
	tree := env.Arena().Tree(v.Prov)
	for _, want := range []string{"h", "input", "g0", "CODATA 2022"} {
		if !strings.Contains(tree, want) {
			t.Errorf("trace missing %q:\n%s", want, tree)
		}
	}
}

func TestShadowCatalogueWarnsAndStaysSilent(t *testing.T) {
	warns := []string{
		"(1e16 + 1) - 1e16",
		"sqrt(x + 1) - sqrt(x)",
		"1 - cos(1e-8)",
		"log(1 + 1e-15)",
	}
	silent := []string{
		"2 + 2",
		"355 / 113",
		"exp(1)",
		"1e10 + 1e-10",
	}
	for _, src := range warns {
		env := NewEnv()
		run(t, env, "x = 1e10")
		n := mustParse(t, src)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w == "" {
			t.Errorf("%s: no warning; shadow = %+v", src, env.Shadow(n))
		}
	}
	for _, src := range silent {
		env := NewEnv()
		n := mustParse(t, src)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q", src, w)
		}
	}
}

func TestWarningNamesTheWorstSite(t *testing.T) {
	env := NewEnv()
	src := "(1e16 + 1) - 1e16"
	n := mustParse(t, src)
	w := env.Warning(n, mustEval(t, env, src))
	for _, want := range []string{"float64", "-", "bits"} {
		if !strings.Contains(w, want) {
			t.Errorf("warning %q missing %q", w, want)
		}
	}
}

func TestWhyExplainsBothAnswers(t *testing.T) {
	env := NewEnv()
	src := "1 - cos(1e-8)"
	n := mustParse(t, src)
	why := env.Why(n, mustEval(t, env, src))
	for _, want := range []string{"float64", "proven", "bits"} {
		if !strings.Contains(why, want) {
			t.Errorf("why %q missing %q:\n%s", src, want, why)
		}
	}
}

func TestWhyStaysQuietlyHonestOnBenignArithmetic(t *testing.T) {
	env := NewEnv()
	src := "2 + 3 * 4"
	n := mustParse(t, src)
	why := env.Why(n, mustEval(t, env, src))
	if !strings.Contains(why, "14") {
		t.Errorf("why = %q, want it to name the shared answer", why)
	}
	if strings.Contains(why, "bits at") {
		t.Errorf("why = %q, want no loss site", why)
	}
}

func TestShadowStaysSilentWhereItCannotModelTheEvaluator(t *testing.T) {
	env := NewEnv()
	run(t, env, "f(x) = x + 1")
	for _, src := range []string{"20 degC to K", "20 degC + 5 K", "f(2)", "6!", "gcd(12, 18)"} {
		n := mustParse(t, src)
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("Eval(%q): %v", src, err)
		}
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q", src, w)
		}
	}
}

func TestShadowTracksUnitsInSI(t *testing.T) {
	env := NewEnv()
	src := "3 km + 200 m"
	n := mustParse(t, src)
	sh := env.Shadow(n)
	if !sh.modelled || math.Abs(sh.Value-3200) > 1e-9 {
		t.Errorf("shadow = %+v, want 3200 in SI", sh)
	}
}

func TestShadowMirrorsThePercentRule(t *testing.T) {
	env := NewEnv()
	n := mustParse(t, "500 + 10%")
	sh := env.Shadow(n)
	if !sh.modelled || math.Abs(sh.Value-550) > 1e-9 {
		t.Errorf("shadow = %+v, want 550", sh)
	}
	if w := env.Warning(n, mustEval(t, env, "500 + 10%")); w != "" {
		t.Errorf("warned %q on a percent increase", w)
	}
}

func TestShadowSeesLossOneOperationFurtherOut(t *testing.T) {
	env := NewEnv()
	for _, src := range []string{"((1e16 + 1) - 1e16) * 3", "2 * (1 - cos(1e-8))", "(1 - cos(1e-8)) + 1e-30"} {
		n := mustParse(t, src)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w == "" {
			t.Errorf("%s: no warning one operation further out", src)
		}
	}
}

func TestWarningIsAbsentWhenNoProvenValueDisagrees(t *testing.T) {
	env := NewEnv()
	for _, src := range []string{"1 + 2 - 3", "(1e16 + 1e16) - 1e16"} {
		n := mustParse(t, src)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q although float64 agrees", src, w)
		}
	}
}

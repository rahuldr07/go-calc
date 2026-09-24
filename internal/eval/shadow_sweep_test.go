package eval

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/prov"
)

var benignOperands = []string{"2", "7.5", "0.125", "3/4", "1e6", "1e-6", "12345", "0.1", "12.25", "1000"}

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

func TestShadowWarnsWhenAPrintedIntegerDigitIsWrong(t *testing.T) {
	cases := []struct {
		src   string
		wrong string
		gives string
	}{
		{"25!", "the last 10 digits are wrong", "15511210043330986055303168"},
		{"3 ^ 200", "the last 80 digits are wrong", ""},
		{"12345678901234567890 + 1", "the last 3 digits are wrong", "12345678901234567168"},
		{"123456789 * 123456789", "the last digit is wrong", "15241578750190520"},
		{"9007199254740992 + 1", "the last digit is wrong", "9007199254740992"},
	}
	for _, c := range cases {
		env := NewEnv()
		n := mustParse(t, c.src)
		v := mustEval(t, env, c.src)
		w := env.Warning(n, v)
		if w == "" {
			t.Errorf("%s: no warning although float64 prints wrong digits", c.src)
			continue
		}
		if !strings.Contains(w, c.wrong) {
			t.Errorf("%s: warning %q does not say %q", c.src, w, c.wrong)
		}
		if c.gives != "" && !strings.Contains(w, "float64 gives "+c.gives) {
			t.Errorf("%s: warning %q does not name the float64 answer %s", c.src, w, c.gives)
		}
	}
}

func TestExactIntegersInsideFloat64StaySilent(t *testing.T) {
	for _, src := range []string{
		"2 + 2",
		"2 ^ 100",
		"20!",
		"12345 * 12345",
		"1e15 + 1",
		"9007199254740992",
		"1000000 * 1000000",
		"-5 * 7",
		"7 - 7",
		"3 km to m",
		"1 lakh * 3",
	} {
		env := NewEnv()
		n := mustParse(t, src)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q", src, w)
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

func TestTheWorstLossSiteIsTheWorstOne(t *testing.T) {
	env := NewEnv()
	src := "((1e16 + 3) - 1e16) - ((1e16 + 2) - 1e16)"
	sh := env.Shadow(mustParse(t, src))
	if len(sh.Lost) != 3 {
		t.Fatalf("Lost = %+v, want three sites", sh.Lost)
	}
	worst := sh.Lost[0]
	if worst.Op != "-" || worst.Pos.Line != 1 || worst.Pos.Col != 35 {
		t.Errorf("worst site = %+v, want the - at 1:35", worst)
	}
	if worst.BitsLost < 52.0 || worst.BitsLost > 52.3 {
		t.Errorf("worst BitsLost = %v, want about 52.15", worst.BitsLost)
	}
	for i := 1; i < len(sh.Lost); i++ {
		if sh.Lost[i-1].BitsLost < sh.Lost[i].BitsLost {
			t.Errorf("Lost is not ordered worst first: %+v", sh.Lost)
		}
	}
	if sh.Lost[1].Pos.Col != 13 || sh.Lost[1].BitsLost < 51.0 || sh.Lost[1].BitsLost > 51.3 {
		t.Errorf("second site = %+v, want about 51.15 bits at 1:13", sh.Lost[1])
	}
	if sh.Lost[2].Pos.Col != 21 || sh.Lost[2].BitsLost < 0.9 || sh.Lost[2].BitsLost > 1.1 {
		t.Errorf("third site = %+v, want about 1 bit at 1:21", sh.Lost[2])
	}
	env.SetLine(4)
	if again := env.Shadow(mustParse(t, src)); again.Lost[0].Pos.Line != 4 {
		t.Errorf("a sheet line was not stamped onto the worst site: %+v", again.Lost[0])
	}
}

func TestBitsLostIsALogBaseTwoOfTheShrinkage(t *testing.T) {
	cases := []struct {
		src       string
		low, high float64
	}{
		{"1 - cos(1e-6)", 40.8, 41.0},
		{"sqrt(1e10 + 1) - sqrt(1e10)", 34.1, 34.3},
		{"(1e16 + 2) - 1e16", 52.0, 52.3},
		{"(1e16 + 4) - 1e16", 51.0, 51.3},
		{"1e16 - 1e12", 0.0, 0.1},
	}
	for _, c := range cases {
		env := NewEnv()
		sh := env.Shadow(mustParse(t, c.src))
		if c.high <= minBitsLost {
			if len(sh.Lost) != 0 {
				t.Errorf("%s: recorded %+v, want no site", c.src, sh.Lost)
			}
			continue
		}
		if len(sh.Lost) == 0 {
			t.Errorf("%s: no site recorded", c.src)
			continue
		}
		if got := sh.Lost[0].BitsLost; got < c.low || got > c.high {
			t.Errorf("%s: BitsLost = %v, want between %v and %v", c.src, got, c.low, c.high)
		}
	}
}

func TestAWrongPrintedDigitWarnsWhateverUnitItWears(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"12345678901234567890 + 1", "float64 gives 12345678901234567168 here; the last 3 digits are wrong"},
		{"12345678901234567890 m + 1 m", "float64 gives 12345678901234567168 here; the last 3 digits are wrong"},
		{"12345678901234567890 km + 1 km", "float64 gives 12345678901234567168 here; the last 3 digits are wrong"},
		{"12345678901234567890 s + 1 s", "float64 gives 12345678901234567168 here; the last 3 digits are wrong"},
		{"12345678901234567890 h + 1 h", "float64 gives 12345678901234567168 here; the last 3 digits are wrong"},
	}
	for _, c := range cases {
		env := NewEnv()
		n := mustParse(t, c.src)
		v := mustEval(t, env, c.src)
		if got := env.Warning(n, v); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.src, got, c.want)
		}
	}
}

func TestAnIntegerShadowIsNeverInvented(t *testing.T) {
	for _, src := range []string{"0.3 / 0.1", "root(27, 3)", "root(1000, 3)", "1 / 49 * 49", "0.1 * 30", "1e17 / 1e16", "sqrt(9)"} {
		env := NewEnv()
		n := mustParse(t, src)
		sh := env.Shadow(n)
		v := mustEval(t, env, src)
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q although float64 holds %v", src, w, sh.Value)
		}
		why := env.Why(n, v)
		if why.WrongDigits != 0 {
			t.Errorf("%s: why = %+v, want no wrong printed digit", src, why)
		}
		if sh.Value != math.Trunc(sh.Value) && why.ShadowExact != "" {
			t.Errorf("%s: why named the integer %q although float64 holds %v", src, why.ShadowExact, sh.Value)
		}
	}
}

func TestTheShadowKeepsTheNumberTheCalculatorShows(t *testing.T) {
	cases := []struct {
		src       string
		shown, si float64
	}{
		{"3 km + 200 m", 3.2, 3200},
		{"3 km to m", 3000, 3000},
		{"2 h + 30 min", 2.5, 9000},
		{"5 + 3", 8, 8},
		{"2 lakh", 200000, 200000},
		{"100 cm to m", 1, 1},
	}
	for _, c := range cases {
		env := NewEnv()
		sh := env.Shadow(mustParse(t, c.src))
		if !sh.modelled {
			t.Errorf("%s: unmodelled", c.src)
			continue
		}
		if math.Abs(sh.Value-c.shown) > 1e-9 {
			t.Errorf("%s: shown = %v, want %v", c.src, sh.Value, c.shown)
		}
		if math.Abs(sh.si-c.si) > 1e-9 {
			t.Errorf("%s: SI = %v, want %v", c.src, sh.si, c.si)
		}
	}
}

func TestUnitDetailNamesTheFactorWhicheverUnitItIs(t *testing.T) {
	env := NewEnv()
	for _, c := range []struct{ src, want string }{
		{"3 m to ft", "1 ft = 381/1250 m (length)"},
		{"3 km to m", "1 m = 1 m (length)"},
		{"2 lakh", "1 lakh = 100000 (dimensionless)"},
		{"3 ft", "1 ft = 381/1250 m (length)"},
	} {
		v := mustEval(t, env, c.src)
		if tree := env.Arena().Tree(v.Prov); !strings.Contains(tree, c.want) {
			t.Errorf("%s: trace missing %q:\n%s", c.src, c.want, tree)
		}
	}
}

func TestTheShadowTracksTheProvenValueInSI(t *testing.T) {
	shapes := []string{
		"%[1]s km + %[2]s m",
		"%[1]s h + %[2]s min",
		"%[1]s ft + %[2]s inch",
		"%[1]s kg * %[2]s m",
		"%[1]s km / %[2]s h",
		"%[1]s m to cm",
		"%[1]s lb to g",
		"%[1]s bar * %[2]s",
		"sqrt(%[1]s m * %[2]s m)",
		"abs(%[1]s km - %[2]s km)",
		"floor(%[1]s m)",
		"(%[1]s km) ^ 2",
		"(%[1]s cm) ^ 3",
		"(%[1]s h) ^ 2",
		"(%[1]s ft) ^ 2 * %[2]s",
		"(%[1]s g) ^ -1",
		"(%[1]s min) ^ 1",
		"(%[1]s km) ^ 2 / (%[2]s h) ^ 2",
		"(%[1]s lakh) ^ 2",
		"sqrt((%[1]s m) ^ 2)",
	}
	checked := 0
	for _, shape := range shapes {
		for _, a := range []string{"2", "7.5", "0.125", "1000", "3/4"} {
			for _, b := range []string{"3", "12.5", "0.5", "250"} {
				src := fmt.Sprintf(shape, a, b)
				env := NewEnv()
				n := mustParse(t, src)
				v, err := env.Eval(n)
				if err != nil {
					continue
				}
				sh := env.Shadow(n)
				if !sh.modelled {
					continue
				}
				checked++
				want := siFloat(v, DefaultCtx().Prec)
				if want == 0 {
					continue
				}
				if rel := math.Abs(sh.si-want) / math.Abs(want); rel > 1e-12 {
					t.Errorf("%s: shadow SI = %v, proven SI = %v (relative gap %v)", src, sh.si, want, rel)
				}
				shownWant := numFloat(v.Num, DefaultCtx().Prec)
				if shownWant != 0 {
					if rel := math.Abs(sh.Value-shownWant) / math.Abs(shownWant); rel > 1e-12 {
						t.Errorf("%s: shadow shows %v, calculator shows %v", src, sh.Value, shownWant)
					}
				}
			}
		}
	}
	if checked < 250 {
		t.Fatalf("only %d expressions were checked", checked)
	}
}

func TestPowerKeepsTheUnitTheCalculatorPrints(t *testing.T) {
	cases := []struct {
		src   string
		shown float64
	}{
		{"3 km ^ 2", 3},
		{"3 cm ^ 2", 3},
		{"2 ft ^ 3", 2},
		{"5 h ^ 1", 5},
		{"4 g ^ 2", 4},
		{"2 inch ^ 2", 2},
		{"10 min ^ 2", 10},
		{"3 m ^ 2", 3},
		{"3 km ^ 0", 3},
		{"(3 km) ^ 2", 9},
		{"(3 cm) ^ 2", 9},
		{"(2 ft) ^ 3", 8},
		{"(4 g) ^ 2", 16},
		{"(3 km) ^ 0", 1},
		{"lakh ^ 2", 1e10},
		{"2 lakh ^ 2", 2e10},
		{"(2 lakh) ^ 2", 4e10},
		{"crore ^ 2", 1e14},
		{"(lakh + 1) ^ 2", 10000200001},
		{"2 ^ 10", 1024},
	}
	for _, c := range cases {
		env := NewEnv()
		n := mustParse(t, c.src)
		sh := env.Shadow(n)
		if !sh.modelled {
			t.Errorf("%s: unmodelled", c.src)
			continue
		}
		if math.Abs(sh.Value-c.shown) > math.Abs(c.shown)*1e-12+1e-12 {
			t.Errorf("%s: shadow shows %v, calculator shows %v", c.src, sh.Value, c.shown)
		}
		v := mustEval(t, env, c.src)
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: warned %q about an answer float64 gets right", c.src, w)
		}
	}
}

func TestPowerThroughAVariableInASheet(t *testing.T) {
	env := NewEnv()
	run(t, env, "side = 3 km")
	src := "area = side ^ 2"
	n := mustParse(t, src)
	v := mustEval(t, env, src)
	if w := env.Warning(n, v); w != "" {
		t.Errorf("warned %q on the simplest two-line sheet", w)
	}
	if sh := env.Shadow(n); math.Abs(sh.Value-9) > 1e-12 || math.Abs(sh.si-9e6) > 1 {
		t.Errorf("shadow = %+v, want 9 shown and 9e6 in SI", sh)
	}
}

func TestPowerStillCatchesAWideExactInteger(t *testing.T) {
	env := NewEnv()
	src := "3 ^ 200"
	n := mustParse(t, src)
	if w := env.Warning(n, mustEval(t, env, src)); !strings.Contains(w, "the last 80 digits are wrong") {
		t.Errorf("warning = %q, want the 80 wrong digits", w)
	}
}

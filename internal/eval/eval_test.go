package eval

import (
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func run(t *testing.T, env *Env, src string) string {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	v, err := env.Eval(n)
	if err != nil {
		t.Fatalf("Eval(%q): %v", src, err)
	}
	out := v.Num.String()
	if v.Unit != "" {
		out += " " + v.Unit
	}
	return out
}

func TestArithmeticIsExact(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "0.1 + 0.2"); got != "3/10" {
		t.Errorf("0.1 + 0.2 = %s, want 3/10", got)
	}
	if got := run(t, env, "(1e16 + 1) - 1e16"); got != "1" {
		t.Errorf("cancellation case = %s, want 1", got)
	}
	if got := run(t, env, "2 ^ 10"); got != "1024" {
		t.Errorf("2^10 = %s, want 1024", got)
	}
}

func TestPercent(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "500 + 10%"); got != "550" {
		t.Errorf("500 + 10%% = %s, want 550", got)
	}
	if got := run(t, env, "10%"); got != "1/10" {
		t.Errorf("10%% = %s, want 1/10", got)
	}
	if got := run(t, env, "500 - 10%"); got != "450" {
		t.Errorf("500 - 10%% = %s, want 450", got)
	}
}

func TestModuloAndFactorial(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "17 mod 5"); got != "2" {
		t.Errorf("17 mod 5 = %s, want 2", got)
	}
	if got := run(t, env, "5!"); got != "120" {
		t.Errorf("5! = %s, want 120", got)
	}
}

func TestUnitsAndConversion(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "3 km to m"); got != "3000 m" {
		t.Errorf("3 km to m = %s, want 3000 m", got)
	}
	if got := run(t, env, "5 km + 300 m"); got != "53/10 km" {
		t.Errorf("5 km + 300 m = %s, want 53/10 km", got)
	}
	if got := run(t, env, "20 degC to K"); got != "5863/20 K" {
		t.Errorf("20 degC to K = %s, want 5863/20 K", got)
	}
}

func TestDimensionMismatchNamesBoth(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("5 m + 3 s")
	_, err := env.Eval(n)
	if err == nil {
		t.Fatal("want error")
	}
	if want := "cannot add length (m) and time (s)"; err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestUnitPositionBeatsVariable(t *testing.T) {
	env := NewEnv()
	run(t, env, "h = 3")
	if got := run(t, env, "5 h"); got != "5 h" {
		t.Errorf("5 h = %s, want 5 h (hours), not 15", got)
	}
	if got := run(t, env, "h + 1"); got != "4" {
		t.Errorf("h + 1 = %s, want 4", got)
	}
}

func TestVariablesAndFunctions(t *testing.T) {
	env := NewEnv()
	run(t, env, "f(x) = x ^ 2 + 1")
	if got := run(t, env, "f(3)"); got != "10" {
		t.Errorf("f(3) = %s, want 10", got)
	}
	run(t, env, "salary = 18,50,000 / 12")
	if got := run(t, env, "salary + 10%"); got != "2035000/12" && got != "508750/3" {
		t.Errorf("salary + 10%% = %s, want 508750/3", got)
	}
}

func TestExactRootsAndIntegerFunctions(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "sqrt(16/9)"); got != "4/3" {
		t.Errorf("sqrt(16/9) = %s, want 4/3", got)
	}
	if got := run(t, env, "gcd(48, 18)"); got != "6" {
		t.Errorf("gcd(48,18) = %s, want 6", got)
	}
	if got := run(t, env, "floor(7/2)"); got != "3" {
		t.Errorf("floor(7/2) = %s, want 3", got)
	}
	if got := run(t, env, "round(-7/2)"); got != "-3" && got != "-4" {
		t.Errorf("round(-7/2) = %s, want -3 or -4", got)
	}
}

func TestConstantsResolve(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "c"); got != "299792458 m/s" {
		t.Errorf("c = %s, want 299792458 m/s", got)
	}
	if got := run(t, env, "g0"); got != "196133/20000 m/s^2" {
		t.Errorf("g0 = %s, want 196133/20000 m/s^2", got)
	}
}

func TestIrrationalIsProven(t *testing.T) {
	const sqrt2 = "1.41421356237309504880168872420969807856967187537694"
	true2, _, err := big.ParseFloat(sqrt2, 10, 300, big.ToNearestEven)
	if err != nil {
		t.Fatal(err)
	}
	env := NewEnv()
	ctx := DefaultCtx()
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("sqrt(2): %v", err)
	}
	if v.Num.IsExact() {
		t.Fatal("sqrt(2) is not a rational")
	}
	b := v.Num.Ball(ctx.Prec)
	if !b.Contains(true2) {
		t.Fatalf("ball %s does not contain the true square root of two", b)
	}
	if got := format.GuaranteedDigits(b); got < 12 {
		t.Fatalf("only %d digits guaranteed, want at least 12", got)
	}
	n, _, _ = parse.Parse("sin(1)")
	if _, err := env.EvalCtx(n, DefaultCtx()); err != nil {
		t.Fatalf("sin(1): %v", err)
	}
}

func TestLakhAndCrore(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "5 lakh"); got != "500000" {
		t.Errorf("5 lakh = %s, want 500000", got)
	}
	if got := run(t, env, "2 crore"); got != "20000000" {
		t.Errorf("2 crore = %s, want 20000000", got)
	}
}

func TestDivisionByZeroIsReported(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("1 / 0")
	if _, err := env.Eval(n); err == nil {
		t.Fatal("want division by zero error")
	}
}

func TestCommaSeparatedArgumentsWithoutASpace(t *testing.T) {
	env := NewEnv()
	if got := run(t, env, "gcd(48,18)"); got != "6" {
		t.Errorf("gcd(48,18) = %s, want 6", got)
	}
	if got := run(t, env, "root(27,3)"); got != "3" {
		t.Errorf("root(27,3) = %s, want 3", got)
	}
	if got := run(t, env, "18,50,000 + 1,000"); got != "1851000" {
		t.Errorf("grouped numbers = %s, want 1851000", got)
	}
}

func TestUnderscoresAndCommaGroupingCompose(t *testing.T) {
	env := NewEnv()
	for _, c := range []struct{ src, want string }{
		{"1_000,000", "1000000"},
		{"1_000_000", "1000000"},
		{"18,50,000", "1850000"},
		{"1,000", "1000"},
	} {
		if got := run(t, env, c.src); got != c.want {
			t.Errorf("%s = %s, want %s", c.src, got, c.want)
		}
	}
}

func TestRoundingKeepsTheUnitItRoundedIn(t *testing.T) {
	env := NewEnv()
	for _, c := range []struct {
		src  string
		want string
		unit string
	}{
		{"floor(1.5 km)", "1 km", "km"},
		{"ceil(1.5 km)", "2 km", "km"},
		{"round(1.5 km)", "2 km", "km"},
		{"floor(1500 m)", "1500 m", "m"},
		{"floor(2.5 m/s)", "2 m/s", "m/s"},
		{"floor(2 lakh in crore)", "0 crore", "crore"},
		{"ceil(2 lakh in crore)", "1 crore", "crore"},
		{"floor(3.7 degC)", "3 degC", "degC"},
		{"ceil(3.2 degC)", "4 degC", "degC"},
		{"round(3.5 degC)", "4 degC", "degC"},
		{"floor(7/2)", "3", ""},
		{"floor(1.5 lakh)", "150000", ""},
	} {
		v := mustEval(t, env, c.src)
		if got := format.Text(v, nil); got != c.want {
			t.Errorf("%s = %q, want %q", c.src, got, c.want)
		}
		if v.Unit != c.unit {
			t.Errorf("%s kept unit %q, want %q", c.src, v.Unit, c.unit)
		}
		if c.unit == "" {
			continue
		}
		operand := mustEval(t, env, strings.TrimSuffix(strings.SplitN(c.src, "(", 2)[1], ")"))
		if v.Dim != operand.Dim || v.FactorOrOne().Cmp(operand.FactorOrOne()) != 0 || v.Offset.Cmp(operand.Offset) != 0 {
			t.Errorf("%s changed the unit it rounded in: %v %v %v, want %v %v %v",
				c.src, v.Dim, v.FactorOrOne(), v.Offset, operand.Dim, operand.FactorOrOne(), operand.Offset)
		}
	}
}

func TestRoundingCarriesItsUnitOneOperationFurtherOut(t *testing.T) {
	env := NewEnv()
	for _, c := range [][2]string{
		{"floor(1.5 km) + (500 m)", "1.5 km"},
		{"floor(1.5 km) in m", "1000 m"},
		{"(floor(1.5 km)) ^ 2", "1 km^2"},
		{"floor(floor(1.5 km))", "1 km"},
		{"floor(1.5 km) - (1 km)", "0 km"},
		{"(floor(1.5 km)) * 2", "2000 m"},
		{"(floor(2 lakh in crore)) * 1", "0"},
		{"(ceil(2 lakh in crore)) * 1", "10000000"},
		{"ceil(2 lakh in crore) in lakh", "100 lakh"},
		{"(floor(3.7 degC)) ^ 2", "76258.8225 K^2"},
		{"floor(3.7 degC) + (1 K)", "4 degC"},
		{"sqrt((floor(2 m)) ^ 2)", "2 m"},
	} {
		if got := format.Text(mustEval(t, env, c[0]), nil); got != c[1] {
			t.Errorf("%s = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestIntegerFunctionsRefuseAUnitCarryingOperand(t *testing.T) {
	env := NewEnv()
	for _, src := range []string{
		"gcd(6 km, 4 km)", "gcd(4, 6 km)", "lcm(6 km, 4)", "lcm(4, 6 km)",
		"isprime(7 m)", "isprime(3 degC)", "(6 km) mod 4", "4 mod (6 km)",
		"isprime(2 lakh in crore)", "(2 lakh in crore) mod 3", "gcd(2 lakh in crore, 4)",
		"gcd(3 degC, 4)", "(3 degC) mod 2",
	} {
		n, _, err := parse.Parse(src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		v, err := env.Eval(n)
		if err == nil {
			t.Errorf("%s was accepted and answered %s, want a refusal", src, format.Text(v, nil))
			continue
		}
		if !strings.Contains(err.Error(), "plain") {
			t.Errorf("%s was refused with %q, which does not say a plain number is needed", src, err)
		}
	}
	for _, c := range [][2]string{
		{"gcd(48, 18)", "6"},
		{"lcm(4, 6)", "12"},
		{"isprime(7)", "1"},
		{"200000 mod 7", "3"},
		{"gcd(2 lakh, 4)", "4"},
		{"isprime(2 lakh)", "0"},
		{"(2 lakh) mod 7", "3"},
	} {
		if got := format.Text(mustEval(t, env, c[0]), nil); got != c[1] {
			t.Errorf("%s = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestPlainNumberRejectsAScaleNoSymbolCarries(t *testing.T) {
	scaled := value.Value{Num: num.NumExact(num.FromInt(3)), Factor: num.FromInt(1000)}
	if plainNumber(scaled) {
		t.Error("a value with no symbol but a scale of 1000 passed as a plain number")
	}
	shifted := value.Value{Num: num.NumExact(num.FromInt(3)), Factor: num.FromInt(1), Offset: num.FromInt(273)}
	if plainNumber(shifted) {
		t.Error("a value with no symbol but an offset passed as a plain number")
	}
	if !plainNumber(value.Scalar(num.FromInt(3))) {
		t.Error("a plain 3 was refused as a plain number")
	}
}

func TestANameInUnitPositionIsAUnit(t *testing.T) {
	for _, c := range [][2]string{
		{"60 km/h", "50/3 m/s"},
		{"60 km/h to m/s", "50/3 m/s"},
		{"60 km/hour", "50/3 m/s"},
		{"9.8 m/s^2", "49/5 m/s^2"},
		{"1 J/(mol·K)", "1 J/(mol·K)"},
		{"100 km/h * 2 h", "200000 m"},
		{"(60 km/h) / (2 h)", "1/432 m/s^2"},
		{"2 * (60 km/h)", "100/3 m/s"},
		{"60 km/h + 10 km/h", "175/9 m/s"},
		{"3 kW·h", "10800000 m^2·kg/s^2"},
		{"km/h", "5/18 m/s"},
		{"60 km/h to mile/h", "156250/4191 mile/h"},
		{"h/km", "18/5 s/m"},
		{"h/s^2", "3600 1/s"},
		{"2 mile/hour", "2794/3125 m/s"},
	} {
		if got := run(t, NewEnv(), c[0]); got != c[1] {
			t.Errorf("%s = %s, want %s", c[0], got, c[1])
		}
	}
}

func TestAnUnambiguousUnitPositionSaysNothing(t *testing.T) {
	for _, src := range []string{
		"60 km/h", "9.8 m/s^2", "1 J/(mol·K)", "60 km/hour", "100 km/h * 2 h",
		"(60 km/h) / (2 h)", "h/km", "3 kW·h", "60 km/h to m/s",
	} {
		env := NewEnv()
		run(t, env, src)
		if notes := env.Notes(); len(notes) != 0 {
			t.Errorf("%s left notes %v, want none: no reasonable reader meant the constant", src, notes)
		}
	}
}

func TestABoundNameInUnitPositionSaysWhichWayItWent(t *testing.T) {
	plain := NewEnv()
	run(t, plain, "h = 3")
	run(t, plain, "5 h")
	assertOneNote(t, plain, "5 h after h = 3", "read as the unit h (time), not the variable h; the variable is a plain number")

	measured := NewEnv()
	run(t, measured, "h = 12.3 ± 0.2 m")
	run(t, measured, "2h")
	assertOneNote(t, measured, "2h after a measured h", "read as the variable h, not the unit h (time); the variable carries a measured uncertainty")

	carrying := NewEnv()
	run(t, carrying, "h = 3 m")
	run(t, carrying, "60 km/h")
	assertOneNote(t, carrying, "60 km/h after h = 3 m", "read as the variable h, not the unit h (time); the variable already carries a unit")

	divided := NewEnv()
	run(t, divided, "h = 3")
	run(t, divided, "60 km/h")
	assertOneNote(t, divided, "60 km/h after h = 3", "read as the unit h (time), not the variable h; the variable is a plain number")

	param := NewEnv()
	run(t, param, "g(h) = 2 m / h")
	run(t, param, "g(4)")
	assertOneNote(t, param, "a parameter named like a unit", "read as the unit h (time), not the variable h; the variable is a plain number")
}

func assertOneNote(t *testing.T, env *Env, what, want string) {
	t.Helper()
	notes := env.Notes()
	if len(notes) != 1 {
		t.Errorf("%s left %d notes %v, want one", what, len(notes), notes)
		return
	}
	if notes[0].Msg != want {
		t.Errorf("%s says %q, want %q", what, notes[0].Msg, want)
	}
}

func TestTheRuledNameReadingsSurviveUnitPosition(t *testing.T) {
	shadowed := NewEnv()
	run(t, shadowed, "h = 3")
	if got := run(t, shadowed, "5 h"); got != "5 h" {
		t.Errorf("after h = 3, 5 h = %s, want 5 h", got)
	}
	measured := NewEnv()
	run(t, measured, "h = 12.3 ± 0.2 m")
	if got := run(t, measured, "2h"); got != "123/5 m" {
		t.Errorf("after a measured h, 2h = %s, want 123/5 m", got)
	}
	if got := run(t, measured, "(3 km) / (2 h)"); got != "5000/41" {
		t.Errorf("(3 km) / (2 h) with a measured h = %s, want 5000/41, the variable reading", got)
	}
	bare := NewEnv()
	if got := run(t, bare, "h"); got != "132521403/200000000000000000000000000000000000000000 J·s" {
		t.Errorf("bare h = %s, want Planck's constant", got)
	}
	if got := run(t, bare, "g0"); got != "196133/20000 m/s^2" {
		t.Errorf("g0 = %s, want standard gravity", got)
	}
	if got := run(t, bare, "c"); got != "299792458 m/s" {
		t.Errorf("c = %s, want the speed of light", got)
	}
	if got := run(t, bare, "1/h"); !strings.HasSuffix(got, "1/(J·s)") {
		t.Errorf("1/h = %s, want Planck's constant inverted: 1 carries no unit", got)
	}
}

func TestAVariableCarriesItsUnitExpressionThroughAConversion(t *testing.T) {
	env := NewEnv()
	run(t, env, "x = 60 km/h")
	if got := run(t, env, "x to m/s"); got != "50/3 m/s" {
		t.Errorf("x to m/s = %s, want 50/3 m/s", got)
	}
	if got := run(t, env, "x to km/h"); got != "60 km/h" {
		t.Errorf("x to km/h = %s, want 60 km/h", got)
	}
}

func TestAConversionTargetMayBeAUnitExpression(t *testing.T) {
	for _, c := range [][2]string{
		{"5 m/s to km/h", "18 km/h"},
		{"1 m^2 to cm^2", "10000 cm^2"},
		{"1 J/(mol·K) to J/(mol·K)", "1 J/(mol·K)"},
		{"1 N·m to J", "1 J"},
		{"90 minute to hour", "3/2 hour"},
		{"1 m/s^2 to km/h^2", "12960 km/h^2"},
		{"2 m·kg/s^2 to N", "2 N"},
	} {
		if got := run(t, NewEnv(), c[0]); got != c[1] {
			t.Errorf("%s = %s, want %s", c[0], got, c[1])
		}
	}
}

func TestAConversionTargetThatIsNotAUnitIsRefused(t *testing.T) {
	for _, c := range [][2]string{
		{"5 m to 2", "convert to what unit?"},
		{"5 m to (2 + 3)", "convert to what unit?"},
		{"5 m to sqrt(2)", "convert to what unit?"},
		{"2 km/h to bogus/s", "unknown unit bogus/s"},
		{"2 km/h to m/", "unexpected end of input"},
		{"5 m to m^2.5", "convert to what unit?"},
	} {
		n, _, err := parse.Parse(c[0])
		if err != nil {
			if !strings.Contains(err.Error(), c[1]) {
				t.Errorf("%s failed with %q, want %q", c[0], err, c[1])
			}
			continue
		}
		if _, err := NewEnv().Eval(n); err == nil || !strings.Contains(err.Error(), c[1]) {
			t.Errorf("%s gave error %v, want %q", c[0], err, c[1])
		}
	}
}

func compoundSymbols() []string {
	var out []string
	for _, top := range []string{"m", "km", "J", "N", "g", "mile", "W", "mol", "cm", "kJ"} {
		for _, bottom := range []string{"s", "h", "min", "K", "day", "hour", "second"} {
			for _, shape := range []string{
				"%[1]s/%[2]s", "%[1]s·%[2]s", "%[1]s/%[2]s^2", "%[1]s^2/%[2]s",
				"%[1]s/(%[2]s·K)", "%[1]s^2/%[2]s^2", "%[1]s/%[2]s/%[2]s",
				"%[1]s·%[2]s/(mol·K)", "%[1]s/(%[2]s^3)",
			} {
				out = append(out, fmt.Sprintf(shape, top, bottom))
			}
		}
	}
	return out
}

func TestEveryCompoundUnitReadsAsTheUnitItSpells(t *testing.T) {
	prec := DefaultCtx().Prec
	checked := 0
	for _, sym := range compoundSymbols() {
		want, ok := units.Lookup(sym)
		if !ok {
			t.Fatalf("the unit table cannot read %q, so the sweep is built wrong", sym)
		}
		v := mustEval(t, NewEnv(), "1 "+sym)
		checked++
		if v.Dim != want.Dim {
			t.Errorf("1 %s has dimension %v, want %v", sym, v.Dim, want.Dim)
			continue
		}
		si := v.SI(prec)
		if !si.IsExact() || si.Exact().Cmp(want.Factor) != 0 {
			t.Errorf("1 %s is %v in SI, want %v", sym, si, want.Factor)
		}
		if v.Unit == "" {
			if !v.Dim.IsNone() {
				t.Errorf("1 %s printed no unit at all, though it holds %v", sym, v.Dim)
			}
			continue
		}
		back, ok := units.Lookup(v.Unit)
		if !ok {
			t.Errorf("1 %s printed unit %q, which this engine cannot read back", sym, v.Unit)
			continue
		}
		if back.Dim != v.Dim || back.Factor.Cmp(v.FactorOrOne()) != 0 || back.Offset.Cmp(v.Offset) != 0 {
			t.Errorf("1 %s printed unit %q, which reads back as %v scale %v, want %v scale %v",
				sym, v.Unit, back.Dim, back.Factor, v.Dim, v.FactorOrOne())
		}
	}
	if checked < 400 {
		t.Fatalf("only %d compound units were checked", checked)
	}
}

func TestEveryCompoundUnitIsAConversionTarget(t *testing.T) {
	for _, sym := range compoundSymbols() {
		want, _ := units.Lookup(sym)
		v := mustEval(t, NewEnv(), "1 "+sym+" to "+sym)
		if !v.Num.IsExact() || v.Num.Exact().Cmp(num.FromInt(1)) != 0 {
			t.Errorf("1 %s to %s = %v, want 1", sym, sym, v.Num)
		}
		back, ok := units.Lookup(v.Unit)
		if !ok {
			t.Errorf("1 %s to %s printed unit %q, which this engine cannot read back", sym, sym, v.Unit)
			continue
		}
		if back.Dim != want.Dim || back.Factor.Cmp(want.Factor) != 0 {
			t.Errorf("1 %s to %s landed in %q (%v scale %v), want %v scale %v",
				sym, sym, v.Unit, back.Dim, back.Factor, want.Dim, want.Factor)
		}
	}
}

func TestTheShadowReadsAUnitPositionTheSameWay(t *testing.T) {
	for _, src := range []string{
		"60 km/h", "60 km/h to m/s", "9.8 m/s^2", "100 km/h * 2 h",
		"(60 km/h) / (2 h)", "h/km", "3 kW·h", "1 J/(mol·K)", "60 km/hour",
	} {
		env := NewEnv()
		n := mustParse(t, src)
		v, err := env.Eval(n)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if w := env.Warning(n, v); w != "" {
			t.Errorf("%s: the float64 shadow disagrees: %s", src, w)
		}
	}
}

func TestDefiningAFunctionAnswersWithItsDefinition(t *testing.T) {
	env := NewEnv()
	v, err := env.Eval(mustParse(t, "f(x) = x^2 + 1"))
	if err != nil {
		t.Fatal(err)
	}
	if !v.IsDefinition() || v.Def != "f(x)" {
		t.Errorf("Def = %q, IsDefinition = %v", v.Def, v.IsDefinition())
	}
	out, err := env.Eval(mustParse(t, "f(3)"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Num.Exact().String() != "10" {
		t.Errorf("f(3) = %s, want 10", out.Num.Exact().String())
	}
}

func TestUnsetForgetsNamesAndFunctions(t *testing.T) {
	env := NewEnv()
	if _, err := env.Eval(mustParse(t, "h = 3")); err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(mustParse(t, "f(x) = x")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"h", "f"} {
		if !env.Unset(name) {
			t.Errorf("Unset(%q) found nothing", name)
		}
		if env.Unset(name) {
			t.Errorf("Unset(%q) found it twice", name)
		}
	}
	v, err := env.Eval(mustParse(t, "h"))
	if err != nil {
		t.Fatal(err)
	}
	if v.Unit != "J·s" {
		t.Errorf("h = %v %q, want the Planck constant", v.Num, v.Unit)
	}
	if _, err := env.Eval(mustParse(t, "f(1)")); err == nil {
		t.Error("the function survived unset")
	}
}

func TestAnExactDomainFailureStatesTheFactAndABallHedges(t *testing.T) {
	certain := map[string]string{
		"sqrt(-1)":     "square root of a negative number",
		"sqrt(-1.5)":   "square root of a negative number",
		"sqrt(0 - 4)":  "square root of a negative number",
		"sqrt(-1 m^2)": "square root of a negative number",
		"ln(0)":        "log of zero",
		"log(0)":       "log of zero",
		"log10(0)":     "log of zero",
		"log2(-2)":     "log of a negative number",
		"ln(-2)":       "log of a negative number",
	}
	for src, want := range certain {
		_, err := answer(t, src)
		if err == nil {
			t.Errorf("%s was answered", src)
			continue
		}
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s = %q, want %q", src, err, want)
		}
		if strings.Contains(err.Error(), "may ") {
			t.Errorf("%s hedged about an exact value: %q", src, err)
		}
	}
	hedged := []string{"ln(sqrt(2) - sqrt(2))", "sqrt(sin(pi))", "sqrt(0 - sin(pi))"}
	for _, src := range hedged {
		_, err := answer(t, src)
		if err == nil || !strings.Contains(err.Error(), "may ") {
			t.Errorf("%s = %v, want the hedge a ball earns", src, err)
		}
	}
}

func TestAnUncertaintyMayBeGivenAsATemperatureDifference(t *testing.T) {
	v, err := answer(t, "20 degC ± 0.5 K")
	if err != nil {
		t.Fatalf("20 degC ± 0.5 K: %v", err)
	}
	if got := format.Text(v, nil); got != "20.00 ± 0.50 degC" {
		t.Errorf("= %q, want 20.00 ± 0.50 degC", got)
	}
	converted, err := answer(t, "(20 degC ± 0.5 K) to K")
	if err != nil {
		t.Fatal(err)
	}
	if got := format.Text(converted, nil); got != "293.15 ± 0.50 K" {
		t.Errorf("to K = %q, want 293.15 ± 0.50 K", got)
	}
	_, err = answer(t, "20 degC ± 0.5 degC")
	if err == nil {
		t.Fatal("an absolute temperature was accepted as an uncertainty")
	}
	if !strings.Contains(err.Error(), "0.5 K") || !strings.Contains(err.Error(), "degC") {
		t.Errorf("= %q, want the right operand blamed with a difference to use instead", err)
	}
}

func TestNamedArgumentsAreAcceptedWhereTheSpecSpellsThem(t *testing.T) {
	spelt, err := answer(t, "expand(12.3 ± 0.2, k=2)")
	if err != nil {
		t.Fatalf("expand(x, k=2): %v", err)
	}
	positional, err := answer(t, "expand(12.3 ± 0.2, 2)")
	if err != nil {
		t.Fatal(err)
	}
	if format.Text(spelt, nil) != format.Text(positional, nil) {
		t.Errorf("k=2 gives %q, 2 gives %q", format.Text(spelt, nil), format.Text(positional, nil))
	}
	for _, src := range []string{"expand(12.3 ± 0.2, j=2)", "sqrt(k=4)", "expand(k=2, 12.3 ± 0.2)"} {
		if _, err := answer(t, src); err == nil {
			t.Errorf("%s was accepted", src)
		}
	}
}

func TestOfIsTheMultiplicationItIs(t *testing.T) {
	for src, want := range map[string]string{
		"18% of 500": "90",
		"15% of 80":  "12",
		"50% of 2 m": "1 m",
		"10% of 10%": "0.01",
	} {
		v, err := answer(t, src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if got := format.Text(v, nil); got != want {
			t.Errorf("%s = %q, want %q", src, got, want)
		}
	}
}

func TestASmallProvenMagnitudeIsWrittenInScientificForm(t *testing.T) {
	for _, src := range []string{"sin(1) * 1e-30", "sin(1e400) * 1e-3000", "pi * 1e-40", "exp(-100)", "-pi * 1e-40"} {
		v, err := answer(t, src)
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		got := format.Text(v, nil)
		for _, line := range strings.Split(got, "\n") {
			if len(line) > 80 {
				t.Errorf("%s wrote a %d-character line", src, len(line))
			}
		}
		if !strings.Contains(got, "e-") {
			t.Errorf("%s = %q, want scientific form", src, got)
		}
	}
	for _, c := range [][2]string{{"sqrt(2)", "1.41421356237"}, {"sin(1)", "0.841470984808"}, {"exp(1) * 1e-5", "0.0000271828182846"}} {
		v, err := answer(t, c[0])
		if err != nil {
			t.Fatal(err)
		}
		if got := format.Text(v, nil); got != c[1] {
			t.Errorf("%s = %q, want %q", c[0], got, c[1])
		}
	}
}

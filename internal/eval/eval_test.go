package eval

import (
	"math/big"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
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

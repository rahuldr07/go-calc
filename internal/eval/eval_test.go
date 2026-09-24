package eval

import (
	"testing"

	"github.com/rahuldr07/go-calc/internal/parse"
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

func TestIrrationalIsRefusedNotApproximated(t *testing.T) {
	env := NewEnv()
	n, _, _ := parse.Parse("sqrt(2)")
	if _, err := env.Eval(n); err == nil {
		t.Fatal("sqrt(2) must be refused until ball arithmetic lands")
	}
	n, _, _ = parse.Parse("sin(1)")
	if _, err := env.Eval(n); err == nil {
		t.Fatal("sin(1) must be refused until ball arithmetic lands")
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

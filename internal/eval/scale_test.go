package eval

import (
	"strings"
	"testing"
	"time"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const refusalBudget = 2 * time.Second

var tooLarge = []string{
	"2^10000000.10",
	"2^1000000.5",
	"2^-10000000.1",
	"1e300^100000",
	"1e300^-100000",
	"(1e300)^(100000)",
	"1e-300^100000",
	"(1e300 m) ^ 100000",
	"exp(1000000)",
	"exp(7000000)",
	"ln(exp(1000000))",
	"10^100000 * 10^100000",
	"1e50000 * 1e50000 * 1e50000",
	"1/(10^100000 * 10^100000)",
	"-(10^100000 * 10^100000)",
	"1e50000 / 1e-50000 / 1e50000 * 1e100000",
	"abs(1e300^100000)",
	"floor(exp(1000000))",
	"round(2^1000000.5)",
	"sqrt(1e300^100000)",
	"sin(pi) * 10^9999 * 10^9999",
	"(exp(sqrt(2)) - exp(sqrt(2))) * 10^9999 * 10^9999",
	"R^8000",
	"10^100000",
	"10^-100000",
	"20000!",
	"exp(100000)",
	"1e40000 + 0.5",
	"2^40000",
}

func answer(t *testing.T, src string) (value.Value, error) {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	return NewEnv().Eval(n)
}

func TestAMagnitudeTooLargeToWriteIsRefusedHoweverItIsReached(t *testing.T) {
	for _, src := range tooLarge {
		_, err := answer(t, src)
		if err == nil {
			t.Errorf("%s was answered instead of refused", src)
			continue
		}
		if !strings.Contains(err.Error(), TooLargeMessage) {
			t.Errorf("%s was refused with %q", src, err)
		}
	}
}

func TestAMagnitudeTooLargeToWriteIsRefusedInsideAFunctionBody(t *testing.T) {
	env := NewEnv()
	def, _, err := parse.Parse("f(x) = x^100000")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(def); err != nil {
		t.Fatalf("defining f: %v", err)
	}
	call, _, err := parse.Parse("f(1e300)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(call); err == nil || !strings.Contains(err.Error(), TooLargeMessage) {
		t.Errorf("f(1e300) was refused with %v", err)
	}
	small, _, err := parse.Parse("f(1)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.Eval(small); err != nil {
		t.Errorf("f(1) was refused with %v", err)
	}
}

func TestARefusalCostsAlmostNothing(t *testing.T) {
	start := time.Now()
	for _, src := range tooLarge {
		if _, err := answer(t, src); err == nil {
			t.Fatalf("%s was answered instead of refused", src)
		}
	}
	if spent := time.Since(start); spent > refusalBudget {
		t.Errorf("refusing %d magnitudes took %v, over the %v budget", len(tooLarge), spent, refusalBudget)
	}
}

func TestPrintingARefusedMagnitudeIsNeverAttempted(t *testing.T) {
	start := time.Now()
	for _, src := range tooLarge {
		v, err := answer(t, src)
		if err != nil {
			continue
		}
		_ = format.Default.Text(v, nil)
	}
	if spent := time.Since(start); spent > refusalBudget {
		t.Errorf("printing what was refused took %v, over the %v budget", spent, refusalBudget)
	}
}

func TestWhatItCanStillWriteItStillAnswers(t *testing.T) {
	start := time.Now()
	for _, src := range []string{
		"10^9999",
		"2^20000",
		"1000!",
		"R^500",
		"km^100000",
		"m^8000",
		"1e200 * 1e200",
		"10^9999 / 10^9999",
		"60 km/h to m/s",
		"sqrt(2)",
		"sqrt(4)",
		"root(8, 3)",
		"1/3",
		"0.1 + 0.2",
	} {
		v, err := answer(t, src)
		if err != nil {
			t.Errorf("%s was refused: %v", src, err)
			continue
		}
		if format.Default.Text(v, nil) == "" {
			t.Errorf("%s printed nothing", src)
		}
	}
	if spent := time.Since(start); spent > refusalBudget {
		t.Errorf("what it can still write took %v, over the %v budget", spent, refusalBudget)
	}
}

func TestTheBoundIsOnWhatWouldBeWrittenNotOnTheExponent(t *testing.T) {
	if _, err := answer(t, "10^9999"); err != nil {
		t.Errorf("a 10000-digit answer was refused: %v", err)
	}
	if _, err := answer(t, "10^10001"); err == nil {
		t.Error("a 10002-digit answer was allowed")
	}
	if _, err := answer(t, "2^20000"); err != nil {
		t.Errorf("2^20000 is only 6021 digits and was refused: %v", err)
	}
	if _, err := answer(t, "km^100000"); err != nil {
		t.Errorf("a unit raised to a power writes one digit and was refused: %v", err)
	}
}

func TestAnExactRootIsNotSearchedForBeyondThePrecisionCap(t *testing.T) {
	start := time.Now()
	for _, src := range []string{"sqrt(1e5000)", "sqrt(1e9998)", "root(1e5000, 3)"} {
		v, err := answer(t, src)
		if err != nil {
			t.Errorf("%s was refused: %v", src, err)
			continue
		}
		if v.Num.IsExact() {
			t.Errorf("%s searched for an exact root after all", src)
		}
	}
	if spent := time.Since(start); spent > refusalBudget {
		t.Errorf("rooting wide numbers took %v, over the %v budget", spent, refusalBudget)
	}
	for _, src := range []string{"sqrt(4)", "sqrt(1/4)", "root(8, 3)", "sqrt(1e2000)"} {
		v, err := answer(t, src)
		if err != nil {
			t.Fatalf("%s was refused: %v", src, err)
		}
		if !v.Num.IsExact() {
			t.Errorf("%s gave up on an exact root it could have found", src)
		}
	}
}

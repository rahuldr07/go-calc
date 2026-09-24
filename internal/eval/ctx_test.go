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

func TestProvenDigitsForFamousValue(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 30
	n, _, _ := parse.Parse("exp(pi * sqrt(163))")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	text, digits, capped := format.Default.NumberBall(v.Num.Ball(ctx.Prec))
	if capped {
		t.Fatalf("hit the precision cap for 30 digits")
	}
	if digits < 30 {
		t.Fatalf("only %d digits guaranteed, want 30", digits)
	}
	if !strings.HasPrefix(text, "262537412640768743.99999999999925007") {
		t.Fatalf("got %s", text)
	}
}

func TestRetryStopsAtTheCapAndSaysSo(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 400
	ctx.MaxPrec = 160
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if ctx.Prec > ctx.MaxPrec {
		t.Fatalf("precision %d ran past the cap %d", ctx.Prec, ctx.MaxPrec)
	}
	text, digits, capped := format.Style{Digits: ctx.Digits}.NumberBall(v.Num.Ball(ctx.Prec))
	if !capped {
		t.Fatalf("400 digits from a 160 bit cap should be capped, got %d digits", digits)
	}
	if !strings.HasPrefix(text, "1.4142135623730950488016887242096980785") {
		t.Fatalf("got %s", text)
	}
}

func TestRetryRaisesPrecisionUntilTheDigitsAreThere(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 200
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if ctx.Prec <= 120 {
		t.Fatalf("precision stayed at %d, want the retry loop to raise it", ctx.Prec)
	}
	if got := format.GuaranteedDigits(v.Num.Ball(ctx.Prec)); got < 200 {
		t.Fatalf("guaranteed %d digits, want 200", got)
	}
}

func TestRetryStopsWhenAnInputCarriesUncertainty(t *testing.T) {
	env := NewEnv()
	env.Set("w", wobbly())
	ctx := DefaultCtx()
	ctx.Digits = 40
	n, _, _ := parse.Parse("w * 2")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if ctx.Prec != 120 {
		t.Fatalf("precision moved to %d, but retrying cannot sharpen an uncertain input", ctx.Prec)
	}
	if got := format.GuaranteedDigits(v.Num.Ball(ctx.Prec)); got > 6 {
		t.Fatalf("guaranteed %d digits from an input good to 1e-6", got)
	}
}

func TestExactTierSurvivesTheRetryLoop(t *testing.T) {
	env := NewEnv()
	for _, c := range []struct{ src, want string }{
		{"sqrt(16/9)", "4/3"},
		{"0.1 + 0.2", "3/10"},
		{"(1e16 + 1) - 1e16", "1"},
		{"2 ^ 10", "1024"},
	} {
		n, _, err := parse.Parse(c.src)
		if err != nil {
			t.Fatalf("parse %q: %v", c.src, err)
		}
		v, err := env.EvalCtx(n, DefaultCtx())
		if err != nil {
			t.Fatalf("eval %q: %v", c.src, err)
		}
		if !v.Num.IsExact() {
			t.Fatalf("%s left the exact tier", c.src)
		}
		if got := v.Num.String(); got != c.want {
			t.Errorf("%s = %s, want %s", c.src, got, c.want)
		}
	}
}

func TestDegreesModeTurnsTheCircle(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Degrees = true
	n, _, _ := parse.Parse("sin(30)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	b := v.Num.Ball(ctx.Prec)
	if !b.Contains(big.NewFloat(0.5)) {
		t.Fatalf("sin(30 deg) = %s, want a ball around 1/2", b)
	}
}

func wobbly() value.Value {
	mid := new(big.Float).SetPrec(120).SetFloat64(1.25)
	return value.ScalarNum(num.NumBall(num.NewBall(mid, num.MagFromFloat(1e-6))))
}

func TestIrrationalConstantsResolveAndCanBeShadowed(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	for _, c := range []struct {
		src  string
		want string
	}{
		{"pi", "3.14159265358979323846264338327950288419716939937510582097494"},
		{"e", "2.71828182845904523536028747135266249775724709369995957496697"},
	} {
		n, _, err := parse.Parse(c.src)
		if err != nil {
			t.Fatalf("parse %q: %v", c.src, err)
		}
		v, err := env.EvalCtx(n, ctx)
		if err != nil {
			t.Fatalf("eval %q: %v", c.src, err)
		}
		want, _, err := big.ParseFloat(c.want, 10, 300, big.ToNearestEven)
		if err != nil {
			t.Fatal(err)
		}
		if !v.Num.Ball(ctx.Prec).Contains(want) {
			t.Errorf("%s = %s, want a ball around %s", c.src, v.Num, c.want)
		}
	}
	shadow := NewEnv()
	n, _, _ := parse.Parse("pi = 3")
	if _, err := shadow.EvalCtx(n, DefaultCtx()); err != nil {
		t.Fatalf("assign: %v", err)
	}
	n, _, _ = parse.Parse("pi")
	v, err := shadow.EvalCtx(n, DefaultCtx())
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := v.Num.String(); got != "3" {
		t.Errorf("shadowed pi = %s, want 3", got)
	}
}

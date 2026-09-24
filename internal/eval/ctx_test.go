package eval

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func TestProvenDigitsForFamousValue(t *testing.T) {
	env := NewEnv()
	ctx := provingCtx(t, 35)
	n, _, _ := parse.Parse("exp(pi * sqrt(163))")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	mustProve(t, v, "exp(pi * sqrt(163))")
	if ctx.Prec <= DefaultCtx().Prec {
		t.Fatalf("precision stayed at %d, but 35 digits need more than the default", ctx.Prec)
	}
	text, digits, capped := format.Default.NumberBall(v.Num.Ball(ctx.Prec))
	if capped {
		t.Fatalf("hit the precision cap for 35 digits")
	}
	if digits < 35 {
		t.Fatalf("only %d digits guaranteed, want 35", digits)
	}
	if !strings.HasPrefix(text, "262537412640768743.99999999999925007") {
		t.Fatalf("got %s", text)
	}
}

func TestTwelveDigitsNeedNoRetry(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	n, _, _ := parse.Parse("exp(pi * sqrt(163))")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if ctx.Prec != DefaultCtx().Prec {
		t.Fatalf("precision moved to %d, but the default precision already proves 12 digits", ctx.Prec)
	}
	text, digits, capped := format.Default.NumberBall(v.Num.Ball(ctx.Prec))
	if capped || digits < 12 {
		t.Fatalf("%d digits, capped=%v, want at least 12 uncapped", digits, capped)
	}
	if !strings.HasPrefix(text, "262537412640768743.9999") {
		t.Fatalf("got %s", text)
	}
	if got := format.Default.Number(v); got != "2.62537412641e+17" {
		t.Fatalf("styled number = %s, want 12 significant digits", got)
	}
}

func TestRetryStopsAtTheCapAndSaysSo(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 400
	ctx.MaxPrec = 160
	ctx.Deadline = time.Now().Add(time.Hour)
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
	ctx := provingCtx(t, 200)
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	mustProve(t, v, "sqrt(2)")
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

func TestProvableZeroIsPrintedAsZeroNotAsARange(t *testing.T) {
	env := NewEnv()
	ctx := provingCtx(t, 30)
	n, _, _ := parse.Parse("sqrt(2) - sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v.Stopped == value.StopDeadline {
		t.Fatal("the retry loop hit its deadline, so this run proves nothing")
	}
	b := v.Num.Ball(ctx.Prec)
	if !b.Contains(new(big.Float)) {
		t.Fatalf("ball %s should contain zero", b)
	}
	text := format.Default.Text(v, nil)
	if !strings.HasPrefix(text, "0\n  zero to ") {
		t.Fatalf("text = %q, want a bare zero and a note saying how far it is proven", text)
	}
}

func TestCapReachedIsNamedOnlyWhenItIs(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 400
	ctx.MaxPrec = 160
	ctx.Deadline = time.Now().Add(time.Hour)
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if v.Stopped != value.StopPrecisionCap {
		t.Fatalf("stop reason = %d, want StopPrecisionCap", v.Stopped)
	}
	if !strings.Contains(format.Style{Digits: 400}.Text(v, nil), "digits guaranteed (precision cap reached)") {
		t.Fatalf("text = %q, want the cap named", format.Style{Digits: 400}.Text(v, nil))
	}
}

func TestExactTierHonoursTheSameDigitBudget(t *testing.T) {
	env := NewEnv()
	n, _, err := parse.Parse("18,50,000 / 12")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	v, err := env.EvalCtx(n, DefaultCtx())
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if got := format.Text(v, nil); got != "462500/3  \u2248 154166.666667" {
		t.Fatalf("text = %q, want 12 significant digits after the fraction", got)
	}
}

func TestDeadlineStopsTheRetryAndIsNamedAsItself(t *testing.T) {
	env := NewEnv()
	ctx := DefaultCtx()
	ctx.Digits = 400
	ctx.Deadline = time.Now().Add(-time.Second)
	n, _, _ := parse.Parse("sqrt(2)")
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if ctx.Prec != DefaultCtx().Prec {
		t.Fatalf("precision moved to %d, but the deadline had already passed", ctx.Prec)
	}
	if v.Stopped != value.StopDeadline {
		t.Fatalf("stop reason = %d, want StopDeadline", v.Stopped)
	}
	text := format.Style{Digits: ctx.Digits}.Text(v, nil)
	if !strings.Contains(text, "digits guaranteed (time limit reached)") {
		t.Fatalf("text = %q, want the deadline named", text)
	}
	if strings.Contains(text, "precision cap") {
		t.Fatalf("text = %q, names a cap that was never reached", text)
	}
}

func bound(t *testing.T, lit string) *big.Float {
	t.Helper()
	f, _, err := big.ParseFloat(lit, 10, 300, big.ToNearestEven)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCancellationDoesNotCollapseToZero(t *testing.T) {
	cases := []struct {
		src       string
		low, high string
	}{
		{"exp(1e-80) - 1", "0.999e-80", "1.001e-80"},
		{"1 - cos(1e-40)", "4.99e-81", "5.01e-81"},
		{"log(1 + 1e-80)", "0.999e-80", "1.001e-80"},
	}
	for _, c := range cases {
		n, _, err := parse.Parse(c.src)
		if err != nil {
			t.Fatalf("parse %q: %v", c.src, err)
		}
		ctx := provingCtx(t, DefaultCtx().Digits)
		v, err := NewEnv().EvalCtx(n, ctx)
		if err != nil {
			t.Fatalf("eval %q: %v", c.src, err)
		}
		mustProve(t, v, c.src)
		b := v.Num.Ball(ctx.Prec)
		if text := format.Default.Text(v, nil); strings.HasPrefix(text, "0\n") || text == "0" {
			t.Fatalf("%s printed %q, but the true value is near %s", c.src, text, c.low)
		}
		if b.Lo().Sign() <= 0 {
			t.Errorf("%s: lower bound %s is not provably positive", c.src, b.Lo().Text('g', 6))
		}
		if b.Lo().Cmp(bound(t, c.low)) < 0 || b.Hi().Cmp(bound(t, c.high)) > 0 {
			t.Errorf("%s: ball [%s .. %s] escapes the Taylor window [%s .. %s]",
				c.src, b.Lo().Text('g', 6), b.Hi().Text('g', 6), c.low, c.high)
		}
		if got := format.GuaranteedDigits(b); got < ctx.Digits {
			t.Errorf("%s: only %d digits guaranteed, want %d", c.src, got, ctx.Digits)
		}
	}
}

func TestHugeTrigArgumentStillConverges(t *testing.T) {
	n, _, _ := parse.Parse("sin(1e400)")
	ctx := provingCtx(t, DefaultCtx().Digits)
	v, err := NewEnv().EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	mustProve(t, v, "sin(1e400)")
	b := v.Num.Ball(ctx.Prec)
	if got := format.GuaranteedDigits(b); got < ctx.Digits {
		t.Fatalf("only %d digits guaranteed for sin(1e400), want %d", got, ctx.Digits)
	}
	if b.Lo().Cmp(big.NewFloat(-1)) < 0 || b.Hi().Cmp(big.NewFloat(1)) > 0 {
		t.Fatalf("sin(1e400) = %s, which leaves [-1, 1]", b)
	}
}

func TestRootDegreeIsBounded(t *testing.T) {
	for _, src := range []string{"root(4, 1e30)", "root(4, 0)", "root(4, -2)"} {
		n, _, err := parse.Parse(src)
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		if _, err := NewEnv().EvalCtx(n, DefaultCtx()); err == nil {
			t.Errorf("%s was accepted, want a refusal", src)
		}
	}
}

func TestWideBallFromAnUncertainInputIsNotPrintedAsZero(t *testing.T) {
	env := NewEnv()
	seed, _, _ := parse.Parse("x = sqrt(2)")
	if _, err := env.EvalCtx(seed, DefaultCtx()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	n, _, err := parse.Parse("(x - 1.4142135623730950488016887242096980785696718753769) * 1e36")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ctx := DefaultCtx()
	v, err := env.EvalCtx(n, ctx)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	b := v.Num.Ball(ctx.Prec)
	if !b.ContainsZero() {
		t.Fatalf("ball %s no longer straddles zero, the case under test has drifted", b)
	}
	if b.Rad.Float().Cmp(big.NewFloat(1)) < 0 {
		t.Fatalf("ball %s is not wide, the case under test has drifted", b)
	}
	text := format.Default.Text(v, nil)
	if strings.HasPrefix(text, "0\n") || text == "0" {
		t.Fatalf("text = %q, but the true value is anywhere in [%s .. %s]",
			text, b.Lo().Text('g', 4), b.Hi().Text('g', 4))
	}
	if !strings.HasPrefix(text, "0 ± ") {
		t.Fatalf("text = %q, want the midpoint and radius", text)
	}
}

func provingCtx(t *testing.T, digits int) *Ctx {
	t.Helper()
	ctx := DefaultCtx()
	ctx.Digits = digits
	ctx.Deadline = time.Now().Add(time.Hour)
	return ctx
}

func mustProve(t *testing.T, v value.Value, src string) {
	t.Helper()
	switch v.Stopped {
	case value.StopDeadline:
		t.Fatalf("%s: the retry loop hit its deadline, so this run proves nothing about the digits", src)
	case value.StopPrecisionCap:
		t.Fatalf("%s: the retry loop hit the precision cap, so this run proves nothing about the digits", src)
	case value.StopUncertainInput:
		t.Fatalf("%s: an input carried uncertainty, so the retry loop never ran to completion", src)
	}
}

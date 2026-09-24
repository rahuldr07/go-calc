package eval

import (
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	typicalExpression = "2 * sin(pi/4) + sqrt(2) / 3"
	deepIrrational    = "exp(pi * sqrt(163))"
	provablyZero      = "sqrt(2) - sqrt(2)"
	tooWideToWrite    = "exp(1000000)"
)

var benchText string

func benchLine(b *testing.B, src string, ctx func() *Ctx, check func(*testing.B, *Ctx, value.Value)) {
	b.Helper()
	first := true
	b.ReportAllocs()
	for b.Loop() {
		n, notes, err := parse.Parse(src)
		if err != nil {
			b.Fatalf("Parse(%q): %v", src, err)
		}
		c := ctx()
		v, err := NewEnv().EvalCtx(n, c)
		if err != nil {
			b.Fatalf("Eval(%q): %v", src, err)
		}
		if first {
			check(b, c, v)
			first = false
		}
		benchText = format.Text(v, notes)
	}
}

func provesAtLeast(want int) func(*testing.B, *Ctx, value.Value) {
	return func(b *testing.B, c *Ctx, v value.Value) {
		if v.Stopped != value.StopNone {
			b.Fatalf("stopped early: %v", v.Stopped)
		}
		if got := v.Num.Ball(c.Prec).GuaranteedDigits(); got < want {
			b.Fatalf("proved %d digits, want at least %d", got, want)
		}
	}
}

func BenchmarkTypicalExpression(b *testing.B) {
	benchLine(b, typicalExpression, DefaultCtx, provesAtLeast(12))
}

func BenchmarkThousandDigits(b *testing.B) {
	benchLine(b, deepIrrational, func() *Ctx {
		return &Ctx{Prec: 120, MaxPrec: 1 << 16, Digits: 1000}
	}, provesAtLeast(1000))
}

func BenchmarkProvablyZeroToMaxPrec(b *testing.B) {
	benchLine(b, provablyZero, DefaultCtx, func(b *testing.B, c *Ctx, v value.Value) {
		if v.Stopped != value.StopPrecisionCap {
			b.Fatalf("%s stopped with %v, want the precision cap", provablyZero, v.Stopped)
		}
		if c.Prec != DefaultCtx().MaxPrec {
			b.Fatalf("%s stopped at %d bits, want %d", provablyZero, c.Prec, DefaultCtx().MaxPrec)
		}
	})
}

func BenchmarkMeasuredWithBudget(b *testing.B) {
	setup := []string{"h = 12.3 ± 0.2 m", "w = 4.5 ± 0.1 m", "d = 2.25 ± 0.05"}
	const target = "h * w / sqrt(d)"
	b.ReportAllocs()
	for b.Loop() {
		env := NewEnv()
		for _, src := range setup {
			n, _, err := parse.Parse(src)
			if err != nil {
				b.Fatalf("Parse(%q): %v", src, err)
			}
			if _, err := env.Eval(n); err != nil {
				b.Fatalf("Eval(%q): %v", src, err)
			}
		}
		n, notes, err := parse.Parse(target)
		if err != nil {
			b.Fatalf("Parse(%q): %v", target, err)
		}
		v, err := env.Eval(n)
		if err != nil {
			b.Fatalf("Eval(%q): %v", target, err)
		}
		if len(v.Unc.Budget) != len(setup) {
			b.Fatalf("%s reported %d contributions, want %d", target, len(v.Unc.Budget), len(setup))
		}
		benchText = format.Text(v, notes)
	}
}

func BenchmarkRefusedTooWide(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		n, _, err := parse.Parse(tooWideToWrite)
		if err != nil {
			b.Fatalf("Parse(%q): %v", tooWideToWrite, err)
		}
		v, err := NewEnv().EvalCtx(n, DefaultCtx())
		if err == nil {
			b.Fatalf("%s answered %v instead of being refused", tooWideToWrite, v)
		}
		benchText = err.Error()
	}
}

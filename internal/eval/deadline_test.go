package eval

import (
	"testing"
	"time"

	"github.com/rahuldr07/go-calc/internal/parse"
)

func evalWithin(t *testing.T, src string, limit time.Duration) (time.Duration, error) {
	t.Helper()
	n, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q): %v", src, err)
	}
	ctx := DefaultCtx()
	ctx.Deadline = time.Now().Add(limit)
	start := time.Now()
	_, err = NewEnv().EvalCtx(n, ctx)
	return time.Since(start), err
}

func TestDeadlineStopsLongSolve(t *testing.T) {
	elapsed, err := evalWithin(t, "solve(exp(sin(x)*cos(x)*x)-exp(1), x, -1000, 1000)", 50*time.Millisecond)
	if !IsDeadline(err) {
		t.Fatalf("err = %v, want a deadline error (took %v)", err, elapsed)
	}
	if elapsed > time.Second {
		t.Fatalf("stopped after %v, want well under a second", elapsed)
	}
}

func TestDeadlineLeavesFastWorkAlone(t *testing.T) {
	_, err := evalWithin(t, "2 * sin(pi/4) + sqrt(2) / 3", time.Minute)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestDeadlineStopsLongSolveAcrossLimits(t *testing.T) {
	for _, limit := range []time.Duration{20 * time.Millisecond, 100 * time.Millisecond, 500 * time.Millisecond} {
		elapsed, err := evalWithin(t, "solve(exp(sin(x)*cos(x)*x)-exp(1), x, -1000, 1000)", limit)
		t.Logf("limit %v: stopped after %v, err = %v", limit, elapsed, err)
		if !IsDeadline(err) {
			t.Fatalf("limit %v: err = %v", limit, err)
		}
	}
}

func benchDeadline(b *testing.B, src string, limit time.Duration) {
	n, _, err := parse.Parse(src)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		ctx := DefaultCtx()
		if limit > 0 {
			ctx.Deadline = time.Now().Add(limit)
		}
		if _, err := NewEnv().EvalCtx(n, ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTypicalWithoutDeadline(b *testing.B) { benchDeadline(b, typicalExpression, 0) }

func BenchmarkTypicalWithDeadline(b *testing.B) { benchDeadline(b, typicalExpression, time.Hour) }

func BenchmarkSumWithoutDeadline(b *testing.B) { benchDeadline(b, "sum(1..20000)", 0) }

func BenchmarkSumWithDeadline(b *testing.B) { benchDeadline(b, "sum(1..20000)", time.Hour) }

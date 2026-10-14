package gocalc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEvalReportsHowFarTheAnswerCanBeTrusted(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		expr   string
		text   string
		status Status
	}{
		{"1/3 + 1/6", "0.5", Exact},
		{"0.1 + 0.2", "0.3", Exact},
		{"2^0.5", "1.41421356237", Proven},
		{"(12.4 +- 0.2 cm) * (7.1 +- 0.1 cm)", "0.00880 ± 0.00019 m^2", Estimate},
		{"isprime(97)", "true", Exact},
		{"isprime(2^89 - 1)", "probably true", Estimate},
	} {
		r, err := Eval(ctx, c.expr)
		if err != nil {
			t.Fatalf("%s: %v", c.expr, err)
		}
		if r.Text != c.text || r.Status != c.status {
			t.Errorf("%s = %q %s, want %q %s", c.expr, r.Text, r.Status, c.text, c.status)
		}
	}
}

func TestEvalCarriesTheExactValueBoundsAndFloat64(t *testing.T) {
	r, err := Eval(context.Background(), "0.1 + 0.2")
	if err != nil {
		t.Fatal(err)
	}
	if r.Exact != "3/10" || r.Float64 != "0.30000000000000004" || !r.Trusted() {
		t.Errorf("got %+v", r)
	}
	p, err := Eval(context.Background(), "pi", Digits(30))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(p.Text, "3.14159265358979323846264338328") || !strings.HasPrefix(p.Bounds, "[3.14159") {
		t.Errorf("pi at 30 digits = %q bounds %q", p.Text, p.Bounds)
	}
}

func TestARefusalIsAnErrorWithItsPlace(t *testing.T) {
	r, err := Eval(context.Background(), "1 m + 2 s")
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want a *Refusal", err)
	}
	if r.Status != Refused || refusal.Line != 1 || !strings.Contains(refusal.Reason, "cannot add length") {
		t.Errorf("refusal = %+v, status %s", refusal, r.Status)
	}
	if strings.HasPrefix(refusal.Reason, "1:") {
		t.Errorf("the reason repeats its position: %q", refusal.Reason)
	}
}

func TestASheetLinksItsLines(t *testing.T) {
	sheet, err := EvalSheet(context.Background(), []string{"r = 2", "area = pi * r^2", "area in m^2", "1/0"})
	if err != nil {
		t.Fatal(err)
	}
	if got := sheet.Results[1].Text; got != "12.5663706144" {
		t.Errorf("area = %q", got)
	}
	if len(sheet.Results) != 4 || sheet.Results[1].Source != "area = pi * r^2" {
		t.Errorf("results = %+v", sheet.Results)
	}
	if refused := sheet.Refusals(); len(refused) != 2 || refused[1].Line != 4 {
		t.Errorf("refusals = %+v", refused)
	}
}

func TestOptionsAreChecked(t *testing.T) {
	ctx := context.Background()
	if _, err := Eval(ctx, "1", Digits(0)); !errors.Is(err, ErrBadDigits) {
		t.Errorf("Digits(0) gave %v", err)
	}
	if _, err := EvalSheet(ctx, nil); !errors.Is(err, ErrNoLines) {
		t.Errorf("no lines gave %v", err)
	}
	deg, err := Eval(ctx, "sin(30)", Degrees())
	if err != nil || deg.Text != "0.5" {
		t.Errorf("sin(30) in degrees = %q, %v", deg.Text, err)
	}
}

func TestACancelledContextStopsWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Eval(ctx, "1 + 1"); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context gave %v", err)
	}
	short, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	start := time.Now()
	_, err := Eval(short, "sum(1..100000)^2 * 3^400000 / 7^300000")
	if time.Since(start) > 3*time.Second {
		t.Errorf("a 50ms deadline took %v", time.Since(start))
	}
	if err == nil {
		return
	}
	var refusal *Refusal
	if !errors.As(err, &refusal) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a deadline gave %v", err)
	}
}

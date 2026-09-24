package format

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

func TestNeverPrintsUnprovenDigits(t *testing.T) {
	const prec = 64
	b := num.NewBall(big.NewFloat(1.0/3.0), num.MagFromFloat(1e-6))
	text, digits, _ := Default.NumberBall(b)
	if digits > 6 {
		t.Fatalf("claimed %d digits from a ball of radius 1e-6: %s", digits, text)
	}
}

func TestRoundingBoundaryTerminates(t *testing.T) {
	done := make(chan struct{})
	go func() {
		b := num.NewBall(big.NewFloat(0.125), num.MagFromFloat(1e-30))
		Default.NumberBall(b)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("digit printer did not terminate on a rounding boundary")
	}
}

func TestRoundingBoundaryDropsADigitRatherThanLying(t *testing.T) {
	b := num.NewBall(big.NewFloat(0.125), num.MagFromFloat(1e-30))
	text, digits, _ := Default.NumberBall(b)
	if !strings.HasPrefix(text, "0.125") {
		t.Fatalf("text = %q, want it to start with 0.125", text)
	}
	if digits < 20 || digits > 28 {
		t.Fatalf("digits = %d, want a count dropped below the 29 the radius alone allows", digits)
	}
}

func TestPrintedDigitsMatchAKnownExpansion(t *testing.T) {
	const sqrt2 = "1.41421356237309504880168872420969807856967187537694"
	mid, _, err := big.ParseFloat(sqrt2, 10, 200, big.ToNearestEven)
	if err != nil {
		t.Fatal(err)
	}
	for _, radius := range []float64{1e-5, 1e-12, 1e-25} {
		b := num.NewBall(mid, num.MagFromFloat(radius))
		text, digits, _ := Style{Digits: 60}.NumberBall(b)
		if digits < 4 {
			t.Fatalf("radius %g: only %d digits", radius, digits)
		}
		want := truncateSignificant(sqrt2, digits-1)
		if !strings.HasPrefix(text, want) {
			t.Fatalf("radius %g: printed %q, want it to start with %q", radius, text, want)
		}
	}
}

func truncateSignificant(decimal string, digits int) string {
	seen := 0
	for i := 0; i < len(decimal); i++ {
		if decimal[i] >= '0' && decimal[i] <= '9' {
			seen++
		}
		if seen == digits {
			return decimal[:i+1]
		}
	}
	return decimal
}

func TestCappedIsReportedWhenTheBallCannotJustifyTheStyle(t *testing.T) {
	b := num.NewBall(big.NewFloat(2.5), num.MagFromFloat(1e-3))
	text, digits, capped := Default.NumberBall(b)
	if !capped {
		t.Fatalf("want capped for %q with %d digits", text, digits)
	}
	if digits >= Default.Digits {
		t.Fatalf("digits = %d, want fewer than %d", digits, Default.Digits)
	}
}

func TestGuaranteedDigitsIsRelativeAccuracy(t *testing.T) {
	cases := []struct {
		mid    float64
		radius float64
		want   int
	}{
		{1, 2e-10, 9},
		{1000, 2e-10, 12},
		{0.001, 2e-10, 6},
		{0.5, 0.1, 0},
		{2, 4, 0},
	}
	for _, c := range cases {
		got := GuaranteedDigits(num.NewBall(big.NewFloat(c.mid), num.MagFromFloat(c.radius)))
		if got != c.want {
			t.Errorf("GuaranteedDigits(%g ± %g) = %d, want %d", c.mid, c.radius, got, c.want)
		}
	}
}

func TestTextNamesWhyTheDigitsRanOut(t *testing.T) {
	cases := []struct {
		stop value.Stop
		want string
	}{
		{value.StopPrecisionCap, "3 digits guaranteed (precision cap reached)"},
		{value.StopNoProgress, "3 digits guaranteed (more precision did not help)"},
		{value.StopUncertainInput, "3 digits guaranteed (an input carries uncertainty)"},
		{value.StopNone, "3 digits guaranteed"},
	}
	for _, c := range cases {
		v := value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(2.5), num.MagFromFloat(1e-3))))
		v.Stopped = c.stop
		got := Text(v, nil)
		if !strings.HasPrefix(got, "2.5\n  ") {
			t.Fatalf("Text = %q, want it to open with 2.5 and a note", got)
		}
		if strings.TrimPrefix(got, "2.5\n  ") != c.want {
			t.Errorf("note for stop %d = %q, want %q", c.stop, strings.TrimPrefix(got, "2.5\n  "), c.want)
		}
	}
}

func TestProvablyZeroPrintsZeroAndSaysHowFar(t *testing.T) {
	v := value.ScalarNum(num.NumBall(num.NewBall(new(big.Float), num.MagFromFloat(1e-30))))
	v.Stopped = value.StopNoProgress
	got := Text(v, nil)
	if !strings.HasPrefix(got, "0\n  ") {
		t.Fatalf("Text = %q, want a bare 0 and a note", got)
	}
	note := strings.TrimPrefix(got, "0\n  ")
	if note != "zero to 29 decimal places, the precision available" {
		t.Errorf("note = %q, want it to say how far the zero is proven", note)
	}
	if strings.Contains(got, "precision cap") {
		t.Errorf("note = %q, must not name a cap that was never reached", got)
	}
}

func TestTextStaysQuietWhenEveryRequestedDigitIsProven(t *testing.T) {
	v := value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(2.5), num.MagFromFloat(1e-30))))
	if got := Text(v, nil); got != "2.5" {
		t.Fatalf("Text = %q, want a bare 2.5", got)
	}
}

func TestNonTerminatingRationalsShowSignificantDigits(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"462500/3", "462500/3  \u2248 154166.666667"},
		{"1/3", "1/3  \u2248 0.333333333333"},
		{"-2/3", "-2/3  \u2248 -0.666666666667"},
		{"1/300000", "1/300000  \u2248 0.00000333333333333"},
	}
	for _, c := range cases {
		r, err := num.ParseExact(c.src)
		if err != nil {
			t.Fatal(err)
		}
		if got := Text(value.Scalar(r), nil); got != c.want {
			t.Errorf("Text(%s) = %q, want %q", c.src, got, c.want)
		}
	}
}

func TestTerminatingRationalsWithinBudgetPrintInFullAndExact(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"3/10", "0.3"},
		{"1/1024", "0.0009765625"},
		{"1/16384", "0.00006103515625"},
	} {
		r, err := num.ParseExact(c.src)
		if err != nil {
			t.Fatal(err)
		}
		got := Text(value.Scalar(r), nil)
		if got != c.want {
			t.Errorf("Text(%s) = %q, want %q", c.src, got, c.want)
		}
		if strings.Contains(got, "\u2248") {
			t.Errorf("Text(%s) = %q, a complete decimal must not be marked approximate", c.src, got)
		}
	}
}

func TestLongTerminatingDecimalsRoundToTheBudget(t *testing.T) {
	tiny := value.Scalar(num.FromRat(new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(2), big.NewInt(500), nil))))
	got := Text(tiny, nil)
	marker := "  \u2248 "
	i := strings.Index(got, marker)
	if i < 0 {
		t.Fatalf("Text = %q, want a rounded decimal marked approximate", got)
	}
	decimal := got[i+len(marker):]
	if want := "0." + strings.Repeat("0", 150) + "30549363635"; decimal != want {
		t.Fatalf("decimal = %q, want %q", decimal, want)
	}
	if n := significantDigits(decimal); n > Digits {
		t.Fatalf("decimal carries %d significant digits, want at most %d", n, Digits)
	}
	if !strings.HasPrefix(got, "1/") {
		t.Fatalf("Text = %q, want the exact fraction beside the decimal", got[:20])
	}
}

func TestDefaultBudgetIsTwelveSignificantDigitsInBothTiers(t *testing.T) {
	r, err := num.ParseExact("462500/3")
	if err != nil {
		t.Fatal(err)
	}
	exact := strings.TrimPrefix(Text(value.Scalar(r), nil), "462500/3  \u2248 ")
	if got := significantDigits(exact); got != Digits {
		t.Errorf("exact tier printed %d significant digits (%s), want %d", got, exact, Digits)
	}
	b := num.NewBall(big.NewFloat(154166.666666666666), num.MagFromFloat(1e-30))
	if got := significantDigits(Default.Number(value.ScalarNum(num.NumBall(b)))); got != Digits {
		t.Errorf("ball tier printed %d significant digits, want %d", got, Digits)
	}
}

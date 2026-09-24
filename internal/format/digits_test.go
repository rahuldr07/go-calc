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
			t.Fatalf("radius %g: only %d places", radius, digits)
		}
		want := truncateDecimals(sqrt2, digits-1)
		if !strings.HasPrefix(text, want) {
			t.Fatalf("radius %g: printed %q, want it to start with %q", radius, text, want)
		}
	}
}

func truncateDecimals(decimal string, places int) string {
	dot := strings.IndexByte(decimal, '.')
	if dot < 0 || dot+1+places > len(decimal) {
		return decimal
	}
	if places <= 0 {
		return decimal[:dot]
	}
	return decimal[:dot+1+places]
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

func TestGuaranteedDigitsTracksTheRadius(t *testing.T) {
	cases := []struct {
		mid    float64
		radius float64
		want   int
	}{
		{1, 2e-10, 9},
		{1000, 2e-10, 9},
		{0.001, 2e-10, 9},
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

func TestTextSaysWhenTheBallRanOutOfDigits(t *testing.T) {
	v := value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(2.5), num.MagFromFloat(1e-3))))
	got := Text(v, nil)
	if !strings.HasPrefix(got, "2.5") {
		t.Fatalf("Text = %q, want it to open with 2.5", got)
	}
	if !strings.Contains(got, "2 digits guaranteed (precision cap reached)") {
		t.Fatalf("Text = %q, want it to say how many digits are guaranteed", got)
	}
}

func TestTextStaysQuietWhenEveryRequestedDigitIsProven(t *testing.T) {
	v := value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(2.5), num.MagFromFloat(1e-30))))
	if got := Text(v, nil); got != "2.5" {
		t.Fatalf("Text = %q, want a bare 2.5", got)
	}
}

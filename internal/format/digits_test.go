package format

import (
	"math/big"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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
	v.Stopped = value.StopPrecisionCap
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
		{"1/300000", "1/300000  \u2248 3.33333333333e-06"},
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
	if want := "3.0549363635e-151"; decimal != want {
		t.Fatalf("decimal = %q, want %q", decimal, want)
	}
	mantissa, _, _ := strings.Cut(decimal, "e")
	if n := significantDigits(mantissa); n > Digits {
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
	b := num.BallFromExact(r, 256)
	if got := significantDigits(Default.Number(value.ScalarNum(num.NumBall(b)))); got != Digits {
		t.Errorf("ball tier printed %d significant digits, want %d", got, Digits)
	}
}

func TestWideBallsStraddlingZeroAreNotCollapsedToZero(t *testing.T) {
	cases := []struct {
		mid    float64
		radius float64
		want   string
	}{
		{0.5, 1, "[-0.5, 1.5]"},
		{0.05, 0.25, "[-0.2, 0.3001]"},
		{0, 0.4, "[-0.4001, 0.4001]"},
		{-0.5, 1, "[-1.5, 0.5]"},
	}
	for _, c := range cases {
		b := num.NewBall(big.NewFloat(c.mid), num.MagFromFloat(c.radius))
		text, digits, _ := Default.NumberBall(b)
		if text == "0" {
			t.Errorf("ball %g ± %g printed as 0, but the true value is anywhere in [%s .. %s]",
				c.mid, c.radius, b.Lo().Text('g', 4), b.Hi().Text('g', 4))
		}
		if text != c.want {
			t.Errorf("ball %g ± %g printed %q, want %q", c.mid, c.radius, text, c.want)
		}
		if strings.Contains(text, "±") {
			t.Errorf("ball %g ± %g printed %q, but ± is reserved for statistical uncertainty", c.mid, c.radius, text)
		}
		if lo, hi, ok := strings.Cut(strings.Trim(text, "[]"), ", "); !ok {
			t.Errorf("ball %g ± %g printed %q, which names no interval", c.mid, c.radius, text)
		} else {
			low, _ := new(big.Rat).SetString(lo)
			high, _ := new(big.Rat).SetString(hi)
			trueLo, _ := b.Lo().Rat(nil)
			trueHi, _ := b.Hi().Rat(nil)
			if low == nil || high == nil || low.Cmp(trueLo) > 0 || high.Cmp(trueHi) < 0 {
				t.Errorf("ball %g ± %g printed %q, which is narrower than the proven range [%s .. %s]",
					c.mid, c.radius, text, b.Lo().Text('g', 12), b.Hi().Text('g', 12))
			}
		}
		if digits != 0 {
			t.Errorf("ball %g ± %g claimed %d digits", c.mid, c.radius, digits)
		}
	}
}

func TestZeroIsCollapsedOnlyWhenItIsProven(t *testing.T) {
	proven := num.NewBall(new(big.Float), num.MagFromFloat(0.004))
	if text, _, _ := Default.NumberBall(proven); text != "0" {
		t.Errorf("0 ± 0.004 printed %q, want 0", text)
	}
	unproven := num.NewBall(new(big.Float), num.MagFromFloat(0.4))
	if text, _, _ := Default.NumberBall(unproven); text == "0" {
		t.Errorf("0 ± 0.4 printed as 0, but no decimal place is settled")
	}
}

func TestGroupingLeavesNonDecimalTextAlone(t *testing.T) {
	styles := []Style{{Digits: 12, Grouped: true}, {Digits: 12, Grouped: true, Indian: true}}
	balls := []num.Ball{
		num.NewBall(big.NewFloat(2e20), num.MagFromFloat(1e20)),
		num.NewBall(big.NewFloat(2e21), num.MagFromFloat(1e5)),
		num.NewBall(big.NewFloat(0.5), num.MagFromFloat(1)),
	}
	for _, s := range styles {
		for _, b := range balls {
			plain, _, _ := Style{Digits: 12}.NumberBall(b)
			grouped, _, _ := s.NumberBall(b)
			if plainDecimal(plain) {
				continue
			}
			if grouped != plain {
				t.Errorf("grouping rewrote %q as %q", plain, grouped)
			}
			if !utf8.ValidString(grouped) {
				t.Errorf("grouping produced invalid UTF-8: %q", grouped)
			}
		}
	}
}

func TestGroupingStillGroupsRealDecimals(t *testing.T) {
	b := num.NewBall(big.NewFloat(1850000.5), num.MagFromFloat(1e-9))
	if got, _, _ := (Style{Digits: 12, Grouped: true}).NumberBall(b); got != "1,850,000.5" {
		t.Errorf("grouped ball = %q, want 1,850,000.5", got)
	}
	if got, _, _ := (Style{Digits: 12, Grouped: true, Indian: true}).NumberBall(b); got != "18,50,000.5" {
		t.Errorf("Indian grouped ball = %q, want 18,50,000.5", got)
	}
}

func TestOneDigitReadsAsSingular(t *testing.T) {
	v := value.ScalarNum(num.NumBall(num.NewBall(big.NewFloat(2), num.MagFromFloat(0.15))))
	v.Stopped = value.StopPrecisionCap
	got := Text(v, nil)
	if !strings.Contains(got, "1 digit guaranteed") || strings.Contains(got, "1 digits") {
		t.Errorf("Text = %q, want the singular form", got)
	}
	zero := value.ScalarNum(num.NumBall(num.NewBall(new(big.Float), num.MagFromFloat(0.04))))
	if got := Text(zero, nil); !strings.Contains(got, "zero to 1 decimal place,") {
		t.Errorf("Text = %q, want the singular decimal place", got)
	}
}

func TestZeroNoteHoldsForEveryPointInTheBall(t *testing.T) {
	balls := []num.Ball{
		num.NewBall(new(big.Float), num.MagFromFloat(1e-30)),
		num.NewBall(new(big.Float), num.MagFromFloat(0.004)),
		num.NewBall(big.NewFloat(0.04), num.MagFromFloat(0.004)),
		num.NewBall(big.NewFloat(-0.0300926553811), num.MagFromFloat(0.0300927)),
		num.NewBall(big.NewFloat(0.5), num.MagFromFloat(1)),
		num.NewBall(big.NewFloat(-0.009), num.MagFromFloat(0.01)),
	}
	for _, b := range balls {
		v := value.ScalarNum(num.NumBall(b))
		note := Default.precisionNote(v)
		if !strings.HasPrefix(note, "zero to ") {
			continue
		}
		places := b.GuaranteedPlaces()
		for _, edge := range []*big.Float{b.Lo(), b.Hi()} {
			if rendered := unsignZero(edge.Text('f', places)); strings.Trim(rendered, "0.") != "" {
				t.Errorf("ball %s claims %q but its edge %s reads %s at %d places",
					b, note, edge.Text('g', 6), rendered, places)
			}
		}
	}
}

func TestLopsidedBallAroundZeroIsNotCalledZero(t *testing.T) {
	b := num.NewBall(big.NewFloat(-0.0300926553811), num.MagFromFloat(0.0300927))
	if !b.ContainsZero() {
		t.Fatal("the case under test has drifted: the ball no longer straddles zero")
	}
	if got := b.GuaranteedPlaces(); got != 0 {
		t.Errorf("GuaranteedPlaces = %d, but the ball reaches %s, which is not 0 to any decimal place",
			got, b.Lo().Text('g', 6))
	}
	text, digits, _ := Default.NumberBall(b)
	if text == "0" || digits != 0 {
		t.Errorf("printed %q with %d digits, want the midpoint and radius", text, digits)
	}
	v := value.ScalarNum(num.NumBall(b))
	if note := Default.precisionNote(v); strings.HasPrefix(note, "zero to ") {
		t.Errorf("note = %q, but the ball reaches %s", note, b.Lo().Text('g', 6))
	}
}

func TestSmallMagnitudesAreWrittenInScientificForm(t *testing.T) {
	cases := []struct{ src, want string }{
		{"1/16384", "0.00006103515625"},
		{"1/100000", "0.00001"},
		{"1/1000000", "1e-06"},
		{"662607015/1000000000000000000000000000000000000000000", "6.62607015e-34"},
		{"-1/1000000", "-1e-06"},
		{"1/10", "0.1"},
		{"0", "0"},
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

func TestNoAnswerIsWiderThanAScreenful(t *testing.T) {
	wide := []string{
		"662607015/1000000000000000000000000000000000000000000",
		"1/1000000000000000000000000000000000000000000000000",
		"-1/1000000000000000000000000000000000000000000000000",
	}
	for _, src := range wide {
		r, err := num.ParseExact(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(Text(value.Scalar(r), nil), "\n") {
			if len(line) > 80 {
				t.Errorf("Text(%s) wrote a %d-character line: %s", src, len(line), line)
			}
		}
	}
}

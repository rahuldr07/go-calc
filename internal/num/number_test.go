package num

import (
	"math/big"
	"testing"
)

const numberPrec = 64

func TestExactDivisionStaysExact(t *testing.T) {
	got, err := NumExact(FromInt(2)).Div(NumExact(FromInt(4)), numberPrec)
	if err != nil {
		t.Fatalf("Div: %v", err)
	}
	if !got.IsExact() {
		t.Fatalf("2/4 fell to kind %d, want exact", got.Kind())
	}
	if s := got.Exact().String(); s != "1/2" {
		t.Errorf("2/4 = %s, want 1/2", s)
	}
}

func TestExactChainStaysExact(t *testing.T) {
	a := NumExact(FromInt(7))
	b := NumExact(FromInt(3))
	sum := a.Add(b, numberPrec).Mul(b, numberPrec).Sub(a, numberPrec)
	if !sum.IsExact() {
		t.Fatalf("exact chain fell to kind %d", sum.Kind())
	}
	if s := sum.Exact().String(); s != "23" {
		t.Errorf("(7+3)*3-7 = %s, want 23", s)
	}
}

func TestBallInfectsExact(t *testing.T) {
	ball := NumBall(NewBall(big.NewFloat(0.5), MagFromFloat(1e-9)))
	got := ball.Add(NumExact(FromInt(1)), numberPrec)
	if got.IsExact() {
		t.Fatal("ball + exact stayed exact")
	}
	if got.Kind() != KindBall {
		t.Fatalf("ball + exact gave kind %d, want KindBall", got.Kind())
	}
	if !got.Ball(numberPrec).Contains(big.NewFloat(1.5)) {
		t.Errorf("ball + exact = %s, does not contain 1.5", got)
	}
	back := NumExact(FromInt(1)).Add(ball, numberPrec)
	if back.Kind() != KindBall {
		t.Fatalf("exact + ball gave kind %d, want KindBall", back.Kind())
	}
}

func TestNumberPowNegativeExponent(t *testing.T) {
	got, err := NumExact(FromInt(2)).Pow(-3, numberPrec)
	if err != nil {
		t.Fatalf("Pow: %v", err)
	}
	if !got.IsExact() || got.Exact().String() != "1/8" {
		t.Errorf("2^-3 = %s, want exact 1/8", got)
	}
	if _, err := NumExact(FromInt(0)).Pow(-1, numberPrec); err == nil {
		t.Error("0^-1 returned no error")
	}
	third, _ := ParseExact("1/3")
	ball := NumBall(BallFromExact(third, numberPrec))
	powed, err := ball.Pow(-2, numberPrec)
	if err != nil {
		t.Fatalf("ball Pow: %v", err)
	}
	if powed.Kind() != KindBall {
		t.Fatalf("ball Pow gave kind %d, want KindBall", powed.Kind())
	}
	if !powed.Ball(numberPrec).Contains(new(big.Float).SetPrec(200).SetInt64(9)) {
		t.Errorf("(1/3)^-2 = %s, does not contain 9", powed)
	}
	zeroBall := NumBall(NewBall(big.NewFloat(0), MagFromFloat(1)))
	if _, err := zeroBall.Pow(-1, numberPrec); err == nil {
		t.Error("straddling ball to a negative power returned no error")
	}
}

func TestNumberIntervalBall(t *testing.T) {
	lo, _ := ParseExact("0.9")
	hi, _ := ParseExact("1.1")
	n := NumInterval(hi, lo)
	if n.Kind() != KindInterval {
		t.Fatalf("NumInterval gave kind %d", n.Kind())
	}
	b := n.Ball(numberPrec)
	for _, v := range []string{"0.9", "1.0", "1.1"} {
		f, _, err := big.ParseFloat(v, 10, 200, big.ToNearestEven)
		if err != nil {
			t.Fatal(err)
		}
		if !b.Contains(f) {
			t.Errorf("interval ball %s does not contain %s", b, v)
		}
	}
	if b.Contains(big.NewFloat(1.5)) {
		t.Errorf("interval ball %s wrongly contains 1.5", b)
	}
	sum := n.Add(NumExact(FromInt(1)), numberPrec)
	if sum.Kind() != KindBall {
		t.Fatalf("interval + exact gave kind %d, want KindBall", sum.Kind())
	}
}

func TestBallNegAbsAndBounds(t *testing.T) {
	b := NewBall(big.NewFloat(-2), MagFromFloat(0.5))
	if s, ok := b.Neg().Sign(); !ok || s != 1 {
		t.Errorf("Neg sign = (%d, %v), want (1, true)", s, ok)
	}
	if s, ok := b.Abs().Sign(); !ok || s != 1 {
		t.Errorf("Abs sign = (%d, %v), want (1, true)", s, ok)
	}
	if b.Lo().Cmp(big.NewFloat(-2.5)) > 0 || b.Hi().Cmp(big.NewFloat(-1.5)) < 0 {
		t.Errorf("bounds [%s .. %s] are too tight", b.Lo().Text('g', 6), b.Hi().Text('g', 6))
	}
}

func TestNumberNegAndAbsAcrossKinds(t *testing.T) {
	exact := NumExact(FromInt(-3))
	if got := exact.Neg().String(); got != "3" {
		t.Errorf("exact Neg = %s, want 3", got)
	}
	if got := exact.Abs().String(); got != "3" {
		t.Errorf("exact Abs = %s, want 3", got)
	}
	spanning := NumInterval(FromInt(-2), FromInt(5))
	if got := spanning.Abs().String(); got != "[0 .. 5]" {
		t.Errorf("interval Abs = %s, want [0 .. 5]", got)
	}
	if got := spanning.Neg().String(); got != "[-5 .. 2]" {
		t.Errorf("interval Neg = %s, want [-5 .. 2]", got)
	}
}

func TestGuaranteedDigitsCountsSignificantDigits(t *testing.T) {
	cases := []struct {
		mid    float64
		radius float64
		want   int
	}{
		{1, 2e-10, 9},
		{1000, 2e-10, 12},
		{0.001, 2e-10, 6},
		{1e9, 2e-10, 18},
	}
	for _, c := range cases {
		b := NewBall(big.NewFloat(c.mid), MagFromFloat(c.radius))
		if got := b.GuaranteedDigits(); got != c.want {
			t.Errorf("GuaranteedDigits(%g ± %g) = %d, want %d", c.mid, c.radius, got, c.want)
		}
	}
	wide := NewBall(big.NewFloat(1), MagFromFloat(50))
	if got := wide.GuaranteedDigits(); got != 0 {
		t.Errorf("GuaranteedDigits of a ball wider than its midpoint = %d, want 0", got)
	}
}

func TestGuaranteedPlacesAccountsForTheMidpointOffset(t *testing.T) {
	cases := []struct {
		mid    float64
		radius float64
		want   int
	}{
		{0, 0.004, 2},
		{0.04, 0.004, 1},
		{-0.0300926553811, 0.0300927, 0},
		{0, 0.4, 0},
	}
	for _, c := range cases {
		b := NewBall(big.NewFloat(c.mid), MagFromFloat(c.radius))
		got := b.GuaranteedPlaces()
		if got != c.want {
			t.Errorf("GuaranteedPlaces(%g ± %g) = %d, want %d", c.mid, c.radius, got, c.want)
		}
		if got == 0 {
			continue
		}
		half := new(big.Float).Quo(big.NewFloat(1), new(big.Float).SetPrec(200).SetInt(pow10Int(got)))
		half.Quo(half, big.NewFloat(2))
		for _, edge := range []*big.Float{b.Lo(), b.Hi()} {
			if new(big.Float).Abs(edge).Cmp(half) >= 0 {
				t.Errorf("%g ± %g claims %d places but reaches %s", c.mid, c.radius, got, edge.Text('g', 6))
			}
		}
	}
}

func pow10Int(k int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(k)), nil)
}

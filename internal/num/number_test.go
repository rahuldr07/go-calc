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

func TestNumberNegAbsAndSignAcrossKinds(t *testing.T) {
	exact := NumExact(FromInt(-3))
	if got := exact.Neg().String(); got != "3" {
		t.Errorf("exact Neg = %s, want 3", got)
	}
	if got := exact.Abs().String(); got != "3" {
		t.Errorf("exact Abs = %s, want 3", got)
	}
	if s, ok := exact.Sign(); !ok || s != -1 {
		t.Errorf("exact Sign = (%d, %v), want (-1, true)", s, ok)
	}
	straddling := NumBall(NewBall(big.NewFloat(0.25), MagFromFloat(1)))
	if _, ok := straddling.Sign(); ok {
		t.Error("a ball straddling zero must not claim a sign")
	}
	spanning := NumInterval(FromInt(-2), FromInt(5))
	if _, ok := spanning.Sign(); ok {
		t.Error("an interval spanning zero must not claim a sign")
	}
	if got := spanning.Abs().String(); got != "[0 .. 5]" {
		t.Errorf("interval Abs = %s, want [0 .. 5]", got)
	}
	if got := spanning.Neg().String(); got != "[-5 .. 2]" {
		t.Errorf("interval Neg = %s, want [-5 .. 2]", got)
	}
}

func TestGuaranteedDigitsFollowsTheRadiusNotTheMagnitude(t *testing.T) {
	for _, mid := range []float64{1e-3, 1, 1e9} {
		b := NewBall(big.NewFloat(mid), MagFromFloat(2e-10))
		if got := b.GuaranteedDigits(); got != 9 {
			t.Errorf("GuaranteedDigits(%g ± 2e-10) = %d, want 9", mid, got)
		}
	}
	wide := NewBall(big.NewFloat(1), MagFromFloat(50))
	if got := wide.GuaranteedDigits(); got != 0 {
		t.Errorf("GuaranteedDigits of a ball wider than 1 = %d, want 0", got)
	}
}

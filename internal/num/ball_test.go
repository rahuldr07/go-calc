package num

import (
	"math/big"
	"math/rand"
	"testing"
)

func randExact(rnd *rand.Rand) Exact {
	if rnd.Intn(16) == 0 {
		return FromInt(0)
	}
	num := big.NewInt(rnd.Int63n(1000000) + 1)
	den := big.NewInt(rnd.Int63n(1000000) + 1)
	if rnd.Intn(8) == 0 {
		den = new(big.Int).Lsh(big.NewInt(1), uint(rnd.Intn(20)))
	}
	r := new(big.Rat).SetFrac(num, den)
	scale := rnd.Intn(13) - 6
	ten := new(big.Rat).SetInt64(10)
	shift := new(big.Rat).SetInt64(1)
	for i := 0; i < scale; i++ {
		shift.Mul(shift, ten)
	}
	for i := 0; i > scale; i-- {
		shift.Quo(shift, ten)
	}
	r.Mul(r, shift)
	if rnd.Intn(2) == 0 {
		r.Neg(r)
	}
	return FromRat(r)
}

func TestBallContainsTruth(t *testing.T) {
	const prec = 64
	const hi = 400
	rnd := rand.New(rand.NewSource(7))
	for i := 0; i < 200000; i++ {
		a := randExact(rnd)
		b := randExact(rnd)
		ba := BallFromExact(a, prec)
		bb := BallFromExact(b, prec)
		for _, op := range []struct {
			name  string
			ball  func() (Ball, error)
			truth func() (Exact, error)
		}{
			{"add", func() (Ball, error) { return ba.Add(bb, prec), nil }, func() (Exact, error) { return a.Add(b), nil }},
			{"sub", func() (Ball, error) { return ba.Sub(bb, prec), nil }, func() (Exact, error) { return a.Sub(b), nil }},
			{"mul", func() (Ball, error) { return ba.Mul(bb, prec), nil }, func() (Exact, error) { return a.Mul(b), nil }},
			{"div", func() (Ball, error) { return ba.Div(bb, prec) }, func() (Exact, error) { return a.Div(b) }},
		} {
			got, berr := op.ball()
			want, eerr := op.truth()
			if eerr != nil || berr != nil {
				continue
			}
			exactFloat := new(big.Float).SetPrec(hi).SetRat(want.Rat())
			if !got.Contains(exactFloat) {
				t.Fatalf("%s(%s, %s): ball [%s .. %s] does not contain %s", op.name, a, b, got.Lo().Text('g', 20), got.Hi().Text('g', 20), exactFloat.Text('g', 20))
			}
		}
	}
}

func TestDivByStraddlingBallFails(t *testing.T) {
	const prec = 64
	numerator := BallFromExact(FromInt(1), prec)
	divisor := NewBall(big.NewFloat(0.5), MagFromFloat(1))
	if _, err := numerator.Div(divisor, prec); err == nil {
		t.Fatal("Div by a ball straddling zero returned no error")
	} else if err.Error() != "divisor may be zero" {
		t.Fatalf("Div error = %q, want %q", err.Error(), "divisor may be zero")
	}
	touching := NewBall(big.NewFloat(1), MagFromFloat(1))
	if _, err := numerator.Div(touching, prec); err == nil {
		t.Fatal("Div by a ball whose radius reaches zero returned no error")
	}
	clear := NewBall(big.NewFloat(1), MagFromFloat(0.25))
	if _, err := numerator.Div(clear, prec); err != nil {
		t.Fatalf("Div by a ball clear of zero: %v", err)
	}
}

func TestBallSign(t *testing.T) {
	cases := []struct {
		name    string
		ball    Ball
		sign    int
		decided bool
	}{
		{"positive", NewBall(big.NewFloat(3), MagFromFloat(1)), 1, true},
		{"negative", NewBall(big.NewFloat(-3), MagFromFloat(1)), -1, true},
		{"straddling", NewBall(big.NewFloat(0.5), MagFromFloat(2)), 0, false},
		{"touching", NewBall(big.NewFloat(1), MagFromFloat(1)), 0, false},
		{"exact zero", NewBall(big.NewFloat(0), Mag{}), 0, true},
		{"fuzzy zero", NewBall(big.NewFloat(0), MagFromFloat(1e-30)), 0, false},
	}
	for _, c := range cases {
		sign, decided := c.ball.Sign()
		if sign != c.sign || decided != c.decided {
			t.Errorf("%s: Sign() = (%d, %v), want (%d, %v)", c.name, sign, decided, c.sign, c.decided)
		}
	}
}

func TestBallFromExactKeepsRepresentableValuesTight(t *testing.T) {
	const prec = 64
	dyadic, _ := ParseExact("0.125")
	if rad := BallFromExact(dyadic, prec).Rad; !rad.IsZero() {
		t.Errorf("BallFromExact(0.125) radius = %s, want 0", rad)
	}
	third, _ := ParseExact("1/3")
	if rad := BallFromExact(third, prec).Rad; rad.IsZero() {
		t.Error("BallFromExact(1/3) radius = 0, want a positive radius")
	}
}

func TestMagRoundsUp(t *testing.T) {
	a := MagFromFloat(1)
	b := MagFromFloat(1e-30)
	sum := a.Add(b)
	if sum.Cmp(a) <= 0 {
		t.Errorf("Mag(1).Add(Mag(1e-30)) = %s, want strictly greater than 1", sum)
	}
	if MagFromFloat(-4).Cmp(MagFromFloat(4)) != 0 {
		t.Error("MagFromFloat should ignore sign")
	}
	if !MagFromFloat(0).IsZero() {
		t.Error("MagFromFloat(0) should be zero")
	}
	if got := MagFromFloat(2).MulFloat(big.NewFloat(-3)); got.Cmp(MagFromFloat(6)) != 0 {
		t.Errorf("Mag(2).MulFloat(-3) = %s, want 6", got)
	}
}

func randRadius(rnd *rand.Rand, v Exact) Exact {
	scales := []string{"1e-30", "1e-12", "1e-6", "1e-3", "1/10", "1/2", "9/10", "2"}
	s, err := ParseExact(scales[rnd.Intn(len(scales))])
	if err != nil {
		panic(err)
	}
	if rnd.Intn(4) == 0 {
		return s
	}
	r := s.Mul(v)
	if r.Sign() < 0 {
		r = r.Neg()
	}
	return r
}

func randUnit(rnd *rand.Rand) Exact {
	return FromRat(big.NewRat(rnd.Int63n(2000001)-1000000, 1000000))
}

func widen(b Ball, r Exact) Ball {
	b.Rad = b.Rad.Add(magFromExact(r))
	return b
}

func TestBallContainsPerturbedTruth(t *testing.T) {
	const prec = 64
	const hi = 400
	rnd := rand.New(rand.NewSource(11))
	for i := 0; i < 50000; i++ {
		a := randExact(rnd)
		b := randExact(rnd)
		ra := randRadius(rnd, a)
		rb := randRadius(rnd, b)
		ba := widen(BallFromExact(a, prec), ra)
		bb := widen(BallFromExact(b, prec), rb)
		pa := a.Add(ra.Mul(randUnit(rnd)))
		pb := b.Add(rb.Mul(randUnit(rnd)))
		if !ba.Contains(new(big.Float).SetPrec(hi).SetRat(pa.Rat())) {
			t.Fatalf("widened ball %s does not contain its own point %s", ba, pa)
		}
		if !bb.Contains(new(big.Float).SetPrec(hi).SetRat(pb.Rat())) {
			t.Fatalf("widened ball %s does not contain its own point %s", bb, pb)
		}
		for _, op := range []struct {
			name  string
			ball  func() (Ball, error)
			truth func() (Exact, error)
		}{
			{"add", func() (Ball, error) { return ba.Add(bb, prec), nil }, func() (Exact, error) { return pa.Add(pb), nil }},
			{"sub", func() (Ball, error) { return ba.Sub(bb, prec), nil }, func() (Exact, error) { return pa.Sub(pb), nil }},
			{"mul", func() (Ball, error) { return ba.Mul(bb, prec), nil }, func() (Exact, error) { return pa.Mul(pb), nil }},
			{"div", func() (Ball, error) { return ba.Div(bb, prec) }, func() (Exact, error) { return pa.Div(pb) }},
		} {
			got, berr := op.ball()
			want, eerr := op.truth()
			if eerr != nil || berr != nil {
				continue
			}
			exactFloat := new(big.Float).SetPrec(hi).SetRat(want.Rat())
			if !got.Contains(exactFloat) {
				t.Fatalf("%s(%s±%s, %s±%s): ball [%s .. %s] does not contain %s", op.name, a, ra, b, rb, got.Lo().Text('g', 20), got.Hi().Text('g', 20), exactFloat.Text('g', 20))
			}
		}
	}
}

func TestBallPowContainsTruth(t *testing.T) {
	const prec = 64
	const hi = 400
	rnd := rand.New(rand.NewSource(23))
	for i := 0; i < 20000; i++ {
		a := randExact(rnd)
		ra := randRadius(rnd, a)
		ba := widen(BallFromExact(a, prec), ra)
		pa := a.Add(ra.Mul(randUnit(rnd)))
		k := rnd.Intn(13) - 6
		got, err := ba.Pow(k, prec)
		if err != nil {
			continue
		}
		if k < 0 && pa.Sign() == 0 {
			continue
		}
		want := pa.Pow(k)
		exactFloat := new(big.Float).SetPrec(hi).SetRat(want.Rat())
		if !got.Contains(exactFloat) {
			t.Fatalf("pow(%s±%s, %d): ball [%s .. %s] does not contain %s", a, ra, k, got.Lo().Text('g', 20), got.Hi().Text('g', 20), exactFloat.Text('g', 20))
		}
	}
}

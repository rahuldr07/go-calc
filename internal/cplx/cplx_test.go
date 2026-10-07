package cplx

import (
	"errors"
	"math"
	"math/big"
	"math/cmplx"
	"math/rand"
	"testing"

	"github.com/rahuldr07/go-calc/internal/elem"
	"github.com/rahuldr07/go-calc/internal/num"
)

const prec = 128

func exactNum(n, d int64) num.Number { return num.NumExact(num.FromRat(big.NewRat(n, d))) }

func ci(re, im int64) Complex { return Rect(exactNum(re, 1), exactNum(im, 1)) }

func cq(reN, reD, imN, imD int64) Complex { return Rect(exactNum(reN, reD), exactNum(imN, imD)) }

func ballNum(mid, rad float64) num.Number {
	return num.NumBall(num.NewBall(new(big.Float).SetPrec(prec).SetFloat64(mid), num.MagFromFloat(rad)))
}

func mustOK(t *testing.T) func(Complex, error) Complex {
	return func(c Complex, err error) Complex {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return c
	}
}

func mustNum(t *testing.T) func(num.Number, error) num.Number {
	return func(n num.Number, err error) num.Number {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return n
	}
}

func requireExact(t *testing.T, name string, c Complex, want string) {
	t.Helper()
	if !c.IsExact() {
		t.Fatalf("%s is not exact: %s", name, c)
	}
	if c.String() != want {
		t.Fatalf("%s = %s, want %s", name, c, want)
	}
}

func rationalIn(n num.Number, q *big.Rat) bool {
	if n.IsExact() {
		return n.Exact().Rat().Cmp(q) == 0
	}
	b := n.Ball(prec)
	lo, _ := b.Lo().Rat(nil)
	hi, _ := b.Hi().Rat(nil)
	if lo == nil || hi == nil {
		return true
	}
	return lo.Cmp(q) <= 0 && hi.Cmp(q) >= 0
}

func enclosesZero(n num.Number) bool {
	if n.IsExact() {
		return n.Exact().Sign() == 0
	}
	return n.Ball(prec).ContainsZero()
}

func float64Of(n num.Number) float64 {
	if n.IsExact() {
		f, _ := n.Exact().Rat().Float64()
		return f
	}
	f, _ := n.Ball(prec).Mid.Float64()
	return f
}

func complexOf(c Complex) complex128 { return complex(float64Of(c.re), float64Of(c.im)) }

func requireNear(t *testing.T, name string, got Complex, want complex128) {
	t.Helper()
	diff := cmplx.Abs(complexOf(got) - want)
	if diff > 1e-9*(1+cmplx.Abs(want)) {
		t.Fatalf("%s = %s, want about %v", name, got, want)
	}
}

func TestExactArithmeticStaysExact(t *testing.T) {
	cases := []struct {
		name string
		got  func() (Complex, error)
		want string
	}{
		{"add", func() (Complex, error) { return ci(1, 2).Add(ci(3, 4), prec), nil }, "4+6i"},
		{"sub", func() (Complex, error) { return ci(1, 2).Sub(ci(3, 4), prec), nil }, "-2-2i"},
		{"mul", func() (Complex, error) { return ci(1, 2).Mul(ci(3, 4), prec), nil }, "-5+10i"},
		{"div", func() (Complex, error) { return ci(1, 0).Div(ci(1, 1), prec) }, "1/2-(1/2)i"},
		{"div gaussian", func() (Complex, error) { return ci(-5, 10).Div(ci(3, 4), prec) }, "1+2i"},
		{"div ratio", func() (Complex, error) { return cq(1, 3, 1, 5).Div(cq(2, 7, 0, 1), prec) }, "7/6+(7/10)i"},
		{"neg", func() (Complex, error) { return ci(1, -2).Neg(), nil }, "-1+2i"},
		{"conj", func() (Complex, error) { return ci(3, 4).Conj(), nil }, "3-4i"},
		{"pow i^2", func() (Complex, error) { return ci(0, 1).Pow(2, prec) }, "-1"},
		{"pow i^3", func() (Complex, error) { return ci(0, 1).Pow(3, prec) }, "-i"},
		{"pow i^4", func() (Complex, error) { return ci(0, 1).Pow(4, prec) }, "1"},
		{"pow (1+i)^8", func() (Complex, error) { return ci(1, 1).Pow(8, prec) }, "16"},
		{"pow (1+i)^-2", func() (Complex, error) { return ci(1, 1).Pow(-2, prec) }, "-(1/2)i"},
		{"pow zero exponent", func() (Complex, error) { return ci(3, 4).Pow(0, prec) }, "1"},
		{"pow one exponent", func() (Complex, error) { return ci(3, 4).Pow(1, prec) }, "3+4i"},
		{"pow 2+3i cubed", func() (Complex, error) { return ci(2, 3).Pow(3, prec) }, "-46+9i"},
	}
	for _, tc := range cases {
		got, err := tc.got()
		if err != nil {
			t.Fatalf("%s: unexpected error %v", tc.name, err)
		}
		requireExact(t, tc.name, got, tc.want)
	}
}

func TestExactFieldIdentitiesRandom(t *testing.T) {
	rnd := rand.New(rand.NewSource(11))
	pick := func() Complex {
		return cq(rnd.Int63n(41)-20, rnd.Int63n(9)+1, rnd.Int63n(41)-20, rnd.Int63n(9)+1)
	}
	for i := 0; i < 2000; i++ {
		a, b, c := pick(), pick(), pick()
		left := a.Mul(b.Add(c, prec), prec)
		right := a.Mul(b, prec).Add(a.Mul(c, prec), prec)
		if !left.IsExact() || left.String() != right.String() {
			t.Fatalf("distributivity failed: %s vs %s", left, right)
		}
		if b.re.Exact().Sign() == 0 && b.im.Exact().Sign() == 0 {
			continue
		}
		quotient := mustOK(t)(a.Div(b, prec))
		back := quotient.Mul(b, prec)
		if !back.IsExact() || back.String() != a.String() {
			t.Fatalf("(a/b)*b = %s, want %s", back, a)
		}
		conjProduct := b.Mul(b.Conj(), prec)
		if !conjProduct.IsReal() || !conjProduct.IsExact() {
			t.Fatalf("z*conj(z) is not exactly real: %s", conjProduct)
		}
	}
}

func TestIsReal(t *testing.T) {
	if !Real(exactNum(5, 1)).IsReal() {
		t.Fatal("Real(5) should be real")
	}
	if ci(1, 1).IsReal() {
		t.Fatal("1+i should not be real")
	}
	if Rect(exactNum(1, 1), ballNum(0, 1e-30)).IsReal() {
		t.Fatal("a ball imaginary part is not proven real")
	}
	if !ci(1, 1).Mul(ci(1, -1), prec).IsReal() {
		t.Fatal("(1+i)(1-i) should be exactly real")
	}
}

func TestString(t *testing.T) {
	cases := []struct {
		c    Complex
		want string
	}{
		{ci(3, 0), "3"},
		{ci(0, 2), "2i"},
		{ci(0, 1), "i"},
		{ci(0, -1), "-i"},
		{ci(0, -2), "-2i"},
		{ci(3, 2), "3+2i"},
		{ci(3, -2), "3-2i"},
		{ci(3, 1), "3+i"},
		{ci(3, -1), "3-i"},
		{ci(0, 0), "0"},
		{ci(-3, 0), "-3"},
		{cq(1, 2, 0, 1), "1/2"},
		{cq(1, 2, 3, 4), "1/2+(3/4)i"},
		{cq(-1, 2, -3, 4), "-1/2-(3/4)i"},
		{cq(0, 1, 3, 4), "(3/4)i"},
	}
	for _, tc := range cases {
		if got := tc.c.String(); got != tc.want {
			t.Fatalf("String() = %q, want %q", got, tc.want)
		}
	}
}

func TestDivisionByZeroErrors(t *testing.T) {
	if _, err := ci(1, 1).Div(ci(0, 0), prec); !errors.Is(err, num.ErrDivByZero) {
		t.Fatalf("exact zero divisor: err = %v", err)
	}
	zeroBall := Rect(ballNum(0, 1e-10), ballNum(0, 1e-10))
	if _, err := ci(1, 1).Div(zeroBall, prec); !errors.Is(err, num.ErrDivisorMayBeZero) {
		t.Fatalf("zero-containing ball divisor: err = %v", err)
	}
	ballAndZero := Rect(ballNum(0, 1e-10), exactNum(0, 1))
	if _, err := ci(1, 1).Div(ballAndZero, prec); err == nil {
		t.Fatal("ball containing zero with exact zero imaginary part must not divide")
	}
	if _, err := ci(0, 0).Pow(-1, prec); err == nil {
		t.Fatal("0^-1 must error")
	}
	if _, err := ci(0, 0).Pow(-3, prec); err == nil {
		t.Fatal("0^-3 must error")
	}
	if _, err := Real(ballNum(0, 0.5)).Pow(-1, prec); err == nil {
		t.Fatal("(ball containing zero)^-1 must error")
	}
}

func TestDivisionByBallNotContainingZero(t *testing.T) {
	divisor := Rect(ballNum(0, 1e-10), ballNum(2, 1e-10))
	q, err := ci(4, 0).Div(divisor, prec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rationalIn(q.re, big.NewRat(0, 1)) || !rationalIn(q.im, big.NewRat(-2, 1)) {
		t.Fatalf("4/(2i) = %s should enclose -2i", q)
	}
}

func TestPowExponentTooLarge(t *testing.T) {
	if _, err := ci(1, 1).Pow(math.MaxInt, prec); !errors.Is(err, ErrExponentTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if _, err := ci(1, 1).Pow(math.MinInt, prec); !errors.Is(err, ErrExponentTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestSqrtExactPerfectSquares(t *testing.T) {
	cases := []struct {
		in   Complex
		want string
	}{
		{ci(-1, 0), "i"},
		{ci(-4, 0), "2i"},
		{ci(4, 0), "2"},
		{ci(0, 0), "0"},
		{ci(-9, 0), "3i"},
		{cq(-1, 4, 0, 1), "(1/2)i"},
		{ci(3, 4), "2+i"},
		{ci(3, -4), "2-i"},
		{ci(-3, 4), "1+2i"},
		{ci(-3, -4), "1-2i"},
		{ci(0, 2), "1+i"},
		{ci(-5, 12), "2+3i"},
	}
	for _, tc := range cases {
		got := mustOK(t)(Sqrt(tc.in, prec))
		requireExact(t, "sqrt("+tc.in.String()+")", got, tc.want)
	}
}

func TestSqrtOfTwoIsBallWhoseSquareEnclosesTwo(t *testing.T) {
	root := mustOK(t)(Sqrt(ci(2, 0), prec))
	if root.re.IsExact() {
		t.Fatalf("sqrt(2) real part must be a ball: %s", root)
	}
	if !isExactZero(root.im) {
		t.Fatalf("sqrt(2) imaginary part must be exactly zero: %s", root)
	}
	square := root.Mul(root, prec)
	if !rationalIn(square.re, big.NewRat(2, 1)) || !rationalIn(square.im, big.NewRat(0, 1)) {
		t.Fatalf("sqrt(2)^2 = %s does not enclose 2", square)
	}
	if digits := root.re.Ball(prec).GuaranteedDigits(); digits < 30 {
		t.Fatalf("sqrt(2) guaranteed digits = %d, want at least 30", digits)
	}
	negative := mustOK(t)(Sqrt(ci(-2, 0), prec))
	if !isExactZero(negative.re) || negative.im.IsExact() {
		t.Fatalf("sqrt(-2) should be exactly 0 + ball i: %s", negative)
	}
	if !rationalIn(negative.Mul(negative, prec).re, big.NewRat(-2, 1)) {
		t.Fatalf("sqrt(-2)^2 = %s does not enclose -2", negative.Mul(negative, prec))
	}
}

func TestSqrtPrincipalBranchMatchesFloat(t *testing.T) {
	rnd := rand.New(rand.NewSource(3))
	for i := 0; i < 400; i++ {
		z := cq(rnd.Int63n(81)-40, rnd.Int63n(7)+1, rnd.Int63n(81)-40, rnd.Int63n(7)+1)
		got := mustOK(t)(Sqrt(z, prec))
		requireNear(t, "sqrt("+z.String()+")", got, cmplx.Sqrt(complexOf(z)))
		if float64Of(got.re) < 0 {
			t.Fatalf("principal sqrt must have non-negative real part: %s", got)
		}
		square := got.Mul(got, prec)
		if !rationalIn(square.re, z.re.Exact().Rat()) || !rationalIn(square.im, z.im.Exact().Rat()) {
			t.Fatalf("sqrt(%s)^2 = %s does not enclose the input", z, square)
		}
	}
}

func TestSqrtBranchCutAndAxes(t *testing.T) {
	straddle := Rect(exactNum(-4, 1), ballNum(0, 1e-9))
	if _, err := Sqrt(straddle, prec); !errors.Is(err, ErrBranchCut) {
		t.Fatalf("ball straddling the negative real axis: err = %v", err)
	}
	right := Rect(exactNum(4, 1), ballNum(0, 1e-9))
	got := mustOK(t)(Sqrt(right, prec))
	if !rationalIn(got.re, big.NewRat(2, 1)) || !rationalIn(got.im, big.NewRat(0, 1)) {
		t.Fatalf("sqrt(4 + tiny ball i) = %s should enclose 2", got)
	}
	for _, sample := range []float64{-1e-9, -3e-10, 0, 4e-10, 1e-9} {
		requireEnclosesFloat(t, "sqrt near positive axis", got, cmplx.Sqrt(complex(4, sample)))
	}
	negativeImaginary := mustOK(t)(Sqrt(Rect(exactNum(-4, 1), exactNum(-1, 1000)), prec))
	if float64Of(negativeImaginary.im) >= 0 {
		t.Fatalf("sqrt just below the negative axis must have negative imaginary part: %s", negativeImaginary)
	}
	positiveImaginary := mustOK(t)(Sqrt(Rect(exactNum(-4, 1), exactNum(1, 1000)), prec))
	if float64Of(positiveImaginary.im) <= 0 {
		t.Fatalf("sqrt just above the negative axis must have positive imaginary part: %s", positiveImaginary)
	}
}

func TestAbs(t *testing.T) {
	exactCases := []struct {
		in   Complex
		want string
	}{
		{ci(3, 4), "5"},
		{ci(-3, 4), "5"},
		{ci(5, 12), "13"},
		{ci(0, -7), "7"},
		{ci(-7, 0), "7"},
		{ci(0, 0), "0"},
		{cq(3, 5, 4, 5), "1"},
		{cq(3, 10, 2, 5), "1/2"},
	}
	for _, tc := range exactCases {
		got := mustNum(t)(tc.in.Abs(prec))
		if !got.IsExact() || got.String() != tc.want {
			t.Fatalf("Abs(%s) = %s (exact=%v), want exact %s", tc.in, got, got.IsExact(), tc.want)
		}
	}
	root := mustNum(t)(ci(1, 1).Abs(prec))
	if root.IsExact() {
		t.Fatalf("|1+i| must be a ball: %s", root)
	}
	square := root.Mul(root, prec)
	if !rationalIn(square, big.NewRat(2, 1)) {
		t.Fatalf("|1+i|^2 = %s does not enclose 2", square)
	}
}

func TestAbsOfBallsEncloses(t *testing.T) {
	rnd := rand.New(rand.NewSource(5))
	for i := 0; i < 300; i++ {
		reMid, imMid := rnd.Float64()*20-10, rnd.Float64()*20-10
		reRad, imRad := rnd.Float64()*0.5, rnd.Float64()*0.5
		z := Rect(ballNum(reMid, reRad), ballNum(imMid, imRad))
		got := mustNum(t)(z.Abs(prec))
		for j := 0; j < 20; j++ {
			x := reMid + (rnd.Float64()*2-1)*reRad
			y := imMid + (rnd.Float64()*2-1)*imRad
			truth := math.Hypot(x, y)
			b := got.Ball(prec)
			lo, _ := b.Lo().Float64()
			hi, _ := b.Hi().Float64()
			if truth < lo-1e-12 || truth > hi+1e-12 {
				t.Fatalf("|%v+%vi| = %v outside [%v, %v]", x, y, truth, lo, hi)
			}
		}
	}
	straddle := Rect(ballNum(0, 1), ballNum(0, 1))
	got := mustNum(t)(straddle.Abs(prec))
	if lo := got.Ball(prec).Lo(); lo.Sign() > 0 {
		t.Fatalf("|ball around origin| lower bound %v must not exceed 0", lo)
	}
	if !got.Ball(prec).Contains(new(big.Float).SetFloat64(1.4142135)) {
		t.Fatalf("|ball around origin| = %s should contain 1.4142135", got)
	}
}

func TestArgAllQuadrantsAndAxes(t *testing.T) {
	cases := []struct {
		name string
		in   Complex
		want float64
	}{
		{"positive real", ci(5, 0), 0},
		{"first quadrant", ci(1, 1), math.Pi / 4},
		{"positive imaginary", ci(0, 3), math.Pi / 2},
		{"second quadrant", ci(-1, 1), 3 * math.Pi / 4},
		{"negative real", ci(-2, 0), math.Pi},
		{"third quadrant", ci(-1, -1), -3 * math.Pi / 4},
		{"negative imaginary", ci(0, -3), -math.Pi / 2},
		{"fourth quadrant", ci(1, -1), -math.Pi / 4},
		{"shallow second", ci(-5, 1), math.Atan2(1, -5)},
		{"shallow third", ci(-5, -1), math.Atan2(-1, -5)},
	}
	for _, tc := range cases {
		got := mustNum(t)(tc.in.Arg(prec))
		if math.Abs(float64Of(got)-tc.want) > 1e-12 {
			t.Fatalf("%s: Arg(%s) = %s, want %v", tc.name, tc.in, got, tc.want)
		}
		if upper := got.Ball(prec).Hi(); upper.Cmp(new(big.Float).SetFloat64(math.Pi+1e-9)) > 0 {
			t.Fatalf("%s: Arg exceeds pi: %s", tc.name, got)
		}
	}
	positiveReal := mustNum(t)(ci(5, 0).Arg(prec))
	if !positiveReal.IsExact() || positiveReal.String() != "0" {
		t.Fatalf("Arg(5) must be exactly 0: %s", positiveReal)
	}
	negativeReal := mustNum(t)(ci(-2, 0).Arg(prec))
	if !negativeReal.Ball(prec).Contains(elem.Pi(prec).Mid) {
		t.Fatalf("Arg(-2) must enclose pi: %s", negativeReal)
	}
	if negativeReal.Ball(prec).Lo().Sign() <= 0 {
		t.Fatalf("Arg(-2) must be positive pi, never -pi: %s", negativeReal)
	}
}

func TestArgErrors(t *testing.T) {
	if _, err := ci(0, 0).Arg(prec); !errors.Is(err, ErrArgOfZero) {
		t.Fatalf("Arg(0): err = %v", err)
	}
	if _, err := Real(ballNum(0, 1e-9)).Arg(prec); !errors.Is(err, ErrArgOfZero) {
		t.Fatalf("Arg(ball around 0): err = %v", err)
	}
	if _, err := Rect(ballNum(0, 1e-9), ballNum(0, 1e-9)).Arg(prec); err == nil {
		t.Fatal("Arg of a ball containing the origin must error")
	}
	if _, err := Rect(exactNum(-1, 1), ballNum(0, 1e-9)).Arg(prec); err == nil {
		t.Fatal("Arg of a ball straddling the negative real axis must error")
	}
}

func TestEulerIdentity(t *testing.T) {
	pi := num.NumBall(elem.Pi(prec))
	got := mustOK(t)(Exp(Rect(exactNum(0, 1), pi), prec))
	sum := got.Add(Real(exactNum(1, 1)), prec)
	if !enclosesZero(sum.re) || !enclosesZero(sum.im) {
		t.Fatalf("exp(i*pi) + 1 = %s does not enclose 0", sum)
	}
	if got.re.IsExact() || got.im.IsExact() {
		t.Fatalf("exp(i*pi) must be a ball: %s", got)
	}
	if digits := sum.re.Ball(prec).GuaranteedPlaces(); digits < 30 {
		t.Fatalf("exp(i*pi)+1 real part pinned only to %d places", digits)
	}
}

func TestExpKnownValues(t *testing.T) {
	zero := mustOK(t)(Exp(ci(0, 0), prec))
	requireExact(t, "exp(0)", zero, "1")
	cases := []Complex{ci(1, 2), ci(-3, 1), cq(1, 2, -7, 3), ci(0, 5), ci(2, 0), ci(-1, -4), cq(1, 7, 1, 7)}
	for _, z := range cases {
		got := mustOK(t)(Exp(z, prec))
		requireNear(t, "exp("+z.String()+")", got, cmplx.Exp(complexOf(z)))
	}
	real := mustOK(t)(Exp(ci(1, 0), prec))
	if !isExactZero(real.im) {
		t.Fatalf("exp of a real must keep an exactly-zero imaginary part: %s", real)
	}
	pureImaginary := mustOK(t)(Exp(ci(0, 2), prec))
	requireNear(t, "exp(2i)", pureImaginary, cmplx.Exp(complex(0, 2)))
}

func TestLogKnownValues(t *testing.T) {
	cases := []Complex{ci(1, 2), ci(-3, 1), ci(-3, -1), cq(1, 2, -7, 3), ci(0, 5), ci(0, -5), ci(2, 0), ci(-2, 0), ci(3, -4)}
	for _, z := range cases {
		got := mustOK(t)(Log(z, prec))
		requireNear(t, "log("+z.String()+")", got, cmplx.Log(complexOf(z)))
	}
	one := mustOK(t)(Log(ci(1, 0), prec))
	requireExact(t, "log(1)", one, "0")
	logI := mustOK(t)(Log(ci(0, 1), prec))
	if !isExactZero(logI.re) {
		t.Fatalf("log(i) must have exactly zero real part: %s", logI)
	}
	if !rationalIn(num.NumBall(logI.im.Ball(prec)), new(big.Rat).SetFloat64(math.Pi/2)) &&
		math.Abs(float64Of(logI.im)-math.Pi/2) > 1e-15 {
		t.Fatalf("log(i) imaginary part %s should be pi/2", logI)
	}
	logNegative := mustOK(t)(Log(ci(-1, 0), prec))
	if !isExactZero(logNegative.re) || float64Of(logNegative.im) <= 3.14 {
		t.Fatalf("log(-1) must be exactly 0 + pi i: %s", logNegative)
	}
}

func TestLogErrors(t *testing.T) {
	if _, err := Log(ci(0, 0), prec); !errors.Is(err, ErrLogOfZero) {
		t.Fatalf("log(0): err = %v", err)
	}
	if _, err := Log(Rect(ballNum(0, 1e-9), ballNum(0, 1e-9)), prec); err == nil {
		t.Fatal("log of a ball containing the origin must error")
	}
	if _, err := Log(Real(ballNum(0, 1e-9)), prec); err == nil {
		t.Fatal("log of a real ball containing zero must error")
	}
	if _, err := Log(Rect(exactNum(-1, 1), ballNum(0, 1e-9)), prec); err == nil {
		t.Fatal("log of a ball straddling the branch cut must error")
	}
}

func TestLogExpRoundTrip(t *testing.T) {
	cases := []Complex{
		ci(1, 2), ci(-1, 1), ci(0, 3), ci(2, -3), cq(1, 3, 1, 2), ci(-2, -2), ci(0, 0), ci(3, 0), ci(-1, 3),
		cq(-1, 2, 11, 4), ci(5, -1), ci(-4, 2),
	}
	for _, z := range cases {
		back := mustOK(t)(Log(mustOK(t)(Exp(z, prec)), prec))
		if !rationalIn(back.re, z.re.Exact().Rat()) || !rationalIn(back.im, z.im.Exact().Rat()) {
			t.Fatalf("log(exp(%s)) = %s does not enclose the input", z, back)
		}
	}
}

func TestLogExpRoundTripOutsidePrincipalStripWrapsIntoIt(t *testing.T) {
	z := ci(1, 4)
	back := mustOK(t)(Log(mustOK(t)(Exp(z, prec)), prec))
	wrapped := 4 - 2*math.Pi
	if math.Abs(float64Of(back.im)-wrapped) > 1e-12 {
		t.Fatalf("log(exp(1+4i)) imaginary part = %s, want %v", back, wrapped)
	}
}

func TestExpLogRoundTrip(t *testing.T) {
	cases := []Complex{ci(1, 2), ci(-3, 1), ci(0, 2), ci(-5, -5), cq(1, 2, 1, 3), ci(2, 0), ci(-2, 0), ci(0, -3)}
	for _, z := range cases {
		back := mustOK(t)(Exp(mustOK(t)(Log(z, prec)), prec))
		if !rationalIn(back.re, z.re.Exact().Rat()) || !rationalIn(back.im, z.im.Exact().Rat()) {
			t.Fatalf("exp(log(%s)) = %s does not enclose the input", z, back)
		}
	}
}

func TestPowerLawsOnBalls(t *testing.T) {
	z := mustOK(t)(Exp(ci(1, 1), prec))
	cubed := mustOK(t)(z.Pow(3, prec))
	direct := mustOK(t)(Exp(ci(3, 3), prec))
	diff := cubed.Sub(direct, prec)
	if !enclosesZero(diff.re) || !enclosesZero(diff.im) {
		t.Fatalf("exp(1+i)^3 - exp(3+3i) = %s does not enclose 0", diff)
	}
	inverse := mustOK(t)(z.Pow(-2, prec))
	product := inverse.Mul(mustOK(t)(z.Pow(2, prec)), prec)
	if !rationalIn(product.re, big.NewRat(1, 1)) || !rationalIn(product.im, big.NewRat(0, 1)) {
		t.Fatalf("z^-2 * z^2 = %s does not enclose 1", product)
	}
}

func sampleInside(rnd *rand.Rand, mid, rad float64) *big.Rat {
	t := rnd.Float64()*2 - 1
	return new(big.Rat).Add(new(big.Rat).SetFloat64(mid), new(big.Rat).Mul(new(big.Rat).SetFloat64(rad), new(big.Rat).SetFloat64(t)))
}

func TestBallMulDivEncloseExactSamples(t *testing.T) {
	rnd := rand.New(rand.NewSource(17))
	for i := 0; i < 1500; i++ {
		aMid := [2]float64{rnd.Float64()*10 - 5, rnd.Float64()*10 - 5}
		bMid := [2]float64{rnd.Float64()*10 - 5, rnd.Float64()*10 - 5}
		aRad := [2]float64{rnd.Float64() * 0.4, rnd.Float64() * 0.4}
		bRad := [2]float64{rnd.Float64() * 0.4, rnd.Float64() * 0.4}
		a := Rect(ballNum(aMid[0], aRad[0]), ballNum(aMid[1], aRad[1]))
		b := Rect(ballNum(bMid[0], bRad[0]), ballNum(bMid[1], bRad[1]))
		product := a.Mul(b, prec)
		sum := a.Add(b, prec)
		difference := a.Sub(b, prec)
		quotient, quotientErr := a.Div(b, prec)
		for j := 0; j < 12; j++ {
			ar, ai := sampleInside(rnd, aMid[0], aRad[0]), sampleInside(rnd, aMid[1], aRad[1])
			br, bi := sampleInside(rnd, bMid[0], bRad[0]), sampleInside(rnd, bMid[1], bRad[1])
			trueProduct := ci(0, 0).Add(Rect(num.NumExact(num.FromRat(ar)), num.NumExact(num.FromRat(ai))).
				Mul(Rect(num.NumExact(num.FromRat(br)), num.NumExact(num.FromRat(bi))), prec), prec)
			if !rationalIn(product.re, trueProduct.re.Exact().Rat()) || !rationalIn(product.im, trueProduct.im.Exact().Rat()) {
				t.Fatalf("product misses sample: %s", product)
			}
			if !rationalIn(sum.re, new(big.Rat).Add(ar, br)) || !rationalIn(sum.im, new(big.Rat).Add(ai, bi)) {
				t.Fatalf("sum misses sample")
			}
			if !rationalIn(difference.re, new(big.Rat).Sub(ar, br)) || !rationalIn(difference.im, new(big.Rat).Sub(ai, bi)) {
				t.Fatalf("difference misses sample")
			}
			if quotientErr != nil {
				continue
			}
			exactA := Rect(num.NumExact(num.FromRat(ar)), num.NumExact(num.FromRat(ai)))
			exactB := Rect(num.NumExact(num.FromRat(br)), num.NumExact(num.FromRat(bi)))
			trueQuotient, err := exactA.Div(exactB, prec)
			if err != nil {
				continue
			}
			if !rationalIn(quotient.re, trueQuotient.re.Exact().Rat()) || !rationalIn(quotient.im, trueQuotient.im.Exact().Rat()) {
				t.Fatalf("quotient misses sample: %s / %s = %s, true %s", a, b, quotient, trueQuotient)
			}
		}
	}
}

func TestBallDivisionAcceptsOnlyProvenNonzeroDivisors(t *testing.T) {
	rnd := rand.New(rand.NewSource(23))
	accepted, rejected := 0, 0
	for i := 0; i < 600; i++ {
		reMid, imMid := rnd.Float64()*2-1, rnd.Float64()*2-1
		rad := rnd.Float64() * 1.2
		divisor := Rect(ballNum(reMid, rad), ballNum(imMid, rad))
		_, err := ci(1, 0).Div(divisor, prec)
		originInside := math.Abs(reMid) <= rad && math.Abs(imMid) <= rad
		if originInside && err == nil {
			t.Fatalf("divisor %s contains the origin but division succeeded", divisor)
		}
		if err == nil {
			accepted++
		} else {
			rejected++
		}
	}
	if accepted == 0 || rejected == 0 {
		t.Fatalf("expected a mix of accepted and rejected divisors, got %d and %d", accepted, rejected)
	}
}

func TestBallSqrtExpLogEncloseSamples(t *testing.T) {
	rnd := rand.New(rand.NewSource(29))
	for i := 0; i < 300; i++ {
		mid := [2]float64{rnd.Float64()*6 - 3, rnd.Float64()*6 - 3}
		rad := [2]float64{rnd.Float64() * 0.05, rnd.Float64() * 0.05}
		z := Rect(ballNum(mid[0], rad[0]), ballNum(mid[1], rad[1]))
		sqrtZ, sqrtErr := Sqrt(z, prec)
		expZ := mustOK(t)(Exp(z, prec))
		logZ, logErr := Log(z, prec)
		for j := 0; j < 10; j++ {
			x, y := sampleInside(rnd, mid[0], rad[0]), sampleInside(rnd, mid[1], rad[1])
			xf, _ := x.Float64()
			yf, _ := y.Float64()
			point := complex(xf, yf)
			if sqrtErr == nil {
				truth := cmplx.Sqrt(point)
				requireEnclosesFloat(t, "sqrt", sqrtZ, truth)
			}
			requireEnclosesFloat(t, "exp", expZ, cmplx.Exp(point))
			if logErr == nil {
				requireEnclosesFloat(t, "log", logZ, cmplx.Log(point))
			}
		}
	}
}

func requireEnclosesFloat(t *testing.T, name string, c Complex, truth complex128) {
	t.Helper()
	slack := 1e-12 * (1 + cmplx.Abs(truth))
	for _, part := range []struct {
		n     num.Number
		value float64
	}{{c.re, real(truth)}, {c.im, imag(truth)}} {
		b := part.n.Ball(prec)
		lo, _ := b.Lo().Float64()
		hi, _ := b.Hi().Float64()
		if part.value < lo-slack || part.value > hi+slack {
			t.Fatalf("%s result %s does not enclose %v", name, c, truth)
		}
	}
}

func TestBallResultsStayBallsAndExactStayExact(t *testing.T) {
	b := Rect(ballNum(1.5, 1e-20), ballNum(2.5, 1e-20))
	for name, got := range map[string]Complex{
		"add": b.Add(ci(1, 1), prec),
		"mul": b.Mul(ci(1, 1), prec),
	} {
		if got.IsExact() {
			t.Fatalf("%s of a ball produced an exact value: %s", name, got)
		}
	}
	zeroTimesBall := ci(0, 0).Mul(b, prec)
	requireExact(t, "0*ball", zeroTimesBall, "0")
	oneTimesBall := ci(1, 0).Mul(b, prec)
	if oneTimesBall.re.Ball(prec).Rad.Cmp(b.re.Ball(prec).Rad) != 0 {
		t.Fatal("multiplying by exactly one must not widen the ball")
	}
}

func TestNeverPanicsOnExtremeInputs(t *testing.T) {
	huge := new(big.Float).SetPrec(prec).SetMantExp(big.NewFloat(1), 2000)
	hugeBall := num.NumBall(num.NewBall(huge, num.MagFromFloat(1)))
	infinite := num.NumBall(num.NewBall(new(big.Float).SetPrec(prec), num.MagFromBigFloat(new(big.Float).SetInf(false))))
	inputs := []Complex{
		Rect(hugeBall, hugeBall), Rect(infinite, infinite), Rect(infinite, exactNum(1, 1)),
		Rect(exactNum(1, 1), infinite), Real(infinite), ci(0, 0), Rect(hugeBall, exactNum(0, 1)),
	}
	for _, a := range inputs {
		for _, b := range inputs {
			a.Add(b, prec)
			a.Sub(b, prec)
			a.Mul(b, prec)
			a.Div(b, prec)
		}
		a.Abs(prec)
		a.Arg(prec)
		_ = a.String()
		a.Pow(3, prec)
		a.Pow(-3, prec)
		Sqrt(a, prec)
		Exp(a, prec)
		Log(a, prec)
	}
}

func TestExactInputsNeverProduceBallsForClosedOperations(t *testing.T) {
	rnd := rand.New(rand.NewSource(41))
	for i := 0; i < 500; i++ {
		a := cq(rnd.Int63n(21)-10, rnd.Int63n(5)+1, rnd.Int63n(21)-10, rnd.Int63n(5)+1)
		b := cq(rnd.Int63n(21)-10, rnd.Int63n(5)+1, rnd.Int63n(21)-10, rnd.Int63n(5)+1)
		k := rnd.Intn(13) - 6
		results := []Complex{a.Add(b, prec), a.Sub(b, prec), a.Mul(b, prec), a.Neg(), a.Conj()}
		if q, err := a.Div(b, prec); err == nil {
			results = append(results, q)
		}
		if p, err := a.Pow(k, prec); err == nil {
			results = append(results, p)
		}
		for _, r := range results {
			if !r.IsExact() {
				t.Fatalf("closed operation on exact inputs produced a ball: %s", r)
			}
		}
	}
}

func TestNeverPanicsOnRandomExtremeBalls(t *testing.T) {
	rnd := rand.New(rand.NewSource(57))
	component := func() num.Number {
		if rnd.Intn(4) == 0 {
			return exactNum(rnd.Int63n(7)-3, 1)
		}
		mid := new(big.Float).SetPrec(prec).SetMantExp(big.NewFloat(rnd.Float64()*2-1), rnd.Intn(6001)-3000)
		rad := new(big.Float).SetMantExp(big.NewFloat(rnd.Float64()), rnd.Intn(6001)-3000)
		return num.NumBall(num.NewBall(mid, num.MagFromBigFloat(rad)))
	}
	for i := 0; i < 3000; i++ {
		a := Rect(component(), component())
		b := Rect(component(), component())
		a.Mul(b, prec)
		a.Div(b, prec)
		a.Abs(prec)
		a.Arg(prec)
		a.Pow(5, prec)
		a.Pow(-5, prec)
		_ = a.String()
		Sqrt(a, prec)
		Exp(a, prec)
		Log(a, prec)
	}
}

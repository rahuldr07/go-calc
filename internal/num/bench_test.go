package num

import (
	"math/big"
	"testing"
)

const benchPrec = 120

func benchBall(s string) Ball {
	f, _, err := big.ParseFloat(s, 10, benchPrec, big.ToNearestEven)
	if err != nil {
		panic(err)
	}
	return NewBall(f, RoundingError(f, benchPrec))
}

var (
	benchRoot2 = benchBall("1.41421356237309504880168872420969807857")
	benchPi    = benchBall("3.14159265358979323846264338327950288420")
	benchSink  Ball
)

func BenchmarkBallMul120(b *testing.B) {
	l, r := benchRoot2, benchPi
	b.ReportAllocs()
	for b.Loop() {
		benchSink = l.Mul(r, benchPrec)
	}
}

func BenchmarkBallAdd120(b *testing.B) {
	l, r := benchRoot2, benchPi
	b.ReportAllocs()
	for b.Loop() {
		benchSink = l.Add(r, benchPrec)
	}
}

func BenchmarkBallDiv120(b *testing.B) {
	l, r := benchRoot2, benchPi
	b.ReportAllocs()
	for b.Loop() {
		q, err := l.Div(r, benchPrec)
		if err != nil {
			b.Fatal(err)
		}
		benchSink = q
	}
}

package solve

import (
	"math/big"

	"github.com/rahuldr07/go-calc/internal/num"
)

type Sign func(x num.Exact) (sign int, decided bool)

type Continuous func(lo, hi num.Exact) bool

type Bracket struct {
	Lo, Hi num.Exact
}

type Found struct {
	Lo, Hi     num.Exact
	Bisections int
}

func (f Found) Exact() bool { return f.Lo.Cmp(f.Hi) == 0 }

type Scan struct {
	Exact    []num.Exact
	Runs     []Run
	Brackets []Bracket
	Decided  int
	Step     num.Exact
}

type Run struct {
	Lo, Hi num.Exact
}

func Grid(sign Sign, lo, hi num.Exact, steps int) Scan {
	var out Scan
	width, _ := hi.Sub(lo).Div(num.FromInt(int64(steps)))
	out.Step = width
	var prevSign int
	var prevAt num.Exact
	havePrev := false
	var zeros []num.Exact
	settle := func() {
		switch {
		case len(zeros) == 1:
			out.Exact = append(out.Exact, zeros[0])
		case len(zeros) > 1:
			out.Runs = append(out.Runs, Run{Lo: zeros[0], Hi: zeros[len(zeros)-1]})
		}
		zeros = nil
	}
	for i := 0; i <= steps; i++ {
		x := lo.Add(width.Mul(num.FromInt(int64(i))))
		s, ok := sign(x)
		if !ok {
			settle()
			havePrev = false
			continue
		}
		out.Decided++
		if s != 0 {
			settle()
		}
		switch {
		case s == 0:
			zeros = append(zeros, x)
			havePrev = false
		default:
			if havePrev && s != prevSign {
				out.Brackets = append(out.Brackets, Bracket{Lo: prevAt, Hi: x})
			}
			prevSign, prevAt, havePrev = s, x, true
		}
	}
	settle()
	return out
}

func Refine(sign Sign, continuous Continuous, b Bracket, bits uint) (Found, bool) {
	loSign, _ := sign(b.Lo)
	tolerance := toleranceFor(b, bits)
	lo, hi := b.Lo, b.Hi
	bisections := 0
	for hi.Sub(lo).Cmp(tolerance) > 0 {
		mid, _ := lo.Add(hi).Div(num.FromInt(2))
		s, ok := sign(mid)
		if !ok {
			break
		}
		bisections++
		if s == 0 {
			return Found{Lo: mid, Hi: mid, Bisections: bisections}, true
		}
		if s == loSign {
			lo = mid
		} else {
			hi = mid
		}
	}
	if !continuous(lo, hi) {
		return Found{}, false
	}
	return Found{Lo: lo, Hi: hi, Bisections: bisections}, true
}

func toleranceFor(b Bracket, bits uint) num.Exact {
	scale := num.FromInt(1)
	for _, end := range []num.Exact{b.Lo, b.Hi} {
		if magnitude := absolute(end); magnitude.Cmp(scale) > 0 {
			scale = magnitude
		}
	}
	denominator := new(big.Int).Lsh(big.NewInt(1), bits)
	unit := num.FromRat(new(big.Rat).SetFrac(big.NewInt(1), denominator))
	return scale.Mul(unit)
}

func absolute(x num.Exact) num.Exact {
	if x.Sign() < 0 {
		return x.Neg()
	}
	return x
}

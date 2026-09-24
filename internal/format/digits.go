package format

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

const ballPrec = 256

const boundDigits = 4

func GuaranteedDigits(b num.Ball) int { return b.GuaranteedDigits() }

func (s Style) NumberBall(b num.Ball) (string, int, bool) {
	text, digits := provenDecimal(b, -1)
	return s.groupDecimal(text), digits, digits < s.digitBudget()
}

func (s Style) digitBudget() int {
	if s.Digits <= 0 {
		return Digits
	}
	return s.Digits
}

func provenDecimal(b num.Ball, limit int) (string, int) {
	digits := b.GuaranteedDigits()
	if limit >= 0 && digits > limit {
		digits = limit
	}
	lo, hi, scale := b.Lo(), b.Hi(), b.DecimalScale()
	for ; digits >= 1; digits-- {
		text := significantText(lo, digits, scale)
		if text != significantText(hi, digits, scale) {
			continue
		}
		return text, digits
	}
	if provablyZero(b) {
		return "0", 0
	}
	return bounds(b), 0
}

func bounds(b num.Ball) string { return boundsAt(b, boundDigits) }

func boundsAt(b num.Ball, digits int) string {
	lo, _ := b.Lo().Rat(nil)
	hi, _ := b.Hi().Rat(nil)
	if lo == nil || hi == nil {
		return "[" + b.Lo().Text('g', digits) + ", " + b.Hi().Text('g', digits) + "]"
	}
	scale := ratScale(lo)
	if s := ratScale(hi); s > scale {
		scale = s
	}
	decimals := digits - 1 - scale
	low, high := outward(lo, decimals, false), outward(hi, decimals, true)
	if decimals < 0 || scale < -5 {
		return "[" + scientific(low, digits) + ", " + scientific(high, digits) + "]"
	}
	return "[" + tidy(at(low, decimals)) + ", " + tidy(at(high, decimals)) + "]"
}

func scientific(r *big.Rat, digits int) string {
	if r.Sign() == 0 {
		return "0"
	}
	e := ratScale(r)
	m := new(big.Rat).Quo(r, ratPow10(e))
	return tidy(m.FloatString(digits-1)) + "e" + strconv.Itoa(e)
}

func tightBounds(b num.Ball) string {
	digits := b.GuaranteedDigits() + 2
	if digits < boundDigits {
		digits = boundDigits
	}
	if digits > Digits+2 {
		digits = Digits + 2
	}
	return boundsAt(b, digits)
}

func outward(r *big.Rat, decimals int, up bool) *big.Rat {
	got, ok := new(big.Rat).SetString(at(r, decimals))
	if !ok {
		return new(big.Rat).Set(r)
	}
	switch {
	case up && got.Cmp(r) < 0:
		got.Add(got, ratPow10(-decimals))
	case !up && got.Cmp(r) > 0:
		got.Sub(got, ratPow10(-decimals))
	}
	return got
}

func significantText(x *big.Float, digits, scale int) string {
	if places := digits - 1 - scale; places >= 0 {
		return tidy(x.Text('f', places))
	}
	return tidy(x.Text('e', digits-1))
}

func tidy(text string) string {
	if i := strings.IndexByte(text, 'e'); i >= 0 {
		return unsignZero(trimDecimal(text[:i])) + text[i:]
	}
	return unsignZero(trimDecimal(text))
}

func unsignZero(text string) string {
	if strings.IndexFunc(text, func(r rune) bool { return r >= '1' && r <= '9' }) < 0 {
		return strings.TrimPrefix(text, "-")
	}
	return text
}

func (s Style) precisionNote(v value.Value) string {
	if v.Num.IsExact() {
		return ""
	}
	b := v.Num.Ball(ballPrec)
	_, digits, capped := s.NumberBall(b)
	if !capped {
		return ""
	}
	if digits == 0 && provablyZero(b) {
		return "zero to " + places(b.GuaranteedPlaces()) + ", the precision available"
	}
	return plural(digits, "digit") + " guaranteed" + becauseOf(v.Stopped)
}

func becauseOf(stop value.Stop) string {
	switch stop {
	case value.StopPrecisionCap:
		return " (precision cap reached)"
	case value.StopUncertainInput:
		return " (an input carries uncertainty)"
	case value.StopDeadline:
		return " (time limit reached)"
	}
	return ""
}

func provablyZero(b num.Ball) bool { return b.ContainsZero() && b.GuaranteedPlaces() >= 1 }

func places(n int) string { return plural(n, "decimal place") }

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

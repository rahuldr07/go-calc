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

func bounds(b num.Ball) string {
	lo, _ := b.Lo().Rat(nil)
	hi, _ := b.Hi().Rat(nil)
	if lo == nil || hi == nil {
		return b.String()
	}
	scale := ratScale(lo)
	if s := ratScale(hi); s > scale {
		scale = s
	}
	decimals := boundDigits - 1 - scale
	return "[" + outward(lo, decimals, false) + ", " + outward(hi, decimals, true) + "]"
}

func outward(r *big.Rat, decimals int, up bool) string {
	text := at(r, decimals)
	got, ok := new(big.Rat).SetString(text)
	if !ok {
		return tidy(text)
	}
	switch {
	case up && got.Cmp(r) < 0:
		got.Add(got, ratPow10(-decimals))
	case !up && got.Cmp(r) > 0:
		got.Sub(got, ratPow10(-decimals))
	default:
		return tidy(text)
	}
	return tidy(at(got, decimals))
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

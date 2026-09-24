package format

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/value"
)

const BallPrec = 256

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
	if b.ContainsZero() {
		return "0", 0
	}
	return b.String(), 0
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
	b := v.Num.Ball(BallPrec)
	_, digits, capped := s.NumberBall(b)
	if !capped {
		return ""
	}
	if digits == 0 && b.ContainsZero() {
		return "zero to " + strconv.Itoa(b.GuaranteedPlaces()) + " decimal places, the precision available"
	}
	return strconv.Itoa(digits) + " digits guaranteed" + becauseOf(v.Stopped)
}

func becauseOf(stop value.Stop) string {
	switch stop {
	case value.StopPrecisionCap:
		return " (precision cap reached)"
	case value.StopNoProgress:
		return " (more precision did not help)"
	case value.StopUncertainInput:
		return " (an input carries uncertainty)"
	case value.StopDeadline:
		return " (time limit reached)"
	}
	return ""
}

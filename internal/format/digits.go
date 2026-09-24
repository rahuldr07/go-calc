package format

import (
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
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
	places := b.GuaranteedDigits()
	if limit >= 0 && places > limit {
		places = limit
	}
	lo, hi := b.Lo(), b.Hi()
	for ; places >= 0; places-- {
		text := unsignZero(lo.Text('f', places))
		if text != unsignZero(hi.Text('f', places)) {
			continue
		}
		return trimDecimal(text), places
	}
	return b.String(), 0
}

func unsignZero(text string) string {
	if strings.IndexFunc(text, func(r rune) bool { return r >= '1' && r <= '9' }) < 0 {
		return strings.TrimPrefix(text, "-")
	}
	return text
}

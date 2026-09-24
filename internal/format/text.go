package format

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const Digits = 12

const UnknownUncertainty = "?"

type Style struct {
	Digits  int
	Grouped bool
	Indian  bool
}

var Default = Style{Digits: Digits}

func Text(v value.Value, notes []parse.Note) string { return Default.Text(v, notes) }

const definedSuffix = " defined"

func (s Style) Text(v value.Value, notes []parse.Note) string {
	if v.IsDefinition() {
		return v.Def + definedSuffix
	}
	var b strings.Builder
	mid, u, rounded := s.Uncertain(v)
	switch {
	case v.Unc.Degenerate:
		b.WriteString(s.Number(v) + " ± " + UnknownUncertainty)
		if v.Unit != "" {
			b.WriteString(" " + v.Unit)
		}
		if note := s.precisionNote(v); note != "" {
			b.WriteString("\n  " + note)
		}
	case rounded:
		b.WriteString(mid + " ± " + u)
		if v.Unit != "" {
			b.WriteString(" " + v.Unit)
		}
		if label := v.Unc.Cover.Label; label != "" {
			b.WriteString(" (" + label + ")")
		}
	default:
		b.WriteString(s.Number(v))
		if v.Unit != "" {
			b.WriteString(" " + v.Unit)
		}
		if note := s.precisionNote(v); note != "" {
			b.WriteString("\n  " + note)
		}
	}
	if note := covarianceNote(v); note != "" {
		b.WriteString("\n  " + note)
	}
	if v.Unc.Warning != "" {
		b.WriteString("\n  " + v.Unc.Warning)
	}
	for _, n := range notes {
		b.WriteString("\n  read as " + strings.TrimPrefix(n.Msg, "read as "))
	}
	return b.String()
}

func (s Style) Number(v value.Value) string {
	text, complete := s.Decimal(v)
	if complete {
		return text
	}
	return v.Num.Exact().Rat().RatString() + "  ≈ " + text
}

func (s Style) Decimal(v value.Value) (string, bool) {
	if !v.Num.IsExact() {
		text, _ := provenDecimal(v.Num.Ball(ballPrec), s.digitBudget())
		return s.groupDecimal(text), true
	}
	r := v.Num.Exact().Rat()
	if r.IsInt() {
		return s.group(r.Num().String()), true
	}
	if places, ok := terminatingPlaces(r); ok {
		if full := trimDecimal(r.FloatString(places)); significantDigits(full) <= s.digitBudget() {
			if tooSmallToSpellOut(ratScale(r)) {
				return significantRat(r, s.digitBudget()), true
			}
			return s.groupDecimal(full), true
		}
	}
	return s.groupDecimal(significantRat(r, s.digitBudget())), false
}

const smallestSpeltOutScale = -5

func tooSmallToSpellOut(scale int) bool { return scale < smallestSpeltOutScale }

func trimDecimal(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

func (s Style) groupDecimal(dec string) string {
	if !plainDecimal(dec) {
		return dec
	}
	if i := strings.IndexByte(dec, '.'); i >= 0 {
		return s.group(dec[:i]) + dec[i:]
	}
	return s.group(dec)
}

func plainDecimal(dec string) bool {
	body := strings.TrimPrefix(dec, "-")
	if body == "" {
		return false
	}
	dots := 0
	for _, r := range body {
		switch {
		case r >= '0' && r <= '9':
		case r == '.':
			dots++
		default:
			return false
		}
	}
	return dots <= 1
}

func (s Style) group(digits string) string {
	if !s.Grouped || len(digits) == 0 {
		return digits
	}
	neg := digits[0] == '-'
	if neg {
		digits = digits[1:]
	}
	var out string
	if s.Indian {
		out = groupIndian(digits)
	} else {
		out = groupThousands(digits)
	}
	if neg {
		return "-" + out
	}
	return out
}

func groupThousands(d string) string {
	var parts []string
	for len(d) > 3 {
		parts = append([]string{d[len(d)-3:]}, parts...)
		d = d[:len(d)-3]
	}
	return strings.Join(append([]string{d}, parts...), ",")
}

func groupIndian(d string) string {
	if len(d) <= 3 {
		return d
	}
	head, tail := d[:len(d)-3], d[len(d)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

func terminatingPlaces(r *big.Rat) (int, bool) {
	d := new(big.Int).Set(r.Denom())
	counts := [2]int{}
	for i, p := range []int64{2, 5} {
		bp := big.NewInt(p)
		for {
			q, m := new(big.Int).QuoRem(d, bp, new(big.Int))
			if m.Sign() != 0 {
				break
			}
			d = q
			counts[i]++
		}
	}
	if d.Cmp(bigOne) != 0 {
		return 0, false
	}
	if counts[0] > counts[1] {
		return counts[0], true
	}
	return counts[1], true
}

var bigOne = big.NewInt(1)

func significantRat(r *big.Rat, digits int) string {
	scale := ratScale(r)
	if places := digits - 1 - scale; places >= 0 && !tooSmallToSpellOut(scale) {
		return trimDecimal(r.FloatString(places))
	}
	mantissa, scale := ratMantissa(r, scale, digits)
	return mantissa + exponentSuffix(scale)
}

func ratMantissa(r *big.Rat, scale, digits int) (string, int) {
	text := trimDecimal(new(big.Rat).Quo(r, ratPow10(scale)).FloatString(digits - 1))
	if len(strings.TrimLeft(text, "-0.")) == 0 {
		return text, scale
	}
	if whole := strings.TrimPrefix(text, "-"); len(whole) > 1 && whole[1] != '.' {
		scale++
		text = trimDecimal(new(big.Rat).Quo(r, ratPow10(scale)).FloatString(digits - 1))
	}
	return text, scale
}

func exponentSuffix(scale int) string {
	sign := "+"
	if scale < 0 {
		sign = "-"
		scale = -scale
	}
	text := strconv.Itoa(scale)
	if len(text) < 2 {
		text = "0" + text
	}
	return "e" + sign + text
}

func ratPow10(k int) *big.Rat {
	p := new(big.Int).Exp(bigTen, big.NewInt(int64(abs(k))), nil)
	if k < 0 {
		return new(big.Rat).SetFrac(bigOne, p)
	}
	return new(big.Rat).SetInt(p)
}

func ratScale(r *big.Rat) int {
	n := new(big.Int).Abs(r.Num())
	if n.Sign() == 0 {
		return 0
	}
	d := r.Denom()
	k := len(n.String()) - len(d.String())
	for cmpPow10(n, d, k) < 0 {
		k--
	}
	for cmpPow10(n, d, k+1) >= 0 {
		k++
	}
	return k
}

func cmpPow10(n, d *big.Int, k int) int {
	p := new(big.Int).Exp(bigTen, big.NewInt(int64(abs(k))), nil)
	if k >= 0 {
		return n.Cmp(new(big.Int).Mul(d, p))
	}
	return new(big.Int).Mul(n, p).Cmp(d)
}

func abs(k int) int {
	if k < 0 {
		return -k
	}
	return k
}

var bigTen = big.NewInt(10)

func significantDigits(text string) int {
	seen, started := 0, false
	for _, r := range text {
		if r < '0' || r > '9' {
			continue
		}
		if r != '0' {
			started = true
		}
		if started {
			seen++
		}
	}
	return seen
}

func exactSignificantDigits(r *big.Rat) int {
	places, ok := terminatingPlaces(r)
	if !ok {
		return 0
	}
	scaled := new(big.Int).Mul(new(big.Int).Abs(r.Num()), new(big.Int).Exp(bigTen, big.NewInt(int64(places)), nil))
	scaled.Quo(scaled, r.Denom())
	if scaled.Sign() == 0 {
		return 0
	}
	return len(scaled.String())
}

func covarianceNote(v value.Value) string {
	for _, c := range v.Unc.Budget {
		if c.Covariance {
			return "the budget includes a covariance term from declared correlation"
		}
	}
	return ""
}

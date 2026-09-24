package format

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

const Digits = 12

type Style struct {
	Digits  int
	Grouped bool
	Indian  bool
}

var Default = Style{Digits: Digits}

func Text(v value.Value, notes []parse.Note) string { return Default.Text(v, notes) }

func (s Style) Text(v value.Value, notes []parse.Note) string {
	var b strings.Builder
	b.WriteString(s.Number(v))
	if v.Unit != "" {
		b.WriteString(" " + v.Unit)
	}
	if !v.Num.IsExact() {
		if _, digits, capped := s.NumberBall(v.Num.Ball(BallPrec)); capped {
			b.WriteString("\n  " + strconv.Itoa(digits) + " digits guaranteed (precision cap reached)")
		}
	}
	for _, n := range notes {
		b.WriteString("\n  read as " + strings.TrimPrefix(n.Msg, "read as "))
	}
	return b.String()
}

func (s Style) Number(v value.Value) string {
	if !v.Num.IsExact() {
		text, _ := provenDecimal(v.Num.Ball(BallPrec), s.digitBudget())
		return s.groupDecimal(text)
	}
	r := v.Num.Exact().Rat()
	if r.IsInt() {
		return s.group(r.Num().String())
	}
	dec := trimDecimal(r.FloatString(s.digitBudget()))
	if terminates(r) {
		return s.groupDecimal(dec)
	}
	return r.RatString() + "  ≈ " + s.groupDecimal(dec)
}

func trimDecimal(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

func (s Style) groupDecimal(dec string) string {
	if i := strings.IndexByte(dec, '.'); i >= 0 {
		return s.group(dec[:i]) + dec[i:]
	}
	return s.group(dec)
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

func terminates(r *big.Rat) bool {
	d := new(big.Int).Set(r.Denom())
	for _, p := range []int64{2, 5} {
		bp := big.NewInt(p)
		for {
			q, m := new(big.Int).QuoRem(d, bp, new(big.Int))
			if m.Sign() != 0 {
				break
			}
			d = q
		}
	}
	return d.Cmp(big.NewInt(1)) == 0
}

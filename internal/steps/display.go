package steps

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/units"
)

const significantShown = 12

type unitTerm struct {
	sym string
	exp int
}

type display struct {
	terms []unitTerm
	known bool
}

func scalarDisplay() display { return display{known: true} }

func (d display) add(o display, sign int) display {
	out := display{known: d.known && o.known, terms: append([]unitTerm(nil), d.terms...)}
	for _, t := range o.terms {
		merged := false
		for i := range out.terms {
			if out.terms[i].sym == t.sym {
				out.terms[i].exp += sign * t.exp
				merged = true
				break
			}
		}
		if !merged {
			out.terms = append(out.terms, unitTerm{t.sym, sign * t.exp})
		}
	}
	return out
}

func (d display) scale(k int) display {
	out := display{known: d.known}
	for _, t := range d.terms {
		out.terms = append(out.terms, unitTerm{t.sym, t.exp * k})
	}
	return out
}

func (d display) simplified() display {
	out := display{known: d.known, terms: append([]unitTerm(nil), d.terms...)}
	for i := range out.terms {
		for j := range out.terms {
			a, b := &out.terms[i], &out.terms[j]
			if i == j || a.exp <= 0 || b.exp >= 0 {
				continue
			}
			ua, okA := units.Lookup(a.sym)
			ub, okB := units.Lookup(b.sym)
			if !okA || !okB || ua.Dim != ub.Dim || ua.Offset.Sign() != 0 || ub.Offset.Sign() != 0 {
				continue
			}
			shared := min(a.exp, -b.exp)
			a.exp -= shared
			b.exp += shared
		}
	}
	return out
}

func (d display) text() string {
	d = d.simplified()
	var top, bottom []string
	part := func(sym string, exp int) string {
		if strings.ContainsAny(sym, "/·^") && (exp != 1 || len(d.terms) > 1) {
			sym = "(" + sym + ")"
		}
		if exp == 1 {
			return sym
		}
		return sym + "^" + strconv.Itoa(exp)
	}
	for _, t := range d.terms {
		switch {
		case t.exp > 0:
			top = append(top, part(t.sym, t.exp))
		case t.exp < 0:
			bottom = append(bottom, part(t.sym, -t.exp))
		}
	}
	if len(top) == 0 && len(bottom) == 0 {
		return ""
	}
	text := strings.Join(top, "·")
	if text == "" {
		text = "1"
	}
	for _, b := range bottom {
		text += "/" + b
	}
	return text
}

func splitValue(value string) (string, string) {
	number, unit, _ := strings.Cut(value, " ")
	return number, unit
}

func ownDisplay(value string) display {
	_, unit := splitValue(value)
	if unit == "" {
		return scalarDisplay()
	}
	if terms, ok := parseUnitText(unit); ok {
		return display{known: true, terms: terms}
	}
	if strings.ContainsAny(unit, " ±") {
		return display{}
	}
	return display{known: true, terms: []unitTerm{{unit, 1}}}
}

func parseUnitText(text string) ([]unitTerm, bool) {
	text = strings.TrimSuffix(strings.TrimPrefix(text, "("), ")")
	if strings.ContainsAny(text, "() ") {
		return nil, false
	}
	var out []unitTerm
	sign := 1
	start := 0
	flush := func(end int) bool {
		part := text[start:end]
		if part == "" {
			return false
		}
		sym, power, hasPower := strings.Cut(part, "^")
		exp := 1
		if hasPower {
			k, err := strconv.Atoi(power)
			if err != nil {
				return false
			}
			exp = k
		}
		if sym != "1" {
			out = append(out, unitTerm{sym, sign * exp})
		}
		return true
	}
	for i, r := range text {
		if r == '/' || r == '·' || r == '*' {
			if !flush(i) {
				return nil, false
			}
			if r == '/' {
				sign = -1
			}
			start = i + len(string(r))
		}
	}
	if !flush(len(text)) {
		return nil, false
	}
	return out, true
}

func (w *walker) displayOf(id prov.NodeID) display {
	if d, ok := w.shown[id]; ok {
		return d
	}
	n, ok := w.nodes[id]
	if !ok {
		return display{}
	}
	w.shown[id] = display{}
	d := w.computeDisplay(n)
	w.shown[id] = d
	return d
}

func (w *walker) computeDisplay(n prov.NodeOut) display {
	switch n.Kind {
	case kindLiteral, kindConstant, kindVariable, kindUnit:
		return ownDisplay(n.Value)
	case kindOperation:
	default:
		return display{}
	}
	kids := n.Children
	switch {
	case len(kids) == 1 && (n.Label == "-" || n.Label == "+"):
		return w.displayOf(kids[0])
	case len(kids) == 1 && n.Label == "%":
		return scalarDisplay()
	case len(kids) != 2:
		return display{}
	}
	left, right := w.displayOf(kids[0]), w.displayOf(kids[1])
	switch n.Label {
	case "*", "·", "×", "of":
		return left.add(right, 1)
	case "/", "÷":
		return left.add(right, -1)
	case "+", "-":
		if left.known {
			return left
		}
		return right
	case "^":
		exponent, err := strconv.Atoi(w.nodes[kids[1]].Value)
		if err != nil {
			return display{}
		}
		return left.scale(exponent)
	}
	return display{}
}

func unitFactor(text string) (*big.Rat, units.Dim, bool) {
	if text == "" {
		return big.NewRat(1, 1), units.Dim{}, true
	}
	u, ok := units.Lookup(text)
	if !ok || u.Offset.Sign() != 0 {
		return nil, units.Dim{}, false
	}
	return new(big.Rat).Set(u.Factor.Rat()), u.Dim, true
}

func (w *walker) shownValue(n prov.NodeOut) string {
	if n.Kind != kindOperation {
		return n.Value
	}
	d := w.displayOf(n.ID)
	if !d.known {
		w.rawSI = w.rawSI || hasUnit(n.Value)
		return n.Value
	}
	number, unit := splitValue(n.Value)
	target := d.text()
	if unit == target {
		return n.Value
	}
	amount, ok := new(big.Rat).SetString(number)
	if !ok {
		w.rawSI = w.rawSI || hasUnit(n.Value)
		return n.Value
	}
	from, fromDim, ok := unitFactor(unit)
	if !ok {
		w.rawSI = true
		return n.Value
	}
	to, toDim, ok := unitFactor(target)
	if !ok || fromDim != toDim || to.Sign() == 0 {
		w.rawSI = w.rawSI || hasUnit(n.Value)
		return n.Value
	}
	converted := new(big.Rat).Quo(new(big.Rat).Mul(amount, from), to)
	text := shortNumber(converted)
	if target != "" {
		text += " " + target
	}
	return text
}

func hasUnit(value string) bool {
	_, unit := splitValue(value)
	return unit != ""
}

func shortNumber(r *big.Rat) string {
	if r.Sign() == 0 {
		return "0"
	}
	d := new(big.Int).Set(r.Denom())
	for _, p := range []int64{2, 5} {
		for new(big.Int).Mod(d, big.NewInt(p)).Sign() == 0 {
			d.Quo(d, big.NewInt(p))
		}
	}
	if d.Cmp(big.NewInt(1)) == 0 {
		text := decimalText(r)
		if digitCount(text) <= significantShown+3 {
			return text
		}
	}
	f, _ := new(big.Float).SetPrec(200).SetRat(r).Float64()
	return strconv.FormatFloat(f, 'g', significantShown, 64)
}

func digitCount(text string) int {
	count := 0
	for _, r := range text {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

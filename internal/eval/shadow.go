package eval

import (
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/num"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	mantissaBits   = 53
	minBitsLost    = 1
	trustedDigits  = 6
	maxShadowDepth = 512
)

type LossSite struct {
	Op       string
	Pos      lex.Pos
	BitsLost float64
}

type Shadow struct {
	Value    float64
	Lost     []LossSite
	si       float64
	modelled bool
}

func (e *Env) Shadow(n parse.Node) Shadow {
	s := &shadowPass{env: e, prec: DefaultCtx().Prec}
	v, ok := s.walk(n, nil, 0)
	if !ok {
		return Shadow{Value: math.NaN(), si: math.NaN()}
	}
	sort.SliceStable(s.lost, func(i, j int) bool { return s.lost[i].BitsLost > s.lost[j].BitsLost })
	for i := range s.lost {
		s.lost[i].Pos = e.at(s.lost[i].Pos)
	}
	return Shadow{Value: v.num, si: v.si(), Lost: s.lost, modelled: true}
}

type shadowPass struct {
	env  *Env
	prec uint
	lost []LossSite
}

type shown struct {
	num    float64
	factor float64
}

func plain(x float64) shown { return shown{num: x, factor: 1} }

func (v shown) si() float64 { return v.num * v.factor }

func (v shown) whole() bool {
	return !math.IsInf(v.num, 0) && !math.IsNaN(v.num) && v.num == math.Trunc(v.num)
}

type scope map[string]shown

func (s *shadowPass) walk(n parse.Node, sc scope, depth int) (shown, bool) {
	if depth > maxShadowDepth {
		return shown{}, false
	}
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return shown{}, false
		}
		return plain(exactFloat(x)), true
	case *parse.Name:
		return s.name(t, sc)
	case *parse.Unary:
		return s.unary(t, sc, depth)
	case *parse.Binary:
		return s.binary(t, sc, depth)
	case *parse.Call:
		return s.call(t, sc, depth)
	case *parse.Assign:
		if len(t.Params) > 0 {
			return shown{}, false
		}
		return s.walk(t.Body, sc, depth+1)
	}
	return shown{}, false
}

func (s *shadowPass) name(t *parse.Name, sc scope) (shown, bool) {
	if x, ok := sc[t.Text]; ok {
		return x, true
	}
	if v, ok := s.env.vars[t.Text]; ok {
		if v.IsAffine() {
			return shown{}, false
		}
		return shown{numFloat(v.Num, s.prec), exactFloat(v.FactorOrOne())}, true
	}
	if c, ok := consts.Lookup(t.Text); ok {
		if u, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
			if u.Offset.Sign() != 0 {
				return shown{}, false
			}
			return shown{exactFloat(c.Value), exactFloat(u.Factor)}, true
		}
		return plain(exactFloat(c.Value)), true
	}
	switch t.Text {
	case "pi", "π":
		return plain(math.Pi), true
	case "e":
		return plain(math.E), true
	}
	if u, ok := units.Lookup(t.Text); ok {
		if u.Offset.Sign() != 0 {
			return shown{}, false
		}
		return shown{1, exactFloat(u.Factor)}, true
	}
	return shown{}, false
}

func (s *shadowPass) unary(t *parse.Unary, sc scope, depth int) (shown, bool) {
	x, ok := s.walk(t.X, sc, depth+1)
	if !ok {
		return shown{}, false
	}
	switch t.Op {
	case "-":
		return shown{-x.num, x.factor}, true
	case "%":
		return plain(x.si() / 100), true
	case "!":
		return factorialFloat(x.si())
	}
	return shown{}, false
}

func factorialFloat(x float64) (shown, bool) {
	if x != math.Trunc(x) || x < 0 || x > maxFactorial {
		return shown{}, false
	}
	out := 1.0
	for k := 2.0; k <= x; k++ {
		out *= k
	}
	return plain(out), true
}

const maxFactorial = 20000

func (s *shadowPass) binary(t *parse.Binary, sc scope, depth int) (shown, bool) {
	if t.Implicit {
		if x, ok, handled := s.implicit(t, sc, depth); handled {
			return x, ok
		}
	}
	if t.Op == "to" || t.Op == "in" {
		return s.conversion(t, sc, depth)
	}
	l, ok := s.walk(t.L, sc, depth+1)
	if !ok {
		return shown{}, false
	}
	if t.Op == "±" || t.Op == "+-" {
		return l, true
	}
	r, ok := s.walk(t.R, sc, depth+1)
	if !ok {
		return shown{}, false
	}
	switch t.Op {
	case "+", "-":
		if isPercentNode(t.R) {
			return shown{l.num * (1 + signOf(t.Op)*r.si()), l.factor}, true
		}
		return s.sum(t, l, r)
	case "*", "×", "·":
		return plain(l.si() * r.si()), true
	case "/", "÷":
		return plain(l.si() / r.si()), true
	case "^":
		return plain(math.Pow(l.si(), r.si())), true
	case "mod":
		if r.si() == 0 {
			return shown{}, false
		}
		m := math.Mod(l.si(), r.si())
		if m != 0 && (m < 0) != (r.si() < 0) {
			m += r.si()
		}
		return plain(m), true
	}
	return shown{}, false
}

func (s *shadowPass) sum(t *parse.Binary, l, r shown) (shown, bool) {
	addend := r.si() / l.factor
	out := l.num + addend
	if t.Op == "-" {
		out = l.num - addend
	}
	s.note(t.Op, t.P, bitsLost(l.num, addend, out))
	return shown{out, l.factor}, true
}

func (s *shadowPass) implicit(t *parse.Binary, sc scope, depth int) (shown, bool, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return shown{}, false, false
	}
	if _, bound := sc[name.Text]; bound {
		return shown{}, false, false
	}
	u, _, _, asUnit := s.env.implicitReading(name)
	if !asUnit {
		return shown{}, false, false
	}
	if u.Offset.Sign() != 0 {
		return shown{}, false, true
	}
	l, ok := s.walk(t.L, sc, depth+1)
	if !ok {
		return shown{}, false, true
	}
	factor := exactFloat(u.Factor)
	if u.Dim.IsNone() {
		return plain(l.num * factor), true, true
	}
	return shown{l.num, factor}, true, true
}

func (s *shadowPass) conversion(t *parse.Binary, sc scope, depth int) (shown, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return shown{}, false
	}
	to, ok := units.Lookup(name.Text)
	if !ok || to.Offset.Sign() != 0 {
		return shown{}, false
	}
	l, ok := s.walk(t.L, sc, depth+1)
	if !ok {
		return shown{}, false
	}
	factor := exactFloat(to.Factor)
	if factor == 0 {
		return shown{}, false
	}
	return shown{l.si() / factor, factor}, true
}

func (s *shadowPass) call(t *parse.Call, sc scope, depth int) (shown, bool) {
	if f, ok := s.env.funcs[t.Fn]; ok {
		if len(f.params) != len(t.Args) {
			return shown{}, false
		}
		inner := scope{}
		for i, param := range f.params {
			a, ok := s.walk(t.Args[i], sc, depth+1)
			if !ok {
				return shown{}, false
			}
			inner[param] = a
		}
		return s.walk(f.body, inner, depth+1)
	}
	wearing := shown{}
	args := make([]float64, len(t.Args))
	for i, a := range t.Args {
		x, ok := s.walk(a, sc, depth+1)
		if !ok {
			return shown{}, false
		}
		if i == 0 {
			wearing = x
		}
		args[i] = x.si()
	}
	if len(args) == 1 {
		switch t.Fn {
		case "abs":
			return shown{math.Abs(wearing.num), wearing.factor}, true
		case "floor", "ceil", "round":
			return shown{keepsItsUnit[t.Fn](wearing.num), wearing.factor}, true
		}
	}
	if f, ok := shadowUnary[t.Fn]; ok && len(args) == 1 {
		return plain(f(args[0])), true
	}
	switch t.Fn {
	case "root":
		if len(args) == 2 && args[1] != 0 {
			return plain(math.Pow(args[0], 1/args[1])), true
		}
	case "atan2":
		if len(args) == 2 {
			return plain(math.Atan2(args[0], args[1])), true
		}
	case "expand":
		if len(args) == 2 {
			return wearing, true
		}
	}
	return shown{}, false
}

var keepsItsUnit = map[string]func(float64) float64{
	"floor": math.Floor,
	"ceil":  math.Ceil,
	"round": func(x float64) float64 { return math.Floor(x + 0.5) },
}

var shadowUnary = map[string]func(float64) float64{
	"abs":   math.Abs,
	"sqrt":  math.Sqrt,
	"exp":   math.Exp,
	"ln":    math.Log,
	"log":   math.Log,
	"log10": math.Log10,
	"log2":  math.Log2,
	"sinh":  math.Sinh,
	"cosh":  math.Cosh,
	"tanh":  math.Tanh,
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"asin":  math.Asin,
	"acos":  math.Acos,
	"atan":  math.Atan,
}

func (s *shadowPass) note(op string, pos lex.Pos, bits float64) {
	if bits < minBitsLost {
		return
	}
	s.lost = append(s.lost, LossSite{Op: op, Pos: pos, BitsLost: bits})
}

func bitsLost(a, b, out float64) float64 {
	largest := math.Max(math.Abs(a), math.Abs(b))
	if largest == 0 || math.IsInf(largest, 0) || math.IsNaN(out) {
		return 0
	}
	if out == 0 {
		return mantissaBits
	}
	bits := math.Log2(largest / math.Abs(out))
	switch {
	case math.IsNaN(bits) || bits < 0:
		return 0
	case bits > mantissaBits:
		return mantissaBits
	}
	return bits
}

func digitsAgreed(sh Shadow, proven value.Value) (float64, bool) {
	if !sh.modelled || math.IsNaN(sh.Value) {
		return 0, false
	}
	p := provenFloat(proven, DefaultCtx().Prec)
	if p == nil {
		return 0, false
	}
	if math.IsInf(sh.Value, 0) {
		return 0, true
	}
	s := new(big.Float).SetPrec(p.Prec()).SetFloat64(sh.Value)
	gap := new(big.Float).SetPrec(p.Prec()).Sub(s, p)
	if gap.Sign() == 0 {
		return math.Inf(1), true
	}
	if p.Sign() == 0 {
		return 0, true
	}
	rel := new(big.Float).SetPrec(p.Prec()).Quo(new(big.Float).Abs(gap), new(big.Float).Abs(p))
	f, _ := rel.Float64()
	if f <= 0 {
		return math.Inf(1), true
	}
	if math.IsInf(f, 0) {
		return 0, true
	}
	return -math.Log10(f), true
}

func provenFloat(v value.Value, prec uint) *big.Float {
	if v.Num.IsExact() {
		return new(big.Float).SetPrec(prec).SetRat(v.Num.Exact().Rat())
	}
	mid := v.Num.Ball(prec).Mid
	if mid == nil {
		return nil
	}
	return new(big.Float).SetPrec(prec).Set(mid)
}

type Explanation struct {
	Shadow      float64
	ShadowExact string
	Proven      value.Value
	Digits      float64
	WrongDigits int
	Modelled    bool
	Lost        []LossSite
	Warning     string
}

func (x Explanation) Worst() (LossSite, bool) {
	if len(x.Lost) == 0 {
		return LossSite{}, false
	}
	return x.Lost[0], true
}

func (e *Env) Why(n parse.Node, proven value.Value) Explanation {
	sh := e.Shadow(n)
	digits, ok := digitsAgreed(sh, proven)
	gap, _ := integerCheck(sh, proven)
	return Explanation{
		Shadow:      sh.Value,
		ShadowExact: gap.Float,
		Proven:      proven,
		Digits:      digits,
		WrongDigits: gap.Wrong,
		Modelled:    ok,
		Lost:        sh.Lost,
		Warning:     warn(sh, proven),
	}
}

func (e *Env) Warning(n parse.Node, proven value.Value) string {
	return warn(e.Shadow(n), proven)
}

func warn(sh Shadow, proven value.Value) string {
	if !sh.modelled {
		return ""
	}
	if digits, ok := digitsAgreed(sh, proven); ok && digits < trustedDigits {
		return "float64 would answer " + shortFloat(sh.Value) + " here, agreeing with the proven value to " + digitWord(digits) + worstClause(sh)
	}
	if gap, ok := integerCheck(sh, proven); ok && gap.Wrong > 0 {
		return "float64 gives " + gap.Float + " here; " + wrongDigits(gap.Wrong) + worstClause(sh)
	}
	return ""
}

func worstClause(sh Shadow) string {
	if len(sh.Lost) == 0 {
		return ""
	}
	return "; the worst loss is " + bitWord(sh.Lost[0].BitsLost) + " at the " + sh.Lost[0].Op + " at " + sh.Lost[0].Pos.String()
}

type integerGap struct {
	Float   string
	Printed string
	Wrong   int
}

func integerCheck(sh Shadow, proven value.Value) (integerGap, bool) {
	printed, ok := printedInteger(proven)
	if !sh.modelled || !ok {
		return integerGap{}, false
	}
	if math.IsInf(sh.Value, 0) || math.IsNaN(sh.Value) || sh.Value != math.Trunc(sh.Value) {
		return integerGap{}, false
	}
	f, _ := new(big.Float).SetPrec(mantissaBits).SetFloat64(sh.Value).Int(nil)
	gap := integerGap{Printed: printed.String(), Float: f.String()}
	if f.Cmp(printed) == 0 {
		return gap, true
	}
	relative, _ := digitsAgreed(sh, proven)
	gap.Wrong = digitCount(printed) - agreeingDigits(f, printed, relative)
	switch {
	case gap.Wrong < 1:
		gap.Wrong = 0
	case gap.Wrong > digitCount(printed):
		gap.Wrong = digitCount(printed)
	}
	return gap, true
}

func printedInteger(v value.Value) (*big.Int, bool) {
	if !v.Num.IsExact() || !v.Num.Exact().IsInt() || v.Offset.Sign() != 0 {
		return nil, false
	}
	return new(big.Int).Set(v.Num.Exact().Rat().Num()), true
}

func digitCount(n *big.Int) int { return len(new(big.Int).Abs(n).String()) }

func agreeingDigits(f, p *big.Int, relative float64) int {
	fs := new(big.Int).Abs(f).String()
	ps := new(big.Int).Abs(p).String()
	if f.Sign() == p.Sign() && len(fs) == len(ps) {
		n := 0
		for n < len(ps) && fs[n] == ps[n] {
			n++
		}
		return n
	}
	if math.IsInf(relative, 1) {
		return len(ps)
	}
	if relative < 0 || math.IsNaN(relative) {
		return 0
	}
	return int(relative)
}

func wrongDigits(n int) string {
	if n == 1 {
		return "the last digit is wrong"
	}
	return "the last " + strconv.Itoa(n) + " digits are wrong"
}

func (e *Env) at(p lex.Pos) lex.Pos {
	if e.line > 0 {
		p.Line = e.line
	}
	return p
}

func digitWord(digits float64) string {
	if math.IsInf(digits, 1) {
		return "every digit it has"
	}
	whole := int(digits)
	switch whole {
	case 0:
		return "no significant digits"
	case 1:
		return "1 significant digit"
	}
	return strconv.Itoa(whole) + " significant digits"
}

func bitWord(bits float64) string {
	rounded := int(math.Round(bits))
	if rounded == 1 {
		return "1 bit"
	}
	return strconv.Itoa(rounded) + " bits"
}

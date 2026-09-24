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
	modelled bool
}

func (e *Env) Shadow(n parse.Node) Shadow {
	s := &shadowPass{env: e, prec: DefaultCtx().Prec}
	v, ok := s.walk(n, nil, 0)
	if !ok {
		return Shadow{Value: math.NaN()}
	}
	sort.SliceStable(s.lost, func(i, j int) bool { return s.lost[i].BitsLost > s.lost[j].BitsLost })
	for i := range s.lost {
		s.lost[i].Pos = e.at(s.lost[i].Pos)
	}
	return Shadow{Value: v, Lost: s.lost, modelled: true}
}

type shadowPass struct {
	env  *Env
	prec uint
	lost []LossSite
}

type scope map[string]float64

func (s *shadowPass) walk(n parse.Node, sc scope, depth int) (float64, bool) {
	if depth > maxShadowDepth {
		return 0, false
	}
	switch t := n.(type) {
	case *parse.NumLit:
		x, err := num.ParseExact(t.Text)
		if err != nil {
			return 0, false
		}
		return exactFloat(x), true
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
			return 0, false
		}
		return s.walk(t.Body, sc, depth+1)
	}
	return 0, false
}

func (s *shadowPass) name(t *parse.Name, sc scope) (float64, bool) {
	if x, ok := sc[t.Text]; ok {
		return x, true
	}
	if v, ok := s.env.vars[t.Text]; ok {
		if v.IsAffine() {
			return 0, false
		}
		return siFloat(v, s.prec), true
	}
	if c, ok := consts.Lookup(t.Text); ok {
		if u, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
			if u.Offset.Sign() != 0 {
				return 0, false
			}
			return exactFloat(c.Value) * exactFloat(u.Factor), true
		}
		return exactFloat(c.Value), true
	}
	switch t.Text {
	case "pi", "π":
		return math.Pi, true
	case "e":
		return math.E, true
	}
	if u, ok := units.Lookup(t.Text); ok {
		if u.Offset.Sign() != 0 {
			return 0, false
		}
		return exactFloat(u.Factor), true
	}
	return 0, false
}

func (s *shadowPass) unary(t *parse.Unary, sc scope, depth int) (float64, bool) {
	x, ok := s.walk(t.X, sc, depth+1)
	if !ok {
		return 0, false
	}
	switch t.Op {
	case "-":
		return -x, true
	case "%":
		return x / 100, true
	case "!":
		return factorialFloat(x)
	}
	return 0, false
}

func factorialFloat(x float64) (float64, bool) {
	if x != math.Trunc(x) || x < 0 || x > maxFactorial {
		return 0, false
	}
	out := 1.0
	for k := 2.0; k <= x; k++ {
		out *= k
	}
	return out, true
}

const maxFactorial = 20000

func (s *shadowPass) binary(t *parse.Binary, sc scope, depth int) (float64, bool) {
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
		return 0, false
	}
	if t.Op == "±" || t.Op == "+-" {
		return l, true
	}
	r, ok := s.walk(t.R, sc, depth+1)
	if !ok {
		return 0, false
	}
	switch t.Op {
	case "+", "-":
		if isPercentNode(t.R) {
			return l * (1 + signOf(t.Op)*r), true
		}
		return s.sum(t, l, r), true
	case "*", "×", "·":
		return l * r, true
	case "/", "÷":
		return l / r, true
	case "^":
		return math.Pow(l, r), true
	case "mod":
		if r == 0 {
			return 0, false
		}
		m := math.Mod(l, r)
		if m != 0 && (m < 0) != (r < 0) {
			m += r
		}
		return m, true
	}
	return 0, false
}

func (s *shadowPass) sum(t *parse.Binary, l, r float64) float64 {
	out := l + r
	if t.Op == "-" {
		out = l - r
	}
	s.note(t.Op, t.P, bitsLost(l, r, out))
	return out
}

func (s *shadowPass) implicit(t *parse.Binary, sc scope, depth int) (float64, bool, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return 0, false, false
	}
	if _, bound := sc[name.Text]; bound {
		return 0, false, false
	}
	u, _, _, asUnit := s.env.implicitReading(name)
	if !asUnit {
		return 0, false, false
	}
	if u.Offset.Sign() != 0 {
		return 0, false, true
	}
	l, ok := s.walk(t.L, sc, depth+1)
	if !ok {
		return 0, false, true
	}
	return l * exactFloat(u.Factor), true, true
}

func (s *shadowPass) conversion(t *parse.Binary, sc scope, depth int) (float64, bool) {
	name, ok := t.R.(*parse.Name)
	if !ok {
		return 0, false
	}
	to, ok := units.Lookup(name.Text)
	if !ok || to.Offset.Sign() != 0 {
		return 0, false
	}
	return s.walk(t.L, sc, depth+1)
}

func (s *shadowPass) call(t *parse.Call, sc scope, depth int) (float64, bool) {
	if f, ok := s.env.funcs[t.Fn]; ok {
		if len(f.params) != len(t.Args) {
			return 0, false
		}
		inner := scope{}
		for i, param := range f.params {
			a, ok := s.walk(t.Args[i], sc, depth+1)
			if !ok {
				return 0, false
			}
			inner[param] = a
		}
		return s.walk(f.body, inner, depth+1)
	}
	args := make([]float64, len(t.Args))
	for i, a := range t.Args {
		x, ok := s.walk(a, sc, depth+1)
		if !ok {
			return 0, false
		}
		args[i] = x
	}
	if f, ok := shadowUnary[t.Fn]; ok && len(args) == 1 {
		return f(args[0]), true
	}
	switch t.Fn {
	case "round":
		if len(args) == 1 {
			return math.Floor(args[0] + 0.5), true
		}
	case "root":
		if len(args) == 2 && args[1] != 0 {
			return math.Pow(args[0], 1/args[1]), true
		}
	case "atan2":
		if len(args) == 2 {
			return math.Atan2(args[0], args[1]), true
		}
	case "expand":
		if len(args) == 2 {
			return args[0], true
		}
	}
	return 0, false
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
	"floor": math.Floor,
	"ceil":  math.Ceil,
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
	n := v.SI(prec)
	if n.IsExact() {
		return new(big.Float).SetPrec(prec).SetRat(n.Exact().Rat())
	}
	mid := n.Ball(prec).Mid
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
	exact, wrong, _ := integerCheck(sh, proven)
	return Explanation{
		Shadow:      sh.Value,
		ShadowExact: exact,
		Proven:      proven,
		Digits:      digits,
		WrongDigits: wrong,
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
	if exact, wrong, ok := integerCheck(sh, proven); ok && wrong > 0 {
		return "float64 gives " + exact + " here; " + wrongDigits(wrong) + worstClause(sh)
	}
	return ""
}

func worstClause(sh Shadow) string {
	if len(sh.Lost) == 0 {
		return ""
	}
	return "; the worst loss is " + bitWord(sh.Lost[0].BitsLost) + " at the " + sh.Lost[0].Op + " at " + sh.Lost[0].Pos.String()
}

func integerCheck(sh Shadow, proven value.Value) (string, int, bool) {
	p, ok := printedInteger(proven)
	if !sh.modelled || !ok || math.IsInf(sh.Value, 0) || math.IsNaN(sh.Value) {
		return "", 0, false
	}
	f, _ := new(big.Float).SetPrec(mantissaBits).SetFloat64(sh.Value).Int(nil)
	return f.String(), trailingWrong(f, p), true
}

func printedInteger(v value.Value) (*big.Int, bool) {
	if !v.Num.IsExact() || !v.Num.Exact().IsInt() {
		return nil, false
	}
	if v.Offset.Sign() != 0 || v.FactorOrOne().Cmp(num.FromInt(1)) != 0 {
		return nil, false
	}
	return new(big.Int).Set(v.Num.Exact().Rat().Num()), true
}

func wrongDigits(n int) string {
	if n == 1 {
		return "the last digit is wrong"
	}
	return "the last " + strconv.Itoa(n) + " digits are wrong"
}

func trailingWrong(f, p *big.Int) int {
	fs := new(big.Int).Abs(f).String()
	ps := new(big.Int).Abs(p).String()
	if f.Sign() != p.Sign() || len(fs) != len(ps) {
		return len(ps)
	}
	for i := range ps {
		if fs[i] != ps[i] {
			return len(ps) - i
		}
	}
	return 0
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

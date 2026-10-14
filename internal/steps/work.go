package steps

import (
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/prov"
)

const (
	maxWorkItems    = 12
	maxPowerShown   = 6
	maxFactorLines  = 14
	maxFactorValue  = 1_000_000_000_000
	maxBaseLines    = 16
	maxFibListed    = 30
	maxChooseShown  = 20
	maxModpowShown  = 18
	maxSearchedFrom = 1_000_000_000_000
)

func exactOf(text string) (*big.Rat, bool) {
	if text == "" || strings.ContainsAny(text, " …≈±i[") {
		return nil, false
	}
	return new(big.Rat).SetString(text)
}

func exactAll(texts []string) ([]*big.Rat, bool) {
	out := make([]*big.Rat, len(texts))
	for i, t := range texts {
		r, ok := exactOf(t)
		if !ok {
			return nil, false
		}
		out[i] = r
	}
	return out, true
}

func wholeOf(text string) (*big.Int, bool) {
	r, ok := exactOf(text)
	if !ok || !r.IsInt() {
		return nil, false
	}
	return new(big.Int).Set(r.Num()), true
}

func n(r *big.Rat) string { return shortNumber(r) }

func joinRats(rs []*big.Rat, sep string) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		parts[i] = parenNegative(n(r))
	}
	return strings.Join(parts, sep)
}

func parenNegative(text string) string {
	if strings.HasPrefix(text, "-") {
		return "(" + text + ")"
	}
	return text
}

func (w *walker) items(child prov.NodeOut) ([]string, bool) {
	switch child.Label {
	case "list":
		return w.values(child), true
	case "range":
		ends := w.values(child)
		if len(ends) < 2 {
			return nil, false
		}
		lo, okLo := wholeOf(ends[0])
		hi, okHi := wholeOf(ends[1])
		step := big.NewInt(1)
		if len(ends) == 3 {
			s, ok := wholeOf(ends[2])
			if !ok || s.Sign() == 0 {
				return nil, false
			}
			step = s
		}
		if !okLo || !okHi {
			return nil, false
		}
		var out []string
		for at := new(big.Int).Set(lo); (step.Sign() > 0 && at.Cmp(hi) <= 0) || (step.Sign() < 0 && at.Cmp(hi) >= 0); at.Add(at, step) {
			out = append(out, at.String())
			if len(out) > maxWorkItems {
				return nil, false
			}
		}
		return out, true
	}
	text := strings.TrimSuffix(strings.TrimPrefix(child.Value, "["), "]")
	if text == child.Value || strings.Contains(text, "…") {
		return nil, false
	}
	return strings.Split(text, ", "), true
}

func (w *walker) listArgument(n prov.NodeOut) ([]*big.Rat, bool) {
	kids := w.kids(n)
	var texts []string
	if len(kids) == 1 {
		items, ok := w.items(kids[0])
		if !ok {
			if !strings.HasPrefix(kids[0].Value, "[") && !strings.HasPrefix(kids[0].Value, "list of") {
				texts = []string{kids[0].Value}
			} else {
				return nil, false
			}
		} else {
			texts = items
		}
	} else {
		for _, k := range kids {
			texts = append(texts, w.shownValue(k))
		}
	}
	if len(texts) == 0 || len(texts) > maxWorkItems {
		return nil, false
	}
	return exactAll(texts)
}

var exactFunctions = map[string]bool{
	"sum": true, "product": true, "mean": true, "median": true, "min": true, "max": true, "count": true, "len": true,
	"gcd": true, "lcm": true, "nCr": true, "nPr": true, "fib": true, "mod": true, "modpow": true, "modinv": true,
	"totient": true, "digitsum": true, "abs": true, "floor": true, "ceil": true, "round": true, "trunc": true,
	"frac": true, "sign": true, "range": true, "list": true, "index": true, "sort": true, "reverse": true,
	"unique": true, "mode": true, "cumsum": true, "diff": true, "percentile": true, "divisors": true,
	"primes": true, "nextprime": true, "prevprime": true, "isprime": true, "factor": true,
}

func (w *walker) exactNode(id prov.NodeID) bool {
	if w.line.Status == "exact" {
		return true
	}
	node, ok := w.nodes[id]
	if !ok {
		return false
	}
	switch node.Kind {
	case kindLiteral:
		return true
	case kindConstant:
		return false
	case kindOperation:
		if node.Label == "^" && len(node.Children) == 2 {
			if exponent, ok := w.nodes[node.Children[1]]; !ok || !isWholeText(exponent.Value) {
				return false
			}
		}
	case kindFunction:
		if !exactFunctions[node.Label] {
			return false
		}
	}
	for _, child := range node.Children {
		if !w.exactNode(child) {
			return false
		}
	}
	return true
}

func (w *walker) work(node prov.NodeOut) []string {
	if !w.exactNode(node.ID) && !approximateWorkAllowed[node.Label] {
		return nil
	}
	args := w.values(node)
	switch node.Kind {
	case kindOperation:
		return w.operationWork(node, args)
	case kindFunction:
		return w.functionWork(node, args)
	}
	return nil
}

func (w *walker) operationWork(node prov.NodeOut, args []string) []string {
	kids := w.kids(node)
	if len(args) != 2 || len(kids) != 2 {
		return nil
	}
	a, okA := exactOf(args[0])
	b, okB := exactOf(args[1])
	result, okR := exactOf(w.shownValue(node))
	if percent := kids[1]; percent.Kind == kindOperation && percent.Label == "%" && okA && okB && okR {
		factor := new(big.Rat).Add(big.NewRat(1, 1), b)
		sign := "+"
		if node.Label == "-" {
			factor = new(big.Rat).Sub(big.NewRat(1, 1), b)
			sign = "-"
		}
		return []string{n(a) + " × (1 " + sign + " " + n(b) + ") = " + n(a) + " × " + n(factor) + " = " + n(result)}
	}
	if !okA || !okB {
		return nil
	}
	switch node.Label {
	case "of":
		if okR {
			return []string{n(a) + " × " + n(b) + " = " + n(result)}
		}
	case "/", "÷":
		if b.Sign() == 0 {
			return nil
		}
		q := new(big.Rat).Quo(a, b)
		if q.IsInt() || (q.RatString() == a.RatString()+"/"+b.RatString() && terminates(q)) {
			return nil
		}
		return []string{n(a) + " ÷ " + n(b) + " = " + q.RatString() + fractionTail(q)}
	case "^", "**":
		return powerWork(a, b, result, okR)
	case "mod":
		return modWork(a, b)
	}
	return nil
}

func terminates(r *big.Rat) bool {
	d := new(big.Int).Set(r.Denom())
	for _, p := range []int64{2, 5} {
		for new(big.Int).Mod(d, big.NewInt(p)).Sign() == 0 {
			d.Quo(d, big.NewInt(p))
		}
	}
	return d.Cmp(big.NewInt(1)) == 0
}

func fractionTail(q *big.Rat) string {
	if terminates(q) {
		return " = " + decimalText(q)
	}
	return ", which never ends as a decimal, so it is kept as a fraction"
}

func powerWork(a, b, result *big.Rat, okR bool) []string {
	if !b.IsInt() {
		if b.Cmp(big.NewRat(1, 2)) == 0 {
			return []string{n(a) + "^(1/2) = √" + n(a)}
		}
		return nil
	}
	k := b.Num().Int64()
	switch {
	case k >= 2 && k <= maxPowerShown && okR:
		factors := make([]string, k)
		for i := range factors {
			factors[i] = parenNegative(n(a))
		}
		return []string{n(a) + "^" + strconv.FormatInt(k, 10) + " = " + strings.Join(factors, " × ") + " = " + n(result)}
	case k < 0 && k >= -maxPowerShown && a.Sign() != 0:
		positive := new(big.Rat).SetInt64(1)
		for i := int64(0); i < -k; i++ {
			positive.Mul(positive, a)
		}
		inverse := new(big.Rat).Inv(positive)
		return []string{n(a) + "^" + strconv.FormatInt(k, 10) + " = 1 ÷ " + n(a) + "^" + strconv.FormatInt(-k, 10) + " = 1 ÷ " + n(positive) + " = " + inverse.RatString() + tailIfDecimal(inverse)}
	case k == 0:
		return []string{"anything except 0 raised to the power 0 is 1"}
	}
	return nil
}

func tailIfDecimal(r *big.Rat) string {
	if r.IsInt() || !terminates(r) {
		return ""
	}
	return " = " + decimalText(r)
}

func modWork(a, m *big.Rat) []string {
	if !a.IsInt() || !m.IsInt() || m.Sign() == 0 {
		return nil
	}
	q := new(big.Int)
	r := new(big.Int)
	q.DivMod(a.Num(), m.Num(), r)
	if m.Sign() < 0 && r.Sign() > 0 {
		r.Add(r, m.Num())
		q.Add(q, big.NewInt(1))
	}
	return []string{a.Num().String() + " = " + parenNegative(q.String()) + " × " + parenNegative(m.Num().String()) + " + " + r.String() + ", so the remainder is " + r.String()}
}

func (w *walker) functionWork(node prov.NodeOut, args []string) []string {
	switch node.Label {
	case "sum", "product", "mean":
		values, ok := w.listArgument(node)
		if !ok || len(values) < 2 {
			return nil
		}
		return aggregateWork(node.Label, values)
	case "median":
		values, ok := w.listArgument(node)
		if !ok {
			return nil
		}
		return medianWork(values)
	case "variance", "variancep", "stdev", "stdevp":
		values, ok := w.listArgument(node)
		if !ok || len(values) < 2 {
			return nil
		}
		return spreadWork(node.Label, values, node.Value)
	case "gcd", "lcm":
		values, ok := w.listArgument(node)
		if !ok || len(values) < 2 {
			return nil
		}
		return divisorWork(node.Label, values)
	case "nCr", "nPr":
		return chooseWork(node.Label, args, node.Value)
	case "hypot":
		return hypotWork(args, node.Value)
	case "mod":
		if len(args) == 2 {
			a, okA := exactOf(args[0])
			m, okM := exactOf(args[1])
			if okA && okM {
				return modWork(a, m)
			}
		}
	case "modinv":
		return modinvWork(args, node.Value)
	case "modpow":
		return modpowWork(args, node.Value)
	case "factor":
		return factorWork(args)
	case "divisors":
		return divisorPairs(args)
	case "totient":
		return totientWork(args, node.Value)
	case "isprime", "nextprime", "prevprime":
		return primeWork(node.Label, args, node.Value)
	case "fib":
		return fibWork(args, node.Value)
	case "digitsum":
		return digitWork(args, node.Value)
	case "tobase", "hex", "bin", "oct":
		return baseWork(node.Label, args, node.Value)
	case "percentile":
		return w.percentileWork(node, args)
	case "round", "floor", "ceil", "trunc":
		return placesWork(node.Label, args, node.Value)
	case "deg", "rad":
		return angleWork(node.Label, args, node.Value)
	case "sin", "cos", "tan", "sec", "csc", "cot":
		return degreeNote(args)
	case "log", "log2", "log10":
		return logWork(node.Label, args, node.Value)
	case "root", "cbrt":
		return rootWork(node.Label, args, node.Value)
	case "solve":
		return solveWork(node, w.result(node))
	}
	return nil
}

func aggregateWork(name string, values []*big.Rat) []string {
	total := new(big.Rat)
	product := big.NewRat(1, 1)
	for _, v := range values {
		total.Add(total, v)
		product.Mul(product, v)
	}
	switch name {
	case "sum":
		return []string{joinRats(values, " + ") + " = " + n(total)}
	case "product":
		return []string{joinRats(values, " × ") + " = " + n(product)}
	}
	count := big.NewRat(int64(len(values)), 1)
	mean := new(big.Rat).Quo(total, count)
	return []string{"(" + joinRats(values, " + ") + ") ÷ " + n(count) + " = " + n(total) + " ÷ " + n(count) + " = " + exactText(mean)}
}

func exactText(r *big.Rat) string {
	if terminates(r) {
		return decimalText(r)
	}
	return r.RatString()
}

func sortedRats(values []*big.Rat) []*big.Rat {
	out := append([]*big.Rat(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].Cmp(out[j]) < 0 })
	return out
}

func medianWork(values []*big.Rat) []string {
	sorted := sortedRats(values)
	lines := []string{"sorted: " + joinRats(sorted, ", ")}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return append(lines, "the middle value, number "+strconv.Itoa(mid+1)+" of "+strconv.Itoa(len(sorted))+", is "+n(sorted[mid]))
	}
	avg := new(big.Rat).Quo(new(big.Rat).Add(sorted[mid-1], sorted[mid]), big.NewRat(2, 1))
	return append(lines, "an even count has two middle values: ("+n(sorted[mid-1])+" + "+n(sorted[mid])+") ÷ 2 = "+exactText(avg))
}

func spreadWork(name string, values []*big.Rat, shown string) []string {
	count := int64(len(values))
	total := new(big.Rat)
	for _, v := range values {
		total.Add(total, v)
	}
	mean := new(big.Rat).Quo(total, big.NewRat(count, 1))
	lines := []string{"mean = " + n(total) + " ÷ " + strconv.FormatInt(count, 10) + " = " + exactText(mean)}
	squares := new(big.Rat)
	var terms []string
	for _, v := range values {
		d := new(big.Rat).Sub(v, mean)
		sq := new(big.Rat).Mul(d, d)
		squares.Add(squares, sq)
		terms = append(terms, "("+n(v)+" - "+exactText(mean)+")²")
	}
	lines = append(lines, "squared distances from the mean: "+strings.Join(terms, " + ")+" = "+exactText(squares))
	divisor := count - 1
	label := "n - 1 = "
	if strings.HasSuffix(name, "p") {
		divisor = count
		label = "n = "
	}
	variance := new(big.Rat).Quo(squares, big.NewRat(divisor, 1))
	lines = append(lines, "divide by "+label+strconv.FormatInt(divisor, 10)+": "+exactText(squares)+" ÷ "+strconv.FormatInt(divisor, 10)+" = "+exactText(variance))
	if strings.HasPrefix(name, "stdev") {
		lines = append(lines, "take the square root: √"+wrapFraction(exactText(variance))+" = "+shown)
	}
	return lines
}

func wrapFraction(text string) string {
	if strings.Contains(text, "/") {
		return "(" + text + ")"
	}
	return text
}

var approximateWorkAllowed = map[string]bool{
	"sin": true, "cos": true, "tan": true, "sec": true, "csc": true, "cot": true, "deg": true, "rad": true,
	"stdev": true, "stdevp": true, "hypot": true, "solve": true, "round": true, "floor": true, "ceil": true, "trunc": true,
}

func divisorWork(name string, values []*big.Rat) []string {
	whole := make([]*big.Int, len(values))
	for i, v := range values {
		if !v.IsInt() {
			return nil
		}
		whole[i] = new(big.Int).Abs(v.Num())
	}
	var lines []string
	acc := new(big.Int).Set(whole[0])
	for _, next := range whole[1:] {
		g := euclid(acc, next, &lines, len(whole) == 2 && name == "gcd")
		if name == "gcd" {
			if len(whole) > 2 {
				lines = append(lines, "gcd("+acc.String()+", "+next.String()+") = "+g.String())
			}
			acc = g
			continue
		}
		if g.Sign() == 0 {
			acc = big.NewInt(0)
			continue
		}
		l := new(big.Int).Quo(new(big.Int).Mul(acc, next), g)
		lines = append(lines, "lcm("+acc.String()+", "+next.String()+") = "+acc.String()+" × "+next.String()+" ÷ gcd = "+new(big.Int).Mul(acc, next).String()+" ÷ "+g.String()+" = "+l.String())
		acc = l
	}
	return lines
}

func euclid(a, b *big.Int, lines *[]string, show bool) *big.Int {
	x, y := new(big.Int).Set(a), new(big.Int).Set(b)
	if x.Cmp(y) < 0 {
		x, y = y, x
	}
	steps := 0
	for y.Sign() != 0 {
		q, r := new(big.Int).QuoRem(x, y, new(big.Int))
		if show && steps < maxWorkItems {
			*lines = append(*lines, x.String()+" = "+q.String()+" × "+y.String()+" + "+r.String())
		}
		x, y = y, r
		steps++
	}
	if show {
		*lines = append(*lines, "the last non-zero remainder is "+x.String())
	}
	return x
}

func smallWhole(text string) (int64, bool) {
	v, ok := wholeOf(text)
	if !ok || !v.IsInt64() {
		return 0, false
	}
	return v.Int64(), true
}

func factorialText(k int64) string {
	if k < 2 {
		return "1"
	}
	return new(big.Int).MulRange(2, k).String()
}

func chooseWork(name string, args []string, shown string) []string {
	if len(args) != 2 {
		return nil
	}
	total, okN := smallWhole(args[0])
	k, okK := smallWhole(args[1])
	if !okN || !okK || total > maxChooseShown || k < 0 || k > total {
		return nil
	}
	rest := strconv.FormatInt(total-k, 10)
	if name == "nCr" {
		return []string{args[0] + "! ÷ (" + args[1] + "! × " + rest + "!) = " + factorialText(total) + " ÷ (" + factorialText(k) + " × " + factorialText(total-k) + ") = " + shown}
	}
	return []string{args[0] + "! ÷ " + rest + "! = " + factorialText(total) + " ÷ " + factorialText(total-k) + " = " + shown}
}

func hypotWork(args []string, shown string) []string {
	if len(args) != 2 {
		return nil
	}
	a, okA := exactOf(args[0])
	b, okB := exactOf(args[1])
	if !okA || !okB {
		return nil
	}
	sa, sb := new(big.Rat).Mul(a, a), new(big.Rat).Mul(b, b)
	total := new(big.Rat).Add(sa, sb)
	return []string{"√(" + parenNegative(n(a)) + "² + " + parenNegative(n(b)) + "²) = √(" + n(sa) + " + " + n(sb) + ") = √" + n(total) + " = " + shown}
}

func modinvWork(args []string, shown string) []string {
	if len(args) != 2 {
		return nil
	}
	a, okA := wholeOf(args[0])
	m, okM := wholeOf(args[1])
	x, okX := wholeOf(shown)
	if !okA || !okM || !okX {
		return nil
	}
	product := new(big.Int).Mul(a, x)
	q := new(big.Int).Quo(product, m)
	return []string{"check: " + args[0] + " × " + shown + " = " + product.String() + " = " + q.String() + " × " + args[1] + " + 1"}
}

func modpowWork(args []string, shown string) []string {
	if len(args) != 3 {
		return nil
	}
	b, okB := wholeOf(args[0])
	e, okE := wholeOf(args[1])
	m, okM := wholeOf(args[2])
	if !okB || !okE || !okM || e.Sign() < 0 || m.Sign() <= 0 {
		return nil
	}
	if e.IsInt64() && e.Int64() <= 64 {
		full := new(big.Int).Exp(b, e, nil)
		if len(full.String()) <= maxModpowShown {
			q := new(big.Int).Quo(full, m)
			return []string{args[0] + "^" + args[1] + " = " + full.String(), full.String() + " = " + q.String() + " × " + args[2] + " + " + shown}
		}
	}
	return []string{"the full power is too large to write out, so it is reduced mod " + args[2] + " after every squaring and multiplication"}
}

func factorWork(args []string) []string {
	if len(args) != 1 {
		return nil
	}
	v, ok := wholeOf(args[0])
	if !ok || v.Sign() <= 0 || !v.IsInt64() || v.Int64() > maxFactorValue || v.Int64() < 2 {
		return nil
	}
	rest := v.Int64()
	var lines []string
	for d := int64(2); d*d <= rest; d++ {
		for rest%d == 0 {
			if len(lines) < maxFactorLines {
				lines = append(lines, strconv.FormatInt(rest, 10)+" ÷ "+strconv.FormatInt(d, 10)+" = "+strconv.FormatInt(rest/d, 10))
			}
			rest /= d
		}
	}
	if rest > 1 {
		lines = append(lines, strconv.FormatInt(rest, 10)+" has no divisor up to its square root, so it is prime")
	}
	return lines
}

func divisorPairs(args []string) []string {
	if len(args) != 1 {
		return nil
	}
	v, ok := smallWhole(args[0])
	if !ok || v <= 0 || v > maxFactorValue {
		return nil
	}
	var pairs []string
	for d := int64(1); d*d <= v; d++ {
		if v%d == 0 {
			pairs = append(pairs, strconv.FormatInt(d, 10)+" × "+strconv.FormatInt(v/d, 10))
			if len(pairs) > maxWorkItems {
				return nil
			}
		}
	}
	return []string{"divisors come in pairs that multiply to " + args[0] + ": " + strings.Join(pairs, ", ")}
}

func primeFactors(v int64) []int64 {
	var out []int64
	for d := int64(2); d*d <= v; d++ {
		if v%d == 0 {
			out = append(out, d)
			for v%d == 0 {
				v /= d
			}
		}
	}
	if v > 1 {
		out = append(out, v)
	}
	return out
}

func totientWork(args []string, shown string) []string {
	if len(args) != 1 {
		return nil
	}
	v, ok := smallWhole(args[0])
	if !ok || v <= 1 || v > maxFactorValue {
		return nil
	}
	ps := primeFactors(v)
	names := make([]string, len(ps))
	terms := make([]string, len(ps))
	for i, p := range ps {
		names[i] = strconv.FormatInt(p, 10)
		terms[i] = "(1 - 1/" + names[i] + ")"
	}
	return []string{"the prime factors of " + args[0] + " are " + spokenList(names), args[0] + " × " + strings.Join(terms, " × ") + " = " + shown}
}

func primeWork(name string, args []string, shown string) []string {
	if len(args) != 1 {
		return nil
	}
	target := args[0]
	if name != "isprime" {
		target = shown
	}
	v, ok := smallWhole(target)
	if !ok || v < 2 || v > maxSearchedFrom {
		return nil
	}
	limit := int64(math.Sqrt(float64(v)))
	for d := int64(2); d <= limit; d++ {
		if v%d == 0 {
			return []string{target + " = " + strconv.FormatInt(d, 10) + " × " + strconv.FormatInt(v/d, 10) + ", so it is not prime"}
		}
	}
	if limit < 2 {
		return []string{target + " has no divisor other than 1 and itself"}
	}
	return []string{"no whole number from 2 to " + strconv.FormatInt(limit, 10) + " (the square root of " + target + ", rounded down) divides " + target + ", so it is prime"}
}

func fibWork(args []string, shown string) []string {
	if len(args) != 1 {
		return nil
	}
	k, ok := smallWhole(args[0])
	if !ok || k < 2 {
		return nil
	}
	if k <= maxFibListed {
		seq := []string{"0", "1"}
		a, b := big.NewInt(0), big.NewInt(1)
		for i := int64(2); i <= k; i++ {
			a, b = b, new(big.Int).Add(a, b)
			seq = append(seq, b.String())
		}
		return []string{strings.Join(seq, ", ")}
	}
	return []string{"fib(" + strconv.FormatInt(k, 10) + ") = fib(" + strconv.FormatInt(k-1, 10) + ") + fib(" + strconv.FormatInt(k-2, 10) + ") = " + shown + ", computed by doubling steps rather than one at a time"}
}

func digitWork(args []string, shown string) []string {
	if len(args) != 1 {
		return nil
	}
	v, ok := wholeOf(args[0])
	if !ok {
		return nil
	}
	digits := strings.Split(new(big.Int).Abs(v).String(), "")
	if len(digits) > maxWorkItems*2 {
		return nil
	}
	return []string{strings.Join(digits, " + ") + " = " + shown}
}

var baseOf = map[string]int64{"hex": 16, "bin": 2, "oct": 8}

func baseWork(name string, args []string, shown string) []string {
	if len(args) == 0 {
		return nil
	}
	v, ok := wholeOf(args[0])
	if !ok {
		return nil
	}
	base, known := baseOf[name]
	if name == "tobase" {
		if len(args) != 2 {
			return nil
		}
		b, okB := smallWhole(args[1])
		if !okB {
			return nil
		}
		base = b
		known = true
	}
	if !known || base < 2 {
		return nil
	}
	rest := new(big.Int).Abs(v)
	if rest.Sign() == 0 {
		return nil
	}
	b := big.NewInt(base)
	var lines []string
	for rest.Sign() > 0 {
		q, r := new(big.Int).QuoRem(rest, b, new(big.Int))
		digit := r.Text(int(base))
		line := rest.String() + " ÷ " + strconv.FormatInt(base, 10) + " = " + q.String() + " remainder " + r.String()
		if r.String() != digit {
			line += " (" + digit + ")"
		}
		lines = append(lines, line)
		if len(lines) > maxBaseLines {
			return nil
		}
		rest = q
	}
	answer := strings.TrimPrefix(shown, "-")
	for _, prefix := range []string{"0x", "0b", "0o"} {
		answer = strings.TrimPrefix(answer, prefix)
	}
	return append(lines, "read the remainders from last to first: "+answer)
}

func (w *walker) percentileWork(node prov.NodeOut, args []string) []string {
	kids := w.kids(node)
	if len(kids) != 2 || len(args) != 2 {
		return nil
	}
	texts, ok := w.items(kids[0])
	if !ok {
		return nil
	}
	values, ok := exactAll(texts)
	if !ok || len(values) < 2 {
		return nil
	}
	p, ok := exactOf(args[1])
	if !ok {
		return nil
	}
	sorted := sortedRats(values)
	rank := new(big.Rat).Mul(new(big.Rat).Quo(p, big.NewRat(100, 1)), big.NewRat(int64(len(sorted)-1), 1))
	below := new(big.Int).Quo(rank.Num(), rank.Denom())
	at := int(below.Int64())
	lines := []string{"sorted: " + joinRats(sorted, ", "), "position = " + n(p) + "% × (" + strconv.Itoa(len(sorted)) + " - 1) = " + exactText(rank) + ", counting from 0"}
	if at >= len(sorted)-1 || rank.IsInt() {
		return append(lines, "that position holds "+n(sorted[min(at, len(sorted)-1)]))
	}
	weight := new(big.Rat).Sub(rank, new(big.Rat).SetInt(below))
	lo, hi := sorted[at], sorted[at+1]
	value := new(big.Rat).Add(lo, new(big.Rat).Mul(weight, new(big.Rat).Sub(hi, lo)))
	return append(lines, "between "+n(lo)+" and "+n(hi)+": "+n(lo)+" + "+exactText(weight)+" × ("+n(hi)+" - "+n(lo)+") = "+exactText(value))
}

func placesWork(name string, args []string, shown string) []string {
	if len(args) != 2 {
		return nil
	}
	k, ok := smallWhole(args[1])
	if !ok || k <= 0 || k > 12 {
		return nil
	}
	scale := "1" + strings.Repeat("0", int(k))
	value := strings.TrimSuffix(args[0], "…")
	return []string{"shift the decimal point " + args[1] + " places: " + value + " × " + scale, "apply " + name + " to the whole-number part, then shift back by dividing by " + scale + ": " + shown}
}

func angleWork(name string, args []string, shown string) []string {
	if len(args) != 1 {
		return nil
	}
	if name == "deg" {
		return []string{args[0] + " × 180 ÷ π = " + shown}
	}
	return []string{args[0] + " × π ÷ 180 = " + shown}
}

func degreeNote(args []string) []string {
	if len(args) != 1 {
		return nil
	}
	x, err := strconv.ParseFloat(strings.TrimSuffix(args[0], "…"), 64)
	if err != nil {
		return nil
	}
	degrees := x * 180 / math.Pi
	rounded := math.Round(degrees*1e6) / 1e6
	if math.Abs(degrees-math.Round(degrees)) < 1e-7 {
		return []string{args[0] + " radians is " + strconv.FormatFloat(math.Round(degrees), 'f', -1, 64) + "°"}
	}
	return []string{args[0] + " radians is about " + strconv.FormatFloat(rounded, 'g', 6, 64) + "°"}
}

func logWork(name string, args []string, shown string) []string {
	k, ok := smallWhole(shown)
	if !ok || len(args) == 0 {
		return nil
	}
	base := map[string]string{"log2": "2", "log10": "10"}[name]
	if name == "log" {
		if len(args) != 2 {
			return nil
		}
		base = args[1]
	}
	if base == "" {
		return nil
	}
	return []string{"check: " + base + "^" + strconv.FormatInt(k, 10) + " = " + args[0]}
}

func rootWork(name string, args []string, shown string) []string {
	if _, ok := wholeOf(shown); !ok || len(args) == 0 {
		return nil
	}
	degree := "3"
	if name == "root" {
		if len(args) != 2 {
			return nil
		}
		degree = args[1]
	}
	k, ok := smallWhole(degree)
	if !ok || k < 2 || k > maxPowerShown {
		return nil
	}
	factors := make([]string, k)
	for i := range factors {
		factors[i] = parenNegative(shown)
	}
	return []string{"check: " + strings.Join(factors, " × ") + " = " + args[0]}
}

func solveWork(node prov.NodeOut, answer string) []string {
	terms := termsByName(node.Terms)
	v := variableOf(terms)
	a, b, c, disc := terms["a"], terms["b"], terms["c"], terms[termDiscriminant]
	switch node.Detail {
	case quadraticRule:
		if a == "" || b == "" || c == "" {
			return nil
		}
		return []string{
			"write it as a·" + v + "² + b·" + v + " + c = 0 with a = " + a + ", b = " + b + ", c = " + c,
			"discriminant b² - 4ac = " + factor(b) + "² - 4·" + factor(a) + "·" + factor(c) + " = " + disc,
			v + " = (-b ± √discriminant) ÷ 2a = (" + negated(b) + " ± √" + disc + ") ÷ " + doubled(a) + ", giving " + strings.ReplaceAll(answer, ",  ", " or "),
		}
	case linearRule:
		if a == "" || b == "" {
			return nil
		}
		return []string{"write it as a·" + v + " + b = 0 with a = " + a + ", b = " + b, v + " = -b ÷ a = " + negated(b) + " ÷ " + factor(a) + ", giving " + answer}
	}
	return nil
}

func negated(text string) string {
	if strings.HasPrefix(text, "-") {
		return strings.TrimPrefix(text, "-")
	}
	return "-" + text
}

func doubled(text string) string {
	r, ok := new(big.Rat).SetString(text)
	if !ok {
		return "2·" + factor(text)
	}
	return exactText(r.Mul(r, big.NewRat(2, 1)))
}

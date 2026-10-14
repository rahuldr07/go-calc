package steps

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/prov"
)

const (
	kindVariable  = "variable"
	kindOperation = "operation"
	maxSpelledOut = 10
	imaginaryUnit = "i"
)

var oneArgument = map[string]string{
	"sqrt":      "Take the square root of %s",
	"cbrt":      "Take the cube root of %s",
	"abs":       "Take the absolute value of %s, its distance from zero",
	"exp":       "Raise e to the power %s",
	"exp2":      "Raise 2 to the power %s",
	"ln":        "Take the natural logarithm of %s, the power e must be raised to",
	"log2":      "Take the base-2 logarithm of %s, the power 2 must be raised to",
	"log10":     "Take the base-10 logarithm of %s, the power 10 must be raised to",
	"sin":       "Take the sine of %s",
	"cos":       "Take the cosine of %s",
	"tan":       "Take the tangent of %s",
	"asin":      "Take the arcsine of %s, the angle whose sine it is",
	"acos":      "Take the arccosine of %s, the angle whose cosine it is",
	"atan":      "Take the arctangent of %s, the angle whose tangent it is",
	"sinh":      "Take the hyperbolic sine of %s",
	"cosh":      "Take the hyperbolic cosine of %s",
	"tanh":      "Take the hyperbolic tangent of %s",
	"asinh":     "Take the inverse hyperbolic sine of %s",
	"acosh":     "Take the inverse hyperbolic cosine of %s",
	"atanh":     "Take the inverse hyperbolic tangent of %s",
	"sec":       "Take the secant of %s, which is 1 ÷ its cosine",
	"csc":       "Take the cosecant of %s, which is 1 ÷ its sine",
	"cot":       "Take the cotangent of %s, which is 1 ÷ its tangent",
	"deg":       "Turn %s radians into degrees by multiplying by 180/π",
	"rad":       "Turn %s degrees into radians by multiplying by π/180",
	"frac":      "Keep only the fractional part of %s",
	"sign":      "Find the sign of %s: -1 if negative, 0 if zero, 1 if positive",
	"gamma":     "Evaluate the gamma function at %s, which extends the factorial: gamma(n) = (n - 1)!",
	"erf":       "Evaluate the error function at %s",
	"erfc":      "Evaluate the complementary error function at %s, which is 1 - erf",
	"fib":       "Find Fibonacci number %s, where each number is the sum of the two before it",
	"factor":    "Split %s into prime factors",
	"isprime":   "Test whether %s is prime: divisible only by 1 and itself",
	"divisors":  "List every whole number that divides %s exactly",
	"totient":   "Count the whole numbers from 1 to %s that share no factor with it",
	"nextprime": "Find the first prime above %s",
	"prevprime": "Find the last prime below %s",
	"hex":       "Write %s in base 16",
	"bin":       "Write %s in base 2",
	"oct":       "Write %s in base 8",
	"sum":       "Add up %s",
	"product":   "Multiply together %s",
	"mean":      "Average %s: add the values and divide by how many there are",
	"median":    "Find the middle value of %s once it is sorted",
	"min":       "Pick the smallest of %s",
	"max":       "Pick the largest of %s",
	"count":     "Count the values in %s",
	"len":       "Count the items in %s",
	"stdev":     "Take the sample standard deviation of %s, dividing by n - 1",
	"stdevp":    "Take the population standard deviation of %s, dividing by n",
	"variance":  "Take the sample variance of %s, dividing by n - 1",
	"variancep": "Take the population variance of %s, dividing by n",
	"sort":      "Sort %s from smallest to largest",
	"reverse":   "Reverse the order of %s",
	"unique":    "Remove repeated values from %s, keeping the first of each",
	"mode":      "Find the most common value in %s",
	"cumsum":    "Keep a running total through %s",
	"diff":      "Take the difference between each neighbouring pair in %s",
}

var rounding = map[string]string{
	"floor": "Round %s down",
	"ceil":  "Round %s up",
	"round": "Round %s to the nearest",
	"trunc": "Cut %s toward zero",
}

var operatorVerbs = map[string]string{
	"<": "is below", ">": "is above", "<=": "is at most", ">=": "is at least", "==": "equals", "!=": "differs from",
}

var solveMethods = map[string]string{
	"exact linear solution":                                    "by rearranging the linear equation exactly",
	"quadratic formula":                                        "with the quadratic formula",
	"bisection on proven sign-change brackets":                 "by bisection: halving each interval where the equation changes sign until the root is pinned down",
	"interval evaluation over each stretch of the search grid": "by interval evaluation over each stretch where the equation is zero",
}

func (w *walker) kids(n prov.NodeOut) []prov.NodeOut {
	out := make([]prov.NodeOut, 0, len(n.Children))
	for _, id := range n.Children {
		if child, ok := w.nodes[id]; ok {
			out = append(out, child)
		}
	}
	return out
}

func (w *walker) values(n prov.NodeOut) []string {
	kids := w.kids(n)
	out := make([]string, 0, len(kids))
	for _, child := range kids {
		out = append(out, w.spoken(child))
	}
	return out
}

func (w *walker) spoken(child prov.NodeOut) string {
	switch {
	case isBareUnit(child):
		if definition := definitionOf(child.Detail); !trivialDefinition(definition) && !strings.Contains(definition, " + ") {
			return child.Label + " (" + definition + ")"
		}
		return child.Label
	case strings.HasPrefix(child.Value, "list of "):
		return w.expression(child)
	}
	return child.Value
}

func isBareUnit(n prov.NodeOut) bool { return n.Kind == kindUnit && len(n.Children) == 0 }

func definitionOf(detail string) string {
	text, _, _ := strings.Cut(detail, " (")
	return text
}

func trivialDefinition(definition string) bool {
	left, right, ok := strings.Cut(definition, " = ")
	return ok && left == right
}

func affineReading(symbol, definition string) (string, bool) {
	_, right, ok := strings.Cut(definition, " = ")
	if !ok {
		return "", false
	}
	scaled, offset, ok := strings.Cut(right, " + ")
	if !ok {
		return "", false
	}
	factor, base, ok := strings.Cut(scaled, " ")
	if !ok {
		return "", false
	}
	offset = strings.TrimSuffix(offset, " "+base)
	text := base + " = " + symbol
	if factor != "1" {
		text += " × " + factor
	}
	return text + " + " + offset, true
}

func siReading(value, detail string) (string, bool) {
	amount, symbol, ok := strings.Cut(value, " ")
	if !ok {
		return "", false
	}
	left, right, ok := strings.Cut(definitionOf(detail), " = ")
	if !ok || left != "1 "+symbol || strings.Contains(right, " + ") {
		return "", false
	}
	factorText, siUnit, ok := strings.Cut(right, " ")
	if !ok {
		return "", false
	}
	a, ok := new(big.Rat).SetString(amount)
	if !ok {
		return "", false
	}
	f, ok := new(big.Rat).SetString(factorText)
	if !ok {
		return "", false
	}
	return decimalText(new(big.Rat).Mul(a, f)) + " " + siUnit, true
}

func decimalText(r *big.Rat) string {
	d := new(big.Int).Set(r.Denom())
	for _, p := range []int64{2, 5} {
		for new(big.Int).Mod(d, big.NewInt(p)).Sign() == 0 {
			d.Quo(d, big.NewInt(p))
		}
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		return r.RatString()
	}
	text := strings.TrimRight(r.FloatString(40), "0")
	return strings.TrimSuffix(text, ".")
}

func (w *walker) explain(n prov.NodeOut) string {
	args := w.values(n)
	switch n.Kind {
	case kindConstant:
		if n.Label == imaginaryUnit {
			return "Use i, the imaginary unit, where i × i = -1"
		}
		return "Use the constant " + n.Label + " = " + n.Value
	case kindVariable:
		return "Use " + n.Label + ", defined earlier as " + n.Value
	case kindUnit:
		return unitExplanation(n, args)
	case kindOperation:
		return operationExplanation(n, args, w.kids(n))
	case kindFunction:
		return functionExplanation(n, args)
	}
	return ""
}

func unitExplanation(n prov.NodeOut, args []string) string {
	definition := definitionOf(n.Detail)
	for _, prefix := range []string{"in ", conversionPrefix} {
		if target, ok := strings.CutPrefix(n.Label, prefix); ok {
			return conversionExplanation(target, definition)
		}
	}
	if len(args) != 1 {
		return "The unit " + n.Label + " stands for " + definition
	}
	quantity := args[0] + " " + n.Label
	if scale, ok := affineReading(n.Label, definition); ok {
		return "Read " + quantity + " as a temperature on the " + n.Label + " scale, where " + scale
	}
	if trivialDefinition(definition) {
		return "Read " + quantity + ", already in SI base units"
	}
	if !strings.Contains(n.Value, " ") {
		return "Read " + quantity + ": since " + definition + ", that is " + n.Value
	}
	if si, ok := siReading(n.Value, n.Detail); ok {
		return "Read " + quantity + ". In SI base units that is " + si + ", because " + definition
	}
	return "Read " + quantity + ", where " + definition
}

func conversionExplanation(target, definition string) string {
	if scale, ok := affineReading(target, definition); ok {
		return "Convert the result to " + target + ", using " + scale
	}
	if trivialDefinition(definition) {
		return "Express the result in " + target
	}
	return "Convert the result to " + target + ", using " + definition
}

func operationExplanation(n prov.NodeOut, args []string, kids []prov.NodeOut) string {
	op := n.Label
	if len(args) == 1 {
		a := args[0]
		switch op {
		case "-":
			return "Negate " + a
		case "%":
			return a + "% means " + a + " ÷ 100"
		case "!":
			return factorialExplanation(a, n.Value)
		}
		return "Apply " + op + " to " + a
	}
	if len(args) != 2 || len(kids) != 2 {
		return ""
	}
	a, b := args[0], args[1]
	if percent := kids[1]; percent.Kind == kindOperation && percent.Label == "%" && (op == "+" || op == "-") {
		share := percentOf(percent.Value)
		if op == "+" {
			return "Add " + share + "% of " + a + " to " + a + ", which means " + a + " × (1 + " + b + ")"
		}
		return "Take " + share + "% off " + a + ", which means " + a + " × (1 - " + b + ")"
	}
	switch op {
	case "+":
		return "Add " + a + " and " + b
	case "-":
		return "Subtract " + b + " from " + a
	case "*", "·", "×":
		return "Multiply " + a + " by " + b
	case "/", "÷":
		if isBareUnit(kids[1]) {
			return "Express " + a + " per " + b
		}
		return "Divide " + a + " by " + b
	case "^", "**":
		switch b {
		case "2":
			return "Square " + a
		case "3":
			return "Cube " + a
		case "0.5", "1/2":
			return "Raise " + a + " to the power 1/2, which is its square root"
		}
		return "Raise " + a + " to the power " + b
	case "of":
		return "Take " + a + " of " + b + ", which means " + a + " × " + b
	case "mod":
		return "Find the remainder when " + a + " is divided by " + b
	}
	if verb, ok := operatorVerbs[op]; ok {
		return "Check whether " + a + " " + verb + " " + b
	}
	return "Combine " + a + " and " + b + " with " + op
}

func percentOf(value string) string {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return value
	}
	return decimalText(r.Mul(r, big.NewRat(100, 1)))
}

func factorialExplanation(a, value string) string {
	n, err := strconv.Atoi(a)
	if err != nil || n < 0 {
		return "Take the factorial of " + a
	}
	if n <= 1 {
		return a + "! is 1 by definition"
	}
	if n > maxSpelledOut {
		return a + "! multiplies every whole number from 1 to " + a
	}
	parts := make([]string, 0, n)
	for k := n; k >= 1; k-- {
		parts = append(parts, strconv.Itoa(k))
	}
	return a + "! = " + strings.Join(parts, " × ") + " = " + value
}

func ordinal(text string) string {
	n, err := strconv.Atoi(text)
	if err != nil {
		return text + "th"
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return text + suffix
}

func places(text string) string {
	if text == "1" {
		return "1 decimal place"
	}
	if digits, negative := strings.CutPrefix(text, "-"); negative {
		n, err := strconv.Atoi(digits)
		if err == nil && n > 0 {
			return "the nearest 1" + strings.Repeat("0", n)
		}
	}
	return text + " decimal places"
}

func spokenList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

func functionExplanation(n prov.NodeOut, args []string) string {
	subject := spokenList(args)
	switch n.Label {
	case "range":
		if len(args) >= 2 {
			text := "List the whole numbers from " + args[0] + " to " + args[1]
			if len(args) == 3 {
				text += " in steps of " + args[2]
			}
			return text
		}
	case "list":
		return "Collect " + strconv.Itoa(len(args)) + " values into a list"
	case "solve":
		return solveExplanation(n)
	case "deriv":
		return derivExplanation(n.Detail)
	case "index":
		if len(args) == 2 {
			return "Pick item " + args[1] + " of the list, counting from 0, so -1 is the last"
		}
	case "root":
		if len(args) == 2 {
			return "Take the " + ordinal(args[1]) + " root of " + args[0] + ": the number that, raised to the power " + args[1] + ", gives " + args[0]
		}
	case "atan2":
		if len(args) == 2 {
			return "Find the angle of the point (x = " + args[1] + ", y = " + args[0] + ") measured from the positive x-axis"
		}
	case "log":
		switch len(args) {
		case 1:
			return "Take the natural logarithm of " + args[0] + ", the power e must be raised to"
		case 2:
			return "Take the base-" + args[1] + " logarithm of " + args[0] + ", the power " + args[1] + " must be raised to"
		}
	case "floor", "ceil", "round", "trunc":
		if len(args) >= 1 {
			text := strings.Replace(rounding[n.Label], "%s", args[0], 1)
			if len(args) == 2 {
				if n.Label == "round" {
					return "Round " + args[0] + " to " + places(args[1]) + ", halves going up"
				}
				return text + " to " + places(args[1])
			}
			if n.Label == "round" {
				return text + " whole number, halves going up"
			}
			return text + " to a whole number"
		}
	case "sqrt":
		if len(args) == 1 && strings.HasPrefix(args[0], "-") {
			return "Take the square root of " + args[0] + ": a negative number has no real square root, so the answer is imaginary, a multiple of i"
		}
		if len(args) == 1 && isWholeText(n.Value) && isWholeText(args[0]) {
			return "Take the square root of " + args[0] + ": it is " + n.Value + ", because " + n.Value + " × " + n.Value + " = " + args[0]
		}
	case "clamp":
		if len(args) == 3 {
			return "Keep " + args[0] + " between " + args[1] + " and " + args[2] + ", moving it to the nearer limit if it is outside"
		}
	case "hypot":
		if len(args) == 2 {
			return "Find the hypotenuse of a right triangle with sides " + args[0] + " and " + args[1] + ": the square root of " + args[0] + "² + " + args[1] + "²"
		}
	case "gcd":
		return "Find the greatest common divisor of " + subject + ": the largest whole number dividing them all"
	case "lcm":
		return "Find the least common multiple of " + subject + ": the smallest whole number they all divide"
	case "nCr":
		if len(args) == 2 {
			return "Count the ways to choose " + args[1] + " of " + args[0] + " items when order does not matter: " + args[0] + "! ÷ (" + args[1] + "! × (" + args[0] + " - " + args[1] + ")!)"
		}
	case "nPr":
		if len(args) == 2 {
			return "Count the ways to arrange " + args[1] + " of " + args[0] + " items when order matters: " + args[0] + "! ÷ (" + args[0] + " - " + args[1] + ")!"
		}
	case "mod":
		if len(args) == 2 {
			return "Find " + args[0] + " mod " + args[1] + ": the remainder after dividing, never negative for a positive modulus"
		}
	case "modpow":
		if len(args) == 3 {
			return "Raise " + args[0] + " to the power " + args[1] + " and keep only the remainder mod " + args[2] + ", reducing mod " + args[2] + " at every step so the number never grows"
		}
	case "modinv":
		if len(args) == 2 {
			return "Find the number that, multiplied by " + args[0] + ", leaves remainder 1 mod " + args[1]
		}
	case "digitsum":
		if len(args) == 2 {
			return "Add up the digits of " + args[0] + " written in base " + args[1]
		}
		return "Add up the decimal digits of " + subject
	case "tobase":
		if len(args) == 2 {
			return "Write " + args[0] + " in base " + args[1]
		}
	case "primes":
		switch len(args) {
		case 1:
			if strings.HasPrefix(args[0], "[") || strings.Contains(args[0], "..") {
				return "Keep only the primes in " + args[0]
			}
			return "List every prime up to " + args[0]
		case 2:
			return "List every prime from " + args[0] + " to " + args[1]
		}
	case "percentile":
		if len(args) == 2 {
			return "Find the " + ordinal(args[1]) + " percentile of " + args[0] + ", interpolating between the two nearest values"
		}
	case "expand":
		if len(args) == 2 {
			return "Expand the uncertainty of " + args[0] + " by the coverage factor k = " + args[1]
		}
	}
	if phrase, ok := oneArgument[n.Label]; ok {
		return strings.Replace(phrase, "%s", subject, 1)
	}
	if subject == "" {
		return "Apply " + n.Label
	}
	return "Apply " + n.Label + " to " + subject
}

func solveExplanation(n prov.NodeOut) string {
	variable := ""
	for _, t := range n.Terms {
		if t.Name == termVariable {
			variable = t.Value
		}
	}
	text := "Solve the equation"
	if variable != "" {
		text += " for " + variable
	}
	if method, ok := solveMethods[n.Detail]; ok {
		text += " " + method
	}
	return text
}

func derivExplanation(detail string) string {
	head, rest, ok := strings.Cut(detail, " ")
	variable, isDeriv := strings.CutPrefix(head, "d/d")
	if !ok || !isDeriv {
		return "Differentiate"
	}
	formula, point, atPoint := strings.Cut(rest, ", at ")
	original, derived, hasResult := strings.Cut(formula, " = ")
	if !hasResult {
		return "Differentiate with respect to " + variable
	}
	text := "Differentiate " + original + " with respect to " + variable + ": the derivative is " + derived
	if atPoint {
		text += ", then evaluate it at " + point
	}
	return text
}

func isWholeText(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func summaryOf(line Line, w *walker) []string {
	var out []string
	switch line.Status {
	case doc.StatusExact:
		out = append(out, "Every step used exact arithmetic on whole numbers and fractions, so nothing was rounded and the answer is exact.")
	case doc.StatusProven:
		out = append(out, "Part of this needs irrational numbers such as π or a square root. go-calc works those as intervals that are guaranteed to contain the true value, so every digit shown is certain, not estimated.")
	case doc.StatusEstimate:
		out = append(out, "The answer depends on measured values. Each input's ± is carried through to first order, the standard GUM method, so the ± on the answer is a standard uncertainty, not a guaranteed bound.")
	}
	if w != nil && w.usesUnits() {
		if w.converts() {
			out = append(out, "Quantities with units are worked in SI base units (metres, kilograms, seconds), which is why some steps show m or m^3; the conversion at the end turns the answer back into the unit you asked for.")
		} else {
			out = append(out, "Quantities with units are worked in SI base units (metres, kilograms, seconds), which is why some steps show m or m^3.")
		}
	}
	return out
}

func (w *walker) usesUnits() bool {
	for _, n := range w.nodes {
		if n.Kind == kindUnit {
			return true
		}
	}
	return false
}

func (w *walker) converts() bool {
	for _, n := range w.nodes {
		if n.Kind == kindUnit && (strings.HasPrefix(n.Label, "in ") || strings.HasPrefix(n.Label, conversionPrefix)) {
			return true
		}
	}
	return false
}

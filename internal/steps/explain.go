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
)

var functionPhrases = map[string]string{
	"sqrt":      "Take the square root of %s",
	"cbrt":      "Take the cube root of %s",
	"abs":       "Take the absolute value of %s",
	"exp":       "Raise e to the power %s",
	"exp2":      "Raise 2 to the power %s",
	"ln":        "Take the natural logarithm of %s",
	"log":       "Take the logarithm of %s",
	"log2":      "Take the base-2 logarithm of %s",
	"log10":     "Take the base-10 logarithm of %s",
	"sin":       "Take the sine of %s",
	"cos":       "Take the cosine of %s",
	"tan":       "Take the tangent of %s",
	"asin":      "Take the arcsine of %s",
	"acos":      "Take the arccosine of %s",
	"atan":      "Take the arctangent of %s",
	"sinh":      "Take the hyperbolic sine of %s",
	"cosh":      "Take the hyperbolic cosine of %s",
	"tanh":      "Take the hyperbolic tangent of %s",
	"floor":     "Round %s down to a whole number",
	"ceil":      "Round %s up to a whole number",
	"round":     "Round %s to the nearest whole number",
	"trunc":     "Drop the fractional part of %s",
	"gamma":     "Evaluate the gamma function at %s",
	"erf":       "Evaluate the error function at %s",
	"fib":       "Find Fibonacci number %s",
	"factor":    "Split %s into prime factors",
	"isprime":   "Test whether %s is prime",
	"sum":       "Add up %s",
	"product":   "Multiply together %s",
	"mean":      "Average %s: add the values and divide by how many there are",
	"median":    "Find the middle value of %s once sorted",
	"min":       "Pick the smallest of %s",
	"max":       "Pick the largest of %s",
	"count":     "Count the values in %s",
	"stdev":     "Take the sample standard deviation of %s, dividing by n - 1",
	"stdevp":    "Take the population standard deviation of %s, dividing by n",
	"variance":  "Take the sample variance of %s, dividing by n - 1",
	"variancep": "Take the population variance of %s, dividing by n",
	"sort":      "Sort %s from smallest to largest",
	"deriv":     "Differentiate %s",
	"expand":    "Widen the uncertainty of %s by the coverage factor",
}

var operatorVerbs = map[string]string{
	"<": "is below", ">": "is above", "<=": "is at most", ">=": "is at least", "==": "equals", "!=": "differs from",
}

func (w *walker) values(n prov.NodeOut) []string {
	out := make([]string, 0, len(n.Children))
	for _, id := range n.Children {
		child, ok := w.nodes[id]
		if !ok {
			continue
		}
		switch {
		case child.Kind == kindUnit && len(child.Children) == 0:
			out = append(out, child.Label+" ("+definitionOf(child.Detail)+")")
		case strings.HasPrefix(child.Value, "list of "):
			out = append(out, w.expression(child))
		default:
			out = append(out, child.Value)
		}
	}
	return out
}

func definitionOf(detail string) string {
	text, _, _ := strings.Cut(detail, " (")
	return text
}

func siReading(value, detail string) (string, bool) {
	amount, symbol, ok := strings.Cut(value, " ")
	if !ok {
		return "", false
	}
	left, right, ok := strings.Cut(definitionOf(detail), " = ")
	if !ok || left != "1 "+symbol {
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
		return "Use the constant " + n.Label + " = " + n.Value
	case kindVariable:
		return "Use " + n.Label + ", defined earlier as " + n.Value
	case kindUnit:
		return unitExplanation(n, args)
	case kindOperation:
		return operationExplanation(n.Label, args, n.Value)
	case kindFunction:
		return functionExplanation(n, args)
	}
	return ""
}

func unitExplanation(n prov.NodeOut, args []string) string {
	for _, prefix := range []string{"in ", conversionPrefix} {
		if target, ok := strings.CutPrefix(n.Label, prefix); ok {
			return conversionExplanation(target, definitionOf(n.Detail))
		}
	}
	if len(args) == 1 {
		text := "Read " + n.Value
		if si, ok := siReading(n.Value, n.Detail); ok && si != n.Value {
			return text + ". In SI base units that is " + si + ", because " + definitionOf(n.Detail)
		}
		return text + ", where " + definitionOf(n.Detail)
	}
	return "The unit " + n.Label + " stands for " + n.Detail
}

func conversionExplanation(target, definition string) string {
	left, right, ok := strings.Cut(definition, " = ")
	if !ok || left == right {
		return "Express the result in " + target
	}
	return "Convert the result to " + target + ", using " + definition
}

func operationExplanation(op string, args []string, value string) string {
	if len(args) == 1 {
		a := args[0]
		switch op {
		case "-":
			return "Negate " + a
		case "%":
			return a + "% means " + a + " ÷ 100"
		case "!":
			return factorialExplanation(a, value)
		}
		return "Apply " + op + " to " + a
	}
	if len(args) != 2 {
		return ""
	}
	a, b := args[0], args[1]
	switch op {
	case "+":
		return "Add " + a + " and " + b
	case "-":
		return "Subtract " + b + " from " + a
	case "*", "·", "×":
		return "Multiply " + a + " by " + b
	case "/", "÷":
		if strings.Contains(b, " (1 ") {
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

func functionExplanation(n prov.NodeOut, args []string) string {
	subject := strings.Join(args, ", ")
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
		return "Solve the equation"
	case "index":
		if len(args) == 2 {
			return "Pick item " + args[1] + " of the list, counting from 0"
		}
	case "root":
		if len(args) == 2 {
			return "Take root " + args[1] + " of " + args[0]
		}
	case "atan2":
		if len(args) == 2 {
			return "Find the angle of the point (" + args[1] + ", " + args[0] + ")"
		}
	}
	if n.Label == "sqrt" && len(args) == 1 && isWholeText(n.Value) && isWholeText(args[0]) {
		return "Take the square root of " + args[0] + ": it is " + n.Value + ", because " + n.Value + " × " + n.Value + " = " + args[0]
	}
	if phrase, ok := functionPhrases[n.Label]; ok {
		return strings.Replace(phrase, "%s", subject, 1)
	}
	if subject == "" {
		return "Apply " + n.Label
	}
	return "Apply " + n.Label + " to " + subject
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

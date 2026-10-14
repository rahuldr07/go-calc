package eval

type FunctionDoc struct {
	Signature string `json:"signature"`
	Summary   string `json:"summary"`
	Category  string `json:"category"`
	Example   string `json:"example"`
}

const (
	catRoots    = "Powers and roots"
	catLogs     = "Exponentials and logarithms"
	catTrig     = "Trigonometry"
	catHyper    = "Hyperbolic"
	catRounding = "Rounding and sign"
	catWhole    = "Whole numbers"
	catStats    = "Statistics"
	catLists    = "Lists"
	catCalculus = "Algebra and calculus"
	catSpecial  = "Special functions"
	catMeasure  = "Measurement"
)

var CategoryOrder = []string{catRoots, catLogs, catTrig, catHyper, catRounding, catWhole, catStats, catLists, catCalculus, catSpecial, catMeasure}

var functionDocs = map[string]FunctionDoc{
	"sqrt":  {"sqrt(x)", "Square root. Exact for perfect squares, proven otherwise; negative x gives an imaginary answer.", catRoots, "sqrt(2)"},
	"cbrt":  {"cbrt(x)", "Cube root, defined for negative numbers too.", catRoots, "cbrt(-27)"},
	"root":  {"root(x, n)", "The n-th root of x.", catRoots, "root(32, 5)"},
	"abs":   {"abs(x)", "Absolute value: the distance from zero.", catRoots, "abs(-7.5)"},
	"hypot": {"hypot(a, b)", "Hypotenuse √(a² + b²), with units if both sides share a kind.", catRoots, "hypot(3, 4)"},
	"exp2":  {"exp2(x)", "2 raised to the power x; exact for whole x.", catRoots, "exp2(10)"},

	"exp":   {"exp(x)", "e raised to the power x.", catLogs, "exp(1)"},
	"ln":    {"ln(x)", "Natural logarithm, base e.", catLogs, "ln(10)"},
	"log":   {"log(x)  log(x, base)", "Natural logarithm, or the logarithm in any base; exact when the answer is a whole power.", catLogs, "log(8, 2)"},
	"log2":  {"log2(x)", "Base-2 logarithm.", catLogs, "log2(1024)"},
	"log10": {"log10(x)", "Base-10 logarithm.", catLogs, "log10(1000)"},

	"sin":   {"sin(x)", "Sine. x is in radians, or degrees when the deg switch is on.", catTrig, "sin(pi/6)"},
	"cos":   {"cos(x)", "Cosine.", catTrig, "cos(pi/3)"},
	"tan":   {"tan(x)", "Tangent.", catTrig, "tan(pi/4)"},
	"sec":   {"sec(x)", "Secant, 1 ÷ cos(x).", catTrig, "sec(0)"},
	"csc":   {"csc(x)", "Cosecant, 1 ÷ sin(x).", catTrig, "csc(pi/2)"},
	"cot":   {"cot(x)", "Cotangent, 1 ÷ tan(x).", catTrig, "cot(pi/4)"},
	"asin":  {"asin(x)", "Arcsine: the angle whose sine is x.", catTrig, "asin(1)"},
	"acos":  {"acos(x)", "Arccosine: the angle whose cosine is x.", catTrig, "acos(0)"},
	"atan":  {"atan(x)", "Arctangent: the angle whose tangent is x.", catTrig, "atan(1)"},
	"atan2": {"atan2(y, x)", "The angle of the point (x, y) from the positive x-axis, in the right quadrant.", catTrig, "atan2(1, -1)"},
	"deg":   {"deg(x)", "Turns radians into degrees.", catTrig, "deg(pi)"},
	"rad":   {"rad(x)", "Turns degrees into radians.", catTrig, "rad(180)"},

	"sinh":  {"sinh(x)", "Hyperbolic sine.", catHyper, "sinh(1)"},
	"cosh":  {"cosh(x)", "Hyperbolic cosine.", catHyper, "cosh(1)"},
	"tanh":  {"tanh(x)", "Hyperbolic tangent.", catHyper, "tanh(1)"},
	"asinh": {"asinh(x)", "Inverse hyperbolic sine.", catHyper, "asinh(1)"},
	"acosh": {"acosh(x)", "Inverse hyperbolic cosine, for x of 1 and above.", catHyper, "acosh(2)"},
	"atanh": {"atanh(x)", "Inverse hyperbolic tangent, for x strictly between -1 and 1.", catHyper, "atanh(0.5)"},

	"floor": {"floor(x)  floor(x, places)", "Round down, optionally to some decimal places.", catRounding, "floor(2.789, 2)"},
	"ceil":  {"ceil(x)  ceil(x, places)", "Round up.", catRounding, "ceil(2.1)"},
	"round": {"round(x)  round(x, places)", "Round to nearest, halves going up; works on proven values when the result is certain.", catRounding, "round(pi, 3)"},
	"trunc": {"trunc(x)  trunc(x, places)", "Cut toward zero.", catRounding, "trunc(-2.7)"},
	"frac":  {"frac(x)", "The fractional part, x - trunc(x).", catRounding, "frac(2.75)"},
	"sign":  {"sign(x)", "-1, 0 or 1; undecided if x might be zero.", catRounding, "sign(-4)"},
	"clamp": {"clamp(x, lo, hi)", "Keeps x between two limits.", catRounding, "clamp(12, 0, 10)"},

	"gcd":       {"gcd(a, b, …)", "Greatest common divisor of two or more whole numbers, or a list.", catWhole, "gcd(48, 18)"},
	"lcm":       {"lcm(a, b, …)", "Least common multiple.", catWhole, "lcm(4, 6, 10)"},
	"mod":       {"mod(a, m)", "Remainder after division, never negative for a positive modulus. Also written a mod m.", catWhole, "mod(-7, 3)"},
	"modpow":    {"modpow(b, e, m)", "b to the power e, mod m, without building the huge power.", catWhole, "modpow(2, 1000, 97)"},
	"modinv":    {"modinv(a, m)", "The number that, multiplied by a, leaves 1 mod m.", catWhole, "modinv(3, 11)"},
	"factor":    {"factor(n)", "Prime factorisation.", catWhole, "factor(360)"},
	"divisors":  {"divisors(n)", "Every divisor of n, as a list.", catWhole, "divisors(28)"},
	"totient":   {"totient(n)", "How many numbers up to n share no factor with it.", catWhole, "totient(36)"},
	"isprime":   {"isprime(n)", "True or false; above 3.3 × 10^24 it says probably true.", catWhole, "isprime(97)"},
	"nextprime": {"nextprime(n)", "The first prime above n.", catWhole, "nextprime(100)"},
	"prevprime": {"prevprime(n)", "The last prime below n.", catWhole, "prevprime(100)"},
	"primes":    {"primes(n)  primes(a, b)  primes(list)", "The primes up to n, between a and b, or within a list.", catWhole, "primes(30)"},
	"fib":       {"fib(n)", "The n-th Fibonacci number, exact at any size.", catWhole, "fib(100)"},
	"nCr":       {"nCr(n, k)", "Ways to choose k of n when order does not matter.", catWhole, "nCr(52, 5)"},
	"nPr":       {"nPr(n, k)", "Ways to arrange k of n when order matters.", catWhole, "nPr(5, 2)"},
	"digitsum":  {"digitsum(n)  digitsum(n, base)", "Sum of the digits.", catWhole, "digitsum(12345)"},
	"tobase":    {"tobase(n, base)", "Writes n in any base from 2 to 36.", catWhole, "tobase(255, 2)"},
	"hex":       {"hex(n)", "Writes n in base 16.", catWhole, "hex(255)"},
	"bin":       {"bin(n)", "Writes n in base 2.", catWhole, "bin(10)"},
	"oct":       {"oct(n)", "Writes n in base 8.", catWhole, "oct(64)"},

	"sum":        {"sum(list)", "Adds the values; exact for exact input.", catStats, "sum(1..100)"},
	"product":    {"product(list)", "Multiplies the values.", catStats, "product(1..10)"},
	"mean":       {"mean(list)", "The average.", catStats, "mean([2, 4, 9])"},
	"median":     {"median(list)", "The middle value once sorted.", catStats, "median([3, 1, 4, 1, 5])"},
	"mode":       {"mode(list)", "The most common value, or all of them if tied.", catStats, "mode([1, 2, 2, 3])"},
	"min":        {"min(list)", "The smallest value; works across units of one kind.", catStats, "min(3 m, 250 cm)"},
	"max":        {"max(list)", "The largest value.", catStats, "max(4, 9, 2)"},
	"count":      {"count(list)", "How many values.", catStats, "count(1..10)"},
	"stdev":      {"stdev(list)", "Sample standard deviation, dividing by n - 1.", catStats, "stdev([2, 4, 4, 4, 5, 5, 7, 9])"},
	"stdevp":     {"stdevp(list)", "Population standard deviation, dividing by n.", catStats, "stdevp([2, 4, 4, 4, 5, 5, 7, 9])"},
	"variance":   {"variance(list)", "Sample variance, dividing by n - 1.", catStats, "variance([1, 2, 3, 4])"},
	"variancep":  {"variancep(list)", "Population variance, dividing by n.", catStats, "variancep([1, 2, 3, 4])"},
	"percentile": {"percentile(list, p)", "The p-th percentile, interpolating between neighbours.", catStats, "percentile(1..100, 90)"},

	"len":     {"len(list)", "How many items.", catLists, "len([4, 8, 15])"},
	"sort":    {"sort(list)", "Smallest to largest, across units of one kind.", catLists, "sort([3, 1, 2])"},
	"reverse": {"reverse(list)", "The list backwards.", catLists, "reverse(1..5)"},
	"unique":  {"unique(list)", "Drops repeats, keeping the first.", catLists, "unique([1, 1, 2, 3, 3])"},
	"cumsum":  {"cumsum(list)", "Running totals.", catLists, "cumsum([10, 20, 30])"},
	"diff":    {"diff(list)", "Differences between neighbours.", catLists, "diff([1, 4, 9, 16])"},

	"solve":  {"solve(equation, x)  solve(equation, x, lo, hi)", "Exact roots where algebra allows, proven brackets otherwise.", catCalculus, "solve(x^2 - 5x + 6 = 0, x)"},
	"deriv":  {"deriv(f, x)  deriv(f, x, at)", "Symbolic derivative, or its value at a point.", catCalculus, "deriv(x^3, x, 2)"},
	"gamma":  {"gamma(x)", "The gamma function, extending the factorial: gamma(n) = (n - 1)!.", catSpecial, "gamma(0.5)"},
	"erf":    {"erf(x)", "The error function.", catSpecial, "erf(1)"},
	"erfc":   {"erfc(x)", "The complementary error function, 1 - erf(x).", catSpecial, "erfc(2)"},
	"expand": {"expand(x, k=2)", "Widens a measured uncertainty by a coverage factor.", catMeasure, "expand(9.81 +- 0.02, k=2)"},
}

func Docs() map[string]FunctionDoc { return functionDocs }

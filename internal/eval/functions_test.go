package eval

import (
	"strings"
	"testing"
)

func TestNewFunctionsAnswer(t *testing.T) {
	for src, want := range map[string]string{
		"fib(10)":                   "55",
		"fib(100)":                  "354224848179261915075",
		"fib(-8)":                   "-21",
		"mod(-7, 3)":                "2",
		"mod(7, -3)":                "-2",
		"mod(7.5, 2)":               "1.5",
		"modpow(2, 1000, 97)":       "36",
		"modpow(3, -1, 11)":         "4",
		"modinv(3, 11)":             "4",
		"factor(360)":               "2^3 · 3^2 · 5",
		"factor(-84)":               "-2^2 · 3 · 7",
		"factor(600851475143)":      "71 · 839 · 1471 · 6857",
		"factor(2^64 + 1)":          "274177 · 67280421310721",
		"divisors(12)":              "[1, 2, 3, 4, 6, 12]",
		"totient(36)":               "12",
		"nextprime(100)":            "101",
		"prevprime(100)":            "97",
		"primes(30)":                "[2, 3, 5, 7, 11, 13, 17, 19, 23, 29]",
		"primes(10, 30)":            "[11, 13, 17, 19, 23, 29]",
		"primes(1..20)":             "[2, 3, 5, 7, 11, 13, 17, 19]",
		"digitsum(12345)":           "15",
		"tobase(255, 16)":           "ff",
		"hex(255)":                  "0xff",
		"bin(10)":                   "0b1010",
		"oct(-8)":                   "-0o10",
		"gcd(12, 18, 24)":           "6",
		"lcm(4, 6, 10)":             "60",
		"sign(-3)":                  "-1",
		"sign(pi - 3)":              "1",
		"trunc(-2.7)":               "-2",
		"trunc(2.789, 2)":           "2.78",
		"frac(-2.75)":               "-0.75",
		"frac(2.5 m)":               "0.5 m",
		"clamp(5, 0, 3)":            "3",
		"clamp(120 cm, 0 m, 1 m)":   "1 m",
		"cbrt(-8)":                  "-2",
		"cbrt(8/27)":                "2/3  ≈ 0.666666666667",
		"cbrt(2)":                   "1.25992104989",
		"hypot(3, 4)":               "5",
		"hypot(30 cm, 0.4 m)":       "0.5 m",
		"exp2(10)":                  "1024",
		"exp2(-3)":                  "0.125",
		"csc(pi/6)":                 "2",
		"asinh(-1)":                 "-0.88137358702",
		"acosh(1)":                  "0",
		"atanh(0.5)":                "0.549306144334",
		"deg(pi)":                   "180",
		"rad(180)":                  "3.14159265359",
		"gamma(5)":                  "24",
		"gamma(0.5)":                "1.77245385091",
		"gamma(-0.5)":               "-3.54490770181",
		"gamma(10.1)":               "454760.751442",
		"erf(1)":                    "0.84270079295",
		"erfc(3)":                   "0.0000220904969986",
		"sort(3, pi, 1/3, sqrt(2))": "[1/3, 1.41421356237, 3, 3.14159265359]",
		"sort([2 m, 30 cm, 1 km])":  "[30 cm, 2 m, 1 km]",
		"reverse(1..5)":             "[5, 4, 3, 2, 1]",
		"unique([1, 2, 2, 3, 1])":   "[1, 2, 3]",
		"len(1..100)":               "100",
		"mode([1, 1, 2, 2, 3])":     "[1, 2]",
		"percentile(1..100, 90)":    "90.1",
		"cumsum([1 m, 50 cm])":      "[1 m, 1.5 m]",
		"diff([1, 4, 9, 16])":       "[3, 5, 7]",
	} {
		if got, err := firstLineOf(t, src); got != want {
			t.Errorf("%s = %q (%s), want %q", src, got, err, want)
		}
	}
}

func TestNewFunctionsRefuseWhatTheyCannotPromise(t *testing.T) {
	for src, why := range map[string]string{
		"fib(2.5)":                   "whole number",
		"mod(5, 0)":                  "mod by zero",
		"modinv(4, 8)":               "share the factor 4",
		"factor(0)":                  "no prime factorisation",
		"prevprime(2)":               "no prime below 2",
		"nextprime(4 * 10^24)":       "where primality can be proven",
		"sign(sqrt(2)*sqrt(2) - 2)":  "undecided",
		"clamp(5, 3, 0)":             "lower limit below the upper",
		"hypot(1 m, 1 s)":            "same kind of quantity",
		"cot(0)":                     "undefined where tan is zero",
		"acosh(0.5)":                 "1 and above",
		"atanh(1)":                   "strictly between -1 and 1",
		"gamma(0)":                   "pole",
		"gamma(-3)":                  "pole",
		"sort([sqrt(2)*sqrt(2), 2])": "undecided",
		"unique([pi, pi])":           "needs exact values",
		"cumsum([1 +- 0.1, 2])":      "measured uncertainty",
		"diff([20 degC, 25 degC])":   "absolute temperatures",
		"percentile([1, 2], 101)":    "from 0 to 100",
		"gcd(6 km, 4)":               "plain numbers",
		"cbrt(2 +- 0.1)":             "measured uncertainty",
	} {
		if _, err := firstLineOf(t, src); !strings.Contains(err, why) {
			t.Errorf("%s refused with %q, want it to mention %q", src, err, why)
		}
	}
}

func TestFactorMarksAProbablePrimeFactor(t *testing.T) {
	got, err := solvedText(t, "factor(2^89 - 1)")
	if err != "" || !strings.Contains(got, "618970019642690137449562111") || !strings.Contains(got, "only a probable prime") {
		t.Errorf("factor(2^89 - 1) = %q (%s)", got, err)
	}
}

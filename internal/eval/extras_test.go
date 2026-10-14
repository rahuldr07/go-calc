package eval

import (
	"math/big"
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

func firstLineOf(t *testing.T, src string) (string, string) {
	t.Helper()
	node, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	v, err := NewEnv().Eval(node)
	if err != nil {
		return "", err.Error()
	}
	head, _, _ := strings.Cut(format.Text(v, nil), "\n")
	return head, ""
}

func TestLogTakesAnOptionalBase(t *testing.T) {
	for src, want := range map[string]string{
		"log(8, 2)":     "3",
		"log(1/8, 2)":   "-3",
		"log(1, 7)":     "0",
		"log(1000, 10)": "3",
		"log(10, 3)":    "2.09590327429",
	} {
		if got, err := firstLineOf(t, src); got != want {
			t.Errorf("%s = %q (%s), want %q", src, got, err, want)
		}
	}
	for src, why := range map[string]string{
		"log(5, 1)":        "cannot be 1",
		"log(0, 2)":        "not positive",
		"log(8, -2)":       "must be positive",
		"log(8, 2, 3)":     "optional base",
		"log(2 +- 0.1, 2)": "measured uncertainty",
	} {
		if _, err := firstLineOf(t, src); !strings.Contains(err, why) {
			t.Errorf("%s refused with %q, want it to mention %q", src, err, why)
		}
	}
}

func TestRoundingToPlacesIsExactAndRoundsProvenValuesSafely(t *testing.T) {
	for src, want := range map[string]string{
		"round(pi, 3)":    "3.142",
		"round(pi)":       "3",
		"floor(e, 4)":     "2.7182",
		"ceil(e, 4)":      "2.7183",
		"round(2.345, 2)": "2.35",
		"round(1234, -2)": "1200",
		"round(1/3, 5)":   "0.33333",
		"round(2.5 m, 0)": "3 m",
	} {
		if got, err := firstLineOf(t, src); got != want {
			t.Errorf("%s = %q (%s), want %q", src, got, err, want)
		}
	}
	if _, err := firstLineOf(t, "round(pi, 1.5)"); !strings.Contains(err, "whole number of decimal places") {
		t.Errorf("fractional places refused with %q", err)
	}
}

func TestCombinationsAndPermutationsAreExact(t *testing.T) {
	for src, want := range map[string]string{
		"nCr(5, 2)":  "10",
		"nCr(52, 5)": "2598960",
		"nCr(7, 0)":  "1",
		"nPr(5, 2)":  "20",
		"nPr(5, 0)":  "1",
		"nPr(5, 5)":  "120",
	} {
		if got, err := firstLineOf(t, src); got != want {
			t.Errorf("%s = %q (%s), want %q", src, got, err, want)
		}
	}
	for _, src := range []string{"nCr(2, 5)", "nCr(5.5, 2)", "nPr(-1, 0)", "nCr(5)"} {
		if _, err := firstLineOf(t, src); err == "" {
			t.Errorf("%s was accepted", src)
		}
	}
}

func TestListsIndexFromZeroAndFromTheEnd(t *testing.T) {
	for src, want := range map[string]string{
		"[10, 20, 30][0]":     "10",
		"[10, 20, 30][-1]":    "30",
		"(1..10)[4]":          "5",
		"[1 m, 2 m][1] in cm": "200 cm",
		"[1, 2, 3][1] * 2":    "4",
	} {
		if got, err := firstLineOf(t, src); got != want {
			t.Errorf("%s = %q (%s), want %q", src, got, err, want)
		}
	}
	for src, why := range map[string]string{
		"[1, 2, 3][3]":   "index 3 is outside a list of 3",
		"[1, 2, 3][-4]":  "index -4 is outside",
		"[1, 2, 3][1.5]": "whole number",
		"5[0]":           "only a list can be indexed",
		"[][0]":          "empty",
	} {
		if _, err := firstLineOf(t, src); !strings.Contains(err, why) {
			t.Errorf("%s refused with %q, want it to mention %q", src, err, why)
		}
	}
}

func solvedText(t *testing.T, src string) (string, string) {
	t.Helper()
	node, _, err := parse.Parse(src)
	if err != nil {
		t.Fatalf("%s: %v", src, err)
	}
	v, err := NewEnv().Eval(node)
	if err != nil {
		return "", err.Error()
	}
	return format.Text(v, nil), ""
}

func TestASolutionThatFillsAStretchIsReportedAsOne(t *testing.T) {
	for src, want := range map[string]string{
		"solve(floor(x) = 0, x)":        "x in [0, 0.9]",
		"solve(round(x) = 3, x)":        "x in [2.5, 3.4]",
		"solve(ceil(x) = 2, x, 0, 4)":   "x in [1.002, 2]",
		"solve(floor(x) = 0, x, -5, 5)": "x in [0, 0.995]",
	} {
		got, err := solvedText(t, src)
		if err != "" {
			t.Fatalf("%s refused: %s", src, err)
		}
		head, rest, _ := strings.Cut(got, "\n")
		if head != want {
			t.Errorf("%s = %q, want %q", src, head, want)
		}
		if !strings.Contains(rest, "proven over the whole stretch") || strings.Contains(rest, "bisection") {
			t.Errorf("%s explained itself as %q", src, rest)
		}
	}
}

func TestAStretchThatCannotBeProvenIsNotCalledARoot(t *testing.T) {
	_, err := solvedText(t, "solve(abs(x) - abs(x) = 0, x, 0, 1)")
	if !strings.Contains(err, "could not be proven for the values between them") {
		t.Errorf("an unprovable stretch gave %q", err)
	}
}

func TestAGridSearchSaysWhatItCanMiss(t *testing.T) {
	got, err := solvedText(t, "solve(cos(x) = x, x)")
	if err != "" {
		t.Fatal(err)
	}
	if !strings.Contains(got, "roots closer together than the grid step of 0.1") {
		t.Errorf("the search did not admit its blind spots: %q", got)
	}
	exact, _ := solvedText(t, "solve(x^2 - 2 = 0, x)")
	if strings.Contains(exact, "can be missed") {
		t.Errorf("a closed-form solution claimed it could miss roots: %q", exact)
	}
}

func TestPrimalityIsProvenBelowTheBoundAndProbableAbove(t *testing.T) {
	const limit = 200000
	composite := make([]bool, limit)
	for i := 2; i*i < limit; i++ {
		if !composite[i] {
			for j := i * i; j < limit; j += i {
				composite[j] = true
			}
		}
	}
	for n := 0; n < limit; n++ {
		want := n >= 2 && !composite[n]
		if got := primality(big.NewInt(int64(n))); got.Holds() != want || got.Truth == value.Likely {
			t.Fatalf("isprime(%d) = %v, want %v", n, got.Truth, want)
		}
	}
	for _, fooler := range []string{"561", "3215031751", "3825123056546413051", "318665857834031151167461"} {
		n, _ := new(big.Int).SetString(fooler, 10)
		if primality(n).Holds() {
			t.Errorf("%s is composite but was called prime", fooler)
		}
	}
	mersenne := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 89), big.NewInt(1))
	if got := primality(mersenne); got.Truth != value.Likely || got.Caveat == "" {
		t.Errorf("2^89 - 1 lies above the proven bound and must be marked probable, got %+v", got.Truth)
	}
	if got, _ := firstLineOf(t, "isprime(2^89 - 1)"); got != "probably true" {
		t.Errorf("isprime(2^89 - 1) printed %q", got)
	}
}

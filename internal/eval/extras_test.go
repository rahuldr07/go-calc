package eval

import (
	"strings"
	"testing"

	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
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

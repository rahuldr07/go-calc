package suggest

import "testing"

func TestClosest(t *testing.T) {
	names := []string{"sqrt", "sin", "isprime", "solve", "pi", "Pa"}
	cases := []struct {
		word string
		want string
		ok   bool
	}{
		{"sqre", "sqrt", true},
		{"sqtr", "sqrt", true},
		{"isprim", "isprime", true},
		{"slove", "solve", true},
		{"SQRE", "sqrt", true},
		{"PA", "Pa", true},
		{"pii", "pi", true},
		{"xyzzy", "", false},
		{"sqrt", "", false},
		{"", "", false},
		{"ab", "", false},
		{"sx", "", false},
	}
	for _, c := range cases {
		got, ok := Closest(c.word, names)
		if got != c.want || ok != c.ok {
			t.Errorf("Closest(%q) = %q, %v; want %q, %v", c.word, got, ok, c.want, c.ok)
		}
	}
}

func TestClosestReturnsRealCasing(t *testing.T) {
	got, ok := Closest("kb", []string{"kB", "KB"})
	if !ok || got != "KB" {
		t.Errorf("Closest(kb) = %q, %v; want KB, true", got, ok)
	}
}

func TestClosestNeverSuggestsTheWordItself(t *testing.T) {
	if got, ok := Closest("sqrt", []string{"sqrt"}); ok {
		t.Errorf("suggested %q for an exact match", got)
	}
}

func TestClosestWithNoCandidates(t *testing.T) {
	if _, ok := Closest("sqre", nil); ok {
		t.Error("suggested something from an empty list")
	}
}

func TestClosestBreaksTiesAlphabetically(t *testing.T) {
	got, _ := Closest("cat", []string{"cot", "cut", "cab"})
	if got != "cab" {
		t.Errorf("tie went to %q, want cab", got)
	}
}

func TestTranspositionCostsOne(t *testing.T) {
	if d := distance([]rune("sqtr"), []rune("sqrt")); d != 1 {
		t.Errorf("distance = %d, want 1", d)
	}
}

func TestDid(t *testing.T) {
	if got := Did("unknown function sqre", "sqre", []string{"sqrt"}); got != "unknown function sqre: did you mean sqrt?" {
		t.Errorf("got %q", got)
	}
	if got := Did("unknown function xyzzy", "xyzzy", []string{"sqrt"}); got != "unknown function xyzzy" {
		t.Errorf("got %q", got)
	}
}

func TestSortedMergesAndDeduplicates(t *testing.T) {
	got := Sorted([]string{"b", "a"}, []string{"a", "c"})
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("got %v", got)
	}
}

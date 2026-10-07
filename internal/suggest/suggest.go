package suggest

import (
	"sort"
	"strings"
)

func Closest(word string, candidates []string) (string, bool) {
	if word == "" {
		return "", false
	}
	target := []rune(strings.ToLower(word))
	limit := allowedDistance(len(target))
	best, bestDistance := "", limit+1
	for _, candidate := range candidates {
		if candidate == word || candidate == "" {
			continue
		}
		d := distance(target, []rune(strings.ToLower(candidate)))
		if d < bestDistance || (d == bestDistance && candidate < best) {
			best, bestDistance = candidate, d
		}
	}
	return best, bestDistance <= limit
}

func Sorted(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, s := range list {
			if !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	sort.Strings(out)
	return out
}

func Did(message, word string, candidates []string) string {
	if s, ok := Closest(word, candidates); ok {
		return message + ": did you mean " + s + "?"
	}
	return message
}

func allowedDistance(length int) int {
	switch {
	case length < 3:
		return 0
	case length < 6:
		return 1
	}
	return 2
}

func distance(a, b []rune) int {
	rows := make([][]int, len(a)+1)
	for i := range rows {
		rows[i] = make([]int, len(b)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d := min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d = min(d, rows[i-2][j-2]+1)
			}
			rows[i][j] = d
		}
	}
	return rows[len(a)][len(b)]
}

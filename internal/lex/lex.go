package lex

import (
	"strings"
	"unicode"
)

const Version = "0.1.0"

const operators = "+-*/^!%×÷·±=<>≤≥≠"

const pairedWithEquals = "<>=!"

const RangeOp = ".."

var wordOperators = map[string]bool{"mod": true, "to": true, "in": true, "of": true}

var comparisonSymbols = map[rune]string{'≤': "<=", '≥': ">=", '≠': "!="}

func Scan(src string) ([]Token, error) { return ScanFrom(src, 1) }

func ScanFrom(src string, first int) ([]Token, error) {
	var toks []Token
	if first < 1 {
		first = 1
	}
	line, col := first, 1
	rs := []rune(src)
	var separating []bool
	groupsDigits := func() bool { return len(separating) == 0 || !separating[len(separating)-1] }
	for i := 0; i < len(rs); {
		r := rs[i]
		start := Pos{Line: line, Col: col}
		switch {
		case r == '\n':
			i++
			line++
			col = 1
		case unicode.IsSpace(r):
			i++
			col++
		case r == '#':
			j := i
			for j < len(rs) && rs[j] != '\n' {
				j++
			}
			toks = append(toks, Token{Comment, string(rs[i:j]), start})
			col += j - i
			i = j
		case unicode.IsDigit(r) || (r == '.' && i+1 < len(rs) && unicode.IsDigit(rs[i+1])):
			j := scanNumber(rs, i, groupsDigits())
			toks = append(toks, Token{Num, string(rs[i:j]), start})
			col += j - i
			i = j
		case r == '.' && i+1 < len(rs) && rs[i+1] == '.':
			toks = append(toks, Token{Op, RangeOp, start})
			i += 2
			col += 2
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
				j++
			}
			toks = append(toks, Token{Ident, string(rs[i:j]), start})
			col += j - i
			i = j
		case r == '(':
			separating = append(separating, followsCallName(toks))
			toks = append(toks, Token{LParen, "(", start})
			i++
			col++
		case r == ')':
			separating = pop(separating)
			toks = append(toks, Token{RParen, ")", start})
			i++
			col++
		case r == '[':
			separating = append(separating, true)
			toks = append(toks, Token{LBracket, "[", start})
			i++
			col++
		case r == ']':
			separating = pop(separating)
			toks = append(toks, Token{RBracket, "]", start})
			i++
			col++
		case r == ',':
			toks = append(toks, Token{Comma, ",", start})
			i++
			col++
		case strings.ContainsRune(operators, r):
			text := string(r)
			if r == '+' && i+1 < len(rs) && rs[i+1] == '-' {
				text = "+-"
				i++
				col++
			}
			if strings.ContainsRune(pairedWithEquals, r) && i+1 < len(rs) && rs[i+1] == '=' {
				text = string(r) + "="
				i++
				col++
			}
			if symbol, ok := comparisonSymbols[r]; ok {
				text = symbol
			}
			toks = append(toks, Token{Op, text, start})
			i++
			col++
		default:
			return nil, &Error{Pos: start, Msg: "unexpected character " + string(r)}
		}
	}
	return append(toks, Token{EOF, "", Pos{Line: line, Col: col}}), nil
}

func followsCallName(toks []Token) bool {
	if len(toks) == 0 {
		return false
	}
	last := toks[len(toks)-1]
	return last.Kind == Ident && !wordOperators[last.Text]
}

func pop(stack []bool) []bool {
	if len(stack) == 0 {
		return stack
	}
	return stack[:len(stack)-1]
}

func scanNumber(rs []rune, i int, groupsDigits bool) int {
	if rs[i] == '0' && i+1 < len(rs) && strings.ContainsRune("xXbBoO", rs[i+1]) {
		j := i + 2
		for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
			j++
		}
		return j
	}
	j := i
	seenDot := false
	seenExp := false
	seenSpread := false
	for j < len(rs) {
		switch {
		case unicode.IsDigit(rs[j]) || rs[j] == '_':
			j++
		case rs[j] == '(' && seenDot && !seenExp && !seenSpread:
			end := spreadDigits(rs, j)
			if end == j {
				return j
			}
			seenSpread = true
			j = end
		case rs[j] == ',' && groupsDigits && !seenDot && !seenExp && leadingGroup(rs, i, j):
			end := groupedDigits(rs, j)
			if end == j {
				return j
			}
			j = end
		case rs[j] == '.' && !seenDot && j+1 < len(rs) && unicode.IsDigit(rs[j+1]):
			seenDot = true
			j++
		case (rs[j] == 'e' || rs[j] == 'E') && j+1 < len(rs) && (unicode.IsDigit(rs[j+1]) || ((rs[j+1] == '-' || rs[j+1] == '+') && j+2 < len(rs) && unicode.IsDigit(rs[j+2]))):
			seenExp = true
			j += 2
		default:
			return j
		}
	}
	return j
}

func leadingGroup(rs []rune, i, j int) bool {
	start := i
	for k := i; k < j; k++ {
		switch {
		case rs[k] == '_':
			start = k + 1
		case !unicode.IsDigit(rs[k]):
			return false
		}
	}
	return j-start >= 1 && j-start <= 3
}

func groupedDigits(rs []rune, j int) int {
	end := j
	for j < len(rs) && rs[j] == ',' {
		k := j + 1
		for k < len(rs) && unicode.IsDigit(rs[k]) {
			k++
		}
		size := k - j - 1
		if size != 2 && size != 3 {
			return end
		}
		if size == 3 {
			end = k
		}
		j = k
	}
	return end
}

func spreadDigits(rs []rune, j int) int {
	k := j + 1
	for k < len(rs) && unicode.IsDigit(rs[k]) {
		k++
	}
	if k == j+1 || k >= len(rs) || rs[k] != ')' {
		return j
	}
	return k + 1
}

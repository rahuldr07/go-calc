package lex

import (
	"strings"
	"unicode"
)

const Version = "0.1.0"

const operators = "+-*/^!%×÷·±="

func Scan(src string) ([]Token, error) {
	var toks []Token
	line, col := 1, 1
	rs := []rune(src)
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
			j := scanNumber(rs, i)
			toks = append(toks, Token{Num, string(rs[i:j]), start})
			col += j - i
			i = j
		case unicode.IsLetter(r) || r == '_':
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
				j++
			}
			toks = append(toks, Token{Ident, string(rs[i:j]), start})
			col += j - i
			i = j
		case r == '(':
			toks = append(toks, Token{LParen, "(", start})
			i++
			col++
		case r == ')':
			toks = append(toks, Token{RParen, ")", start})
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
			toks = append(toks, Token{Op, text, start})
			i++
			col++
		default:
			return nil, &Error{Pos: start, Msg: "unexpected character " + string(r)}
		}
	}
	return append(toks, Token{EOF, "", Pos{Line: line, Col: col}}), nil
}

func scanNumber(rs []rune, i int) int {
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
	for j < len(rs) {
		switch {
		case unicode.IsDigit(rs[j]) || rs[j] == '_':
			j++
		case rs[j] == ',' && !seenDot && !seenExp && leadingGroup(rs, i, j):
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
	if j-i < 1 || j-i > 3 {
		return false
	}
	for k := i; k < j; k++ {
		if !unicode.IsDigit(rs[k]) {
			return false
		}
	}
	return true
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

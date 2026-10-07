package parse

import (
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/lex"
)

var functionNames = map[string]bool{
	"sin": true, "cos": true, "tan": true, "asin": true, "acos": true, "atan": true,
	"sqrt": true, "exp": true, "log": true, "log10": true, "log2": true, "abs": true,
	"floor": true, "ceil": true, "round": true, "gcd": true, "lcm": true, "isprime": true,
	"root": true, "binomial": true, "integrate": true, "diff": true, "plot": true, "expand": true, "solve": true,
}

var keywords = map[string]bool{"mod": true, "to": true, "in": true, "of": true}

type parser struct {
	toks  []lex.Token
	i     int
	notes []Note
}

func Parse(src string) (Node, []Note, error) { return ParseFrom(src, 1) }

func ParseFrom(src string, line int) (Node, []Note, error) {
	toks, err := lex.ScanFrom(src, line)
	if err != nil {
		return nil, nil, err
	}
	kept := make([]lex.Token, 0, len(toks))
	for _, t := range toks {
		if t.Kind != lex.Comment {
			kept = append(kept, t)
		}
	}
	p := &parser{toks: kept}
	n, err := p.parseStatement()
	if err != nil {
		return nil, nil, err
	}
	if p.peek().Kind != lex.EOF {
		return nil, nil, p.errorAt(p.peek(), "unexpected "+p.peek().Kind.String())
	}
	return n, regroupings(n, p.notes), nil
}

func (p *parser) peek() lex.Token { return p.toks[p.i] }

func (p *parser) next() lex.Token {
	t := p.toks[p.i]
	if p.i < len(p.toks)-1 {
		p.i++
	}
	return t
}

func (p *parser) errorAt(t lex.Token, msg string) error {
	return &lex.Error{Pos: t.Pos, Msg: msg}
}

func (p *parser) parseStatement() (Node, error) {
	start := p.i
	if p.peek().Kind == lex.Ident {
		name := p.next()
		var params []string
		if p.peek().Kind == lex.LParen {
			save := p.i
			p.next()
			ok := true
			for p.peek().Kind == lex.Ident {
				params = append(params, p.next().Text)
				if p.peek().Kind == lex.Comma {
					p.next()
					continue
				}
				break
			}
			if p.peek().Kind != lex.RParen {
				ok = false
			} else {
				p.next()
			}
			if !ok || p.peek().Text != "=" {
				p.i = save
				params = nil
			}
		}
		if p.peek().Kind == lex.Op && p.peek().Text == "=" {
			p.next()
			body, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			return &Assign{Name: name.Text, Params: params, Body: body, P: name.Pos}, nil
		}
		p.i = start
	}
	return p.parseExpr(0)
}

var binPower = map[string]int{
	"<": 2, ">": 2, "<=": 2, ">=": 2, "==": 2, "!=": 2,
	"to": 5, "in": 5,
	"+": 10, "-": 10, "±": 10, "+-": 10,
	"*": 20, "/": 20, "×": 20, "÷": 20, "·": 20, "mod": 20, "of": 20,
	"^": 30,
}

const (
	implicitPower = 20
	powerPower    = 30
)

func (p *parser) parseExpr(min int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		op := t.Text
		isKeyword := t.Kind == lex.Ident && keywords[op]
		if t.Kind != lex.Op && !isKeyword {
			if !p.implicitFollows(t) || implicitPower < min {
				return left, nil
			}
			if name, ok := left.(*Name); ok && functionNames[name.Text] {
				return nil, p.errorAt(t, "function call needs parentheses: did you mean "+name.Text+"(…)?")
			}
			right, err := p.parseExpr(powerPower)
			if err != nil {
				return nil, err
			}
			left = &Binary{Op: "*", L: left, R: right, P: t.Pos, Implicit: true}
			continue
		}
		if op == "=" {
			return left, nil
		}
		power, ok := binPower[op]
		if !ok || power < min {
			return left, nil
		}
		p.next()
		nextMin := power + 1
		if op == "^" {
			nextMin = power
		}
		right, err := p.parseExpr(nextMin)
		if err != nil {
			return nil, err
		}
		left = &Binary{Op: op, L: left, R: right, P: t.Pos}
	}
}

func (p *parser) namedArgument() string {
	if p.peek().Kind != lex.Ident || p.i+1 >= len(p.toks) {
		return ""
	}
	next := p.toks[p.i+1]
	if next.Kind != lex.Op || next.Text != "=" {
		return ""
	}
	name := p.next().Text
	p.next()
	return name
}

func (p *parser) implicitFollows(t lex.Token) bool {
	switch t.Kind {
	case lex.Num, lex.LParen:
		return true
	case lex.Ident:
		return !keywords[t.Text]
	}
	return false
}

func (p *parser) parseUnary() (Node, error) {
	t := p.peek()
	if t.Kind == lex.Op && (t.Text == "-" || t.Text == "+") {
		p.next()
		x, err := p.parseExpr(implicitPower + 1)
		if err != nil {
			return nil, err
		}
		if t.Text == "+" {
			return x, nil
		}
		return &Unary{Op: "-", X: x, P: t.Pos}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (Node, error) {
	n, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.Kind == lex.Op && (t.Text == "%" || t.Text == "!") {
			p.next()
			n = &Unary{Op: t.Text, X: n, P: t.Pos}
			continue
		}
		return n, nil
	}
}

func (p *parser) parsePrimary() (Node, error) {
	t := p.peek()
	if t.Kind == lex.EOF {
		return nil, p.errorAt(t, "unexpected end of input")
	}
	p.next()
	switch t.Kind {
	case lex.Num:
		if n, ok := concise(t); ok {
			return n, nil
		}
		return &NumLit{Text: t.Text, P: t.Pos}, nil
	case lex.Ident:
		if p.peek().Kind == lex.LParen {
			p.next()
			var args []Node
			var keys []string
			named := false
			if p.peek().Kind != lex.RParen {
				for {
					key := ""
					if t.Text != "solve" {
						key = p.namedArgument()
					}
					a, err := p.parseExpr(0)
					if err != nil {
						return nil, err
					}
					if t.Text == "solve" && p.peek().Kind == lex.Op && p.peek().Text == "=" {
						equals := p.next()
						right, err := p.parseExpr(0)
						if err != nil {
							return nil, err
						}
						a = &Binary{Op: "=", L: a, R: right, P: equals.Pos}
					}
					args = append(args, a)
					keys = append(keys, key)
					named = named || key != ""
					if p.peek().Kind == lex.Comma {
						p.next()
						continue
					}
					break
				}
			}
			if p.peek().Kind != lex.RParen {
				return nil, p.errorAt(p.peek(), "expected )")
			}
			p.next()
			if !named {
				keys = nil
			}
			return &Call{Fn: t.Text, Args: args, Keys: keys, P: t.Pos}, nil
		}
		return &Name{Text: t.Text, P: t.Pos}, nil
	case lex.LParen:
		n, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().Kind != lex.RParen {
			return nil, p.errorAt(p.peek(), "expected )")
		}
		p.next()
		MarkGrouped(n)
		return n, nil
	}
	return nil, p.errorAt(t, "unexpected "+t.Kind.String())
}

func concise(t lex.Token) (Node, bool) {
	open := strings.IndexByte(t.Text, '(')
	if open < 0 {
		return nil, false
	}
	end := strings.IndexByte(t.Text, ')')
	if end < open {
		return nil, false
	}
	mantissa, spread, tail := t.Text[:open], t.Text[open+1:end], t.Text[end+1:]
	scale := 0
	if dot := strings.IndexByte(mantissa, '.'); dot >= 0 {
		scale = -len(strings.ReplaceAll(mantissa[dot+1:], "_", ""))
	}
	if tail != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(tail[1:], "+"))
		if err != nil {
			return nil, false
		}
		scale += n
	}
	value := &NumLit{Text: mantissa + tail, P: t.Pos}
	sigma := &NumLit{Text: spread + "e" + strconv.Itoa(scale), P: t.Pos}
	return &Binary{Op: "±", L: value, R: sigma, P: t.Pos}, true
}

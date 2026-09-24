package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/value"
)

func main() {
	asJSON := flag.Bool("json", false, "print machine-readable output")
	indian := flag.Bool("indian", false, "group digits the Indian way")
	grouped := flag.Bool("group", false, "group digits")
	flag.Parse()
	style := format.Style{Digits: format.Digits, Grouped: *grouped || *indian, Indian: *indian}
	args := flag.Args()
	if len(args) > 1 {
		switch args[0] {
		case "run":
			out, code := runFile(args[1], style, *asJSON)
			emit(out, code)
		case "trace":
			out, code := explain(strings.Join(args[1:], " "), false, style)
			emit(out, code)
		case "why":
			out, code := explain(strings.Join(args[1:], " "), true, style)
			emit(out, code)
		}
	}
	out, code := run(args, style, *asJSON)
	emit(out, code)
}

func emit(out string, code int) {
	if code == 0 {
		fmt.Println(out)
	} else {
		fmt.Fprintln(os.Stderr, out)
	}
	os.Exit(code)
}

func run(args []string, style format.Style, asJSON bool) (string, int) {
	if len(args) == 0 {
		return "usage: gocalc [--json] [--indian] \"expression\"\n       gocalc run sheet.calc\n       gocalc trace \"expression\"\n       gocalc why \"expression\"", 2
	}
	src := strings.Join(args, " ")
	return evalLine(eval.NewEnv(), src, style, asJSON)
}

func evalLine(env *eval.Env, src string, style format.Style, asJSON bool) (string, int) {
	node, notes, err := parse.Parse(src)
	if err != nil {
		return err.Error(), 1
	}
	v, err := env.Eval(node)
	if err != nil {
		return err.Error(), 1
	}
	line := doc.Line{Src: src, Value: v, Notes: append(notes, env.Notes()...), Warning: env.Warning(node, v)}
	return render(line, style, asJSON, env.Arena())
}

func explain(src string, why bool, style format.Style) (string, int) {
	env := eval.NewEnv()
	node, _, err := parse.Parse(src)
	if err != nil {
		return err.Error(), 1
	}
	v, err := env.Eval(node)
	if err != nil {
		return err.Error(), 1
	}
	if why {
		return comparison(env.Why(node, v), style), 0
	}
	tree := strings.TrimRight(env.Arena().Tree(v.Prov), "\n")
	if tree == "" {
		return "no provenance was recorded for this expression", 1
	}
	return tree, 0
}

func comparison(x eval.Explanation, style format.Style) string {
	if !x.Modelled {
		return "this expression has no float64 shadow, so there is nothing to compare"
	}
	var b strings.Builder
	b.WriteString("proven:  " + provenText(x.Proven, style) + "\n")
	b.WriteString("float64: " + strconv.FormatFloat(x.Shadow, 'g', -1, 64) + "\n")
	b.WriteString("they agree to " + agreement(x.Digits))
	if x.WrongDigits > 0 {
		b.WriteString("\nprinted in full, float64 gives " + x.ShadowExact + ", wrong in its last " + plural(x.WrongDigits, "digit"))
	}
	if len(x.Lost) == 0 {
		b.WriteString("\nno step loses a significant number of bits")
		return b.String()
	}
	b.WriteString("\nwhere the digits went:")
	for _, site := range x.Lost {
		b.WriteString("\n  " + bits(site.BitsLost) + " at the " + site.Op + " at " + site.Pos.String())
	}
	return b.String()
}

func provenText(v value.Value, style format.Style) string {
	out := style.Number(v)
	if v.Unit != "" {
		out += " " + v.Unit
	}
	return out
}

func agreement(digits float64) string {
	if math.IsInf(digits, 1) {
		return "every digit it has"
	}
	whole := int(digits)
	switch whole {
	case 0:
		return "no significant digits"
	case 1:
		return "1 significant digit"
	}
	return strconv.Itoa(whole) + " significant digits"
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func bits(lost float64) string {
	rounded := int(math.Round(lost))
	if rounded == 1 {
		return "1 bit"
	}
	return strconv.Itoa(rounded) + " bits"
}

func render(l doc.Line, style format.Style, asJSON bool, arena *prov.Arena) (string, int) {
	if l.Err != nil {
		return l.Err.Error(), 1
	}
	if asJSON {
		b, err := format.JSONDetail(l.Src, l.Value, l.Notes, format.Detail{Arena: arena, Warning: l.Warning})
		if err != nil {
			return err.Error(), 1
		}
		return string(b), 0
	}
	out := style.Text(l.Value, l.Notes)
	if l.Warning != "" {
		out += "\n  " + l.Warning
	}
	return out, 0
}

func runFile(path string, style format.Style, asJSON bool) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return err.Error(), 1
	}
	defer f.Close()
	var src []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		src = append(src, strings.TrimSpace(sc.Text()))
	}
	if err := sc.Err(); err != nil {
		return err.Error(), 1
	}
	var out []string
	code := 0
	sheet := doc.New()
	for _, l := range sheet.SetLines(src) {
		if doc.Blank(l.Src) {
			continue
		}
		text, c := render(l, style, asJSON, sheet.Arena())
		if c != 0 {
			code = c
			text = fmt.Sprintf("line %d: %s", l.N+1, text)
		}
		out = append(out, l.Src+"\n  = "+strings.ReplaceAll(text, "\n", "\n  "))
	}
	return strings.Join(out, "\n"), code
}

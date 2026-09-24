package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
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
			out, code := explain(strings.Join(args[1:], " "), false)
			emit(out, code)
		case "why":
			out, code := explain(strings.Join(args[1:], " "), true)
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

func explain(src string, why bool) (string, int) {
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
		return env.Why(node, v), 0
	}
	tree := strings.TrimRight(env.Arena().Tree(v.Prov), "\n")
	if tree == "" {
		return "no provenance was recorded for this expression", 1
	}
	return tree, 0
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

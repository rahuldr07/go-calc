package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/value"
)

func main() {
	opts, args, err := options(os.Args[1:])
	if err != nil {
		emit(err.Error(), 2)
	}
	style := styleFor(opts)
	out, code := dispatch(opts, args, style)
	if code == startREPL {
		os.Exit(repl(os.Stdin, os.Stdout, style))
	}
	emit(out, code)
}

const startREPL = -1

func styleFor(opts settings) format.Style {
	return format.Style{Digits: format.Digits, Grouped: opts.grouped || opts.indian, Indian: opts.indian}
}

func dispatch(opts settings, args []string, style format.Style) (string, int) {
	if len(args) == 1 && args[0] == "repl" {
		if opts.json {
			return noJSONFor("repl"), 2
		}
		return "", startREPL
	}
	if len(args) > 1 {
		switch args[0] {
		case "run":
			if opts.json {
				return runFileJSON(args[1], style)
			}
			return runFile(args[1], style)
		case "trace", "why":
			if opts.json {
				return noJSONFor(args[0]), 2
			}
			return explain(strings.Join(args[1:], " "), args[0] == "why", style)
		}
	}
	return run(args, style, opts.json)
}

type settings struct {
	json    bool
	indian  bool
	grouped bool
}

const usage = "usage: gocalc [--json] [--indian] [--group] \"expression\"\n" +
	"       gocalc repl\n" +
	"       gocalc [--json] run sheet.calc\n" +
	"       gocalc trace \"expression\"\n" +
	"       gocalc why \"expression\"\n" +
	"options start with two dashes, so a leading - begins an expression: gocalc \"-2^2\""

var errHelp = errors.New(usage)

func options(argv []string) (settings, []string, error) {
	var out settings
	for len(argv) > 0 {
		arg := argv[0]
		if arg == "--" {
			return out, argv[1:], nil
		}
		name, ok := strings.CutPrefix(arg, "--")
		if !ok {
			break
		}
		switch name {
		case "json":
			out.json = true
		case "indian":
			out.indian = true
		case "group":
			out.grouped = true
		case "help":
			return out, nil, errHelp
		default:
			return out, nil, errors.New("unknown option " + arg + "\n" + usage)
		}
		argv = argv[1:]
	}
	return out, argv, nil
}

func noJSONFor(command string) string {
	return "--json is not available for " + command + "; gocalc --json \"expression\" carries the same answer, warning and provenance graph as machine-readable " + format.Schema
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
		return usage, 2
	}
	src := strings.Join(args, " ")
	return evalLine(eval.NewEnv(), src, style, asJSON)
}

func evalLine(env *eval.Env, src string, style format.Style, asJSON bool) (string, int) {
	out, _, code := evalWith(env, src, eval.DefaultCtx(), style, asJSON)
	return out, code
}

func evalWith(env *eval.Env, src string, ctx *eval.Ctx, style format.Style, asJSON bool) (string, eval.Explanation, int) {
	node, notes, err := parse.Parse(src)
	if err != nil {
		return err.Error(), eval.Explanation{}, 1
	}
	v, err := env.EvalCtx(node, ctx)
	if err != nil {
		return err.Error(), eval.Explanation{}, 1
	}
	x := env.WhyCtx(node, v, ctx)
	line := doc.Line{Src: src, Value: v, Notes: append(notes, env.Notes()...), Warning: x.Warning}
	out, code := render(line, style, asJSON, env.Arena())
	return out, x, code
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
	return strings.ReplaceAll(style.Text(v, nil), "\n", "\n"+provenIndent)
}

const provenIndent = "         "

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

func sheetLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var src []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		src = append(src, strings.TrimSpace(sc.Text()))
	}
	return src, sc.Err()
}

func runFile(path string, style format.Style) (string, int) {
	src, err := sheetLines(path)
	if err != nil {
		return err.Error(), 1
	}
	var out []string
	code := 0
	sheet := doc.New()
	for _, l := range sheet.SetLines(src) {
		if doc.Blank(l.Src) {
			continue
		}
		text, c := render(l, style, false, sheet.Arena())
		if c != 0 {
			code = c
			if !located(l.Err) {
				text = fmt.Sprintf("line %d: %s", l.N+1, text)
			}
		}
		out = append(out, l.Src+"\n  = "+strings.ReplaceAll(text, "\n", "\n  "))
	}
	return strings.Join(out, "\n"), code
}

func located(err error) bool {
	var pointed *lex.Error
	return errors.As(err, &pointed)
}

func runFileJSON(path string, style format.Style) (string, int) {
	src, err := sheetLines(path)
	if err != nil {
		return err.Error(), 1
	}
	req, err := json.Marshal(struct {
		Lines  []string `json:"lines"`
		Digits int      `json:"digits"`
	}{Lines: src, Digits: style.Digits})
	if err != nil {
		return err.Error(), 1
	}
	answer := doc.EvalRequest(req)
	var read struct {
		Lines []struct {
			Status string `json:"status"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(answer, &read); err != nil {
		return err.Error(), 1
	}
	for _, l := range read.Lines {
		if l.Status == doc.StatusError {
			return string(answer), 1
		}
	}
	return string(answer), 0
}

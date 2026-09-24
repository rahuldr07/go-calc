package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
)

func main() {
	asJSON := flag.Bool("json", false, "print machine-readable output")
	indian := flag.Bool("indian", false, "group digits the Indian way")
	grouped := flag.Bool("group", false, "group digits")
	flag.Parse()
	style := format.Style{Digits: format.Digits, Grouped: *grouped || *indian, Indian: *indian}
	args := flag.Args()
	if len(args) > 1 && args[0] == "run" {
		out, code := runFile(args[1], style, *asJSON)
		emit(out, code)
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
		return "usage: gocalc [--json] [--indian] \"expression\"\n       gocalc run sheet.calc", 2
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
	if asJSON {
		b, err := format.JSON(src, v, notes)
		if err != nil {
			return err.Error(), 1
		}
		return string(b), 0
	}
	return style.Text(v, notes), 0
}

func runFile(path string, style format.Style, asJSON bool) (string, int) {
	f, err := os.Open(path)
	if err != nil {
		return err.Error(), 1
	}
	defer f.Close()
	env := eval.NewEnv()
	var out []string
	code := 0
	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		src := strings.TrimSpace(sc.Text())
		if src == "" || strings.HasPrefix(src, "#") {
			continue
		}
		text, c := evalLine(env, src, style, asJSON)
		if c != 0 {
			code = c
			text = fmt.Sprintf("line %d: %s", line, text)
		}
		out = append(out, src+"\n  = "+strings.ReplaceAll(text, "\n", "\n  "))
	}
	if err := sc.Err(); err != nil {
		return err.Error(), 1
	}
	return strings.Join(out, "\n"), code
}

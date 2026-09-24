package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/doc"
	"github.com/rahuldr07/go-calc/internal/eval"
	"github.com/rahuldr07/go-calc/internal/format"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	prompt        = "> "
	banner        = "go-calc — every answer exact or proven. help lists the commands, exit leaves."
	maxDigits     = 1000
	lookupLimit   = 40
	lookupForm    = "?<prefix>"
	escapeCommand = "unset"
)

var functions = []string{
	"abs", "acos", "asin", "atan", "atan2", "ceil", "cos", "cosh", "exp",
	"expand", "floor", "gcd", "isprime", "lcm", "ln", "log", "log10", "log2",
	"root", "round", "sin", "sinh", "sqrt", "tan", "tanh",
}

var functionAbout = map[string]string{
	"abs":     "abs(x) is the size of x without its sign",
	"acos":    "acos(x) is the angle whose cosine is x, in radians unless deg is set",
	"asin":    "asin(x) is the angle whose sine is x, in radians unless deg is set",
	"atan":    "atan(x) is the angle whose tangent is x, in radians unless deg is set",
	"atan2":   "atan2(y, x) is the angle from the x axis to the point (x, y)",
	"ceil":    "ceil(x) is the whole number at or above x",
	"cos":     "cos(x) is the cosine of x, read in radians unless deg is set",
	"cosh":    "cosh(x) is the hyperbolic cosine of x",
	"exp":     "exp(x) is e raised to the power x",
	"expand":  "expand(x, k=2) widens an uncertainty by a coverage factor; expand(x, 95%) by a probability",
	"floor":   "floor(x) is the whole number at or below x",
	"gcd":     "gcd(a, b) is the greatest common divisor of two whole numbers",
	"isprime": "isprime(n) answers 1 when n is prime and 0 when it is not",
	"lcm":     "lcm(a, b) is the least common multiple of two whole numbers",
	"ln":      "ln(x) is the natural logarithm of x",
	"log":     "log(x) is the natural logarithm of x; log10 and log2 are the other bases",
	"log10":   "log10(x) is the base-10 logarithm of x",
	"log2":    "log2(x) is the base-2 logarithm of x",
	"root":    "root(x, n) is the nth root of x, for a whole n",
	"round":   "round(x) is x rounded to the nearest whole number, halves going up",
	"sin":     "sin(x) is the sine of x, read in radians unless deg is set",
	"sinh":    "sinh(x) is the hyperbolic sine of x",
	"sqrt":    "sqrt(x) is the square root of x, exact when x is a perfect square",
	"tan":     "tan(x) is the tangent of x, read in radians unless deg is set",
	"tanh":    "tanh(x) is the hyperbolic tangent of x",
}

type repler struct {
	env     *eval.Env
	style   format.Style
	digits  int
	degrees bool
	last    eval.Explanation
	answer  bool
}

type command struct {
	name  string
	arg   string
	about string
	run   func(*repler, io.Writer, string) bool
}

var (
	commands []command
	byName   map[string]command
)

func init() {
	commands = []command{
		{"why", "[expression]", "compare the proven answer with what float64 would say", (*repler).why},
		{"trace", "[name]", "show where every number in a result came from", (*repler).trace},
		{"info", "<name>", "say what a name, unit or constant is", (*repler).info},
		{"digits", "[n]", "print this many significant digits", (*repler).setDigits},
		{"deg", "", "read and write angles in degrees", (*repler).setDegrees},
		{"rad", "", "read and write angles in radians", (*repler).setRadians},
		{"unset", "<name>", "give a name back to the calculator", (*repler).unset},
		{"help", "", "list these commands", (*repler).help},
		{"exit", "", "leave, saving the history", (*repler).quit},
	}
	byName = make(map[string]command, len(commands))
	for _, c := range commands {
		byName[c.name] = c
	}
}

func repl(in io.Reader, out io.Writer, style format.Style) int {
	r := &repler{env: eval.NewEnv(), style: style, digits: style.Digits}
	if r.digits < 1 {
		r.digits = format.Digits
	}
	r.style.Digits = r.digits
	past := openHistory()
	reader := bufio.NewReader(in)
	interactive := isTerminal(out)
	if interactive {
		fmt.Fprintln(out, banner)
	}
	code := 0
	for {
		if interactive {
			fmt.Fprint(out, prompt)
		}
		text, err := reader.ReadString('\n')
		line := strings.TrimSpace(text)
		if line != "" {
			past.add(line)
			if !doc.Blank(line) && r.handle(out, line) {
				break
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				fmt.Fprintln(out, "error: "+err.Error())
				code = 1
			} else if interactive {
				fmt.Fprintln(out)
			}
			break
		}
	}
	past.save(out)
	return code
}

func (r *repler) handle(out io.Writer, line string) bool {
	if rest, ok := strings.CutPrefix(line, "?"); ok {
		return r.lookup(out, strings.TrimSpace(rest))
	}
	verb, rest := split(line)
	c, ok := byName[verb]
	if !ok || (c.name != escapeCommand && r.defines(verb)) || claims(c, rest) {
		return r.compute(out, line)
	}
	return c.run(r, out, rest)
}

func (r *repler) defines(name string) bool {
	for _, known := range r.env.Names() {
		if known == name {
			return true
		}
	}
	return false
}

func claims(c command, rest string) bool {
	if strings.HasPrefix(rest, "=") {
		return true
	}
	return c.arg == "" && rest != ""
}

func split(line string) (string, string) {
	verb, rest, _ := strings.Cut(line, " ")
	return verb, strings.TrimSpace(rest)
}

func (r *repler) ctx() *eval.Ctx {
	c := eval.DefaultCtx()
	c.Digits = r.digits
	c.Degrees = r.degrees
	return c
}

func (r *repler) fail(out io.Writer, msg string) bool {
	fmt.Fprintln(out, "error: "+msg)
	return false
}

func (r *repler) compute(out io.Writer, src string) bool {
	was := r.shadowed()
	text, x, code := evalWith(r.env, src, r.ctx(), r.style, false)
	if code != 0 {
		return r.fail(out, text)
	}
	r.last, r.answer = x, true
	fmt.Fprintln(out, text)
	now := r.shadowed()
	for _, c := range commands {
		if now[c.name] && !was[c.name] {
			fmt.Fprintln(out, "  "+c.name+" is now a name of yours, so the "+c.name+" command is out of reach until "+escapeCommand+" "+c.name)
		}
	}
	return false
}

func (r *repler) shadowed() map[string]bool {
	out := map[string]bool{}
	for _, name := range r.env.Names() {
		if _, ok := byName[name]; ok {
			out[name] = true
		}
	}
	return out
}

func (r *repler) value(src string) (value.Value, eval.Explanation, error) {
	node, _, err := parse.Parse(src)
	if err != nil {
		return value.Value{}, eval.Explanation{}, err
	}
	c := r.ctx()
	v, err := r.env.EvalCtx(node, c)
	if err != nil {
		return value.Value{}, eval.Explanation{}, err
	}
	return v, r.env.WhyCtx(node, v, c), nil
}

func (r *repler) why(out io.Writer, rest string) bool {
	x := r.last
	if rest != "" {
		_, fresh, err := r.value(rest)
		if err != nil {
			return r.fail(out, err.Error())
		}
		x, r.last, r.answer = fresh, fresh, true
	} else if !r.answer {
		return r.fail(out, "nothing has been worked out yet, so there is nothing to explain")
	}
	fmt.Fprintln(out, comparison(x, r.style))
	return false
}

func (r *repler) trace(out io.Writer, rest string) bool {
	held := r.last.Proven
	switch v, ok := r.env.Get(rest); {
	case rest == "" && !r.answer:
		return r.fail(out, "nothing has been worked out yet, so there is nothing to trace")
	case rest == "":
	case ok:
		held = v
	default:
		fresh, _, err := r.value(rest)
		if err != nil {
			return r.fail(out, err.Error())
		}
		held = fresh
	}
	tree := strings.TrimRight(r.env.Arena().Tree(held.Prov), "\n")
	if tree == "" {
		return r.fail(out, "no provenance was recorded for that value")
	}
	fmt.Fprintln(out, tree)
	return false
}

func (r *repler) info(out io.Writer, name string) bool {
	if name == "" {
		return r.fail(out, "info needs a name")
	}
	if v, ok := r.env.Get(name); ok {
		fmt.Fprintln(out, name+"  [name]  defined in this session")
		fmt.Fprintln(out, indented(r.style.Text(v, nil)))
		return false
	}
	if r.defines(name) {
		fmt.Fprintln(out, name+"  [function]  defined in this session")
		return false
	}
	v, _, err := r.value(name)
	if err != nil {
		if about, ok := functionAbout[name]; ok {
			fmt.Fprintln(out, name+"  [function]  built in")
			fmt.Fprintln(out, "  "+about)
			return false
		}
		return r.fail(out, "nothing is known about "+name)
	}
	if c, ok := consts.Lookup(name); ok {
		v.Unc.Sigma = stated(c)
	}
	if tree := strings.TrimRight(r.env.Arena().Tree(v.Prov), "\n"); tree != "" {
		fmt.Fprintln(out, tree)
	}
	fmt.Fprintln(out, indented(r.style.Text(v, nil)))
	return false
}

func stated(c consts.Const) float64 {
	u, _ := c.Uncertainty.Rat().Float64()
	if scale, ok := units.Lookup(c.Unit); ok && c.Unit != "" {
		f, _ := scale.Factor.Rat().Float64()
		u *= f
	}
	return u
}

func indented(text string) string {
	return "  = " + strings.ReplaceAll(text, "\n", "\n  ")
}

func (r *repler) setDigits(out io.Writer, rest string) bool {
	if rest == "" {
		fmt.Fprintln(out, "digits is "+strconv.Itoa(r.digits))
		return false
	}
	n, err := strconv.Atoi(rest)
	if err != nil || n < 1 || n > maxDigits {
		return r.fail(out, "digits needs a whole number between 1 and "+strconv.Itoa(maxDigits))
	}
	r.digits, r.style.Digits = n, n
	fmt.Fprintln(out, "digits is "+strconv.Itoa(n))
	return false
}

func (r *repler) setDegrees(out io.Writer, _ string) bool {
	r.degrees = true
	fmt.Fprintln(out, "angles are in degrees")
	return false
}

func (r *repler) setRadians(out io.Writer, _ string) bool {
	r.degrees = false
	fmt.Fprintln(out, "angles are in radians")
	return false
}

func (r *repler) unset(out io.Writer, name string) bool {
	if name == "" {
		return r.fail(out, escapeCommand+" needs a name")
	}
	if !r.env.Unset(name) {
		return r.fail(out, "nothing of yours is called "+name)
	}
	fmt.Fprintln(out, name+" is no longer a name of yours"+restored(name))
	return false
}

func restored(name string) string {
	if _, ok := byName[name]; ok {
		return ", so the " + name + " command works again"
	}
	if c, ok := consts.Lookup(name); ok {
		return ", so it reads as " + c.Name + " again"
	}
	if u, ok := units.Lookup(name); ok {
		return ", so it reads as a unit of " + u.Dim.Words() + " again"
	}
	return ""
}

func (r *repler) quit(io.Writer, string) bool { return true }

func (r *repler) help(out io.Writer, _ string) bool {
	forms := make([]string, len(commands))
	width := len(lookupForm)
	for i, c := range commands {
		forms[i] = c.name
		if c.arg != "" {
			forms[i] += " " + c.arg
		}
		if len(forms[i]) > width {
			width = len(forms[i])
		}
	}
	for i, c := range commands {
		fmt.Fprintf(out, "  %-*s  %s\n", width, forms[i], c.about)
	}
	fmt.Fprintf(out, "  %-*s  %s\n", width, lookupForm, "list the names, units and constants that start with it")
	fmt.Fprintln(out, "anything else is an expression. functions: "+strings.Join(functions, " "))
	return false
}

func (r *repler) lookup(out io.Writer, prefix string) bool {
	groups := []struct {
		label string
		found []string
	}{
		{"names", r.names(prefix)},
		{"functions", matching(functions, prefix)},
		{"constants", matching(symbols(), prefix)},
		{"units", unitsMatching(prefix)},
	}
	empty := true
	for _, g := range groups {
		if len(g.found) == 0 {
			continue
		}
		empty = false
		fmt.Fprintln(out, "  "+g.label+": "+strings.Join(g.found, " "))
	}
	if empty {
		return r.fail(out, "nothing known starts with "+quoted(prefix))
	}
	return false
}

func quoted(s string) string {
	if s == "" {
		return "an empty prefix"
	}
	return strconv.Quote(s)
}

func (r *repler) names(prefix string) []string {
	var out []string
	for _, name := range r.env.Names() {
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return capped(out)
}

func symbols() []string {
	all := consts.All()
	out := make([]string, 0, len(all))
	for _, c := range all {
		out = append(out, c.Symbol)
	}
	sort.Strings(out)
	return out
}

func matching(pool []string, prefix string) []string {
	var out []string
	for _, name := range pool {
		if strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return capped(out)
}

func unitsMatching(prefix string) []string {
	out := matching(units.Symbols(), prefix)
	if prefix == "" {
		return out
	}
	if _, ok := units.Lookup(prefix); !ok {
		return out
	}
	for _, name := range out {
		if name == prefix {
			return out
		}
	}
	return capped(append([]string{prefix}, out...))
}

func capped(names []string) []string {
	if len(names) <= lookupLimit {
		return names
	}
	return append(names[:lookupLimit:lookupLimit], "…")
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

const historyLimit = 1000

type history struct {
	path  string
	lines []string
	err   error
}

func openHistory() *history {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return &history{err: errors.New("neither XDG_STATE_HOME nor HOME is set")}
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return &history{path: filepath.Join(dir, "go-calc", "history")}
}

func (h *history) add(line string) { h.lines = append(h.lines, line) }

func (h *history) save(out io.Writer) {
	if h.err == nil && len(h.lines) == 0 {
		return
	}
	if err := h.write(); err != nil {
		fmt.Fprintln(out, "history not saved: "+err.Error())
	}
}

func (h *history) write() error {
	if h.err != nil {
		return h.err
	}
	kept, err := readHistory(h.path)
	if err != nil {
		return err
	}
	kept = append(kept, h.lines...)
	if len(kept) > historyLimit {
		kept = kept[len(kept)-historyLimit:]
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(h.path, []byte(strings.Join(kept, "\n")+"\n"), 0o600)
}

func readHistory(path string) ([]string, error) {
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimRight(string(body), "\n")
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

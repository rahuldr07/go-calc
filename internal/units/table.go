package units

import (
	"errors"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
)

type Unit struct {
	Symbol string
	Dim    Dim
	Factor num.Exact
	Offset num.Exact
	Kind   Kind
}

var (
	none        = Dim{}
	length      = Dim{1, 0, 0, 0, 0, 0, 0, 0}
	mass        = Dim{0, 1, 0, 0, 0, 0, 0, 0}
	timeDim     = Dim{0, 0, 1, 0, 0, 0, 0, 0}
	current     = Dim{0, 0, 0, 1, 0, 0, 0, 0}
	temperature = Dim{0, 0, 0, 0, 1, 0, 0, 0}
	amount      = Dim{0, 0, 0, 0, 0, 1, 0, 0}
	luminous    = Dim{0, 0, 0, 0, 0, 0, 1, 0}
	information = Dim{0, 0, 0, 0, 0, 0, 0, 1}
	force       = Dim{1, 1, -2, 0, 0, 0, 0, 0}
	energy      = Dim{2, 1, -2, 0, 0, 0, 0, 0}
	power       = Dim{2, 1, -3, 0, 0, 0, 0, 0}
	pressure    = Dim{-1, 1, -2, 0, 0, 0, 0, 0}
)

func rat(s string) num.Exact {
	v, err := num.ParseExact(s)
	if err != nil {
		panic("bad unit factor " + s)
	}
	return v
}

var base map[string]Unit

func init() {
	base = map[string]Unit{
		"m":     {"m", length, rat("1"), rat("0"), Absolute},
		"g":     {"g", mass, rat("1/1000"), rat("0"), Absolute},
		"s":     {"s", timeDim, rat("1"), rat("0"), Absolute},
		"A":     {"A", current, rat("1"), rat("0"), Absolute},
		"K":     {"K", temperature, rat("1"), rat("0"), Absolute},
		"mol":   {"mol", amount, rat("1"), rat("0"), Absolute},
		"cd":    {"cd", luminous, rat("1"), rat("0"), Absolute},
		"bit":   {"bit", information, rat("1"), rat("0"), Absolute},
		"B":     {"B", information, rat("8"), rat("0"), Absolute},
		"N":     {"N", force, rat("1"), rat("0"), Absolute},
		"J":     {"J", energy, rat("1"), rat("0"), Absolute},
		"W":     {"W", power, rat("1"), rat("0"), Absolute},
		"Pa":    {"Pa", pressure, rat("1"), rat("0"), Absolute},
		"bar":   {"bar", pressure, rat("100000"), rat("0"), Absolute},
		"inch":  {"inch", length, rat("127/5000"), rat("0"), Absolute},
		"ft":    {"ft", length, rat("381/1250"), rat("0"), Absolute},
		"yd":    {"yd", length, rat("1143/1250"), rat("0"), Absolute},
		"mile":  {"mile", length, rat("201168/125"), rat("0"), Absolute},
		"lb":    {"lb", mass, rat("45359237/100000000"), rat("0"), Absolute},
		"oz":    {"oz", mass, rat("45359237/1600000000"), rat("0"), Absolute},
		"min":   {"min", timeDim, rat("60"), rat("0"), Absolute},
		"h":     {"h", timeDim, rat("3600"), rat("0"), Absolute},
		"day":   {"day", timeDim, rat("86400"), rat("0"), Absolute},
		"week":  {"week", timeDim, rat("604800"), rat("0"), Absolute},
		"year":  {"year", timeDim, rat("31557600"), rat("0"), Absolute},
		"degC":  {"degC", temperature, rat("1"), rat("5463/20"), Absolute},
		"degF":  {"degF", temperature, rat("5/9"), rat("45967/180"), Absolute},
		"lakh":  {"lakh", none, rat("100000"), rat("0"), Absolute},
		"crore": {"crore", none, rat("10000000"), rat("0"), Absolute},
	}
}

var prefixes = map[string]string{
	"Y": "1e24", "Z": "1e21", "E": "1e18", "P": "1e15", "T": "1e12",
	"G": "1e9", "M": "1e6", "k": "1e3", "h": "1e2", "da": "1e1",
	"d": "1e-1", "c": "1e-2", "m": "1e-3", "u": "1e-6", "n": "1e-9",
	"p": "1e-12", "f": "1e-15", "a": "1e-18",
}

var binaryPrefixes = map[string]string{
	"Ki": "1024", "Mi": "1048576", "Gi": "1073741824", "Ti": "1099511627776",
}

func Lookup(sym string) (Unit, bool) {
	if u, ok := base[sym]; ok {
		return u, true
	}
	for p, mul := range binaryPrefixes {
		if strings.HasPrefix(sym, p) && len(sym) > len(p) {
			if u, ok := base[sym[len(p):]]; ok && u.Dim == information {
				u.Symbol = sym
				u.Factor = u.Factor.Mul(rat(mul))
				return u, true
			}
		}
	}
	for p, mul := range prefixes {
		if !strings.HasPrefix(sym, p) || len(sym) == len(p) {
			continue
		}
		rest := sym[len(p):]
		u, ok := base[rest]
		if !ok || u.Offset.Sign() != 0 {
			continue
		}
		u.Symbol = sym
		u.Factor = u.Factor.Mul(rat(mul))
		return u, true
	}
	return Unit{}, false
}

func Convert(v num.Exact, from, to Unit) (num.Exact, error) {
	if from.Dim != to.Dim {
		return num.Exact{}, errors.New("cannot convert " + from.Dim.Words() + " (" + from.Symbol + ") to " + to.Dim.Words() + " (" + to.Symbol + ")")
	}
	si := v.Mul(from.Factor).Add(from.Offset)
	out, err := si.Sub(to.Offset).Div(to.Factor)
	if err != nil {
		return num.Exact{}, err
	}
	return out, nil
}

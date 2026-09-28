package consts

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"strings"

	"github.com/rahuldr07/go-calc/internal/num"
)

//go:embed codata2022.txt
var raw string

const Source = "CODATA 2022 · physics.nist.gov/cuu/Constants"

type Const struct {
	Name        string
	Symbol      string
	Unit        string
	Source      string
	Formula     string
	Value       num.Exact
	Uncertainty num.Exact
	Exact       bool
}

var table = map[string]Const{}

func init() {
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 6 && len(f) != 7 {
			continue
		}
		formula := ""
		if len(f) == 7 {
			formula = f[6]
		}
		v, err := num.ParseExact(f[2])
		if err != nil {
			continue
		}
		u, err := num.ParseExact(f[3])
		if err != nil {
			continue
		}
		table[f[0]] = Const{
			Symbol:      f[0],
			Name:        f[1],
			Value:       v,
			Uncertainty: u,
			Unit:        f[4],
			Exact:       f[5] == "exact",
			Formula:     formula,
			Source:      Source,
		}
	}
}

func Lookup(sym string) (Const, bool) {
	c, ok := table[sym]
	return c, ok
}

func All() []Const {
	out := make([]Const, 0, len(table))
	for _, c := range table {
		out = append(out, c)
	}
	return out
}

func Checksum() string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

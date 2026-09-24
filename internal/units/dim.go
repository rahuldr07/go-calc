package units

import (
	"strconv"
	"strings"
)

type Dim [8]int

var dimNames = [8]string{"length", "mass", "time", "current", "temperature", "amount", "luminous intensity", "information"}

func (d Dim) Add(o Dim) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] + o[i]
	}
	return out
}

func (d Dim) Sub(o Dim) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] - o[i]
	}
	return out
}

func (d Dim) Scale(n int) Dim {
	var out Dim
	for i := range d {
		out[i] = d[i] * n
	}
	return out
}

func (d Dim) IsNone() bool { return d == Dim{} }

func (d Dim) Words() string {
	var pos, neg []string
	for i, e := range d {
		if e == 0 {
			continue
		}
		name := dimNames[i]
		mag := e
		if mag < 0 {
			mag = -mag
		}
		if mag > 1 {
			name += "^" + strconv.Itoa(mag)
		}
		if e > 0 {
			pos = append(pos, name)
		} else {
			neg = append(neg, name)
		}
	}
	switch {
	case len(pos) == 0 && len(neg) == 0:
		return "dimensionless"
	case len(pos) == 0:
		return "1/" + denominator(neg)
	case len(neg) == 0:
		return strings.Join(pos, "·")
	}
	return strings.Join(pos, "·") + "/" + denominator(neg)
}

type Kind int

const (
	Absolute Kind = iota
	Difference
)

func (d Dim) Half() (Dim, bool) {
	var out Dim
	for i, e := range d {
		if e%2 != 0 {
			return Dim{}, false
		}
		out[i] = e / 2
	}
	return out, true
}

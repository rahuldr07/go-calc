package eval

import (
	"errors"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/rahuldr07/go-calc/internal/value"
)

const (
	maxFibonacci    = 1000000
	maxSieve        = 1299709
	maxDivisors     = 100000
	rhoBudget       = 1 << 18
	trialDivisionTo = 10000
)

var (
	errFactorBudget = errors.New("could not finish factoring within the search limit")
	one             = big.NewInt(1)
	two             = big.NewInt(2)
)

func init() {
	register("fib", builtinFunc{min: 1, max: 1, usage: "fib takes one whole number, n", run: fibonacci})
	register("mod", builtinFunc{min: 2, max: 2, usage: "mod takes a value and a modulus", run: modulo})
	register("modpow", builtinFunc{min: 3, max: 3, usage: "modpow takes a base, an exponent and a modulus", run: modpow})
	register("modinv", builtinFunc{min: 2, max: 2, usage: "modinv takes a value and a modulus", run: modinv})
	register("factor", builtinFunc{min: 1, max: 1, usage: "factor takes one whole number", run: factorCall})
	register("divisors", builtinFunc{min: 1, max: 1, usage: "divisors takes one whole number", run: divisorsCall})
	register("totient", builtinFunc{min: 1, max: 1, usage: "totient takes one whole number", run: totientCall})
	register("nextprime", builtinFunc{min: 1, max: 1, usage: "nextprime takes one whole number", run: neighbourPrime(1)})
	register("prevprime", builtinFunc{min: 1, max: 1, usage: "prevprime takes one whole number", run: neighbourPrime(-1)})
	register("primes", builtinFunc{min: 1, max: 2, lists: true, usage: "primes takes an upper limit, a lower and upper limit, or a list to filter", run: primesCall})
	register("digitsum", builtinFunc{min: 1, max: 2, usage: "digitsum takes a whole number and an optional base", run: digitsum})
	register("tobase", builtinFunc{min: 2, max: 2, usage: "tobase takes a whole number and a base from 2 to 36", run: toBase})
	register("hex", builtinFunc{min: 1, max: 1, usage: "hex takes one whole number", run: prefixedBase(16, "0x")})
	register("bin", builtinFunc{min: 1, max: 1, usage: "bin takes one whole number", run: prefixedBase(2, "0b")})
	register("oct", builtinFunc{min: 1, max: 1, usage: "oct takes one whole number", run: prefixedBase(8, "0o")})
}

func fibonacci(c call) (value.Value, error) {
	n, err := c.bounded(0, -maxFibonacci, maxFibonacci)
	if err != nil {
		return value.Value{}, err
	}
	negative := n < 0
	if negative {
		n = -n
	}
	f, _ := fibPair(n)
	if negative && n%2 == 0 {
		f.Neg(f)
	}
	return integer(f), nil
}

func fibPair(n int64) (*big.Int, *big.Int) {
	if n == 0 {
		return big.NewInt(0), big.NewInt(1)
	}
	a, b := fibPair(n / 2)
	twiceB := new(big.Int).Lsh(b, 1)
	c := new(big.Int).Mul(a, new(big.Int).Sub(twiceB, a))
	d := new(big.Int).Add(new(big.Int).Mul(a, a), new(big.Int).Mul(b, b))
	if n%2 == 0 {
		return c, d
	}
	return d, c.Add(c, d)
}

func modulo(c call) (value.Value, error) {
	a, err := c.plain(0)
	if err != nil {
		return value.Value{}, err
	}
	m, err := c.plain(1)
	if err != nil {
		return value.Value{}, err
	}
	if !a.IsExact() || !m.IsExact() {
		return value.Value{}, errors.New("mod needs exact values")
	}
	if m.Exact().Sign() == 0 {
		return value.Value{}, errors.New("mod by zero is undefined")
	}
	q, _ := a.Exact().Div(m.Exact())
	whole := roundTo(q, "floor")
	return value.Scalar(a.Exact().Sub(m.Exact().Mul(whole))), nil
}

func modpow(c call) (value.Value, error) {
	b, err := c.whole(0)
	if err != nil {
		return value.Value{}, err
	}
	e, err := c.whole(1)
	if err != nil {
		return value.Value{}, err
	}
	m, err := c.whole(2)
	if err != nil {
		return value.Value{}, err
	}
	if m.Sign() <= 0 {
		return value.Value{}, errors.New("modpow needs a positive modulus")
	}
	if e.Sign() < 0 {
		inverse := new(big.Int).ModInverse(new(big.Int).Mod(b, m), m)
		if inverse == nil {
			return value.Value{}, errors.New("modpow with a negative exponent needs the base to have an inverse mod " + m.String())
		}
		return integer(new(big.Int).Exp(inverse, new(big.Int).Neg(e), m)), nil
	}
	return integer(new(big.Int).Exp(new(big.Int).Mod(b, m), e, m)), nil
}

func modinv(c call) (value.Value, error) {
	a, err := c.whole(0)
	if err != nil {
		return value.Value{}, err
	}
	m, err := c.whole(1)
	if err != nil {
		return value.Value{}, err
	}
	if m.Cmp(one) <= 0 {
		return value.Value{}, errors.New("modinv needs a modulus above 1")
	}
	inverse := new(big.Int).ModInverse(new(big.Int).Mod(a, m), m)
	if inverse == nil {
		g := new(big.Int).GCD(nil, nil, new(big.Int).Abs(a), m)
		return value.Value{}, errors.New(a.String() + " has no inverse mod " + m.String() + ": they share the factor " + g.String())
	}
	return integer(inverse), nil
}

type primePower struct {
	p        *big.Int
	k        int
	probable bool
}

func factorize(n *big.Int) ([]primePower, error) {
	rest := new(big.Int).Abs(n)
	counts := map[string]*primePower{}
	add := func(p *big.Int, probable bool) {
		key := p.String()
		if pp, ok := counts[key]; ok {
			pp.k++
			return
		}
		counts[key] = &primePower{p: new(big.Int).Set(p), k: 1, probable: probable}
	}
	for d := int64(2); d <= trialDivisionTo; d++ {
		div := big.NewInt(d)
		if new(big.Int).Mul(div, div).Cmp(rest) > 0 {
			break
		}
		for new(big.Int).Mod(rest, div).Sign() == 0 {
			add(div, false)
			rest.Quo(rest, div)
		}
	}
	budget := rhoBudget
	pending := []*big.Int{}
	if rest.Cmp(one) > 0 {
		pending = append(pending, rest)
	}
	for len(pending) > 0 {
		m := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		verdict := primality(m)
		if verdict.Holds() || verdict.Truth == value.Likely {
			add(m, verdict.Truth == value.Likely)
			continue
		}
		d, used := pollardBrent(m, budget)
		budget -= used
		if d == nil {
			return nil, errors.New(errFactorBudget.Error() + ": " + m.String() + " remains, and it is composite")
		}
		pending = append(pending, d, new(big.Int).Quo(m, d))
	}
	out := make([]primePower, 0, len(counts))
	for _, pp := range counts {
		out = append(out, *pp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].p.Cmp(out[j].p) < 0 })
	return out, nil
}

func pollardBrent(n *big.Int, budget int) (*big.Int, int) {
	if n.Bit(0) == 0 {
		return big.NewInt(2), 1
	}
	used := 0
	for seed := int64(1); seed < 64 && used < budget; seed++ {
		y, c := big.NewInt(seed+1), big.NewInt(seed)
		g, r, q := big.NewInt(1), 1, big.NewInt(1)
		var x, ys *big.Int
		step := func(v *big.Int) *big.Int {
			v.Mul(v, v).Add(v, c).Mod(v, n)
			return v
		}
		for g.Cmp(one) == 0 && used < budget {
			x = new(big.Int).Set(y)
			for i := 0; i < r; i++ {
				step(y)
			}
			for k := 0; k < r && g.Cmp(one) == 0; k += 128 {
				ys = new(big.Int).Set(y)
				for i := 0; i < 128 && i < r-k; i++ {
					step(y)
					q.Mul(q, new(big.Int).Abs(new(big.Int).Sub(x, y))).Mod(q, n)
					used++
				}
				g.GCD(nil, nil, q, n)
			}
			r *= 2
		}
		if g.Cmp(n) == 0 {
			for {
				step(ys)
				g.GCD(nil, nil, new(big.Int).Abs(new(big.Int).Sub(x, ys)), n)
				used++
				if g.Cmp(one) > 0 || used >= budget {
					break
				}
			}
		}
		if g.Cmp(one) > 0 && g.Cmp(n) < 0 {
			return g, used
		}
	}
	return nil, used
}

func positiveWhole(c call) (*big.Int, error) {
	n, err := c.whole(0)
	if err != nil {
		return nil, err
	}
	if n.Sign() <= 0 {
		return nil, errors.New(c.name + " takes a whole number above zero")
	}
	return n, nil
}

func factorCall(c call) (value.Value, error) {
	n, err := c.whole(0)
	if err != nil {
		return value.Value{}, err
	}
	if n.Sign() == 0 {
		return value.Value{}, errors.New("0 has no prime factorisation")
	}
	if new(big.Int).Abs(n).Cmp(one) == 0 {
		return value.Worked(n.String()), nil
	}
	powers, err := factorize(n)
	if err != nil {
		return value.Value{}, err
	}
	parts := make([]string, 0, len(powers))
	probable := false
	for _, pp := range powers {
		part := pp.p.String()
		if pp.k > 1 {
			part += "^" + strconv.Itoa(pp.k)
		}
		parts = append(parts, part)
		probable = probable || pp.probable
	}
	text := strings.Join(parts, " · ")
	if n.Sign() < 0 {
		text = "-" + text
	}
	out := value.Worked(text)
	if probable {
		out.Caveat = "a factor above 3.3 × 10^24 is only a probable prime, so the factorisation is not proven"
	}
	return out, nil
}

func divisorsCall(c call) (value.Value, error) {
	n, err := positiveWhole(c)
	if err != nil {
		return value.Value{}, err
	}
	powers, err := factorize(n)
	if err != nil {
		return value.Value{}, err
	}
	count := 1
	for _, pp := range powers {
		count *= pp.k + 1
		if count > maxDivisors {
			return value.Value{}, errors.New(n.String() + " has more than " + strconv.Itoa(maxDivisors) + " divisors, too many to list")
		}
	}
	divisors := []*big.Int{big.NewInt(1)}
	for _, pp := range powers {
		grown := make([]*big.Int, 0, len(divisors)*(pp.k+1))
		for _, d := range divisors {
			power := new(big.Int).Set(d)
			for i := 0; i <= pp.k; i++ {
				grown = append(grown, new(big.Int).Set(power))
				power.Mul(power, pp.p)
			}
		}
		divisors = grown
	}
	sort.Slice(divisors, func(i, j int) bool { return divisors[i].Cmp(divisors[j]) < 0 })
	return integers(divisors), nil
}

func totientCall(c call) (value.Value, error) {
	n, err := positiveWhole(c)
	if err != nil {
		return value.Value{}, err
	}
	powers, err := factorize(n)
	if err != nil {
		return value.Value{}, err
	}
	phi := new(big.Int).Set(n)
	for _, pp := range powers {
		phi.Quo(phi, pp.p)
		phi.Mul(phi, new(big.Int).Sub(pp.p, one))
	}
	return integer(phi), nil
}

func neighbourPrime(direction int64) func(c call) (value.Value, error) {
	return func(c call) (value.Value, error) {
		n, err := c.whole(0)
		if err != nil {
			return value.Value{}, err
		}
		step := big.NewInt(direction)
		at := new(big.Int).Add(n, step)
		for {
			if at.Cmp(two) < 0 {
				return value.Value{}, errors.New("there is no prime below " + n.String())
			}
			if at.Cmp(provenPrimeBound) >= 0 {
				return value.Value{}, errors.New(c.name + " works below 3.3 × 10^24, where primality can be proven")
			}
			if primality(at).Holds() {
				return integer(at), nil
			}
			at.Add(at, step)
		}
	}
}

func sieve(limit int64) []bool {
	composite := make([]bool, limit+1)
	for i := int64(2); i*i <= limit; i++ {
		if !composite[i] {
			for j := i * i; j <= limit; j += i {
				composite[j] = true
			}
		}
	}
	return composite
}

func primesCall(c call) (value.Value, error) {
	if c.args[0].IsList() {
		if len(c.args) != 1 {
			return value.Value{}, errors.New("primes filters one list")
		}
		var kept []value.Value
		for _, item := range c.args[0].List.Items {
			n, ok := wholeInt(item)
			if !ok {
				return value.Value{}, errors.New("primes can only filter whole numbers")
			}
			verdict := primality(n)
			if verdict.Truth == value.Likely {
				return value.Value{}, errors.New(n.String() + " is above 3.3 × 10^24, where primality is only probable")
			}
			if verdict.Holds() {
				kept = append(kept, item)
			}
		}
		return value.List(kept), nil
	}
	lo, hi := int64(2), int64(0)
	var err error
	if len(c.args) == 1 {
		hi, err = c.bounded(0, 0, maxSieve)
	} else {
		lo, err = c.bounded(0, 0, maxSieve)
		if err == nil {
			hi, err = c.bounded(1, 0, maxSieve)
		}
	}
	if err != nil {
		return value.Value{}, errors.New("primes lists primes up to " + strconv.Itoa(maxSieve) + ", the 100000th prime")
	}
	composite := sieve(hi)
	var found []*big.Int
	for i := max(lo, 2); i <= hi; i++ {
		if !composite[i] {
			found = append(found, big.NewInt(i))
		}
	}
	return integers(found), nil
}

func digitsum(c call) (value.Value, error) {
	n, err := c.whole(0)
	if err != nil {
		return value.Value{}, err
	}
	base := int64(10)
	if len(c.args) == 2 {
		if base, err = c.bounded(1, 2, 36); err != nil {
			return value.Value{}, err
		}
	}
	rest := new(big.Int).Abs(n)
	b := big.NewInt(base)
	total := new(big.Int)
	digit := new(big.Int)
	for rest.Sign() > 0 {
		rest.QuoRem(rest, b, digit)
		total.Add(total, digit)
	}
	return integer(total), nil
}

func toBase(c call) (value.Value, error) {
	n, err := c.whole(0)
	if err != nil {
		return value.Value{}, err
	}
	base, err := c.bounded(1, 2, 36)
	if err != nil {
		return value.Value{}, err
	}
	return value.Worked(n.Text(int(base))), nil
}

func prefixedBase(base int, prefix string) func(c call) (value.Value, error) {
	return func(c call) (value.Value, error) {
		n, err := c.whole(0)
		if err != nil {
			return value.Value{}, err
		}
		text := prefix + new(big.Int).Abs(n).Text(base)
		if n.Sign() < 0 {
			text = "-" + text
		}
		return value.Worked(text), nil
	}
}

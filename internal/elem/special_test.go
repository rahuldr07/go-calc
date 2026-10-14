package elem

import (
	"math/big"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
)

func ballOf(t *testing.T, s string, prec uint) num.Ball {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad input %s", s)
	}
	return num.BallFromExact(num.FromRat(r), prec)
}

func encloses(t *testing.T, name string, b num.Ball, want string, digits int) {
	t.Helper()
	w, ok := new(big.Float).SetPrec(512).SetString(want)
	if !ok {
		t.Fatalf("bad reference %s", want)
	}
	if b.Lo().Cmp(w) > 0 || b.Hi().Cmp(w) < 0 {
		t.Errorf("%s = [%s, %s] does not hold %s", name, b.Lo().Text('g', 30), b.Hi().Text('g', 30), want)
	}
	width := new(big.Float).Sub(b.Hi(), b.Lo())
	limit := new(big.Float).SetMantExp(big.NewFloat(1), -int(float64(digits)*3.33))
	scale := new(big.Float).Abs(w)
	if scale.Cmp(big.NewFloat(1)) < 0 {
		scale = big.NewFloat(1)
	}
	if width.Cmp(new(big.Float).Mul(limit, scale)) > 0 {
		t.Errorf("%s is too wide for %d digits: %s", name, digits, width.Text('g', 4))
	}
}

func TestGammaEnclosesReferenceValues(t *testing.T) {
	const prec = 200
	for in, want := range map[string]string{
		"1/2":    "1.7724538509055160272981674833411451827975494561223871282138077898529112845910322",
		"1/3":    "2.6789385347077476336556929409746776441286893779573011009504283275904176101677438",
		"5":      "24",
		"101/10": "454760.7514415859508673358368319076190405047458218916492282448363280859459342243",
		"-1/2":   "-3.5449077018110320545963349666822903655950989122447742564276155797058225691820644",
		"-5/2":   "-0.94530872048294188122568932444861076415869304326527313504736415458821935178188383",
		"1/1000": "999.4237724845954661149822012996440004652176101456122324695421716913960238118284",
		"30":     "8841761993739701954543616000000",
	} {
		g, err := Gamma(ballOf(t, in, prec), prec)
		if err != nil {
			t.Fatalf("gamma(%s): %v", in, err)
		}
		encloses(t, "gamma("+in+")", g, want, 45)
	}
	for _, pole := range []string{"0", "-1", "-7"} {
		if _, err := Gamma(ballOf(t, pole, prec), prec); err != ErrPole {
			t.Errorf("gamma(%s) gave %v, want the pole", pole, err)
		}
	}
}

func TestErfEnclosesReferenceValues(t *testing.T) {
	const prec = 200
	for in, want := range map[string]string{
		"1":    "0.84270079294971486934122063508260925929606699796630290845993789783471725409601084",
		"1/2":  "0.52049987781304653768274665389196452873645157575796370005880572564719352171685357",
		"3":    "0.99997790950300141455862722387041767962015229291260075034276104515705754331637987",
		"-2":   "-0.99532226501895273416206925636725292861089179704006007673835232620043728071999518",
		"6":    "0.99999999999999997848026328750108688340664960081261536952248593831145789947210795",
		"1/10": "0.11246291601828489220327507174396838322169629915970254753449414481759924402553136",
	} {
		e, err := Erf(ballOf(t, in, prec), prec)
		if err != nil {
			t.Fatalf("erf(%s): %v", in, err)
		}
		encloses(t, "erf("+in+")", e, want, 45)
	}
	big, err := Erf(ballOf(t, "40", prec), prec)
	if err != nil {
		t.Fatal(err)
	}
	encloses(t, "erf(40)", big, "1", 45)
}

package elem

import (
	"errors"
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/rahuldr07/go-calc/internal/num"
)

const refDigits = `
pi                 3.14159265358979323846264338327950288419716939937510582097494459230781640628620899862803482534211706798e+0
e                  2.71828182845904523536028747135266249775724709369995957496696762772407663035354759457138217852516642742e+0
sin_1              8.41470984807896506652502321630298999622563060798371065672751709991910404391239668948639743543053e-1
cos_1              5.40302305868139717400936607442976603732310420617922227670097255381100394774471764517951856087183e-1
sin_2              9.09297426825681695396019865911744842702254971447890268378973011530967301540783544620126688924959e-1
cos_2              -4.16146836547142386997568229500762189766000771075544890755149973781964936124079169074531777860169e-1
sin_3              1.41120008059867222100744802808110279846933264252265584151882641232422009967014471911282172853450e-1
cos_3              -9.89992496600445457271572794731261302393679096615588328814085932928329197513133220428294479355693e-1
sin_4              -7.56802495307928251372639094511829094135912887336472571485416773401310493619179416423572810562423e-1
cos_4              -6.53643620863611914639168183097750381424133596646218247007010283852737655810603379901230645646179e-1
sin_5              -9.58924274663138468893154406155993973352461543964601778131672454235102558086559603076995955429533e-1
cos_5              2.83662185463226264466639171513557308334422592252215944930359066586151456767382702286176981668345e-1
sin_7              6.56986598718789090396999091593635177936870010497490074657854334189292837131227031509935121601055e-1
cos_7              7.53902254343304638141197521719182012218313391460126839543613880813876026720717405625428391089302e-1
sin_-1             -8.41470984807896506652502321630298999622563060798371065672751709991910404391239668948639743543053e-1
cos_-1             5.40302305868139717400936607442976603732310420617922227670097255381100394774471764517951856087183e-1
sin_-2             -9.09297426825681695396019865911744842702254971447890268378973011530967301540783544620126688924959e-1
cos_-2             -4.16146836547142386997568229500762189766000771075544890755149973781964936124079169074531777860169e-1
sin_100            -5.06365641109758793656557610459785432065032721290657323443392473594357913419476696499236664512927e-1
cos_100            8.62318872287683934101938513950842535510084008535510829280162112692721088050926624103095105684277e-1
tan_1              1.55740772465490223050697480745836017308725077238152003838394660569886139715172728955509996520224e+0
tan_-2             2.18503986326151899164330610231368254343201774622766316456295586996677374720919418231974354210473e+0
atan_1             7.85398163397448309615660845819875721049292349843776455243736148076954101571552249657008706335529e-1
atan_10            1.47112767430373459185287557176173085185530637718323826247196351934388045569555384489340478823677e+0
atan_0.5           4.63647609000806116214256231461214402028537054286120263810933088720197864165741705300600283984888e-1
atan_-0.5          -4.63647609000806116214256231461214402028537054286120263810933088720197864165741705300600283984888e-1
atan_-10           -1.47112767430373459185287557176173085185530637718323826247196351934388045569555384489340478823677e+0
atan_0.0009765625  9.76562189559319430403430199717290851634197015810087590049007252267637520355084544235815603027727e-4
exp_1              2.71828182845904523536028747135266249775724709369995957496696762772407663035354759457138217852517e+0
exp_0.5            1.64872127070012814684865078781416357165377610071014801157507931164066102119421560863277652005637e+0
exp_-3             4.97870683678639429793424156500617766316995921884232155676277276060606677301995501540542442366333e-2
exp_10             2.20264657948067165169579006452842443663535126185567810742354263552252028185707925751991209681645e+4
exp_-10            4.53999297624848515355915155605506102379180888665649692590713056509994216143022816525250045459478e-5
exp_0.0009765625   1.00097703949241653524284529261160650646585162918174419940186408264916250428896869173656853690882e+0
log_2              6.93147180559945309417232121458176568075500134360255254120680009493393621969694715605863326996419e-1
log2_2             1.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+0
log10_2            3.01029995663981195213738894724493026768189881462108541310427461127108189274424509486927252118186e-1
log_10             2.30258509299404568401799145468436420760110148862877297603332790096757260967735248023599720508960e+0
log2_10            3.32192809488736234787031942948939017586483139302458061205475639581593477660862521585013974335937e+0
log10_10           1.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+0
log_0.5            -6.93147180559945309417232121458176568075500134360255254120680009493393621969694715605863326996419e-1
log2_0.5           -1.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+0
log10_0.5          -3.01029995663981195213738894724493026768189881462108541310427461127108189274424509486927252118186e-1
log_1000000        1.38155105579642741041079487281061852456066089317726378561999674058054356580641148814159832305376e+1
log2_1000000       1.99315685693241740872219165769363410551889883581474836723285383748956086596517512951008384601562e+1
log10_1000000      6.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+0
log_0.00390625     -5.54517744447956247533785697166541254460400107488204203296544007594714897575755772484690661597135e+0
log2_0.00390625    -8.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+0
log10_0.00390625   -2.40823996531184956170991115779594421414551905169686833048341968901686551419539607589541801694549e+0
asin_0.5           5.23598775598298873077107230546583814032861566562517636829157432051302734381034833104672470890353e-1
acos_0.5           1.04719755119659774615421446109316762806572313312503527365831486410260546876206966620934494178071e+0
asin_-0.5          -5.23598775598298873077107230546583814032861566562517636829157432051302734381034833104672470890353e-1
acos_-0.5          2.09439510239319549230842892218633525613144626625007054731662972820521093752413933241868988356141e+0
asin_0.875         1.06543581651073931226000681765232949759419723349387652321962473867275808218106205449051992821503e+0
acos_0.875         5.05360510284157306971314873987421944504387466193676387267847557481150120962042444823497484456029e-1
asin_-0.875        -1.06543581651073931226000681765232949759419723349387652321962473867275808218106205449051992821503e+0
acos_-0.875        2.63623214330563593149132850929208093969278193318142943370709703482666628532416655380453734088609e+0
asin_0.96875       1.32014066445876582548218771892558200600228924601020816001954423418092132451541656054529303108535e+0
acos_0.96875       2.50655662336130793749133972714169436096295453677344750467928061972986878627687938768724381585712e-1
sinh_1             1.17520119364380145688238185059560081515571798133409587022956541301330756730432389560711745208962e+0
cosh_1             1.54308063481524377847790562075706168260152911236586370473740221471076906304922369896426472643554e+0
tanh_1             7.61594155955764888119458282604793590412768597257936551596810500121953244576638483458947521673677e-1
sinh_-2            -3.62686040784701876766821398280126170488634201232113572130948447493425021098878503672360718129423e+0
cosh_-2            3.76219569108363145956221347777374610829397355823071160277764334758832358509027272666070530378489e+0
tanh_-2            -9.64027580075816883946413724100923150255029976240934776048263217413107946317610202559474850045208e-1
sinh_3             1.00178749274099018989745936194658280601781041231828634644056532510463926051808870905252214580082e+1
cosh_3             1.00676619957777658419539360351158898368098037153712866799732809786524532729110866406792757022448e+1
tanh_3             9.95054753686730451331880185255488475097813854700282491823878815130664702782559176719395960022312e-1
sinh_0.0009765625  9.76562655220436504066751500638538073324774201609506945848098026682110368012531090266020246166342e-4
cosh_0.0009765625  1.00000047683719609873877854111096796839252685498013469245601598462248039392095616064630251666266e+0
tanh_0.0009765625  9.76562189559260218584075270146771762847008858368259065160093840393376370842760182923768321765566e-4
atan2_1_1          7.85398163397448309615660845819875721049292349843776455243736148076954101571552249657008706335529e-1
atan2_1_-1         2.35619449019234492884698253745962716314787704953132936573120844423086230471465674897102611900659e+0
atan2_-1_-1        -2.35619449019234492884698253745962716314787704953132936573120844423086230471465674897102611900659e+0
atan2_1_0          1.57079632679489661923132169163975144209858469968755291048747229615390820314310449931401741267106e+0
atan2_-1_0         -1.57079632679489661923132169163975144209858469968755291048747229615390820314310449931401741267106e+0
atan2_-1_1         -7.85398163397448309615660845819875721049292349843776455243736148076954101571552249657008706335529e-1
atan2_3_-4         2.49809154479650885165983415456218024615565880825979343810933847359430393147458790991521798064083e+0
pow_2_0.5          1.41421356237309504880168872420969807856967187537694807317667973799073247846210703885038753432764e+0
pow_2_-0.5         7.07106781186547524400844362104849039284835937688474036588339868995366239231053519425193767163821e-1
pow_10_0.25        1.77827941003892280122542119519268484473579052640225535801183072277630188153949380490030039927870e+0
sqrt_2             1.41421356237309504880168872420969807856967187537694807317667973799073247846210703885038753432764e+0
sqrt_0.5           7.07106781186547524400844362104849039284835937688474036588339868995366239231053519425193767163821e-1
sqrt_1000000       1.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e+3
halfpi             1.57079632679489661923132169163975144209858469968755291048747229615390820314310449931401741267106e+0
threequarterpi     2.35619449019234492884698253745962716314787704953132936573120844423086230471465674897102611900659e+0
`

func refs(t *testing.T) map[string]*big.Float {
	t.Helper()
	out := map[string]*big.Float{}
	for _, line := range splitLines(refDigits) {
		name, digits, ok := splitField(line)
		if !ok {
			continue
		}
		f, _, err := big.ParseFloat(digits, 10, 512, big.ToNearestEven)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out[name] = f
	}
	return out
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

func splitField(line string) (string, string, bool) {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	j := i
	for j < len(line) && line[j] != ' ' {
		j++
	}
	k := j
	for k < len(line) && line[k] == ' ' {
		k++
	}
	m := k
	for m < len(line) && line[m] != ' ' {
		m++
	}
	if i == j || k == m {
		return "", "", false
	}
	return line[i:j], line[k:m], true
}

func pointBall(v float64, prec uint) num.Ball {
	return num.NewBall(new(big.Float).SetPrec(prec).SetFloat64(v), num.Mag{})
}

func TestKnownConstants(t *testing.T) {
	const prec = 200
	want := refs(t)
	cases := []struct {
		name string
		got  num.Ball
		key  string
	}{
		{"pi", Pi(prec), "pi"},
		{"e", E(prec), "e"},
	}
	for _, c := range cases {
		if !c.got.Contains(want[c.key]) {
			t.Errorf("%s: ball %s does not contain the true value", c.name, c.got)
		}
	}
}

func randBall(rnd *rand.Rand, prec uint) num.Ball {
	scale := math.Ldexp(1, rnd.Intn(17)-8)
	mid := new(big.Float).SetPrec(prec).SetFloat64(rnd.NormFloat64() * scale)
	var rad num.Mag
	switch rnd.Intn(3) {
	case 1:
		rad = num.MagFromFloat(math.Abs(rnd.NormFloat64()) * math.Ldexp(1, -rnd.Intn(80)-20))
	case 2:
		rad = num.MagFromFloat(math.Abs(rnd.NormFloat64()) * math.Ldexp(1, rnd.Intn(5)-3))
	}
	return num.NewBall(mid, rad)
}

func TestIdentitiesHoldOnBalls(t *testing.T) {
	const prec = 120
	rnd := rand.New(rand.NewSource(11))
	for i := 0; i < 12000; i++ {
		x := randBall(rnd, prec)
		s, err1 := Sin(x, prec)
		c, err2 := Cos(x, prec)
		if err1 != nil || err2 != nil {
			continue
		}
		sum := s.Mul(s, prec).Add(c.Mul(c, prec), prec)
		if !sum.Contains(big.NewFloat(1)) {
			t.Fatalf("sin²+cos² does not contain 1 for x=%s", x.Mid.Text('g', 20))
		}
		if e, err := Exp(x, prec); err == nil {
			if l, err := Log(e, prec); err == nil && !l.Contains(x.Mid) {
				t.Fatalf("log(exp(x)) does not contain x for x=%s", x.Mid.Text('g', 20))
			}
		}
	}
}

func randWideBall(rnd *rand.Rand, prec uint) num.Ball {
	mid := new(big.Float).SetPrec(prec).SetFloat64(rnd.NormFloat64() * math.Ldexp(1, rnd.Intn(13)-6))
	rad := num.MagFromFloat(math.Abs(rnd.NormFloat64()) * math.Ldexp(1, rnd.Intn(8)-5))
	return num.NewBall(mid, rad)
}

func samplePoint(b num.Ball, theta float64, prec uint) *big.Float {
	off := new(big.Float).SetPrec(prec).SetFloat64(theta)
	off.Mul(off, b.Rad.Float())
	return new(big.Float).SetPrec(prec).Add(b.Mid, off)
}

func overlap(a, b num.Ball) bool {
	return a.Lo().Cmp(b.Hi()) <= 0 && b.Lo().Cmp(a.Hi()) <= 0
}

var unaryFuncs = []struct {
	name string
	f    func(num.Ball, uint) (num.Ball, error)
}{
	{"sqrt", Sqrt}, {"exp", Exp}, {"log", Log}, {"log2", Log2}, {"log10", Log10},
	{"sin", Sin}, {"cos", Cos}, {"tan", Tan}, {"atan", Atan},
	{"asin", Asin}, {"acos", Acos}, {"sinh", Sinh}, {"cosh", Cosh}, {"tanh", Tanh},
}

func TestWideBallsEncloseEveryPointValue(t *testing.T) {
	const prec = 120
	const refPrec = 300
	rnd := rand.New(rand.NewSource(977))
	thetas := []float64{-0.98, -0.5, 0, 0.31, 0.98}
	for i := 0; i < 600; i++ {
		x := randWideBall(rnd, prec)
		if i%2 == 0 {
			x = shrinkIntoUnit(x)
		}
		for _, fn := range unaryFuncs {
			outer, err := fn.f(x, prec)
			if err != nil {
				continue
			}
			for _, th := range thetas {
				p := samplePoint(x, th, refPrec)
				inner, err := fn.f(num.NewBall(p, num.Mag{}), refPrec)
				if err != nil {
					continue
				}
				if !overlap(outer, inner) {
					t.Fatalf("%s: ball %s misses the value at %s inside x=%s",
						fn.name, outer, p.Text('g', 20), x)
				}
			}
		}
	}
}

func shrinkIntoUnit(b num.Ball) num.Ball {
	hi := new(big.Float).Abs(b.Mid)
	hi.Add(hi, b.Rad.Float())
	if hi.Sign() == 0 {
		return b
	}
	scale := new(big.Float).SetPrec(b.Mid.Prec()).Quo(big.NewFloat(0.75), hi)
	mid := new(big.Float).SetPrec(b.Mid.Prec()).Mul(b.Mid, scale)
	rad := b.Rad.MulFloat(scale)
	return num.NewBall(mid, rad)
}

func TestInverseIdentitiesOnWideBalls(t *testing.T) {
	const prec = 120
	rnd := rand.New(rand.NewSource(4242))
	for i := 0; i < 2000; i++ {
		x := randWideBall(rnd, prec)
		if s, err := Sqrt(x, prec); err == nil {
			if !s.Mul(s, prec).Contains(x.Mid) {
				t.Fatalf("sqrt(x)² does not contain x for x=%s", x)
			}
		}
		u := shrinkIntoUnit(x)
		if a, err := Asin(u, prec); err == nil {
			if s, err := Sin(a, prec); err == nil && !s.Contains(u.Mid) {
				t.Fatalf("sin(asin(x)) does not contain x for x=%s", u)
			}
		}
		if a, err := Atan(x, prec); err == nil {
			if tn, err := Tan(a, prec); err == nil && !tn.Contains(x.Mid) {
				t.Fatalf("tan(atan(x)) does not contain x for x=%s", x)
			}
		}
		ch, err1 := Cosh(x, prec)
		sh, err2 := Sinh(x, prec)
		if err1 == nil && err2 == nil {
			d := ch.Mul(ch, prec).Sub(sh.Mul(sh, prec), prec)
			if !d.Contains(big.NewFloat(1)) {
				t.Fatalf("cosh²-sinh² does not contain 1 for x=%s", x)
			}
		}
	}
}

func TestDomainErrors(t *testing.T) {
	const prec = 120
	cases := []struct {
		name string
		run  func() (num.Ball, error)
		want error
	}{
		{"sqrt of negative", func() (num.Ball, error) { return Sqrt(pointBall(-1, prec), prec) }, ErrNegativeSqrt},
		{"sqrt of straddling", func() (num.Ball, error) {
			return Sqrt(num.NewBall(big.NewFloat(0.5), num.MagFromFloat(1)), prec)
		}, ErrNegativeSqrt},
		{"log of zero", func() (num.Ball, error) { return Log(pointBall(0, prec), prec) }, ErrNonPositiveLog},
		{"log of negative", func() (num.Ball, error) { return Log(pointBall(-2, prec), prec) }, ErrNonPositiveLog},
		{"log of straddling", func() (num.Ball, error) {
			return Log(num.NewBall(big.NewFloat(1), num.MagFromFloat(2)), prec)
		}, ErrNonPositiveLog},
		{"asin above one", func() (num.Ball, error) { return Asin(pointBall(1.5, prec), prec) }, ErrAsinDomain},
		{"asin straddling one", func() (num.Ball, error) {
			return Asin(num.NewBall(big.NewFloat(0.9), num.MagFromFloat(0.2)), prec)
		}, ErrAsinDomain},
		{"acos below minus one", func() (num.Ball, error) { return Acos(pointBall(-1.5, prec), prec) }, ErrAcosDomain},
		{"pow of negative base", func() (num.Ball, error) {
			return Pow(pointBall(-2, prec), pointBall(0.5, prec), prec)
		}, ErrNonPositivePow},
		{"atan2 at origin", func() (num.Ball, error) {
			return Atan2(pointBall(0, prec), pointBall(0, prec), prec)
		}, ErrOriginAtan2},
	}
	for _, c := range cases {
		_, err := c.run()
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestBoundaryValuesDoNotPanic(t *testing.T) {
	const prec = 120
	vals := []float64{0, 1, -1, 0.5, -0.5, 1e-30, -1e-30, 1e20, -1e20, 1e-300}
	for _, v := range vals {
		b := pointBall(v, prec)
		for _, fn := range unaryFuncs {
			if out, err := fn.f(b, prec); err == nil && out.Mid == nil {
				t.Fatalf("%s(%v) returned a ball without a midpoint", fn.name, v)
			}
		}
	}
	halfPi := new(big.Float).SetMantExp(Pi(prec+8).Mid, -1)
	for _, v := range []float64{-1, 1} {
		want := new(big.Float).Mul(big.NewFloat(v), halfPi)
		out, err := Asin(pointBall(v, prec), prec)
		if err != nil {
			t.Fatalf("asin(%v): %v", v, err)
		}
		if !out.Contains(want) {
			t.Fatalf("asin(%v) = %s, want %s", v, out, want.Text('g', 20))
		}
	}
	for _, v := range []float64{-1, 1} {
		if _, err := Acos(pointBall(v, prec), prec); err != nil {
			t.Fatalf("acos(%v): %v", v, err)
		}
	}
}

func TestPowIntegerFastPath(t *testing.T) {
	const prec = 120
	base := pointBall(3, prec)
	got, err := Pow(base, pointBall(4, prec), prec)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Contains(big.NewFloat(81)) {
		t.Fatalf("3^4 = %s", got)
	}
	neg, err := Pow(pointBall(2, prec), pointBall(-3, prec), prec)
	if err != nil {
		t.Fatal(err)
	}
	if !neg.Contains(big.NewFloat(0.125)) {
		t.Fatalf("2^-3 = %s", neg)
	}
	odd, err := Pow(pointBall(-2, prec), pointBall(3, prec), prec)
	if err != nil {
		t.Fatal(err)
	}
	if !odd.Contains(big.NewFloat(-8)) {
		t.Fatalf("(-2)^3 = %s", odd)
	}
}

func TestLargeTrigArgumentsWidenHonestly(t *testing.T) {
	const prec = 120
	big1 := new(big.Float).SetPrec(prec).SetFloat64(1e18)
	s, err := Sin(num.NewBall(big1, num.Mag{}), prec)
	if err != nil {
		t.Fatal(err)
	}
	if s.Rad.Cmp(num.MagFromFloat(1e-20)) > 0 {
		t.Fatalf("sin(1e18) radius too wide: %s", s)
	}
	c, err := Cos(num.NewBall(big1, num.Mag{}), prec)
	if err != nil {
		t.Fatal(err)
	}
	sum := s.Mul(s, prec).Add(c.Mul(c, prec), prec)
	if !sum.Contains(big.NewFloat(1)) {
		t.Fatalf("sin²+cos² at 1e18 = %s", sum)
	}
	wide := num.NewBall(big1, num.MagFromFloat(1e6))
	w, err := Sin(wide, prec)
	if err != nil {
		t.Fatal(err)
	}
	if w.Rad.Cmp(num.MagFromFloat(0.9)) < 0 {
		t.Fatalf("sin of a very wide ball should be wide, got %s", w)
	}
}

func TestPrecisionRefinementStaysConsistent(t *testing.T) {
	const low = 64
	const high = 300
	rnd := rand.New(rand.NewSource(20260924))
	for i := 0; i < 900; i++ {
		x := randBall(rnd, high)
		for _, fn := range unaryFuncs {
			in := x
			if fn.name == "asin" || fn.name == "acos" {
				in = shrinkIntoUnit(x)
			}
			coarse, err1 := fn.f(in, low)
			fine, err2 := fn.f(in, high)
			if err1 != nil || err2 != nil {
				continue
			}
			if !coarse.Contains(fine.Mid) {
				t.Fatalf("%s: coarse ball %s does not contain the refined value %s for x=%s",
					fn.name, coarse, fine.Mid.Text('g', 25), in)
			}
		}
	}
}

func TestReferenceValues(t *testing.T) {
	const prec = 200
	want := refs(t)
	b := func(v float64) num.Ball { return pointBall(v, prec) }
	must := func(ball num.Ball, err error) num.Ball {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return ball
	}
	cases := []struct {
		name string
		got  num.Ball
		key  string
	}{
		{"sin_1", must(Sin(b(1), prec)), "sin_1"},
		{"cos_1", must(Cos(b(1), prec)), "cos_1"},
		{"sin_2", must(Sin(b(2), prec)), "sin_2"},
		{"cos_2", must(Cos(b(2), prec)), "cos_2"},
		{"sin_3", must(Sin(b(3), prec)), "sin_3"},
		{"cos_3", must(Cos(b(3), prec)), "cos_3"},
		{"sin_4", must(Sin(b(4), prec)), "sin_4"},
		{"cos_4", must(Cos(b(4), prec)), "cos_4"},
		{"sin_5", must(Sin(b(5), prec)), "sin_5"},
		{"cos_5", must(Cos(b(5), prec)), "cos_5"},
		{"sin_7", must(Sin(b(7), prec)), "sin_7"},
		{"cos_7", must(Cos(b(7), prec)), "cos_7"},
		{"sin_-1", must(Sin(b(-1), prec)), "sin_-1"},
		{"cos_-1", must(Cos(b(-1), prec)), "cos_-1"},
		{"sin_-2", must(Sin(b(-2), prec)), "sin_-2"},
		{"cos_-2", must(Cos(b(-2), prec)), "cos_-2"},
		{"sin_100", must(Sin(b(100), prec)), "sin_100"},
		{"cos_100", must(Cos(b(100), prec)), "cos_100"},
		{"tan_1", must(Tan(b(1), prec)), "tan_1"},
		{"tan_-2", must(Tan(b(-2), prec)), "tan_-2"},
		{"atan_1", must(Atan(b(1), prec)), "atan_1"},
		{"atan_10", must(Atan(b(10), prec)), "atan_10"},
		{"atan_0.5", must(Atan(b(0.5), prec)), "atan_0.5"},
		{"atan_-0.5", must(Atan(b(-0.5), prec)), "atan_-0.5"},
		{"atan_-10", must(Atan(b(-10), prec)), "atan_-10"},
		{"atan_0.0009765625", must(Atan(b(0.0009765625), prec)), "atan_0.0009765625"},
		{"exp_1", must(Exp(b(1), prec)), "exp_1"},
		{"exp_0.5", must(Exp(b(0.5), prec)), "exp_0.5"},
		{"exp_-3", must(Exp(b(-3), prec)), "exp_-3"},
		{"exp_10", must(Exp(b(10), prec)), "exp_10"},
		{"exp_-10", must(Exp(b(-10), prec)), "exp_-10"},
		{"exp_0.0009765625", must(Exp(b(0.0009765625), prec)), "exp_0.0009765625"},
		{"log_2", must(Log(b(2), prec)), "log_2"},
		{"log2_2", must(Log2(b(2), prec)), "log2_2"},
		{"log10_2", must(Log10(b(2), prec)), "log10_2"},
		{"log_10", must(Log(b(10), prec)), "log_10"},
		{"log2_10", must(Log2(b(10), prec)), "log2_10"},
		{"log10_10", must(Log10(b(10), prec)), "log10_10"},
		{"log_0.5", must(Log(b(0.5), prec)), "log_0.5"},
		{"log2_0.5", must(Log2(b(0.5), prec)), "log2_0.5"},
		{"log10_0.5", must(Log10(b(0.5), prec)), "log10_0.5"},
		{"log_1000000", must(Log(b(1000000), prec)), "log_1000000"},
		{"log2_1000000", must(Log2(b(1000000), prec)), "log2_1000000"},
		{"log10_1000000", must(Log10(b(1000000), prec)), "log10_1000000"},
		{"log_0.00390625", must(Log(b(0.00390625), prec)), "log_0.00390625"},
		{"log2_0.00390625", must(Log2(b(0.00390625), prec)), "log2_0.00390625"},
		{"log10_0.00390625", must(Log10(b(0.00390625), prec)), "log10_0.00390625"},
		{"asin_0.5", must(Asin(b(0.5), prec)), "asin_0.5"},
		{"acos_0.5", must(Acos(b(0.5), prec)), "acos_0.5"},
		{"asin_-0.5", must(Asin(b(-0.5), prec)), "asin_-0.5"},
		{"acos_-0.5", must(Acos(b(-0.5), prec)), "acos_-0.5"},
		{"asin_0.875", must(Asin(b(0.875), prec)), "asin_0.875"},
		{"acos_0.875", must(Acos(b(0.875), prec)), "acos_0.875"},
		{"asin_-0.875", must(Asin(b(-0.875), prec)), "asin_-0.875"},
		{"acos_-0.875", must(Acos(b(-0.875), prec)), "acos_-0.875"},
		{"asin_0.96875", must(Asin(b(0.96875), prec)), "asin_0.96875"},
		{"acos_0.96875", must(Acos(b(0.96875), prec)), "acos_0.96875"},
		{"sinh_1", must(Sinh(b(1), prec)), "sinh_1"},
		{"cosh_1", must(Cosh(b(1), prec)), "cosh_1"},
		{"tanh_1", must(Tanh(b(1), prec)), "tanh_1"},
		{"sinh_-2", must(Sinh(b(-2), prec)), "sinh_-2"},
		{"cosh_-2", must(Cosh(b(-2), prec)), "cosh_-2"},
		{"tanh_-2", must(Tanh(b(-2), prec)), "tanh_-2"},
		{"sinh_3", must(Sinh(b(3), prec)), "sinh_3"},
		{"cosh_3", must(Cosh(b(3), prec)), "cosh_3"},
		{"tanh_3", must(Tanh(b(3), prec)), "tanh_3"},
		{"sinh_0.0009765625", must(Sinh(b(0.0009765625), prec)), "sinh_0.0009765625"},
		{"cosh_0.0009765625", must(Cosh(b(0.0009765625), prec)), "cosh_0.0009765625"},
		{"tanh_0.0009765625", must(Tanh(b(0.0009765625), prec)), "tanh_0.0009765625"},
		{"atan2_1_1", must(Atan2(b(1), b(1), prec)), "atan2_1_1"},
		{"atan2_1_-1", must(Atan2(b(1), b(-1), prec)), "atan2_1_-1"},
		{"atan2_-1_-1", must(Atan2(b(-1), b(-1), prec)), "atan2_-1_-1"},
		{"atan2_1_0", must(Atan2(b(1), b(0), prec)), "atan2_1_0"},
		{"atan2_-1_0", must(Atan2(b(-1), b(0), prec)), "atan2_-1_0"},
		{"atan2_-1_1", must(Atan2(b(-1), b(1), prec)), "atan2_-1_1"},
		{"atan2_3_-4", must(Atan2(b(3), b(-4), prec)), "atan2_3_-4"},
		{"pow_2_0.5", must(Pow(b(2), b(0.5), prec)), "pow_2_0.5"},
		{"pow_2_-0.5", must(Pow(b(2), b(-0.5), prec)), "pow_2_-0.5"},
		{"pow_10_0.25", must(Pow(b(10), b(0.25), prec)), "pow_10_0.25"},
		{"sqrt_2", must(Sqrt(b(2), prec)), "sqrt_2"},
		{"sqrt_0.5", must(Sqrt(b(0.5), prec)), "sqrt_0.5"},
		{"sqrt_1000000", must(Sqrt(b(1000000), prec)), "sqrt_1000000"},
	}
	for _, c := range cases {
		w, ok := want[c.key]
		if !ok {
			t.Fatalf("missing reference %s", c.key)
		}
		if !c.got.Contains(w) {
			t.Errorf("%s: ball %s does not contain %s", c.name, c.got, w.Text('g', 30))
		}
	}
}

package eval

import (
	"time"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/value"
)

type Ctx struct {
	Prec     uint
	MaxPrec  uint
	Digits   int
	Degrees  bool
	Deadline time.Time
}

func DefaultCtx() *Ctx {
	return &Ctx{Prec: 120, MaxPrec: 8192, Digits: 12}
}

type pass struct {
	ctx       *Ctx
	uncertain bool
}

func (p *pass) prec() uint { return p.ctx.Prec }

func (p *pass) note(v value.Value) value.Value {
	if !v.Num.IsExact() {
		p.uncertain = true
	}
	return v
}

func (e *Env) Eval(n parse.Node) (value.Value, error) { return e.EvalCtx(n, DefaultCtx()) }

func (e *Env) EvalCtx(n parse.Node, ctx *Ctx) (value.Value, error) {
	if ctx == nil {
		ctx = DefaultCtx()
	}
	if ctx.Prec == 0 {
		ctx.Prec = DefaultCtx().Prec
	}
	if ctx.MaxPrec < ctx.Prec {
		ctx.MaxPrec = ctx.Prec
	}
	for {
		p := &pass{ctx: ctx}
		v, err := e.eval(n, p)
		if err != nil {
			return value.Value{}, err
		}
		if v.Num.IsExact() {
			return v, nil
		}
		digits := v.Num.Ball(ctx.Prec).GuaranteedDigits()
		switch {
		case digits >= ctx.Digits:
			return v, nil
		case p.uncertain:
			v.Stopped = value.StopUncertainInput
			return v, nil
		case ctx.Prec >= ctx.MaxPrec:
			v.Stopped = value.StopPrecisionCap
			return v, nil
		case !ctx.Deadline.IsZero() && !time.Now().Before(ctx.Deadline):
			v.Stopped = value.StopDeadline
			return v, nil
		}
		if next := ctx.Prec * 2; next < ctx.MaxPrec {
			ctx.Prec = next
		} else {
			ctx.Prec = ctx.MaxPrec
		}
	}
}

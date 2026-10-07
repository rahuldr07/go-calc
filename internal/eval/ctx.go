package eval

import (
	"time"

	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/unc"
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
	naming    string
	assigned  bool
	notes     []parse.Note
	prov      map[parse.Node]prov.NodeID
	units     unitScope
	divisor   map[parse.Node]bool
}

func (p *pass) prec() uint { return p.ctx.Prec }

func (p *pass) note(v value.Value) value.Value {
	if !v.Num.IsExact() || v.IsMeasured() {
		p.uncertain = true
	}
	return v
}

func (e *Env) Eval(n parse.Node) (value.Value, error) { return e.EvalCtx(n, DefaultCtx()) }

func (e *Env) EvalCtx(n parse.Node, ctx *Ctx) (value.Value, error) {
	e.bound.clear()
	v, err := e.evalCtx(n, ctx)
	if err != nil {
		return value.Value{}, err
	}
	return e.report(v)
}

func (e *Env) report(v value.Value) (value.Value, error) {
	if v.IsList() {
		return e.reportList(v)
	}
	if !v.IsMeasured() {
		return v, nil
	}
	if err := e.table.Check(v.Sens); err != nil {
		return value.Value{}, err
	}
	if err := e.table.CheckCurv(v.Curv); err != nil {
		return value.Value{}, err
	}
	v.Unc.Sigma = e.table.Combined(v.Sens)
	v.Unc.Budget = e.table.Contributions(v.Sens)
	v.Unc.Degenerate = e.table.Degenerate(v.Sens, v.Curv)
	switch {
	case e.table.NonlinearityUnchecked(v.Sens, v.Curv):
		v.Unc.Warning = unc.UncheckedWarning
	case e.table.Nonlinear(v.Sens, v.Curv):
		v.Unc.Warning = unc.NonlinearWarning
	}
	return v, nil
}

func (e *Env) reportList(v value.Value) (value.Value, error) {
	items := make([]value.Value, len(v.List.Items))
	for i, item := range v.List.Items {
		reported, err := e.report(item)
		if err != nil {
			return value.Value{}, err
		}
		items[i] = reported
	}
	v.List = &value.Listing{Items: items}
	return v, nil
}

func (e *Env) evalCtx(n parse.Node, ctx *Ctx) (value.Value, error) {
	if ctx == nil {
		ctx = DefaultCtx()
	}
	if ctx.Prec == 0 {
		ctx.Prec = DefaultCtx().Prec
	}
	if ctx.MaxPrec < ctx.Prec {
		ctx.MaxPrec = ctx.Prec
	}
	mark := e.arena.Len()
	reclaim := true
	for {
		if reclaim {
			e.arena.Truncate(mark)
		}
		p := &pass{ctx: ctx, prov: map[parse.Node]prov.NodeID{}, divisor: map[parse.Node]bool{}}
		v, err := e.eval(n, p)
		reclaim = reclaim && !p.assigned
		if err == nil {
			p.announceRegroupings(n)
		}
		e.notes = distinct(p.notes)
		if err != nil {
			if isUndecided(err) && raisePrecision(ctx) {
				continue
			}
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
		raise(ctx)
	}
}

func raise(ctx *Ctx) {
	if next := ctx.Prec * 2; next < ctx.MaxPrec {
		ctx.Prec = next
	} else {
		ctx.Prec = ctx.MaxPrec
	}
}

func raisePrecision(ctx *Ctx) bool {
	if ctx.Prec >= ctx.MaxPrec || (!ctx.Deadline.IsZero() && !time.Now().Before(ctx.Deadline)) {
		return false
	}
	raise(ctx)
	return true
}

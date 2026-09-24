package eval

import (
	"strconv"

	"github.com/rahuldr07/go-calc/internal/consts"
	"github.com/rahuldr07/go-calc/internal/lex"
	"github.com/rahuldr07/go-calc/internal/parse"
	"github.com/rahuldr07/go-calc/internal/prov"
	"github.com/rahuldr07/go-calc/internal/units"
	"github.com/rahuldr07/go-calc/internal/value"
)

func (e *Env) eval(n parse.Node, p *pass) (value.Value, error) {
	v, err := e.evalNode(n, p)
	if err != nil {
		return value.Value{}, err
	}
	if beyondWritableWidth(v.Num.RoughWidth()) {
		return value.Value{}, &lex.Error{Pos: n.Pos(), Msg: TooLargeMessage}
	}
	v.Prov = e.stamp(n, p)
	return v, nil
}

func (p *pass) mark(n parse.Node, id prov.NodeID) {
	if p.prov == nil {
		p.prov = map[parse.Node]prov.NodeID{}
	}
	p.prov[n] = id
}

func (p *pass) idOf(n parse.Node) prov.NodeID { return p.prov[n] }

func (e *Env) record(n parse.Node, p *pass, node prov.Node) prov.NodeID {
	id := e.arena.Add(node)
	p.mark(n, id)
	return id
}

func (e *Env) stamp(n parse.Node, p *pass) prov.NodeID {
	if id, ok := p.prov[n]; ok {
		return id
	}
	if t, ok := n.(*parse.Assign); ok && len(t.Params) == 0 {
		if id, ok := p.prov[t.Body]; ok {
			p.mark(n, id)
			return id
		}
	}
	return e.record(n, p, describe(n, p))
}

func describe(n parse.Node, p *pass) prov.Node {
	switch t := n.(type) {
	case *parse.NumLit:
		return prov.Node{Kind: prov.Literal, Label: t.Text}
	case *parse.Name:
		return prov.Node{Kind: prov.Variable, Label: t.Text}
	case *parse.Unary:
		return prov.Node{Kind: prov.Operation, Label: t.Op, Children: []prov.NodeID{p.idOf(t.X)}}
	case *parse.Binary:
		return prov.Node{Kind: prov.Operation, Label: t.Op, Children: []prov.NodeID{p.idOf(t.L), p.idOf(t.R)}}
	case *parse.Call:
		kids := make([]prov.NodeID, 0, len(t.Args))
		for _, a := range t.Args {
			kids = append(kids, p.idOf(a))
		}
		return prov.Node{Kind: prov.Function, Label: t.Fn, Children: kids}
	case *parse.Assign:
		return prov.Node{Kind: prov.Variable, Label: t.Name, Detail: definitionDetail(t)}
	}
	return prov.Node{Kind: prov.Operation, Label: n.String()}
}

func definitionDetail(t *parse.Assign) string {
	if len(t.Params) == 0 {
		return ""
	}
	return t.String()
}

func (e *Env) namedProv(t *parse.Name, p *pass, held value.Value) prov.NodeID {
	inner := e.arena.Get(held.Prov)
	if e.arena.Has(held.Prov) && inner.Label == t.Text && (inner.Kind == prov.Input || inner.Kind == prov.Variable) {
		p.mark(t, held.Prov)
		return held.Prov
	}
	return e.record(t, p, prov.Node{Kind: prov.Variable, Label: t.Text, Children: []prov.NodeID{held.Prov}})
}

func constantDetail(c consts.Const) string {
	detail := c.Name
	if detail == "" {
		detail = c.Symbol
	}
	return detail + " · " + c.Source
}

func unitDetail(u units.Unit) string {
	base := units.SISymbol(u.Dim)
	detail := "1 " + u.Symbol + " = " + u.Factor.String()
	if base != "" {
		detail += " " + base
	}
	if u.Offset.Sign() != 0 {
		detail += " + " + u.Offset.String() + " " + base
	}
	return detail + " (" + u.Dim.Words() + ")"
}

func inputDetail(v value.Value, sigma float64, prec uint) string {
	scale := exactFloat(v.FactorOrOne())
	if scale == 0 {
		scale = 1
	}
	detail := shortFloat(numFloat(v.Num, prec)) + " ± " + shortFloat(sigma/scale)
	if v.Unit != "" {
		detail += " " + v.Unit
	}
	return detail
}

func shortFloat(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

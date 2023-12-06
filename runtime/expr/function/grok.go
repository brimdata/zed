package function

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/grok"
	"github.com/brimdata/zed/zcode"
)

type Grok struct {
	zctx    *zed.Context
	builder zcode.Builder
	hosts   map[string]*host
}

func newGrok(zctx *zed.Context) *Grok {
	return &Grok{
		zctx:  zctx,
		hosts: make(map[string]*host),
	}
}

func (g *Grok) Call(ectx zed.Allocator, vals []zed.Value) *zed.Value {
	patternArg, inputArg, defArg := vals[0], vals[1], zed.NullString
	if len(vals) == 3 {
		defArg = &vals[2]
	}
	switch {
	case zed.TypeUnder(defArg.Type) != zed.TypeString:
		return g.error(ectx, "definitions argument must be a string", defArg)
	case zed.TypeUnder(patternArg.Type) != zed.TypeString:
		return g.error(ectx, "pattern argument must be a string", &patternArg)
	case zed.TypeUnder(inputArg.Type) != zed.TypeString:
		return g.error(ectx, "input argument must be a string", &inputArg)
	}
	h, err := g.getHost(defArg.AsString())
	if err != nil {
		return g.error(ectx, err.Error(), defArg)
	}
	p, err := h.getPattern(g.zctx, patternArg.AsString())
	if err != nil {
		return g.error(ectx, err.Error(), &patternArg)
	}
	ss := p.ParseValues(inputArg.AsString())
	if ss == nil {
		return g.error(ectx, "value does not match pattern", &inputArg)
	}
	g.builder.Reset()
	for _, s := range ss {
		g.builder.Append([]byte(s))
	}
	return ectx.NewValue(p.typ, g.builder.Bytes())
}

func (g *Grok) error(ectx zed.Allocator, err string, val *zed.Value) *zed.Value {
	err = fmt.Sprintf("grok(): %s", err)
	if val == nil {
		return ectx.CopyValue(*g.zctx.NewErrorf(err))
	}
	return ectx.CopyValue(*g.zctx.WrapError(err, val))
}

func (g *Grok) getHost(defs string) (*host, error) {
	h, ok := g.hosts[defs]
	if !ok {
		h = &host{Host: grok.NewBase(), patterns: make(map[string]*pattern)}
		if err := h.AddFromReader(strings.NewReader(defs)); err != nil {
			return nil, err
		}
		g.hosts[defs] = h
	}
	return h, nil
}

type host struct {
	grok.Host
	patterns map[string]*pattern
}

func (h *host) getPattern(zctx *zed.Context, patternArg string) (*pattern, error) {
	p, ok := h.patterns[patternArg]
	if !ok {
		pat, err := h.Host.Compile(patternArg)
		if err != nil {
			return nil, err
		}
		var fields []zed.Field
		for _, name := range pat.Names() {
			fields = append(fields, zed.NewField(name, zed.TypeString))
		}
		typ, err := zctx.LookupTypeRecord(fields)
		if err != nil {
			return nil, err
		}
		p = &pattern{Pattern: pat, typ: typ}
		h.patterns[patternArg] = p
	}
	return p, nil
}

type pattern struct {
	*grok.Pattern
	typ zed.Type
}

package function

import (
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/grok"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zcode"
)

type Grok struct {
	zctx    *zed.Context
	builder zcode.Builder
	hosts   map[string]*host
	// fields is used as a scratch space to avoid allocating a new slice.
	fields []zed.Field
}

func newGrok(zctx *zed.Context) *Grok {
	return &Grok{
		zctx:  zctx,
		hosts: make(map[string]*host),
	}
}

func (g *Grok) Call(ectx expr.Context, args []zed.Value) zed.Value {
	arena := ectx.Arena()
	patternArg, inputArg, defArg := args[0], args[1], zed.NullString
	if len(args) == 3 {
		defArg = args[2]
	}
	switch {
	case zed.TypeUnder(defArg.Type()) != zed.TypeString:
		return g.error(arena, "definitions argument must be a string", defArg)
	case zed.TypeUnder(patternArg.Type()) != zed.TypeString:
		return g.error(arena, "pattern argument must be a string", patternArg)
	case zed.TypeUnder(inputArg.Type()) != zed.TypeString:
		return g.error(arena, "input argument must be a string", inputArg)
	}
	h, err := g.getHost(defArg.AsString())
	if err != nil {
		return g.error(arena, err.Error(), defArg)
	}
	p, err := h.getPattern(patternArg.AsString())
	if err != nil {
		return g.error(arena, err.Error(), patternArg)
	}
	keys, vals := p.ParseKeyValues(inputArg.AsString())
	if vals == nil {
		return g.error(arena, "value does not match pattern", inputArg)
	}
	g.fields = g.fields[:0]
	for _, key := range keys {
		g.fields = append(g.fields, zed.NewField(key, zed.TypeString))
	}
	typ := g.zctx.MustLookupTypeRecord(g.fields)
	g.builder.Reset()
	for _, s := range vals {
		g.builder.Append([]byte(s))
	}
	return arena.New(typ, g.builder.Bytes())
}

func (g *Grok) error(arena *zed.Arena, msg string, val zed.Value) zed.Value {
	return g.zctx.WrapError(arena, "grok(): "+msg, val)
}

func (g *Grok) getHost(defs string) (*host, error) {
	h, ok := g.hosts[defs]
	if !ok {
		h = &host{Host: grok.NewBase(), patterns: make(map[string]*grok.Pattern)}
		if err := h.AddFromReader(strings.NewReader(defs)); err != nil {
			return nil, err
		}
		g.hosts[defs] = h
	}
	return h, nil
}

type host struct {
	grok.Host
	patterns map[string]*grok.Pattern
}

func (h *host) getPattern(patternArg string) (*grok.Pattern, error) {
	p, ok := h.patterns[patternArg]
	if !ok {
		var err error
		p, err = h.Host.Compile(patternArg)
		if err != nil {
			return nil, err
		}
		h.patterns[patternArg] = p
	}
	return p, nil
}

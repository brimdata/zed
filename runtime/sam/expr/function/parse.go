package function

import (
	"net/url"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_uri
type ParseURI struct {
	zctx      *zed.Context
	marshaler *zson.MarshalZNGContext
}

func (p *ParseURI) Call(ectx expr.Context, args []zed.Value) zed.Value {
	in := args[0]
	if !in.IsString() || in.IsNull() {
		return p.zctx.WrapError(ectx.Arena(), "parse_uri: non-empty string arg required", in)
	}
	s := zed.DecodeString(in.Bytes())
	u, err := url.Parse(s)
	if err != nil {
		return p.zctx.WrapError(ectx.Arena(), "parse_uri: "+err.Error(), in)
	}
	var v struct {
		Scheme   *string    `zed:"scheme"`
		Opaque   *string    `zed:"opaque"`
		User     *string    `zed:"user"`
		Password *string    `zed:"password"`
		Host     *string    `zed:"host"`
		Port     *uint16    `zed:"port"`
		Path     *string    `zed:"path"`
		Query    url.Values `zed:"query"`
		Fragment *string    `zed:"fragment"`
	}
	if u.Scheme != "" {
		v.Scheme = &u.Scheme
	}
	if u.Opaque != "" {
		v.Opaque = &u.Opaque
	}
	if s := u.User.Username(); s != "" {
		v.User = &s
	}
	if s, ok := u.User.Password(); ok {
		v.Password = &s
	}
	if s := u.Hostname(); s != "" {
		v.Host = &s
	}
	if portString := u.Port(); portString != "" {
		u64, err := strconv.ParseUint(portString, 10, 16)
		if err != nil {
			return p.zctx.WrapError(ectx.Arena(), "parse_uri: invalid port: "+portString, in)
		}
		u16 := uint16(u64)
		v.Port = &u16
	}
	if u.Path != "" {
		v.Path = &u.Path
	}
	if q := u.Query(); len(q) > 0 {
		v.Query = q
	}
	if u.Fragment != "" {
		v.Fragment = &u.Fragment
	}
	out, err := p.marshaler.Marshal(ectx.Arena(), v)
	if err != nil {
		panic(err)
	}
	return out
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_zson
type ParseZSON struct {
	zctx *zed.Context
}

func newParseZSON(zctx *zed.Context) *ParseZSON {
	return &ParseZSON{zctx}
}

func (p *ParseZSON) Call(ectx expr.Context, args []zed.Value) zed.Value {
	in := args[0]
	if !in.IsString() {
		return p.zctx.WrapError(ectx.Arena(), "parse_zson: string arg required", in)
	}
	if in.IsNull() {
		return zed.Null
	}
	val, err := zson.ParseValue(p.zctx, ectx.Arena(), zed.DecodeString(in.Bytes()))
	if err != nil {
		return p.zctx.WrapError(ectx.Arena(), "parse_zson: "+err.Error(), in)
	}
	return val
}

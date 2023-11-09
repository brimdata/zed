package function

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_uri
type ParseURI struct {
	zctx      *zed.Context
	marshaler *zson.MarshalZNGContext
}

func (p *ParseURI) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	in := args[0]
	if !in.IsString() || in.IsNull() {
		return wrapError(p.zctx, ctx, "parse_uri: non-empty string arg required", &in)
	}
	s := zed.DecodeString(in.Bytes())
	u, err := url.Parse(s)
	if err != nil {
		return wrapError(p.zctx, ctx, "parse_uri: "+err.Error(), &in)
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
			return wrapError(p.zctx, ctx, "parse_uri: invalid port: "+portString, &in)
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
	out, err := p.marshaler.Marshal(v)
	if err != nil {
		panic(err)
	}
	return ctx.CopyValue(*out)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_zson
type ParseZSON struct {
	zctx *zed.Context
	sr   *strings.Reader
	zr   *zsonio.Reader
}

func newParseZSON(zctx *zed.Context) *ParseZSON {
	var sr strings.Reader
	return &ParseZSON{zctx, &sr, zsonio.NewReader(zctx, &sr)}
}

func (p *ParseZSON) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	in := args[0]
	if !in.IsString() {
		return wrapError(p.zctx, ctx, "parse_zson: string arg required", &in)
	}
	if in.IsNull() {
		return zed.Null
	}
	p.sr.Reset(zed.DecodeString(in.Bytes()))
	val, err := p.zr.Read()
	if err != nil {
		return wrapError(p.zctx, ctx, "parse_zson: "+err.Error(), &in)
	}
	if val == nil {
		return zed.Null
	}
	return ctx.CopyValue(*val)
}

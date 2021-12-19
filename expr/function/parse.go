package function

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_uri
type ParseURI struct {
	marshaler *zson.MarshalZNGContext
	stash     result.Value
}

func (p *ParseURI) Call(args []zed.Value) *zed.Value {
	in := args[0]
	if !in.IsStringy() {
		return p.stash.Error(errors.New("parse_uri: string arg required"))
	}
	if in.Bytes == nil {
		return zed.Null
	}
	s, err := zed.DecodeString(in.Bytes)
	if err != nil {
		panic(fmt.Errorf("parse_uri: corrupt Zed bytes: %w", err))
	}
	u, err := url.Parse(s)
	if err != nil {
		return p.stash.Error(fmt.Errorf("parse_uri: %w (%q)", err, s))
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
	if ss := u.Port(); ss != "" {
		u64, err := strconv.ParseUint(ss, 10, 16)
		if err != nil {
			return p.stash.Error(fmt.Errorf("parse_uri: %q: invalid port: %s", s, err))
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
		panic(fmt.Errorf("parse_uri: Zed marshaler failed: %w", err))
	}
	return p.stash.CopyVal(out)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#parse_zson
type ParseZSON struct {
	zctx  *zed.Context
	stash result.Value
}

func (p *ParseZSON) Call(args []zed.Value) *zed.Value {
	in := args[0]
	if !in.IsStringy() {
		return p.stash.Error(errors.New("parse_zson: string arg required"))
	}
	if in.Bytes == nil {
		return zed.Null
	}
	s, err := zed.DecodeString(in.Bytes)
	if err != nil {
		panic(fmt.Errorf("parse_zson: corrupt Zed bytes: %w", err))
	}
	parser := zson.NewParser(strings.NewReader(s))
	ast, err := parser.ParseValue()
	if err != nil {
		//XXX this will be better as a structured error
		return p.stash.Error(fmt.Errorf("parse_zson: parse error: %w (%q)", err, s))
	}
	if ast == nil {
		return zed.Null
	}
	val, err := zson.NewAnalyzer().ConvertValue(p.zctx, ast)
	if err != nil {
		return p.stash.Error(fmt.Errorf("parse_zson: semantic error: %w (%q)", err, s))
	}
	result, err := zson.Build(zcode.NewBuilder(), val)
	if err != nil {
		return p.stash.Error(fmt.Errorf("parse_zson: build error: %w (%q)", err, s))
	}
	return p.stash.CopyVal(result)
}

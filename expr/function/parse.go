package function

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type parseURI struct {
	marshaler *zson.MarshalZNGContext
}

func (p *parseURI) Call(args []zng.Value) (zng.Value, error) {
	in := args[0]
	if !in.IsStringy() {
		return badarg("parse_uri: input must be string")
	}
	if in.Bytes == nil {
		return badarg("parse_uri: input must not be null")
	}
	s, err := zng.DecodeString(in.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	u, err := url.Parse(s)
	if err != nil {
		return zng.Value{}, fmt.Errorf("parse_uri: %q: %w", s, errors.Unwrap(err))
	}
	var v struct {
		Scheme   *string    `zng:"scheme"`
		Opaque   *string    `zng:"opaque"`
		User     *string    `zng:"user"`
		Password *string    `zng:"password"`
		Host     *string    `zng:"host"`
		Port     *uint16    `zng:"port"`
		Path     *string    `zng:"path"`
		Query    url.Values `zng:"query"`
		Fragment *string    `zng:"fragment"`
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
			return zng.Value{}, fmt.Errorf("parse_uri: %q: invalid port: %s", s, errors.Unwrap(err))
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
	return p.marshaler.Marshal(v)
}

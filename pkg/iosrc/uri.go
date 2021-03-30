package iosrc

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type URI url.URL

const (
	Stdin  = "stdio:///stdin"
	Stdout = "stdio:///stdout"
	Stderr = "stdio:///stderr"
)

// uriRegexp is the regular expression used to determine if a path is treated
// as a URI. A path's prefix must be in the form of scheme://path. This deviates
// from the RFC for a URI's generic syntax which allows for scheme:path. There
// may be a valid relative file path that matches scheme:path. For our purposes
// we want to err on the side of reading a path as a file.
var uriRegexp = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9+-.]*://")

// ParseURI parses the path using `url.Parse`. If the provided uri does not
// contain a scheme, the scheme will set to file. Relative paths will be
// treated as files and resolved as absolute paths using filepath.Abs.
// path is an empty, Scheme is set to file.
func ParseURI(path string) (URI, error) {
	if path == "" {
		return URI{}, nil
	}
	if u, ok := stdio(path); ok {
		return u, nil
	}
	if uriRegexp.MatchString(path) {
		u, err := url.Parse(path)
		if err != nil {
			return URI{}, err
		}
		return URI(*u), nil
	}
	return parseBarePath(path)
}

func MustParseURI(path string) URI {
	u, err := ParseURI(path)
	if err != nil {
		panic(err)
	}
	return u
}

func stdio(path string) (URI, bool) {
	switch path {
	case "stdin":
		return URI{Scheme: "stdio", Path: "/stdin"}, true
	case "stdout":
		return URI{Scheme: "stdio", Path: "/stdout"}, true
	case "stderr":
		return URI{Scheme: "stdio", Path: "/stderr"}, true
	default:
		return URI{}, false
	}
}

func (p URI) AppendPath(elem ...string) URI {
	for _, el := range elem {
		p.Path = p.Path + "/" + el
	}
	return p
}

func (u URI) String() string {
	url := url.URL(u)
	return url.String()
}

func (u URI) RelPath(target URI) string {
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return strings.TrimPrefix(target.Path, u.Path)
}

func (u URI) IsZero() bool {
	return u == URI{}
}

func (u URI) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URI) UnmarshalText(b []byte) error {
	uri, err := ParseURI(string(b))
	if err != nil {
		return err
	}
	*u = uri
	return nil
}

func (u URI) MarshalZNG(mc *zson.MarshalZNGContext) (zng.Type, error) {
	return mc.MarshalValue(u.String())
}

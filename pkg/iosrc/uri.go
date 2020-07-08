package iosrc

import (
	"errors"
	"net/url"
	"strings"
)

type URI url.URL

const (
	stdin  = "stdio:///stdin"
	stdout = "stdio:///stdout"
	stderr = "stdio:///stderr"
)

// ParseURI parses the path using `url.Parse`. If the provided uri does not
// contain a scheme, the scheme will set to file. Relative paths will be
// treated as files and resolved as absolute paths using filepath.Abs.
// path is an empty, Scheme is set to file.
func ParseURI(path string) (URI, error) {
	// First resolve stdio keywords in to fully-formed uri.
	path = stdio(path)
	var err error
	path, err = normalizeFilepaths(path)
	if err != nil {
		return URI{}, err
	}
	u, err := url.Parse(path)
	if err != nil {
		return URI{}, err
	}
	return URI(*u), nil
}

func stdio(path string) string {
	switch path {
	case "stdin":
		return stdin
	case "stdout":
		return stdout
	case "stderr":
		return stderr
	default:
		return path
	}
}

func (p URI) AppendPath(elem ...string) URI {
	for _, el := range elem {
		p.Path = p.Path + "/" + el
	}
	return p
}

func (p URI) String() string {
	u := url.URL(p)
	return (&u).String()
}

func (u URI) RelPath(target URI) string {
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return strings.TrimPrefix(target.Path, u.Path)
}

// Maybe rawurl is of the form scheme:path.
// (Scheme must be [a-zA-Z][a-zA-Z0-9+-.]*)
// If so, return scheme, path; else return "", rawurl.
// Adapted from url package: https://golang.org/src/net/url/url.go?s=27728:27773#L973
func getscheme(rawurl string) (scheme string, err error) {
	for i := 0; i < len(rawurl); i++ {
		c := rawurl[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", nil
			}
		case c == ':':
			if i == 0 {
				return "", errors.New("missing protocol scheme")
			}
			return rawurl[:i], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", nil
		}
	}
	return "", nil
}

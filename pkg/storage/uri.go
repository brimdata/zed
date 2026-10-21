package storage

import (
	"net/url"
	"strings"

	"github.com/superdb/super"
)

type URI url.URL

var _ super.CustomMarshaler = (*URI)(nil)

// ParseURI parses the path using `url.Parse`. If the provided uri does not
// contain a scheme, the scheme is set to file. Relative paths are
// treated as files and resolved as absolute paths using filepath.Abs.
// If path is an empty, a pointer to zero-valued URI is returned.
func ParseURI(path string) (*URI, error) {
	if path == "" {
		return &URI{}, nil
	}
	if i := strings.IndexByte(path, ':'); i < 0 || !knownScheme(Scheme(path[:i])) {
		return parseBarePath(path)
	}
	u, err := url.Parse(path)
	return (*URI)(u), err
}

func MustParseURI(path string) *URI {
	u, err := ParseURI(path)
	if err != nil {
		panic(err)
	}
	return u
}

// String returns a local file's URI as a plain file system path (e.g.,
// /databases/test) and any other URI in URL form (e.g., s3://bucket/key).
// Since ParseURI treats a bare path as a file URI, the result always parses
// back to an equivalent URI.
func (u URI) String() string {
	if u.isLocalFile() {
		return u.Filepath()
	}
	return (*url.URL)(&u).String()
}

// isLocalFile reports whether u is a file URI that maps to a local path.
// File URIs with a host or an opaque part don't map to a local path so they
// keep their URL form.
func (u *URI) isLocalFile() bool {
	return Scheme(u.Scheme) == FileScheme && u.Host == "" && u.Opaque == "" && u.Path != ""
}

func (u *URI) HasScheme(s Scheme) bool {
	return Scheme(u.Scheme) == s
}

func (p *URI) JoinPath(elem ...string) *URI {
	return (*URI)((*url.URL)(p).JoinPath(elem...))
}

func (u *URI) RelPath(target URI) string {
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return strings.TrimPrefix(target.Path, u.Path)
}

func (u *URI) IsZero() bool {
	return *u == URI{}
}

func (u *URI) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URI) UnmarshalText(b []byte) error {
	uri, err := ParseURI(string(b))
	if err != nil {
		return err
	}
	*u = *uri
	return nil
}

func (u *URI) MarshalSuper(mc *super.Marshaler) (super.Type, error) {
	return mc.MarshalValue(u.String())
}

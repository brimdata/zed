package iosrc

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	uncPrefixRe = regexp.MustCompile("^(//|\\\\)")
	winVolumeRe = regexp.MustCompile("^[a-zA-Z]:")
)

func parseBarePath(path string) (URI, bool, error) {
	var host string
	if uncPrefixRe.MatchString(path) {
		return parseUNCPath(path)
	}
	if !winVolumeRe.MatchString(path) {
		if scheme, err := getscheme(path); err != nil || scheme != "" {
			return URI{}, false, err
		}
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return URI{}, false, err
	}
	// absolute file path for windows will start with a volume so preprepend
	// slash in front of path.
	return URI{
		Scheme: FileScheme,
		Path:   "/" + filepath.ToSlash(path),
		Host:   host,
	}, true, nil
}

// parseUNCPath parses a microsoft windows UNC path. This not a full
// implementation.
// See: https://en.wikipedia.org/wiki/Path_(computing)#POSIX_pathname_definition
func parseUNCPath(path string) (URI, bool, error) {
	// Trim first two slashes.
	path = path[2:]
	path = filepath.ToSlash(path)
	z := strings.SplitN(path, "/", 2)
	// if z is nil then we have just the host name
	if z == nil {
		return URI{Scheme: FileScheme, Host: path}, true, nil
	}
	u := URI{Scheme: FileScheme, Host: z[0], Path: z[1]}
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	return u, true, nil
}

func (p URI) Filepath() string {
	path := p.Path
	// Path should always be absolute and therefore we should always be able to
	// to strip the first '/', but for robustness check for windows volume.
	if path[0] == '/' && winVolumeRe.MatchString(path[1:]) {
		path = path[1:]
	}
	path = filepath.FromSlash(path)
	// If uri has a host, represent file as a UNC path.
	if p.Host != "" {
		path = `\\` + filepath.Join(p.Host, path)
	}
	return path
}

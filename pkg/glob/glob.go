// Package glob implements glob-style pattern matching
package glob

import (
	"regexp"

	"github.com/brimdata/zq/reglob"
)

func Glob(matches []string, pattern string, candidates []string) ([]string, error) {
	// transform a glob-style pattern into a regular expression
	pattern = reglob.Reglob(pattern)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if re.MatchString(candidate) {
			matches = append(matches, candidate)
		}
	}
	return matches, nil
}

func Globv(patterns, candidates []string) ([]string, error) {
	var matches []string
	for _, pattern := range patterns {
		var err error
		matches, err = Glob(matches, pattern, candidates)
		if err != nil {
			return nil, err
		}
	}
	return uniq(matches), nil
}

func uniq(v []string) []string {
	m := make(map[string]struct{})
	for _, s := range v {
		m[s] = struct{}{}
	}
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	return out
}

package cmd

import (
	"errors"
	"fmt"
	"sort"

	"github.com/brimsec/zq/pkg/glob"
)

var ErrNoMatch = errors.New("no match")

func SpaceGlob(api *API, patterns []string) ([]string, error) {
	spaces, err := api.SpaceList()
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch spaces: %s", err)
	}
	if len(spaces) == 0 {
		return nil, errors.New("no spaces exist")
	}
	var matches []string
	if len(patterns) == 0 {
		matches = spaces
	} else {
		matches, err = glob.Globv(patterns, spaces)
		if err != nil {
			return nil, err
		}
		if matches == nil || len(matches) == 0 {
			return nil, ErrNoMatch
		}
	}
	sort.Strings(matches)
	return matches, nil
}

package lakeparse

import (
	"errors"
	"fmt"
	"strings"
)

type Commitish struct {
	Pool   string
	Branch string
}

func ParseCommitish(commitish string) (*Commitish, error) {
	if strings.IndexByte(commitish, '\'') >= 0 {
		return nil, errors.New("pool and branch names may not contain single quote characters")
	}
	if i := strings.LastIndexByte(commitish, '@'); i > -1 {
		return &Commitish{Pool: commitish[:i], Branch: commitish[i+1:]}, nil
	}
	return &Commitish{Branch: commitish}, nil
}

var ErrNoPool = errors.New("no pool")

func (c *Commitish) FromSpec(meta string) (string, error) {
	if c.Pool == "" {
		return "", ErrNoPool
	}
	var s string
	if _, err := ParseID(c.Branch); err == nil {
		s = fmt.Sprintf("from '%s'@%s", c.Pool, c.Branch)
	} else {
		s = fmt.Sprintf("from '%s'@'%s'", c.Pool, c.Branch)
	}
	if meta != "" {
		s += ":" + meta
	}
	return s, nil
}

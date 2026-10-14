package dbid

import (
	"errors"
	"fmt"
	"strings"
)

type Committish struct {
	Table  string `super:"table"`
	Branch string `super:"branch"`
}

func ParseCommittish(committish string) (*Committish, error) {
	if committish == "" {
		return nil, errors.New("empty table and branch")
	}
	if strings.IndexByte(committish, '\'') >= 0 {
		return nil, errors.New("table and branch names may not contain single quote characters")
	}
	if i := strings.LastIndexByte(committish, '@'); i > -1 {
		return &Committish{Table: committish[:i], Branch: committish[i+1:]}, nil
	}
	return &Committish{Table: committish}, nil
}

var ErrNoTable = errors.New("no table")

func (c *Committish) FromSpec(meta string) (string, error) {
	if c.Table == "" {
		return "", ErrNoTable
	}
	var s string
	if _, err := ParseID(c.Branch); err == nil {
		s = fmt.Sprintf("from '%s'@%s", c.Table, c.Branch)
	} else {
		s = fmt.Sprintf("from '%s'@'%s'", c.Table, c.Branch)
	}
	if meta != "" {
		s += ":" + meta
	}
	return s, nil
}

func (c *Committish) String() string {
	return fmt.Sprintf("%s@%s", c.Table, c.Branch)
}

func (c *Committish) IsZero() bool {
	return c.Table == "" && c.Branch == ""
}

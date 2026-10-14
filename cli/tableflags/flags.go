package tableflags

import (
	"errors"
	"flag"

	"github.com/superdb/super/dbid"
)

type Flags struct {
	defaultHead string
}

func (l *Flags) SetFlags(fs *flag.FlagSet) {
	defaultHead, _ := readHead()
	fs.StringVar(&l.defaultHead, "use", defaultHead, "commit to use, i.e., table, table@branch, or table@commit")
}

func (f *Flags) HEAD() (*dbid.Committish, error) {
	if f.defaultHead == "" {
		return nil, errors.New(`table and branch are unspecified
(specify with -use flag or "super db use" command)`)
	}
	c, err := dbid.ParseCommittish(f.defaultHead)
	if err != nil {
		return nil, err
	}
	if c.Table == "" {
		return nil, errors.New("table unspecified")
	}
	if c.Branch == "" {
		c.Branch = "main"
	}
	return c, nil
}

package lakeflags

import (
	"flag"
	"strings"

	"github.com/brimdata/zed/compiler/parser"
)

type Flags struct {
	Quiet    bool
	PoolName string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.BoolVar(&f.Quiet, "q", false, "quiet mode")
	fs.StringVar(&f.PoolName, "p", "", "name of pool")
}

func (f *Flags) Names() (string, string) {
	list := strings.Split(f.PoolName, "/")
	if len(list) == 1 {
		return list[0], ""
	}
	last := len(list) - 1
	return strings.Join(list[0:last], "/"), list[last]
}

func (f *Flags) Branch() (string, string) {
	pool, branch := f.Names()
	if branch == "" {
		branch = "main"
	}
	return cleanse(pool), cleanse(branch)
}

// cleanse normalizes 0x bytes ksuids into a base62 string
func cleanse(s string) string {
	id, err := parser.ParseID(s)
	if err == nil {
		return id.String()
	}
	return s
}

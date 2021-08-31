package lakeflags

import (
	"errors"
	"flag"
	"fmt"
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
	if i := strings.LastIndexByte(f.PoolName, '@'); i > -1 {
		return f.PoolName[:i], f.PoolName[i+1:]
	}
	return f.PoolName, ""
}

func (f *Flags) Branch() (string, string) {
	pool, branch := f.Names()
	if branch == "" {
		branch = "main"
	}
	return cleanse(pool), cleanse(branch)
}

func (f *Flags) FromSpec(meta string) (string, error) {
	poolName, branchName := f.Branch()
	if strings.IndexByte(poolName, '\'') >= 0 || strings.IndexByte(branchName, '\'') >= 0 {
		return "", errors.New("pool name may not contain quote characters")
	}
	var s string
	if _, err := parser.ParseID(branchName); err == nil {
		s = fmt.Sprintf("'%s'@%s", poolName, branchName)
	} else {
		s = fmt.Sprintf("from '%s'@'%s'", poolName, branchName)
	}
	if meta != "" {
		s += ":" + meta
	}
	return s, nil
}

// cleanse normalizes 0x bytes ksuids into a base62 string
func cleanse(s string) string {
	id, err := parser.ParseID(s)
	if err == nil {
		return id.String()
	}
	return s
}

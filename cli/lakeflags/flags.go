package lakeflags

import (
	"errors"
	"flag"

	"github.com/brimdata/zed/lakeparse"
)

var ErrNoHEAD = errors.New("HEAD not specified: indicate with -use or run the \"use\" command")

type Flags struct {
	Quiet       bool
	defaultHead string
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	defaultHead, _ := readHead()
	fs.BoolVar(&f.Quiet, "q", false, "quiet mode")
	fs.StringVar(&f.defaultHead, "use", defaultHead, "commit to use, i.e., pool@branch or pool@commit")
}

func (f *Flags) HEAD() (*lakeparse.Commitish, error) {
	return lakeparse.ParseCommitish(f.defaultHead)
}

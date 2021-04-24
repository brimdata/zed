package zbuf

import (
	"io"

	"github.com/brimdata/zed/zio"
)

type File struct {
	zio.Reader
	c    io.Closer
	name string
}

func NewFile(r zio.Reader, c io.Closer, name string) *File {
	return &File{r, c, name}
}

func (r *File) Close() error {
	return r.c.Close()
}

func (r *File) String() string {
	return r.name
}

package zbuf

import (
	"io"
)

type File struct {
	Reader
	c    io.Closer
	name string
}

func NewFile(r Reader, c io.Closer, name string) *File {
	return &File{r, c, name}
}

func (r *File) Close() error {
	return r.c.Close()
}

func (r *File) String() string {
	return r.name
}

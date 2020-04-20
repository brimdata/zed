package zbuf

import (
	"os"
)

type File struct {
	Reader
	file *os.File
}

func NewFile(r Reader, f *os.File) *File {
	return &File{r, f}
}

func (r *File) Close() error {
	return r.file.Close()
}

func (r *File) String() string {
	return r.file.Name()
}

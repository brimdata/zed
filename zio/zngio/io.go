package zngio

import (
	"io"

	"github.com/brimsec/zq/zbuf"
)

type ioreader struct {
	reader io.Reader
	writer *io.PipeWriter
}

func IOReader(r zbuf.Reader, opts WriterOpts) io.ReadCloser {
	pr, pw := io.Pipe()
	i := &ioreader{reader: pr, writer: pw}
	go i.run(r, NewWriter(pw, opts))
	return i
}

func (i *ioreader) run(r zbuf.Reader, w zbuf.Writer) {
	err := zbuf.Copy(w, r)
	if err != nil {
		i.writer.CloseWithError(err)
	}
	i.writer.Close()
}

func (i *ioreader) Read(b []byte) (int, error) {
	return i.reader.Read(b)
}

func (i *ioreader) Close() error {
	return i.writer.Close()
}

package zson

import (
	"io"

	"github.com/mccanne/zq/pkg/nano"
)

type Reader interface {
	Read() (*Record, error)
}

type WriteCloser interface {
	Write(*Record) error
	Close() error
}

// Writer is a simple, embeddable object for zson writers who don't need
// to do anything special at close.  Instead this object will just call the
// Close method on the wrapped io.WriteCloser.
type Writer struct {
	io.WriteCloser
}

func (w *Writer) Close() error {
	return w.WriteCloser.Close()
}

/* XXX
type BatchReader interface {
	Read() (Batch, error)
}


type BatchWriter interface {
	Write(Batch) error
}
*/

// Batch is an inteface to a bundle of Records.
// Batches can be shared across goroutines via reference counting and should be
// copied on modification when the reference count is greater than 1.
type Batch interface {
	Ref()
	Unref()
	Index(int) *Record
	Length() int
	Records() []*Record
	//XXX span should go in here?
	Span() nano.Span
}

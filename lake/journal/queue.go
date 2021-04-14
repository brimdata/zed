package journal

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/iosrc"
)

const ext = "zng"

var (
	ErrEmpty  = errors.New("empty log")
	ErrFailed = errors.New("transaction failed")
)

type ID uint64

const Nil ID = 0

type Queue struct {
	path     iosrc.URI
	headPath iosrc.URI
	tailPath iosrc.URI
}

func New(path iosrc.URI) *Queue {
	return &Queue{
		path:     path,
		headPath: path.AppendPath("HEAD"),
		tailPath: path.AppendPath("TAIL"),
	}
}

func (q *Queue) ReadHead(ctx context.Context) (ID, error) {
	//XXX The head file can be wrong due to races but it will always be
	// close so we should probe for the next slot(s) and update the HEAD
	// object if we find a hit.  See issue #XXX.
	return readID(ctx, q.headPath)
}

func (q *Queue) writeHead(ctx context.Context, id ID) error {
	return writeID(ctx, q.headPath, id)
}

func (q *Queue) ReadTail(ctx context.Context) (ID, error) {
	return readID(ctx, q.tailPath)
}

func (q *Queue) writeTail(ctx context.Context, id ID) error {
	return writeID(ctx, q.tailPath, id)
}

func (q *Queue) Boundaries(ctx context.Context) (ID, ID, error) {
	head, err := q.ReadHead(ctx)
	if err != nil {
		return Nil, Nil, err
	}
	tail, err := q.ReadTail(ctx)
	if err != nil {
		return Nil, Nil, err
	}
	return head, tail, nil
}

//XXX This needs concurrency work. See issue #2546.
func (q *Queue) Commit(ctx context.Context, b []byte) error {
	head, err := q.ReadHead(ctx)
	if err != nil {
		return err
	}
	uri := q.uri(head + 1)
	if err := iosrc.WriteFile(ctx, uri, b); err != nil {
		return err
	}
	if head == 0 {
		if err := q.writeTail(ctx, 1); err != nil {
			return nil
		}
	}
	return q.writeHead(ctx, head+1)
}

// NewReader returns a zngio.Reader that concatenates the journal files
// in sequence from tail to head.  Since ZNG is stored in the journal,
// this produce a byte stream suitable for wrapper in a zngio.Reader.
func (q *Queue) NewReader(ctx context.Context, head, tail ID) *Reader {
	return newReader(ctx, q, head, tail)
}

func (q *Queue) uri(id ID) iosrc.URI {
	return q.path.AppendPath(fmt.Sprintf("%d.%s", id, ext))
}

func (q *Queue) Load(ctx context.Context, id ID) ([]byte, error) {
	return iosrc.ReadFile(ctx, q.uri(id))
}

func writeID(ctx context.Context, path iosrc.URI, id ID) error {
	return iosrc.WriteFile(ctx, path, []byte(strconv.FormatUint(uint64(id), 10)))
}

func readID(ctx context.Context, path iosrc.URI) (ID, error) {
	b, err := iosrc.ReadFile(ctx, path)
	if err != nil {
		return Nil, err
	}
	id, err := byteconv.ParseUint64(b)
	if err != nil {
		return Nil, err
	}
	return ID(id), nil
}

func Create(ctx context.Context, path iosrc.URI) (*Queue, error) {
	if err := iosrc.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	q := New(path)
	if err := q.writeHead(ctx, Nil); err != nil {
		return nil, err
	}
	if err := q.writeTail(ctx, Nil); err != nil {
		return nil, err
	}
	return q, nil
}

func Open(ctx context.Context, path iosrc.URI) (*Queue, error) {
	q := New(path)
	if _, err := q.ReadHead(ctx); err != nil {
		return nil, fmt.Errorf("%s: no such journal", path)
	}
	return q, nil
}

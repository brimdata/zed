package journal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

const ext = "zng"

var (
	ErrEmpty  = errors.New("empty log")
	ErrFailed = errors.New("transaction failed")
)

type ID uint64

const Nil ID = 0

const MaxReadRetry = 10

type Queue struct {
	engine   storage.Engine
	path     *storage.URI
	headPath *storage.URI
	tailPath *storage.URI
}

func New(engine storage.Engine, path *storage.URI) *Queue {
	return &Queue{
		engine:   engine,
		path:     path,
		headPath: path.AppendPath("HEAD"),
		tailPath: path.AppendPath("TAIL"),
	}
}

func (q *Queue) Path() *storage.URI {
	return q.path
}

func (q *Queue) ReadHead(ctx context.Context) (ID, error) {
	//XXX The head file can be wrong due to races but it will always be
	// close so we should probe for the next slot(s) and update the HEAD
	// object if we find a hit.  See issue #XXX.
	return readID(ctx, q.engine, q.headPath)
}

func (q *Queue) writeHead(ctx context.Context, id ID) error {
	return writeID(ctx, q.engine, q.headPath, id)
}

func (q *Queue) ReadTail(ctx context.Context) (ID, error) {
	return readID(ctx, q.engine, q.tailPath)
}

func (q *Queue) writeTail(ctx context.Context, id ID) error {
	return writeID(ctx, q.engine, q.tailPath, id)
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
	return q.CommitAt(ctx, head, b)
}

func (q *Queue) CommitAt(ctx context.Context, at ID, b []byte) error {
	uri := q.uri(at + 1)
	if err := q.engine.PutIfNotExists(ctx, uri, b); err != nil {
		if err != storage.ErrNotSupported {
			return err
		}
		//XXX Here, we need to emulate PutIfNotExists using S3's
		// strong ordering guarantees.  Currently, this is incorrect
		// and can race with multiple writers.  See issue #2686.
		w, err := q.engine.Put(ctx, uri)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, bytes.NewReader(b))
		if err != nil {
			w.Close()
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}
	if at == 0 {
		if err := q.writeTail(ctx, 1); err != nil {
			return nil
		}
	}
	return q.writeHead(ctx, at+1)
}

// NewReader returns a zngio.Reader that concatenates the journal files
// in sequence from tail to head.  Since ZNG is stored in the journal,
// this produce a byte stream suitable for wrapper in a zngio.Reader.
func (q *Queue) NewReader(ctx context.Context, head, tail ID) *Reader {
	return newReader(ctx, q, head, tail)
}

func (q *Queue) uri(id ID) *storage.URI {
	return q.path.AppendPath(fmt.Sprintf("%d.%s", id, ext))
}

func (q *Queue) Load(ctx context.Context, id ID) ([]byte, error) {
	return storage.Get(ctx, q.engine, q.uri(id))
}

func (q *Queue) Open(ctx context.Context, head, tail ID) (io.Reader, error) {
	if head == Nil {
		var err error
		head, err = q.ReadHead(ctx)
		if err != nil {
			return nil, err
		}
		if head == Nil {
			return nil, ErrEmpty
		}
	}
	if tail == Nil {
		var err error
		tail, err = q.ReadTail(ctx)
		if err != nil {
			return nil, err
		}
	}
	return q.NewReader(ctx, head, tail), nil
}

func (q *Queue) OpenAsZNG(ctx context.Context, head, tail ID) (*zngio.Reader, error) {
	r, err := q.Open(ctx, head, tail)
	if err != nil {
		return nil, err
	}
	return zngio.NewReader(r, zson.NewContext()), nil
}

func writeID(ctx context.Context, engine storage.Engine, u *storage.URI, id ID) error {
	r := strings.NewReader(strconv.FormatUint(uint64(id), 10))
	return storage.Put(ctx, engine, u, r)
}

func readID(ctx context.Context, engine storage.Engine, path *storage.URI) (ID, error) {
	var retry int
	timeout := time.Millisecond
	for {
		b, err := storage.Get(ctx, engine, path)
		if err != nil {
			return Nil, err
		}
		if id, err := byteconv.ParseUint64(b); err == nil {
			return ID(id), nil
		}
		retry++
		if retry > MaxReadRetry || timeout > 5*time.Second {
			return Nil, fmt.Errorf("can read but not parse contents of journal HEAD: %s", b)
		}
		select {
		case <-time.After(timeout):
		case <-ctx.Done():
			return Nil, ctx.Err()
		}
		t := 2 * int(timeout)
		timeout = time.Duration(t + rand.Intn(t))
	}
}

func Create(ctx context.Context, engine storage.Engine, path *storage.URI) (*Queue, error) {
	q := New(engine, path)
	if err := q.writeHead(ctx, Nil); err != nil {
		return nil, err
	}
	if err := q.writeTail(ctx, Nil); err != nil {
		return nil, err
	}
	return q, nil
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI) (*Queue, error) {
	q := New(engine, path)
	if _, err := q.ReadHead(ctx); err != nil {
		return nil, fmt.Errorf("%s: no such journal", path)
	}
	return q, nil
}

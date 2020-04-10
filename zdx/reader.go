package zdx

import (
	"io"
	"os"

	"github.com/mccanne/zq/pkg/zng"
)

type reader struct {
	filename string
	file     *os.File
}

// Reader reads a zdx file and implements the zbuf.Reader interface.
type Reader struct {
	reader
	in []byte
}

func (r *reader) init(path string, level int) {
	r.filename = filename(path, level)
}

// NewReader returns a Reader ready to read zdx file.
// Close() should be called when done.
func NewReader(path string) *Reader {
	r := &Reader{}
	r.init(path, 0)
	return r
}

func (r *reader) Open() error {
	var err error
	r.file, err = os.Open(r.filename)
	return err
}

func (r *reader) Close() error {
	err := r.file.Close()
	r.file = nil
	return err
}

func (r *Reader) Read() (zng.Record, error) {
	if len(r.in) == 0 {
		if err := r.readFrame(); err != nil {
			if err == io.EOF {
				return zng.Record{}, nil
			}
			return zng.Record{}, err
		}
	}
	key, err := r.decode()
	if err != nil {
		return Pair{}, err
	}
	value, err := r.decode()
	if err != nil {
		return Pair{}, err
	}
	// this key and value point into the frame buffer so the caller
	// needs to copy them before the next call to read
	// XXX for a merge we don't need to convert to a string
	return Pair{key, value}, nil
}

func (r *reader) grow(target int) {
	size := cap(r.frame)
	for size < target {
		size *= 2
	}
	r.frame = make([]byte, 0, target)
}

type FrameReader struct {
	reader
}

func NewFrameReader(path string, level int) *FrameReader {
	r := &FrameReader{}
	r.init(path, level)
	return r
}

func (r *FrameReader) ReadFrameAt(off int64) ([]byte, error) {
	var hdr [FrameHeaderLen]byte
	n, err := r.file.ReadAt(hdr[:], off)
	if err != nil {
		return nil, err
	}
	if n != FrameHeaderLen {
		return nil, ErrCorruptFile
	}
	framelen := decodeInt(hdr[1:5])
	//XXX
	if framelen > 10*1024*1024 {
		return nil, ErrCorruptFile
	}
	r.grow(framelen)
	n, err = r.file.ReadAt(r.frame[0:framelen], off+FrameHeaderLen)
	if err != nil {
		return nil, err
	}
	if n != framelen {
		return nil, ErrCorruptFile
	}
	return r.frame[:framelen], nil
}

func (r *FrameReader) ReadFrame() ([]byte, error) {
	var hdr [FrameHeaderLen]byte
	n, err := r.file.Read(hdr[:])
	if err != nil {
		return nil, err
	}
	if n != FrameHeaderLen {
		return nil, ErrCorruptFile
	}
	framelen := decodeInt(hdr[1:5])
	//XXX
	if framelen > 10*1024*1024 {
		return nil, ErrCorruptFile
	}
	r.grow(framelen)
	n, err = r.file.Read(r.frame[0:framelen])
	if err != nil {
		return nil, err
	}
	if n != framelen {
		return nil, ErrCorruptFile
	}
	return r.frame[:framelen], nil
}

type IndexReader struct {
	Reader
}

func NewIndexReader(path string, level int) *IndexReader {
	r := &IndexReader{}
	r.init(path, level)
	return r
}

func (r *IndexReader) Read() ([]byte, int64, error) {
	if len(r.in) == 0 {
		if err := r.readFrame(); err != nil {
			if err == io.EOF {
				return nil, 0, nil
			}
			return nil, 0, err
		}
	}
	key, off, n := DecodeIndex(r.in)
	if off < 0 {
		return nil, 0, ErrCorruptFile
	}
	r.in = r.in[n:]
	// this key and value point into the frame buffer so the caller
	// needs to copy them before the next call to read
	return key, off, nil
}

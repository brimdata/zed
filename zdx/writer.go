package zdx

import (
	"os"
)

// Writer creates an SST file and related index files.
// An SST file is a sequence of key,value pairs sorted lexicographically
// by key.  On disk, the key is a counted string and the value is a counted
// byte array (unless all the value sizes are the same, then the length is
// encoded in the file header).
// After the file is written, the index files comprise a
// constant B-Tree to make key lookups efficient.  The index files
// are written in the same fashion as the base SST file since an index
// is simply an SST where the key is the key and the value is the offset
// into the child file where that key is located.  The framesize parameter
// indicates how many bytes to consume for before terminating a frame and
// thus causing an index key to be written to the parent index file.
// This process is repeated until the topmost parent file
// has a sequence of key/offset pairs that fit in framesize bytes.
type Writer struct {
	path      string
	framesize int // XXX we might want different sizes for base and index
	level     int
	file      *os.File
	parent    *Writer
	frame     []byte
	frameKey  []byte
	offset    int64
	valsize   int
	Stats     Stats
}

type Stats struct {
	Count      int64
	KeyBytes   int64
	ValueBytes int64
}

// NewWriter returns a Writer ready to write an SST file and related
// index files via subsequent calls to Write(), or it returns an error.
// All files will be written to the directory indicated by path
// with the form $path, $path.1, and so forth.  Calls to Write must
// provide keys in increasing lexicographic order.  Duplicate keys are not
// allowed but will not be detected.  Close() must be called when done writing.
func NewWriter(path string, framesize, valSize int) (*Writer, error) {
	return newWriter(path, framesize, valSize, 0)
}

func newWriter(path string, framesize, valsize, level int) (*Writer, error) {
	if level > 5 {
		panic("something wrong")
	}
	name := filename(path, level)
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}
	w := &Writer{
		path:      path,
		framesize: framesize,
		level:     level,
		file:      f,
		// avoid realloc with 2*framesize
		frame:   make([]byte, 0, 2*framesize),
		valsize: valsize,
	}
	if err := w.writeFileHeader(valsize, framesize); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) writeFileHeader(valsize, framesize int) error {
	var hdr [FileHeaderLen]byte
	encodeInt(hdr[0:4], magic)
	hdr[4] = versionMajor
	hdr[5] = versionMinor
	encodeInt(hdr[6:10], valsize)
	encodeInt(hdr[10:14], framesize)
	_, err := w.file.Write(hdr[:])
	w.offset = FileHeaderLen
	return err
}

func (w *Writer) Close() error {
	if len(w.frame) > 0 {
		// Make sure to pass up framekeys to parent trees, even though frames aren't
		// full.
		var err error
		if w.parent != nil {
			err = w.writeFrame()
		} else {
			err = w.flushFrame()
		}
		if err != nil {
			return err
		}
	}
	if err := w.file.Close(); err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.Close()
	}
	return nil
}

func (w *Writer) grow(target int) {
	off := len(w.frame)
	size := cap(w.frame)
	for size < target {
		size *= 2
	}
	p := make([]byte, off, size)
	copy(p, w.frame)
	w.frame = p
}

func (w *Writer) encode(v []byte) {
	off := len(w.frame)
	n := len(v)
	if off+n > cap(w.frame) {
		//XXX this should be very rare and happens only
		// when keys and/or values are very large
		w.grow(off + n)
	}
	copy(w.frame[off:off+n], v)
	w.frame = w.frame[:off+n]
}

func (w *Writer) encodeCounted(v []byte) {
	var cnt [4]byte
	encodeInt(cnt[:], len(v))
	w.encode(cnt[:])
	w.encode(v)
}

func (w *Writer) Write(pair Pair) error {
	if len(pair.Key)+len(pair.Value)+8 > w.framesize && w.level > 0 {
		// encoding a single pair in a frame larger than
		// the framesize will cause an infinite loop writing
		// an index files that hold just the one pair
		return ErrPairTooBig
	}
	w.Stats.Count++
	w.Stats.KeyBytes += int64(len(pair.Key))
	w.Stats.ValueBytes += int64(len(pair.Value))
	if len(w.frame) >= w.framesize {
		// the frame in place is already big enough... flush it and
		// start going on the next
		if err := w.writeFrame(); err != nil {
			return err
		}
	}
	if len(w.frame) == 0 {
		w.frameKey = append(w.frameKey[:0], pair.Key...)
	}
	//XXX should use unsafe pointer
	w.encodeCounted([]byte(pair.Key))
	if w.valsize != 0 {
		if w.valsize != len(pair.Value) {
			return ErrValueSize
		}
		w.encode(pair.Value)
	} else {
		w.encodeCounted(pair.Value)
	}
	return nil
}

func (w *Writer) flushFrame() error {
	var hdr [FrameHeaderLen]byte
	hdr[0] = 0 // compression type XXX
	//XXX need to compress frame here
	encodeInt(hdr[1:5], len(w.frame))
	if _, err := w.file.Write(hdr[:]); err != nil {
		return err
	}
	if _, err := w.file.Write(w.frame); err != nil {
		return err
	}
	w.offset += int64(len(hdr) + len(w.frame))
	w.frame = w.frame[:0]
	return nil
}

func (w *Writer) writeFrame() error {
	if err := w.writeIndex(w.frameKey, w.offset); err != nil {
		return err
	}
	return w.flushFrame()
}

func (w *Writer) writeIndex(key []byte, offset int64) error {
	if w.parent == nil {
		var err error
		w.parent, err = newWriter(w.path, w.framesize, 6, w.level+1)
		if err != nil {
			return err
		}
	}
	var val [6]byte
	encodeInt48(val[:], offset)
	return w.parent.Write(Pair{key, val[:]})
}

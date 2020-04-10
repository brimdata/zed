// XXX This package cmment will get re-written in a subsequent PR.
//
// Package sst provides an API for creating, merging, indexing, and querying sorted string
// tables (SST), a la LSM trees, where an SST holds a sequence of key,value pairs
// and the pairs are sorted by key.  The keys and the values are stored as
// byte slices.
//
// A table always starts out in memory then is written to storage using
// a Writer.  Tables can be combined with a Combiner.  They are merged in an
// efficient LSM like fashion.
//
// A table on disk consists of the base table with zero or more index files.
// These files are all named with the same path prefix, e.g., "sst", where the
// base table is sst and the index files, if any, are sst.1, sst.2, sst.3,
// and so forth.  The index files can always be recreated from the base table.
//
// A file constists of a sequence of frames, where each frame consists of a
// frame header, indicating its length (on disk) and compression type, followed
// by a possibly compressed body, where the body costs of a sequence of key, value
// pairs.  Each key and each value is encoded as a 4-byte length and sequence of bytes
// representing the key's string or the value's byte slice.  The body is compressed
// according to the compression type.
//
// When an SST file has values of all the same length, the length is indicated
// in the file header and omitted from each key-value pair.  XXX we could do
// the same for the key, but haven't done so yet.
//
// An index file contains a key,value pair for each frame in the file below
// in the hiearchy where the key is the first key found in that frame and
// the value is the offset or the frame in the file below.  Each frame in an
// index file is terminated with an end-of-frame key whose value is the
// beginning key of the next frame.
package zdx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
)

var (
	ErrCorruptFile = errors.New("corrupt zdx file")
	ErrValueSize   = errors.New("bad value size")
	ErrBadMagic    = errors.New("bad magic number in header of zdx file")
	ErrFileVersion = errors.New("unsupported version number in zdx file")
	ErrPairTooBig  = errors.New("length of a single key/value larger than frame")
)

const (
	magic        = 0xfeed2019
	versionMajor = 1
	versionMinor = 0

	// File header format is in big-endian format comprising:
	// 4 bytes of magic
	// one byte of major version
	// one byte of minor version
	// 4 bytes length of the values, when all the
	//     values have the same size (in which case we don't need to encode
	//     the value length with each value), or 0 for variable length values.
	// 4 bytes length of the minimum frame size
	FileHeaderLen = 14

	// Frame header format is in big-endian format comprising:
	// one byte of compression type
	// 4 bytes length of the frame (in uncompressed bytes)
	FrameHeaderLen = 5
)

type Pair struct {
	Key   []byte
	Value []byte
}

// Stream is an interface to enumerate all the key/values from an SST.
// A stream may be generated from an in-memory table, from a table on disk,
// from a combiner that combines two or more such tables from memory and/or disk,
// or from a custom implementation of the interface.
type Stream interface {
	// Open positions the reader at the lowest valued key so that Read() may
	// enumerate all the keys from the SST in sorted order.  Close() should
	// be called after the caller is done reading, at which time, Open() can
	// be called again.
	Open() error
	// Read returns the next key/value pair (or an error) in the enumeration
	// of key/values from the SST and returns zero values at end of table.
	// The byte slices of the pair should be copied by the caller if the
	// want to be retained beyond the next call to Read.
	Read() (Pair, error)
	// Close releases the underlying resources associated with the Stream
	Close() error
}

func filename(path string, level int) string {
	if level == 0 {
		return path
	}
	return fmt.Sprintf("%s.%d", path, level)
}

func Remove(path string) error {
	level := 0
	for {
		err := os.Remove(filename(path, level))
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
			}
			return err
		}
		level++
	}
}

func Rename(oldpath, newpath string) error {
	level := 0
	for {
		err := os.Rename(filename(oldpath, level), filename(newpath, level))
		if err != nil {
			if os.IsNotExist(err) {
				err = nil
			}
			return err
		}
		level++
	}
}

func encodeInt(dst []byte, v int) {
	// assume cap(dst) >= 4
	dst[0] = byte(v >> 24)
	dst[1] = byte(v >> 16)
	dst[2] = byte(v >> 8)
	dst[3] = byte(v)
}

func encodeInt48(dst []byte, v int64) {
	// assume cap(dst) >= 6
	dst[0] = byte(v >> 40)
	dst[1] = byte(v >> 32)
	dst[2] = byte(v >> 24)
	dst[3] = byte(v >> 16)
	dst[4] = byte(v >> 8)
	dst[5] = byte(v)
}

func decodeInt(buf []byte) int {
	v := int(buf[0]) << 24
	v |= int(buf[1]) << 16
	v |= int(buf[2]) << 8
	v |= int(buf[3])
	return v
}

func decodeInt48(buf []byte) int64 {
	v := int64(buf[0]) << 40
	v |= int64(buf[1]) << 32
	v |= int64(buf[2]) << 24
	v |= int64(buf[3]) << 16
	v |= int64(buf[4]) << 8
	v |= int64(buf[5])
	return v
}

func decodeCounted(buf []byte) []byte {
	n := decodeInt(buf[0:4])
	buf = buf[4:]
	if n > len(buf) {
		return nil
	}
	return buf[:n]
}

// Copy copies src to dst a la io.Copy.  The src stream is opened, read from,
// and closed in accordance with the interface.  The dst writer is written to
// and closed.
func Copy(dst *Writer, src Stream) error {
	return CopyWithContext(context.Background(), dst, src)
}

// CopyWithContext is just like Copy except it will abruptly end and return
// the context's error if it is canceled.  If the src was successfully opened,
// it will be closed whether or not errors or canceled context arise.
func CopyWithContext(ctx context.Context, dst *Writer, src Stream) error {
	if err := src.Open(); err != nil {
		return err
	}
	var err error
	for ctx.Err() == nil {
		var pair Pair
		pair, err = src.Read()
		if err != nil {
			return err
		}
		if pair.Key == nil {
			break
		}
		err = dst.Write(pair)
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

//XXX
func FirstKey(frame []byte) []byte {
	return decodeCounted(frame)
}

// DecodePair decodes a key and a value from the base sst file.
// To do so, it decodes a counted key and a counted value and returns
// the key and value byte slices as a Pair.  If the buffer is
// not large enough to decode these items, then the underlying data is
// corrupt and a zero-valued Pair is returned.  The second return
// value is the number of bytes consumed in decoding the Pair.
func DecodePair(buf []byte) (Pair, int) {
	n1 := decodeInt(buf[0:4])
	if n1+4 > len(buf) {
		return Pair{}, 0
	}
	n2 := decodeInt(buf[n1+4:])
	n := n1 + n2 + 8
	if n > len(buf) {
		return Pair{}, 0
	}
	return Pair{buf[4 : n1+4], buf[n1+8 : n]}, n
}

// DecodeIndex decodes a key and a file offset where this entry points
// to frame in that file (at the given offset) where keys with that
// same key begin.  To do so, it decodes a counted key and a 6-byte
// encoded offset and returns the key and the offset.  If the buffer is
// not large enough to decode these items, then the underlying data is
// corrupt and nil is returned for the key.  The third return value
// is the number of bytes consumed in decoding the key,offset pair.
func DecodeIndex(buf []byte) ([]byte, int64, int) {
	n := decodeInt(buf[0:4])
	if n+10 > len(buf) {
		return nil, -1, 0
	}
	key := buf[4 : n+4]
	off := decodeInt48(buf[n+4 : n+10])
	return key, off, n + 10
}

// Equal returns true iff the underlying byte slice contents of the two keys and
// and two values are equal to eachother.
func (p Pair) Equal(v Pair) bool {
	return bytes.Equal(p.Key, v.Key) && bytes.Equal(p.Value, v.Value)
}

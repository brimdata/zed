package zdx

import (
	"bytes"
	"fmt"
	"os"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// Finder looks up values in a zdx using its hierarchical index files.
type Finder struct {
	path  string
	files []*Reader
}

// NewFinder returns an object that is used to lookup keys in a zdx.
func NewFinder(path string) *Finder {
	return &Finder{
		path: path,
	}
}

// Open prepares a zdx bundle for lookups and return the zng.Type of the
// keys stored in this index.  If the bundle exists but is empty, zero
// values are returned.  If the bundle does not exist, os.ErrNotExist is returned.
func (f *Finder) Open() (zng.Type, error) {
	level := 0
	for {
		r, err := newReader(f.path, level)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}
			if len(f.files) > 0 {
				// Ignore this error; the prior is more
				// interesting.
				_ = f.Close()
			}
			return nil, err
		}
		level += 1
		f.files = append(f.files, r)
	}
	if len(f.files) == 0 {
		return nil, os.ErrNotExist
	}
	// Read the first record to determine the key type, then seek back
	// to the beginning.
	rec, err := f.files[0].Read()
	if err != nil {
		return nil, err
	}
	if rec == nil {
		// index exists but is empty
		return nil, nil
	}
	ncol := len(rec.Type.Columns)
	if (ncol != 1 && ncol != 2) || rec.Type.Columns[0].Name != "key" {
		return nil, fmt.Errorf("%s: not a zdx budnle", f.path)
	}
	keyType := rec.Type.Columns[0].Type
	_, err = f.files[0].Seek(0)
	return keyType, err
}

func (f *Finder) Path() string {
	return f.path
}

func (f *Finder) Close() error {
	for _, r := range f.files {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// lookupOffset finds the largest key that is smaller than the input key
// in the record stream read from the reader, where key is the first column
// of each record in the stream.  The second column in the record must be an
// integer and that value is returned for the largest said key.
func lookupOffset(reader zbuf.Reader, key zcode.Bytes) (int64, error) {
	var lastOff int64 = -1
	for {
		rec, err := reader.Read()
		if rec == nil {
			break
		}
		// Since we know that each record is a key in column 0 and
		// and an int64 offset in column 1, we can pull the zcode.Bytes
		// encoding out directly with Slice.
		k, err := rec.Slice(0)
		if err != nil {
			return -1, err
		}
		if bytes.Compare(key, k) < 0 {
			break
		}
		off, err := rec.Slice(1)
		if err != nil {
			return -1, err
		}
		lastOff, err = zng.DecodeInt(off)
		if err != nil {
			return -1, err
		}
	}
	return lastOff, nil
}

// lookupExact searches for an exact match of the given key compared to the
// first columns of the records read from the reader.  If a match is found,
// the returned boolean is true and, if a second column in the matched record
// exists, the zng.Value of that column is returned.  If there is no match found,
// then the boolean return value is false and the zng.Value is the zero value.
func lookup(reader zbuf.Reader, key zcode.Bytes, exact bool) (*zng.Record, error) {
	var prev *zng.Record
	for {
		rec, err := reader.Read()
		if rec == nil {
			return nil, err
		}
		k, err := rec.Slice(0)
		if err != nil {
			return nil, err
		}
		if bytes.Compare(k, key) >= 0 {
			if bytes.Equal(key, k) {
				return rec, nil
			}
			if exact {
				return nil, nil
			}
			return prev, nil
		}
		prev = rec

	}
}

func (f *Finder) search(key zcode.Bytes) error {
	n := len(f.files)
	if n == 0 {
		return ErrCorruptFile
	}
	// We start with the topmost index file of the zdx bundle and
	// find the greatest key smaller than or equal to othe lookup key then repeat
	// the process for that frame in the next index file till we get to the
	// base file and return that offset.
	level := n - 1
	off := int64(0)
	for level > 0 {
		reader := f.files[level]
		if _, err := reader.Seek(off); err != nil {
			return err
		}
		var err error
		off, err = lookupOffset(reader, key)
		if err != nil {
			return err
		}
		if off == -1 {
			// This key can't be in the zdx since it is smaller than
			// the smallest key in the zdx's index files.
			return nil
		}
		level -= 1
	}
	_, err := f.files[0].Seek(off)
	return err
}

func (f *Finder) Lookup(key zcode.Bytes) (*zng.Record, error) {
	if err := f.search(key); err != nil {
		return nil, err
	}
	return lookup(f.files[0], key, true)
}

func (f *Finder) LookupClosest(key zcode.Bytes) (*zng.Record, error) {
	if err := f.search(key); err != nil {
		return nil, err
	}
	return lookup(f.files[0], key, false)
}

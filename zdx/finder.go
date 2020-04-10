package zdx

import (
	"bytes"
	"os"
)

// Finder looks up values in a zdx using its hierarchical index files.
type Finder struct {
	files []*FrameReader
}

// NewFinder returns an object that is used to lookup keys in a zdx.
func NewFinder(path string) (*Finder, error) {
	level := 0
	f := &Finder{}
	for {
		r := NewFrameReader(path, level)
		if err := r.Open(); err != nil {
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
	return f, nil
}

func (f *Finder) Close() error {
	for _, r := range f.files {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// find the largest key that is smaller than the input key and
// return the value interpreted as a 6-byte offset
func lookupOffset(key, buf []byte) (int64, error) {
	var lastOff int64 = -1
	for len(buf) > 10 {
		k, off, n := DecodeIndex(buf)
		if k == nil {
			return -1, ErrCorruptFile
		}
		if bytes.Compare(key, k) < 0 {
			break
		}
		lastOff = off
		buf = buf[n:]
	}
	return lastOff, nil
}

// find an exact match for the key and return the corresponding value
// if no match exists, return nil
func lookupValue(key, buf []byte) ([]byte, error) {
	for len(buf) > 8 {
		pair, n := DecodePair(buf)
		if pair.Key == nil {
			return nil, ErrCorruptFile
		}
		if bytes.Equal(key, pair.Key) {
			return pair.Value, nil
		}
		buf = buf[n:]
	}
	return nil, nil
}

func (f *Finder) Lookup(key []byte) ([]byte, error) {
	n := len(f.files)
	if n == 0 {
		//XXX
		return nil, ErrCorruptFile
	}
	// We start with the topmost file (which always is one frame) and
	// find the greatest key smaller than the lookup key then repeat
	// the process for that frame in the next index file till we get to the
	// base file and we look for the exact key.
	level := n - 1
	off := int64(FileHeaderLen)
	for level > 0 {
		frame, err := f.files[level].ReadFrameAt(off)
		if err != nil {
			return nil, err
		}
		off, err = lookupOffset(key, frame)
		if err != nil {
			return nil, err
		}
		if off == -1 {
			// This key can't be in the zdx since it is smaller than
			// the smallest key in the zdx's index files.
			return nil, nil
		}
		level -= 1
	}
	frame, err := f.files[level].ReadFrameAt(off)
	if err != nil {
		return nil, err
	}
	return lookupValue(key, frame)
}

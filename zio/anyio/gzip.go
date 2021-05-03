package anyio

import (
	"compress/gzip"
	"io"
)

func GzipReader(r io.Reader) io.Reader {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	_, err := gzip.NewReader(track)
	if err == nil {
		// create a new reader from recorder (track keeps a copy of read data)
		r, _ := gzip.NewReader(recorder)
		return r
	}
	return recorder
}

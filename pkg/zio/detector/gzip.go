package detector

import (
	"compress/gzip"
	"io"
)

func GzipReader(r io.Reader) io.Reader {
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	r, err := gzip.NewReader(track)
	if err == nil {
		return r
	}
	return recorder
}

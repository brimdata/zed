package anyio

import (
	"compress/gzip"
	"io"
)

func GzipReader(r io.Reader) (io.Reader, error) {
	if rs, ok := r.(io.ReadSeeker); ok {
		if n, err := rs.Seek(0, io.SeekCurrent); err == nil {
			if r, err := gzip.NewReader(rs); err == nil {
				return r, nil
			}
			if _, err := rs.Seek(n, io.SeekStart); err != nil {
				return nil, err
			}
			return rs, nil
		}
	}
	recorder := NewRecorder(r)
	track := NewTrack(recorder)
	_, err := gzip.NewReader(track)
	if err == nil {
		// create a new reader from recorder (track keeps a copy of read data)
		return gzip.NewReader(recorder)
	}
	return recorder, nil
}

package anyio

const TrackSize = InitBufferSize

type Track struct {
	recorder *Recorder
	off      int
}

func NewTrack(r *Recorder) *Track {
	return &Track{
		recorder: r,
	}
}

func (t *Track) Reset() {
	t.off = 0
}

func (t *Track) Read(b []byte) (int, error) {
	n, err := t.recorder.ReadAt(t.off, b)
	t.off += n
	return n, err
}

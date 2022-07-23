package column

import "io"

type PresenceWriter struct {
	IntWriter
	run  int32
	null bool
}

func NewPresenceWriter(spiller *Spiller) *PresenceWriter {
	return &PresenceWriter{
		IntWriter: *NewIntWriter(spiller),
	}
}

func (p *PresenceWriter) TouchValue() {
	if !p.null {
		p.run++
	} else {
		p.Write(p.run)
		p.run = 1
		p.null = false
	}
}

func (p *PresenceWriter) TouchNull() {
	if p.null {
		p.run++
	} else {
		p.Write(p.run)
		p.run = 1
		p.null = true
	}
}

func (p *PresenceWriter) Finish() {
	p.Write(p.run)
}

type PresenceReader struct {
	IntReader
	null bool
	run  int
}

func NewPresenceReader(segmap []Segment, r io.ReaderAt) *PresenceReader {
	// We start out with null true so it is immediately flipped to
	// false on the first call to Read.
	return &PresenceReader{
		IntReader: *NewIntReader(segmap, r),
		null:      true,
	}
}

func (p *PresenceReader) IsEmpty() bool {
	return len(p.segmap) == 0
}

func (p *PresenceReader) Read() (bool, error) {
	run := p.run
	for run == 0 {
		p.null = !p.null
		v, err := p.IntReader.Read()
		if err != nil {
			return false, err
		}
		run = int(v)
	}
	p.run = run - 1
	return !p.null, nil
}

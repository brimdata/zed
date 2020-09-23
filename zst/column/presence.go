package column

type PresenceWriter struct {
	IntWriter
	run   int32
	unset bool
}

func NewPresenceWriter(spiller *Spiller) *PresenceWriter {
	return &PresenceWriter{
		IntWriter: *NewIntWriter(spiller),
	}
}

func (p *PresenceWriter) TouchValue() {
	if !p.unset {
		p.run++
	} else {
		p.Write(p.run)
		p.run = 1
		p.unset = false
	}
}

func (p *PresenceWriter) TouchUnset() {
	if p.unset {
		p.run++
	} else {
		p.Write(p.run)
		p.run = 1
		p.unset = true
	}
}

func (p *PresenceWriter) Finish() {
	p.Write(p.run)
}

type Presence struct {
	Int
	unset bool
	run   int
}

func NewPresence() *Presence {
	// We start out with unset true so it is immediately flipped to
	// false on the first call to Read.
	return &Presence{unset: true}
}

func (p *Presence) IsEmpty() bool {
	return len(p.segmap) == 0
}

func (p *Presence) Read() (bool, error) {
	run := p.run
	for run == 0 {
		p.unset = !p.unset
		v, err := p.Int.Read()
		if err != nil {
			return false, err
		}
		run = int(v)
	}
	p.run = run - 1
	return !p.unset, nil
}

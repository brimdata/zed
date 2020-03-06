package pcap

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"sync"

	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/ranger"
	"github.com/brimsec/zq/pkg/slicer"
)

type Index []Section

// Section indicates the seek offset of a pcap section.  For legacy pcaps,
// there is just one section at the beginning of the file.  For nextgen pcaps,
// there can be multiple sections.
type Section struct {
	Blocks []slicer.Slice
	Index  ranger.Envelope
}

// CreateIndex creates an index for a pcap file.  If the file isn't
// a pcap file, an error is returned.
func CreateIndex(r io.Reader, limit int) (Index, error) {
	reader, err := pcapio.NewPcapReader(r) // XXX TBD: lookup the right reader
	if err != nil {
		return nil, err
	}
	var offsets []ranger.Point
	for {
		off := reader.Offset()
		data, info, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if data == nil {
			break
		}
		y := uint64(info.Ts)
		offsets = append(offsets, ranger.Point{X: off, Y: y})
	}
	n := len(offsets)
	if n == 0 {
		return nil, errors.New("no packets found")
	}
	fileHeaderLen := uint64(24) // XXX this will go away in next PR
	// legacy pcap file has just the file header at the start of the file
	blocks := []slicer.Slice{{0, fileHeaderLen}}
	return Index{
		{
			Blocks: blocks,
			Index:  ranger.NewEnvelope(offsets, limit),
		},
	}, nil
}

type IndexWriter struct {
	io.WriteCloser
	err error
	idx Index
	wg  sync.WaitGroup
}

func (w *IndexWriter) run(r *io.PipeReader, limit int) {
	w.idx, w.err = CreateIndex(r, limit)
	if w.err != nil {
		r.CloseWithError(w.err)
	}
	w.wg.Done()
}

func NewIndexWriter(limit int) *IndexWriter {
	pr, pw := io.Pipe()
	i := &IndexWriter{WriteCloser: pw}
	i.wg.Add(1)
	go i.run(pr, limit)
	return i
}

func (w *IndexWriter) Close() (Index, error) {
	w.WriteCloser.Close()
	w.wg.Wait()
	return w.idx, w.err
}

func LoadIndex(path string) (Index, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var index Index
	err = json.Unmarshal(b, &index)
	return index, err
}

package pcap

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/ranger"
	"github.com/brimsec/zq/pkg/slicer"
)

type Index []Section

// Span returns the entire time span covered by the index.
func (i Index) Span() nano.Span {
	var span nano.Span
	for _, s := range i {
		for _, bin := range s.Index {
			binspan := nano.NewSpanTs(nano.Ts(bin.Range.Y0), nano.Ts(bin.Range.Y1))
			if span.Ts == 0 {
				span = binspan
				continue
			}
			span = span.Union(binspan)
		}
	}
	return span
}

// Section indicates the seek offset of a pcap section.  For legacy pcaps,
// there is just one section at the beginning of the file.  For nextgen pcaps,
// there can be multiple sections.
type Section struct {
	Blocks []slicer.Slice
	Index  ranger.Envelope
}

// CreateIndex creates an index for a pcap presented as an io.Reader.
// The size parameter indicates how many bins the index should contain.
func CreateIndex(r io.Reader, size int) (Index, error) {
	reader, err := pcapio.NewReader(r)
	if err != nil {
		return nil, err
	}
	var offsets []ranger.Point
	var index Index
	var section *Section
	for {
		off := reader.Offset()
		block, typ, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if block == nil {
			break
		}
		switch typ {
		default:
			if section == nil {
				err := errors.New("missing section header")
				return nil, pcapio.NewErrInvalidPcap(err)
			}
			slice := slicer.Slice{
				Offset: off,
				Length: uint64(len(block)),
			}
			section.Blocks = append(section.Blocks, slice)

		case pcapio.TypePacket:
			pkt, ts, _, err := reader.Packet(block)
			if pkt == nil {
				return nil, err
			}
			y := uint64(ts)
			offsets = append(offsets, ranger.Point{X: off, Y: y})

		case pcapio.TypeSection:
			// end previous section and start a new one
			if section == nil && offsets != nil {
				err = errors.New("missing section header")
				return nil, pcapio.NewErrInvalidPcap(err)
			}
			if section != nil && offsets != nil {
				section.Index = ranger.NewEnvelope(offsets, size)
				index = append(index, *section)
			}
			slice := slicer.Slice{
				Offset: off,
				Length: uint64(len(block)),
			}
			section = &Section{
				Blocks: []slicer.Slice{slice},
			}
			offsets = nil
		}
	}
	// end last section
	if section != nil && offsets != nil {
		section.Index = ranger.NewEnvelope(offsets, size)
		index = append(index, *section)
	}
	if len(index) == 0 {
		return nil, ErrNoPcapsFound
	}
	return index, nil
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

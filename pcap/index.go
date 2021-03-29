package pcap

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"unsafe"

	"github.com/brimdata/zq/pcap/pcapio"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/pkg/ranger"
	"github.com/brimdata/zq/pkg/slicer"
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

const (
	pointSize    = unsafe.Sizeof(ranger.Point{})
	offsetMaxMem = 1024 * 1024 * 64 // 64MB
)

// offsetThresh is the max number of pcap offsets collected before offset array
// in CreateIndex is condensed with ranger.NewEvelope.
var offsetThresh = offsetMaxMem / int(pointSize)

// CreateIndex creates an index for a pcap presented as an io.Reader.
// The size parameter indicates how many bins the index should contain.
func CreateIndex(r io.Reader, size int) (Index, error) {
	return CreateIndexWithWarnings(r, size, nil)
}

func CreateIndexWithWarnings(r io.Reader, size int, c chan<- string) (Index, error) {
	reader, err := pcapio.NewReaderWithWarnings(r, c)
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
		case pcapio.TypePacket:
			pkt, ts, _, err := reader.Packet(block)
			if pkt == nil {
				return nil, err
			}
			y := uint64(ts)
			offsets = append(offsets, ranger.Point{X: off, Y: y})
			// In order to avoid running out of memory for large pcap sections,
			// condense offsets with ranger.NewEnvelope once offsetThresh has
			// been reached.
			if len(offsets) > offsetThresh {
				env := ranger.NewEnvelope(offsets, size)
				section.Index = env.Merge(section.Index)
				offsets = offsets[:0]
			}

		case pcapio.TypeSection:
			// end previous section and start a new one
			if section == nil && offsets != nil {
				err := errors.New("missing section header")
				return nil, pcapio.NewErrInvalidPcap(err)
			}
			if section != nil {
				if offsets != nil {
					env := ranger.NewEnvelope(offsets, size)
					section.Index = env.Merge(section.Index)
				}
				index = append(index, *section)
			}
			slice := slicer.Slice{
				Offset: off,
				Length: uint64(len(block)),
			}
			section = &Section{
				Blocks: []slicer.Slice{slice},
			}
			offsets = offsets[:0]

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
		}
	}
	// end last section
	if section != nil && offsets != nil {
		env := ranger.NewEnvelope(offsets, size)
		section.Index = env.Merge(section.Index)
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

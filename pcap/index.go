package pcap

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

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
				return nil, errors.New("missing section header")
			}
			slice := slicer.Slice{
				Offset: off,
				Length: uint64(len(block)),
			}
			section.Blocks = append(section.Blocks, slice)

		case pcapio.TypePacket:
			pkt, ts, _ := reader.Packet(block)
			if pkt == nil {
				return nil, pcapio.ErrCorruptPcap
			}
			y := uint64(ts)
			offsets = append(offsets, ranger.Point{X: off, Y: y})

		case pcapio.TypeSection:
			// end previous section and start a new one
			if section == nil && offsets != nil {
				return nil, errors.New("packets found without section header")
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
		return nil, ErrNoPacketsFound
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

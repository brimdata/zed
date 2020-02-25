package pcap

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/ranger"
	"github.com/brimsec/zq/pkg/slicer"
)

type Index []Section

type Section struct {
	Blocks []slicer.Slice
	Index  ranger.Envelope
}

// CreateIndex creates an index for a pcap file.  If the file isn't
// a pcap file, an error is returned.
func CreateIndex(path string, limit int) (Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	reader, err := pcapio.NewPcapReader(f) // XXX TBD NewReader
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
		return nil, fmt.Errorf("%s: no packets found in pcap file", path)
	}
	// legacy pcap file has just the file header at the start of the file
	fileHeaderLen := uint64(24) // XXX this will go away
	blocks := []slicer.Slice{{0, fileHeaderLen}}
	return Index{
		{
			Blocks: blocks,
			Index:  ranger.NewEnvelope(offsets, limit),
		},
	}, nil
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

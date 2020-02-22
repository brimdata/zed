package pcap

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/pkg/ranger"
)

// Slicer implements io.Reader reading the sliced regions provided to it from
// the underlying file thus extracting subsets of an underlying file without
// modifying or copying the file.
type Slicer struct {
	slices []Slice
	file   *os.File
	reader *io.SectionReader
}

func NewSlicer(file *os.File, index *Index, span nano.Span) (*Slicer, error) {
	slices, err := GenerateSlices(index, span)
	if err != nil {
		return nil, err
	}
	first := slices[0]
	return &Slicer{
		slices: slices[1:],
		file:   file,
		reader: first.NewReader(file),
	}, nil
}

func (s *Slicer) Read(b []byte) (int, error) {
	for s.reader != nil {
		n, err := s.reader.Read(b)
		if n != 0 {
			if err == io.EOF {
				err = nil
			}
			return n, err
		}
		if err != io.EOF {
			return 0, err
		}
		if len(s.slices) != 0 {
			s.reader = s.slices[0].NewReader(s.file)
			s.slices = s.slices[1:]
		} else {
			s.reader = nil
		}
	}
	return 0, io.EOF
}

func (s Slice) Overlaps(x Slice) bool {
	return x.Offset >= s.Offset && x.Offset < s.Offset+x.Length
}

func (s Slice) NewReader(r io.ReaderAt) *io.SectionReader {
	return io.NewSectionReader(r, int64(s.Offset), int64(s.Length))
}

// GenerateSlices takes an index and time span and generates a list of
// slices that should be read to enumerate the relevant chunks of an
// underlying pcap file.  Extra packets may appear in the resulting stream
// but all packets that fall within the time range will be produced, i.e.,
// another layering of time filtering should be applied to resulting packets.
func GenerateSlices(index *Index, span nano.Span) ([]Slice, error) {
	var slices []Slice
	for _, section := range index.Sections {
		pslice, err := FindPacketSlice(section.Index, span)
		if err != nil {
			return nil, err
		}
		for _, slice := range section.Blocks {
			if !pslice.Overlaps(slice) {
				slices = append(slices, slice)
			}
		}
		slices = append(slices, pslice)
	}
	return slices, nil
}

func FindPacketSlice(e ranger.Envelope, span nano.Span) (Slice, error) {
	if len(e) == 0 {
		return Slice{}, errors.New("no packets")
	}
	d := e.FindSmallestDomain(ranger.Range{uint64(span.Ts), uint64(span.End())})
	//XXX check for empty domain.. though seems like this will do the right thing
	return Slice{d.X0, d.X1 - d.X0}, nil
}

func LoadIndex(path string) (*Index, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var index *Index
	err = json.Unmarshal(b, &index)
	return index, err
}

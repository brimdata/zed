package pcap

import (
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/mccanne/zq/pkg/nano"
)

type file struct {
	path string
	ts   nano.Ts
}

func newFile(path string) (*file, error) {
	ts, err := ParseFileName(path)
	if err != nil {
		return nil, err
	}

	return &file{
		path: path,
		ts:   ts,
	}, nil
}

type pcapfiles []*file

func (p pcapfiles) Len() int           { return len(p) }
func (p pcapfiles) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p pcapfiles) Less(i, j int) bool { return p[i].ts < p[j].ts }

// Store manages a sorted list of pcaps files in a store.
type Store struct {
	files pcapfiles
}

// All returns a list of all pcap files in the store.
func (s *Store) All() []string {
	files := []string{}
	for _, p := range s.files {
		files = append(files, p.path)
	}

	return files
}

// findGreatest returns the index of the file with the greatest timestamp that
// is less than provided timestamp.
func (s *Store) findGreatest(ts nano.Ts) int {
	index := sort.Search(len(s.files), func(i int) bool {
		return ts-s.files[i].ts < 0
	})

	if index != 0 {
		index--
	}

	return index
}

// Range returns an array of files that overlaps the provided time boundaries.
func (s *Store) Range(span nano.Span) []string {
	end := span.End()

	index := s.findGreatest(span.Ts)
	files := []string{s.files[index].path}
	for _, pfile := range s.files[index+1:] {
		if pfile.ts > end {
			break
		}

		files = append(files, pfile.path)
	}

	return files
}

// NewStore reads the files from the provided store and returns a PcapStore
// object with the derived information.
func NewStore(dir string) (*Store, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	pcfs := make(pcapfiles, 0)
	for _, f := range files {
		pc, err := newFile(filepath.Join(dir, f.Name()))
		if err == nil {
			pcfs = append(pcfs, pc)
		}
	}

	sort.Sort(pcfs)

	return &Store{
		files: pcfs,
	}, nil
}

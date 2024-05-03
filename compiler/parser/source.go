package parser

import (
	"bytes"
	"os"
	"sort"
)

// ConcatSource concatenates the source files in filenames followed by src,
// returning the result and a corresponding slice of SourceInfos.
func ConcatSource(filenames []string, src string) (*SourceSet, error) {
	var b bytes.Buffer
	set := new(SourceSet)
	for _, f := range filenames {
		bb, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		set.Sources = append(set.Sources, newSourceInfo(f, b.Len(), bb))
		b.Write(bb)
		b.WriteByte('\n')
	}
	if b.Len() == 0 && src == "" {
		src = "*"
	}
	set.Sources = append(set.Sources, newSourceInfo("", b.Len(), []byte(src)))
	b.WriteString(src)
	set.Contents = b.Bytes()
	return set, nil
}

type SourceSet struct {
	Contents []byte
	Sources  []*SourceInfo
}

func (s *SourceSet) SourceOf(pos int) *SourceInfo {
	i := sort.Search(len(s.Sources), func(i int) bool { return s.Sources[i].start > pos }) - 1
	return s.Sources[i]
}

// SourceInfo holds source file offsets.
type SourceInfo struct {
	filename string
	lines    []int
	size     int
	start    int
}

func newSourceInfo(filename string, start int, src []byte) *SourceInfo {
	var lines []int
	line := 0
	for offset, b := range src {
		if line >= 0 {
			lines = append(lines, line)
		}
		line = -1
		if b == '\n' {
			line = offset + 1
		}
	}
	return &SourceInfo{
		filename: filename,
		lines:    lines,
		size:     len(src),
		start:    start,
	}
}

func (s *SourceInfo) Position(pos int) (string, Position) {
	if pos < 0 {
		return "", Position{-1, -1, -1, -1}
	}
	offset := pos - s.start
	i := searchLine(s.lines, offset)
	return s.filename, Position{
		Pos:    pos,
		Offset: offset,
		Line:   i + 1,
		Column: offset - s.lines[i] + 1,
	}
}

func (s *SourceInfo) LineOfPos(set *SourceSet, pos int) string {
	i := searchLine(s.lines, pos-s.start)
	start := s.lines[i]
	end := s.size
	if i+1 < len(s.lines) {
		end = s.lines[i+1]
	}
	b := set.Contents[s.start+start : s.start+end]
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return string(b)
}

func searchLine(lines []int, offset int) int {
	return sort.Search(len(lines), func(i int) bool { return lines[i] > offset }) - 1

}

type Position struct {
	Pos    int `json:"pos"`    // Offset relative to SourceSet.
	Offset int `json:"offset"` // Offset relative to file start.
	Line   int `json:"line"`   // 1-based line number.
	Column int `json:"column"` // 1-based column number.
}

func (p Position) IsValid() bool { return p.Pos >= 0 }

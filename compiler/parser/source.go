package parser

import (
	"sort"
)

type SourceSet struct {
	Text    string
	Sources []*SourceInfo
}

func (s *SourceSet) SourceOf(pos int) *SourceInfo {
	i := sort.Search(len(s.Sources), func(i int) bool { return s.Sources[i].start > pos }) - 1
	return s.Sources[i]
}

// SourceInfo holds source file offsets.
type SourceInfo struct {
	Filename string
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
		Filename: filename,
		lines:    lines,
		size:     len(src),
		start:    start,
	}
}

func (s *SourceInfo) Position(pos int) Position {
	if pos < 0 {
		return Position{-1, -1, -1, -1}
	}
	offset := pos - s.start
	i := searchLine(s.lines, offset)
	return Position{
		Pos:    pos,
		Offset: offset,
		Line:   i + 1,
		Column: offset - s.lines[i] + 1,
	}
}

func (s *SourceInfo) LineOfPos(src string, pos int) string {
	i := searchLine(s.lines, pos-s.start)
	start := s.lines[i]
	end := s.size
	if i+1 < len(s.lines) {
		end = s.lines[i+1]
	}
	b := src[s.start+start : s.start+end]
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

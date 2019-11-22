package skim

import (
	"bytes"
	"errors"
	"io"
)

// ErrLineTooLong means there was a line encountered that exceeded max_line_size
// for the space.
var ErrLineTooLong = errors.New("line too long")

type Stats struct {
	Bytes       int `json:"bytes_read"`
	Lines       int `json:"lines_read"`
	BlankLines  int `json:"blank_lines"`
	LineTooLong int `json:"line_too_long"`
}

// Scanner is like bufio.Scanner but it
// it understands how to skip over and report lines that are too long.
type Scanner struct {
	Stats
	reader io.Reader
	buffer []byte
	limit  int
	window []byte
}

// we handle only newline-terminated lines
const token = byte('\n')

func NewScanner(r io.Reader, buffer []byte, limit int) *Scanner {
	return &Scanner{Stats{}, r, buffer, limit, nil}
}

// grow the buffer and copy the data from the old buffer to
// the new buffer.  also, update the window.  returns false if the buffer
// has already hit the max allowable size and doesn't do anything.
func (s *Scanner) grow() bool {
	n := len(s.buffer)
	if n >= s.limit {
		return false
	}
	newsize := n * 2
	if newsize > s.limit {
		newsize = s.limit
	}
	s.buffer = make([]byte, newsize)
	cc := copy(s.buffer, s.window)
	s.window = s.buffer[0:cc]
	return true
}

func (s *Scanner) more() error {
	var cushion int
	if s.window != nil {
		cushion = copy(s.buffer, s.window)
	}
	cc, err := s.reader.Read(s.buffer[cushion:])
	if cc <= 0 {
		return err
	}
	n := cc + cushion
	s.window = s.buffer[0:n]
	return nil
}

// Skip discards all input up to and including the next newline or
// end of file, and returns the number of bytes skipped.  Returns an
// error if the underlying reader returns an error, except for EOF,
// which is ignored since the caller will detect EOF on the next
// call to Scan.
func (s *Scanner) Skip() (int, error) {
	var nskip int
	for {
		if s.window == nil {
			if err := s.more(); err != nil {
				// Don't return EOF here...
				// client might have more data to
				// process and can rely upon calling
				// the Scan returning nil, eof
				if err == io.EOF {
					err = nil
				}
				return nskip, err
			}
		}
		off := bytes.IndexByte(s.window, token)
		if off < 0 {
			nskip += len(s.window)
			s.window = nil
		} else {
			n := off + 1
			if n < len(s.window) {
				s.window = s.window[n:]
			} else {
				// the newline is precisely at the end
				// of the current buffer so we can start
				// fresh on the next call to Scan
				s.window = nil
			}
			nskip += n
			return nskip, nil
		}
	}
}

func (s *Scanner) check() error {
	if len(s.window) == 0 {
		return s.more()
	}
	return nil
}

func (s *Scanner) Peek() byte {
	if err := s.check(); err != nil {
		return 0
	}
	return s.window[0]
}

// Scan returns the next line of input as a byte slice or nil and
// an error indicating the state of things.  The newline terminating
// the line is returned in the slice.  When a line is encountered
// that is larger than the max line size, then the partial line is
// returned along with ErrLineTooLong.  In this case, Scan can be
// subsequently called for the rest of the line, possibly with another
// line too long error, and so on.  Skip can also be called to
// easily skip over the rest of the line.  At EOF, nil is returned.
// XXX If Scan is called directly instead of ScanLine, then Stats are
// not properly tracked.
// for the slice and io.EOF for the error.
func (s *Scanner) Scan() ([]byte, error) {
	if err := s.check(); err != nil {
		return nil, err
	}
	for {
		if off := bytes.IndexByte(s.window, token); off >= 0 {
			off++
			result := s.window[:off]
			// we found a line... advance the window
			// if the newline lands exactly at the end of
			// the buffer, just start over fresh for the
			// next call
			if off == len(s.window) {
				s.window = nil
			} else {
				s.window = s.window[off:]
			}
			return result, nil
		}
		// we didn't find a line.
		// if the buffer is full, it means it's too small to
		// hold a whole line... grow it and try again
		// if there is just a partial line at the end of the
		// buffer, then read more input and try again
		if len(s.window) == len(s.buffer) {
			// if we hit the max line size and can't
			// fit a line in the buffer, then we return
			// the current, partial line with an error.
			// and start over fresh
			if !s.grow() {
				result := s.window
				s.window = nil
				return result, ErrLineTooLong
			}
			// otherwise, we grew the buffer and fall
			// through here to read more input
		}
		if err := s.more(); err != nil {
			return nil, err
		}
	}
}

// Scan returns the next line skipping blank lines and too-long lines
// and accumulating statistics.
func (s *Scanner) ScanLine() ([]byte, error) {
	for {
		line, err := s.Scan()
		s.Bytes += len(line)
		s.Lines++
		if err == nil {
			if line == nil {
				return nil, nil
			}
			if len(line) <= 1 {
				// blank line, keep going
				s.BlankLines++
				continue
			}
			return line, nil
		}
		if err == io.EOF {
			return nil, nil
		}
		if err == ErrLineTooLong {
			s.LineTooLong++
			n, err := s.Skip()
			s.Bytes += n
			if err == nil {
				continue
			}
		}
		return nil, err
	}
}
